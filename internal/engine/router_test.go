package engine

import (
	"testing"

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
