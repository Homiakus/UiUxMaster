package controlplane

import (
	"context"
	"reflect"
	"testing"
)

type fakeExecutor struct {
	calls    []string
	plan     EvidencePlan
	result   ValidationResult
	decision Decision
}

func (f *fakeExecutor) PlanEvidence(_ context.Context, _ Change) (EvidencePlan, error) {
	f.calls = append(f.calls, "plan")
	return f.plan, nil
}

func (f *fakeExecutor) CollectVerify(_ context.Context, _ Change, _ EvidencePlan) (ValidationResult, error) {
	f.calls = append(f.calls, "collect")
	return f.result, nil
}

func (f *fakeExecutor) Decide(_ context.Context, _ Change, _ EvidencePlan, _ ValidationResult) (Decision, error) {
	f.calls = append(f.calls, "decide")
	return f.decision, nil
}

func TestStartAndRunCompletesAndProjectsHistory(t *testing.T) {
	executor := &fakeExecutor{
		plan: EvidencePlan{Structural: true, Diagnostics: true, BrowserTruth: true},
		result: ValidationResult{DiagnosticsComplete: true, Summary: "clean"},
		decision: DecisionPass,
	}
	runner, err := NewMemory(executor)
	if err != nil {
		t.Fatal(err)
	}
	if runner.PlanDigest() == "" {
		t.Fatal("empty plan digest")
	}

	run, err := runner.StartAndRun(context.Background(), "run-1", Change{Intent: "quick_structural"}, Budget{MaxBrowserFetches: 1})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "completed" {
		t.Fatalf("status = %q, want completed; failure=%q", run.Status, run.Failure)
	}
	if run.Decision != DecisionPass || !run.Evidence.Structural || !run.Validation.DiagnosticsComplete {
		t.Fatalf("unexpected projection: %#v", run)
	}
	if run.Usage.BrowserFetches != 1 {
		t.Fatalf("browser fetches = %d, want 1", run.Usage.BrowserFetches)
	}
	if !reflect.DeepEqual(executor.calls, []string{"plan", "collect", "decide"}) {
		t.Fatalf("calls = %v", executor.calls)
	}
	if len(run.History) < 4 {
		t.Fatalf("history too small: %#v", run.History)
	}
	loaded, err := runner.Load(context.Background(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.PlanDigest != run.PlanDigest || loaded.Decision != run.Decision {
		t.Fatalf("loaded run diverged: %#v vs %#v", loaded, run)
	}
}

func TestCancelBeforeExecutionPreventsActivities(t *testing.T) {
	executor := &fakeExecutor{
		plan: EvidencePlan{Structural: true, Diagnostics: true},
		decision: DecisionPass,
	}
	runner, err := NewMemory(executor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Start(context.Background(), "cancel-me", Change{Intent: "full_deterministic"}, Budget{}); err != nil {
		t.Fatal(err)
	}
	run, err := runner.Cancel(context.Background(), "cancel-me")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "canceled" {
		t.Fatalf("status = %q, want canceled; failure=%q", run.Status, run.Failure)
	}
	if len(executor.calls) != 0 {
		t.Fatalf("activities ran after pre-run cancel: %v", executor.calls)
	}
	foundCancel := false
	for _, entry := range run.History {
		if entry.Type == "event_ingested" || entry.Type == "execution_canceled" {
			foundCancel = true
		}
	}
	if !foundCancel {
		t.Fatalf("cancel not represented in history: %#v", run.History)
	}
}

func TestPixelEscalationIsAccountedAgainstBrowserBudget(t *testing.T) {
	executor := &fakeExecutor{
		plan: EvidencePlan{Structural: true, Diagnostics: true, Accessibility: true, Fonts: true, Pixels: true, BrowserTruth: true},
		result: ValidationResult{DiagnosticsComplete: true, VisualRegions: 1},
		decision: DecisionSemantic,
	}
	runner, err := NewMemory(executor)
	if err != nil {
		t.Fatal(err)
	}
	run, err := runner.StartAndRun(context.Background(), "budgeted", Change{Intent: "visual_region"}, Budget{MaxBrowserFetches: 1})
	if err != nil && run.Status == "" {
		t.Fatal(err)
	}
	if run.Status != "failed" {
		t.Fatalf("status = %q, want failed from browser budget; usage=%#v failure=%q err=%v", run.Status, run.Usage, run.Failure, err)
	}
	if run.Usage.BrowserFetches != 2 {
		t.Fatalf("browser fetches = %d, want 2", run.Usage.BrowserFetches)
	}
	if reflect.DeepEqual(executor.calls, []string{"plan", "collect", "decide"}) {
		t.Fatalf("decision must not run after budget is exceeded: %v", executor.calls)
	}
}
