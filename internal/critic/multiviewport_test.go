package critic

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type mockMultiViewportCollector struct {
	elementsByVP map[int][]evidence.ElementRef
	docsByVP     map[int][]evidence.DocumentMetrics
}

func (m *mockMultiViewportCollector) CollectForViewport(_ context.Context, vp evidence.Viewport) (evidence.Packet, error) {
	docs := m.docsByVP[vp.Width]
	if len(docs) == 0 {
		docs = []evidence.DocumentMetrics{{URL: "https://hydropilot.local/dashboard", ContentWidth: float64(vp.Width), ContentHeight: float64(vp.Height)}}
	}
	return evidence.Packet{
		RunID:        "run-multivp",
		Viewport:     vp,
		Documents:    docs,
		Elements:     m.elementsByVP[vp.Width],
		AriaSnapshot: "root: main",
	}, nil
}

func TestMultiViewportAuditor_LocalizationAndDefectDetection(t *testing.T) {
	ctx := context.Background()

	// Setup a responsive layout that behaves differently across viewports:
	// - On mobile (375): horizontal overflow (content width 420 > 375) + touch target too small (< 24px)
	// - On tablet (768): normal layout, but missing aria-label on icon button
	// - On desktop (1440): clean, fully accessible layout
	collector := &mockMultiViewportCollector{
		docsByVP: map[int][]evidence.DocumentMetrics{
			375:  {{URL: "https://hydropilot.local/dashboard", ContentWidth: 420, ContentHeight: 667}}, // overflow on mobile!
			768:  {{URL: "https://hydropilot.local/dashboard", ContentWidth: 768, ContentHeight: 1024}},
			1440: {{URL: "https://hydropilot.local/dashboard", ContentWidth: 1440, ContentHeight: 900}},
		},
		elementsByVP: map[int][]evidence.ElementRef{
			375: {
				{ID: "h1-title", Tag: "h1", Role: "heading", Attributes: map[string]string{"aria-level": "1"}, Visible: true},
				{ID: "btn-mobile-tiny", Role: "button", Bounds: evidence.Rect{X: 10, Y: 10, Width: 18, Height: 18}, Clickable: true, Visible: true, Name: "Compact action"},
			},
			768: {
				{ID: "h1-title", Tag: "h1", Role: "heading", Attributes: map[string]string{"aria-level": "1"}, Visible: true},
				{ID: "btn-icon-unnamed", Role: "button", Bounds: evidence.Rect{X: 20, Y: 20, Width: 44, Height: 44}, Clickable: true, Visible: true, Name: ""}, // missing accessible name!
			},
			1440: {
				{ID: "h1-title", Tag: "h1", Role: "heading", Attributes: map[string]string{"aria-level": "1"}, Visible: true},
				{ID: "btn-desktop-ok", Role: "button", Bounds: evidence.Rect{X: 30, Y: 30, Width: 120, Height: 40}, Clickable: true, Visible: true, Name: "Primary Action"},
			},
		},
	}

	auditor := NewMultiViewportAuditor(New(), verifier.DefaultPolicy())
	report, err := auditor.Audit(ctx, collector, MultiViewportRequest{
		RunID:   "audit-hydropilot-mvp",
		Profile: design.FindProfile("saas-modern"),
	})
	if err != nil {
		t.Fatalf("auditor.Audit failed: %v", err)
	}

	// 1. Verify 3 standard viewports were evaluated
	if len(report.Viewports) != 3 {
		t.Fatalf("expected 3 viewports evaluated, got %d", len(report.Viewports))
	}

	// 2. Verify mobile viewport (375) detected overflow and touch target issues
	mobileVP := report.Viewports[0]
	if mobileVP.Viewport.Width != 375 {
		t.Fatalf("viewports[0].Width = %d, want 375", mobileVP.Viewport.Width)
	}

	hasOverflow := false
	hasSmallTarget := false
	for _, iss := range mobileVP.Verification.Issues {
		if iss.Code == verifier.CodeViewportHorizontalOverflow {
			hasOverflow = true
		}
		if iss.Code == verifier.CodeTargetTooSmall {
			hasSmallTarget = true
		}
	}
	if !hasOverflow {
		t.Fatal("expected mobile horizontal overflow issue")
	}
	if !hasSmallTarget {
		t.Fatal("expected mobile target too small issue")
	}

	// 3. Verify tablet viewport (768) detected unnamed accessible button
	tabletVP := report.Viewports[1]
	hasA11yNameIssue := false
	for _, f := range tabletVP.Critique.Findings {
		if f.Category == "name" && f.Axis == "accessibility" {
			hasA11yNameIssue = true
		}
	}
	if !hasA11yNameIssue {
		t.Fatal("expected tablet accessible name missing finding")
	}

	// 4. Verify 100% element defect localization index
	if len(report.LocalizedByEl["btn-mobile-tiny"]) == 0 {
		t.Fatal("expected findings localized to #btn-mobile-tiny")
	}
	if len(report.LocalizedByEl["btn-icon-unnamed"]) == 0 {
		t.Fatal("expected findings localized to #btn-icon-unnamed")
	}

	t.Logf("Multi-viewport audit summary: TotalFindings=%d, HardViolations=%d, Score=%.1f, LocalizedElements=%d, Duration=%v",
		report.TotalFindings, report.HardViolations, report.GroundedScore, len(report.LocalizedByEl), report.TotalDuration)
}
