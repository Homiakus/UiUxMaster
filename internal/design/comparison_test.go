package design

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestRelativeComparator_CandidatePreferred(t *testing.T) {
	comp := NewComparator()

	// Baseline has accessibility missing name and font error
	baselinePacket := evidence.Packet{
		Viewport: evidence.Viewport{Width: 1280, Height: 800},
		Elements: []evidence.ElementRef{
			{ID: "btn-1", Tag: "button", Role: "button", Visible: true, Clickable: true, Name: ""},
			{ID: "h1", Tag: "h1", Visible: true},
		},
		Fonts: &evidence.FontEvidence{
			Faces: []evidence.FontFaceEvidence{
				{Family: "Inter", Status: "error"},
			},
		},
	}

	// Candidate fixed the accessibility name and font error
	candidatePacket := evidence.Packet{
		Viewport: evidence.Viewport{Width: 1280, Height: 800},
		Elements: []evidence.ElementRef{
			{ID: "btn-1", Tag: "button", Role: "button", Visible: true, Clickable: true, Name: "Submit Order"},
			{ID: "h1", Tag: "h1", Visible: true},
		},
		Fonts: &evidence.FontEvidence{
			Faces: []evidence.FontFaceEvidence{
				{Family: "Inter", Status: "loaded"},
			},
		},
	}

	res, err := comp.Compare(context.Background(), ComparisonRequest{
		RunID:           "test-run-1",
		BaselineID:      "v1",
		CandidateID:     "v2",
		BaselinePacket:  baselinePacket,
		CandidatePacket: candidatePacket,
		ProtectedAxes:   []string{"accessibility", "responsive"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.PassedConstraints {
		t.Errorf("expected PassedConstraints = true, got false with violations: %d (%s)", res.HardViolations, res.Rationale)
	}
	if res.PreferredCandidate != "v2" {
		t.Errorf("expected PreferredCandidate 'v2', got %q (%s)", res.PreferredCandidate, res.Rationale)
	}
	if len(res.RegressedAxes) != 0 {
		t.Errorf("expected 0 regressed axes, got %v", res.RegressedAxes)
	}
}

func TestRelativeComparator_CandidateRejectedOnHardConstraint(t *testing.T) {
	comp := NewComparator()

	// Baseline is clean
	baselinePacket := evidence.Packet{
		Viewport: evidence.Viewport{Width: 1280, Height: 800},
		Documents: []evidence.DocumentMetrics{
			{ContentWidth: 1200},
		},
		Elements: []evidence.ElementRef{
			{ID: "h1", Tag: "h1", Visible: true},
		},
	}

	// Candidate causes horizontal overflow
	candidatePacket := evidence.Packet{
		Viewport: evidence.Viewport{Width: 1280, Height: 800},
		Documents: []evidence.DocumentMetrics{
			{ContentWidth: 1400}, // overflow!
		},
		Elements: []evidence.ElementRef{
			{ID: "h1", Tag: "h1", Visible: true},
		},
	}

	res, err := comp.Compare(context.Background(), ComparisonRequest{
		BaselineID:      "v1",
		CandidateID:     "v2",
		BaselinePacket:  baselinePacket,
		CandidatePacket: candidatePacket,
		ProtectedAxes:   []string{"responsive"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.PassedConstraints {
		t.Errorf("expected PassedConstraints = false due to overflow, got true")
	}
	if res.PreferredCandidate != "v1" {
		t.Errorf("expected PreferredCandidate 'v1' (baseline preserved), got %q", res.PreferredCandidate)
	}
	if res.HardViolations == 0 {
		t.Errorf("expected hard violations > 0")
	}
}

func TestRelativeComparator_ProtectedAxisRegression(t *testing.T) {
	comp := NewComparator()

	// Baseline has high accessibility score
	baselinePacket := evidence.Packet{
		Elements: []evidence.ElementRef{
			{ID: "b1", Tag: "button", Role: "button", Visible: true, Clickable: true, Name: "Save"},
		},
	}

	// Candidate broke accessible name on button
	candidatePacket := evidence.Packet{
		Elements: []evidence.ElementRef{
			{ID: "b1", Tag: "button", Role: "button", Visible: true, Clickable: true, Name: ""},
		},
	}

	res, err := comp.Compare(context.Background(), ComparisonRequest{
		BaselineID:      "v1",
		CandidateID:     "v2",
		BaselinePacket:  baselinePacket,
		CandidatePacket: candidatePacket,
		ProtectedAxes:   []string{"accessibility"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.PassedConstraints {
		t.Errorf("expected PassedConstraints = false due to protected axis regression")
	}
	if len(res.RegressedAxes) == 0 {
		t.Errorf("expected accessibility in RegressedAxes, got %v", res.RegressedAxes)
	}
}
