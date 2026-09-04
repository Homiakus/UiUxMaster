package uiuxadapter

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastcdp"
)

func TestAxiomFastCDPEndToEndIntegration(t *testing.T) {
	if os.Getenv("UIUX_AXIOM_FASTCDP_INTEGRATION") != "1" {
		t.Skip("set UIUX_AXIOM_FASTCDP_INTEGRATION=1 to run against a real Chromium binary")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	runtime, err := fastcdp.StartResidentRuntime(ctx, fastcdp.RuntimeConfig{
		Browser: fastcdp.BrowserConfig{Executable: os.Getenv("UIUX_CHROME_BIN")},
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
		Viewport: evidence.Viewport{Width: 320, Height: 200, DeviceScale: 1},
		Scenario: "axiom-fastcdp-e2e",
		FidelityID: "blink-l2",
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
		window.__UIUX_SIGNAL_RENDER__(1);
	`)

	clean, err := runner.StartAndRun(ctx, "chromium-clean", controlplane.Change{Intent: "quick_structural"}, controlplane.Budget{MaxBrowserFetches: 4})
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
		window.__UIUX_SIGNAL_RENDER__(2);
	`)

	broken, err := runner.StartAndRun(ctx, "chromium-broken", controlplane.Change{Intent: "quick_structural"}, controlplane.Budget{MaxBrowserFetches: 4})
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
		"expression": expression,
		"returnByValue": true,
	}, nil); err != nil {
		lease.Release()
		t.Fatal(err)
	}
	lease.Release()
}
