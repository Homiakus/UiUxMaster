package verifier

import (
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestVerifyDeterministicFindsMissingAccessibleName(t *testing.T) {
	packet := evidence.Packet{
		Elements: []evidence.ElementRef{{
			ID: "save", BackendNodeID: 42, Tag: "button", Role: "button", Visible: true,
			Bounds: evidence.Rect{Width: 80, Height: 32},
		}},
		Accessibility: []evidence.AccessibilityNode{{
			ID: "ax-1", BackendNodeID: 42, Role: "button", Name: "",
		}},
	}
	result := VerifyDeterministic(packet, DefaultPolicy())
	if !hasIssue(result.Issues, CodeA11yNameMissing) {
		t.Fatalf("issues = %#v", result.Issues)
	}
}

func TestVerifyDeterministicFindsClickableGenericRole(t *testing.T) {
	packet := evidence.Packet{
		Elements: []evidence.ElementRef{{
			ID: "custom", BackendNodeID: 7, Tag: "div", Clickable: true, Visible: true,
			Bounds: evidence.Rect{Width: 80, Height: 32},
		}},
		Accessibility: []evidence.AccessibilityNode{{
			ID: "ax-7", BackendNodeID: 7, Role: "generic", Name: "Open",
		}},
	}
	result := VerifyDeterministic(packet, DefaultPolicy())
	if !hasIssue(result.Issues, CodeA11yRoleMissing) {
		t.Fatalf("issues = %#v", result.Issues)
	}
}

func TestVerifyDeterministicSkipsDecorativeIgnoredImage(t *testing.T) {
	packet := evidence.Packet{
		Elements: []evidence.ElementRef{{
			ID: "decorative", BackendNodeID: 8, Tag: "img", Visible: true,
			Bounds: evidence.Rect{Width: 100, Height: 100},
		}},
		Accessibility: []evidence.AccessibilityNode{{
			ID: "ax-8", BackendNodeID: 8, Ignored: true, Role: "none",
		}},
	}
	result := VerifyDeterministic(packet, DefaultPolicy())
	if hasIssue(result.Issues, CodeA11yNameMissing) || hasIssue(result.Issues, CodeA11yIgnored) {
		t.Fatalf("decorative image should not be flagged: %#v", result.Issues)
	}
}

func TestVerifyDeterministicFindsFontError(t *testing.T) {
	packet := evidence.Packet{Fonts: &evidence.FontEvidence{
		Status: "loaded",
		Faces: []evidence.FontFaceEvidence{{Family: "Brand Sans", Status: "error"}},
	}}
	result := VerifyDeterministic(packet, DefaultPolicy())
	if !hasIssue(result.Issues, CodeFontFaceError) {
		t.Fatalf("issues = %#v", result.Issues)
	}
}
