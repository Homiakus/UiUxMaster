package verifier

import (
	"fmt"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

const (
	CodeA11yMissingAXNode = "a11y.interactive_missing_ax_node"
	CodeA11yIgnored       = "a11y.interactive_ignored"
	CodeA11yRoleMissing   = "a11y.interactive_role_missing"
	CodeA11yNameMissing   = "a11y.accessible_name_missing"
	CodeFontSetLoading    = "font.fontset_loading"
	CodeFontFaceError     = "font.face_error"
)

func verifyAccessibility(packet evidence.Packet) []evidence.RuntimeIssue {
	if len(packet.Accessibility) == 0 || len(packet.Elements) == 0 {
		return nil
	}
	byBackend := make(map[int64]evidence.AccessibilityNode, len(packet.Accessibility))
	for _, node := range packet.Accessibility {
		if node.BackendNodeID > 0 {
			byBackend[node.BackendNodeID] = node
		}
	}

	issues := make([]evidence.RuntimeIssue, 0)
	for _, element := range packet.Elements {
		if element.BackendNodeID <= 0 || !element.Visible || isDisabled(element) || ariaHidden(element) {
			continue
		}
		interactive := isInteractive(element)
		image := strings.EqualFold(element.Tag, "img")
		if !interactive && !image {
			continue
		}
		ax, ok := byBackend[element.BackendNodeID]
		if !ok {
			if interactive {
				issues = append(issues, evidence.RuntimeIssue{
					Code: CodeA11yMissingAXNode, Severity: evidence.SeverityHigh,
					ElementIDs: []string{element.ID},
					Message: fmt.Sprintf("interactive %s %q has no correlated accessibility node", elementLabel(element), element.ID),
				})
			}
			continue
		}
		if ax.Ignored {
			if interactive {
				issues = append(issues, evidence.RuntimeIssue{
					Code: CodeA11yIgnored, Severity: evidence.SeverityHigh,
					ElementIDs: []string{element.ID},
					Message: fmt.Sprintf("interactive %s %q is ignored by the accessibility tree", elementLabel(element), element.ID),
					Details: map[string]string{"reasons": strings.Join(ax.IgnoredReasons, ", ")},
				})
			}
			continue
		}

		role := strings.ToLower(strings.TrimSpace(ax.Role))
		if interactive && element.Role == "" && (role == "" || role == "generic" || role == "none" || role == "presentation") {
			issues = append(issues, evidence.RuntimeIssue{
				Code: CodeA11yRoleMissing, Severity: evidence.SeverityHigh,
				ElementIDs: []string{element.ID},
				Message: fmt.Sprintf("clickable element %q has no actionable accessibility role", element.ID),
			})
		}
		if requiresAccessibleName(role, element) && strings.TrimSpace(ax.Name) == "" {
			issues = append(issues, evidence.RuntimeIssue{
				Code: CodeA11yNameMissing, Severity: evidence.SeverityHigh,
				ElementIDs: []string{element.ID},
				Message: fmt.Sprintf("%s %q has an empty computed accessible name", elementLabel(element), element.ID),
			})
		}
	}
	return issues
}

func requiresAccessibleName(axRole string, element evidence.ElementRef) bool {
	role := axRole
	if role == "" {
		role = strings.ToLower(element.Role)
	}
	switch role {
	case "button", "link", "checkbox", "radio", "switch", "textbox", "combobox", "listbox", "slider", "spinbutton", "menuitem", "option", "tab", "treeitem", "img":
		return true
	}
	return false
}

func verifyFonts(fonts *evidence.FontEvidence) []evidence.RuntimeIssue {
	if fonts == nil {
		return nil
	}
	issues := make([]evidence.RuntimeIssue, 0)
	if strings.EqualFold(strings.TrimSpace(fonts.Status), "loading") {
		issues = append(issues, evidence.RuntimeIssue{
			Code: CodeFontSetLoading, Severity: evidence.SeverityMedium,
			Message: "document font set is still loading; typography evidence is not settled",
		})
	}
	for _, face := range fonts.Faces {
		if !strings.EqualFold(strings.TrimSpace(face.Status), "error") {
			continue
		}
		label := strings.TrimSpace(face.Family)
		if label == "" {
			label = "unknown font"
		}
		issues = append(issues, evidence.RuntimeIssue{
			Code: CodeFontFaceError, Severity: evidence.SeverityHigh,
			Message: fmt.Sprintf("web font %q failed to load", label),
			Details: map[string]string{
				"family": face.Family, "style": face.Style,
				"weight": face.Weight, "stretch": face.Stretch,
			},
		})
	}
	return issues
}
