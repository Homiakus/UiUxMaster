package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/sideeffect"
)

func TestFMEA007CommitOnceReusesReceiptWithoutDuplicatingGraph(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	ns, err := NewProjectKnowledgeNamespace("project-fmea007")
	if err != nil { t.Fatal(err) }
	prov := ProvenanceRecord{RunID: "run-fmea007", EvidenceDigest: "sha256:evidence", Renderer: "playwright", Timestamp: time.Now()}
	bundle := AdmissionBundle{
		Atoms: []MemoryAtom{
			{ID: "pattern:a", Kind: NodeRepairPattern, Namespace: ns, Provenance: prov, Confidence: 1, Data: RepairPatternAtom{PatternID: "a"}},
			{ID: "finding:a", Kind: NodeDesignFinding, Namespace: ns, Provenance: prov, Confidence: 1, Data: DesignFindingAtom{FindingID: "a"}},
		},
		Edges: []MemoryEdge{{FromID: "pattern:a", ToID: "finding:a", Relation: RelRepairedBy, Weight: 1, Provenance: prov, CreatedAt: time.Now()}},
	}
	payload := sideeffect.DigestBytes([]byte("repair-pattern:a"))
	op := sideeffect.Operation{RunID: "run-fmea007", Activity: "memory_admission", Iteration: 1, Kind: "memory_admission", PayloadDigest: payload, RetryClass: sideeffect.RetryIdempotent}
	first, err := store.CommitOnce(ctx, op, bundle)
	if err != nil { t.Fatal(err) }
	second, err := store.CommitOnce(ctx, op, bundle)
	if err != nil { t.Fatal(err) }
	if !first.Applied || first.Reused || second.Applied || !second.Reused || first.OperationID != second.OperationID {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
	edges, err := store.GetEdges(ctx, "pattern:a")
	if err != nil { t.Fatal(err) }
	if len(edges) != 1 { t.Fatalf("edges=%d want 1", len(edges)) }
	q, err := store.Query(ctx, QueryRequest{Namespace: ns})
	if err != nil { t.Fatal(err) }
	if q.Total != 2 { t.Fatalf("atoms=%d want 2", q.Total) }
}

func TestFMEA007LegacyCommitStillDeduplicatesEdgesAndConflicts(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	ns, _ := NewProjectKnowledgeNamespace("project-fmea007")
	prov := ProvenanceRecord{RunID: "run-conflict", EvidenceDigest: "sha256:conflict", Renderer: "playwright", Timestamp: time.Now()}
	bundle := AdmissionBundle{
		Atoms: []MemoryAtom{
			{ID: "pattern:x", Kind: NodeRepairPattern, Namespace: ns, Provenance: prov, Confidence: 1},
			{ID: "ce:x", Kind: NodeCounterexample, Namespace: ns, Provenance: prov, Confidence: 1},
		},
		Edges: []MemoryEdge{{FromID: "ce:x", ToID: "pattern:x", Relation: RelRefutes, Weight: 1, Provenance: prov, CreatedAt: time.Now()}},
	}
	if err := store.Commit(ctx, bundle); err != nil { t.Fatal(err) }
	if err := store.Commit(ctx, bundle); err != nil { t.Fatal(err) }
	edges, _ := store.GetEdges(ctx, "ce:x")
	if len(edges) != 1 { t.Fatalf("edges=%d want 1", len(edges)) }
	if len(store.conflicts) != 1 { t.Fatalf("conflicts=%d want 1", len(store.conflicts)) }
	stored := store.atoms["pattern:x"]
	if stored == nil || len(stored.Conflicts) != 1 { t.Fatalf("atom conflicts=%#v", stored) }
}

func TestFMEA007CommitOnceRejectsOperationPayloadConflict(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	op := sideeffect.Operation{RunID: "run", Activity: "memory", Kind: "memory_admission", PayloadDigest: "sha256:one", RetryClass: sideeffect.RetryIdempotent}
	if _, err := store.CommitOnce(ctx, op, AdmissionBundle{}); err != nil { t.Fatal(err) }
	// Operation ID includes payload by design, so a semantically different payload
	// is a different operation. Explicit conflict protection is exercised by
	// corrupting/reusing the durable receipt slot below, matching recovery-defense behavior.
	id, _ := op.ID()
	store.mu.Lock()
	prior := store.committedOperations[id]
	prior.PayloadDigest = "sha256:other"
	store.committedOperations[id] = prior
	store.mu.Unlock()
	_, err := store.CommitOnce(ctx, op, AdmissionBundle{})
	if !errors.Is(err, sideeffect.ErrOperationConflict) { t.Fatalf("err=%v want operation conflict", err) }
}
