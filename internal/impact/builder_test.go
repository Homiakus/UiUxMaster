package impact

import (
	"context"
	"reflect"
	"testing"
)

func TestBuilderImportPropagatesDependencyToImporter(t *testing.T) {
	b := NewBuilder()
	if err := b.Import("module:app", "module:button"); err != nil {
		t.Fatal(err)
	}
	if err := b.ModuleRendersComponent("module:app", "component:app"); err != nil {
		t.Fatal(err)
	}

	resolver, err := NewResolver(b.Graph())
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolver.ApplyChanges(context.Background(), ChangeSet{NodeIDs: []string{"module:button"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"component:app"}
	if !reflect.DeepEqual(got.ComponentIDs, want) {
		t.Fatalf("components = %#v, want %#v", got.ComponentIDs, want)
	}
}

func TestBuilderTokenAffectsRenderedRegion(t *testing.T) {
	b := NewBuilder()
	if err := b.TokenAffects("token:radius-control", "component:button", NodeComponent); err != nil {
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
	got, err := resolver.ResolveToken(context.Background(), "token:radius-control")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.RegionIDs, []string{"region:hero-actions"}) {
		t.Fatalf("regions = %#v", got.RegionIDs)
	}
	if !reflect.DeepEqual(got.RouteIDs, []string{"page:home"}) {
		t.Fatalf("routes = %#v", got.RouteIDs)
	}
}

func TestBuilderRuntimeBindingPropagatesToObservedRegion(t *testing.T) {
	b := NewBuilder()
	if err := b.ComponentInstance("component:button", "instance:hero-cta"); err != nil {
		t.Fatal(err)
	}
	if err := b.BindInstanceRuntime("instance:hero-cta", "dom:button-publish", "region:42,80,160,44"); err != nil {
		t.Fatal(err)
	}

	resolver, err := NewResolver(b.Graph())
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolver.ResolveComponent(context.Background(), "component:button")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.RegionIDs, []string{"region:42,80,160,44"}) {
		t.Fatalf("regions = %#v", got.RegionIDs)
	}
}
