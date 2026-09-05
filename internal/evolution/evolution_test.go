package evolution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/memory"
)

func TestEvolutionManager_ExtractAndPromote(t *testing.T) {
	mgr := NewEvolutionManager()
	ctx := context.Background()
	baseVersion := SkillVersion{
		VersionID: "skill_responsive_v1.0.0",
		SkillID:   "skill_responsive",
		ActiveRules: []memory.DesignRuleAtom{{RuleID: "no_horizontal_overflow", Axis: "layout", Category: "overflow", Title: "No Viewport Overflow", HardConstraint: true, Weight: 1.0}},
		IsActive: true, CreatedAt: time.Now(),
	}
	prov := memory.ProvenanceRecord{RunID: "run_eval_1", EvidenceDigest: "sha256:evaldigest", Renderer: "fastcdp", Tier: fidelity.TierL2, Environment: "chromium", Timestamp: time.Now(), Outcome: "CONFIRMED"}
	findingAtoms := []memory.MemoryAtom{{ID: "f_atom_1", Kind: memory.NodeDesignFinding, Namespace: memory.NewGlobalDesignNamespace(), Provenance: prov, Confidence: 0.9, Data: memory.DesignFindingAtom{FindingID: "f_btn_pad", Category: "touch_target", Title: "Button padding minimum 12px"}}}
	heuristic, err := mgr.ExtractHeuristic("skill_responsive", findingAtoms)
	if err != nil { t.Fatalf("failed to extract heuristic: %v", err) }
	if heuristic.Category != "touch_target" { t.Fatalf("expected touch_target category, got %s", heuristic.Category) }

	candVersion := mgr.CreateCandidateVersion(baseVersion, *heuristic)
	if candVersion.IsActive || len(candVersion.ActiveRules) != 2 { t.Fatalf("unexpected candidate: %#v", candVersion) }
	corpus := []ReplayCase{{CaseID: "case_mobile_nav", Description: "Mobile navigation bar layout", InputCritique: design.CritiquePass{ID: "critique_1", Level: design.LevelPage, GroundedScore: 9.0, HardViolations: 0}, ExpectedHardViolations: 0, BaselineScore: 8.5}}

	better, reason := mgr.RunShadowEval(ctx, baseVersion, candVersion, corpus)
	if !better { t.Fatalf("expected candidate to pass shadow eval, failed: %s", reason) }
	if _, err := mgr.PromoteCandidate(ctx, candVersion, corpus); !errors.Is(err, ErrPromotionUnauthorized) { t.Fatalf("promotion without authorization err=%v", err) }
	auth, err := mgr.AuthorizePromotion(ctx, baseVersion, candVersion, corpus, "independent-evolution-verifier-v1")
	if err != nil { t.Fatalf("AuthorizePromotion: %v", err) }
	if !auth.ReplayPassed || !auth.ShadowPassed || auth.CorpusDigest == "" { t.Fatalf("incomplete authorization: %#v", auth) }
	promoted, err := mgr.PromoteCandidate(ctx, candVersion, corpus)
	if err != nil { t.Fatalf("failed to promote candidate: %v", err) }
	if !promoted.IsActive { t.Fatalf("expected promoted version to be active") }
	promotedReplay, err := mgr.PromoteCandidate(ctx, candVersion, corpus)
	if err != nil || promotedReplay.VersionID != promoted.VersionID { t.Fatalf("promotion replay=%#v err=%v", promotedReplay, err) }

	badCorpus := []ReplayCase{{CaseID: "case_broken_overflow", Description: "Layout with severe overflow", InputCritique: design.CritiquePass{ID: "critique_2", Level: design.LevelPage, GroundedScore: 4.0, HardViolations: 2}, ExpectedHardViolations: 0, BaselineScore: 8.0}}
	candBad := mgr.CreateCandidateVersion(promoted, *heuristic)
	if _, err := mgr.AuthorizePromotion(ctx, promoted, candBad, badCorpus, "independent-evolution-verifier-v1"); err == nil { t.Fatalf("expected authorization to fail due to non-regression gate") }
	if _, err = mgr.PromoteCandidate(ctx, candBad, badCorpus); err == nil { t.Fatalf("expected unauthorized bad candidate promotion to fail") }

	rolledBack, err := mgr.Rollback(ctx, baseVersion)
	if err != nil { t.Fatalf("failed rollback: %v", err) }
	if !rolledBack.IsActive || rolledBack.VersionID != baseVersion.VersionID { t.Fatalf("unexpected rollback state: %+v", rolledBack) }
}

func TestEvolutionPromotionRejectsEmptyCorpusAndCorpusDrift(t *testing.T) {
	mgr := NewEvolutionManager()
	ctx := context.Background()
	base := SkillVersion{VersionID: "s-v1", SkillID: "s", IsActive: true}
	candidate := SkillVersion{VersionID: "s-v2-cand", SkillID: "s"}
	if report := mgr.RunReplayEval(ctx, candidate, nil); report.PassedGate { t.Fatalf("empty corpus must not pass: %#v", report) }
	if _, err := mgr.AuthorizePromotion(ctx, base, candidate, nil, "verifier"); err == nil { t.Fatalf("empty corpus authorization unexpectedly succeeded") }

	corpus := []ReplayCase{{CaseID: "a", InputCritique: design.CritiquePass{GroundedScore: 9}, BaselineScore: 8.5}}
	if _, err := mgr.AuthorizePromotion(ctx, base, candidate, corpus, "verifier"); err != nil { t.Fatal(err) }
	drifted := append([]ReplayCase(nil), corpus...)
	drifted[0].BaselineScore = 9.8
	if _, err := mgr.PromoteCandidate(ctx, candidate, drifted); !errors.Is(err, ErrPromotionUnauthorized) { t.Fatalf("corpus drift err=%v want ErrPromotionUnauthorized", err) }
}
