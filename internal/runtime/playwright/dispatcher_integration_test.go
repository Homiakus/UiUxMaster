package playwright_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/runtime/dispatcher"
	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
)

func TestTruthPathDispatcherRealChromium(t *testing.T) {
	if os.Getenv("UIUX_TRUTHPATH_INTEGRATION") != "1" {
		t.Skip("set UIUX_TRUTHPATH_INTEGRATION=1 to run the real Playwright worker")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	adapter := playwright.New(playwright.Config{
		WorkerScript: filepath.Join("worker", "worker.cjs"),
		ProbeTimeout: 2 * time.Minute,
		Timeout:      time.Minute,
	})
	if _, err := adapter.Probe(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}

	l3 := playwright.NewCollector(adapter, playwright.BrowserChromium)
	d := dispatcher.New(dispatcher.Config{L3Collector: l3})
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{Tier: engine.TierTruthPath},
		EvidencePlan: evidenceplan.Plan{
			BrowserTruth:  true,
			Structural:    true,
			Accessibility: true,
			Diagnostics:   true,
		},
	}
	packet, err := d.Collect(ctx, engine.ValidationRequest{
		RunID: "truthpath-dispatcher-real",
		HTML: []byte(`<!doctype html><html><body><main><button aria-label="Approve">Approve</button></main></body></html>`),
	}, plan)
	if err != nil {
		t.Fatalf("dispatcher L3 collect: %v", err)
	}
	if packet.Renderer.Tier != "L3" {
		t.Fatalf("actual tier = %q, want L3", packet.Renderer.Tier)
	}
	if packet.Renderer.FidelityID == "" {
		t.Fatal("real TruthPath packet lacks runtime fidelity identity")
	}
}
