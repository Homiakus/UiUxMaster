package invalidation

import (
	"reflect"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/impact"
)

func TestPolicy_MinimalImpactNoWidening(t *testing.T) {
	policy := DefaultPolicy()

	set := impact.ImpactSet{
		NodeIDs:      []string{"file:button.tsx", "component:button"},
		ComponentIDs: []string{"component:button"},
		RouteIDs:     []string{"page:settings"},
		RegionIDs:    []string{"region:button-area"},
		Broad:        false,
	}

	scope := policy.Invalidate(set, Options{})

	if scope.WholeSite {
		t.Errorf("expected WholeSite=false, got true")
	}
	if scope.Widened {
		t.Errorf("expected Widened=false, got true")
	}
	if len(scope.WideningReasons) != 0 {
		t.Errorf("expected no widening reasons, got %v", scope.WideningReasons)
	}

	wantComponents := []string{"component:button"}
	if !reflect.DeepEqual(scope.Components, wantComponents) {
		t.Errorf("Components = %v, want %v", scope.Components, wantComponents)
	}

	wantRoutes := []string{"page:settings"}
	if !reflect.DeepEqual(scope.Routes, wantRoutes) {
		t.Errorf("Routes = %v, want %v", scope.Routes, wantRoutes)
	}

	wantRegions := []string{"region:button-area"}
	if !reflect.DeepEqual(scope.Regions, wantRegions) {
		t.Errorf("Regions = %v, want %v", scope.Regions, wantRegions)
	}
}

func TestPolicy_GlobalTokenWidensToWholeSite(t *testing.T) {
	policy := DefaultPolicy()
	policy.AllRoutes = []string{"page:about", "page:home", "page:pricing"}

	set := impact.ImpactSet{
		NodeIDs:      []string{"token:global:colors-primary"},
		ComponentIDs: []string{"component:header"},
		RouteIDs:     []string{"page:home"},
	}

	scope := policy.Invalidate(set, Options{})

	if !scope.WholeSite {
		t.Errorf("expected WholeSite=true")
	}
	if !scope.Widened {
		t.Errorf("expected Widened=true")
	}

	hasReason := false
	for _, r := range scope.WideningReasons {
		if r == string(ReasonGlobalToken) {
			hasReason = true
			break
		}
	}
	if !hasReason {
		t.Errorf("expected ReasonGlobalToken in %v", scope.WideningReasons)
	}

	// Should include all registered routes
	for _, r := range policy.AllRoutes {
		found := false
		for _, sr := range scope.Routes {
			if sr == r {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected route %s in scope %v", r, scope.Routes)
		}
	}
}

func TestPolicy_SharedTokenWidensToSharedRoutes(t *testing.T) {
	policy := DefaultPolicy()
	policy.SharedTokens["token:card-spacing"] = []string{"page:dashboard", "page:explore"}

	set := impact.ImpactSet{
		NodeIDs:      []string{"token:card-spacing"},
		ComponentIDs: []string{"component:card"},
		RouteIDs:     []string{"page:home"},
	}

	scope := policy.Invalidate(set, Options{})

	if scope.WholeSite {
		t.Errorf("shared token should not widen to whole site")
	}
	if !scope.Widened {
		t.Errorf("expected Widened=true")
	}

	wantRoutes := []string{"page:dashboard", "page:explore", "page:home"}
	if !reflect.DeepEqual(scope.Routes, wantRoutes) {
		t.Errorf("Routes = %v, want %v", scope.Routes, wantRoutes)
	}
}

func TestPolicy_UnknownDependencyWidensConservatively(t *testing.T) {
	policy := DefaultPolicy()

	set := impact.ImpactSet{
		NodeIDs:    []string{"file:dynamic-loader.ts"},
		UnknownIDs: []string{"unknown:dynamic-import"},
		Broad:      true,
	}

	scope := policy.Invalidate(set, Options{})

	if !scope.WholeSite {
		t.Errorf("expected WholeSite=true on dynamic/unknown dependency")
	}
	if !scope.Widened {
		t.Errorf("expected Widened=true")
	}

	hasReason := false
	for _, r := range scope.WideningReasons {
		if r == string(ReasonUnknownDependency) {
			hasReason = true
			break
		}
	}
	if !hasReason {
		t.Errorf("expected ReasonUnknownDependency in %v", scope.WideningReasons)
	}
}

func TestPolicy_SCCCycleWidensToAllMutualDependents(t *testing.T) {
	policy := DefaultPolicy()

	// SCC containing recursive/cyclic components
	scc := []string{"component:tree-node", "component:tree-item", "component:tree-view"}

	set := impact.ImpactSet{
		NodeIDs:      []string{"component:tree-node"},
		ComponentIDs: []string{"component:tree-node"},
		RouteIDs:     []string{"page:files"},
	}

	scope := policy.Invalidate(set, Options{
		KnownSCCs: [][]string{scc},
	})

	if !scope.Widened {
		t.Errorf("expected Widened=true for SCC cycle")
	}

	wantComponents := []string{"component:tree-item", "component:tree-node", "component:tree-view"}
	if !reflect.DeepEqual(scope.Components, wantComponents) {
		t.Errorf("Components = %v, want %v", scope.Components, wantComponents)
	}

	hasReason := false
	for _, r := range scope.WideningReasons {
		if r == string(ReasonSCCCycle) {
			hasReason = true
			break
		}
	}
	if !hasReason {
		t.Errorf("expected ReasonSCCCycle in %v", scope.WideningReasons)
	}
}

func TestPolicy_CriticalRouteProtectedWhenLayoutModified(t *testing.T) {
	policy := DefaultPolicy()

	// component:navbar is a registered layout component
	set := impact.ImpactSet{
		NodeIDs:      []string{"component:navbar"},
		ComponentIDs: []string{"component:navbar"},
		RouteIDs:     []string{"page:dashboard"},
	}

	scope := policy.Invalidate(set, Options{})

	if !scope.Widened {
		t.Errorf("expected Widened=true when layout component modified")
	}

	// Critical routes (page:home, route:home, route:root, page:index) must be added
	for cr := range policy.CriticalRoutes {
		found := false
		for _, r := range scope.Routes {
			if r == cr {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected critical route %s in scope %v", cr, scope.Routes)
		}
	}

	hasReason := false
	for _, r := range scope.WideningReasons {
		if r == string(ReasonCriticalRoute) {
			hasReason = true
			break
		}
	}
	if !hasReason {
		t.Errorf("expected ReasonCriticalRoute in %v", scope.WideningReasons)
	}
}

func TestPolicy_UserForcedWidening(t *testing.T) {
	policy := DefaultPolicy()

	set := impact.ImpactSet{
		NodeIDs:      []string{"component:button"},
		ComponentIDs: []string{"component:button"},
	}

	opts := Options{
		ForceWholeSite: true,
		ForceRoutes:    []string{"page:checkout", "page:promo"},
		ForceViewports: []string{"tablet", "ultrawide"},
		ForceThemes:    []string{"dark"},
	}

	scope := policy.Invalidate(set, opts)

	if !scope.WholeSite {
		t.Errorf("expected WholeSite=true when user forced")
	}
	if !scope.Widened {
		t.Errorf("expected Widened=true when user forced")
	}

	wantViewports := []string{"tablet", "ultrawide"}
	if !reflect.DeepEqual(scope.Viewports, wantViewports) {
		t.Errorf("Viewports = %v, want %v", scope.Viewports, wantViewports)
	}

	wantThemes := []string{"dark"}
	if !reflect.DeepEqual(scope.Themes, wantThemes) {
		t.Errorf("Themes = %v, want %v", scope.Themes, wantThemes)
	}

	hasForcedRoute := false
	for _, r := range scope.Routes {
		if r == "page:checkout" {
			hasForcedRoute = true
			break
		}
	}
	if !hasForcedRoute {
		t.Errorf("expected forced route page:checkout in %v", scope.Routes)
	}
}
