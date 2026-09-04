package engine

import (
	"fmt"

	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
)

// EvidenceTier is the progressive validation ladder.
type EvidenceTier string

const (
	TierStatic      EvidenceTier = "L0_static"
	TierFastRender  EvidenceTier = "L1_fastrender"
	TierFastBrowser EvidenceTier = "L2_fastbrowser"
	TierTruthPath   EvidenceTier = "L3_truthpath"
	TierSemantic    EvidenceTier = "L4_semantic"
)

// EvidenceNeed states what must be proven, not which vendor should do it.
type EvidenceNeed struct {
	Pixels       bool `json:"pixels,omitempty"`
	Geometry     bool `json:"geometry,omitempty"`
	Styles       bool `json:"styles,omitempty"`
	Scenario     bool `json:"scenario,omitempty"`
	CleanState   bool `json:"clean_state,omitempty"`
	CrossBrowser bool `json:"cross_browser,omitempty"`
	Semantic     bool `json:"semantic,omitempty"`
}

// RouteDecision is explainable so MCP callers and Axiom runs can understand
// why escalation happened.
type RouteDecision struct {
	Tier    EvidenceTier `json:"tier"`
	Reasons []string     `json:"reasons,omitempty"`
}

// ValidationPlan represents the converged decision: the evidence shape to collect,
// the execution tier, and the underlying needs and fidelity assessment.
type ValidationPlan struct {
	Route        RouteDecision       `json:"route"`
	EvidencePlan evidenceplan.Plan   `json:"evidence_plan"`
	Need         EvidenceNeed        `json:"need"`
	Assessment   fidelity.Assessment `json:"assessment"`
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

// PlanValidationRoute coordinates evidenceplan and RouteValidation into a single
// converged policy path: deciding evidence shape, fidelity risk, and renderer tier.
func PlanValidationRoute(req ValidationRequest, assessment fidelity.Assessment, l1Caps fastrender.Capabilities) ValidationPlan {
	req.Normalize()
	need := req.DeriveNeed()
	signals := req.Signals(assessment.Risk)
	evidencePlan := evidenceplan.Build(signals)

	// Converge needs from evidence plan into EvidenceNeed
	if evidencePlan.Pixels {
		need.Pixels = true
	}
	if evidencePlan.Accessibility {
		need.Scenario = true
	}
	if evidencePlan.Fonts {
		need.Styles = true
	}

	route := RouteValidation(need, assessment, l1Caps)

	// Enforce L1 fastrender constraints: fastrender cannot collect accessibility or complex font state
	if route.Tier == TierFastRender {
		if evidencePlan.Accessibility {
			route = RouteDecision{Tier: TierFastBrowser, Reasons: []string{"accessibility_requires_browser"}}
		} else if evidencePlan.Fonts && !l1Caps.SupportsStyles {
			route = RouteDecision{Tier: TierFastBrowser, Reasons: []string{"l1_fonts_unsupported"}}
		}
	}

	// Calibrate evidence plan to the actual chosen execution tier:
	if route.Tier == TierFastRender {
		evidencePlan.BrowserTruth = false
		evidencePlan.Accessibility = false
		evidencePlan.Diagnostics = false
		evidencePlan.Structural = l1Caps.SupportsGeometry
		evidencePlan.Fonts = l1Caps.SupportsStyles
		evidencePlan.Pixels = true
	} else {
		evidencePlan.BrowserTruth = (route.Tier == TierFastBrowser || route.Tier == TierTruthPath)
	}

	allReasons := make([]string, 0, len(route.Reasons)+len(evidencePlan.Reasons))
	allReasons = append(allReasons, route.Reasons...)
	allReasons = append(allReasons, evidencePlan.Reasons...)
	route.Reasons = uniqueSorted(allReasons)

	return ValidationPlan{
		Route:        route,
		EvidencePlan: evidencePlan,
		Need:         need,
		Assessment:   assessment,
	}
}

// RouteEvidencePlan converges an existing evidenceplan.Plan with fidelity assessment and capabilities.
func RouteEvidencePlan(plan evidenceplan.Plan, assessment fidelity.Assessment, l1Caps fastrender.Capabilities) (evidenceplan.Plan, RouteDecision) {
	need := EvidenceNeed{
		Pixels:   plan.Pixels,
		Geometry: plan.Structural,
		Styles:   plan.Structural || plan.Fonts,
		Scenario: plan.Accessibility,
	}
	route := RouteValidation(need, assessment, l1Caps)
	if route.Tier == TierFastRender {
		if plan.Accessibility {
			route = RouteDecision{Tier: TierFastBrowser, Reasons: []string{"accessibility_requires_browser"}}
		} else if plan.Fonts && !l1Caps.SupportsStyles {
			route = RouteDecision{Tier: TierFastBrowser, Reasons: []string{"l1_fonts_unsupported"}}
		}
	}
	plan.BrowserTruth = (route.Tier == TierFastBrowser || route.Tier == TierTruthPath)
	allReasons := make([]string, 0, len(route.Reasons)+len(plan.Reasons))
	allReasons = append(allReasons, route.Reasons...)
	allReasons = append(allReasons, plan.Reasons...)
	route.Reasons = uniqueSorted(allReasons)
	return plan, route
}
