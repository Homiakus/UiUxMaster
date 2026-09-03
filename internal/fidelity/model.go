package fidelity

// RiskLevel describes how likely a frontend feature set is to diverge between
// an approximate renderer and browser truth.
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// Feature is a normalized frontend/runtime capability relevant to renderer fidelity.
type Feature string

const (
	FeatureCustomFont        Feature = "custom_font"
	FeatureTransform         Feature = "transform"
	FeatureAnimation         Feature = "animation"
	FeatureComplexSVG        Feature = "complex_svg"
	FeatureCustomElement     Feature = "custom_element"
	FeatureShadowDOM         Feature = "shadow_dom"
	FeatureCSSFilter         Feature = "css_filter"
	FeatureCSSMask           Feature = "css_mask"
	FeatureCanvas            Feature = "canvas"
	FeatureWebGL             Feature = "webgl"
	FeatureBrowserAPI        Feature = "browser_api"
	FeatureDynamicMeasure    Feature = "dynamic_measurement"
	FeatureUnresolvedDynamic Feature = "unresolved_dynamic_dependency"
)

// RendererCapabilities is intentionally vendor-neutral.
type RendererCapabilities struct {
	Name            string
	BrowserAccurate bool
	Supported       map[Feature]bool
}

// Assessment drives the validation router.
type Assessment struct {
	Risk                     RiskLevel
	Unsupported              []Feature
	Reasons                  []string
	MayVerify                bool
	NeedsBrowserConfirmation bool
}
