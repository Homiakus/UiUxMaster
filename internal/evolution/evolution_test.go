package evolution

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/memory"
)

func TestEvolutionManager_ExtractAndPromote(t *testing.T) {
	mgr := NewEvolutionManager()
	ctx := context.Background()

	// 1. Base Active Version
	baseVersion := SkillVersion{
		VersionID: "skill_responsive_v1.0.0",
		SkillID:   "skill_responsive",
		ActiveRules: []memory.DesignRuleAtom{
			{
				RuleID:         "no_horizontal_overflow",
				Axis:           "layout",
				Category:       "overflow",
				Title:          "No Viewport Overflow",
				HardConstraint: true,
				Weight:         1.0,
			},
		},
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	// 2. Extract Candidate Heuristic from Admitted Findings
	prov := memory.ProvenanceRecord{
		RunID:          "run_eval_1",
		EvidenceDigest: "sha256:evaldigest",
		Renderer:       "fastcdp",
		Tier:           fidelity.TierL2,
		Environment:    "chromium",
		Timestamp:      time.Now(),
		Outcome:        "CONFIRMED",
	}

	findingAtoms := []memory.MemoryAtom{
		{
			ID:         "f_atom_1",
			Kind:       memory.NodeDesignFinding,
			Namespace:  memory.NewGlobalDesignNamespace(),
			Provenance: prov,
			Confidence: 0.9,
			Data: memory.DesignFindingAtom{
				FindingID: "f_btn_pad",
				Category:  "touch_target",
				Title:     "Button padding minimum 12px",
			},
		},
	}

	heuristic, err := mgr.ExtractHeuristic("skill_responsive", findingAtoms)
	if err != nil {
		t.Fatalf("failed to extract heuristic: %v", err)
	}
	if heuristic.Category != "touch_target" {
		t.Fatalf("expected touch_target category, got %s", heuristic.Category)
	}

	// 3. Create Candidate Version
	candVersion := mgr.CreateCandidateVersion(baseVersion, *heuristic)
	if candVersion.IsActive {
		t.Fatalf("candidate version must not be active immediately")
	}
	if len(candVersion.ActiveRules) != 2 {
		t.Fatalf("expected 2 active rules in candidate, got %d", len(candVersion.ActiveRules))
	}

	// 4. Replay Corpus
	corpus := []ReplayCase{
		{
			CaseID:      "case_mobile_nav",
			Description: "Mobile navigation bar layout",
			InputCritique: design.CritiquePass{
				ID:             "critique_1",
				Level:          design.LevelPage,
				GroundedScore:  9.0,
				HardViolations: 0,
			},
			ExpectedHardViolations: 0,
			BaselineScore:          8.5,
		},
	}

	// 5. Shadow Evaluation
	better, reason := mgr.RunShadowEval(ctx, baseVersion, candVersion, corpus)
	if !better {
		t.Fatalf("expected candidate to pass shadow eval, failed: %s", reason)
	}

	// 6. Promotion
	promoted, err := mgr.PromoteCandidate(ctx, candVersion, corpus)
	if err != nil {
		t.Fatalf("failed to promote candidate: %v", err)
	}
	if !promoted.IsActive {
		t.Fatalf("expected promoted version to be active")
	}

	// 7. Test Non-Regression Gate Rejection
	badCorpus := []ReplayCase{
		{
			CaseID:      "case_broken_overflow",
			Description: "Layout with severe overflow",
			InputCritique: design.CritiquePass{
				ID:             "critique_2",
				Level:          design.LevelPage,
				GroundedScore:  4.0,
				HardViolations: 2, // Unexpected hard violations
			},
			ExpectedHardViolations: 0,
			BaselineScore:          8.0,
		},
	}

	candBad := mgr.CreateCandidateVersion(promoted, *heuristic)
	_, err = mgr.PromoteCandidate(ctx, candBad, badCorpus)
	if err == nil {
		t.Fatalf("expected promotion to fail due to non-regression gate")
	}

	// 8. Rollback
	rolledBack, err := mgr.Rollback(ctx, baseVersion)
	if err != nil {
		t.Fatalf("failed rollback: %v", err)
	}
	if !rolledBack.IsActive || rolledBack.VersionID != baseVersion.VersionID {
		t.Fatalf("unexpected rollback state: %+v", rolledBack)
	}
}
