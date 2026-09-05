package uiuxadapter

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastcdp"
)

func TestAxiomFastCDPEndToEndIntegration(t *testing.T) {
	if os.Getenv("UIUX_AXIOM_FASTCDP_INTEGRATION") != "1" {
		t.Skip("set UIUX_AXIOM_FASTCDP_INTEGRATION=1 to run against a real Chromium binary")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	runtime, err := fastcdp.StartResidentRuntime(ctx, fastcdp.RuntimeConfig{
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
		if err := runtime.Close(closeCtx); err != nil {
			t.Errorf("close runtime: %v", err)
		}
	}()

	collector, err := NewFastCDPCollector(ctx, runtime, FastCDPCollectorConfig{
		Viewport:        evidence.Viewport{Width: 320, Height: 200, DeviceScale: 1},
		Scenario:        "axiom-fastcdp-e2e",
		FidelityID:      "blink-l2",
		WaitForNewEpoch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	runner, err := controlplane.NewMemory(New(collector))
	if err != nil {
		t.Fatal(err)
	}

	mutateWarmPage(t, ctx, runtime, `
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
		t.Fatalf("clean run = %#v", clean)
	}
	if clean.Usage.BrowserFetches != 1 || !clean.Validation.DiagnosticsComplete {
		t.Fatalf("clean usage/diagnostics = %#v / %#v", clean.Usage, clean.Validation)
	}

	mutateWarmPage(t, ctx, runtime, `
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

	mutateWarmPage(t, ctx, runtime, `
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

	// Direct real-browser provenance proof: the packet must carry the exact
	// revision that released the epoch waiter.
	mutateWarmPage(t, ctx, runtime, `window.__UIUX_SIGNAL_RENDER__(4, "rev-attested");`)
	packet, err := collector.Collect(ctx, controlplane.Change{SourceDigest: "rev-attested"}, evidenceplan.Plan{Structural: true})
	if err != nil {
		t.Fatalf("attested collection: %v", err)
	}
	if packet.Freshness == nil || packet.Freshness.Epoch != 4 || packet.Freshness.ExpectedRevision != "rev-attested" || packet.Freshness.ObservedRevision != "rev-attested" {
		t.Fatalf("freshness provenance = %#v", packet.Freshness)
	}

	// A newer numeric epoch with the wrong revision must be rejected and the page
	// discarded/reset rather than converted into usable evidence.
	mutateWarmPage(t, ctx, runtime, `window.__UIUX_SIGNAL_RENDER__(5, "rev-wrong");`)
	_, err = collector.Collect(ctx, controlplane.Change{SourceDigest: "rev-expected"}, evidenceplan.Plan{Structural: true})
	if !errors.Is(err, fastcdp.ErrRevisionMismatch) {
		t.Fatalf("mismatch err = %v, want ErrRevisionMismatch", err)
	}

	// After mismatch/discard, a fresh warm page can establish a new token lineage
	// and recover with a matching revision.
	mutateWarmPage(t, ctx, runtime, `
		document.body.innerHTML = '<main><button>Recovered</button></main>';
		window.__UIUX_SIGNAL_RENDER__(1, "rev-recovered");
	`)
	recovered, err := collector.Collect(ctx, controlplane.Change{SourceDigest: "rev-recovered"}, evidenceplan.Plan{Structural: true})
	if err != nil {
		t.Fatalf("recovered collection: %v", err)
	}
	if recovered.Freshness == nil || recovered.Freshness.ObservedRevision != "rev-recovered" || !recovered.Freshness.Matches() {
		t.Fatalf("recovered freshness = %#v", recovered.Freshness)
	}
}

func mutateWarmPage(t *testing.T, ctx context.Context, runtime *fastcdp.ResidentRuntime, expression string) {
	t.Helper()
	lease, err := runtime.Pages.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page := lease.Page()
	if page == nil {
		lease.Release()
		t.Fatal("nil warm page")
	}
	if err := runtime.Conn.Call(ctx, string(page.Session.SessionID), "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
	}, nil); err != nil {
		lease.Release()
		t.Fatal(err)
	}
	lease.Release()
}
