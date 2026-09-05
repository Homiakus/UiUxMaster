package fastcdp

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// DefaultEvidenceStyles is deliberately small: these properties are sufficient
// for deterministic geometry/visibility/interaction checks without turning
// DOMSnapshot into a full computed-style dump.
var DefaultEvidenceStyles = []string{
	"display",
	"position",
	"overflow-x",
	"overflow-y",
	"visibility",
	"opacity",
	"pointer-events",
	"cursor",
	"z-index",
	"font-size",
	"font-weight",
	"line-height",
}

func DefaultSnapshotOptions() SnapshotOptions {
	return SnapshotOptions{
		ComputedStyles:  append([]string(nil), DefaultEvidenceStyles...),
		IncludeDOMRects: true,
	}
}

type PacketOptions struct {
	RunID             string
	Scenario          string
	URL               string
	Viewport          evidence.Viewport
	Browser           BrowserVersion
	FidelityID        string
	Region            *CaptureRegionOptions
	ExpectedRevision  string
}

// ToPacket converts one stable CollectedEvidence result into the repository's
// protocol-independent evidence envelope. Large RGBA bytes remain in the local
// hot path; Packet stores only their dimensions, provenance digest and source
// ROI metadata.
func ToPacket(collected CollectedEvidence, options PacketOptions) evidence.Packet {
	packet := evidence.Packet{
		RunID:    options.RunID,
		URL:      options.URL,
		Scenario: options.Scenario,
		Epoch:    collected.Epoch,
		Viewport: options.Viewport,
		Renderer: evidence.RendererRef{
			Tier:       "L2",
			Name:       "fastcdp",
			Version:    options.Browser.Product,
			FidelityID: options.FidelityID,
		},
		Latency: evidence.RuntimeLatency{
			WaitEpochMS:     durationMS(collected.Timing.WaitEpoch),
			SnapshotMS:      durationMS(collected.Timing.Snapshot),
			PixelsMS:        durationMS(collected.Timing.Pixels),
			AccessibilityMS: durationMS(collected.Timing.Accessibility),
			FontsMS:         durationMS(collected.Timing.Fonts),
			DiagnosticsMS:   durationMS(collected.Timing.Diagnostics),
			TotalMS:         durationMS(collected.Timing.Total),
			Retries:         collected.Timing.Retries,
		},
	}
	if options.ExpectedRevision != "" || collected.Revision != "" {
		packet.Freshness = &evidence.RenderFreshness{
			Epoch:            collected.Epoch,
			ExpectedRevision: strings.TrimSpace(options.ExpectedRevision),
			ObservedRevision: strings.TrimSpace(collected.Revision),
		}
	}
	if packet.Viewport.Browser == "" {
		packet.Viewport.Browser = options.Browser.Product
	}

	if collected.Snapshot != nil {
		projectSnapshotIntoPacket(&packet, *collected.Snapshot)
	}
	if packet.URL == "" && len(packet.Documents) > 0 {
		packet.URL = packet.Documents[0].URL
	}
	if collected.Accessibility != nil {
		projectAccessibilityIntoPacket(&packet, *collected.Accessibility)
	}
	if collected.Fonts != nil {
		projectFontsIntoPacket(&packet, *collected.Fonts)
	}
	if collected.Diagnostics != nil {
		projectDiagnosticsIntoPacket(&packet, *collected.Diagnostics)
	}
	if collected.RGBA != nil {
		bounds := evidence.Rect{}
		if options.Region != nil {
			bounds = evidence.Rect{
				X: options.Region.X, Y: options.Region.Y,
				Width: options.Region.Width, Height: options.Region.Height,
			}
		}
		sum := sha256.Sum256(collected.RGBA.Pix)
		packet.Pixels = &evidence.PixelEvidence{
			Bounds:       bounds,
			Width:        collected.RGBA.Bounds().Dx(),
			Height:       collected.RGBA.Bounds().Dy(),
			EncodedBytes: collected.CaptureStats.EncodedBytes,
			DigestSHA256: fmt.Sprintf("%x", sum[:]),
		}
	}
	return packet
}

func projectSnapshotIntoPacket(packet *evidence.Packet, snapshot Snapshot) {
	for docIndex, doc := range snapshot.Documents {
		frameID := doc.FrameID
		if frameID == "" {
			frameID = fmt.Sprintf("document-%d", docIndex)
		}
		packet.Documents = append(packet.Documents, evidence.DocumentMetrics{
			FrameID:       frameID,
			URL:           doc.DocumentURL,
			ContentWidth:  doc.ContentWidth,
			ContentHeight: doc.ContentHeight,
		})

		idByNodeIndex := make(map[int]string, len(doc.Nodes))
		parentByNodeIndex := make(map[int]int, len(doc.Nodes))
		for _, node := range doc.Nodes {
			parentByNodeIndex[node.NodeIndex] = node.ParentIndex
			if node.NodeType != 1 {
				continue
			}
			idByNodeIndex[node.NodeIndex] = elementID(frameID, node)
		}
		for _, node := range doc.Nodes {
			if node.NodeType != 1 {
				continue
			}
			attrs := cloneStrings(node.Attributes)
			styles := cloneStrings(node.Styles)
			ref := evidence.ElementRef{
				ID:            idByNodeIndex[node.NodeIndex],
				FrameID:       frameID,
				BackendNodeID: node.BackendNodeID,
				Tag:           strings.ToLower(node.Name),
				Role:          semanticRole(node.Name, attrs),
				Name:          semanticName(node, attrs),
				Selector:      semanticSelector(attrs),
				Bounds: evidence.Rect{
					X: node.Bounds.X, Y: node.Bounds.Y,
					Width: node.Bounds.Width, Height: node.Bounds.Height,
				},
				Visible:    visuallyVisible(node.Bounds, styles),
				Clickable:  node.Clickable,
				Styles:     styles,
				Attributes: attrs,
				ParentID:   nearestProjectedParent(node.ParentIndex, parentByNodeIndex, idByNodeIndex),
			}
			packet.Elements = append(packet.Elements, ref)
		}
	}
}

func projectAccessibilityIntoPacket(packet *evidence.Packet, tree AXTree) {
	packet.Accessibility = make([]evidence.AccessibilityNode, 0, len(tree.Nodes))
	for _, node := range tree.Nodes {
		packet.Accessibility = append(packet.Accessibility, evidence.AccessibilityNode{
			ID:             node.ID,
			ParentID:       node.ParentID,
			ChildIDs:       append([]string(nil), node.ChildIDs...),
			BackendNodeID:  node.BackendDOMNodeID,
			FrameID:        node.FrameID,
			Ignored:        node.Ignored,
			IgnoredReasons: append([]string(nil), node.IgnoredReasons...),
			Role:           node.Role,
			Name:           node.Name,
			Description:    node.Description,
			Value:          node.Value,
			Properties:     cloneStrings(node.Properties),
		})
	}
	packet.AriaSnapshot = tree.TextSnapshot()
}

func projectFontsIntoPacket(packet *evidence.Packet, fonts FontState) {
	projected := &evidence.FontEvidence{Status: fonts.Status, Total: fonts.Total, Truncated: fonts.Truncated}
	projected.Faces = make([]evidence.FontFaceEvidence, 0, len(fonts.Faces))
	for _, face := range fonts.Faces {
		projected.Faces = append(projected.Faces, evidence.FontFaceEvidence{
			Family: face.Family, Style: face.Style, Weight: face.Weight,
			Stretch: face.Stretch, Status: face.Status,
		})
	}
	packet.Fonts = projected
}

func projectDiagnosticsIntoPacket(packet *evidence.Packet, diagnostics DiagnosticSnapshot) {
	packet.Diagnostics = &evidence.DiagnosticsEvidence{
		Complete:       diagnostics.Complete,
		DroppedMethods: append([]string(nil), diagnostics.DroppedMethods...),
	}
	for _, event := range diagnostics.Events {
		packet.RuntimeIssues = append(packet.RuntimeIssues, diagnosticRuntimeIssue(event))
	}
}

func diagnosticRuntimeIssue(event DiagnosticEvent) evidence.RuntimeIssue {
	issue := evidence.RuntimeIssue{
		Code:     string(event.Kind),
		Message:  event.Message,
		Severity: evidence.SeverityMedium,
		Details:  map[string]string{},
	}
	switch event.Kind {
	case DiagnosticRuntimeException:
		issue.Severity = evidence.SeverityCritical
	case DiagnosticConsole, DiagnosticLog:
		if event.Level == "error" || event.Level == "assert" {
			issue.Severity = evidence.SeverityHigh
		}
	case DiagnosticNetworkFailed:
		if event.Canceled {
			issue.Severity = evidence.SeverityLow
		} else if criticalResource(event.Resource) {
			issue.Severity = evidence.SeverityHigh
		}
	case DiagnosticHTTPError:
		if event.StatusCode >= 500 || criticalResource(event.Resource) {
			issue.Severity = evidence.SeverityHigh
		}
	}
	if event.URL != "" {
		issue.Details["url"] = event.URL
	}
	if event.RequestID != "" {
		issue.Details["request_id"] = event.RequestID
	}
	if event.Resource != "" {
		issue.Details["resource"] = event.Resource
	}
	if event.StatusCode != 0 {
		issue.Details["status"] = strconv.Itoa(event.StatusCode)
	}
	if event.Canceled {
		issue.Details["canceled"] = "true"
	}
	if len(issue.Details) == 0 {
		issue.Details = nil
	}
	return issue
}

func criticalResource(resource string) bool {
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "document", "script", "stylesheet", "font", "xhr", "fetch":
		return true
	}
	return false
}

func elementID(frameID string, node SnapshotNode) string {
	if node.BackendNodeID > 0 {
		return fmt.Sprintf("dom:%s:%d", frameID, node.BackendNodeID)
	}
	return fmt.Sprintf("dom:%s:node-%d", frameID, node.NodeIndex)
}

func nearestProjectedParent(parentIndex int, parentByNodeIndex map[int]int, idByNodeIndex map[int]string) string {
	seen := make(map[int]struct{})
	for parentIndex >= 0 {
		if id := idByNodeIndex[parentIndex]; id != "" {
			return id
		}
		if _, cycle := seen[parentIndex]; cycle {
			return ""
		}
		seen[parentIndex] = struct{}{}
		next, ok := parentByNodeIndex[parentIndex]
		if !ok {
			return ""
		}
		parentIndex = next
	}
	return ""
}

func semanticRole(tag string, attrs map[string]string) string {
	if role := strings.ToLower(strings.TrimSpace(attrs["role"])); role != "" {
		return role
	}
	tag = strings.ToLower(tag)
	switch tag {
	case "button":
		return "button"
	case "a":
		if attrs["href"] != "" {
			return "link"
		}
	case "textarea":
		return "textbox"
	case "select":
		return "combobox"
	case "img":
		return "img"
	case "nav":
		return "navigation"
	case "main":
		return "main"
	case "input":
		switch strings.ToLower(attrs["type"]) {
		case "button", "submit", "reset", "image":
			return "button"
		case "checkbox":
			return "checkbox"
		case "radio":
			return "radio"
		case "range":
			return "slider"
		case "hidden":
			return ""
		default:
			return "textbox"
		}
	}
	return ""
}

func semanticName(node SnapshotNode, attrs map[string]string) string {
	for _, key := range []string{"aria-label", "alt", "title", "placeholder"} {
		if value := normalizeWhitespace(attrs[key]); value != "" {
			return value
		}
	}
	if strings.EqualFold(node.Name, "input") {
		if value := normalizeWhitespace(attrs["value"]); value != "" {
			return value
		}
	}
	if value := normalizeWhitespace(node.Text); value != "" {
		return value
	}
	return normalizeWhitespace(node.Value)
}

var simpleIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

func semanticSelector(attrs map[string]string) string {
	if testID := attrs["data-testid"]; testID != "" {
		return `[data-testid="` + escapeAttributeValue(testID) + `"]`
	}
	if id := attrs["id"]; id != "" {
		if simpleIdentifier.MatchString(id) {
			return "#" + id
		}
		return `[id="` + escapeAttributeValue(id) + `"]`
	}
	return ""
}

func escapeAttributeValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func visuallyVisible(bounds Rect, styles map[string]string) bool {
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return false
	}
	if strings.EqualFold(styles["display"], "none") {
		return false
	}
	switch strings.ToLower(styles["visibility"]) {
	case "hidden", "collapse":
		return false
	}
	if opacity := strings.TrimSpace(styles["opacity"]); opacity != "" {
		if parsed, err := strconv.ParseFloat(opacity, 64); err == nil && parsed <= 0.001 {
			return false
		}
	}
	return true
}

func cloneStrings(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func durationMS(value time.Duration) float64 {
	return float64(value.Nanoseconds()) / 1e6
}
