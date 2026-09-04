package controlplane

import (
	"context"
	"testing"
	"time"
)

type mockComparisonExecutor struct {
	ranks []CandidateRank
	res   ComparisonRunResult
}

func (m *mockComparisonExecutor) EvaluateCandidates(ctx context.Context, req CandidateComparisonRequest) ([]CandidateRank, error) {
	return m.ranks, nil
}

func (m *mockComparisonExecutor) ConcludeComparison(ctx context.Context, req CandidateComparisonRequest, ranks []CandidateRank) (ComparisonRunResult, error) {
	return m.res, nil
}

func TestComparisonRunner_MemoryWorkflow(t *testing.T) {
	exec := &mockComparisonExecutor{
		ranks: []CandidateRank{
			{CandidateID: "candidate_b", Rank: 1, Score: 9.2, PassedConstraints: true, Rationale: "Optimal aesthetics and zero defects"},
			{CandidateID: "candidate_a", Rank: 2, Score: 8.5, PassedConstraints: true, Rationale: "Good baseline"},
			{CandidateID: "candidate_c", Rank: 3, Score: 7.0, PassedConstraints: false, RegressedAxes: []string{"accessibility"}, Rationale: "Regressed a11y"},
		},
		res: ComparisonRunResult{
			WinnerID: "candidate_b",
			Rankings: []CandidateRank{
				{CandidateID: "candidate_b", Rank: 1, Score: 9.2, PassedConstraints: true},
				{CandidateID: "candidate_a", Rank: 2, Score: 8.5, PassedConstraints: true},
				{CandidateID: "candidate_c", Rank: 3, Score: 7.0, PassedConstraints: false},
			},
			Summary: "candidate_b selected as winner with highest quality score and full constraints pass.",
		},
	}

	runner, err := NewComparisonMemory(exec)
	if err != nil {
		t.Fatalf("failed to create comparison runner: %v", err)
	}

	run, err := runner.StartAndRun(context.Background(), "cmp-test-1", CandidateComparisonRequest{
		BaselineID:    "candidate_a",
		CandidateIDs:  []string{"candidate_b", "candidate_c"},
		ProtectedAxes: []string{"accessibility"},
	}, Budget{MaxDuration: 10 * time.Second, MaxBrowserFetches: 10})
	if err != nil {
		t.Fatalf("comparison run failed: %v", err)
	}

	if run.Status != "completed" {
		t.Errorf("expected status 'completed', got %q (failure=%q)", run.Status, run.Failure)
	}
	if run.Result.WinnerID != "candidate_b" {
		t.Errorf("expected WinnerID 'candidate_b', got %q", run.Result.WinnerID)
	}
	if len(run.Result.Rankings) != 3 {
		t.Errorf("expected 3 ranked candidates, got %d", len(run.Result.Rankings))
	}
}
