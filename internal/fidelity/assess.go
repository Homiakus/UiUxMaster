package fidelity

import "sort"

var featureRisk = map[Feature]RiskLevel{
	FeatureCustomFont:        RiskMedium,
	FeatureTransform:         RiskMedium,
	FeatureAnimation:         RiskMedium,
	FeatureComplexSVG:        RiskMedium,
	FeatureCustomElement:     RiskMedium,
	FeatureShadowDOM:         RiskHigh,
	FeatureCSSFilter:         RiskHigh,
	FeatureCSSMask:           RiskHigh,
	FeatureCanvas:            RiskHigh,
	FeatureWebGL:             RiskHigh,
	FeatureBrowserAPI:        RiskHigh,
	FeatureDynamicMeasure:    RiskHigh,
	FeatureUnresolvedDynamic: RiskHigh,
}

// Assess calculates whether a renderer may prove the requested evidence class
// or should be treated as speculative and escalated to browser confirmation.
func Assess(features []Feature, caps RendererCapabilities) Assessment {
	unique := make(map[Feature]struct{}, len(features))
	unsupportedSet := make(map[Feature]struct{})
	risk := RiskLow

	for _, feature := range features {
		if feature == "" {
			continue
		}
		if _, ok := unique[feature]; ok {
			continue
		}
		unique[feature] = struct{}{}
		if featureRisk[feature] == RiskHigh {
			risk = RiskHigh
		} else if featureRisk[feature] == RiskMedium && risk == RiskLow {
			risk = RiskMedium
		}
		if caps.Supported != nil && !caps.Supported[feature] {
			unsupportedSet[feature] = struct{}{}
		}
	}

	unsupported := make([]Feature, 0, len(unsupportedSet))
	for feature := range unsupportedSet {
		unsupported = append(unsupported, feature)
	}
	sort.Slice(unsupported, func(i, j int) bool { return unsupported[i] < unsupported[j] })

	assessment := Assessment{
		Risk:        risk,
		Unsupported: unsupported,
		MayVerify:   len(unsupported) == 0,
	}

	if len(unsupported) > 0 {
		assessment.Reasons = append(assessment.Reasons, "renderer_missing_required_feature")
	}
	if !caps.BrowserAccurate && risk == RiskHigh {
		assessment.MayVerify = false
		assessment.NeedsBrowserConfirmation = true
		assessment.Reasons = append(assessment.Reasons, "high_fidelity_risk_requires_browser")
	} else if !caps.BrowserAccurate && risk == RiskMedium {
		assessment.NeedsBrowserConfirmation = true
		assessment.Reasons = append(assessment.Reasons, "medium_fidelity_risk_requires_calibration_policy")
	}
	if len(unsupported) > 0 {
		assessment.NeedsBrowserConfirmation = true
	}

	return assessment
}
