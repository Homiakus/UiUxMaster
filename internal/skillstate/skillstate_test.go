package skillstate

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/memory"
)

func TestSkillState_CASAndBudget(t *testing.T) {
	budget := BudgetState{
		MaxIterations:          5,
		RemainingIterations:     5,
		MaxVLMBudget:           3,
		RemainingVLMBudget:      3,
		MaxRepairAttempts:      2,
		RemainingRepairAttempts: 2,
	}

	state := NewSkillState("run_abc", "skill_visual_polish", "Fix spacing and contrast", budget)
	if state.Revision != 1 {
		t.Fatalf("expected initial revision 1, got %d", state.Revision)
	}

	// 1. Valid Patch
	patch1 := StatePatch{
		ExpectedRevision:        1,
		ExpectedLastPatchDigest: "00000000",
		PhaseUpdate:             "CRITIQUE",
		AddFindings:             []string{"f_contrast_1", "f_spacing_1"},
		AddActiveRegions:        []string{"region_header"},
		NewEvidenceDigest:       "sha256:evidence1",
		DeductIterations:        1,
		DeductVLMBudget:         1,
	}

	state2, err := ApplyPatch(state, patch1)
	if err != nil {
		t.Fatalf("unexpected error applying valid patch: %v", err)
	}
	if state2.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", state2.Revision)
	}
	if state2.CurrentPhase != "CRITIQUE" {
		t.Fatalf("expected phase CRITIQUE, got %s", state2.CurrentPhase)
	}
	if len(state2.ActiveFindingIDs) != 2 {
		t.Fatalf("expected 2 active findings, got %d", len(state2.ActiveFindingIDs))
	}
	if state2.Budget.RemainingIterations != 4 || state2.Budget.RemainingVLMBudget != 2 {
		t.Fatalf("unexpected budget remaining: %+v", state2.Budget)
	}

	// 2. CAS Stale Patch (trying to apply patch expecting revision 1 on state2 which is at revision 2)
	stalePatch := StatePatch{
		ExpectedRevision: 1,
		PhaseUpdate:      "REPAIR",
	}
	_, err = ApplyPatch(state2, stalePatch)
	if err == nil {
		t.Fatalf("expected error for stale revision")
	}

	// 3. Digest Mismatch Check
	badDigestPatch := StatePatch{
		ExpectedRevision:        2,
		ExpectedLastPatchDigest: "wrong_digest",
		PhaseUpdate:             "REPAIR",
	}
	_, err = ApplyPatch(state2, badDigestPatch)
	if err == nil {
		t.Fatalf("expected error for digest mismatch")
	}

	// 4. Budget Exhaustion
	excessivePatch := StatePatch{
		ExpectedRevision: state2.Revision,
		DeductIterations: 100,
	}
	_, err = ApplyPatch(state2, excessivePatch)
	if err == nil {
		t.Fatalf("expected error for budget exhaustion")
	}

	// 5. Oscillation Detection
	// Apply patch A -> patch B -> patch A -> patch B
	patchA := StatePatch{
		ExpectedRevision: 2,
		PhaseUpdate:      "FLIP_A",
	}
	state3, err := ApplyPatch(state2, patchA)
	if err != nil {
		t.Fatalf("failed patch A: %v", err)
	}

	patchB := StatePatch{
		ExpectedRevision: 3,
		PhaseUpdate:      "FLIP_B",
	}
	state4, err := ApplyPatch(state3, patchB)
	if err != nil {
		t.Fatalf("failed patch B: %v", err)
	}

	// Re-apply identical patch A
	patchA2 := StatePatch{
		ExpectedRevision: 4,
		PhaseUpdate:      "FLIP_A",
	}
	state5, err := ApplyPatch(state4, patchA2)
	if err != nil {
		t.Fatalf("failed re-applying patch A: %v", err)
	}

	if len(state5.OscillationFlags) == 0 {
		t.Fatalf("expected oscillation flag to be raised on repeated alternating patch")
	}
}

func TestStoreMemoryPort_RetrieveAndAdmit(t *testing.T) {
	store := memory.NewEpMemoryStore()
	port := NewStoreMemoryPort(store)
	ctx := context.Background()

	budget := BudgetState{
		MaxIterations:       5,
		RemainingIterations: 5,
	}
	state := NewSkillState("run_xyz", "skill_typography", "Check font settlement", budget)

	prov := memory.ProvenanceRecord{
		RunID:          "run_xyz",
		EvidenceDigest: "sha256:prov123",
		Renderer:       "fastcdp",
		Tier:           fidelity.TierL2,
		Environment:    "chromium",
		Timestamp:      time.Now(),
		Outcome:        "CONFIRMED",
	}

	// 1. Admit Observation via MemoryPort
	ruleAtom := memory.MemoryAtom{
		ID:         "rule_font_settle",
		Kind:       memory.NodeDesignRule,
		Namespace:  memory.NewGlobalDesignNamespace(),
		Provenance: prov,
		Confidence: 1.0,
		Data: memory.DesignRuleAtom{
			RuleID:         "font_settle",
			Axis:           "typography",
			Category:       "font_loading",
			Title:          "Fonts Ready Gate",
			Description:    "Fonts must be fully resolved before visual verification",
			HardConstraint: true,
			Weight:         1.0,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	bundle := memory.AdmissionBundle{
		Atoms: []memory.MemoryAtom{ruleAtom},
	}

	if err := port.AdmitObservation(ctx, state, bundle); err != nil {
		t.Fatalf("failed to admit observation via memory port: %v", err)
	}

	// 2. Retrieve Bounded ContextPack via MemoryPort
	pack, err := port.RetrieveContext(ctx, state, []string{"typography"}, 1000)
	if err != nil {
		t.Fatalf("failed to retrieve context via memory port: %v", err)
	}

	if len(pack.AdmittedRules) != 1 {
		t.Fatalf("expected 1 admitted rule in retrieved pack, got %d", len(pack.AdmittedRules))
	}
	if pack.AdmittedRules[0].RuleID != "font_settle" {
		t.Fatalf("expected rule_font_settle, got %s", pack.AdmittedRules[0].RuleID)
	}
}
