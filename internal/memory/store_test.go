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

	prov := ProvenanceRecord{
		RunID:          "run_101",
		EvidenceDigest: "sha256:testdigest",
		Renderer:       "fastcdp",
		Tier:           fidelity.TierL2,
		Environment:    "chromium",
		Timestamp:      time.Now(),
		Outcome:        "CONFIRMED",
	}

	ns := NewGlobalDesignNamespace()

	ruleAtom := MemoryAtom{
		ID:         "rule_contrast_45",
		Kind:       NodeDesignRule,
		Namespace:  ns,
		Provenance: prov,
		Confidence: 1.0,
		Data: DesignRuleAtom{
			RuleID:         "contrast_45",
			Axis:           "accessibility",
			Category:       "contrast",
			Title:          "Minimum Contrast 4.5:1",
			Description:    "Text must maintain at least 4.5:1 contrast against background",
			HardConstraint: true,
			Weight:         1.0,
		},
		Tags:      []string{"accessibility", "hard_constraint"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	bundle := AdmissionBundle{
		Atoms: []MemoryAtom{ruleAtom},
	}

	if err := store.Commit(ctx, bundle); err != nil {
		t.Fatalf("failed to commit bundle: %v", err)
	}

	// 1. Get Atom
	retrieved, err := store.GetAtom(ctx, "rule_contrast_45")
	if err != nil {
		t.Fatalf("failed to get atom: %v", err)
	}
	if retrieved.ID != "rule_contrast_45" {
		t.Fatalf("expected ID rule_contrast_45, got %s", retrieved.ID)
	}

	// 2. Query
	qRes, err := store.Query(ctx, QueryRequest{
		Namespace: ns,
		Kind:      NodeDesignRule,
		Tags:      []string{"accessibility"},
		MinConf:   0.8,
	})
	if err != nil || qRes.Total != 1 {
		t.Fatalf("expected 1 query result, got %d, err: %v", qRes.Total, err)
	}

	// 3. Conflict Preservation
	ceAtom := MemoryAtom{
		ID:         "ce_contrast_exception",
		Kind:       NodeCounterexample,
		Namespace:  ns,
		Provenance: prov,
		Confidence: 0.9,
		Data: CounterexampleAtom{
			TargetEntityID: "rule_contrast_45",
			Reason:         "Disabled buttons may have lower contrast ratio",
			RefutingDigest: "sha256:ce_digest",
		},
		CreatedAt: time.Now(),
	}

	ceBundle := AdmissionBundle{
		Atoms: []MemoryAtom{ceAtom},
		Edges: []MemoryEdge{
			{
				FromID:     "ce_contrast_exception",
				ToID:       "rule_contrast_45",
				Relation:   RelCounterexampleTo,
				Weight:     0.9,
				Provenance: prov,
				CreatedAt:  time.Now(),
			},
		},
	}

	if err := store.Commit(ctx, ceBundle); err != nil {
		t.Fatalf("failed to commit counterexample: %v", err)
	}

	// Verify original rule is STILL active (truth preserved), and conflict is registered
	ruleStillActive, err := store.GetAtom(ctx, "rule_contrast_45")
	if err != nil {
		t.Fatalf("original rule should remain accessible: %v", err)
	}
	if ruleStillActive == nil {
		t.Fatalf("original rule is nil")
	}

	// 4. Bounded ContextPack Retrieval
	pack, err := store.RetrieveContextPack(ctx, ContextPackRequest{
		Scope:              ns,
		FocusAxes:          []string{"accessibility"},
		BudgetTokens:       1000,
		MaxSimilarCases:    5,
		MaxCounterexamples: 3,
	})
	if err != nil {
		t.Fatalf("failed to retrieve context pack: %v", err)
	}

	if len(pack.AdmittedRules) != 1 {
		t.Fatalf("expected 1 admitted rule in pack, got %d", len(pack.AdmittedRules))
	}
	if len(pack.Counterexamples) != 1 {
		t.Fatalf("expected 1 counterexample in pack, got %d", len(pack.Counterexamples))
	}
	if len(pack.ActiveConflicts) != 1 {
		t.Fatalf("expected 1 active conflict in pack, got %d", len(pack.ActiveConflicts))
	}
	if pack.EstimatedTokens <= 0 || pack.EstimatedTokens > 1000 {
		t.Fatalf("unexpected estimated tokens: %d", pack.EstimatedTokens)
	}

	// 5. Retract
	if err := store.Retract(ctx, "ce_contrast_exception", "Obsolete finding", prov); err != nil {
		t.Fatalf("failed to retract: %v", err)
	}

	// Query should no longer return retracted atom
	_, err = store.GetAtom(ctx, "ce_contrast_exception")
	if err == nil {
		t.Fatalf("expected error for retracted atom")
	}

	// 6. Supersede
	newerRule := MemoryAtom{
		ID:         "rule_contrast_45_v2",
		Kind:       NodeDesignRule,
		Namespace:  ns,
		Provenance: prov,
		Confidence: 1.0,
		Data: DesignRuleAtom{
			RuleID:         "contrast_45",
			Axis:           "accessibility",
			Category:       "contrast",
			Title:          "Minimum Contrast 4.5:1 (v2)",
			Description:    "Text must maintain at least 4.5:1 contrast against background; disabled inputs exempt",
			HardConstraint: true,
			Weight:         1.0,
			Version:        "v2.0",
		},
		CreatedAt: time.Now(),
	}

	if err := store.Supersede(ctx, "rule_contrast_45", newerRule, prov); err != nil {
		t.Fatalf("failed to supersede: %v", err)
	}

	// Old rule should now be superseded
	_, err = store.GetAtom(ctx, "rule_contrast_45")
	if err == nil {
		t.Fatalf("expected superseded atom to not be active")
	}

	// Newer rule should be active
	newActive, err := store.GetAtom(ctx, "rule_contrast_45_v2")
	if err != nil || newActive.ID != "rule_contrast_45_v2" {
		t.Fatalf("failed to get superseded active rule")
	}
}
