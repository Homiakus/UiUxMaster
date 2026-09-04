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
			{
				ID: "root", FrameID: "main", Tag: "div", Bounds: evidence.Rect{X: 0, Y: 0, Width: 300, Height: 120}, Visible: true,
				Styles: map[string]string{"overflow-x": "hidden", "overflow-y": "hidden"},
			},
			{
				ID: "a", FrameID: "main", Tag: "button", Role: "button", Name: "A", ParentID: "root",
				Bounds: evidence.Rect{X: 280, Y: 10, Width: 30, Height: 20}, Visible: true,
				Styles: map[string]string{"display": "block", "pointer-events": "auto"},
			},
			{
				ID: "b", FrameID: "main", Tag: "button", Role: "button", Name: "B", ParentID: "root",
				Bounds: evidence.Rect{X: 285, Y: 10, Width: 30, Height: 20}, Visible: true,
				Styles: map[string]string{"display": "block", "pointer-events": "none"},
			},
		},
	}

	result := Verify(packet, DefaultPolicy())
	codes := issueCodes(result.Issues)
	wantCodes := []string{
		CodeInteractiveClipped,
		CodeInteractiveClipped,
		CodePointerEventsDisabled,
		CodeTargetTooSmall,
		CodeTargetTooSmall,
		CodeInteractiveOverlap,
		CodeViewportHorizontalOverflow,
	}
	if !reflect.DeepEqual(codes, sortedStrings(wantCodes)) {
		t.Fatalf("codes = %#v, want %#v; issues=%#v", codes, sortedStrings(wantCodes), result.Issues)
	}
}

func TestVerifyDoesNotFlagInlineTextLinkTargetSize(t *testing.T) {
	packet := evidence.Packet{Elements: []evidence.ElementRef{{
		ID: "link", Tag: "a", Role: "link", Name: "Terms", Visible: true,
		Bounds: evidence.Rect{Width: 18, Height: 16},
		Styles: map[string]string{"display": "inline", "pointer-events": "auto"},
	}}}
	result := Verify(packet, DefaultPolicy())
	for _, issue := range result.Issues {
		if issue.Code == CodeTargetTooSmall {
			t.Fatalf("inline text link should use target-size exception: %#v", result.Issues)
		}
	}
}

func TestVerifySkipsIntentionalDocumentHorizontalScroll(t *testing.T) {
	packet := evidence.Packet{
		Viewport: evidence.Viewport{Width: 320, Height: 640},
		Documents: []evidence.DocumentMetrics{{FrameID: "main", ContentWidth: 900, ContentHeight: 640}},
		Elements: []evidence.ElementRef{{
			ID: "html", FrameID: "main", Tag: "html", Visible: true,
			Bounds: evidence.Rect{Width: 320, Height: 640},
			Styles: map[string]string{"overflow-x": "auto"},
		}},
	}
	if result := Verify(packet, DefaultPolicy()); len(result.Issues) != 0 {
		t.Fatalf("issues = %#v", result.Issues)
	}
}

func TestVerifyDoesNotTreatAncestorChildAsOverlap(t *testing.T) {
	packet := evidence.Packet{Elements: []evidence.ElementRef{
		{
			ID: "outer", FrameID: "main", Role: "button", Visible: true,
			Bounds: evidence.Rect{X: 0, Y: 0, Width: 100, Height: 40},
			Styles: map[string]string{"display": "block"},
		},
		{
			ID: "inner", FrameID: "main", Role: "button", ParentID: "outer", Visible: true,
			Bounds: evidence.Rect{X: 10, Y: 5, Width: 80, Height: 30},
			Styles: map[string]string{"display": "block"},
		},
	}}
	for _, issue := range Verify(packet, DefaultPolicy()).Issues {
		if issue.Code == CodeInteractiveOverlap {
			t.Fatalf("ancestor/child geometry is not sibling target overlap: %#v", issue)
		}
	}
}

func TestVerifyCustomStyleInvariant(t *testing.T) {
	packet := evidence.Packet{Elements: []evidence.ElementRef{{
		ID: "danger", Tag: "button", Role: "button", Visible: true,
		Bounds: evidence.Rect{Width: 80, Height: 32},
		Styles: map[string]string{"position": "absolute"},
	}}}
	policy := DefaultPolicy()
	policy.StyleInvariants = []StyleInvariant{{
		ID: "button-position", Role: "button", Property: "position",
		Disallowed: []string{"absolute", "fixed"}, Severity: evidence.SeverityHigh,
	}}
	result := Verify(packet, policy)
	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeStyleInvariant+".button-position" {
			found = true
			if issue.Severity != evidence.SeverityHigh {
				t.Fatalf("severity = %q", issue.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("style invariant issue missing: %#v", result.Issues)
	}
}

func TestApplyPreservesExistingRuntimeIssuesAndSorts(t *testing.T) {
	packet := evidence.Packet{
		RuntimeIssues: []evidence.RuntimeIssue{{Code: "network.failed", Severity: evidence.SeverityHigh, Message: "request failed"}},
		Elements: []evidence.ElementRef{{
			ID: "hidden", Role: "button", Visible: false,
			Bounds: evidence.Rect{Width: 40, Height: 40},
		}},
	}
	Apply(&packet, DefaultPolicy())
	if len(packet.RuntimeIssues) != 2 {
		t.Fatalf("issues = %#v", packet.RuntimeIssues)
	}
	if packet.RuntimeIssues[0].Code != CodeInteractiveHidden || packet.RuntimeIssues[1].Code != "network.failed" {
		t.Fatalf("order = %#v", packet.RuntimeIssues)
	}
}

func issueCodes(issues []evidence.RuntimeIssue) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		out = append(out, issue.Code)
	}
	return out
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
