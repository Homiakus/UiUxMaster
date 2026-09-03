package fidelity

import (
	"bytes"
	"sort"
)

// SourceInput contains already-available source bytes. The scanner performs
// cheap feature detection only; it never fetches or renders anything.
type SourceInput struct {
	HTML                        []byte
	CSS                         []byte
	JS                          []byte
	DynamicDependencyUnresolved bool
}

// ScanSourceFeatures returns a deterministic unique feature set used for
// fidelity routing. It intentionally prefers conservative escalation over
// claiming semantic completeness.
func ScanSourceFeatures(in SourceInput) []Feature {
	set := make(map[Feature]struct{})
	add := func(feature Feature) { set[feature] = struct{}{} }

	css := bytes.ToLower(in.CSS)
	js := bytes.ToLower(in.JS)
	html := bytes.ToLower(in.HTML)

	if bytes.Contains(css, []byte("@font-face")) {
		add(FeatureCustomFont)
	}
	if bytes.Contains(css, []byte("transform:")) {
		add(FeatureTransform)
	}
	if bytes.Contains(css, []byte("animation:")) || bytes.Contains(css, []byte("@keyframes")) {
		add(FeatureAnimation)
	}
	if bytes.Contains(css, []byte("filter:")) || bytes.Contains(css, []byte("backdrop-filter:")) {
		add(FeatureCSSFilter)
	}
	if bytes.Contains(css, []byte("mask:")) || bytes.Contains(css, []byte("-webkit-mask:")) {
		add(FeatureCSSMask)
	}

	if bytes.Contains(html, []byte("<canvas")) || bytes.Contains(js, []byte("getcontext('2d'")) || bytes.Contains(js, []byte("getcontext(\"2d\"")) {
		add(FeatureCanvas)
	}
	if bytes.Contains(js, []byte("getcontext('webgl")) || bytes.Contains(js, []byte("getcontext(\"webgl")) {
		add(FeatureWebGL)
	}
	if bytes.Contains(js, []byte("attachshadow(")) {
		add(FeatureShadowDOM)
	}
	if bytes.Contains(js, []byte("customelements.define(")) {
		add(FeatureCustomElement)
	}
	if bytes.Contains(js, []byte("getboundingclientrect(")) ||
		bytes.Contains(js, []byte("resizeobserver")) ||
		bytes.Contains(js, []byte("intersectionobserver")) ||
		bytes.Contains(js, []byte("getcomputedstyle(")) {
		add(FeatureDynamicMeasure)
		add(FeatureBrowserAPI)
	}
	if bytes.Contains(html, []byte("<svg")) && (bytes.Contains(html, []byte("<filter")) || bytes.Contains(html, []byte("<mask"))) {
		add(FeatureComplexSVG)
	}
	if in.DynamicDependencyUnresolved {
		add(FeatureUnresolvedDynamic)
	}

	out := make([]Feature, 0, len(set))
	for feature := range set {
		out = append(out, feature)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
