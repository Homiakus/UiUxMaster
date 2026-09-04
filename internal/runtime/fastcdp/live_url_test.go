package fastcdp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

func TestLiveServerWarmCapture(t *testing.T) {
	if os.Getenv("UIUX_FASTCDP_INTEGRATION") != "1" {
		t.Skip("set UIUX_FASTCDP_INTEGRATION=1 to run against a real Chromium binary")
	}

	// 1. Setup a live HTTP server simulating a dev server (Vite / Next.js / Vanilla)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>Hydropilot UI Console</title>
	<style>
		:root { --brand: #0070f3; --bg: #ffffff; }
		body { margin: 0; padding: 20px; font-family: sans-serif; background: var(--bg); }
		.card { width: 400px; padding: 16px; border: 1px solid #eaeaea; border-radius: 8px; }
		.btn { background: var(--brand); color: white; border: none; padding: 10px 20px; border-radius: 4px; font-size: 14px; cursor: pointer; }
	</style>
</head>
<body>
	<main id="app">
		<section class="card" id="control-card">
			<h1>Hydropilot Engine Monitor</h1>
			<p>Live system telemetry and pump pressure control.</p>
			<button id="start-pump-btn" class="btn" aria-label="Start primary pump">Start Pump</button>
		</section>
	</main>
	<script>
		window.__UIUX_SIGNAL_RENDER__ && window.__UIUX_SIGNAL_RENDER__(1);
	</script>
</body>
</html>`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 2. Launch resident runtime attached to live server URL
	runtime, err := StartResidentRuntime(ctx, RuntimeConfig{
		Browser: BrowserConfig{Executable: os.Getenv("UIUX_CHROME_BIN")},
		Pages: PagePoolConfig{
			MaxPages:            1,
			DiagnosticsCapacity: 64,
			Page: PageSpec{
				URL:    server.URL,
				Width:  1280,
				Height: 800,
				DPR:    1,
			},
		},
	})
	if err != nil {
		t.Fatalf("StartResidentRuntime failed: %v", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		if err := runtime.Close(closeCtx); err != nil {
			t.Errorf("runtime close error: %v", err)
		}
	}()

	version, err := runtime.Version(ctx)
	if err != nil {
		t.Fatalf("runtime version failed: %v", err)
	}

	lease, err := runtime.Pages.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire page lease failed: %v", err)
	}
	defer lease.Release()
	page := lease.Page()

	mark := page.Diagnostics.Mark()
	before := page.Epoch.Current()

	// 3. Trigger warm render signal and collect complete evidence including ROI screenshot
	captureStart := time.Now()
	collected, err := page.CollectEvidence(ctx, runtime.Conn, EvidenceRequest{
		RequireAfter: before,
		Snapshot:     ptrSnapshotOptions(DefaultSnapshotOptions()),
		Region: &CaptureRegionOptions{
			X:               20,
			Y:               20,
			Width:           400,
			Height:          200,
			Scale:           1,
			OptimizeForSpeed: true,
		},
		Accessibility:    true,
		Fonts:            true,
		DiagnosticsSince: &mark,
	})
	captureDuration := time.Since(captureStart)
	if err != nil {
		t.Fatalf("CollectEvidence failed: %v", err)
	}

	t.Logf("Live server warm capture timing: Total=%v, Snapshot=%v, Pixels=%v, A11y=%v, Fonts=%v",
		collected.Timing.Total, collected.Timing.Snapshot, collected.Timing.Pixels, collected.Timing.Accessibility, collected.Timing.Fonts)

	// 4. Validate captured evidence artifacts
	if collected.Snapshot == nil || len(collected.Snapshot.Documents) == 0 {
		t.Fatal("expected non-empty DOMSnapshot")
	}
	if collected.Accessibility == nil || len(collected.Accessibility.Nodes) == 0 {
		t.Fatal("expected non-empty AXTree")
	}
	if collected.Fonts == nil {
		t.Fatal("expected non-empty FontState")
	}
	if collected.RGBA == nil || collected.RGBA.Bounds().Dx() != 400 || collected.RGBA.Bounds().Dy() != 200 {
		t.Fatalf("invalid ROI RGBA capture: %v", collected.RGBA)
	}

	// 5. Convert to canonical evidence.Packet and assert structure
	packet := ToPacket(collected, PacketOptions{
		RunID:      "live-server-test",
		Scenario:   "hydropilot-telemetry",
		Viewport:   evidence.Viewport{Width: 1280, Height: 800, DeviceScale: 1},
		Browser:    version,
		FidelityID: "blink-l2",
		Region: &CaptureRegionOptions{
			X:      20,
			Y:      20,
			Width:  400,
			Height: 200,
			Scale:  1,
		},
	})

	if packet.Pixels == nil || packet.Pixels.Width != 400 || packet.Pixels.Height != 200 {
		t.Fatalf("invalid packet pixels: %#v", packet.Pixels)
	}
	if len(packet.Elements) == 0 {
		t.Fatal("expected populated element list in packet")
	}
	if len(packet.Accessibility) == 0 {
		t.Fatal("expected populated accessibility list in packet")
	}

	verifier.ApplyDeterministic(&packet, verifier.DefaultPolicy())
	t.Logf("Verification completed: %d runtime issues, capture duration=%v", len(packet.RuntimeIssues), captureDuration)
}
