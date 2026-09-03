package engine

import (
	"fmt"

	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
)

// EvidenceTier is the progressive validation ladder.
type EvidenceTier string

const (
	TierStatic    EvidenceTier = "L0_static"
	TierFastRender EvidenceTier = "L1_fastrender"
	TierFastBrowser EvidenceTier = "L2_fastbrowser"
	TierTruthPath EvidenceTier = "L3_truthpath"
	TierSemantic  EvidenceTier = "L4_semantic"
)

// EvidenceNeed states what must be proven, not which vendor should do it.
type EvidenceNeed struct {
	Pixels       bool
	Geometry     bool
	Styles       bool
	Scenario     bool
	CleanState   bool
	CrossBrowser bool
	Semantic     bool
}

// RouteDecision is explainable so MCP callers and Axiom runs can understand
// why escalation happened.
type RouteDecision struct {
	Tier    EvidenceTier `json:"tier"`
	Reasons []string     `json:"reasons,omitempty"`
}

// RouteValidation chooses the cheapest tier that can prove the requested
// evidence. l1Caps describes the configured FastRender candidate (WGGo today).
func RouteValidation(need EvidenceNeed, assessment fidelity.Assessment, l1Caps fastrender.Capabilities) RouteDecision {
	if need.Semantic {
		return RouteDecision{Tier: TierSemantic, Reasons: []string{"semantic_judgement_requested"}}
	}
	if need.CrossBrowser {
		return RouteDecision{Tier: TierTruthPath, Reasons: []string{"cross_browser_requires_truthpath"}}
	}
	if need.CleanState {
		return RouteDecision{Tier: TierTruthPath, Reasons: []string{"clean_state_requires_truthpath"}}
	}
	if need.Scenario {
		return RouteDecision{Tier: TierFastBrowser, Reasons: []string{"interaction_requires_browser"}}
	}

	requiresRender := need.Pixels || need.Geometry || need.Styles
	if !requiresRender {
		return RouteDecision{Tier: TierStatic, Reasons: []string{"static_evidence_sufficient"}}
	}

	if assessment.NeedsBrowserConfirmation || !assessment.MayVerify {
		reason := "fidelity_requires_browser"
		if len(assessment.Reasons) > 0 {
			reason = assessment.Reasons[0]
		}
		return RouteDecision{Tier: TierFastBrowser, Reasons: []string{reason}}
	}

	if need.Geometry && !l1Caps.SupportsGeometry {
		return RouteDecision{Tier: TierFastBrowser, Reasons: []string{"l1_geometry_unsupported"}}
	}
	if need.Styles && !l1Caps.SupportsStyles {
		return RouteDecision{Tier: TierFastBrowser, Reasons: []string{"l1_styles_unsupported"}}
	}
	if need.Pixels && !l1Caps.SupportsPixels {
		return RouteDecision{Tier: TierFastBrowser, Reasons: []string{"l1_pixels_unsupported"}}
	}

	return RouteDecision{
		Tier: TierFastRender,
		Reasons: []string{fmt.Sprintf("%s_can_prove_requested_low_risk_evidence", l1Caps.Name)},
	}
}
