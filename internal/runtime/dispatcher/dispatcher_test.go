package dispatcher

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
	"github.com/Homiakus/UiUxMaster/internal/runtime/wggo"
)

type mockL1Renderer struct {
	caps fastrender.Capabilities
	ev   fastrender.Evidence
	err  error
}

func (m *mockL1Renderer) Render(_ context.Context, _ fastrender.Request) (fastrender.Evidence, error) {
	return m.ev, m.err
}

func (m *mockL1Renderer) Inspect(_ context.Context, _ fastrender.InspectRequest) (fastrender.StructuralEvidence, error) {
	return fastrender.StructuralEvidence{Renderer: m.caps}, nil
}

func (m *mockL1Renderer) CaptureRegion(_ context.Context, _ fastrender.RegionRequest) (fastrender.Evidence, error) {
	return m.ev, m.err
}

func (m *mockL1Renderer) RunScenario(_ context.Context, _ fastrender.Scenario) (fastrender.ScenarioEvidence, error) {
	return fastrender.ScenarioEvidence{}, nil
}

func (m *mockL1Renderer) Capabilities() fastrender.Capabilities {
	return m.caps
}

type mockL2Collector struct {
	called   bool
	lastReq  engine.ValidationRequest
	lastPlan evidenceplan.Plan
	packet   evidence.Packet
	err      error
}

func (m *mockL2Collector) CollectL2(_ context.Context, req engine.ValidationRequest, plan evidenceplan.Plan) (evidence.Packet, error) {
	m.called = true
	m.lastReq = req
	m.lastPlan = plan
	p := m.packet
	p.RunID = req.RunID
	return p, m.err
}

func TestDispatcher_CollectL0_Static(t *testing.T) {
	d := New(Config{})

	req := engine.ValidationRequest{
		RunID:        "run-static-1",
		ChangedFiles: []string{"tokens.json"},
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierStatic,
		},
	}

	packet, err := d.Collect(context.Background(), req, plan)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if packet.Renderer.Tier != string(engine.TierStatic) {
		t.Fatalf("tier = %s, want %s", packet.Renderer.Tier, engine.TierStatic)
	}
	if packet.RunID != "run-static-1" {
		t.Fatalf("RunID = %s, want run-static-1", packet.RunID)
	}
	if len(packet.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(packet.Documents))
	}
}

func TestDispatcher_CollectL1_FastRender(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	img.Set(10, 10, color.RGBA{R: 255, A: 255})

	mockL1 := &mockL1Renderer{
		caps: fastrender.Capabilities{
			Name:           "mock-fastrender",
			Version:        "v1.0",
			SupportsPixels: true,
		},
		ev: fastrender.Evidence{
			RGBA:       img,
			FidelityID: "mock-l1",
		},
	}

	d := New(Config{
		L1Renderer: mockL1,
	})

	req := engine.ValidationRequest{
		RunID: "run-l1-1",
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierFastRender,
		},
		EvidencePlan: evidenceplan.Plan{
			Pixels: true,
		},
	}

	packet, err := d.Collect(context.Background(), req, plan)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if packet.Renderer.Tier != string(engine.TierFastRender) {
		t.Fatalf("tier = %s, want %s", packet.Renderer.Tier, engine.TierFastRender)
	}
	if packet.Renderer.Name != "mock-fastrender" {
		t.Fatalf("renderer name = %s, want mock-fastrender", packet.Renderer.Name)
	}
	if packet.Pixels == nil {
		t.Fatalf("expected Pixels evidence to be non-nil")
	}
	if packet.Pixels.DigestSHA256 == "" {
		t.Fatalf("expected non-empty SHA256 digest")
	}
	if packet.Pixels.Width != 100 || packet.Pixels.Height != 100 {
		t.Fatalf("pixels bounds = %dx%d, want 100x100", packet.Pixels.Width, packet.Pixels.Height)
	}
}

func TestDispatcher_CollectL2_FastBrowser(t *testing.T) {
	mockL2 := &mockL2Collector{
		packet: evidence.Packet{
			Renderer: evidence.RendererRef{
				Tier: string(engine.TierFastBrowser),
				Name: "chromium-cdp",
			},
			AriaSnapshot: "root: button 'Submit'",
		},
	}

	d := New(Config{
		L2Collector: mockL2,
	})

	req := engine.ValidationRequest{
		RunID: "run-l2-1",
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierFastBrowser,
		},
		EvidencePlan: evidenceplan.Plan{
			Accessibility: true,
			BrowserTruth:  true,
		},
	}

	packet, err := d.Collect(context.Background(), req, plan)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if !mockL2.called {
		t.Fatalf("expected mockL2 to be called")
	}
	if packet.Renderer.Tier != string(engine.TierFastBrowser) {
		t.Fatalf("tier = %s, want %s", packet.Renderer.Tier, engine.TierFastBrowser)
	}
	if packet.AriaSnapshot != "root: button 'Submit'" {
		t.Fatalf("aria snapshot = %q", packet.AriaSnapshot)
	}
}

func TestDispatcher_EscalateL1ToL2OnUnsupported(t *testing.T) {
	mockL1 := &mockL1Renderer{
		caps: fastrender.Capabilities{
			Name: "mock-fastrender",
		},
		err: fastrender.ErrUnsupported,
	}

	mockL2 := &mockL2Collector{
		packet: evidence.Packet{
			Renderer: evidence.RendererRef{
				Tier: string(engine.TierFastBrowser),
				Name: "chromium-cdp",
			},
		},
	}

	d := New(Config{
		L1Renderer:                  mockL1,
		L2Collector:                 mockL2,
		EscalateL1ToL2OnUnsupported: true,
	})

	req := engine.ValidationRequest{
		RunID: "run-escalate-1",
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierFastRender,
		},
	}

	packet, err := d.Collect(context.Background(), req, plan)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if !mockL2.called {
		t.Fatalf("expected mockL2 to be called on escalation")
	}
	if packet.Renderer.Tier != string(engine.TierFastBrowser) {
		t.Fatalf("tier = %s, want %s", packet.Renderer.Tier, engine.TierFastBrowser)
	}
	if len(packet.RuntimeIssues) == 0 || packet.RuntimeIssues[0].Code != "L1_ESCALATION" {
		t.Fatalf("expected L1_ESCALATION runtime issue, got %#v", packet.RuntimeIssues)
	}
}

func TestDispatcher_Execute_EndToEnd(t *testing.T) {
	mockL1 := &mockL1Renderer{
		caps: fastrender.Capabilities{
			Name:           "mock-fastrender",
			SupportsPixels: true,
		},
		ev: fastrender.Evidence{
			RGBA: image.NewRGBA(image.Rect(0, 0, 50, 50)),
		},
	}

	d := New(Config{
		L1Renderer: mockL1,
	})

	req := engine.ValidationRequest{
		RunID: "run-execute-1",
		Need:  engine.EvidenceNeed{Pixels: true},
	}
	assessment := fidelity.Assessment{Risk: fidelity.RiskLow, MayVerify: true}

	plan, packet, err := d.Execute(context.Background(), req, assessment)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if plan.Route.Tier != engine.TierFastRender {
		t.Fatalf("plan tier = %s, want %s", plan.Route.Tier, engine.TierFastRender)
	}
	if packet.Renderer.Tier != string(engine.TierFastRender) {
		t.Fatalf("packet tier = %s, want %s", packet.Renderer.Tier, engine.TierFastRender)
	}
	if packet.Pixels == nil {
		t.Fatalf("expected pixels evidence in packet")
	}
}

func TestDispatcher_CollectL1_VisualDiff(t *testing.T) {
	currentImg := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			currentImg.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	baselineImg := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			baselineImg.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// Introduce a changed region in baseline
	for y := 10; y < 20; y++ {
		for x := 10; x < 20; x++ {
			baselineImg.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
		}
	}

	mockL1 := &mockL1Renderer{
		caps: fastrender.Capabilities{
			Name:           "mock-fastrender",
			SupportsPixels: true,
		},
		ev: fastrender.Evidence{
			RGBA: currentImg,
		},
	}

	d := New(Config{
		L1Renderer: mockL1,
	})

	req := engine.ValidationRequest{
		RunID:        "run-vdiff-1",
		BaselineRGBA: baselineImg,
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierFastRender,
		},
		EvidencePlan: evidenceplan.Plan{
			Pixels: true,
		},
	}

	packet, err := d.Collect(context.Background(), req, plan)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(packet.VisualRegions) == 0 {
		t.Fatalf("expected visual region for visual diff differences")
	}
	vRegion := packet.VisualRegions[0]
	if vRegion.ID != "visualdiff-changed-roi" {
		t.Fatalf("vRegion ID = %s, want visualdiff-changed-roi", vRegion.ID)
	}
	if vRegion.ChangedPixels != 100 {
		t.Fatalf("changed pixels = %d, want 100", vRegion.ChangedPixels)
	}
	if len(packet.VisualFindings) == 0 {
		t.Fatalf("expected visual findings for visual diff differences")
	}
	if packet.VisualFindings[0].Axis != "visual_regression" {
		t.Fatalf("finding axis = %s, want visual_regression", packet.VisualFindings[0].Axis)
	}
}

func TestDispatcher_CollectL1_ProhibitsGeometryPassWithoutEscalation(t *testing.T) {
	mockL1 := &mockL1Renderer{
		caps: fastrender.Capabilities{
			Name:             "mock-fastrender",
			SupportsPixels:   true,
			SupportsGeometry: false,
		},
		ev: fastrender.Evidence{
			RGBA: image.NewRGBA(image.Rect(0, 0, 10, 10)),
		},
	}

	d := New(Config{
		L1Renderer:                  mockL1,
		EscalateL1ToL2OnUnsupported: false,
	})

	req := engine.ValidationRequest{
		RunID: "run-geom-prohibit",
		Need:  engine.EvidenceNeed{Geometry: true},
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierFastRender,
		},
		EvidencePlan: evidenceplan.Plan{
			Structural: true,
		},
	}

	_, err := d.Collect(context.Background(), req, plan)
	if err == nil {
		t.Fatalf("expected error prohibiting L1 geometry pass when L1 cannot prove geometry")
	}
}

func TestDispatcher_CollectL1_EscalatesGeometryPassToL2(t *testing.T) {
	mockL1 := &mockL1Renderer{
		caps: fastrender.Capabilities{
			Name:             "mock-fastrender",
			SupportsPixels:   true,
			SupportsGeometry: false,
		},
		ev: fastrender.Evidence{
			RGBA: image.NewRGBA(image.Rect(0, 0, 10, 10)),
		},
	}

	mockL2 := &mockL2Collector{
		packet: evidence.Packet{
			Renderer: evidence.RendererRef{
				Tier: string(engine.TierFastBrowser),
				Name: "chromium-cdp",
			},
		},
	}

	d := New(Config{
		L1Renderer:                  mockL1,
		L2Collector:                 mockL2,
		EscalateL1ToL2OnUnsupported: true,
	})

	req := engine.ValidationRequest{
		RunID: "run-geom-escalate",
		Need:  engine.EvidenceNeed{Geometry: true},
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierFastRender,
		},
		EvidencePlan: evidenceplan.Plan{
			Structural: true,
		},
	}

	packet, err := d.Collect(context.Background(), req, plan)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if !mockL2.called {
		t.Fatalf("expected mockL2 to be called on geometry escalation")
	}
	if packet.Renderer.Tier != string(engine.TierFastBrowser) {
		t.Fatalf("tier = %s, want %s", packet.Renderer.Tier, engine.TierFastBrowser)
	}
	if len(packet.RuntimeIssues) == 0 || packet.RuntimeIssues[0].Code != "L1_ESCALATION" {
		t.Fatalf("expected L1_ESCALATION issue, got %#v", packet.RuntimeIssues)
	}
}

func TestDispatcher_CollectL1_WGGoIntegration(t *testing.T) {
	renderer := wggo.New(wggo.Config{})
	d := New(Config{
		L1Renderer: renderer,
	})

	html := `<!doctype html><html><body><div style="width:100px;height:50px;background-color:rgb(255,0,0);">Test</div></body></html>`
	req := engine.ValidationRequest{
		RunID: "run-wggo-1",
		HTML:  []byte(html),
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierFastRender,
		},
		EvidencePlan: evidenceplan.Plan{
			Pixels: true,
		},
	}

	packet, err := d.Collect(context.Background(), req, plan)
	if err != nil {
		t.Fatalf("Collect with WGGo failed: %v", err)
	}

	if packet.Renderer.Tier != string(engine.TierFastRender) {
		t.Fatalf("tier = %s, want %s", packet.Renderer.Tier, engine.TierFastRender)
	}
	if packet.Renderer.Name != "wggo" {
		t.Fatalf("renderer name = %s, want wggo", packet.Renderer.Name)
	}
	if packet.Pixels == nil {
		t.Fatalf("expected Pixels evidence to be non-nil")
	}
	if packet.Pixels.Width <= 0 || packet.Pixels.Height <= 0 {
		t.Fatalf("invalid pixel bounds: %dx%d", packet.Pixels.Width, packet.Pixels.Height)
	}
}

func TestParseRegionBounds(t *testing.T) {
	cases := []struct {
		input string
		want  *evidenceplan.Region
		ok    bool
	}{
		{
			input: "region:10,20,100,50",
			want:  &evidenceplan.Region{X: 10, Y: 20, Width: 100, Height: 50, Scale: 1},
			ok:    true,
		},
		{
			input: "42,80,160,44",
			want:  &evidenceplan.Region{X: 42, Y: 80, Width: 160, Height: 44, Scale: 1},
			ok:    true,
		},
		{
			input: "region:hero-actions",
			want:  nil,
			ok:    false,
		},
		{
			input: "10,20,0,50",
			want:  nil,
			ok:    false,
		},
		{
			input: "",
			want:  nil,
			ok:    false,
		},
	}

	for _, tc := range cases {
		got, ok := parseRegionBounds(tc.input)
		if ok != tc.ok {
			t.Errorf("parseRegionBounds(%q) ok = %v, want %v", tc.input, ok, tc.ok)
			continue
		}
		if tc.ok {
			if got.X != tc.want.X || got.Y != tc.want.Y || got.Width != tc.want.Width || got.Height != tc.want.Height {
				t.Errorf("parseRegionBounds(%q) = %#v, want %#v", tc.input, got, tc.want)
			}
		}
	}
}

func TestDispatcher_CollectL2_WiresImpactSetScope(t *testing.T) {
	mockL2 := &mockL2Collector{
		packet: evidence.Packet{
			Renderer: evidence.RendererRef{
				Tier: string(engine.TierFastBrowser),
				Name: "chromium-cdp",
			},
		},
	}

	d := New(Config{
		L2Collector: mockL2,
	})

	req := engine.ValidationRequest{
		RunID: "run-impact-l2",
		Scope: invalidation.ValidationScope{
			Regions: []string{"region:25,35,200,100", "region:hero-actions"},
			Routes:  []string{"/home", "/settings"},
		},
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierFastBrowser,
		},
		EvidencePlan: evidenceplan.Plan{
			Pixels: false,
		},
	}

	packet, err := d.Collect(context.Background(), req, plan)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if !mockL2.called {
		t.Fatalf("expected mockL2 to be called")
	}

	// Verify plan.Region was wired from req.Scope.Regions
	if mockL2.lastPlan.Region == nil {
		t.Fatalf("expected lastPlan.Region to be wired from Scope.Regions")
	}
	if mockL2.lastPlan.Region.X != 25 || mockL2.lastPlan.Region.Y != 35 || mockL2.lastPlan.Region.Width != 200 || mockL2.lastPlan.Region.Height != 100 {
		t.Fatalf("lastPlan.Region = %#v, want 25,35,200,100", mockL2.lastPlan.Region)
	}
	if !mockL2.lastPlan.Pixels {
		t.Fatalf("expected lastPlan.Pixels to be true when region is specified")
	}

	if packet.Renderer.Tier != string(engine.TierFastBrowser) {
		t.Fatalf("tier = %s, want %s", packet.Renderer.Tier, engine.TierFastBrowser)
	}
}

func TestCDPCollector_UninitializedError(t *testing.T) {
	var c *CDPCollector
	_, err := c.CollectL2(context.Background(), engine.ValidationRequest{}, evidenceplan.Plan{})
	if err == nil {
		t.Fatalf("expected error from uninitialized CDPCollector")
	}
}

type mockL3Collector struct {
	called   bool
	lastReq  engine.ValidationRequest
	lastPlan evidenceplan.Plan
	packet   evidence.Packet
	err      error
}

func (m *mockL3Collector) CollectL3(_ context.Context, req engine.ValidationRequest, plan evidenceplan.Plan) (evidence.Packet, error) {
	m.called = true
	m.lastReq = req
	m.lastPlan = plan
	p := m.packet
	p.RunID = req.RunID
	return p, m.err
}

func TestDispatcher_CollectL3_TruthPath(t *testing.T) {
	mockL3 := &mockL3Collector{
		packet: evidence.Packet{
			Renderer: evidence.RendererRef{
				Tier: "L3",
				Name: "playwright-chromium",
			},
			AriaSnapshot: "- button \"Submit\"",
		},
	}

	d := New(Config{
		L3Collector: mockL3,
	})

	req := engine.ValidationRequest{
		RunID: "run-l3-test",
		Scope: invalidation.ValidationScope{
			Regions: []string{"region:10,10,100,50"},
		},
	}
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{
			Tier: engine.TierTruthPath,
		},
		EvidencePlan: evidenceplan.Plan{
			Accessibility: true,
			BrowserTruth:  true,
		},
	}

	packet, err := d.Collect(context.Background(), req, plan)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if !mockL3.called {
		t.Fatalf("expected mockL3 to be called")
	}
	if packet.Renderer.Tier != "L3" {
		t.Fatalf("tier = %s, want L3", packet.Renderer.Tier)
	}
	if packet.Renderer.Name != "playwright-chromium" {
		t.Fatalf("renderer name = %s, want playwright-chromium", packet.Renderer.Name)
	}
	if mockL3.lastPlan.Region == nil || mockL3.lastPlan.Region.X != 10 {
		t.Fatalf("expected region bounds wired to L3 collector")
	}
}



