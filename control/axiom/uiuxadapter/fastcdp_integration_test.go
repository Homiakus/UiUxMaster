package uiuxadapter

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	runtimedispatcher "github.com/Homiakus/UiUxMaster/internal/runtime/dispatcher"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastcdp"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type fmea008TruthIdentityCollector struct {
	env fidelity.CalibrationEnvironment
}

func (c *fmea008TruthIdentityCollector) CollectL3(_ context.Context, req engine.ValidationRequest, _ evidenceplan.Plan) (evidence.Packet, error) {
	return evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{Tier: "L3", Name: c.env.RendererName, Version: c.env.RendererVersion, FidelityID: c.env.FidelityID},
		Viewport: evidence.Viewport{Width: 320, Height: 200, DeviceScale: 1},
	}, nil
}

func (c *fmea008TruthIdentityCollector) CalibrationEnvironment(_ context.Context) (fidelity.CalibrationEnvironment, error) {
	return c.env, nil
}

func TestAxiomFastCDPEndToEndIntegration(t *testing.T) {
	if os.Getenv("UIUX_AXIOM_FASTCDP_INTEGRATION") != "1" {
		t.Skip("set UIUX_AXIOM_FASTCDP_INTEGRATION=1 to run against a real Chromium binary")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	resident, err := fastcdp.StartResidentRuntime(ctx, fastcdp.RuntimeConfig{
		Browser: fastcdp.BrowserConfig{
			Executable:     os.Getenv("UIUX_CHROME_BIN"),
			StartupTimeout: 30 * time.Second,
			NoSandbox:      true,
			ExtraArgs:      []string{"--disable-dev-shm-usage"},
		},
		Pages: fastcdp.PagePoolConfig{
			MaxPages: 1,
			DiagnosticsCapacity: 64,
			Page: fastcdp.PageSpec{URL: "about:blank", Width: 320, Height: 200, DPR: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		if err := resident.Close(closeCtx); err != nil {
			t.Errorf("close runtime: %v", err)
		}
	}()

	// Keep the control-slice collector for the direct FMEA-003 provenance/fault
	// assertions below, but production-like Axiom decisions go through the
	// canonical engine Pipeline so PassAuthority cannot be bypassed.
	directCollector, err := NewFastCDPCollector(ctx, resident, FastCDPCollectorConfig{
		Viewport:        evidence.Viewport{Width: 320, Height: 200, DeviceScale: 1},
		Scenario:        "axiom-fastcdp-e2e",
		FidelityID:      "blink-l2",
		WaitForNewEpoch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	rootCollector, err := runtimedispatcher.NewCDPCollector(ctx, resident, runtimedispatcher.CDPCollectorConfig{
		Viewport:        evidence.Viewport{Width: 320, Height: 200, DeviceScale: 1},
		Scenario:        "axiom-fastcdp-e2e",
		FidelityID:      "blink-l2",
		WaitForNewEpoch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	truthIdentity := &fmea008TruthIdentityCollector{env: fidelity.CalibrationEnvironment{
		RendererName: "playwright-chromium",
		RendererVersion: "worker=axiom-ci;playwright=axiom-ci;browser=chromium-ci",
		FidelityID: "truthpath:axiom-ci",
		BrowserFamily: "chromium",
		BrowserVersion: "chromium-ci",
		WorkerVersion: "axiom-ci",
		RuntimeVersion: "axiom-ci",
	}}
	dispatch := runtimedispatcher.New(runtimedispatcher.Config{L2Collector: rootCollector, L3Collector: truthIdentity})

	// Build parity records against the exact live FastCDP identity and the
	// configured TruthPath oracle identity. The real TruthPath workflow separately
	// proves the oracle side with an actually launched Playwright Chromium.
	approxEnv, err := rootCollector.CalibrationEnvironment(ctx)
	if err != nil {
		t.Fatal(err)
	}
	calibrationCtx, err := dispatch.CalibrationContext(ctx,
		engine.ValidationRequest{Intent: evidenceplan.IntentQuickStructural},
		engine.ValidationPlan{Need: engine.EvidenceNeed{Geometry: true}, EvidencePlan: evidenceplan.Plan{Structural: true}},
		evidence.Packet{
			Renderer: evidence.RendererRef{Tier: "L2", Name: approxEnv.RendererName, Version: approxEnv.RendererVersion, FidelityID: approxEnv.FidelityID},
			Viewport: evidence.Viewport{Width: 320, Height: 200, DeviceScale: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	key, err := calibrationCtx.Key()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	registry := fidelity.NewCalibrationRegistry()
	for _, class := range []fidelity.EvidenceClass{
		fidelity.EvidenceClassStaticLayout,
		fidelity.EvidenceClassTypography,
		fidelity.EvidenceClassPixelRegression,
	} {
		if err := registry.Put(fidelity.CalibrationRecord{
			Class: class, Tier: fidelity.TierL2, Context: calibrationCtx, EnvironmentKey: key,
			CorpusDigest: "sha256:axiom-fastcdp-parity", ArtifactRef: "ci://axiom-control/fastcdp-parity",
			Samples: 100, PassedSamples: 100, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}
	authority := fidelity.NewCalibrationAuthority(registry, fidelity.DefaultCalibrationPolicy())
	authority.Now = func() time.Time { return now.Add(time.Minute) }
	pipeline := &engine.Pipeline{Collector: dispatch, Calibration: authority, VerPolicy: verifier.DefaultPolicy()}
	runner, err := controlplane.NewMemory(NewPipelineAdapter(pipeline))
	if err != nil {
		t.Fatal(err)
	}

	mutateWarmPage(t, ctx, resident, `
		document.documentElement.style.background = "white";
		document.body.style.margin = "0";
		document.body.innerHTML = '<main style="width:300px;height:120px"><button style="width:90px;height:48px">Publish</button></main>';
		window.__UIUX_SIGNAL_RENDER__(1, "rev-clean");
	`)

	clean, err := runner.StartAndRun(ctx, "chromium-clean", controlplane.Change{
		Intent: "quick_structural", SourceDigest: "rev-clean",
	}, controlplane.Budget{MaxBrowserFetches: 4})
	if err != nil {
		t.Fatal(err)
	}
	if clean.Status != "completed" || clean.Decision != controlplane.DecisionPass {
		t.Fatalf("clean calibrated run = %#v", clean)
	}
	if clean.Usage.BrowserFetches != 1 || !clean.Validation.DiagnosticsComplete {
		t.Fatalf("clean usage/diagnostics = %#v / %#v", clean.Usage, clean.Validation)
	}

	mutateWarmPage(t, ctx, resident, `
		console.error("axiom-cycle-error");
		window.__UIUX_SIGNAL_RENDER__(2, "rev-broken");
	`)

	broken, err := runner.StartAndRun(ctx, "chromium-broken", controlplane.Change{
		Intent: "quick_structural", SourceDigest: "rev-broken",
	}, controlplane.Budget{MaxBrowserFetches: 4})
	if err != nil {
		t.Fatal(err)
	}
	if broken.Status != "completed" || broken.Decision != controlplane.DecisionRepair {
		t.Fatalf("broken run = %#v", broken)
	}
	if broken.Validation.HighFindings == 0 {
		t.Fatalf("console error was not grounded as a high finding: %#v", broken.Validation)
	}
	if !broken.Validation.DiagnosticsComplete {
		t.Fatalf("second diagnostic window is incomplete: %#v", broken.Validation)
	}

	mutateWarmPage(t, ctx, resident, `
		document.querySelector("button").style.background = "rgb(20, 90, 200)";
		window.__UIUX_SIGNAL_RENDER__(3, "rev-visual");
	`)

	visual, err := runner.StartAndRun(ctx, "chromium-visual", controlplane.Change{
		Intent:       "visual_region",
		SourceDigest: "rev-visual",
		Region:       &controlplane.Region{X: 0, Y: 0, Width: 160, Height: 100, Scale: 1},
	}, controlplane.Budget{MaxBrowserFetches: 4})
	if err != nil {
		t.Fatal(err)
	}
	if visual.Status != "completed" || visual.Decision != controlplane.DecisionSemantic {
		t.Fatalf("visual run = %#v", visual)
	}
	if visual.Usage.BrowserFetches != 2 {
		t.Fatalf("visual browser usage = %d, want 2", visual.Usage.BrowserFetches)
	}
	if !visual.Validation.PixelEvidence || visual.Validation.VisualRegions != 1 {
		t.Fatalf("visual evidence projection = %#v", visual.Validation)
	}
	if !visual.Validation.DiagnosticsComplete {
		t.Fatalf("visual diagnostic window is incomplete: %#v", visual.Validation)
	}

	// Direct real-browser provenance proof remains on the isolated control
	// collector: the packet must carry the exact revision that released the waiter.
	mutateWarmPage(t, ctx, resident, `window.__UIUX_SIGNAL_RENDER__(4, "rev-attested");`)
	packet, err := directCollector.Collect(ctx, controlplane.Change{SourceDigest: "rev-attested"}, evidenceplan.Plan{Structural: true})
	if err != nil {
		t.Fatalf("attested collection: %v", err)
	}
	if packet.Freshness == nil || packet.Freshness.Epoch != 4 || packet.Freshness.ExpectedRevision != "rev-attested" || packet.Freshness.ObservedRevision != "rev-attested" {
		t.Fatalf("freshness provenance = %#v", packet.Freshness)
	}

	mutateWarmPage(t, ctx, resident, `window.__UIUX_SIGNAL_RENDER__(5, "rev-wrong");`)
	_, err = directCollector.Collect(ctx, controlplane.Change{SourceDigest: "rev-expected"}, evidenceplan.Plan{Structural: true})
	if !errors.Is(err, fastcdp.ErrRevisionMismatch) {
		t.Fatalf("mismatch err = %v, want ErrRevisionMismatch", err)
	}

	mutateWarmPage(t, ctx, resident, `
		document.body.innerHTML = '<main><button>Recovered</button></main>';
		window.__UIUX_SIGNAL_RENDER__(1, "rev-recovered");
	`)
	recovered, err := directCollector.Collect(ctx, controlplane.Change{SourceDigest: "rev-recovered"}, evidenceplan.Plan{Structural: true})
	if err != nil {
		t.Fatalf("recovered collection: %v", err)
	}
	if recovered.Freshness == nil || recovered.Freshness.ObservedRevision != "rev-recovered" || !recovered.Freshness.Matches() {
		t.Fatalf("recovered freshness = %#v", recovered.Freshness)
	}
}

func mutateWarmPage(t *testing.T, ctx context.Context, resident *fastcdp.ResidentRuntime, expression string) {
	t.Helper()
	lease, err := resident.Pages.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page := lease.Page()
	if page == nil {
		lease.Release()
		t.Fatal("nil warm page")
	}
	if err := resident.Conn.Call(ctx, string(page.Session.SessionID), "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
	}, nil); err != nil {
		lease.Release()
		t.Fatal(err)
	}
	lease.Release()
}
