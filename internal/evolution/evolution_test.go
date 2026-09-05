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

func evolutionGlobalFinding(id, findingID, runID, digest string, confidence float64) memory.MemoryAtom {
	global := memory.NewGlobalDesignNamespace()
	prov := memory.ProvenanceRecord{
		RunID: runID, EvidenceDigest: digest, Renderer: "fastcdp", Tier: fidelity.TierL2,
		Environment: "chromium", ProjectScope: "global", SourceNamespace: global.String(), Timestamp: time.Now(), Outcome: "CONFIRMED",
	}
	return memory.MemoryAtom{
		ID: id, Kind: memory.NodeDesignFinding, Namespace: global, Provenance: prov, Confidence: confidence,
		Data: memory.DesignFindingAtom{FindingID: findingID, Category: "touch_target", Title: "Repeated touch target finding"},
	}
}

func TestEvolutionManager_ExtractAndPromote(t *testing.T) {
	mgr := NewEvolutionManager()
	ctx := context.Background()
	baseVersion := SkillVersion{
		VersionID: "skill_responsive_v1.0.0", SkillID: "skill_responsive",
		ActiveRules: []memory.DesignRuleAtom{{RuleID: "no_horizontal_overflow", Axis: "layout", Category: "overflow", Title: "No Viewport Overflow", HardConstraint: true, Weight: 1.0}},
		IsActive: true, CreatedAt: time.Now(),
	}
	findingAtoms := []memory.MemoryAtom{
		evolutionGlobalFinding("f_atom_1", "f_btn_pad_1", "run_eval_1", "sha256:evaldigest1", 0.9),
		evolutionGlobalFinding("f_atom_2", "f_btn_pad_2", "run_eval_2", "sha256:evaldigest2", 0.92),
	}
	heuristic, err := mgr.ExtractHeuristic("skill_responsive", findingAtoms)
	if err != nil { t.Fatalf("failed to extract heuristic: %v", err) }
	if heuristic.Category != "touch_target" || len(heuristic.SourceEvidenceDigests) != 2 || len(heuristic.SourceRunIDs) != 2 || len(heuristic.SourceFindingIDs) != 2 {
		t.Fatalf("incomplete repeated heuristic lineage: %#v", heuristic)
	}

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

func TestEvolutionHeuristicRejectsSinglePrivateAndNonIndependentEvidence(t *testing.T) {
	mgr := NewEvolutionManager()
	one := evolutionGlobalFinding("one", "f1", "run1", "sha256:d1", 0.9)
	if _, err := mgr.ExtractHeuristic("skill", []memory.MemoryAtom{one}); err == nil { t.Fatalf("single finding unexpectedly produced heuristic") }

	privateNS, _ := memory.NewProjectKnowledgeNamespace("private-project")
	privateSource, _ := memory.NewProjectEvidenceNamespace("private-project")
	private := one
	private.ID = "private"
	private.Namespace = privateNS
	private.Provenance.ProjectScope = "private-project"
	private.Provenance.SourceNamespace = privateSource.String()
	second := evolutionGlobalFinding("two", "f2", "run2", "sha256:d2", 0.9)
	if _, err := mgr.ExtractHeuristic("skill", []memory.MemoryAtom{private, second}); !errors.Is(err, ErrPromotionUnauthorized) { t.Fatalf("private finding err=%v", err) }

	sameRun1 := evolutionGlobalFinding("a", "fa", "same-run", "sha256:a", 0.9)
	sameRun2 := evolutionGlobalFinding("b", "fb", "same-run", "sha256:b", 0.9)
	if _, err := mgr.ExtractHeuristic("skill", []memory.MemoryAtom{sameRun1, sameRun2}); err == nil { t.Fatalf("same run duplicated evidence unexpectedly produced heuristic") }

	sameEvidence1 := evolutionGlobalFinding("c", "fc", "run-c", "sha256:same", 0.9)
	sameEvidence2 := evolutionGlobalFinding("d", "fd", "run-d", "sha256:same", 0.9)
	if _, err := mgr.ExtractHeuristic("skill", []memory.MemoryAtom{sameEvidence1, sameEvidence2}); err == nil { t.Fatalf("same evidence digest duplicated across runs unexpectedly produced heuristic") }
}

func TestEvolutionPromotionRejectsEmptyCorpusAndCorpusDrift(t *testing.T) {
	mgr := NewEvolutionManager()
	ctx := context.Background()
	base := SkillVersion{VersionID: "s-v1", SkillID: "s", IsActive: true}
	heuristic, err := mgr.ExtractHeuristic("s", []memory.MemoryAtom{
		evolutionGlobalFinding("a", "fa", "r1", "sha256:e1", 0.9),
		evolutionGlobalFinding("b", "fb", "r2", "sha256:e2", 0.9),
	})
	if err != nil { t.Fatal(err) }
	candidate := mgr.CreateCandidateVersion(base, *heuristic)
	if report := mgr.RunReplayEval(ctx, candidate, nil); report.PassedGate { t.Fatalf("empty corpus must not pass: %#v", report) }
	if _, err := mgr.AuthorizePromotion(ctx, base, candidate, nil, "verifier"); err == nil { t.Fatalf("empty corpus authorization unexpectedly succeeded") }

	corpus := []ReplayCase{{CaseID: "a", InputCritique: design.CritiquePass{GroundedScore: 9}, BaselineScore: 8.5}}
	if _, err := mgr.AuthorizePromotion(ctx, base, candidate, corpus, "verifier"); err != nil { t.Fatal(err) }
	drifted := append([]ReplayCase(nil), corpus...)
	drifted[0].BaselineScore = 9.8
	if _, err := mgr.PromoteCandidate(ctx, candidate, drifted); !errors.Is(err, ErrPromotionUnauthorized) { t.Fatalf("corpus drift err=%v want ErrPromotionUnauthorized", err) }
}
