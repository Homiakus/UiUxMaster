package controlplane

import (
	"context"
	"reflect"
	"testing"
)

func TestFileRunnerResumesAfterReopenAndLoadsWithoutReplay(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	starterExecutor := &fakeExecutor{
		plan:     EvidencePlan{Structural: true, Diagnostics: true, BrowserTruth: true},
		result:   ValidationResult{DiagnosticsComplete: true, Summary: "clean"},
		decision: DecisionPass,
	}
	starter, err := NewFile(starterExecutor, root)
	if err != nil {
		t.Fatal(err)
	}
	started, err := starter.Start(ctx, "durable-reopen", Change{Intent: "quick_structural"}, Budget{MaxBrowserFetches: 4})
	if err != nil {
		t.Fatal(err)
	}
	if started.Status == "completed" {
		t.Fatalf("Start unexpectedly executed activities: %#v", started)
	}
	if len(starterExecutor.calls) != 0 {
		t.Fatalf("Start executed activities: %v", starterExecutor.calls)
	}

	resumeExecutor := &fakeExecutor{
		plan:     EvidencePlan{Structural: true, Diagnostics: true, BrowserTruth: true},
		result:   ValidationResult{DiagnosticsComplete: true, Summary: "clean"},
		decision: DecisionPass,
	}
	resumed, err := NewFile(resumeExecutor, root)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := resumed.Run(ctx, "durable-reopen")
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "completed" || completed.Decision != DecisionPass {
		t.Fatalf("resumed run = %#v", completed)
	}
	if !reflect.DeepEqual(resumeExecutor.calls, []string{"plan", "collect", "decide"}) {
		t.Fatalf("resumed calls = %v", resumeExecutor.calls)
	}
	if completed.Usage.BrowserFetches != 1 {
		t.Fatalf("browser usage = %d, want 1", completed.Usage.BrowserFetches)
	}
	if len(completed.History) == 0 {
		t.Fatal("durable history is empty")
	}

	loadExecutor := &fakeExecutor{}
	reopened, err := NewFile(loadExecutor, root)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Load(ctx, "durable-reopen")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != completed.Status || loaded.Decision != completed.Decision || loaded.PlanDigest != completed.PlanDigest {
		t.Fatalf("reopened run diverged: %#v vs %#v", loaded, completed)
	}
	if len(loaded.History) != len(completed.History) {
		t.Fatalf("history length = %d, want %d", len(loaded.History), len(completed.History))
	}
	if len(loadExecutor.calls) != 0 {
		t.Fatalf("Load replayed activities: %v", loadExecutor.calls)
	}
}
