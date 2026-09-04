package verifier

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

const (
	CodeViewportHorizontalOverflow = "layout.viewport_horizontal_overflow"
	CodeContentClipped             = "layout.content_clipped"
	CodeInteractiveClipped         = "layout.interactive_clipped"
	CodeInteractiveOverlap         = "interaction.targets_overlap"
	CodeTargetTooSmall             = "interaction.target_too_small"
	CodeInteractiveHidden          = "interaction.target_hidden"
	CodePointerEventsDisabled      = "interaction.pointer_events_disabled"
	CodeStyleInvariant             = "style.invariant_violation"
)

type StyleInvariant struct {
	ID         string
	Tag        string
	Role       string
	Property   string
	Allowed    []string
	Disallowed []string
	Severity   evidence.Severity
}

type Policy struct {
	ViewportOverflowTolerance float64
	ClipTolerance             float64
	MinTargetWidth            float64
	MinTargetHeight           float64
	OverlapRatio              float64
	OverlapMinPixels          float64
	StyleInvariants           []StyleInvariant
}

func DefaultPolicy() Policy {
	return Policy{
		ViewportOverflowTolerance: 1,
		ClipTolerance:             1,
		MinTargetWidth:            24,
		MinTargetHeight:           24,
		OverlapRatio:              0.25,
		OverlapMinPixels:          4,
	}
}

type Result struct {
	Issues   []evidence.RuntimeIssue `json:"issues"`
	Duration time.Duration           `json:"-"`
}

// Verify runs deterministic checks only. It never performs visual/aesthetic
// inference and never mutates the packet.
func Verify(packet evidence.Packet, policy Policy) Result {
	policy = normalizePolicy(policy)
	started := time.Now()

	byID := make(map[string]evidence.ElementRef, len(packet.Elements))
	for _, element := range packet.Elements {
		byID[element.ID] = element
	}

	issues := make([]evidence.RuntimeIssue, 0, 8)
	issues = append(issues, verifyViewportOverflow(packet, policy)...)
	issues = append(issues, verifyClipping(packet.Elements, byID, policy)...)
	issues = append(issues, verifyInteractiveState(packet.Elements, policy)...)
	issues = append(issues, verifyInteractiveOverlap(packet.Elements, byID, policy)...)
	issues = append(issues, verifyStyleInvariants(packet.Elements, policy.StyleInvariants)...)

	sortIssues(issues)
	return Result{Issues: issues, Duration: time.Since(started)}
}

// Apply appends this suite's deterministic findings to the canonical packet.
// Existing runtime evidence from console/network/accessibility collectors is
// preserved. The resulting order is deterministic for stable downstream diffs.
func Apply(packet *evidence.Packet, policy Policy) Result {
	if packet == nil {
		return Result{}
	}
	result := Verify(*packet, policy)
	packet.RuntimeIssues = append(packet.RuntimeIssues, result.Issues...)
	sortIssues(packet.RuntimeIssues)
	return result
}

func normalizePolicy(policy Policy) Policy {
	defaults := DefaultPolicy()
	if policy.ViewportOverflowTolerance < 0 {
		policy.ViewportOverflowTolerance = defaults.ViewportOverflowTolerance
	}
	if policy.ClipTolerance < 0 {
		policy.ClipTolerance = defaults.ClipTolerance
	}
	if policy.MinTargetWidth <= 0 {
		policy.MinTargetWidth = defaults.MinTargetWidth
	}
	if policy.MinTargetHeight <= 0 {
		policy.MinTargetHeight = defaults.MinTargetHeight
	}
	if policy.OverlapRatio <= 0 || policy.OverlapRatio > 1 {
		policy.OverlapRatio = defaults.OverlapRatio
	}
	if policy.OverlapMinPixels <= 0 {
		policy.OverlapMinPixels = defaults.OverlapMinPixels
	}
	return policy
}

func verifyViewportOverflow(packet evidence.Packet, policy Policy) []evidence.RuntimeIssue {
	if packet.Viewport.Width <= 0 || len(packet.Documents) == 0 {
		return nil
	}
	document := packet.Documents[0]
	overflow := document.ContentWidth - float64(packet.Viewport.Width)
	if overflow <= policy.ViewportOverflowTolerance || explicitDocumentHorizontalScroll(packet.Elements, document.FrameID) {
		return nil
	}
	severity := evidence.SeverityMedium
	if overflow >= 8 {
		severity = evidence.SeverityHigh
	}
	return []evidence.RuntimeIssue{{
		Code:     CodeViewportHorizontalOverflow,
		Severity: severity,
		Message: fmt.Sprintf("document content width %.1fpx exceeds viewport width %dpx by %.1fpx", document.ContentWidth, packet.Viewport.Width, overflow),
	}}
}

func explicitDocumentHorizontalScroll(elements []evidence.ElementRef, frameID string) bool {
	for _, element := range elements {
		if frameID != "" && element.FrameID != frameID {
			continue
		}
		if element.Tag != "html" && element.Tag != "body" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(element.Styles["overflow-x"])) {
		case "auto", "scroll":
			return true
		}
	}
	return false
}

func verifyClipping(elements []evidence.ElementRef, byID map[string]evidence.ElementRef, policy Policy) []evidence.RuntimeIssue {
	issues := make([]evidence.RuntimeIssue, 0)
	for _, element := range elements {
		if !element.Visible || isIgnoredForGeometry(element) {
			continue
		}
		ancestorID := element.ParentID
		for ancestorID != "" {
			ancestor, ok := byID[ancestorID]
			if !ok {
				break
			}
			clipX := clipsAxis(ancestor.Styles["overflow-x"])
			clipY := clipsAxis(ancestor.Styles["overflow-y"])
			if (clipX && exceedsX(element.Bounds, ancestor.Bounds, policy.ClipTolerance)) ||
				(clipY && exceedsY(element.Bounds, ancestor.Bounds, policy.ClipTolerance)) {
				interactive := isInteractive(element)
				code := CodeContentClipped
				severity := evidence.SeverityMedium
				if interactive {
					code = CodeInteractiveClipped
					severity = evidence.SeverityHigh
				}
				issues = append(issues, evidence.RuntimeIssue{
					Code:       code,
					Severity:   severity,
					ElementIDs: []string{element.ID, ancestor.ID},
					Message:    fmt.Sprintf("%s %q extends outside clipping ancestor %q", elementLabel(element), element.ID, ancestor.ID),
				})
				break
			}
			ancestorID = ancestor.ParentID
		}
	}
	return issues
}

func verifyInteractiveState(elements []evidence.ElementRef, policy Policy) []evidence.RuntimeIssue {
	issues := make([]evidence.RuntimeIssue, 0)
	for _, element := range elements {
		if !isInteractive(element) || isDisabled(element) || ariaHidden(element) {
			continue
		}
		if !element.Visible {
			issues = append(issues, evidence.RuntimeIssue{
				Code: CodeInteractiveHidden, Severity: evidence.SeverityHigh,
				ElementIDs: []string{element.ID},
				Message: fmt.Sprintf("interactive %s %q is not visually available", elementLabel(element), element.ID),
			})
			continue
		}
		if strings.EqualFold(strings.TrimSpace(element.Styles["pointer-events"]), "none") {
			issues = append(issues, evidence.RuntimeIssue{
				Code: CodePointerEventsDisabled, Severity: evidence.SeverityHigh,
				ElementIDs: []string{element.ID},
				Message: fmt.Sprintf("interactive %s %q has pointer-events:none", elementLabel(element), element.ID),
			})
		}
		if targetSizeException(element) {
			continue
		}
		if element.Bounds.Width+1e-9 < policy.MinTargetWidth || element.Bounds.Height+1e-9 < policy.MinTargetHeight {
			issues = append(issues, evidence.RuntimeIssue{
				Code: CodeTargetTooSmall, Severity: evidence.SeverityMedium,
				ElementIDs: []string{element.ID},
				Message: fmt.Sprintf("interactive %s %q is %.1fx%.1fpx; minimum policy target is %.1fx%.1fpx", elementLabel(element), element.ID, element.Bounds.Width, element.Bounds.Height, policy.MinTargetWidth, policy.MinTargetHeight),
			})
		}
	}
	return issues
}

func verifyInteractiveOverlap(elements []evidence.ElementRef, byID map[string]evidence.ElementRef, policy Policy) []evidence.RuntimeIssue {
	targets := make([]evidence.ElementRef, 0)
	for _, element := range elements {
		if isInteractive(element) && element.Visible && !isDisabled(element) && !ariaHidden(element) {
			targets = append(targets, element)
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Bounds.X == targets[j].Bounds.X {
			return targets[i].ID < targets[j].ID
		}
		return targets[i].Bounds.X < targets[j].Bounds.X
	})

	issues := make([]evidence.RuntimeIssue, 0)
	for i := 0; i < len(targets); i++ {
		a := targets[i]
		aRight := a.Bounds.X + a.Bounds.Width
		for j := i + 1; j < len(targets); j++ {
			b := targets[j]
			if b.Bounds.X >= aRight-policy.OverlapMinPixels {
				break
			}
			if a.FrameID != b.FrameID || relatedByAncestry(a.ID, b.ID, byID) {
				continue
			}
			intersectionWidth := overlap1D(a.Bounds.X, aRight, b.Bounds.X, b.Bounds.X+b.Bounds.Width)
			intersectionHeight := overlap1D(a.Bounds.Y, a.Bounds.Y+a.Bounds.Height, b.Bounds.Y, b.Bounds.Y+b.Bounds.Height)
			if intersectionWidth < policy.OverlapMinPixels || intersectionHeight < policy.OverlapMinPixels {
				continue
			}
			intersectionArea := intersectionWidth * intersectionHeight
			smallerArea := math.Min(rectArea(a.Bounds), rectArea(b.Bounds))
			if smallerArea <= 0 || intersectionArea/smallerArea < policy.OverlapRatio {
				continue
			}
			ids := []string{a.ID, b.ID}
			sort.Strings(ids)
			issues = append(issues, evidence.RuntimeIssue{
				Code: CodeInteractiveOverlap, Severity: evidence.SeverityHigh,
				ElementIDs: ids,
				Message: fmt.Sprintf("interactive targets %q and %q overlap by %.0f%% of the smaller target", ids[0], ids[1], 100*intersectionArea/smallerArea),
			})
		}
	}
	return issues
}

func verifyStyleInvariants(elements []evidence.ElementRef, invariants []StyleInvariant) []evidence.RuntimeIssue {
	issues := make([]evidence.RuntimeIssue, 0)
	for _, invariant := range invariants {
		property := strings.TrimSpace(invariant.Property)
		if property == "" {
			continue
		}
		severity := invariant.Severity
		if severity == "" {
			severity = evidence.SeverityMedium
		}
		allowed := normalizedSet(invariant.Allowed)
		disallowed := normalizedSet(invariant.Disallowed)
		for _, element := range elements {
			if invariant.Tag != "" && !strings.EqualFold(element.Tag, invariant.Tag) {
				continue
			}
			if invariant.Role != "" && !strings.EqualFold(element.Role, invariant.Role) {
				continue
			}
			value, exists := element.Styles[property]
			if !exists {
				continue
			}
			normalized := strings.ToLower(strings.TrimSpace(value))
			violates := (len(allowed) > 0 && !allowed[normalized]) || disallowed[normalized]
			if !violates {
				continue
			}
			id := invariant.ID
			if id == "" {
				id = property
			}
			issues = append(issues, evidence.RuntimeIssue{
				Code: CodeStyleInvariant + "." + id,
				Severity: severity,
				ElementIDs: []string{element.ID},
				Message: fmt.Sprintf("%s %q violates style invariant %q: %s=%q", elementLabel(element), element.ID, id, property, value),
			})
		}
	}
	return issues
}

func isInteractive(element evidence.ElementRef) bool {
	switch strings.ToLower(element.Role) {
	case "button", "link", "checkbox", "radio", "slider", "spinbutton", "switch", "textbox", "combobox", "listbox", "menuitem", "option", "tab", "treeitem":
		return true
	}
	if _, ok := element.Attributes["onclick"]; ok {
		return true
	}
	if tabindex, ok := element.Attributes["tabindex"]; ok && strings.TrimSpace(tabindex) != "-1" {
		return true
	}
	return false
}

func targetSizeException(element evidence.ElementRef) bool {
	// WCAG 2.2 Target Size (Minimum) has an inline-text exception. Preserve it
	// rather than treating every ordinary inline link as a deterministic defect.
	return element.Role == "link" && strings.EqualFold(strings.TrimSpace(element.Styles["display"]), "inline") && element.Name != ""
}

func isDisabled(element evidence.ElementRef) bool {
	if _, ok := element.Attributes["disabled"]; ok {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(element.Attributes["aria-disabled"]), "true")
}

func ariaHidden(element evidence.ElementRef) bool {
	return strings.EqualFold(strings.TrimSpace(element.Attributes["aria-hidden"]), "true")
}

func isIgnoredForGeometry(element evidence.ElementRef) bool {
	if ariaHidden(element) {
		return true
	}
	switch strings.ToLower(element.Role) {
	case "presentation", "none":
		return true
	}
	return false
}

func clipsAxis(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "hidden", "clip":
		return true
	}
	return false
}

func exceedsX(inner, outer evidence.Rect, tolerance float64) bool {
	return inner.X < outer.X-tolerance || inner.X+inner.Width > outer.X+outer.Width+tolerance
}

func exceedsY(inner, outer evidence.Rect, tolerance float64) bool {
	return inner.Y < outer.Y-tolerance || inner.Y+inner.Height > outer.Y+outer.Height+tolerance
}

func relatedByAncestry(aID, bID string, byID map[string]evidence.ElementRef) bool {
	return isAncestor(aID, bID, byID) || isAncestor(bID, aID, byID)
}

func isAncestor(ancestorID, nodeID string, byID map[string]evidence.ElementRef) bool {
	seen := make(map[string]struct{})
	for nodeID != "" {
		if nodeID == ancestorID {
			return true
		}
		if _, cycle := seen[nodeID]; cycle {
			return false
		}
		seen[nodeID] = struct{}{}
		node, ok := byID[nodeID]
		if !ok {
			return false
		}
		nodeID = node.ParentID
	}
	return false
}

func overlap1D(a0, a1, b0, b1 float64) float64 {
	return math.Max(0, math.Min(a1, b1)-math.Max(a0, b0))
}

func rectArea(rect evidence.Rect) float64 {
	if rect.Width <= 0 || rect.Height <= 0 {
		return 0
	}
	return rect.Width * rect.Height
}

func elementLabel(element evidence.ElementRef) string {
	if element.Role != "" {
		if element.Name != "" {
			return element.Role + " " + strconvQuote(element.Name)
		}
		return element.Role
	}
	if element.Tag != "" {
		return "<" + element.Tag + ">"
	}
	return "element"
}

func strconvQuote(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	if len(value) > 80 {
		value = value[:77] + "..."
	}
	return fmt.Sprintf("%q", value)
}

func normalizedSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[strings.ToLower(strings.TrimSpace(value))] = true
	}
	return out
}

func sortIssues(issues []evidence.RuntimeIssue) {
	for i := range issues {
		sort.Strings(issues[i].ElementIDs)
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Code != issues[j].Code {
			return issues[i].Code < issues[j].Code
		}
		leftIDs := strings.Join(issues[i].ElementIDs, "\x00")
		rightIDs := strings.Join(issues[j].ElementIDs, "\x00")
		if leftIDs != rightIDs {
			return leftIDs < rightIDs
		}
		return issues[i].Message < issues[j].Message
	})
}
