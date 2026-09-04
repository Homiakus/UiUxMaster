package engine_test

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
	"github.com/Homiakus/UiUxMaster/internal/runtime/dispatcher"
	"github.com/Homiakus/UiUxMaster/internal/runtime/wggo"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type mockCDPCollector struct {
	called   bool
	lastReq  engine.ValidationRequest
	lastPlan evidenceplan.Plan
}

func (m *mockCDPCollector) CollectL2(_ context.Context, req engine.ValidationRequest, plan evidenceplan.Plan) (evidence.Packet, error) {
	m.called = true
	m.lastReq = req
	m.lastPlan = plan

	packet := evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{
			Tier:       string(engine.TierFastBrowser),
			Name:       "chromium-cdp",
			Version:    "126.0",
			FidelityID: "blink-l2",
		},
		Viewport:     evidence.Viewport{Width: 1280, Height: 800, DeviceScale: 1},
		AriaSnapshot: "root: button 'Action'",
		Elements: []evidence.ElementRef{
			{
				ID:     "btn-1",
				Role:   "button",
				Bounds: evidence.Rect{X: 10, Y: 10, Width: 100, Height: 40},
			},
		},
	}
	if plan.Pixels && plan.Region != nil {
		packet.Pixels = &evidence.PixelEvidence{
			Bounds: evidence.Rect{
				X:      plan.Region.X,
				Y:      plan.Region.Y,
				Width:  plan.Region.Width,
				Height: plan.Region.Height,
			},
			Width:  int(plan.Region.Width),
			Height: int(plan.Region.Height),
		}
	}
	return packet, nil
}

func TestPipeline_EndToEnd_CSSToken_WGGo(t *testing.T) {
	ctx := context.Background()

	// 1. Setup frontend dependency graph with impact.Builder
	builder := impact.NewBuilder()
	if err := builder.TokenAffects("token:color-primary", "component:hero-btn", impact.NodeComponent); err != nil {
		t.Fatalf("builder.TokenAffects failed: %v", err)
	}
	if err := builder.ComponentInstance("component:hero-btn", "instance:hero-cta"); err != nil {
		t.Fatalf("builder.ComponentInstance failed: %v", err)
	}
	if err := builder.PlaceInstance("instance:hero-cta", "page:home", "region:10,20,120,40"); err != nil {
		t.Fatalf("builder.PlaceInstance failed: %v", err)
	}
	resolver, err := impact.NewResolver(builder.Graph())
	if err != nil {
		t.Fatalf("impact.NewResolver failed: %v", err)
	}

	// 2. Setup Dispatcher with real WGGo L1 renderer
	wggoRenderer := wggo.New(wggo.Config{})
	d := dispatcher.New(dispatcher.Config{
		L1Renderer: wggoRenderer,
	})

	// 3. Setup Pipeline
	pipeline := engine.Pipeline{
		Resolver:  resolver,
		Policy:    invalidation.DefaultPolicy(),
		Collector: d,
	}

	// 4. Submit ValidationRequest with changed CSS token for pixel validation
	req := engine.ValidationRequest{
		RunID:         "run-e2e-wggo",
		ChangedTokens: []string{"color-primary"},
		HTML:          []byte(`<!doctype html><html><body><button style="width:120px;height:40px;background-color:#3b82f6;">Submit</button></body></html>`),
		CSS:           []byte(`button { border: none; }`),
		Need:          engine.EvidenceNeed{Pixels: true},
	}

	// 5. Execute full canonical pipeline
	res, err := pipeline.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Pipeline.Execute failed: %v", err)
	}

	// Verify Step 1: ImpactSet bounded scope to region:10,20,120,40
	if len(res.Scope.Regions) == 0 {
		t.Fatalf("expected Scope.Regions to contain impacted region, got %#v", res.Scope.Regions)
	}
	if res.Scope.Regions[0] != "region:10,20,120,40" {
		t.Fatalf("scope region = %q, want region:10,20,120,40", res.Scope.Regions[0])
	}

	// Verify Step 2 & 3: Fidelity assessment is Low and routed to TierFastRender
	if res.Plan.Route.Tier != engine.TierFastRender {
		t.Fatalf("route tier = %s, want %s", res.Plan.Route.Tier, engine.TierFastRender)
	}

	// Verify Step 4: Collector dispatched to WGGo and produced valid pixels
	if res.Packet.Renderer.Tier != string(engine.TierFastRender) {
		t.Fatalf("packet tier = %s, want %s", res.Packet.Renderer.Tier, engine.TierFastRender)
	}
	if res.Packet.Renderer.Name != "wggo" {
		t.Fatalf("renderer name = %s, want wggo", res.Packet.Renderer.Name)
	}
	if res.Packet.Pixels == nil {
		t.Fatalf("expected Pixels evidence to be non-nil")
	}
	if res.Packet.Pixels.Width <= 0 || res.Packet.Pixels.Height <= 0 {
		t.Fatalf("invalid pixel bounds: %dx%d", res.Packet.Pixels.Width, res.Packet.Pixels.Height)
	}

	// Verify Step 5: Deterministic verifier executed
	if res.Verification.Duration < 0 {
		t.Fatalf("invalid verification duration")
	}

	// Verify Step 6: Engine Decision & Report generated
	if res.Report.RunID != "run-e2e-wggo" {
		t.Fatalf("report runID = %q, want run-e2e-wggo", res.Report.RunID)
	}
	if res.Report.RecommendedNext == "" {
		t.Fatalf("expected non-empty RecommendedNext in engine report")
	}
}

func TestPipeline_EndToEnd_HighRisk_FastCDP(t *testing.T) {
	ctx := context.Background()

	// 1. Setup frontend dependency graph with route and region
	builder := impact.NewBuilder()
	if err := builder.TokenAffects("token:motion-bounce", "component:hero-card", impact.NodeComponent); err != nil {
		t.Fatalf("builder.TokenAffects failed: %v", err)
	}
	if err := builder.ComponentInstance("component:hero-card", "instance:anim-hero"); err != nil {
		t.Fatalf("builder.ComponentInstance failed: %v", err)
	}
	if err := builder.PlaceInstance("instance:anim-hero", "page:dashboard", "region:0,0,600,400"); err != nil {
		t.Fatalf("builder.PlaceInstance failed: %v", err)
	}
	resolver, err := impact.NewResolver(builder.Graph())
	if err != nil {
		t.Fatalf("impact.NewResolver failed: %v", err)
	}

	// 2. Setup Dispatcher with mock FastCDP collector
	mockL2 := &mockCDPCollector{}
	d := dispatcher.New(dispatcher.Config{
		L2Collector: mockL2,
	})

	pipeline := engine.Pipeline{
		Resolver:  resolver,
		Policy:    invalidation.DefaultPolicy(),
		Collector: d,
	}

	// 3. Request contains animation feature -> high fidelity risk
	req := engine.ValidationRequest{
		RunID:         "run-e2e-cdp",
		ChangedTokens: []string{"motion-bounce"},
		HTML:          []byte(`<!doctype html><html><body><div class="anim">Hero</div></body></html>`),
		CSS:           []byte(`@keyframes pulse { 0% { opacity: 0; } 100% { opacity: 1; } } .anim { animation: pulse 1s infinite; }`),
		Need:          engine.EvidenceNeed{Scenario: true, Geometry: true},
	}

	res, err := pipeline.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Pipeline.Execute failed: %v", err)
	}

	// Verify Step 1: ImpactSet bounded scope
	if len(res.Scope.Regions) == 0 || res.Scope.Regions[0] != "region:0,0,600,400" {
		t.Fatalf("scope regions = %#v, want region:0,0,600,400", res.Scope.Regions)
	}

	// Verify Step 2 & 3: Fidelity assessment escalated to FastBrowser
	if res.Plan.Route.Tier != engine.TierFastBrowser {
		t.Fatalf("route tier = %s, want %s", res.Plan.Route.Tier, engine.TierFastBrowser)
	}

	// Verify Step 4: mockCDPCollector received request and wired plan region
	if !mockL2.called {
		t.Fatalf("expected mockCDPCollector to be called")
	}
	if mockL2.lastPlan.Region == nil || mockL2.lastPlan.Region.Width != 600 {
		t.Fatalf("lastPlan.Region = %#v, want width=600", mockL2.lastPlan.Region)
	}

	// Verify Step 5 & 6: Report generated
	if res.Report.RunID != "run-e2e-cdp" {
		t.Fatalf("report runID = %s, want run-e2e-cdp", res.Report.RunID)
	}
}

func TestPipeline_EndToEnd_VerifierViolation_RepairRecommendation(t *testing.T) {
	ctx := context.Background()

	// Dispatcher returns elements with deliberate clipping/overflow defect
	defectCollector := &defectL2Collector{}
	d := dispatcher.New(dispatcher.Config{
		L2Collector: defectCollector,
	})

	pipeline := engine.Pipeline{
		Collector: d,
		VerPolicy: verifier.Policy{
			ClipTolerance:   1,
			MinTargetWidth:  48,
			MinTargetHeight: 48,
		},
	}

	req := engine.ValidationRequest{
		RunID: "run-e2e-defect",
		Need:  engine.EvidenceNeed{Geometry: true, Styles: true},
	}

	res, err := pipeline.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Pipeline.Execute failed: %v", err)
	}

	// Verify verifier caught the violation
	if len(res.Verification.Issues) == 0 {
		t.Fatalf("expected verifier issues to be detected")
	}

	// Verify engine report highlights blocking findings and recommends repair
	if res.Report.BlockingFindings == 0 && res.Report.HighFindings == 0 {
		t.Fatalf("expected blocking or high findings in engine report")
	}
	if res.Report.RecommendedNext == "" {
		t.Fatalf("expected repair recommendation in engine report")
	}
}

type defectL2Collector struct{}

func (d *defectL2Collector) CollectL2(_ context.Context, req engine.ValidationRequest, _ evidenceplan.Plan) (evidence.Packet, error) {
	return evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{
			Tier: string(engine.TierFastBrowser),
			Name: "chromium-cdp",
		},
		Viewport: evidence.Viewport{Width: 1000, Height: 800, DeviceScale: 1},
		Documents: []evidence.DocumentMetrics{
			{URL: "https://test.invalid/", ContentWidth: 1200, ContentHeight: 800},
		},
		Elements: []evidence.ElementRef{
			{
				ID:        "parent",
				Role:      "container",
				Bounds:    evidence.Rect{X: 0, Y: 0, Width: 100, Height: 100},
				Styles:    map[string]string{"overflow": "hidden"},
				Visible:   true,
				Clickable: false,
			},
			{
				ID:        "child-clipped-btn",
				ParentID:  "parent",
				Role:      "button",
				Bounds:    evidence.Rect{X: 50, Y: 50, Width: 100, Height: 100}, // clipped by parent!
				Visible:   true,
				Clickable: true,
			},
		},
	}, nil
}

func TestPipeline_EndToEnd_TelemetryExpansion(t *testing.T) {
	ctx := context.Background()

	builder := impact.NewBuilder()
	if err := builder.TokenAffects("token:color-accent", "component:box", impact.NodeComponent); err != nil {
		t.Fatalf("builder.TokenAffects failed: %v", err)
	}
	if err := builder.ComponentInstance("component:box", "instance:box-1"); err != nil {
		t.Fatalf("builder.ComponentInstance failed: %v", err)
	}
	if err := builder.PlaceInstance("instance:box-1", "page:home", "region:0,0,100,50"); err != nil {
		t.Fatalf("builder.PlaceInstance failed: %v", err)
	}
	resolver, err := impact.NewResolver(builder.Graph())
	if err != nil {
		t.Fatalf("impact.NewResolver failed: %v", err)
	}

	wggoRenderer := wggo.New(wggo.Config{})
	d := dispatcher.New(dispatcher.Config{
		L1Renderer: wggoRenderer,
	})

	pipeline := engine.Pipeline{
		Resolver:  resolver,
		Policy:    invalidation.DefaultPolicy(),
		Collector: d,
	}

	req := engine.ValidationRequest{
		RunID:         "run-telemetry",
		ChangedTokens: []string{"color-accent"},
		HTML:          []byte(`<!doctype html><html><body><div style="width:100px;height:50px;background:#eee;">Telemetry Test</div></body></html>`),
		Need:          engine.EvidenceNeed{Pixels: true},
	}

	res, err := pipeline.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Pipeline.Execute failed: %v", err)
	}

	// Verify all pipeline telemetry stages are recorded
	tel := res.Telemetry
	if tel.TotalMS <= 0 {
		t.Fatalf("expected positive TotalMS, got %f", tel.TotalMS)
	}
	if tel.Tier != string(engine.TierFastRender) {
		t.Fatalf("telemetry tier = %s, want %s", tel.Tier, engine.TierFastRender)
	}

	// Verify packet.Latency preserves end-to-end stage breakdown
	lat := res.Packet.Latency
	if lat.TotalMS <= 0 {
		t.Fatalf("expected packet.Latency.TotalMS > 0, got %f", lat.TotalMS)
	}
	if lat.FastRenderMS <= 0 {
		t.Fatalf("expected packet.Latency.FastRenderMS > 0 for L1, got %f", lat.FastRenderMS)
	}
}

