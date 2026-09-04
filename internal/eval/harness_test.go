package eval_test

import (
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/eval"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestAdversarialEvalSuite_RecallAndLocalization(t *testing.T) {
	basePacket := evidence.Packet{
		Viewport: evidence.Viewport{Width: 1000, Height: 800},
		Documents: []evidence.DocumentMetrics{
			{FrameID: "main", ContentWidth: 1000, ContentHeight: 800},
		},
		Elements: []evidence.ElementRef{
			{
				ID:        "header",
				Tag:       "header",
				Visible:   true,
				Bounds:    evidence.Rect{X: 0, Y: 0, Width: 1000, Height: 80},
				Styles:    map[string]string{"display": "block", "position": "static"},
			},
			{
				ID:        "title",
				Tag:       "h1",
				Role:      "heading",
				Name:      "UiUxMaster Enterprise",
				Visible:   true,
				Bounds:    evidence.Rect{X: 20, Y: 20, Width: 400, Height: 40},
				Styles:    map[string]string{"display": "block"},
			},
			{
				ID:        "btn-submit",
				Tag:       "button",
				Role:      "button",
				Name:      "Save Changes",
				Visible:   true,
				Clickable: true,
				Bounds:    evidence.Rect{X: 20, Y: 120, Width: 140, Height: 48},
				Styles:    map[string]string{"display": "block", "pointer-events": "auto"},
			},
			{
				ID:        "btn-cancel",
				Tag:       "button",
				Role:      "button",
				Name:      "Cancel",
				Visible:   true,
				Clickable: true,
				Bounds:    evidence.Rect{X: 180, Y: 120, Width: 120, Height: 48},
				Styles:    map[string]string{"display": "block", "pointer-events": "auto"},
			},
		},
		Accessibility: []evidence.AccessibilityNode{
			{ID: "ax-title", Role: "heading", Name: "UiUxMaster Enterprise"},
			{ID: "ax-submit", Role: "button", Name: "Save Changes"},
			{ID: "ax-cancel", Role: "button", Name: "Cancel"},
		},
	}

	cases := []eval.EvalCase{
		{
			Name:       "Horizontal Viewport Overflow Injection",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "overflow-1",
					Kind:            eval.DefectViewportOverflow,
					Description:     "Document content width exceeds viewport by 250px",
					TargetElementID: "",
					Mutate: func(p *evidence.Packet) {
						p.Documents[0].ContentWidth = 1250
					},
				},
			},
		},
		{
			Name:       "Interactive Touch Target Sizing Defect",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "tiny-target-1",
					Kind:            eval.DefectTargetTooSmall,
					Description:     "Button size shrunk to 12x12px (below 24x24px minimum)",
					TargetElementID: "btn-tiny",
					Mutate: func(p *evidence.Packet) {
						p.Elements = append(p.Elements, evidence.ElementRef{
							ID:        "btn-tiny",
							Tag:       "button",
							Role:      "button",
							Name:      "Info",
							Visible:   true,
							Clickable: true,
							Bounds:    evidence.Rect{X: 320, Y: 120, Width: 12, Height: 12},
							Styles:    map[string]string{"display": "block"},
						})
					},
				},
			},
		},
		{
			Name:       "Interactive Target Overlap Defect",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "overlap-1",
					Kind:            eval.DefectInteractiveOverlap,
					Description:     "Two action buttons positioned with 80% overlapping geometry",
					TargetElementID: "btn-overlap-a",
					Mutate: func(p *evidence.Packet) {
						p.Elements = append(p.Elements,
							evidence.ElementRef{
								ID:        "btn-overlap-a",
								Tag:       "button",
								Role:      "button",
								Name:      "Option A",
								Visible:   true,
								Clickable: true,
								Bounds:    evidence.Rect{X: 400, Y: 120, Width: 80, Height: 40},
								Styles:    map[string]string{"display": "block"},
							},
							evidence.ElementRef{
								ID:        "btn-overlap-b",
								Tag:       "button",
								Role:      "button",
								Name:      "Option B",
								Visible:   true,
								Clickable: true,
								Bounds:    evidence.Rect{X: 420, Y: 120, Width: 80, Height: 40},
								Styles:    map[string]string{"display": "block"},
							},
						)
					},
				},
			},
		},
		{
			Name:       "Accessibility Focus Sequence Disruption",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "tabindex-positive",
					Kind:            eval.DefectFocusSequenceAnomaly,
					Description:     "Positive tabindex=5 disrupting natural sequential keyboard focus",
					TargetElementID: "btn-tab",
					Mutate: func(p *evidence.Packet) {
						p.Elements = append(p.Elements, evidence.ElementRef{
							ID:         "btn-tab",
							Tag:        "button",
							Role:       "button",
							Name:       "High Priority Action",
							Visible:    true,
							Clickable:  true,
							Attributes: map[string]string{"tabindex": "5"},
							Bounds:     evidence.Rect{X: 520, Y: 120, Width: 100, Height: 40},
							Styles:     map[string]string{"display": "block"},
						})
					},
				},
			},
		},
		{
			Name:       "Duplicate DOM Element IDs",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "dup-id-1",
					Kind:            eval.DefectDuplicateDOMID,
					Description:     "Two distinct components share id='primary-action'",
					TargetElementID: "action-1",
					Mutate: func(p *evidence.Packet) {
						p.Elements = append(p.Elements,
							evidence.ElementRef{
								ID:         "action-1",
								Tag:        "button",
								Attributes: map[string]string{"id": "primary-action"},
								Bounds:     evidence.Rect{X: 640, Y: 120, Width: 60, Height: 30},
							},
							evidence.ElementRef{
								ID:         "action-2",
								Tag:        "button",
								Attributes: map[string]string{"id": "primary-action"},
								Bounds:     evidence.Rect{X: 710, Y: 120, Width: 60, Height: 30},
							},
						)
					},
				},
			},
		},
		{
			Name:       "Severe Heading Text Truncation",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "trunc-1",
					Kind:            eval.DefectTextTruncation,
					Description:     "Heading text clipped to 10px with text-overflow:ellipsis",
					TargetElementID: "truncated-h2",
					Mutate: func(p *evidence.Packet) {
						p.Elements = append(p.Elements, evidence.ElementRef{
							ID:      "truncated-h2",
							Tag:     "h2",
							Role:    "heading",
							Name:    "Detailed Performance Summary",
							Visible: true,
							Bounds:  evidence.Rect{X: 20, Y: 200, Width: 10, Height: 24},
							Styles:  map[string]string{"text-overflow": "ellipsis", "overflow": "hidden", "white-space": "nowrap"},
						})
					},
				},
			},
		},
		{
			Name:       "Missing Accessible Name for Interactive Control",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "a11y-unnamed",
					Kind:            eval.DefectMissingA11yName,
					Description:     "Clickable icon button without aria-label or accessible text",
					TargetElementID: "btn-icon-only",
					Mutate: func(p *evidence.Packet) {
						p.Elements = append(p.Elements, evidence.ElementRef{
							ID:            "btn-icon-only",
							Tag:           "button",
							Visible:       true,
							Clickable:     true,
							BackendNodeID: 999,
							Bounds:        evidence.Rect{X: 800, Y: 20, Width: 40, Height: 40},
							Styles:        map[string]string{"display": "block"},
						})
						p.Accessibility = append(p.Accessibility, evidence.AccessibilityNode{
							ID:            "ax-999",
							BackendNodeID: 999,
							Role:          "button",
							Name:          "", // empty accessible name
						})
					},
				},
			},
		},
		{
			Name:       "Fixed Header Obstructs Interactive Control",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "fixed-obstruction",
					Kind:            eval.DefectFixedObstruction,
					Description:     "Fixed sticky navigation overlaying an interactive button underneath",
					TargetElementID: "fixed-nav",
					Mutate: func(p *evidence.Packet) {
						p.Elements = append(p.Elements, evidence.ElementRef{
							ID:      "fixed-nav",
							Tag:     "nav",
							Visible: true,
							Bounds:  evidence.Rect{X: 0, Y: 100, Width: 300, Height: 80},
							Styles:  map[string]string{"position": "fixed", "display": "block"},
						})
					},
				},
			},
		},
		{
			Name:       "Multiple H1 Heading Hierarchy Violation",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "dup-h1",
					Kind:            eval.DefectHeadingHierarchy,
					Description:     "Second H1 heading in single page violating semantic hierarchy",
					TargetElementID: "title-second",
					Mutate: func(p *evidence.Packet) {
						p.Elements = append(p.Elements, evidence.ElementRef{
							ID:      "title-second",
							Tag:     "h1",
							Role:    "heading",
							Name:    "Secondary Headline",
							Visible: true,
							Bounds:  evidence.Rect{X: 20, Y: 300, Width: 300, Height: 40},
							Styles:  map[string]string{"display": "block"},
						})
					},
				},
			},
		},
		{
			Name:       "Pointer Events Disabled on Interactive Element",
			BasePacket: basePacket,
			InjectedDefects: []eval.DefectInjection{
				{
					ID:              "pointer-none",
					Kind:            eval.DefectPointerDisabled,
					Description:     "Submit button styled with pointer-events: none",
					TargetElementID: "btn-submit-disabled",
					Mutate: func(p *evidence.Packet) {
						p.Elements = append(p.Elements, evidence.ElementRef{
							ID:        "btn-submit-disabled",
							Tag:       "button",
							Role:      "button",
							Name:      "Disabled Click",
							Visible:   true,
							Clickable: true,
							Bounds:    evidence.Rect{X: 20, Y: 400, Width: 120, Height: 44},
							Styles:    map[string]string{"display": "block", "pointer-events": "none"},
						})
					},
				},
			},
		},
	}

	harness := eval.NewHarness()
	report := harness.Run(cases)

	t.Logf("Adversarial Eval Results:")
	t.Logf("  Total Injected:  %d", report.TotalInjected)
	t.Logf("  Total Detected:  %d", report.TotalDetected)
	t.Logf("  Total Localized: %d", report.TotalLocalized)
	t.Logf("  Recall:          %.2f%%", report.Recall*100)
	t.Logf("  Precision:       %.2f%%", report.Precision*100)
	t.Logf("  Localization:    %.2f%%", report.LocalizationRate*100)

	for kind, stats := range report.ByKind {
		t.Logf("  [%s] Injected: %d, Detected: %d, Localized: %d, Recall: %.1f%%",
			kind, stats.Injected, stats.Detected, stats.Localized, stats.Recall*100)
	}

	if report.Recall < 0.95 {
		t.Errorf("expected detection recall >= 95%%, got %.1f%%", report.Recall*100)
	}
	if report.LocalizationRate < 0.95 {
		t.Errorf("expected defect localization rate >= 95%%, got %.1f%%", report.LocalizationRate*100)
	}
}
