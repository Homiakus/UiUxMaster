package uiuxadapter

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
)

type fakeCollector struct {
	packet evidence.Packet
	plan   evidenceplan.Plan
}

func (f *fakeCollector) Collect(_ context.Context, _ controlplane.Change, plan evidenceplan.Plan) (evidence.Packet, error) {
	f.plan = plan
	return f.packet, nil
}

func TestQuickStructuralPlanStaysNonAXAndCanPass(t *testing.T) {
	collector := &fakeCollector{packet: evidence.Packet{
		Renderer:    evidence.RendererRef{Tier: "L2"},
		Elements:    []evidence.ElementRef{{ID: "main", Tag: "main", Visible: true}},
		Diagnostics: &evidence.DiagnosticsEvidence{Complete: true},
	}}
	adapter := New(collector)
	change := controlplane.Change{Intent: "quick_structural"}
	plan, err := adapter.PlanEvidence(context.Background(), change)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Accessibility || plan.Fonts || plan.Pixels || !plan.Structural || !plan.Diagnostics {
		t.Fatalf("unexpected quick plan: %#v", plan)
	}
	result, err := adapter.CollectVerify(context.Background(), change, plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MissingEvidence) != 0 {
		t.Fatalf("quick pass requested optional evidence: %v", result.MissingEvidence)
	}
	decision, err := adapter.Decide(context.Background(), change, plan, result)
	if err != nil {
		t.Fatal(err)
	}
	if decision != controlplane.DecisionPass {
		t.Fatalf("decision = %q, want pass", decision)
	}
}

func TestTypographyPlanFailsConservativelyWhenFontsMissing(t *testing.T) {
	collector := &fakeCollector{packet: evidence.Packet{
		Renderer:     evidence.RendererRef{Tier: "L2"},
		Elements:     []evidence.ElementRef{{ID: "heading", Tag: "h1", Visible: true}},
		AriaSnapshot: "- heading: Example",
		Diagnostics:  &evidence.DiagnosticsEvidence{Complete: true},
	}}
	adapter := New(collector)
	change := controlplane.Change{Intent: "typography"}
	plan, _ := adapter.PlanEvidence(context.Background(), change)
	if !plan.Accessibility || !plan.Fonts {
		t.Fatalf("typography plan = %#v", plan)
	}
	result, err := adapter.CollectVerify(context.Background(), change, plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MissingEvidence) != 1 || result.MissingEvidence[0] != "font state" {
		t.Fatalf("missing = %v", result.MissingEvidence)
	}
	decision, _ := adapter.Decide(context.Background(), change, plan, result)
	if decision != controlplane.DecisionRecollect {
		t.Fatalf("decision = %q, want recollect", decision)
	}
}

func TestVisualRegionEscalatesPixelsThenSemantic(t *testing.T) {
	adapter := New(&fakeCollector{})
	missingPixels := controlplane.ValidationResult{MissingEvidence: []string{"rendered region pixels"}, VisualRegions: 1}
	decision, _ := adapter.Decide(context.Background(), controlplane.Change{}, controlplane.EvidencePlan{}, missingPixels)
	if decision != controlplane.DecisionPixels {
		t.Fatalf("decision = %q, want pixels", decision)
	}
	withPixels := controlplane.ValidationResult{VisualRegions: 1, PixelEvidence: true}
	decision, _ = adapter.Decide(context.Background(), controlplane.Change{}, controlplane.EvidencePlan{}, withPixels)
	if decision != controlplane.DecisionSemantic {
		t.Fatalf("decision = %q, want semantic", decision)
	}
}

func TestGroundedHighSeverityFindingRoutesRepair(t *testing.T) {
	adapter := New(&fakeCollector{})
	decision, _ := adapter.Decide(context.Background(), controlplane.Change{}, controlplane.EvidencePlan{}, controlplane.ValidationResult{HighFindings: 1})
	if decision != controlplane.DecisionRepair {
		t.Fatalf("decision = %q, want repair", decision)
	}
}
