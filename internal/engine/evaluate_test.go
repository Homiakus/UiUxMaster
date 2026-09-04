package engine

import (
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestEvaluateRequestsCheapEvidenceBeforePixels(t *testing.T) {
	report := Evaluate(evidence.Packet{RunID: "run-1"})
	if len(report.MissingEvidence) != 2 {
		t.Fatalf("expected accessibility + structural evidence only, got %v", report.MissingEvidence)
	}
	for _, missing := range report.MissingEvidence {
		if missing == "rendered screenshot" || missing == "rendered region pixels" {
			t.Fatalf("clean structural path must not demand pixels: %v", report.MissingEvidence)
		}
	}
	if report.RecommendedNext != "collect the cheapest missing deterministic evidence before escalating fidelity" {
		t.Fatalf("unexpected next step: %q", report.RecommendedNext)
	}
}

func TestEvaluateCleanDeterministicPacketDoesNotRequireScreenshot(t *testing.T) {
	report := Evaluate(evidence.Packet{
		RunID: "clean",
		AriaSnapshot: "- button: Publish",
		Elements: []evidence.ElementRef{{ID: "publish", Role: "button", Name: "Publish", Visible: true}},
	})
	if len(report.MissingEvidence) != 0 {
		t.Fatalf("unexpected missing evidence: %v", report.MissingEvidence)
	}
	if report.RecommendedNext != "deterministic evidence is clean; escalate to pixel or semantic comparison only when change risk or policy requires it" {
		t.Fatalf("unexpected next step: %q", report.RecommendedNext)
	}
}

func TestEvaluatePrioritizesBlockingDefects(t *testing.T) {
	report := Evaluate(evidence.Packet{
		RunID: "run-2", AriaSnapshot: "- button: Publish",
		Elements: []evidence.ElementRef{{ID: "publish", Role: "button", Name: "Publish", Visible: true}},
		RuntimeIssues: []evidence.RuntimeIssue{{Code: "page-error", Message: "uncaught exception", Severity: evidence.SeverityCritical}},
	})
	if report.BlockingFindings != 1 {
		t.Fatalf("expected one blocking finding, got %d", report.BlockingFindings)
	}
	if report.RecommendedNext != "repair blocking correctness/accessibility/runtime defects before aesthetic refinement" {
		t.Fatalf("unexpected next step: %q", report.RecommendedNext)
	}
}

func TestEvaluateRequestsPixelsOnlyForLocalizedVisualRegion(t *testing.T) {
	report := Evaluate(evidence.Packet{
		RunID: "run-3", AriaSnapshot: "- heading: Example",
		Elements: []evidence.ElementRef{{ID: "hero", Role: "heading", Name: "Example", Visible: true}},
		VisualRegions: []evidence.VisualRegion{{ID: "region-1", DiffRatio: 0.12}},
	})
	if len(report.MissingEvidence) != 1 || report.MissingEvidence[0] != "rendered region pixels" {
		t.Fatalf("missing evidence = %v", report.MissingEvidence)
	}
}

func TestEvaluateEscalatesLocalizedUnexplainedRegionsWithPixels(t *testing.T) {
	report := Evaluate(evidence.Packet{
		RunID: "run-4", AriaSnapshot: "- heading: Example",
		Elements: []evidence.ElementRef{{ID: "hero", Role: "heading", Name: "Example", Visible: true}},
		VisualRegions: []evidence.VisualRegion{{ID: "region-1", DiffRatio: 0.12}},
		Pixels: &evidence.PixelEvidence{Width: 320, Height: 180, DigestSHA256: "abc"},
	})
	if report.RecommendedNext != "inspect suspicious regions with a local visual critic using cropped pixel evidence" {
		t.Fatalf("unexpected next step: %q", report.RecommendedNext)
	}
}
