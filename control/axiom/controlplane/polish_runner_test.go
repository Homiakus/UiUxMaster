package controlplane

import (
	"context"
	"testing"
)

type mockPolishExecutor struct {
	baselineIter PolishIteration
	stepIters    []PolishIteration
	concludeRes  DesignPolishResult
}

func (m *mockPolishExecutor) InspectBaseline(ctx context.Context, req DesignPolishRequest) (PolishIteration, error) {
	return m.baselineIter, nil
}

func (m *mockPolishExecutor) StepPolish(ctx context.Context, req DesignPolishRequest, iter int) (PolishIteration, error) {
	if iter-1 < len(m.stepIters) {
		return m.stepIters[iter-1], nil
	}
	return PolishIteration{
		Iteration: iter, Score: 9.5, HardViolations: 0, Accepted: true, Rationale: "Fixed remaining defects",
	}, nil
}

func (m *mockPolishExecutor) ConcludePolish(ctx context.Context, req DesignPolishRequest, iters []PolishIteration) (DesignPolishResult, error) {
	return m.concludeRes, nil
}

func TestPolishRunner_MemoryWorkflow(t *testing.T) {
	exec := &mockPolishExecutor{
		baselineIter: PolishIteration{
			Iteration: 0, Score: 6.5, HardViolations: 2, FindingsCount: 3, Accepted: false, Rationale: "Initial baseline with defects",
		},
		stepIters: []PolishIteration{
			{Iteration: 1, Score: 8.0, HardViolations: 1, FindingsCount: 1, Accepted: true, Rationale: "Fixed typography"},
			{Iteration: 2, Score: 9.5, HardViolations: 0, FindingsCount: 0, Accepted: true, Rationale: "Fixed accessibility"},
		},
		concludeRes: DesignPolishResult{
			InitialScore:      6.5,
			FinalScore:        9.5,
			AcceptedCount:     2,
			TotalIterations:   2,
			Converged:         true,
			RemainingFindings: 0,
			Summary:           "Design polish converged successfully in 2 iterations.",
		},
	}

	runner, err := NewPolishMemory(exec)
	if err != nil {
		t.Fatalf("failed to create polish runner: %v", err)
	}

	run, err := runner.StartAndRun(context.Background(), "polish-test-1", DesignPolishRequest{
		Intent:        "Fix typography and a11y",
		MaxIterations: 3,
		ProtectedAxes: []string{"accessibility"},
	}, Budget{MaxBrowserFetches: 10})
	if err != nil {
		t.Fatalf("polish run failed: %v", err)
	}

	t.Logf("Polish run status=%q failure=%q history=%+v", run.Status, run.Failure, run.History)

	if run.Status != "completed" {
		t.Errorf("expected status 'completed', got %q (failure=%q)", run.Status, run.Failure)
	}
	if !run.Result.Converged {
		t.Errorf("expected Converged = true")
	}
	if run.Result.FinalScore != 9.5 {
		t.Errorf("expected FinalScore 9.5, got %v", run.Result.FinalScore)
	}
	if len(run.History) < 3 {
		t.Errorf("expected at least 3 history entries, got %d", len(run.History))
	}
}
