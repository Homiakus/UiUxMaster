package repair

import (
	"context"
	"fmt"

	"github.com/Homiakus/UiUxMaster/internal/critic"
	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// RepairRiskClass controls the minimum independence/fidelity required before an
// autonomous repair may be accepted as complete.
type RepairRiskClass string

const (
	RepairRiskLow      RepairRiskClass = "low"
	RepairRiskHigh     RepairRiskClass = "high"
	RepairRiskCritical RepairRiskClass = "critical"
)

// HeldOutEvaluationRequest intentionally contains only the candidate and final
// independent evidence. The optimization loop never receives private probes.
type HeldOutEvaluationRequest struct {
	RunID         string
	CandidateHTML string
	CandidateCSS  string
	Candidate     evidence.Packet
	Profile       design.ProductProfile
	ProtectedAxes []string
}

// HeldOutReport deliberately exposes aggregate statistics only. Probe identity,
// predicates and failure details stay private so a subsequent repair iteration
// cannot optimize directly against the held-out set.
type HeldOutReport struct {
	Total             int     `json:"total"`
	Passed            int     `json:"passed"`
	Failed            int     `json:"failed"`
	RegressionEscapes int     `json:"regression_escapes"`
	EscapeRate        float64 `json:"escape_rate"`
}

// HeldOutEvaluator owns hidden/perturbed checks that are not part of repair
// proposal signals.
type HeldOutEvaluator interface {
	Evaluate(context.Context, HeldOutEvaluationRequest) (HeldOutReport, error)
}

// HeldOutProbe is one private completion check. No public probe ID is required:
// the proposer receives only aggregate HeldOutReport statistics.
type HeldOutProbe interface {
	Evaluate(context.Context, HeldOutEvaluationRequest) error
}

// PrivateHeldOutSuite executes private probes and exposes only aggregate outcome.
type PrivateHeldOutSuite struct {
	probes []HeldOutProbe
}

func NewPrivateHeldOutSuite(probes ...HeldOutProbe) *PrivateHeldOutSuite {
	return &PrivateHeldOutSuite{probes: append([]HeldOutProbe(nil), probes...)}
}

func (s *PrivateHeldOutSuite) Evaluate(ctx context.Context, req HeldOutEvaluationRequest) (HeldOutReport, error) {
	if s == nil {
		return HeldOutReport{}, nil
	}
	report := HeldOutReport{Total: len(s.probes)}
	for _, probe := range s.probes {
		if err := ctx.Err(); err != nil {
			return HeldOutReport{}, err
		}
		if probe == nil {
			report.Failed++
			report.RegressionEscapes++
			continue
		}
		if err := probe.Evaluate(ctx, req); err != nil {
			report.Failed++
			report.RegressionEscapes++
			continue
		}
		report.Passed++
	}
	if report.Total > 0 {
		report.EscapeRate = float64(report.RegressionEscapes) / float64(report.Total)
	}
	return report, nil
}

// FinalVerificationRequest is the only input accepted by completion authority.
// It deliberately excludes optimization critique/candidate scores so the final
// verifier cannot reuse the reward signal as acceptance evidence.
type FinalVerificationRequest struct {
	RunID         string
	ProjectID     string
	BaselineHTML  string
	BaselineCSS   string
	CandidateHTML string
	CandidateCSS  string
	Profile       design.ProductProfile
	ProtectedAxes []string
	RiskClass     RepairRiskClass
}

// FinalGateResult is the independent completion record.
type FinalGateResult struct {
	VerifierID               string                     `json:"verifier_id"`
	Independent              bool                       `json:"independent"`
	Passed                   bool                       `json:"passed"`
	EvidenceTier             string                     `json:"evidence_tier,omitempty"`
	BaselineEvidenceTier     string                     `json:"baseline_evidence_tier,omitempty"`
	HardViolations           int                        `json:"hard_violations"`
	ProtectedAxisRegressions []string                   `json:"protected_axis_regressions,omitempty"`
	HeldOut                  HeldOutReport              `json:"held_out"`
	Comparison               design.CandidateComparison `json:"comparison"`
	ReasonCodes              []string                   `json:"reason_codes,omitempty"`
}

// FinalGate is the sole authority allowed to turn an improved repair candidate
// into Passed=true. Candidate generation/comparison can veto completion but can
// never grant it.
type FinalGate interface {
	Verify(context.Context, FinalVerificationRequest) (FinalGateResult, error)
	IndependentFrom(optimization *engine.Pipeline) bool
	VerifierID() string
}

// PipelineFinalGate performs a clean-state, final-gate re-capture using a
// distinct canonical Pipeline, an independent critic and private held-out probes.
type PipelineFinalGate struct {
	Pipeline        *engine.Pipeline
	Critic          critic.Critic
	Comparator      design.Comparator
	HeldOut         HeldOutEvaluator
	ID              string
	MinHeldOutCases int
	MaxEscapeRate   float64
}

func NewPipelineFinalGate(pipeline *engine.Pipeline, heldOut HeldOutEvaluator) *PipelineFinalGate {
	return &PipelineFinalGate{
		Pipeline:        pipeline,
		Critic:          critic.New(),
		Comparator:      design.NewComparator(),
		HeldOut:         heldOut,
		ID:              "pipeline-independent-final-gate",
		MinHeldOutCases: 1,
		MaxEscapeRate:   0,
	}
}

func (g *PipelineFinalGate) VerifierID() string {
	if g == nil || g.ID == "" {
		return "pipeline-independent-final-gate"
	}
	return g.ID
}

func (g *PipelineFinalGate) IndependentFrom(optimization *engine.Pipeline) bool {
	return g != nil && g.Pipeline != nil && g.Pipeline != optimization
}

func (g *PipelineFinalGate) Verify(ctx context.Context, req FinalVerificationRequest) (FinalGateResult, error) {
	result := FinalGateResult{VerifierID: g.VerifierID(), Independent: false}
	if g == nil || g.Pipeline == nil {
		result.ReasonCodes = append(result.ReasonCodes, "final_pipeline_unconfigured")
		return result, nil
	}

	// Snapshot configuration into locals. Verify is read-only with respect to the
	// gate object, so a gate can safely be shared by concurrent repair requests.
	criticImpl := g.Critic
	if criticImpl == nil {
		criticImpl = critic.New()
	}
	comparator := g.Comparator
	if comparator == nil {
		comparator = design.NewComparator()
	}
	minHeldOutCases := g.MinHeldOutCases
	if minHeldOutCases <= 0 {
		minHeldOutCases = 1
	}
	maxEscapeRate := g.MaxEscapeRate
	if maxEscapeRate < 0 {
		maxEscapeRate = 0
	}
	heldOutEvaluator := g.HeldOut

	// FinalGate=true forces clean-state evidence. Combined with FMEA-001 this is
	// fail-closed: absent L3 cannot silently become an L2 completion PASS.
	baselineRes, err := g.Pipeline.Execute(ctx, engine.ValidationRequest{
		RunID:     fmt.Sprintf("%s-final-baseline", req.RunID),
		ProjectID: req.ProjectID,
		FinalGate: true,
		Need: engine.EvidenceNeed{
			Geometry: true,
			Styles:   true,
			Scenario: true,
		},
		HTML: []byte(req.BaselineHTML),
		CSS:  []byte(req.BaselineCSS),
	})
	if err != nil {
		return result, fmt.Errorf("repair final gate: baseline verification: %w", err)
	}
	candidateRes, err := g.Pipeline.Execute(ctx, engine.ValidationRequest{
		RunID:     fmt.Sprintf("%s-final-candidate", req.RunID),
		ProjectID: req.ProjectID,
		FinalGate: true,
		Need: engine.EvidenceNeed{
			Geometry: true,
			Styles:   true,
			Scenario: true,
		},
		HTML: []byte(req.CandidateHTML),
		CSS:  []byte(req.CandidateCSS),
	})
	if err != nil {
		return result, fmt.Errorf("repair final gate: candidate verification: %w", err)
	}

	result.BaselineEvidenceTier = baselineRes.Packet.Renderer.Tier
	result.EvidenceTier = candidateRes.Packet.Renderer.Tier
	result.Independent = true

	baseCritique, err := criticImpl.Critique(ctx, critic.CritiqueRequest{
		RunID:         fmt.Sprintf("%s-final-baseline", req.RunID),
		Profile:       req.Profile,
		Packet:        baselineRes.Packet,
		ProtectedAxes: req.ProtectedAxes,
	})
	if err != nil {
		return result, fmt.Errorf("repair final gate: baseline critique: %w", err)
	}
	candCritique, err := criticImpl.Critique(ctx, critic.CritiqueRequest{
		RunID:         fmt.Sprintf("%s-final-candidate", req.RunID),
		Profile:       req.Profile,
		Packet:        candidateRes.Packet,
		ProtectedAxes: req.ProtectedAxes,
	})
	if err != nil {
		return result, fmt.Errorf("repair final gate: candidate critique: %w", err)
	}
	result.HardViolations = candCritique.HardViolations

	comparison, err := comparator.Compare(ctx, design.ComparisonRequest{
		RunID:             fmt.Sprintf("%s-independent-final-comparison", req.RunID),
		BaselineID:        "independent_baseline",
		CandidateID:       "candidate_repaired",
		BaselinePacket:    baselineRes.Packet,
		CandidatePacket:   candidateRes.Packet,
		BaselineCritique:  &baseCritique,
		CandidateCritique: &candCritique,
		ProtectedAxes:     req.ProtectedAxes,
	})
	if err != nil {
		return result, fmt.Errorf("repair final gate: compare: %w", err)
	}
	result.Comparison = comparison
	result.ProtectedAxisRegressions = append([]string(nil), comparison.RegressedAxes...)

	if heldOutEvaluator == nil {
		result.ReasonCodes = append(result.ReasonCodes, "held_out_evaluator_unconfigured")
	} else {
		heldOut, err := heldOutEvaluator.Evaluate(ctx, HeldOutEvaluationRequest{
			RunID:         req.RunID,
			CandidateHTML: req.CandidateHTML,
			CandidateCSS:  req.CandidateCSS,
			Candidate:     candidateRes.Packet,
			Profile:       req.Profile,
			ProtectedAxes: append([]string(nil), req.ProtectedAxes...),
		})
		if err != nil {
			return result, fmt.Errorf("repair final gate: held-out evaluation: %w", err)
		}
		result.HeldOut = heldOut
		if heldOut.Total < minHeldOutCases {
			result.ReasonCodes = append(result.ReasonCodes, "held_out_coverage_insufficient")
		}
		if heldOut.Failed > 0 || heldOut.EscapeRate > maxEscapeRate {
			result.ReasonCodes = append(result.ReasonCodes, "held_out_regression_escape")
		}
	}

	if candCritique.HardViolations > 0 {
		result.ReasonCodes = append(result.ReasonCodes, "final_hard_violation")
	}
	if len(comparison.RegressedAxes) > 0 {
		result.ReasonCodes = append(result.ReasonCodes, "protected_axis_regression")
	}
	if !comparison.PassedConstraints || comparison.PreferredCandidate != "candidate_repaired" {
		result.ReasonCodes = append(result.ReasonCodes, "independent_comparison_rejected")
	}

	result.Passed = result.Independent &&
		len(result.ReasonCodes) == 0 &&
		result.HeldOut.Total >= minHeldOutCases &&
		result.HeldOut.Failed == 0 &&
		result.HeldOut.EscapeRate <= maxEscapeRate &&
		comparison.PassedConstraints &&
		comparison.PreferredCandidate == "candidate_repaired" &&
		candCritique.HardViolations == 0
	return result, nil
}
