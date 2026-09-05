package evolution

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/memory"
	"github.com/Homiakus/UiUxMaster/internal/sideeffect"
)

// EvolutionManager orchestrates controlled skill evolution through replay,
// shadow/non-regression authorization, promotion and rollback.
type EvolutionManager struct {
	history        []SkillVersion
	authorizations map[string]PromotionAuthorization
}

func NewEvolutionManager() *EvolutionManager {
	return &EvolutionManager{history: make([]SkillVersion, 0), authorizations: make(map[string]PromotionAuthorization)}
}

func (m *EvolutionManager) ExtractHeuristic(skillID string, atoms []memory.MemoryAtom) (*CandidateHeuristic, error) {
	if len(atoms) == 0 {
		return nil, fmt.Errorf("%w: no admitted atoms provided", ErrInvalidCandidate)
	}
	var findingIDs []string
	var categories []string
	totalConf := 0.0
	count := 0
	for _, a := range atoms {
		if a.Kind == memory.NodeDesignFinding {
			if f, ok := a.Data.(memory.DesignFindingAtom); ok {
				findingIDs = append(findingIDs, f.FindingID)
				categories = append(categories, f.Category)
				totalConf += a.Confidence
				count++
			}
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("%w: no design finding atoms found", ErrInvalidCandidate)
	}
	avgConf := totalConf / float64(count)
	category := "general"
	if len(categories) > 0 {
		category = categories[0]
	}
	ruleID := fmt.Sprintf("heuristic_%s_%s", category, time.Now().Format("20060102150405"))
	rule := memory.DesignRuleAtom{RuleID: ruleID, Axis: category, Category: category, Title: fmt.Sprintf("Empirical heuristic for %s", category), Description: fmt.Sprintf("Admitted pattern derived from %d empirical findings", count), HardConstraint: false, Weight: 0.75, Version: "cand.1"}
	return &CandidateHeuristic{ID: fmt.Sprintf("heur_%s", ruleID), SkillID: skillID, Category: category, ProposedRule: rule, SourceFindingIDs: findingIDs, Confidence: avgConf, CreatedAt: time.Now()}, nil
}

func (m *EvolutionManager) CreateCandidateVersion(base SkillVersion, heuristic CandidateHeuristic) SkillVersion {
	newRules := append([]memory.DesignRuleAtom(nil), base.ActiveRules...)
	newRules = append(newRules, heuristic.ProposedRule)
	newHeuristics := append([]CandidateHeuristic(nil), base.Heuristics...)
	newHeuristics = append(newHeuristics, heuristic)
	versionNum := len(m.history) + 1
	candVersionID := fmt.Sprintf("%s-v1.%d.0-cand", base.SkillID, versionNum)
	return SkillVersion{VersionID: candVersionID, SkillID: base.SkillID, ActiveRules: newRules, ActiveRepairPatterns: append([]memory.RepairPatternAtom(nil), base.ActiveRepairPatterns...), Heuristics: newHeuristics, IsActive: false, CreatedAt: time.Now()}
}

// RunReplayEval never treats absence of evidence as success. Empty corpus is an
// explicit failed gate so future callers cannot accidentally authorize promotion.
func (m *EvolutionManager) RunReplayEval(ctx context.Context, version SkillVersion, corpus []ReplayCase) EvaluationReport {
	if err := ctx.Err(); err != nil {
		return EvaluationReport{VersionID: version.VersionID, PassedGate: false, Rationale: err.Error()}
	}
	if len(corpus) == 0 {
		return EvaluationReport{VersionID: version.VersionID, PassedGate: false, Rationale: "empty replay corpus cannot authorize promotion"}
	}
	passed, failed := 0, 0
	totalScore := 0.0
	hardViolations := 0
	var regressedAxes []string
	for _, c := range corpus {
		score := c.InputCritique.GroundedScore
		violations := c.InputCritique.HardViolations
		if violations > c.ExpectedHardViolations {
			failed++
			hardViolations += violations - c.ExpectedHardViolations
			regressedAxes = append(regressedAxes, "hard_constraints")
		} else if score < c.BaselineScore-0.5 {
			failed++
			regressedAxes = append(regressedAxes, "score_drop")
		} else {
			passed++
		}
		totalScore += score
	}
	avgScore := totalScore / float64(len(corpus))
	passedGate := failed == 0 && hardViolations == 0
	rationale := "Candidate passed all replay fixtures without regressions"
	if !passedGate {
		rationale = fmt.Sprintf("Candidate failed %d fixtures with %d hard violations", failed, hardViolations)
	}
	return EvaluationReport{VersionID: version.VersionID, TotalCases: len(corpus), PassedCases: passed, FailedCases: failed, AverageScore: avgScore, RegressedAxes: regressedAxes, HardViolations: hardViolations, PassedGate: passedGate, Rationale: rationale}
}

func (m *EvolutionManager) RunShadowEval(ctx context.Context, active SkillVersion, candidate SkillVersion, corpus []ReplayCase) (bool, string) {
	activeReport := m.RunReplayEval(ctx, active, corpus)
	candReport := m.RunReplayEval(ctx, candidate, corpus)
	if !activeReport.PassedGate {
		return false, fmt.Sprintf("active replay baseline is not valid: %s", activeReport.Rationale)
	}
	if !candReport.PassedGate {
		return false, fmt.Sprintf("candidate failed replay gate: %s", candReport.Rationale)
	}
	if candReport.HardViolations > activeReport.HardViolations {
		return false, "candidate introduced more hard violations than active version"
	}
	if candReport.AverageScore < activeReport.AverageScore-0.1 {
		return false, fmt.Sprintf("candidate average score (%.2f) regressed compared to active (%.2f)", candReport.AverageScore, activeReport.AverageScore)
	}
	return true, fmt.Sprintf("candidate verified (score: %.2f vs %.2f; 0 regressions)", candReport.AverageScore, activeReport.AverageScore)
}

func replayCorpusDigest(corpus []ReplayCase) (string, error) {
	return sideeffect.DigestJSON(corpus)
}

// AuthorizePromotion binds an exact active/candidate/corpus tuple to an
// independent verifier after replay and shadow evaluation. PromoteCandidate will
// fail if this authorization is absent or if the corpus changes afterward.
func (m *EvolutionManager) AuthorizePromotion(ctx context.Context, active SkillVersion, candidate SkillVersion, corpus []ReplayCase, verifierID string) (PromotionAuthorization, error) {
	if strings.TrimSpace(verifierID) == "" {
		return PromotionAuthorization{}, fmt.Errorf("%w: verifier id is required", ErrPromotionUnauthorized)
	}
	if candidate.IsActive || strings.TrimSpace(candidate.VersionID) == "" || candidate.SkillID != active.SkillID {
		return PromotionAuthorization{}, fmt.Errorf("%w: active/candidate identity mismatch", ErrInvalidCandidate)
	}
	replay := m.RunReplayEval(ctx, candidate, corpus)
	if !replay.PassedGate {
		return PromotionAuthorization{}, fmt.Errorf("%w: %s", ErrNonRegressionFailed, replay.Rationale)
	}
	shadowPassed, reason := m.RunShadowEval(ctx, active, candidate, corpus)
	if !shadowPassed {
		return PromotionAuthorization{}, fmt.Errorf("%w: %s", ErrNonRegressionFailed, reason)
	}
	digest, err := replayCorpusDigest(corpus)
	if err != nil {
		return PromotionAuthorization{}, err
	}
	auth := PromotionAuthorization{ActiveVersionID: active.VersionID, CandidateVersionID: candidate.VersionID, CorpusDigest: digest, VerifierID: strings.TrimSpace(verifierID), ReplayPassed: true, ShadowPassed: true, Rationale: reason, IssuedAt: time.Now()}
	m.authorizations[candidate.VersionID] = auth
	return auth, nil
}

// PromoteCandidate requires a prior exact authorization and reruns replay at the
// commit boundary. The old API remains intentionally narrow: no authorization,
// no promotion.
func (m *EvolutionManager) PromoteCandidate(ctx context.Context, candidate SkillVersion, corpus []ReplayCase) (SkillVersion, error) {
	if candidate.IsActive || strings.TrimSpace(candidate.VersionID) == "" {
		return candidate, ErrInvalidCandidate
	}
	if existing, ok := m.findHistory(candidate.VersionID); ok && existing.IsActive {
		return existing, nil
	}
	digest, err := replayCorpusDigest(corpus)
	if err != nil {
		return candidate, err
	}
	auth, ok := m.authorizations[candidate.VersionID]
	if !ok || auth.CandidateVersionID != candidate.VersionID || auth.CorpusDigest != digest || !auth.ReplayPassed || !auth.ShadowPassed || strings.TrimSpace(auth.VerifierID) == "" {
		return candidate, fmt.Errorf("%w: missing or stale authorization for %s", ErrPromotionUnauthorized, candidate.VersionID)
	}
	report := m.RunReplayEval(ctx, candidate, corpus)
	if !report.PassedGate {
		return candidate, fmt.Errorf("%w: %s", ErrNonRegressionFailed, report.Rationale)
	}
	promoted := candidate
	promoted.IsActive = true
	promoted.PromotedAt = time.Now()
	m.history = append(m.history, promoted)
	return promoted, nil
}

func (m *EvolutionManager) findHistory(versionID string) (SkillVersion, bool) {
	for i := len(m.history) - 1; i >= 0; i-- {
		if m.history[i].VersionID == versionID {
			return m.history[i], true
		}
	}
	return SkillVersion{}, false
}

func (m *EvolutionManager) Rollback(ctx context.Context, previous SkillVersion) (SkillVersion, error) {
	if err := ctx.Err(); err != nil {
		return previous, err
	}
	if strings.TrimSpace(previous.VersionID) == "" {
		return previous, ErrInvalidCandidate
	}
	restored := previous
	restored.IsActive = true
	restored.PromotedAt = time.Now()
	m.history = append(m.history, restored)
	return restored, nil
}
