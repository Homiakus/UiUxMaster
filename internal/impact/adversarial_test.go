package impact_test

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/impact"
)

// 1. Dynamic import:
// Tests that unresolved dynamic dependencies force broad conservative invalidation
// with explicit reasons, and resolved dynamic imports propagate to importing callers.
func TestAdversarial_DynamicImport_FalseNegative(t *testing.T) {
	ctx := context.Background()

	t.Run("unresolved dynamic import prevents false-negative via conservative broadening", func(t *testing.T) {
		builder := impact.NewBuilder()
		if err := builder.SourceBacksModule("src/app.js", "module:app"); err != nil {
			t.Fatal(err)
		}
		if err := builder.ModuleRendersComponent("module:app", "component:app-view"); err != nil {
			t.Fatal(err)
		}
		resolver, err := impact.NewResolver(builder.Graph())
		if err != nil {
			t.Fatal(err)
		}

		// Adversarial input: module has unresolved dynamic imports (e.g. import('./plugins/' + name))
		cs := impact.ChangeSet{
			NodeIDs:   []string{"module:app"},
			Uncertain: true,
			Reasons:   []string{"dynamic_import_expression_unresolved", "eval_import_pattern"},
		}
		res, err := resolver.ApplyChanges(ctx, cs)
		if err != nil {
			t.Fatal(err)
		}

		// Must mark Broad: true to ensure downstream pipeline escalates to broad/browser validation
		if !res.Broad {
			t.Fatalf("expected Broad=true for unresolved dynamic import to prevent false negative")
		}
		if len(res.Reasons) == 0 {
			t.Fatalf("expected non-empty Reasons for uncertain dynamic import")
		}
		// Known component must still be captured
		if len(res.ComponentIDs) == 0 || res.ComponentIDs[0] != "component:app-view" {
			t.Fatalf("expected component:app-view in ComponentIDs, got %#v", res.ComponentIDs)
		}
	})

	t.Run("resolved dynamic import propagates to importer and its placed instances", func(t *testing.T) {
		builder := impact.NewBuilder()
		// module:chart is dynamically imported by module:dashboard
		if err := builder.Import("module:dashboard", "module:chart"); err != nil {
			t.Fatal(err)
		}
		if err := builder.ModuleRendersComponent("module:dashboard", "component:dash-panel"); err != nil {
			t.Fatal(err)
		}
		if err := builder.ComponentInstance("component:dash-panel", "instance:dash-1"); err != nil {
			t.Fatal(err)
		}
		if err := builder.PlaceInstance("instance:dash-1", "page:analytics", "region:0,0,800,600"); err != nil {
			t.Fatal(err)
		}
		resolver, err := impact.NewResolver(builder.Graph())
		if err != nil {
			t.Fatal(err)
		}

		// Change in dynamically imported module:chart must invalidate module:dashboard and its instance
		res, err := resolver.ApplyChanges(ctx, impact.ChangeSet{NodeIDs: []string{"module:chart"}})
		if err != nil {
			t.Fatal(err)
		}
		if !contains(res.ComponentIDs, "component:dash-panel") {
			t.Fatalf("dynamic import target change failed to invalidate importer component: %#v", res.ComponentIDs)
		}
		if !contains(res.RouteIDs, "page:analytics") {
			t.Fatalf("dynamic import target change failed to invalidate target page: %#v", res.RouteIDs)
		}
		if !contains(res.RegionIDs, "region:0,0,800,600") {
			t.Fatalf("dynamic import target change failed to invalidate region: %#v", res.RegionIDs)
		}
	})
}

// 2. Cycles (Strongly Connected Components):
// Frontend code often has circular dependencies (A <-> B or A -> B -> C -> A).
// Verifies no infinite loop and that modifying any node in the cycle invalidates all cycle members and dependents.
func TestAdversarial_DependencyCycles_SCC_Propagation(t *testing.T) {
	ctx := context.Background()

	builder := impact.NewBuilder()
	// Create cycle: module:a -> module:b -> module:c -> module:a
	// Import(importer, dep) creates edge dep -> importer
	// So:
	// a imports b (b -> a)
	// b imports c (c -> b)
	// c imports a (a -> c)
	if err := builder.Import("module:a", "module:b"); err != nil {
		t.Fatal(err)
	}
	if err := builder.Import("module:b", "module:c"); err != nil {
		t.Fatal(err)
	}
	if err := builder.Import("module:c", "module:a"); err != nil {
		t.Fatal(err)
	}

	// Module C renders component:nav which appears on page:root
	if err := builder.ModuleRendersComponent("module:c", "component:nav"); err != nil {
		t.Fatal(err)
	}
	if err := builder.ComponentInstance("component:nav", "instance:nav-header"); err != nil {
		t.Fatal(err)
	}
	if err := builder.PlaceInstance("instance:nav-header", "page:root", "region:0,0,1200,60"); err != nil {
		t.Fatal(err)
	}

	// Module A renders component:sidebar which appears on page:admin
	if err := builder.ModuleRendersComponent("module:a", "component:sidebar"); err != nil {
		t.Fatal(err)
	}
	if err := builder.ComponentInstance("component:sidebar", "instance:side-drawer"); err != nil {
		t.Fatal(err)
	}
	if err := builder.PlaceInstance("instance:side-drawer", "page:admin", "region:0,60,250,900"); err != nil {
		t.Fatal(err)
	}

	resolver, err := impact.NewResolver(builder.Graph())
	if err != nil {
		t.Fatal(err)
	}

	// Mutating module:b MUST reach all members of the cycle and both components/pages
	res, err := resolver.ApplyChanges(ctx, impact.ChangeSet{NodeIDs: []string{"module:b"}})
	if err != nil {
		t.Fatal(err)
	}

	expectedComponents := []string{"component:nav", "component:sidebar", "instance:nav-header", "instance:side-drawer"}
	for _, expected := range expectedComponents {
		if !contains(res.ComponentIDs, expected) {
			t.Fatalf("cycle traversal missed component %q; got: %#v", expected, res.ComponentIDs)
		}
	}

	expectedRoutes := []string{"page:admin", "page:root"}
	for _, expected := range expectedRoutes {
		if !contains(res.RouteIDs, expected) {
			t.Fatalf("cycle traversal missed route %q; got: %#v", expected, res.RouteIDs)
		}
	}

	expectedRegions := []string{"region:0,0,1200,60", "region:0,60,250,900"}
	for _, expected := range expectedRegions {
		if !contains(res.RegionIDs, expected) {
			t.Fatalf("cycle traversal missed region %q; got: %#v", expected, res.RegionIDs)
		}
	}
}

// 3. Shared tokens:
// A single design token (e.g. token:color-accent) is consumed by multiple components.
// Modifying the shared token must invalidate all consumers without dropping any.
// Modifying one component locally must NOT falsely invalidate the other consumers.
func TestAdversarial_SharedTokens_MultiConsumer_NoOmission(t *testing.T) {
	ctx := context.Background()

	builder := impact.NewBuilder()
	// Shared token: color-primary
	if err := builder.TokenAffects("token:color-primary", "component:btn-primary", impact.NodeComponent); err != nil {
		t.Fatal(err)
	}
	if err := builder.TokenAffects("token:color-primary", "component:badge", impact.NodeComponent); err != nil {
		t.Fatal(err)
	}
	if err := builder.TokenAffects("token:color-primary", "component:link", impact.NodeComponent); err != nil {
		t.Fatal(err)
	}

	if err := builder.ComponentInstance("component:btn-primary", "instance:btn-main"); err != nil {
		t.Fatal(err)
	}
	if err := builder.PlaceInstance("instance:btn-main", "page:landing", "region:100,200,120,40"); err != nil {
		t.Fatal(err)
	}

	if err := builder.ComponentInstance("component:badge", "instance:badge-new"); err != nil {
		t.Fatal(err)
	}
	if err := builder.PlaceInstance("instance:badge-new", "page:profile", "region:20,30,50,20"); err != nil {
		t.Fatal(err)
	}

	resolver, err := impact.NewResolver(builder.Graph())
	if err != nil {
		t.Fatal(err)
	}

	// 1. Changing the shared token must hit all 3 components and both routes
	resToken, err := resolver.ResolveToken(ctx, "token:color-primary")
	if err != nil {
		t.Fatal(err)
	}
	for _, expectedComp := range []string{"component:badge", "component:btn-primary", "component:link"} {
		if !contains(resToken.ComponentIDs, expectedComp) {
			t.Fatalf("shared token missed consumer %q; got: %#v", expectedComp, resToken.ComponentIDs)
		}
	}
	if !contains(resToken.RouteIDs, "page:landing") || !contains(resToken.RouteIDs, "page:profile") {
		t.Fatalf("shared token missed routes; got: %#v", resToken.RouteIDs)
	}

	// 2. Changing component:badge alone must ONLY invalidate component:badge and page:profile, NOT btn-primary or landing
	resLocal, err := resolver.ResolveComponent(ctx, "component:badge")
	if err != nil {
		t.Fatal(err)
	}
	if contains(resLocal.ComponentIDs, "component:btn-primary") || contains(resLocal.ComponentIDs, "component:link") {
		t.Fatalf("local component change falsely leaked to other token consumers: %#v", resLocal.ComponentIDs)
	}
	if contains(resLocal.RouteIDs, "page:landing") {
		t.Fatalf("local component change falsely invalidated unrelated route page:landing")
	}
	if !contains(resLocal.RouteIDs, "page:profile") {
		t.Fatalf("local component change failed to invalidate page:profile")
	}
}

// 4. CSS cascade:
// Multi-hop CSS cascading: global base stylesheet -> theme stylesheet -> component style -> component instance -> page/region.
// Changing base stylesheet must propagate through all cascade tiers.
func TestAdversarial_CSSCascade_MultiHop_Propagation(t *testing.T) {
	ctx := context.Background()

	builder := impact.NewBuilder()
	// style:reset -> style:theme -> style:card-styles -> component:card
	if err := builder.StyleImport("style:theme", "style:reset"); err != nil {
		t.Fatal(err)
	}
	if err := builder.StyleImport("style:card-styles", "style:theme"); err != nil {
		t.Fatal(err)
	}
	if err := builder.StyleComponent("style:card-styles", "component:card"); err != nil {
		t.Fatal(err)
	}
	if err := builder.ComponentInstance("component:card", "instance:product-card"); err != nil {
		t.Fatal(err)
	}
	if err := builder.PlaceInstance("instance:product-card", "page:shop", "region:50,100,300,400"); err != nil {
		t.Fatal(err)
	}

	resolver, err := impact.NewResolver(builder.Graph())
	if err != nil {
		t.Fatal(err)
	}

	// Mutating the root reset stylesheet must propagate down the cascade
	res, err := resolver.ApplyChanges(ctx, impact.ChangeSet{NodeIDs: []string{"style:reset"}})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(res.ComponentIDs, "component:card") {
		t.Fatalf("CSS cascade failed to reach component:card: %#v", res.ComponentIDs)
	}
	if !contains(res.ComponentIDs, "instance:product-card") {
		t.Fatalf("CSS cascade failed to reach instance:product-card: %#v", res.ComponentIDs)
	}
	if !contains(res.RouteIDs, "page:shop") {
		t.Fatalf("CSS cascade failed to reach page:shop: %#v", res.RouteIDs)
	}
	if !contains(res.RegionIDs, "region:50,100,300,400") {
		t.Fatalf("CSS cascade failed to reach target render region: %#v", res.RegionIDs)
	}
}

// 5. Route alias:
// A route or page has an alias (e.g. / and /home, or dashboard aliases).
// Changing a component placed on the target route must invalidate both canonical and alias routes.
func TestAdversarial_RouteAlias_MultiRoute_Coverage(t *testing.T) {
	ctx := context.Background()

	builder := impact.NewBuilder()
	if err := builder.Component("component:hero"); err != nil {
		t.Fatal(err)
	}
	if err := builder.ComponentInstance("component:hero", "instance:hero-inst"); err != nil {
		t.Fatal(err)
	}
	if err := builder.PlaceInstance("instance:hero-inst", "route:/home", "region:0,0,1000,400"); err != nil {
		t.Fatal(err)
	}
	// route:/ aliases route:/home
	if err := builder.RouteAlias("route:/", "route:/home"); err != nil {
		t.Fatal(err)
	}

	resolver, err := impact.NewResolver(builder.Graph())
	if err != nil {
		t.Fatal(err)
	}

	res, err := resolver.ResolveComponent(ctx, "component:hero")
	if err != nil {
		t.Fatal(err)
	}

	if !contains(res.RouteIDs, "route:/home") {
		t.Fatalf("failed to invalidate canonical route: %#v", res.RouteIDs)
	}
	if !contains(res.RouteIDs, "route:/") {
		t.Fatalf("failed to invalidate route alias: %#v", res.RouteIDs)
	}
}

// 6. Stale runtime refs:
// When changes involve unknown, deleted or stale runtime references,
// the impact engine must not silently ignore them (which would cause false-negative PASS);
// it must mark Broad: true with UnknownIDs.
func TestAdversarial_StaleRuntimeRefs_ConservativeFallback(t *testing.T) {
	ctx := context.Background()

	builder := impact.NewBuilder()
	if err := builder.BindInstanceRuntime("instance:valid", "dom:btn-1", "region:0,0,100,40"); err != nil {
		t.Fatal(err)
	}
	resolver, err := impact.NewResolver(builder.Graph())
	if err != nil {
		t.Fatal(err)
	}

	// 1. Change with unknown/stale runtime ref ID
	res, err := resolver.ApplyChanges(ctx, impact.ChangeSet{
		NodeIDs: []string{"dom:stale-ref-deleted-element"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !res.Broad {
		t.Fatalf("expected Broad=true when stale runtime ref is not found in graph")
	}
	if !contains(res.UnknownIDs, "dom:stale-ref-deleted-element") {
		t.Fatalf("expected UnknownIDs to contain stale ref: %#v", res.UnknownIDs)
	}

	// 2. Valid runtime ref correctly resolves to region
	resValid, err := resolver.ApplyChanges(ctx, impact.ChangeSet{
		NodeIDs: []string{"dom:btn-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resValid.Broad {
		t.Fatalf("known runtime ref should not trigger broad fallback")
	}
	if !contains(resValid.RegionIDs, "region:0,0,100,40") {
		t.Fatalf("valid runtime ref failed to resolve to region: %#v", resValid.RegionIDs)
	}
}

// 7. Runtime-only component instances:
// Instances created dynamically at runtime (modals, overlays, toasts, portals)
// bound via BindInstanceRuntime without static file placement.
// Changing the component must invalidate all dynamic runtime instances and their observed regions.
func TestAdversarial_RuntimeOnly_DynamicInstances(t *testing.T) {
	ctx := context.Background()

	builder := impact.NewBuilder()
	if err := builder.Component("component:toast"); err != nil {
		t.Fatal(err)
	}
	// Runtime-only instances bound during browser session
	if err := builder.ComponentInstance("component:toast", "instance:toast-alert-1"); err != nil {
		t.Fatal(err)
	}
	if err := builder.BindInstanceRuntime("instance:toast-alert-1", "dom:toast-container-entry", "region:top-right:300,80"); err != nil {
		t.Fatal(err)
	}

	if err := builder.ComponentInstance("component:toast", "instance:toast-alert-2"); err != nil {
		t.Fatal(err)
	}
	if err := builder.BindInstanceRuntime("instance:toast-alert-2", "dom:toast-container-entry-2", "region:top-right:300,170"); err != nil {
		t.Fatal(err)
	}

	resolver, err := impact.NewResolver(builder.Graph())
	if err != nil {
		t.Fatal(err)
	}

	res, err := resolver.ResolveComponent(ctx, "component:toast")
	if err != nil {
		t.Fatal(err)
	}

	if !contains(res.ComponentIDs, "instance:toast-alert-1") || !contains(res.ComponentIDs, "instance:toast-alert-2") {
		t.Fatalf("runtime-only instances not invalidated: %#v", res.ComponentIDs)
	}
	if !contains(res.RegionIDs, "region:top-right:300,80") || !contains(res.RegionIDs, "region:top-right:300,170") {
		t.Fatalf("runtime-only instance regions not invalidated: %#v", res.RegionIDs)
	}
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
