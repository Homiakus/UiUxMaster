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
// independent evidence. The optimization loop never receives the private held-out
// probes owned by the FinalGate.
type HeldOutEvaluationRequest struct {
	RunID         string
	CandidateHTML string
	CandidateCSS  string
	Candidate     evidence.Packet
	Profile       design.ProductProfile
	ProtectedAxes []string
}

// HeldOutReport is aggregate-only completion evidence. Individual private probe
// definitions remain inside the evaluator and are not fed back to the proposer.
type HeldOutReport struct {
	Total             int      `json:"total"`
	Passed            int      `json:"passed"`
	Failed            int      `json:"failed"`
	RegressionEscapes int      `json:"regression_escapes"`
	EscapeRate        float64  `json:"escape_rate"`
	FailedProbeIDs    []string `json:"failed_probe_ids,omitempty"`
}

// HeldOutEvaluator owns hidden/perturbed checks that are not part of the repair
// proposal signals.
type HeldOutEvaluator interface {
	Evaluate(context.Context, HeldOutEvaluationRequest) (HeldOutReport, error)
}

// HeldOutProbe is one private completion check. Implementations may own a second
// browser/scenario runner, perturb source/environment, or inspect protected-axis
// invariants. The repair proposer never receives this object.
type HeldOutProbe interface {
	ID() string
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
	report := HeldOutReport{Total: len(s.probes)}
	for _, probe := range s.probes {
		if err := ctx.Err(); err != nil {
			return HeldOutReport{}, err
		}
		if probe == nil {
			report.Failed++
			report.RegressionEscapes++
			report.FailedProbeIDs = append(report.FailedProbeIDs, "nil-probe")
			continue
		}
		if err := probe.Evaluate(ctx, req); err != nil {
			report.Failed++
			report.RegressionEscapes++
			report.FailedProbeIDs = append(report.FailedProbeIDs, probe.ID())
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
// verifier cannot accidentally reuse the reward signal as acceptance evidence.
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

// FinalGateResult is the independent completion record. Passed alone is not
// enough: HostRepairEngine also verifies that the gate is independent from the
// optimization pipeline before honoring it.
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
// distinct canonical Pipeline, an independent critic instance and private
// held-out probes.
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
	if g.Critic == nil {
		g.Critic = critic.New()
	}
	if g.Comparator == nil {
		g.Comparator = design.NewComparator()
	}
	if g.MinHeldOutCases <= 0 {
		g.MinHeldOutCases = 1
	}
	if g.MaxEscapeRate < 0 {
		g.MaxEscapeRate = 0
	}

	// FinalGate=true forces clean-state evidence. Combined with FMEA-001 this is
	// fail-closed: absence of L3 cannot silently become an L2 completion PASS.
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

	baseCritique, err := g.Critic.Critique(ctx, critic.CritiqueRequest{
		RunID:         fmt.Sprintf("%s-final-baseline", req.RunID),
		Profile:       req.Profile,
		Packet:        baselineRes.Packet,
		ProtectedAxes: req.ProtectedAxes,
	})
	if err != nil {
		return result, fmt.Errorf("repair final gate: baseline critique: %w", err)
	}
	candCritique, err := g.Critic.Critique(ctx, critic.CritiqueRequest{
		RunID:         fmt.Sprintf("%s-final-candidate", req.RunID),
		Profile:       req.Profile,
		Packet:        candidateRes.Packet,
		ProtectedAxes: req.ProtectedAxes,
	})
	if err != nil {
		return result, fmt.Errorf("repair final gate: candidate critique: %w", err)
	}
	result.HardViolations = candCritique.HardViolations

	comparison, err := g.Comparator.Compare(ctx, design.ComparisonRequest{
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

	if g.HeldOut == nil {
		result.ReasonCodes = append(result.ReasonCodes, "held_out_evaluator_unconfigured")
	} else {
		heldOut, err := g.HeldOut.Evaluate(ctx, HeldOutEvaluationRequest{
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
		if heldOut.Total < g.MinHeldOutCases {
			result.ReasonCodes = append(result.ReasonCodes, "held_out_coverage_insufficient")
		}
		if heldOut.Failed > 0 || heldOut.EscapeRate > g.MaxEscapeRate {
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
		result.HeldOut.Total >= g.MinHeldOutCases &&
		result.HeldOut.Failed == 0 &&
		result.HeldOut.EscapeRate <= g.MaxEscapeRate &&
		comparison.PassedConstraints &&
		comparison.PreferredCandidate == "candidate_repaired" &&
		candCritique.HardViolations == 0
	return result, nil
}
