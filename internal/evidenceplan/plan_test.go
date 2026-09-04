package evidenceplan

import (
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/fidelity"
)

func TestQuickStructuralAvoidsOptionalEvidence(t *testing.T){
	p:=Build(Signals{Intent:IntentQuickStructural,Risk:fidelity.RiskLow})
	if !p.Structural||!p.Diagnostics||p.Accessibility||p.Fonts||p.Pixels{t.Fatalf("plan = %#v",p)}
}
func TestTypographyPromotesAXAndFonts(t *testing.T){
	p:=Build(Signals{Intent:IntentTypography})
	if !p.Structural||!p.Diagnostics||!p.Accessibility||!p.Fonts||p.Pixels{t.Fatalf("plan = %#v",p)}
}
func TestSemanticSignalPromotesAccessibility(t *testing.T){
	p:=Build(Signals{Intent:IntentQuickStructural,SemanticsChanged:true})
	if !p.Accessibility{t.Fatalf("plan = %#v",p)}
}
func TestFinalGateForcesFullNonPixelEvidence(t *testing.T){
	p:=Build(Signals{Intent:IntentQuickStructural,FinalGate:true})
	if !p.Structural||!p.Diagnostics||!p.Accessibility||!p.Fonts||p.Pixels{t.Fatalf("plan = %#v",p)}
}
func TestVisualRegionRequestsPixels(t *testing.T){
	p:=Build(Signals{Intent:IntentVisualRegion,Region:&Region{X:1,Y:2,Width:100,Height:50}})
	if !p.Pixels||!p.Accessibility||!p.Fonts{t.Fatalf("plan = %#v",p)}
}
