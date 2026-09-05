package controlplane

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/memory"
	"github.com/Homiakus/UiUxMaster/internal/sideeffect"
	"github.com/Homiakus/axiom/adgo"
)

type fmea007EffectExecutor struct {
	source          *sideeffect.SourceStore
	memory          *memory.EpMemoryStore
	failAfterEffect bool
	calls           int
	receipts        []sideeffect.Receipt
	activityKeys    []string
}

func (e *fmea007EffectExecutor) InspectBaseline(context.Context, DesignPolishRequest) (PolishIteration, error) {
	return PolishIteration{Iteration: 0, FindingsCount: 1, HardViolations: 1, Score: 1}, nil
}

func (e *fmea007EffectExecutor) StepPolish(ctx context.Context, _ DesignPolishRequest, iter int) (PolishIteration, error) {
	identity, ok := ActivityIdentityFromContext(ctx)
	if !ok { return PolishIteration{}, errors.New("missing Axiom activity identity") }
	e.calls++
	e.activityKeys = append(e.activityKeys, identity.IdempotencyKey)

	initial := sideeffect.SourceState{HTML: "old"}
	desired := sideeffect.SourceState{HTML: "repaired"}
	sourceOp := sideeffect.Operation{
		RunID: identity.ExecutionID, Activity: identity.NodeID, Iteration: iter,
		Kind: "source_repair", PayloadDigest: desired.Digest(), RetryClass: sideeffect.RetryIdempotent,
	}
	sourceReceipt, err := e.source.CompareAndSwap(ctx, sourceOp, initial.Digest(), desired)
	if err != nil { return PolishIteration{}, err }
	e.receipts = append(e.receipts, sourceReceipt)

	ns, _ := memory.NewProjectKnowledgeNamespace("fmea007")
	prov := memory.ProvenanceRecord{RunID: identity.ExecutionID, EvidenceDigest: "sha256:fmea007", Renderer: "playwright", Timestamp: time.Now()}
	bundle := memory.AdmissionBundle{
		Atoms: []memory.MemoryAtom{
			{ID: "pattern:fmea007", Kind: memory.NodeRepairPattern, Namespace: ns, Provenance: prov, Confidence: 1},
			{ID: "finding:fmea007", Kind: memory.NodeDesignFinding, Namespace: ns, Provenance: prov, Confidence: 1},
		},
		Edges: []memory.MemoryEdge{{FromID: "pattern:fmea007", ToID: "finding:fmea007", Relation: memory.RelRepairedBy, Weight: 1, Provenance: prov, CreatedAt: time.Now()}},
	}
	memoryPayload := sideeffect.DigestBytes([]byte("pattern:fmea007"))
	memoryOp := sideeffect.Operation{
		RunID: identity.ExecutionID, Activity: identity.NodeID, Iteration: iter,
		Kind: "memory_admission", PayloadDigest: memoryPayload, RetryClass: sideeffect.RetryIdempotent,
	}
	memoryReceipt, err := e.memory.CommitOnce(ctx, memoryOp, bundle)
	if err != nil { return PolishIteration{}, err }
	e.receipts = append(e.receipts, memoryReceipt)

	if e.failAfterEffect {
		e.failAfterEffect = false
		return PolishIteration{}, adgo.Fail(adgo.FailureTransient, errors.New("injected crash after side effects before activity completion"))
	}
	return PolishIteration{Iteration: iter, FindingsCount: 0, HardViolations: 0, Score: 10, Accepted: true, Rationale: "replayed effects reused"}, nil
}

func (e *fmea007EffectExecutor) ConcludePolish(_ context.Context, _ DesignPolishRequest, iters []PolishIteration) (DesignPolishResult, error) {
	return DesignPolishResult{Converged: true, TotalIterations: len(iters) - 1, Iterations: iters, FinalScore: 10}, nil
}

func TestFMEA007AxiomRetryReusesTargetEffectsAndStableActivityKey(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source, err := sideeffect.NewFileSourceStore(filepath.Join(root, "external-source.json"), sideeffect.SourceState{HTML: "old"})
	if err != nil { t.Fatal(err) }
	executor := &fmea007EffectExecutor{source: source, memory: memory.NewEpMemoryStore(), failAfterEffect: true}
	runner, err := NewPolishFile(executor, filepath.Join(root, "axiom"))
	if err != nil { t.Fatal(err) }

	run, err := runner.StartAndRun(ctx, "fmea007-replay", DesignPolishRequest{MaxIterations: 1}, Budget{})
	if err != nil { t.Fatal(err) }
	if run.Status != "completed" || !run.Result.Converged { t.Fatalf("run=%#v", run) }
	if executor.calls != 2 { t.Fatalf("StepPolish calls=%d want 2", executor.calls) }
	if len(executor.receipts) != 4 { t.Fatalf("receipts=%d want 4", len(executor.receipts)) }
	if !executor.receipts[0].Applied || !executor.receipts[1].Applied { t.Fatalf("first attempt receipts=%#v", executor.receipts[:2]) }
	if !executor.receipts[2].Reused || executor.receipts[2].Applied || !executor.receipts[3].Reused || executor.receipts[3].Applied {
		t.Fatalf("retry receipts=%#v", executor.receipts[2:])
	}
	if len(executor.activityKeys) != 2 || executor.activityKeys[0] == "" || executor.activityKeys[0] != executor.activityKeys[1] {
		t.Fatalf("activity keys=%v", executor.activityKeys)
	}

	edges, err := executor.memory.GetEdges(ctx, "pattern:fmea007")
	if err != nil { t.Fatal(err) }
	if len(edges) != 1 { t.Fatalf("memory edges=%d want exactly 1", len(edges)) }
	if got := source.Current(ctx); got.HTML != "repaired" { t.Fatalf("source=%#v", got) }

	var historyKeys []string
	for _, entry := range run.History {
		if entry.Type != "activity_started" || entry.NodeID != "iterate_repair" { continue }
		if key, ok := entry.Data["idempotencyKey"].(string); ok { historyKeys = append(historyKeys, key) }
	}
	if len(historyKeys) < 2 { t.Fatalf("durable history did not expose retry idempotency keys: %#v", run.History) }
	for _, key := range historyKeys { if key != executor.activityKeys[0] { t.Fatalf("history key=%q executor key=%q", key, executor.activityKeys[0]) } }

	// Reopen the durable workflow store: completed state/history must load without
	// invoking external effects again.
	reopenedExecutor := &fmea007EffectExecutor{source: source, memory: executor.memory}
	reopened, err := NewPolishFile(reopenedExecutor, filepath.Join(root, "axiom"))
	if err != nil { t.Fatal(err) }
	loaded, err := reopened.Load(ctx, "fmea007-replay")
	if err != nil { t.Fatal(err) }
	if loaded.Status != "completed" || len(loaded.History) != len(run.History) { t.Fatalf("loaded=%#v", loaded) }
	if reopenedExecutor.calls != 0 { t.Fatalf("Load replayed effects %d times", reopenedExecutor.calls) }
}
