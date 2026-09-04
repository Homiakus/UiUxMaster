package evidenceplan

import "github.com/Homiakus/UiUxMaster/internal/fidelity"

type Intent string

const (
	IntentQuickStructural Intent = "quick_structural"
	IntentInteraction     Intent = "interaction"
	IntentTypography      Intent = "typography"
	IntentFullDeterministic Intent = "full_deterministic"
	IntentVisualRegion    Intent = "visual_region"
)

type Region struct { X, Y, Width, Height, Scale float64 }

type Signals struct {
	Intent Intent
	Risk fidelity.RiskLevel
	CustomFontsChanged bool
	SemanticsChanged bool
	InteractionChanged bool
	RuntimeChanged bool
	FinalGate bool
	Region *Region
}

type Plan struct {
	Structural bool
	Diagnostics bool
	Accessibility bool
	Fonts bool
	Pixels bool
	BrowserTruth bool
	Reasons []string
}

// Build returns the cheapest evidence shape that can satisfy the requested
// validation intent. FinalGate always promotes to full non-pixel deterministic
// evidence before a clean result can be considered releasable.
func Build(s Signals) Plan {
	p:=Plan{Structural:true,Diagnostics:true,BrowserTruth:true}
	intent:=s.Intent;if intent==""{intent=IntentQuickStructural}
	if s.FinalGate { intent=IntentFullDeterministic; p.Reasons=append(p.Reasons,"final gate requires complete deterministic browser evidence") }
	switch intent {
	case IntentInteraction:
		p.Accessibility=true;p.Reasons=append(p.Reasons,"interaction intent requires accessibility correlation")
	case IntentTypography:
		p.Accessibility=true;p.Fonts=true;p.Reasons=append(p.Reasons,"typography intent requires accessibility and settled font evidence")
	case IntentFullDeterministic:
		p.Accessibility=true;p.Fonts=true;p.Reasons=append(p.Reasons,"full deterministic intent covers structure, runtime, accessibility and fonts")
	case IntentVisualRegion:
		p.Accessibility=true;p.Fonts=true;p.Pixels=true;p.Reasons=append(p.Reasons,"visual region intent requires localized pixels after deterministic evidence")
	default:
		p.Reasons=append(p.Reasons,"quick structural intent avoids optional pull evidence")
	}
	if s.SemanticsChanged||s.InteractionChanged { p.Accessibility=true;p.Reasons=append(p.Reasons,"semantic or interaction change promotes accessibility evidence") }
	if s.CustomFontsChanged { p.Fonts=true;p.Reasons=append(p.Reasons,"custom font change promotes font-state evidence") }
	if s.RuntimeChanged { p.Diagnostics=true }
	if s.Risk==fidelity.RiskHigh { p.Accessibility=true;p.Reasons=append(p.Reasons,"high fidelity risk promotes browser-semantic evidence") }
	if s.Region!=nil { p.Pixels=true;p.Reasons=append(p.Reasons,"explicit visual region requests localized pixel evidence") }
	return p
}
