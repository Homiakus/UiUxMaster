package fidelity

import (
	"reflect"
	"testing"
)

func TestApproximateRendererCannotProveHighRiskFeature(t *testing.T) {
	got := Assess([]Feature{FeatureCSSFilter}, RendererCapabilities{
		Name:            "wggo",
		BrowserAccurate: false,
		Supported:       map[Feature]bool{FeatureCSSFilter: true},
	})
	if got.Risk != RiskHigh || got.MayVerify || !got.NeedsBrowserConfirmation {
		t.Fatalf("unexpected assessment: %#v", got)
	}
}

func TestApproximateRendererMayInspectSupportedMediumRiskButNeedsPolicyConfirmation(t *testing.T) {
	got := Assess([]Feature{FeatureCustomFont}, RendererCapabilities{
		Name:            "wggo",
		BrowserAccurate: false,
		Supported:       map[Feature]bool{FeatureCustomFont: true},
	})
	if got.Risk != RiskMedium || !got.MayVerify || !got.NeedsBrowserConfirmation {
		t.Fatalf("unexpected assessment: %#v", got)
	}
}

func TestUnsupportedFeatureForcesEscalation(t *testing.T) {
	got := Assess([]Feature{FeatureTransform}, RendererCapabilities{
		Name:            "limited",
		BrowserAccurate: false,
		Supported:       map[Feature]bool{},
	})
	if got.MayVerify || !got.NeedsBrowserConfirmation {
		t.Fatalf("unexpected assessment: %#v", got)
	}
	if !reflect.DeepEqual(got.Unsupported, []Feature{FeatureTransform}) {
		t.Fatalf("unsupported = %#v", got.Unsupported)
	}
}

func TestBrowserAccurateRendererCanProveSupportedHighRiskFeature(t *testing.T) {
	got := Assess([]Feature{FeatureCanvas}, RendererCapabilities{
		Name:            "chromium",
		BrowserAccurate: true,
		Supported:       map[Feature]bool{FeatureCanvas: true},
	})
	if got.Risk != RiskHigh || !got.MayVerify || got.NeedsBrowserConfirmation {
		t.Fatalf("unexpected assessment: %#v", got)
	}
}
