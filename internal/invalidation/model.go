package invalidation

// Reason indicates why a validation scope was widened beyond minimal impact.
type Reason string

const (
	ReasonUserForced        Reason = "user_forced_widening"
	ReasonUnknownDependency Reason = "unknown_dynamic_dependency"
	ReasonGlobalToken       Reason = "global_design_token"
	ReasonSharedToken       Reason = "shared_design_token"
	ReasonSCCCycle          Reason = "scc_cycle"
	ReasonCriticalRoute     Reason = "critical_route"
)

// TokenScope classifies the impact radius of a design token.
type TokenScope string

const (
	TokenScopeLocal  TokenScope = "local"
	TokenScopeShared TokenScope = "shared"
	TokenScopeGlobal TokenScope = "global"
)

// ValidationScope is the authoritative execution scope for validation.
// It determines which components, routes, regions, viewports, and themes
// must be rendered and verified.
type ValidationScope struct {
	WholeSite       bool     `json:"whole_site"`
	Components      []string `json:"components,omitempty"`
	Routes          []string `json:"routes,omitempty"`
	Regions         []string `json:"regions,omitempty"`
	Viewports       []string `json:"viewports,omitempty"`
	Themes          []string `json:"themes,omitempty"`
	Widened         bool     `json:"widened"`
	WideningReasons []string `json:"widening_reasons,omitempty"`
}

// Options allows callers to specify runtime overrides or graph topology hints.
type Options struct {
	ForceWholeSite bool
	ForceRoutes    []string
	ForceViewports []string
	ForceThemes    []string
	KnownSCCs      [][]string
}
