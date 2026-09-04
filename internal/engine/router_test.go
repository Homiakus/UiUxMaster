package engine

import (
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
)

func TestRouteLowRiskPixelsToFastRender(t *testing.T) {
	got := RouteValidation(
		EvidenceNeed{Pixels: true},
		fidelity.Assessment{Risk: fidelity.RiskLow, MayVerify: true},
		fastrender.Capabilities{Name: "wggo", SupportsPixels: true},
	)
	if got.Tier != TierFastRender {
		t.Fatalf("tier = %s, want %s (%v)", got.Tier, TierFastRender, got.Reasons)
	}
}

func TestRouteWGGoGeometryToBrowserUntilSupported(t *testing.T) {
	got := RouteValidation(
		EvidenceNeed{Geometry: true},
		fidelity.Assessment{Risk: fidelity.RiskLow, MayVerify: true},
		fastrender.Capabilities{Name: "wggo", SupportsPixels: true, SupportsGeometry: false},
	)
	if got.Tier != TierFastBrowser {
		t.Fatalf("tier = %s, want %s", got.Tier, TierFastBrowser)
	}
}

func TestRouteHighRiskToBrowser(t *testing.T) {
	got := RouteValidation(
		EvidenceNeed{Pixels: true},
		fidelity.Assessment{
			Risk:                     fidelity.RiskHigh,
			MayVerify:                false,
			NeedsBrowserConfirmation: true,
			Reasons:                  []string{"high_fidelity_risk_requires_browser"},
		},
		fastrender.Capabilities{Name: "wggo", SupportsPixels: true},
	)
	if got.Tier != TierFastBrowser {
		t.Fatalf("tier = %s, want %s", got.Tier, TierFastBrowser)
	}
}

func TestRouteCleanCrossBrowserToTruthPath(t *testing.T) {
	for _, need := range []EvidenceNeed{{CleanState: true}, {CrossBrowser: true}} {
		got := RouteValidation(need, fidelity.Assessment{}, fastrender.Capabilities{})
		if got.Tier != TierTruthPath {
			t.Fatalf("need %#v routed to %s", need, got.Tier)
		}
	}
}

func TestRouteStaticWithoutRenderNeed(t *testing.T) {
	got := RouteValidation(EvidenceNeed{}, fidelity.Assessment{}, fastrender.Capabilities{})
	if got.Tier != TierStatic {
		t.Fatalf("tier = %s, want %s", got.Tier, TierStatic)
	}
}

func TestPlanValidationRoute_QuickStructuralEscalatesToFastBrowserIfL1LacksGeometry(t *testing.T) {
	req := ValidationRequest{
		RunID:  "run:1",
		Intent: "quick_structural",
	}
	assessment := fidelity.Assessment{Risk: fidelity.RiskLow, MayVerify: true}
	l1Caps := fastrender.Capabilities{Name: "wggo", SupportsPixels: true, SupportsGeometry: false}

	plan := PlanValidationRoute(req, assessment, l1Caps)

	if plan.Route.Tier != TierFastBrowser {
		t.Fatalf("tier = %s, want %s (%v)", plan.Route.Tier, TierFastBrowser, plan.Route.Reasons)
	}
	if !plan.EvidencePlan.Structural || !plan.EvidencePlan.Diagnostics {
		t.Fatalf("evidence plan = %#v", plan.EvidencePlan)
	}
	if !plan.EvidencePlan.BrowserTruth {
		t.Fatalf("expected browser truth to be true for L2 FastBrowser")
	}
}

func TestPlanValidationRoute_VisualRegion_FastRenderWhenSupported(t *testing.T) {
	req := ValidationRequest{
		RunID: "run:2",
		Need:  EvidenceNeed{Pixels: true},
	}
	assessment := fidelity.Assessment{Risk: fidelity.RiskLow, MayVerify: true}
	l1Caps := fastrender.Capabilities{Name: "wggo", SupportsPixels: true, SupportsGeometry: false}

	plan := PlanValidationRoute(req, assessment, l1Caps)

	if plan.Route.Tier != TierFastRender {
		t.Fatalf("tier = %s, want %s (%v)", plan.Route.Tier, TierFastRender, plan.Route.Reasons)
	}
	if !plan.EvidencePlan.Pixels {
		t.Fatalf("expected plan.Pixels to be true")
	}
	if plan.EvidencePlan.BrowserTruth {
		t.Fatalf("expected browser truth to be false for L1 FastRender")
	}
}

func TestPlanValidationRoute_InteractionRequiresBrowser(t *testing.T) {
	req := ValidationRequest{
		RunID:  "run:3",
		Intent: "interaction",
	}
	assessment := fidelity.Assessment{Risk: fidelity.RiskLow, MayVerify: true}
	l1Caps := fastrender.Capabilities{Name: "wggo", SupportsPixels: true, SupportsGeometry: true, SupportsStyles: true}

	plan := PlanValidationRoute(req, assessment, l1Caps)

	if plan.Route.Tier != TierFastBrowser {
		t.Fatalf("tier = %s, want %s (%v)", plan.Route.Tier, TierFastBrowser, plan.Route.Reasons)
	}
	if !plan.EvidencePlan.Accessibility {
		t.Fatalf("expected accessibility to be planned for interaction")
	}
	if !plan.EvidencePlan.BrowserTruth {
		t.Fatalf("expected browser truth to be true for browser tier")
	}
}

func TestPlanValidationRoute_FinalGateRoutesToTruthPath(t *testing.T) {
	req := ValidationRequest{
		RunID:     "run:4",
		FinalGate: true,
	}
	assessment := fidelity.Assessment{Risk: fidelity.RiskLow, MayVerify: true}
	l1Caps := fastrender.Capabilities{Name: "wggo", SupportsPixels: true, SupportsGeometry: true, SupportsStyles: true}

	plan := PlanValidationRoute(req, assessment, l1Caps)

	if plan.Route.Tier != TierTruthPath {
		t.Fatalf("tier = %s, want %s (%v)", plan.Route.Tier, TierTruthPath, plan.Route.Reasons)
	}
	if !plan.EvidencePlan.BrowserTruth {
		t.Fatalf("expected browser truth to be true for TruthPath")
	}
}

func TestPlanValidationRoute_FontsEscalateIfL1LacksStyles(t *testing.T) {
	req := ValidationRequest{
		RunID:         "run:5",
		Intent:        "typography",
		ChangedTokens: []string{"font.heading"},
	}
	assessment := fidelity.Assessment{Risk: fidelity.RiskLow, MayVerify: true}
	l1Caps := fastrender.Capabilities{Name: "wggo", SupportsPixels: true, SupportsGeometry: true, SupportsStyles: false}

	plan := PlanValidationRoute(req, assessment, l1Caps)

	if plan.Route.Tier != TierFastBrowser {
		t.Fatalf("tier = %s, want %s (%v)", plan.Route.Tier, TierFastBrowser, plan.Route.Reasons)
	}
	if !plan.EvidencePlan.Fonts {
		t.Fatalf("expected fonts to be planned")
	}
}

func TestRouteEvidencePlan_SynchronizesBrowserTruthAndReasons(t *testing.T) {
	ep, route := RouteEvidencePlan(
		evidenceplan.Plan{Pixels: true, Reasons: []string{"roi_pixels"}},
		fidelity.Assessment{Risk: fidelity.RiskLow, MayVerify: true},
		fastrender.Capabilities{Name: "wggo", SupportsPixels: true},
	)
	if route.Tier != TierFastRender {
		t.Fatalf("tier = %s, want %s", route.Tier, TierFastRender)
	}
	if ep.BrowserTruth {
		t.Fatalf("expected BrowserTruth = false for L1 FastRender")
	}
}
