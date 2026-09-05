package playwright_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/runtime/dispatcher"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastcdp"
	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
)

func TestTruthPathCalibrationRealChromium(t *testing.T) {
	if os.Getenv("UIUX_TRUTHPATH_INTEGRATION") != "1" {
		t.Skip("set UIUX_TRUTHPATH_INTEGRATION=1 to run the real calibration identity proof")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	adapter := playwright.New(playwright.Config{
		WorkerScript: filepath.Join("worker", "worker.cjs"),
		ProbeTimeout: 2 * time.Minute,
		Timeout:      time.Minute,
	})
	readiness, err := adapter.Probe(ctx)
	if err != nil { t.Fatalf("probe: %v", err) }

	var executable string
	for _, browser := range readiness.Browsers {
		if browser.Browser == playwright.BrowserChromium && browser.Ready {
			executable = browser.ExecutablePath
			break
		}
	}
	if executable == "" { t.Fatalf("runtime-attested Chromium executable missing: %#v", readiness.Browsers) }

	resident, err := fastcdp.StartResidentRuntime(ctx, fastcdp.RuntimeConfig{
		Browser: fastcdp.BrowserConfig{
			Executable: executable, StartupTimeout: 30 * time.Second, NoSandbox: true,
			ExtraArgs: []string{"--disable-dev-shm-usage"},
		},
		Pages: fastcdp.PagePoolConfig{MaxPages: 1, Page: fastcdp.PageSpec{URL: "about:blank", Width: 640, Height: 480, DPR: 1}},
	})
	if err != nil { t.Fatalf("start FastCDP on Playwright Chromium: %v", err) }
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		if err := resident.Close(closeCtx); err != nil { t.Errorf("close FastCDP: %v", err) }
	}()

	l2, err := dispatcher.NewCDPCollector(ctx, resident, dispatcher.CDPCollectorConfig{
		Viewport: evidence.Viewport{Width: 640, Height: 480, DeviceScale: 1}, FidelityID: "blink-l2",
	})
	if err != nil { t.Fatal(err) }
	l3 := playwright.NewCollector(adapter, playwright.BrowserChromium)
	d := dispatcher.New(dispatcher.Config{L2Collector: l2, L3Collector: l3})

	packet := evidence.Packet{
		Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp", Version: mustL2Version(t, ctx, l2), FidelityID: "blink-l2"},
		Viewport: evidence.Viewport{Width: 640, Height: 480, DeviceScale: 1},
	}
	req := engine.ValidationRequest{RunID: "fmea008-real-calibration", RequireLegalPass: true, Intent: evidenceplan.IntentQuickStructural}
	plan := engine.ValidationPlan{Need: engine.EvidenceNeed{Geometry: true}, EvidencePlan: evidenceplan.Plan{Structural: true}}
	current, err := d.CalibrationContext(ctx, req, plan, packet)
	if err != nil { t.Fatalf("real calibration context: %v", err) }
	if current.Approx.BrowserVersion == "" || current.Truth.BrowserVersion == "" || current.Truth.WorkerVersion != playwright.WorkerProtocolVersion || current.Truth.RuntimeVersion != playwright.PinnedPlaywrightVersion {
		t.Fatalf("runtime identity incomplete: %#v", current)
	}

	key, err := current.Key()
	if err != nil { t.Fatal(err) }
	now := time.Now().UTC()
	registry := fidelity.NewCalibrationRegistry()
	if err := registry.Put(fidelity.CalibrationRecord{
		Class: fidelity.EvidenceClassStaticLayout, Tier: fidelity.TierL2, Context: current, EnvironmentKey: key,
		CorpusDigest: "sha256:real-chromium-parity", ArtifactRef: "ci://truthpath/fmea008-real-chromium",
		Samples: 100, PassedSamples: 100, CreatedAt: now, ExpiresAt: now.Add(24*time.Hour),
	}); err != nil { t.Fatal(err) }
	authority := fidelity.NewCalibrationAuthority(registry, fidelity.DefaultCalibrationPolicy())
	authority.Now = func() time.Time { return now.Add(time.Minute) }
	if _, err := authority.Validate(fidelity.EvidenceClassStaticLayout, fidelity.TierL2, current); err != nil {
		t.Fatalf("actual exact runtime pair should retain PASS: %v", err)
	}

	drifted := current
	drifted.Truth.WorkerVersion = current.Truth.WorkerVersion + "-drift"
	if _, err := authority.Validate(fidelity.EvidenceClassStaticLayout, fidelity.TierL2, drifted); !errors.Is(err, fidelity.ErrCalibrationEnvironmentMismatch) {
		t.Fatalf("real runtime-derived calibration survived TruthPath identity drift: %v", err)
	}
}

func mustL2Version(t *testing.T, ctx context.Context, collector interface {
	CalibrationEnvironment(context.Context) (fidelity.CalibrationEnvironment, error)
}) string {
	t.Helper()
	env, err := collector.CalibrationEnvironment(ctx)
	if err != nil { t.Fatal(err) }
	return env.RendererVersion
}
