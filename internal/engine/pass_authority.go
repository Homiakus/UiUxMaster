package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
)

// CalibrationContextProvider supplies the exact approximate-vs-TruthPath runtime
// pair needed to decide whether an L1/L2 result is still legally calibrated.
// The canonical Pipeline owns PASS semantics; runtime collectors only attest
// environment identity.
type CalibrationContextProvider interface {
	CalibrationContext(context.Context, ValidationRequest, ValidationPlan, evidence.Packet) (fidelity.CalibrationContext, error)
}

// PassAuthority is separate from deterministic verification. A packet can be
// useful diagnostic evidence while still lacking authority to issue PASS.
type PassAuthority struct {
	Required           bool                     `json:"required"`
	Allowed            bool                     `json:"allowed"`
	Tier               fidelity.Tier            `json:"tier,omitempty"`
	Classes            []fidelity.EvidenceClass `json:"classes,omitempty"`
	CalibrationKeys    []string                 `json:"calibration_keys,omitempty"`
	RequiredEscalation fidelity.Tier            `json:"required_escalation,omitempty"`
	Reasons            []string                 `json:"reasons,omitempty"`
}

func calibrationClasses(req ValidationRequest, plan ValidationPlan) []fidelity.EvidenceClass {
	set := map[fidelity.EvidenceClass]struct{}{}
	if req.FinalGate || plan.Need.CrossBrowser {
		set[fidelity.EvidenceClassCrossBrowserRelease] = struct{}{}
	}
	if plan.Need.Scenario || plan.EvidencePlan.Accessibility {
		set[fidelity.EvidenceClassInteractive] = struct{}{}
	}
	if plan.Need.Styles || plan.EvidencePlan.Fonts {
		set[fidelity.EvidenceClassTypography] = struct{}{}
	}
	if plan.Need.Pixels || plan.EvidencePlan.Pixels {
		set[fidelity.EvidenceClassPixelRegression] = struct{}{}
	}
	if plan.Need.Geometry || plan.EvidencePlan.Structural {
		set[fidelity.EvidenceClassStaticLayout] = struct{}{}
	}
	if len(set) == 0 {
		set[fidelity.EvidenceClassStaticLayout] = struct{}{}
	}
	out := make([]fidelity.EvidenceClass, 0, len(set))
	for class := range set {
		out = append(out, class)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func packetFidelityTier(packet evidence.Packet) (fidelity.Tier, error) {
	strength, err := PacketEvidenceStrength(packet.Renderer.Tier)
	if err != nil {
		return "", err
	}
	switch strength {
	case StrengthStatic:
		return fidelity.TierL0, nil
	case StrengthFastRender:
		return fidelity.TierL1, nil
	case StrengthFastBrowser:
		return fidelity.TierL2, nil
	case StrengthTruthPath, StrengthSemantic:
		return fidelity.TierL3, nil
	default:
		return "", fmt.Errorf("engine: unsupported pass-authority evidence strength %d", strength)
	}
}

func tierRank(t fidelity.Tier) int {
	switch t {
	case fidelity.TierL0:
		return 0
	case fidelity.TierL1:
		return 1
	case fidelity.TierL2:
		return 2
	case fidelity.TierL3:
		return 3
	default:
		return -1
	}
}

func strongerEscalation(current fidelity.Tier, requested fidelity.Tier) fidelity.Tier {
	if current == fidelity.TierL2 {
		return fidelity.TierL3
	}
	if current == fidelity.TierL1 && tierRank(requested) <= tierRank(fidelity.TierL1) {
		return fidelity.TierL2
	}
	if current == fidelity.TierL0 && tierRank(requested) <= tierRank(fidelity.TierL0) {
		return fidelity.TierL1
	}
	if tierRank(requested) > tierRank(current) {
		return requested
	}
	return fidelity.TierL3
}

// EvaluatePassAuthority evaluates static tier legality plus exact runtime parity
// calibration. L3 is authoritative without approximate-tier calibration; L1/L2
// require a current matching record for every evidence class they claim to prove.
func EvaluatePassAuthority(ctx context.Context, req ValidationRequest, plan ValidationPlan, packet evidence.Packet, matrix *fidelity.CalibrationMatrix, authority *fidelity.CalibrationAuthority, provider CalibrationContextProvider) PassAuthority {
	result := PassAuthority{Required: req.RequireLegalPass || req.FinalGate}
	if !result.Required {
		return result
	}
	if matrix == nil {
		matrix = fidelity.DefaultCalibrationMatrix()
	}
	classes := calibrationClasses(req, plan)
	result.Classes = classes

	tier, err := packetFidelityTier(packet)
	if err != nil {
		result.Reasons = []string{"unknown_evidence_tier"}
		result.RequiredEscalation = fidelity.TierL3
		return result
	}
	result.Tier = tier

	for _, class := range classes {
		if !matrix.CanLegallyPass(tier, class, req.FinalGate) {
			result.Reasons = append(result.Reasons, "tier_not_legal_for_"+string(class))
			escalation := strongerEscalation(tier, matrix.RequiredEscalationTier(class, req.FinalGate))
			if tierRank(escalation) > tierRank(result.RequiredEscalation) {
				result.RequiredEscalation = escalation
			}
		}
	}
	if len(result.Reasons) > 0 {
		result.Reasons = uniqueReasonCodes(result.Reasons)
		return result
	}

	if tier == fidelity.TierL3 {
		result.Allowed = true
		return result
	}
	if tier == fidelity.TierL0 {
		result.Reasons = []string{"static_evidence_has_no_runtime_pass_authority"}
		result.RequiredEscalation = fidelity.TierL1
		return result
	}
	if authority == nil {
		result.Reasons = []string{"calibration_authority_unconfigured"}
		result.RequiredEscalation = strongerEscalation(tier, fidelity.TierL2)
		return result
	}
	if provider == nil {
		result.Reasons = []string{"calibration_context_provider_unavailable"}
		result.RequiredEscalation = strongerEscalation(tier, fidelity.TierL2)
		return result
	}
	current, err := provider.CalibrationContext(ctx, req, plan, packet)
	if err != nil {
		result.Reasons = []string{"calibration_context_unavailable"}
		result.RequiredEscalation = strongerEscalation(tier, fidelity.TierL2)
		return result
	}

	for _, class := range classes {
		record, err := authority.Validate(class, tier, current)
		if err != nil {
			code := "calibration_invalid"
			switch {
			case fidelity.IsCalibrationMissing(err):
				code = "calibration_missing"
			case fidelity.IsCalibrationEnvironmentMismatch(err):
				code = "calibration_environment_mismatch"
			case fidelity.IsCalibrationExpired(err):
				code = "calibration_expired"
			case fidelity.IsCalibrationCoverage(err):
				code = "calibration_coverage_insufficient"
			case fidelity.IsCalibrationQuality(err):
				code = "calibration_quality_insufficient"
			}
			result.Reasons = append(result.Reasons, code+"_for_"+string(class))
			result.RequiredEscalation = strongerEscalation(tier, matrix.RequiredEscalationTier(class, req.FinalGate))
			continue
		}
		result.CalibrationKeys = append(result.CalibrationKeys, record.EnvironmentKey)
	}
	result.Reasons = uniqueReasonCodes(result.Reasons)
	result.CalibrationKeys = uniqueStrings(result.CalibrationKeys)
	result.Allowed = len(result.Reasons) == 0
	return result
}

func uniqueReasonCodes(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func uniqueStrings(values []string) []string { return uniqueReasonCodes(values) }
