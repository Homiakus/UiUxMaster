package design

import (
	"context"
	"fmt"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// ComparisonRequest encapsulates all inputs required for a pairwise A/B candidate evaluation.
type ComparisonRequest struct {
	RunID                string           `json:"run_id"`
	BaselineID           string           `json:"baseline_id"`
	CandidateID          string           `json:"candidate_id"`
	BaselinePacket       evidence.Packet  `json:"baseline_packet"`
	CandidatePacket      evidence.Packet  `json:"candidate_packet"`
	BaselineCritique     *CritiquePass    `json:"baseline_critique,omitempty"`
	CandidateCritique    *CritiquePass    `json:"candidate_critique,omitempty"`
	Rubric               []Axis           `json:"rubric,omitempty"`
	ProtectedAxes        []string         `json:"protected_axes,omitempty"`
	MaxAllowedRegression float64          `json:"max_allowed_regression,omitempty"` // allowed regression margin (default 0.2)
}

// Comparator defines the interface for evaluating candidate UI revisions against a baseline.
type Comparator interface {
	Compare(ctx context.Context, req ComparisonRequest) (CandidateComparison, error)
}

// RelativeComparator implements relative candidate evaluation against rubrics, hard constraints, and protected axes.
type RelativeComparator struct{}

// NewComparator creates an initialized RelativeComparator.
func NewComparator() *RelativeComparator {
	return &RelativeComparator{}
}

// Compare executes pairwise evaluation between baseline and candidate evidence.
func (c *RelativeComparator) Compare(ctx context.Context, req ComparisonRequest) (CandidateComparison, error) {
	if err := ctx.Err(); err != nil {
		return CandidateComparison{}, err
	}

	if req.BaselineID == "" {
		req.BaselineID = "baseline"
	}
	if req.CandidateID == "" {
		req.CandidateID = "candidate"
	}
	if req.RunID == "" {
		req.RunID = fmt.Sprintf("cmp:%s-vs-%s", req.BaselineID, req.CandidateID)
	}
	if len(req.Rubric) == 0 {
		req.Rubric = DefaultRubric()
	}
	if req.MaxAllowedRegression <= 0 {
		req.MaxAllowedRegression = 0.2
	}

	// 1. Analyze Hard Constraints on Candidate and Baseline
	candHardViolations, candViolationsDesc := evaluateHardConstraints(req.CandidatePacket, req.CandidateCritique)
	baseHardViolations, _ := evaluateHardConstraints(req.BaselinePacket, req.BaselineCritique)

	// 2. Score Per-Axis
	axisScores := make([]AxisScore, 0, len(req.Rubric))
	regressedAxes := make([]string, 0)
	var totalBaselineWeighted, totalCandidateWeighted, totalWeight float64

	for _, axis := range req.Rubric {
		baseScore := scoreAxis(axis.ID, req.BaselinePacket, req.BaselineCritique)
		candScore := scoreAxis(axis.ID, req.CandidatePacket, req.CandidateCritique)

		pref := "neutral"
		rationale := fmt.Sprintf("%s: baseline=%.1f, candidate=%.1f", axis.Name, baseScore, candScore)

		diff := candScore - baseScore
		if diff > 0.3 {
			pref = req.CandidateID
			rationale = fmt.Sprintf("%s improved by +%.1f (%.1f -> %.1f)", axis.Name, diff, baseScore, candScore)
		} else if diff < -0.3 {
			pref = req.BaselineID
			rationale = fmt.Sprintf("%s regressed by %.1f (%.1f -> %.1f)", axis.Name, diff, baseScore, candScore)
		}

		// Check protected axes regression
		if isProtected(axis.ID, req.ProtectedAxes) && diff < -req.MaxAllowedRegression {
			regressedAxes = append(regressedAxes, axis.ID)
		}

		axisScores = append(axisScores, AxisScore{
			AxisID:     axis.ID,
			Baseline:   baseScore,
			Candidate:  candScore,
			Preference: pref,
			Rationale:  rationale,
		})

		weight := axis.Weight
		if weight <= 0 {
			weight = 1.0
		}
		totalBaselineWeighted += baseScore * weight
		totalCandidateWeighted += candScore * weight
		totalWeight += weight
	}

	baseAvg := totalBaselineWeighted / totalWeight
	candAvg := totalCandidateWeighted / totalWeight

	// 3. Determine Constraints Compliance
	passedConstraints := true
	if candHardViolations > 0 && candHardViolations >= baseHardViolations {
		passedConstraints = false
	}
	if len(regressedAxes) > 0 {
		passedConstraints = false
	}

	// 4. Select Preferred Candidate and formulate Rationale
	preferred := req.BaselineID
	var summaryRationale string

	if !passedConstraints {
		preferred = req.BaselineID
		var reasons []string
		if candHardViolations > 0 {
			reasons = append(reasons, fmt.Sprintf("%d hard constraint violation(s) [%s]", candHardViolations, strings.Join(candViolationsDesc, ", ")))
		}
		if len(regressedAxes) > 0 {
			reasons = append(reasons, fmt.Sprintf("regressed protected axis/axes: %s", strings.Join(regressedAxes, ", ")))
		}
		summaryRationale = fmt.Sprintf("Candidate rejected due to constraint failures: %s", strings.Join(reasons, "; "))
	} else if candAvg > baseAvg+0.1 {
		preferred = req.CandidateID
		summaryRationale = fmt.Sprintf("Candidate preferred (weighted quality %.2f vs baseline %.2f) with zero hard constraint regressions.", candAvg, baseAvg)
	} else if baseAvg > candAvg+0.1 {
		preferred = req.BaselineID
		summaryRationale = fmt.Sprintf("Baseline preferred (weighted quality %.2f vs candidate %.2f).", baseAvg, candAvg)
	} else {
		preferred = req.BaselineID // conservative tie-breaking prefers baseline
		summaryRationale = fmt.Sprintf("No significant quality difference (candidate %.2f vs baseline %.2f); baseline preserved.", candAvg, baseAvg)
	}

	return CandidateComparison{
		RunID:             req.RunID,
		BaselineID:        req.BaselineID,
		CandidateID:       req.CandidateID,
		AxisScores:        axisScores,
		PreferredCandidate: preferred,
		Rationale:         summaryRationale,
		HardViolations:    candHardViolations,
		RegressedAxes:     regressedAxes,
		PassedConstraints: passedConstraints,
	}, nil
}

func isProtected(axis string, protected []string) bool {
	for _, p := range protected {
		if strings.EqualFold(axis, p) {
			return true
		}
	}
	return false
}

func evaluateHardConstraints(packet evidence.Packet, critique *CritiquePass) (int, []string) {
	violations := 0
	var desc []string

	// Check runtime issues
	for _, issue := range packet.RuntimeIssues {
		if issue.Severity == evidence.SeverityCritical || issue.Severity == evidence.SeverityHigh {
			violations++
			desc = append(desc, fmt.Sprintf("runtime issue: %s", issue.Message))
		}
	}

	// Check responsive viewport overflow
	for _, doc := range packet.Documents {
		if packet.Viewport.Width > 0 && doc.ContentWidth > float64(packet.Viewport.Width) {
			violations++
			desc = append(desc, fmt.Sprintf("horizontal overflow (content %.0fpx > viewport %dpx)", doc.ContentWidth, packet.Viewport.Width))
		}
	}

	// Check font failures
	if packet.Fonts != nil {
		for _, f := range packet.Fonts.Faces {
			if strings.EqualFold(f.Status, "error") {
				violations++
				desc = append(desc, fmt.Sprintf("font error: %s", f.Family))
			}
		}
	}

	// Check critique hard violations
	if critique != nil {
		for _, f := range critique.Findings {
			if f.HardConstraint {
				violations++
				desc = append(desc, fmt.Sprintf("design finding: %s", f.Title))
			}
		}
	}

	return violations, desc
}

func scoreAxis(axis string, packet evidence.Packet, critique *CritiquePass) float64 {
	baseScore := 8.5 // neutral default baseline score

	switch axis {
	case "accessibility":
		for _, el := range packet.Elements {
			if el.Visible && (el.Clickable || el.Role == "button" || el.Role == "link") && el.Name == "" {
				baseScore -= 1.5
			}
		}
		for _, issue := range packet.RuntimeIssues {
			if strings.Contains(strings.ToLower(issue.Code), "a11y") || strings.Contains(strings.ToLower(issue.Code), "aria") {
				baseScore -= 2.0
			}
		}
	case "responsive":
		for _, doc := range packet.Documents {
			if packet.Viewport.Width > 0 && doc.ContentWidth > float64(packet.Viewport.Width) {
				baseScore -= 3.0
			}
		}
	case "typography":
		h1Count := 0
		for _, el := range packet.Elements {
			if strings.EqualFold(el.Tag, "h1") || (strings.EqualFold(el.Role, "heading") && el.Attributes["aria-level"] == "1") {
				h1Count++
			}
		}
		if h1Count != 1 {
			baseScore -= 1.5
		}
		if packet.Fonts != nil {
			for _, f := range packet.Fonts.Faces {
				if strings.EqualFold(f.Status, "error") {
					baseScore -= 2.5
				}
			}
		}
	case "interaction":
		for _, issue := range packet.RuntimeIssues {
			if strings.Contains(strings.ToLower(issue.Code), "console") || strings.Contains(strings.ToLower(issue.Code), "error") {
				baseScore -= 2.0
			}
		}
	}

	if critique != nil {
		for _, f := range critique.Findings {
			if strings.EqualFold(f.Axis, axis) {
				switch f.Severity {
				case evidence.SeverityCritical:
					baseScore -= 2.5
				case evidence.SeverityHigh:
					baseScore -= 1.5
				case evidence.SeverityMedium:
					baseScore -= 0.8
				case evidence.SeverityLow:
					baseScore -= 0.3
				}
			}
		}
	}

	if baseScore < 0 {
		return 0
	}
	if baseScore > 10 {
		return 10
	}
	return baseScore
}
