package engine_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
)

const fmea005InjectedDelay = 35 * time.Millisecond

type fmea005Collector struct{}

func (fmea005Collector) Collect(_ context.Context, req engine.ValidationRequest, plan engine.ValidationPlan) (evidence.Packet, error) {
	return evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{Tier: string(plan.Route.Tier), Name: "fmea005-stage-test"},
		Viewport: evidence.Viewport{Width: 1280, Height: 800, DeviceScale: 1},
		Documents: []evidence.DocumentMetrics{{ContentWidth: 1000, ContentHeight: 700}},
		Elements: []evidence.ElementRef{{ID: "main", Tag: "main", Role: "main", Visible: true, Bounds: evidence.Rect{Width: 1000, Height: 700}}},
		Accessibility: []evidence.AccessibilityNode{{ID: "main", Role: "main", Name: "Main"}},
		AriaSnapshot: "- main: Main",
		Fonts: &evidence.FontEvidence{Status: "loaded"},
		Diagnostics: &evidence.DiagnosticsEvidence{Complete: true},
	}, nil
}

func TestFMEA005ImpactDelayAffectsOnlyImpactTelemetry(t *testing.T) {
	pipeline := &engine.Pipeline{
		Collector: fmea005Collector{},
		ImpactStage: func(ctx context.Context, req engine.ValidationRequest) (engine.ValidationRequest, impact.ImpactSet, error) {
			time.Sleep(fmea005InjectedDelay)
			return engine.ResolveImpact(ctx, req, nil)
		},
		InvalidationStage: func(ctx context.Context, req engine.ValidationRequest, set impact.ImpactSet) (engine.ValidationRequest, invalidation.ValidationScope, error) {
			return engine.InvalidateImpact(ctx, req, set, invalidation.DefaultPolicy())
		},
	}

	res, err := pipeline.Execute(context.Background(), engine.ValidationRequest{
		RunID: "fmea005-impact-delay", ChangedNodes: []string{"component:changed"},
	})
	if err != nil { t.Fatal(err) }
	if res.Telemetry.ImpactMS < 30 {
		t.Fatalf("ImpactMS=%fms, expected injected impact delay", res.Telemetry.ImpactMS)
	}
	if res.Telemetry.ImpactMS-res.Telemetry.InvalidationMS < 20 {
		t.Fatalf("impact delay leaked into invalidation: impact=%f invalidation=%f", res.Telemetry.ImpactMS, res.Telemetry.InvalidationMS)
	}
	if res.Packet.Latency.ImpactMS != res.Telemetry.ImpactMS || res.Packet.Latency.InvalidationMS != res.Telemetry.InvalidationMS {
		t.Fatalf("packet/telemetry stage mismatch: packet=%#v telemetry=%#v", res.Packet.Latency, res.Telemetry)
	}
}

func TestFMEA005InvalidationDelayAffectsOnlyInvalidationTelemetry(t *testing.T) {
	pipeline := &engine.Pipeline{
		Collector: fmea005Collector{},
		ImpactStage: func(ctx context.Context, req engine.ValidationRequest) (engine.ValidationRequest, impact.ImpactSet, error) {
			return engine.ResolveImpact(ctx, req, nil)
		},
		InvalidationStage: func(ctx context.Context, req engine.ValidationRequest, set impact.ImpactSet) (engine.ValidationRequest, invalidation.ValidationScope, error) {
			time.Sleep(fmea005InjectedDelay)
			return engine.InvalidateImpact(ctx, req, set, invalidation.DefaultPolicy())
		},
	}

	res, err := pipeline.Execute(context.Background(), engine.ValidationRequest{
		RunID: "fmea005-invalidation-delay", ChangedNodes: []string{"component:changed"},
	})
	if err != nil { t.Fatal(err) }
	if res.Telemetry.InvalidationMS < 30 {
		t.Fatalf("InvalidationMS=%fms, expected injected invalidation delay", res.Telemetry.InvalidationMS)
	}
	if res.Telemetry.InvalidationMS-res.Telemetry.ImpactMS < 20 {
		t.Fatalf("invalidation delay leaked into impact: impact=%f invalidation=%f", res.Telemetry.ImpactMS, res.Telemetry.InvalidationMS)
	}
}

func TestFMEA005StageCountersAndAccountingAreIndependent(t *testing.T) {
	pipeline := &engine.Pipeline{
		Collector: fmea005Collector{},
		ImpactStage: func(ctx context.Context, req engine.ValidationRequest) (engine.ValidationRequest, impact.ImpactSet, error) {
			req.Normalize()
			req.Need = req.DeriveNeed()
			return req, impact.ImpactSet{
				NodeIDs: []string{"component:a", "route:a", "region:a"},
				UnknownIDs: []string{"component:unknown"},
				ComponentIDs: []string{"component:a"},
				RouteIDs: []string{"route:a"},
				RegionIDs: []string{"region:a"},
			}, nil
		},
		InvalidationStage: func(_ context.Context, req engine.ValidationRequest, _ impact.ImpactSet) (engine.ValidationRequest, invalidation.ValidationScope, error) {
			scope := invalidation.ValidationScope{
				Components: []string{"component:a", "component:widened"},
				Routes: []string{"route:a", "route:critical"},
				Regions: []string{"region:a"},
				Viewports: []string{"desktop", "mobile"},
				Themes: []string{"light"},
				Widened: true,
			}
			req.Scope = scope
			return req, scope, nil
		},
	}

	res, err := pipeline.Execute(context.Background(), engine.ValidationRequest{RunID: "fmea005-accounting"})
	if err != nil { t.Fatal(err) }
	tel := res.Telemetry
	if tel.ImpactNodes != 3 || tel.ImpactUnknown != 1 {
		t.Fatalf("impact counters = nodes:%d unknown:%d", tel.ImpactNodes, tel.ImpactUnknown)
	}
	if tel.ScopeComponents != 2 || tel.ScopeRoutes != 2 || tel.ScopeRegions != 1 || tel.ScopeViewports != 2 || tel.ScopeThemes != 1 || tel.ScopeSize != 5 {
		t.Fatalf("invalidation counters = %#v", tel)
	}

	wantMeasured := tel.ImpactMS + tel.InvalidationMS + tel.FidelityScanMS + tel.RouteMS + tel.CollectMS + tel.VerifyMS + tel.SynthesisMS
	if math.Abs(tel.MeasuredStageMS-wantMeasured) > 1e-9 {
		t.Fatalf("MeasuredStageMS=%f, sum=%f", tel.MeasuredStageMS, wantMeasured)
	}
	if tel.MeasuredStageMS-tel.TotalMS > 0.01 {
		t.Fatalf("named stages exceed total, measured=%f total=%f", tel.MeasuredStageMS, tel.TotalMS)
	}
	if tel.UnattributedMS < 0 {
		t.Fatalf("negative unattributed time: %f", tel.UnattributedMS)
	}
	if math.Abs((tel.MeasuredStageMS+tel.UnattributedMS)-tel.TotalMS) > 0.01 {
		t.Fatalf("accounting mismatch: measured=%f unattributed=%f total=%f", tel.MeasuredStageMS, tel.UnattributedMS, tel.TotalMS)
	}
}
