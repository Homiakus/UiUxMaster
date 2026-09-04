package invalidation

import (
	"sort"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/impact"
)

// Policy governs how an ImpactSet is expanded into a deterministic ValidationScope.
type Policy struct {
	GlobalTokens     map[string]bool
	SharedTokens     map[string][]string
	CriticalRoutes   map[string]bool
	LayoutComponents map[string]bool
	AllRoutes        []string
	DefaultViewports []string
	DefaultThemes    []string
}

// NewPolicy returns a clean Policy instance.
func NewPolicy() *Policy {
	return &Policy{
		GlobalTokens:     make(map[string]bool),
		SharedTokens:     make(map[string][]string),
		CriticalRoutes:   make(map[string]bool),
		LayoutComponents: make(map[string]bool),
		DefaultViewports: []string{"desktop", "mobile"},
		DefaultThemes:    []string{"light"},
	}
}

// DefaultPolicy returns a Policy pre-configured with standard design system heuristics.
func DefaultPolicy() *Policy {
	p := NewPolicy()
	// Standard global token prefixes and identifiers
	p.GlobalTokens["token:global"] = true
	p.GlobalTokens["token:theme:root"] = true
	p.GlobalTokens["token:typography:base"] = true
	p.GlobalTokens["token:colors:primary"] = true
	p.GlobalTokens["token:spacing:base"] = true

	// Standard critical routes
	p.CriticalRoutes["page:home"] = true
	p.CriticalRoutes["route:home"] = true
	p.CriticalRoutes["route:root"] = true
	p.CriticalRoutes["page:index"] = true

	// Standard layout components
	p.LayoutComponents["component:header"] = true
	p.LayoutComponents["component:navbar"] = true
	p.LayoutComponents["component:footer"] = true
	p.LayoutComponents["component:shell"] = true
	p.LayoutComponents["component:layout"] = true
	return p
}

// ClassifyToken determines whether a token is local, shared, or global.
func (p *Policy) ClassifyToken(tokenID string) TokenScope {
	if p.GlobalTokens[tokenID] || strings.Contains(tokenID, "global") || strings.Contains(tokenID, "root") {
		return TokenScopeGlobal
	}
	if routes, ok := p.SharedTokens[tokenID]; ok && len(routes) > 0 {
		return TokenScopeShared
	}
	if strings.Contains(tokenID, "shared") || strings.Contains(tokenID, "layout") {
		return TokenScopeShared
	}
	return TokenScopeLocal
}

// Invalidate converts an ImpactSet into a fully determined ValidationScope according
// to the invalidation policy rules.
func (p *Policy) Invalidate(set impact.ImpactSet, opts Options) ValidationScope {
	scope := ValidationScope{
		Components: make([]string, 0, len(set.ComponentIDs)),
		Routes:     make([]string, 0, len(set.RouteIDs)),
		Regions:    make([]string, 0, len(set.RegionIDs)),
		Viewports:  make([]string, 0),
		Themes:     make([]string, 0),
	}

	reasonsMap := make(map[string]struct{})
	addReason := func(r Reason) {
		reasonsMap[string(r)] = struct{}{}
		scope.Widened = true
	}

	// 1. Initial assignment from ImpactSet
	componentsMap := make(map[string]struct{}, len(set.ComponentIDs))
	for _, c := range set.ComponentIDs {
		if c != "" {
			componentsMap[c] = struct{}{}
		}
	}

	routesMap := make(map[string]struct{}, len(set.RouteIDs))
	for _, r := range set.RouteIDs {
		if r != "" {
			routesMap[r] = struct{}{}
		}
	}

	regionsMap := make(map[string]struct{}, len(set.RegionIDs))
	for _, reg := range set.RegionIDs {
		if reg != "" {
			regionsMap[reg] = struct{}{}
		}
	}

	// 2. User-forced widening
	if opts.ForceWholeSite {
		scope.WholeSite = true
		addReason(ReasonUserForced)
	}
	for _, fr := range opts.ForceRoutes {
		if fr != "" {
			routesMap[fr] = struct{}{}
			addReason(ReasonUserForced)
		}
	}

	// 3. Unknown / Dynamic dependency widening
	if set.Broad || len(set.UnknownIDs) > 0 {
		scope.WholeSite = true
		addReason(ReasonUnknownDependency)
	}

	// 4. Token classification & widening (check all node IDs in impact set)
	hasGlobalToken := false
	for _, nodeID := range set.NodeIDs {
		if strings.HasPrefix(nodeID, "token:") || strings.Contains(nodeID, "token") {
			switch p.ClassifyToken(nodeID) {
			case TokenScopeGlobal:
				hasGlobalToken = true
			case TokenScopeShared:
				addReason(ReasonSharedToken)
				if sharedRoutes, ok := p.SharedTokens[nodeID]; ok {
					for _, sr := range sharedRoutes {
						routesMap[sr] = struct{}{}
					}
				}
			}
		}
	}
	if hasGlobalToken {
		scope.WholeSite = true
		addReason(ReasonGlobalToken)
	}

	// 5. SCC (strongly connected component / cycle) widening
	for _, scc := range opts.KnownSCCs {
		if len(scc) <= 1 {
			continue
		}
		// Check if any component in this SCC is impacted
		sccHit := false
		for _, member := range scc {
			if _, ok := componentsMap[member]; ok {
				sccHit = true
				break
			}
		}
		if sccHit {
			for _, member := range scc {
				if _, ok := componentsMap[member]; !ok {
					componentsMap[member] = struct{}{}
					addReason(ReasonSCCCycle)
				}
			}
		}
	}

	// 6. Critical-route widening: if a layout component is affected, protect critical routes
	affectsLayout := false
	for c := range componentsMap {
		if p.LayoutComponents[c] || strings.Contains(c, "layout") || strings.Contains(c, "header") || strings.Contains(c, "navbar") {
			affectsLayout = true
			break
		}
	}
	if affectsLayout {
		for cr := range p.CriticalRoutes {
			if _, ok := routesMap[cr]; !ok {
				routesMap[cr] = struct{}{}
				addReason(ReasonCriticalRoute)
			}
		}
	}

	// 7. If WholeSite is true, ensure all registered routes are included
	if scope.WholeSite {
		for _, ar := range p.AllRoutes {
			routesMap[ar] = struct{}{}
		}
		// Also ensure critical routes are included
		for cr := range p.CriticalRoutes {
			routesMap[cr] = struct{}{}
		}
	}

	// 8. Viewports & Themes
	if len(opts.ForceViewports) > 0 {
		scope.Viewports = append(scope.Viewports, opts.ForceViewports...)
	} else if len(p.DefaultViewports) > 0 {
		scope.Viewports = append(scope.Viewports, p.DefaultViewports...)
	}

	if len(opts.ForceThemes) > 0 {
		scope.Themes = append(scope.Themes, opts.ForceThemes...)
	} else if len(p.DefaultThemes) > 0 {
		scope.Themes = append(scope.Themes, p.DefaultThemes...)
	}

	// 9. Deterministic sorting and serialization
	for c := range componentsMap {
		scope.Components = append(scope.Components, c)
	}
	for r := range routesMap {
		scope.Routes = append(scope.Routes, r)
	}
	for reg := range regionsMap {
		scope.Regions = append(scope.Regions, reg)
	}

	sort.Strings(scope.Components)
	sort.Strings(scope.Routes)
	sort.Strings(scope.Regions)
	sort.Strings(scope.Viewports)
	sort.Strings(scope.Themes)

	scope.WideningReasons = make([]string, 0, len(reasonsMap))
	for r := range reasonsMap {
		scope.WideningReasons = append(scope.WideningReasons, r)
	}
	sort.Strings(scope.WideningReasons)

	return scope
}
