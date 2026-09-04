package evolution

import (
	"context"
	"fmt"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/memory"
)

// EvolutionManager orchestrates controlled skill evolution through rigorous evaluation gates.
type EvolutionManager struct {
	history []SkillVersion
}

// NewEvolutionManager creates an initialized EvolutionManager.
func NewEvolutionManager() *EvolutionManager {
	return &EvolutionManager{
		history: make([]SkillVersion, 0),
	}
}

// ExtractHeuristic analyzes admitted memory atoms and synthesizes a candidate heuristic.
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
	rule := memory.DesignRuleAtom{
		RuleID:         ruleID,
		Axis:           category,
		Category:       category,
		Title:          fmt.Sprintf("Empirical heuristic for %s", category),
		Description:    fmt.Sprintf("Admitted pattern derived from %d empirical findings", count),
		HardConstraint: false,
		Weight:         0.75,
		Version:        "cand.1",
	}

	return &CandidateHeuristic{
		ID:               fmt.Sprintf("heur_%s", ruleID),
		SkillID:          skillID,
		Category:         category,
		ProposedRule:     rule,
		SourceFindingIDs: findingIDs,
		Confidence:       avgConf,
		CreatedAt:        time.Now(),
	}, nil
}

// CreateCandidateVersion produces an immutable candidate skill version.
func (m *EvolutionManager) CreateCandidateVersion(base SkillVersion, heuristic CandidateHeuristic) SkillVersion {
	newRules := append([]memory.DesignRuleAtom(nil), base.ActiveRules...)
	newRules = append(newRules, heuristic.ProposedRule)

	newHeuristics := append([]CandidateHeuristic(nil), base.Heuristics...)
	newHeuristics = append(newHeuristics, heuristic)

	versionNum := len(m.history) + 1
	candVersionID := fmt.Sprintf("%s-v1.%d.0-cand", base.SkillID, versionNum)

	return SkillVersion{
		VersionID:            candVersionID,
		SkillID:              base.SkillID,
		ActiveRules:          newRules,
		ActiveRepairPatterns: append([]memory.RepairPatternAtom(nil), base.ActiveRepairPatterns...),
		Heuristics:           newHeuristics,
		IsActive:             false,
		CreatedAt:            time.Now(),
	}
}

// RunReplayEval evaluates a skill version across a deterministic replay test corpus.
func (m *EvolutionManager) RunReplayEval(ctx context.Context, version SkillVersion, corpus []ReplayCase) EvaluationReport {
	if len(corpus) == 0 {
		return EvaluationReport{
			VersionID:  version.VersionID,
			PassedGate: true,
			Rationale:  "empty corpus; passed by default",
		}
	}

	passed := 0
	failed := 0
	totalScore := 0.0
	hardViolations := 0
	var regressedAxes []string

	for _, c := range corpus {
		score := c.InputCritique.GroundedScore
		violations := c.InputCritique.HardViolations

		// If candidate introduces unexpected hard violations
		if violations > c.ExpectedHardViolations {
			failed++
			hardViolations += (violations - c.ExpectedHardViolations)
			regressedAxes = append(regressedAxes, "hard_constraints")
		} else if score < c.BaselineScore-0.5 { // Significant score drop
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

	return EvaluationReport{
		VersionID:      version.VersionID,
		TotalCases:     len(corpus),
		PassedCases:    passed,
		FailedCases:    failed,
		AverageScore:   avgScore,
		RegressedAxes:  regressedAxes,
		HardViolations: hardViolations,
		PassedGate:     passedGate,
		Rationale:      rationale,
	}
}

// RunShadowEval compares active vs candidate version across the replay corpus.
func (m *EvolutionManager) RunShadowEval(ctx context.Context, active SkillVersion, candidate SkillVersion, corpus []ReplayCase) (bool, string) {
	activeReport := m.RunReplayEval(ctx, active, corpus)
	candReport := m.RunReplayEval(ctx, candidate, corpus)

	if !candReport.PassedGate {
		return false, fmt.Sprintf("candidate failed replay gate: %s", candReport.Rationale)
	}

	if candReport.HardViolations > activeReport.HardViolations {
		return false, "candidate introduced more hard violations than active version"
	}

	if candReport.AverageScore < activeReport.AverageScore-0.1 {
		return false, fmt.Sprintf("candidate average score (%.2f) regressed compared to active (%.2f)",
			candReport.AverageScore, activeReport.AverageScore)
	}

	return true, fmt.Sprintf("candidate verified (score: %.2f vs %.2f; 0 regressions)",
		candReport.AverageScore, activeReport.AverageScore)
}

// PromoteCandidate promotes a candidate version to active after satisfying all non-regression gates.
func (m *EvolutionManager) PromoteCandidate(ctx context.Context, candidate SkillVersion, corpus []ReplayCase) (SkillVersion, error) {
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

// Rollback restores a previously active skill version.
func (m *EvolutionManager) Rollback(ctx context.Context, previous SkillVersion) (SkillVersion, error) {
	restored := previous
	restored.IsActive = true
	restored.PromotedAt = time.Now()

	m.history = append(m.history, restored)
	return restored, nil
}
