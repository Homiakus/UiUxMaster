package uiuxadapter

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
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

type fakePipelineCollector struct {
	called bool
}

func (f *fakePipelineCollector) Collect(_ context.Context, _ engine.ValidationRequest, _ engine.ValidationPlan) (evidence.Packet, error) {
	f.called = true
	return evidence.Packet{
		Renderer: evidence.RendererRef{Tier: "L2"},
		Elements: []evidence.ElementRef{
			{
				ID:        "btn-submit",
				Tag:       "button",
				Role:      "button",
				Name:      "Save",
				Visible:   true,
				Clickable: true,
				Bounds:    evidence.Rect{X: 10, Y: 10, Width: 100, Height: 44},
			},
		},
		Diagnostics: &evidence.DiagnosticsEvidence{Complete: true},
	}, nil
}

func TestPipelineAdapter_CanonicalExecution(t *testing.T) {
	pipelineCollector := &fakePipelineCollector{}
	pipeline := &engine.Pipeline{
		Collector: pipelineCollector,
		VerPolicy: verifier.DefaultPolicy(),
	}

	adapter := NewPipelineAdapter(pipeline)
	change := controlplane.Change{
		Intent: "quick_structural",
	}

	plan, err := adapter.PlanEvidence(context.Background(), change)
	if err != nil {
		t.Fatalf("PlanEvidence failed: %v", err)
	}

	result, err := adapter.CollectVerify(context.Background(), change, plan)
	if err != nil {
		t.Fatalf("CollectVerify failed: %v", err)
	}

	if !pipelineCollector.called {
		t.Fatalf("expected canonical pipeline collector to be executed")
	}
	if result.BlockingFindings > 0 || result.HighFindings > 0 {
		t.Fatalf("unexpected findings in clean run: %+v", result)
	}

	decision, err := adapter.Decide(context.Background(), change, plan, result)
	if err != nil {
		t.Fatalf("Decide failed: %v", err)
	}
	if decision != controlplane.DecisionPass {
		t.Fatalf("decision = %s, want pass", decision)
	}
}

