package impact

import (
	"context"
	"reflect"
	"testing"
)

func TestScanCSSCustomProperties(t *testing.T) {
	css := []byte(`
:root {
  --space-4: 16px;
  --radius-control: 12px;
}
.button {
  padding: var(--space-4);
  border-radius: var( --radius-control );
  gap: var(--space-4);
}
/* --ignored: 1px; color: var(--ignored); */
`)
	got := ScanCSSCustomProperties(css)
	want := CSSCustomProperties{
		Definitions: []string{"--radius-control", "--space-4"},
		References:  []string{"--radius-control", "--space-4"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("facts = %#v, want %#v", got, want)
	}
}

func TestCSSReferencePropagatesTokenToComponent(t *testing.T) {
	b := NewBuilder()
	if _, err := b.IndexStyleSheetCSS("style:button", []byte(`.button { border-radius: var(--radius-control); }`)); err != nil {
		t.Fatal(err)
	}
	if err := b.StyleComponent("style:button", "component:button"); err != nil {
		t.Fatal(err)
	}
	if err := b.ComponentInstance("component:button", "instance:hero-cta"); err != nil {
		t.Fatal(err)
	}
	if err := b.PlaceInstance("instance:hero-cta", "page:home", "region:hero-actions"); err != nil {
		t.Fatal(err)
	}

	resolver, err := NewResolver(b.Graph())
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolver.ResolveToken(context.Background(), CSSCustomPropertyNodeID("--radius-control"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.RegionIDs, []string{"region:hero-actions"}) {
		t.Fatalf("regions = %#v", got.RegionIDs)
	}
}
