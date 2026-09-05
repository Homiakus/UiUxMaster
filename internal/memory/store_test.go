package memory

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/fidelity"
)

func TestEpMemoryStore_CommitAndRetrieve(t *testing.T) {
	store := NewEpMemoryStore()
	ctx := context.Background()
	ns := NewGlobalDesignNamespace()
	prov := ProvenanceRecord{
		RunID:           "run_101",
		EvidenceDigest:  "sha256:testdigest",
		Renderer:        "fastcdp",
		Tier:            fidelity.TierL2,
		Environment:     "chromium",
		ProjectScope:    "global",
		SourceNamespace: ns.String(),
		Timestamp:       time.Now(),
		Outcome:         "CONFIRMED",
	}

	ruleAtom := MemoryAtom{
		ID: "rule_contrast_45", Kind: NodeDesignRule, Namespace: ns, Provenance: prov, Confidence: 1.0,
		Data: DesignRuleAtom{RuleID: "contrast_45", Axis: "accessibility", Category: "contrast", Title: "Minimum Contrast 4.5:1", Description: "Text must maintain at least 4.5:1 contrast against background", HardConstraint: true, Weight: 1.0},
		Tags: []string{"accessibility", "hard_constraint"}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: ns, Atoms: []MemoryAtom{ruleAtom}}); err != nil {
		t.Fatalf("failed to commit bundle: %v", err)
	}

	retrieved, err := store.GetAtom(ctx, ns, "rule_contrast_45")
	if err != nil || retrieved.ID != "rule_contrast_45" {
		t.Fatalf("failed scoped get: atom=%#v err=%v", retrieved, err)
	}
	qRes, err := store.Query(ctx, QueryRequest{Namespace: ns, Kind: NodeDesignRule, Tags: []string{"accessibility"}, MinConf: 0.8})
	if err != nil || qRes.Total != 1 {
		t.Fatalf("expected 1 query result, got %#v err=%v", qRes, err)
	}

	ceAtom := MemoryAtom{
		ID: "ce_contrast_exception", Kind: NodeCounterexample, Namespace: ns, Provenance: prov, Confidence: 0.9,
		Data: CounterexampleAtom{TargetEntityID: "rule_contrast_45", Reason: "Disabled buttons may have lower contrast ratio", RefutingDigest: "sha256:ce_digest"},
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	ceBundle := AdmissionBundle{
		SourceNamespace: ns,
		Atoms: []MemoryAtom{ceAtom},
		Edges: []MemoryEdge{{FromID: "ce_contrast_exception", ToID: "rule_contrast_45", Relation: RelCounterexampleTo, Weight: 0.9, Provenance: prov, CreatedAt: time.Now()}},
	}
	if err := store.Commit(ctx, ceBundle); err != nil {
		t.Fatalf("failed to commit counterexample: %v", err)
	}
	if _, err := store.GetAtom(ctx, ns, "rule_contrast_45"); err != nil {
		t.Fatalf("original rule should remain active: %v", err)
	}

	pack, err := store.RetrieveContextPack(ctx, ContextPackRequest{Scope: ns, FocusAxes: []string{"accessibility"}, BudgetTokens: 1000, MaxSimilarCases: 5, MaxCounterexamples: 3})
	if err != nil {
		t.Fatalf("failed to retrieve context pack: %v", err)
	}
	if len(pack.AdmittedRules) != 1 || len(pack.Counterexamples) != 1 || len(pack.ActiveConflicts) != 1 {
		t.Fatalf("unexpected context pack: %#v", pack)
	}

	if err := store.Retract(ctx, ns, "ce_contrast_exception", "Obsolete finding", prov); err != nil {
		t.Fatalf("failed to retract: %v", err)
	}
	if _, err := store.GetAtom(ctx, ns, "ce_contrast_exception"); err == nil {
		t.Fatalf("expected error for retracted atom")
	}

	newerRule := MemoryAtom{
		ID: "rule_contrast_45_v2", Kind: NodeDesignRule, Namespace: ns, Provenance: prov, Confidence: 1.0,
		Data: DesignRuleAtom{RuleID: "contrast_45", Axis: "accessibility", Category: "contrast", Title: "Minimum Contrast 4.5:1 (v2)", Description: "Text must maintain at least 4.5:1 contrast against background; disabled inputs exempt", HardConstraint: true, Weight: 1.0, Version: "v2.0"},
		CreatedAt: time.Now(),
	}
	if err := store.Supersede(ctx, ns, "rule_contrast_45", newerRule, prov); err != nil {
		t.Fatalf("failed to supersede: %v", err)
	}
	if _, err := store.GetAtom(ctx, ns, "rule_contrast_45"); err == nil {
		t.Fatalf("expected superseded atom to be inactive")
	}
	newActive, err := store.GetAtom(ctx, ns, "rule_contrast_45_v2")
	if err != nil || newActive.ID != "rule_contrast_45_v2" {
		t.Fatalf("failed to get superseded active rule: %#v %v", newActive, err)
	}
}
