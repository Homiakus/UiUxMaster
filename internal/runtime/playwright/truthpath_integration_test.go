package playwright_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
)

func TestTruthPathRealChromium(t *testing.T) {
	if os.Getenv("UIUX_TRUTHPATH_INTEGRATION") != "1" {
		t.Skip("set UIUX_TRUTHPATH_INTEGRATION=1 to run the real Playwright worker")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	adapter := playwright.New(playwright.Config{
		WorkerScript: filepath.Join("worker", "worker.cjs"),
		ProbeTimeout: 2 * time.Minute,
		Timeout:      1 * time.Minute,
	})

	if caps := adapter.Capabilities(); caps.Ready || len(caps.Browsers) != 0 {
		t.Fatalf("unprobed TruthPath must not advertise L3 readiness: %#v", caps)
	}

	readiness, err := adapter.Probe(ctx)
	if err != nil {
		t.Fatalf("real readiness probe: %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("readiness = %#v", readiness)
	}
	if readiness.WorkerVersion != playwright.WorkerProtocolVersion {
		t.Fatalf("worker version = %q want %q", readiness.WorkerVersion, playwright.WorkerProtocolVersion)
	}
	if readiness.PlaywrightVersion != playwright.PinnedPlaywrightVersion {
		t.Fatalf("playwright version = %q want %q", readiness.PlaywrightVersion, playwright.PinnedPlaywrightVersion)
	}

	var chromiumVersion string
	for _, browser := range readiness.Browsers {
		if browser.Browser == playwright.BrowserChromium && browser.Ready {
			chromiumVersion = browser.Version
		}
	}
	if chromiumVersion == "" {
		t.Fatalf("Chromium was not runtime-attested: %#v", readiness.Browsers)
	}

	caps := adapter.Capabilities()
	if !caps.Ready || !caps.CleanState || !caps.SupportsARIA || !caps.SupportsFonts || !caps.SupportsScenario || !caps.SupportsROI {
		t.Fatalf("capabilities do not reflect probed worker features: %#v", caps)
	}
	if caps.BrowserVersions[playwright.BrowserChromium] != chromiumVersion {
		t.Fatalf("chromium capability version = %q want %q", caps.BrowserVersions[playwright.BrowserChromium], chromiumVersion)
	}

	html := []byte(`<!doctype html><html><head><style>
		body{margin:0;font-family:sans-serif;background:#fff;color:#111}
		main{width:300px;height:150px;padding:8px}
		button{width:120px;height:48px}
	</style></head><body><main aria-label="TruthPath fixture"><h1>Runtime proof</h1><button id="publish" aria-label="Publish">Publish</button></main>
	<script>document.getElementById('publish').addEventListener('click',e=>{e.currentTarget.textContent='Published';e.currentTarget.setAttribute('aria-label','Published')})</script>
	</body></html>`)

	packet, err := adapter.Capture(ctx, playwright.TruthPathRequest{
		RunID:              "truthpath-real-chromium",
		Browser:            playwright.BrowserChromium,
		HTML:               html,
		Viewport:           evidence.Viewport{Width: 640, Height: 480, DeviceScale: 1},
		Region:             &evidence.Rect{X: 0, Y: 0, Width: 320, Height: 220},
		CapturePixels:      true,
		CaptureARIA:        true,
		CaptureFonts:       true,
		CaptureDiagnostics: true,
		CaptureLayout:      true,
		PauseAnimations:    true,
		FreezeClock:        true,
	})
	if err != nil {
		t.Fatalf("real capture: %v", err)
	}
	assertRealTruthPathPacket(t, packet, chromiumVersion)

	scenarioPacket, err := adapter.RunScenario(ctx, playwright.TruthPathRequest{
		RunID:              "truthpath-real-scenario",
		Browser:            playwright.BrowserChromium,
		HTML:               html,
		Viewport:           evidence.Viewport{Width: 640, Height: 480, DeviceScale: 1},
		CaptureARIA:        true,
		CaptureDiagnostics: true,
		CaptureLayout:      true,
	}, playwright.Scenario{
		ID: "publish-flow",
		Actions: []playwright.ScenarioAction{
			{Kind: "click", Selector: "#publish"},
		},
	})
	if err != nil {
		t.Fatalf("real scenario: %v", err)
	}
	if scenarioPacket.Scenario != "publish-flow" {
		t.Fatalf("scenario = %q", scenarioPacket.Scenario)
	}
	foundPublished := false
	for _, el := range scenarioPacket.Elements {
		if el.Role == "button" && el.Name == "Published" {
			foundPublished = true
			break
		}
	}
	if !foundPublished {
		t.Fatalf("scenario evidence did not capture post-click semantic state: %#v", scenarioPacket.Elements)
	}
}

func assertRealTruthPathPacket(t *testing.T, packet evidence.Packet, chromiumVersion string) {
	t.Helper()
	if packet.RunID != "truthpath-real-chromium" {
		t.Fatalf("run id = %q", packet.RunID)
	}
	if packet.Renderer.Tier != "L3" || packet.Renderer.Name != "playwright-chromium" {
		t.Fatalf("renderer = %#v", packet.Renderer)
	}
	for _, want := range []string{
		"worker=" + playwright.WorkerProtocolVersion,
		"playwright=" + playwright.PinnedPlaywrightVersion,
		"browser=" + chromiumVersion,
	} {
		if !strings.Contains(packet.Renderer.Version, want) {
			t.Fatalf("renderer identity %q lacks %q", packet.Renderer.Version, want)
		}
	}
	if packet.Renderer.FidelityID != "truthpath:"+packet.Renderer.Version {
		t.Fatalf("fidelity id = %q", packet.Renderer.FidelityID)
	}
	if packet.Pixels == nil || packet.Pixels.EncodedBytes == 0 || packet.Pixels.Width != 320 || packet.Pixels.Height != 220 {
		t.Fatalf("pixel evidence = %#v", packet.Pixels)
	}
	if len(packet.Documents) == 0 || packet.Documents[0].ContentWidth <= 0 || packet.Documents[0].ContentHeight <= 0 {
		t.Fatalf("document evidence = %#v", packet.Documents)
	}
	foundButton := false
	for _, el := range packet.Elements {
		if el.Role == "button" && el.Name == "Publish" && el.Visible {
			foundButton = true
			break
		}
	}
	if !foundButton {
		t.Fatalf("button not found in structural evidence: %#v", packet.Elements)
	}
	if len(packet.Accessibility) == 0 || packet.AriaSnapshot == "" {
		t.Fatalf("accessibility evidence incomplete: nodes=%d snapshot=%q", len(packet.Accessibility), packet.AriaSnapshot)
	}
	if packet.Fonts == nil || packet.Fonts.Status == "" {
		t.Fatalf("font evidence = %#v", packet.Fonts)
	}
	if packet.Diagnostics == nil || !packet.Diagnostics.Complete {
		t.Fatalf("diagnostics = %#v", packet.Diagnostics)
	}
}
