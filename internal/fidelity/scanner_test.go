package fidelity

import (
	"reflect"
	"testing"
)

func TestScanSourceFeatures(t *testing.T) {
	got := ScanSourceFeatures(SourceInput{
		HTML: []byte(`<svg><filter id="blur"></filter></svg><canvas id="chart"></canvas>`),
		CSS:  []byte(`@font-face{font-family:X;src:url(x.woff2)} .x{transform:scale(1);filter:blur(2px);animation:pulse 1s}`),
		JS:   []byte(`customElements.define('x-card', XCard); host.attachShadow({mode:'open'}); el.getBoundingClientRect(); gl.getContext('webgl2');`),
		DynamicDependencyUnresolved: true,
	})
	want := []Feature{
		FeatureAnimation,
		FeatureBrowserAPI,
		FeatureCanvas,
		FeatureComplexSVG,
		FeatureCSSFilter,
		FeatureCustomElement,
		FeatureCustomFont,
		FeatureDynamicMeasure,
		FeatureShadowDOM,
		FeatureTransform,
		FeatureUnresolvedDynamic,
		FeatureWebGL,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("features = %#v, want %#v", got, want)
	}
}

func TestScanSourceFeaturesSimpleCSSStaysEmpty(t *testing.T) {
	got := ScanSourceFeatures(SourceInput{CSS: []byte(`.button { display:flex; gap:8px; border-radius:8px }`)})
	if len(got) != 0 {
		t.Fatalf("simple CSS unexpectedly escalated: %#v", got)
	}
}
