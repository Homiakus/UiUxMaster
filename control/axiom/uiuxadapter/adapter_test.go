package uiuxadapter

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
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

func TestQuickStructuralLegacyCollectorIsDiagnosticOnly(t *testing.T) {
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
		t.Fatalf("quick diagnostic requested optional evidence: %v", result.MissingEvidence)
	}
	decision, err := adapter.Decide(context.Background(), change, plan, result)
	if err != nil {
		t.Fatal(err)
	}
	if decision != controlplane.DecisionRecollect {
		t.Fatalf("legacy collector decision = %q, want recollect because no canonical PassAuthority exists", decision)
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
	ctx    fidelity.CalibrationContext
}

func (f *fakePipelineCollector) Collect(_ context.Context, _ engine.ValidationRequest, _ engine.ValidationPlan) (evidence.Packet, error) {
	f.called = true
	return evidence.Packet{
		Renderer: evidence.RendererRef{Tier: "L2", Name: "fake-fastbrowser", Version: "browser-1.0", FidelityID: "fake-l2"},
		Viewport: evidence.Viewport{Width: 1280, Height: 800, DeviceScale: 1},
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

func (f *fakePipelineCollector) CalibrationContext(_ context.Context, _ engine.ValidationRequest, _ engine.ValidationPlan, _ evidence.Packet) (fidelity.CalibrationContext, error) {
	return f.ctx, nil
}

func TestPipelineAdapter_CanonicalExecutionRequiresAndUsesCalibration(t *testing.T) {
	now := time.Date(2026, 9, 5, 6, 0, 0, 0, time.UTC)
	calibrationCtx := fidelity.CalibrationContext{
		Approx: fidelity.CalibrationEnvironment{
			RendererName: "fake-fastbrowser", RendererVersion: "browser-1.0", FidelityID: "fake-l2",
			BrowserFamily: "chromium", BrowserVersion: "browser-1.0", Platform: "test/amd64",
			ViewportWidth: 1280, ViewportHeight: 800, DeviceScale: 1,
		},
		Truth: fidelity.CalibrationEnvironment{
			RendererName: "playwright-chromium", RendererVersion: "worker=1;playwright=1;browser=1",
			BrowserFamily: "chromium", BrowserVersion: "1", WorkerVersion: "1", RuntimeVersion: "1", Platform: "test/amd64",
			ViewportWidth: 1280, ViewportHeight: 800, DeviceScale: 1,
		},
	}
	key, err := calibrationCtx.Key()
	if err != nil {
		t.Fatal(err)
	}
	registry := fidelity.NewCalibrationRegistry()
	if err := registry.Put(fidelity.CalibrationRecord{
		Class: fidelity.EvidenceClassStaticLayout,
		Tier: fidelity.TierL2,
		Context: calibrationCtx,
		EnvironmentKey: key,
		CorpusDigest: "sha256:axiom-canonical-static",
		ArtifactRef: "ci://axiom/canonical-static",
		Samples: 100,
		PassedSamples: 100,
		CreatedAt: now.Add(-time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	authority := fidelity.NewCalibrationAuthority(registry, fidelity.DefaultCalibrationPolicy())
	authority.Now = func() time.Time { return now }

	pipelineCollector := &fakePipelineCollector{ctx: calibrationCtx}
	pipeline := &engine.Pipeline{
		Collector:   pipelineCollector,
		VerPolicy:   verifier.DefaultPolicy(),
		Calibration: authority,
	}

	adapter := NewPipelineAdapter(pipeline)
	change := controlplane.Change{Intent: "quick_structural"}
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
	if result.BlockingFindings > 0 || result.HighFindings > 0 || len(result.MissingEvidence) > 0 {
		t.Fatalf("unexpected findings/missing evidence in calibrated clean run: %+v", result)
	}
	decision, err := adapter.Decide(context.Background(), change, plan, result)
	if err != nil {
		t.Fatalf("Decide failed: %v", err)
	}
	if decision != controlplane.DecisionPass {
		t.Fatalf("decision = %s, want pass with exact valid calibration", decision)
	}
}
