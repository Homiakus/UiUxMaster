package skillstate

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/memory"
)

func TestSkillState_CASAndBudget(t *testing.T) {
	budget := BudgetState{MaxIterations: 5, RemainingIterations: 5, MaxVLMBudget: 3, RemainingVLMBudget: 3, MaxRepairAttempts: 2, RemainingRepairAttempts: 2}
	state := NewSkillState("run_abc", "skill_visual_polish", "Fix spacing and contrast", budget)
	if state.Revision != 1 { t.Fatalf("expected initial revision 1, got %d", state.Revision) }

	patch1 := StatePatch{ExpectedRevision: 1, ExpectedLastPatchDigest: "00000000", PhaseUpdate: "CRITIQUE", AddFindings: []string{"f_contrast_1", "f_spacing_1"}, AddActiveRegions: []string{"region_header"}, NewEvidenceDigest: "sha256:evidence1", DeductIterations: 1, DeductVLMBudget: 1}
	state2, err := ApplyPatch(state, patch1)
	if err != nil { t.Fatalf("unexpected error applying valid patch: %v", err) }
	if state2.Revision != 2 || state2.CurrentPhase != "CRITIQUE" || len(state2.ActiveFindingIDs) != 2 { t.Fatalf("unexpected state2: %#v", state2) }
	if state2.Budget.RemainingIterations != 4 || state2.Budget.RemainingVLMBudget != 2 { t.Fatalf("unexpected budget remaining: %+v", state2.Budget) }

	if _, err = ApplyPatch(state2, StatePatch{ExpectedRevision: 1, PhaseUpdate: "REPAIR"}); err == nil { t.Fatalf("expected error for stale revision") }
	if _, err = ApplyPatch(state2, StatePatch{ExpectedRevision: 2, ExpectedLastPatchDigest: "wrong_digest", PhaseUpdate: "REPAIR"}); err == nil { t.Fatalf("expected error for digest mismatch") }
	if _, err = ApplyPatch(state2, StatePatch{ExpectedRevision: state2.Revision, DeductIterations: 100}); err == nil { t.Fatalf("expected error for budget exhaustion") }

	state3, err := ApplyPatch(state2, StatePatch{ExpectedRevision: 2, PhaseUpdate: "FLIP_A"})
	if err != nil { t.Fatalf("failed patch A: %v", err) }
	state4, err := ApplyPatch(state3, StatePatch{ExpectedRevision: 3, PhaseUpdate: "FLIP_B"})
	if err != nil { t.Fatalf("failed patch B: %v", err) }
	state5, err := ApplyPatch(state4, StatePatch{ExpectedRevision: 4, PhaseUpdate: "FLIP_A"})
	if err != nil { t.Fatalf("failed re-applying patch A: %v", err) }
	if len(state5.OscillationFlags) == 0 { t.Fatalf("expected oscillation flag") }
}

func TestStoreMemoryPort_RetrieveAndAdmit(t *testing.T) {
	store := memory.NewEpMemoryStore()
	port := NewStoreMemoryPort(store)
	ctx := context.Background()
	state := NewSkillState("run_xyz", "skill_typography", "Check font settlement", BudgetState{MaxIterations: 5, RemainingIterations: 5})
	target, err := memory.NewProjectKnowledgeNamespace("run_xyz")
	if err != nil { t.Fatal(err) }

	prov := memory.ProvenanceRecord{RunID: "run_xyz", EvidenceDigest: "sha256:prov123", Renderer: "fastcdp", Tier: fidelity.TierL2, Environment: "chromium", Timestamp: time.Now(), Outcome: "CONFIRMED"}
	ruleAtom := memory.MemoryAtom{
		ID: "rule_font_settle", Kind: memory.NodeDesignRule, Namespace: target, Provenance: prov, Confidence: 1.0,
		Data: memory.DesignRuleAtom{RuleID: "font_settle", Axis: "typography", Category: "font_loading", Title: "Fonts Ready Gate", Description: "Fonts must be fully resolved before visual verification", HardConstraint: true, Weight: 1.0},
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := port.AdmitObservation(ctx, state, memory.AdmissionBundle{Atoms: []memory.MemoryAtom{ruleAtom}}); err != nil { t.Fatalf("failed to admit observation via memory port: %v", err) }

	pack, err := port.RetrieveContext(ctx, state, []string{"typography"}, 1000)
	if err != nil { t.Fatalf("failed to retrieve context via memory port: %v", err) }
	if len(pack.AdmittedRules) != 1 || pack.AdmittedRules[0].RuleID != "font_settle" { t.Fatalf("unexpected pack: %#v", pack) }
}

func TestStoreMemoryPortRejectsGlobalEscape(t *testing.T) {
	store := memory.NewEpMemoryStore()
	port := NewStoreMemoryPort(store)
	state := NewSkillState("run_private", "skill", "test", BudgetState{MaxIterations: 1, RemainingIterations: 1})
	atom := memory.MemoryAtom{ID: "leak", Kind: memory.NodeDesignRule, Namespace: memory.NewGlobalDesignNamespace(), Confidence: 1, Provenance: memory.ProvenanceRecord{RunID: "run_private", EvidenceDigest: "sha256:x", Renderer: "test", Timestamp: time.Now()}}
	if err := port.AdmitObservation(context.Background(), state, memory.AdmissionBundle{Atoms: []memory.MemoryAtom{atom}}); err == nil { t.Fatalf("expected project->global escape rejection") }
}
