package verifier

import (
	"reflect"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestVerifyFindsOverflowClippingSmallTargetsOverlapAndPointerEvents(t *testing.T) {
	packet := evidence.Packet{
		Viewport: evidence.Viewport{Width: 320, Height: 640},
		Documents: []evidence.DocumentMetrics{{FrameID: "main", ContentWidth: 360, ContentHeight: 640}},
		Elements: []evidence.ElementRef{
			{ID: "root", FrameID: "main", Tag: "div", Bounds: evidence.Rect{X: 0, Y: 0, Width: 300, Height: 120}, Visible: true, Styles: map[string]string{"overflow-x": "hidden", "overflow-y": "hidden"}},
			{ID: "a", FrameID: "main", Tag: "button", Role: "button", Name: "A", ParentID: "root", Bounds: evidence.Rect{X: 280, Y: 10, Width: 30, Height: 20}, Visible: true, Styles: map[string]string{"display": "block", "pointer-events": "auto"}},
			{ID: "b", FrameID: "main", Tag: "button", Role: "button", Name: "B", ParentID: "root", Bounds: evidence.Rect{X: 285, Y: 10, Width: 30, Height: 20}, Visible: true, Styles: map[string]string{"display": "block", "pointer-events": "none"}},
		},
	}
	result := Verify(packet, DefaultPolicy())
	codes := issueCodes(result.Issues)
	wantCodes := []string{CodeInteractiveClipped, CodeInteractiveClipped, CodePointerEventsDisabled, CodeTargetTooSmall, CodeTargetTooSmall, CodeInteractiveOverlap, CodeViewportHorizontalOverflow}
	if !reflect.DeepEqual(codes, sortedStrings(wantCodes)) {
		t.Fatalf("codes = %#v, want %#v; issues=%#v", codes, sortedStrings(wantCodes), result.Issues)
	}
}

func TestVerifyBlinkClickableCustomTargetIsInteractive(t *testing.T) {
	packet := evidence.Packet{Elements: []evidence.ElementRef{{
		ID: "custom", Tag: "div", Clickable: true, Visible: true,
		Bounds: evidence.Rect{Width: 12, Height: 12},
		Styles: map[string]string{"display": "block", "pointer-events": "auto"},
	}}}
	result := Verify(packet, DefaultPolicy())
	if !hasIssue(result.Issues, CodeTargetTooSmall) {
		t.Fatalf("Blink-clickable custom target was not verified: %#v", result.Issues)
	}
}

func TestVerifyDoesNotFlagInlineTextLinkTargetSize(t *testing.T) {
	packet := evidence.Packet{Elements: []evidence.ElementRef{{ID: "link", Tag: "a", Role: "link", Name: "Terms", Visible: true, Bounds: evidence.Rect{Width: 18, Height: 16}, Styles: map[string]string{"display": "inline", "pointer-events": "auto"}}}}
	if result := Verify(packet, DefaultPolicy()); hasIssue(result.Issues, CodeTargetTooSmall) {
		t.Fatalf("inline text link should use target-size exception: %#v", result.Issues)
	}
}

func TestVerifySkipsIntentionalDocumentHorizontalScroll(t *testing.T) {
	packet := evidence.Packet{Viewport: evidence.Viewport{Width: 320, Height: 640}, Documents: []evidence.DocumentMetrics{{FrameID: "main", ContentWidth: 900, ContentHeight: 640}}, Elements: []evidence.ElementRef{{ID: "html", FrameID: "main", Tag: "html", Visible: true, Bounds: evidence.Rect{Width: 320, Height: 640}, Styles: map[string]string{"overflow-x": "auto"}}}}
	if result := Verify(packet, DefaultPolicy()); len(result.Issues) != 0 {
		t.Fatalf("issues = %#v", result.Issues)
	}
}

func TestVerifyDoesNotTreatAncestorChildAsOverlap(t *testing.T) {
	packet := evidence.Packet{Elements: []evidence.ElementRef{
		{ID: "outer", FrameID: "main", Role: "button", Visible: true, Bounds: evidence.Rect{X: 0, Y: 0, Width: 100, Height: 40}, Styles: map[string]string{"display": "block"}},
		{ID: "inner", FrameID: "main", Role: "button", ParentID: "outer", Visible: true, Bounds: evidence.Rect{X: 10, Y: 5, Width: 80, Height: 30}, Styles: map[string]string{"display": "block"}},
	}}
	if result := Verify(packet, DefaultPolicy()); hasIssue(result.Issues, CodeInteractiveOverlap) {
		t.Fatalf("ancestor/child geometry is not sibling target overlap: %#v", result.Issues)
	}
}

func TestVerifyCustomStyleInvariant(t *testing.T) {
	packet := evidence.Packet{Elements: []evidence.ElementRef{{ID: "danger", Tag: "button", Role: "button", Visible: true, Bounds: evidence.Rect{Width: 80, Height: 32}, Styles: map[string]string{"position": "absolute"}}}}
	policy := DefaultPolicy()
	policy.StyleInvariants = []StyleInvariant{{ID: "button-position", Role: "button", Property: "position", Disallowed: []string{"absolute", "fixed"}, Severity: evidence.SeverityHigh}}
	result := Verify(packet, policy)
	if !hasIssue(result.Issues, CodeStyleInvariant+".button-position") {
		t.Fatalf("style invariant issue missing: %#v", result.Issues)
	}
}

func TestApplyPreservesExistingRuntimeIssuesSortsAndIsIdempotent(t *testing.T) {
	packet := evidence.Packet{RuntimeIssues: []evidence.RuntimeIssue{{Code: "network.failed", Severity: evidence.SeverityHigh, Message: "request failed"}}, Elements: []evidence.ElementRef{{ID: "hidden", Role: "button", Visible: false, Bounds: evidence.Rect{Width: 40, Height: 40}}}}
	Apply(&packet, DefaultPolicy())
	Apply(&packet, DefaultPolicy())
	if len(packet.RuntimeIssues) != 2 {
		t.Fatalf("issues after repeated apply = %#v", packet.RuntimeIssues)
	}
	if packet.RuntimeIssues[0].Code != CodeInteractiveHidden || packet.RuntimeIssues[1].Code != "network.failed" {
		t.Fatalf("order = %#v", packet.RuntimeIssues)
	}
}

func TestVerifyFixedStickyObstruction(t *testing.T) {
	packet := evidence.Packet{
		Elements: []evidence.ElementRef{
			{ID: "nav", Tag: "nav", Visible: true, Bounds: evidence.Rect{X: 0, Y: 0, Width: 320, Height: 60}, Styles: map[string]string{"position": "fixed"}},
			{ID: "btn", Tag: "button", Role: "button", Visible: true, Bounds: evidence.Rect{X: 20, Y: 10, Width: 100, Height: 40}, Styles: map[string]string{"position": "static"}},
		},
	}
	res := Verify(packet, DefaultPolicy())
	if !hasIssue(res.Issues, CodeFixedStickyObstruction) {
		t.Fatalf("expected fixed/sticky obstruction issue, got %#v", res.Issues)
	}
}

func TestVerifyFocusSequencePositiveTabindex(t *testing.T) {
	packet := evidence.Packet{
		Elements: []evidence.ElementRef{
			{ID: "btn1", Tag: "button", Role: "button", Visible: true, Attributes: map[string]string{"tabindex": "5"}, Bounds: evidence.Rect{Width: 50, Height: 50}},
			{ID: "btn2", Tag: "button", Role: "button", Visible: true, Attributes: map[string]string{"tabindex": "0"}, Bounds: evidence.Rect{Width: 50, Height: 50}},
		},
	}
	res := Verify(packet, DefaultPolicy())
	if !hasIssue(res.Issues, CodeFocusSequenceAnomaly) {
		t.Fatalf("expected focus sequence anomaly for positive tabindex, got %#v", res.Issues)
	}
}

func TestVerifyDuplicateDOMIDs(t *testing.T) {
	packet := evidence.Packet{
		Elements: []evidence.ElementRef{
			{ID: "el-1", Tag: "div", Attributes: map[string]string{"id": "main-content"}},
			{ID: "el-2", Tag: "section", Attributes: map[string]string{"id": "main-content"}},
			{ID: "el-3", Tag: "p", Attributes: map[string]string{"id": "unique-p"}},
		},
	}
	res := Verify(packet, DefaultPolicy())
	if !hasIssue(res.Issues, CodeDuplicateDOMID) {
		t.Fatalf("expected duplicate DOM ID issue, got %#v", res.Issues)
	}
}

func TestVerifyTextTruncationAnomaly(t *testing.T) {
	packet := evidence.Packet{
		Elements: []evidence.ElementRef{
			{
				ID:      "hero-title",
				Tag:     "h1",
				Role:    "heading",
				Name:    "Welcome to UiUxMaster Design Platform",
				Visible: true,
				Bounds:  evidence.Rect{Width: 15, Height: 24},
				Styles:  map[string]string{"text-overflow": "ellipsis", "overflow": "hidden", "white-space": "nowrap"},
			},
		},
	}
	res := Verify(packet, DefaultPolicy())
	if !hasIssue(res.Issues, CodeTextTruncationAnomaly) {
		t.Fatalf("expected text truncation anomaly for severely clipped heading, got %#v", res.Issues)
	}
}

func hasIssue(issues []evidence.RuntimeIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code { return true }
	}
	return false
}

func issueCodes(issues []evidence.RuntimeIssue) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues { out = append(out, issue.Code) }
	return out
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] { out[i], out[j] = out[j], out[i] }
		}
	}
	return out
}
