package critic_test

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/critic"
	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestLocalSemanticCritic_CleanPagePasses(t *testing.T) {
	c := critic.New()

	packet := evidence.Packet{
		RunID:    "clean-run",
		Viewport: evidence.Viewport{Width: 1280, Height: 800},
		Documents: []evidence.DocumentMetrics{
			{ContentWidth: 1280, ContentHeight: 800},
		},
		Elements: []evidence.ElementRef{
			{ID: "heading", Tag: "h1", Name: "Title", Visible: true},
			{ID: "btn", Tag: "button", Role: "button", Name: "Click Me", Visible: true, Clickable: true},
		},
		Fonts: &evidence.FontEvidence{Status: "loaded"},
	}

	req := critic.CritiqueRequest{
		RunID:   "clean-run",
		Level:   design.LevelPage,
		Profile: design.FindProfile("editorial-minimal"),
		Packet:  packet,
	}

	pass, err := c.Critique(context.Background(), req)
	if err != nil {
		t.Fatalf("Critique failed: %v", err)
	}

	if len(pass.Findings) != 0 {
		t.Errorf("expected 0 findings on clean page, got %d: %+v", len(pass.Findings), pass.Findings)
	}
	if pass.HardViolations != 0 {
		t.Errorf("expected 0 hard violations, got %d", pass.HardViolations)
	}
	if pass.GroundedScore != 10.0 {
		t.Errorf("expected grounded score 10.0, got %f", pass.GroundedScore)
	}
}

func TestLocalSemanticCritic_DetectsDefectsAndGeneratesHypotheses(t *testing.T) {
	c := critic.New()

	packet := evidence.Packet{
		RunID:    "defective-run",
		Viewport: evidence.Viewport{Width: 1000, Height: 800},
		Documents: []evidence.DocumentMetrics{
			{ContentWidth: 1150, ContentHeight: 800}, // Viewport overflow
		},
		Elements: []evidence.ElementRef{
			// No H1 element present
			{ID: "icon-btn", Tag: "button", Role: "button", Name: "", Visible: true, Clickable: true}, // Missing accessible name
		},
		Fonts: &evidence.FontEvidence{
			Faces: []evidence.FontFaceEvidence{
				{Family: "CustomFont", Status: "error"}, // Font load failure
			},
		},
	}

	req := critic.CritiqueRequest{
		RunID:   "defective-run",
		Level:   design.LevelPage,
		Profile: design.FindProfile("saas-modern"),
		Packet:  packet,
	}

	pass, err := c.Critique(context.Background(), req)
	if err != nil {
		t.Fatalf("Critique failed: %v", err)
	}

	if len(pass.Findings) < 3 {
		t.Fatalf("expected at least 3 findings (missing h1, overflow, missing name), got %d", len(pass.Findings))
	}
	if pass.HardViolations < 3 {
		t.Errorf("expected at least 3 hard violations, got %d", pass.HardViolations)
	}
	if len(pass.Hypotheses) < 2 {
		t.Fatalf("expected repair hypotheses generated, got %d", len(pass.Hypotheses))
	}
	if pass.GroundedScore >= 10.0 {
		t.Errorf("grounded score should be penalized, got %f", pass.GroundedScore)
	}
}
