package engine

import (
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestEvaluateRequestsCheapEvidenceBeforeVLM(t *testing.T) {
	report := Evaluate(evidence.Packet{RunID: "run-1"})

	if len(report.MissingEvidence) != 3 {
		t.Fatalf("expected 3 missing evidence classes, got %v", report.MissingEvidence)
	}
	if report.RecommendedNext != "collect missing deterministic evidence before escalating to a visual model" {
		t.Fatalf("unexpected next step: %q", report.RecommendedNext)
	}
}

func TestEvaluatePrioritizesBlockingDefects(t *testing.T) {
	report := Evaluate(evidence.Packet{
		RunID:          "run-2",
		AriaSnapshot:   "- button: Publish",
		ScreenshotPath: "shot.png",
		Elements: []evidence.ElementRef{{
			ID:      "publish",
			Role:    "button",
			Name:    "Publish",
			Visible: true,
		}},
		RuntimeIssues: []evidence.RuntimeIssue{{
			Code:     "page-error",
			Message:  "uncaught exception",
			Severity: evidence.SeverityCritical,
		}},
	})

	if report.BlockingFindings != 1 {
		t.Fatalf("expected one blocking finding, got %d", report.BlockingFindings)
	}
	if report.RecommendedNext != "repair blocking correctness/accessibility/runtime defects before aesthetic refinement" {
		t.Fatalf("unexpected next step: %q", report.RecommendedNext)
	}
}

func TestEvaluateEscalatesLocalizedUnexplainedRegions(t *testing.T) {
	report := Evaluate(evidence.Packet{
		RunID:          "run-3",
		AriaSnapshot:   "- heading: Example",
		ScreenshotPath: "shot.png",
		Elements: []evidence.ElementRef{{
			ID:      "hero",
			Role:    "heading",
			Name:    "Example",
			Visible: true,
		}},
		VisualRegions: []evidence.VisualRegion{{
			ID:        "region-1",
			DiffRatio: 0.12,
		}},
	})

	if report.RecommendedNext != "inspect suspicious regions with a local visual critic using cropped evidence" {
		t.Fatalf("unexpected next step: %q", report.RecommendedNext)
	}
}
