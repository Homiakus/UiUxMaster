package uiuxadapter

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

func TestPolishAdapter_EndToEndWorkflow(t *testing.T) {
	pipeline := &engine.Pipeline{
		VerPolicy: verifier.DefaultPolicy(),
	}
	adapter := NewPolishAdapter(pipeline)

	runner, err := controlplane.NewPolishMemory(adapter)
	if err != nil {
		t.Fatalf("failed to create polish runner: %v", err)
	}

	run, err := runner.StartAndRun(context.Background(), "polish-run-e2e", controlplane.DesignPolishRequest{
		Intent:        "improve typography and contrast",
		Profile:       "saas-modern",
		MaxIterations: 2,
	}, controlplane.Budget{MaxDuration: 10 * time.Second, MaxBrowserFetches: 10})
	if err != nil {
		t.Fatalf("polish run failed: %v", err)
	}

	if run.Status != "completed" {
		t.Errorf("expected status 'completed', got %q (%s)", run.Status, run.Failure)
	}
	if run.Result.TotalIterations < 1 {
		t.Errorf("expected at least 1 iteration, got %d", run.Result.TotalIterations)
	}
}

func TestComparisonAdapter_EndToEndWorkflow(t *testing.T) {
	adapter := NewComparisonAdapter()

	runner, err := controlplane.NewComparisonMemory(adapter)
	if err != nil {
		t.Fatalf("failed to create comparison runner: %v", err)
	}

	run, err := runner.StartAndRun(context.Background(), "cmp-run-e2e", controlplane.CandidateComparisonRequest{
		BaselineID:    "base",
		CandidateIDs:  []string{"var_1", "var_2"},
		ProtectedAxes: []string{"accessibility", "responsive"},
	}, controlplane.Budget{MaxDuration: 10 * time.Second, MaxBrowserFetches: 10})
	if err != nil {
		t.Fatalf("comparison run failed: %v", err)
	}

	if run.Status != "completed" {
		t.Errorf("expected status 'completed', got %q (%s)", run.Status, run.Failure)
	}
	if run.Result.WinnerID == "" {
		t.Errorf("expected a winner to be selected")
	}
	if len(run.Result.Rankings) != 2 {
		t.Errorf("expected 2 rankings, got %d", len(run.Result.Rankings))
	}
}
