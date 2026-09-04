package playwright_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
)

func TestPlaywrightAdapter_Capabilities(t *testing.T) {
	adapter := playwright.New(playwright.Config{})
	caps := adapter.Capabilities()

	if caps.Name != "playwright-truthpath" {
		t.Errorf("expected name 'playwright-truthpath', got %q", caps.Name)
	}
	if !caps.CleanState {
		t.Errorf("expected clean state to be true")
	}
	if len(caps.Browsers) != 3 {
		t.Errorf("expected 3 supported browsers, got %d", len(caps.Browsers))
	}
}

func TestPlaywrightAdapter_Capture_MockSuccess(t *testing.T) {
	mockResp := playwright.WorkerResponse{
		Success:      true,
		URL:          "http://localhost:3000/app",
		AriaSnapshot: "- banner: Header\n- main: Content\n- button \"Submit\"",
		ScreenshotB64: base64.StdEncoding.EncodeToString([]byte("fake-png-bytes")),
		Documents: []evidence.DocumentMetrics{
			{URL: "http://localhost:3000/app", ContentWidth: 1280, ContentHeight: 800},
		},
		Elements: []evidence.ElementRef{
			{ID: "btn-1", Tag: "button", Role: "button", Name: "Submit", Visible: true},
		},
		Accessibility: []evidence.AccessibilityNode{
			{ID: "ax-1", Role: "button", Name: "Submit"},
		},
		Fonts: &evidence.FontEvidence{
			Status: "loaded",
			Faces: []evidence.FontFaceEvidence{
				{Family: "Inter", Status: "loaded"},
			},
		},
		Latency: evidence.RuntimeLatency{
			PixelsMS: 15.0,
			TotalMS:  45.0,
		},
	}

	runner := &playwright.MockRunner{Response: mockResp}
	adapter := playwright.New(playwright.Config{
		DefaultBrowser: playwright.BrowserFirefox,
		Runner:         runner,
	})

	ctx := context.Background()
	req := playwright.TruthPathRequest{
		RunID:         "run-test-01",
		Browser:       playwright.BrowserFirefox,
		URL:           "http://localhost:3000/app",
		CapturePixels: true,
		CaptureARIA:   true,
		CaptureFonts:  true,
		Viewport: evidence.Viewport{
			Width:  1280,
			Height: 800,
		},
	}

	packet, err := adapter.Capture(ctx, req)
	if err != nil {
		t.Fatalf("unexpected capture error: %v", err)
	}

	if packet.RunID != "run-test-01" {
		t.Errorf("expected RunID 'run-test-01', got %q", packet.RunID)
	}
	if packet.Renderer.Tier != "L3" {
		t.Errorf("expected renderer tier 'L3', got %q", packet.Renderer.Tier)
	}
	if packet.Renderer.Name != "playwright-firefox" {
		t.Errorf("expected renderer name 'playwright-firefox', got %q", packet.Renderer.Name)
	}
	if packet.AriaSnapshot != mockResp.AriaSnapshot {
		t.Errorf("aria snapshot mismatch: got %q", packet.AriaSnapshot)
	}
	if len(packet.Elements) != 1 || packet.Elements[0].Role != "button" {
		t.Errorf("elements mismatch: got %+v", packet.Elements)
	}
	if len(packet.Accessibility) != 1 || packet.Accessibility[0].Name != "Submit" {
		t.Errorf("accessibility mismatch: got %+v", packet.Accessibility)
	}
	if packet.Fonts == nil || len(packet.Fonts.Faces) != 1 {
		t.Errorf("fonts mismatch: got %+v", packet.Fonts)
	}
	if packet.Pixels == nil || packet.Pixels.EncodedBytes == 0 {
		t.Errorf("expected pixels to be projected, got nil or zero")
	}

	// Verify request payload received by runner
	if runner.LastReq.Browser != "firefox" {
		t.Errorf("expected worker request browser 'firefox', got %q", runner.LastReq.Browser)
	}
	if !runner.LastReq.CapturePixels || !runner.LastReq.CaptureARIA {
		t.Errorf("expected capture flags set in worker request")
	}
}

func TestPlaywrightAdapter_RunScenario(t *testing.T) {
	mockResp := playwright.WorkerResponse{
		Success: true,
		URL:     "http://localhost:3000/form",
		Elements: []evidence.ElementRef{
			{ID: "msg-success", Role: "alert", Name: "Saved successfully", Visible: true},
		},
	}

	runner := &playwright.MockRunner{Response: mockResp}
	adapter := playwright.New(playwright.Config{
		DefaultBrowser: playwright.BrowserWebKit,
		Runner:         runner,
	})

	scenario := playwright.Scenario{
		ID: "submit-form-flow",
		Actions: []playwright.ScenarioAction{
			{Kind: "fill", Selector: "#name-input", Value: "Test User"},
			{Kind: "click", Selector: "#submit-btn"},
			{Kind: "wait", Duration: 50 * time.Millisecond},
		},
	}

	req := playwright.TruthPathRequest{
		RunID:   "run-test-scenario",
		Browser: playwright.BrowserWebKit,
	}

	packet, err := adapter.RunScenario(context.Background(), req, scenario)
	if err != nil {
		t.Fatalf("unexpected scenario run error: %v", err)
	}

	if packet.Scenario != "submit-form-flow" {
		t.Errorf("expected scenario ID 'submit-form-flow', got %q", packet.Scenario)
	}
	if packet.Renderer.Name != "playwright-webkit" {
		t.Errorf("expected renderer name 'playwright-webkit', got %q", packet.Renderer.Name)
	}
	if len(packet.Elements) != 1 || packet.Elements[0].Role != "alert" {
		t.Errorf("unexpected elements in scenario packet: %+v", packet.Elements)
	}
	if runner.LastReq.Scenario == nil || len(runner.LastReq.Scenario.Actions) != 3 {
		t.Errorf("expected 3 actions passed in runner scenario request")
	}
}

func TestPlaywrightAdapter_Errors(t *testing.T) {
	runner := &playwright.MockRunner{
		Err: errors.New("worker process killed"),
	}
	adapter := playwright.New(playwright.Config{Runner: runner})

	_, err := adapter.Capture(context.Background(), playwright.TruthPathRequest{RunID: "err-1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	workerErrRunner := &playwright.MockRunner{
		Response: playwright.WorkerResponse{
			Success: false,
			Error:   "navigation timeout 30000ms exceeded",
		},
	}
	adapterErr := playwright.New(playwright.Config{Runner: workerErrRunner})
	_, err = adapterErr.Capture(context.Background(), playwright.TruthPathRequest{RunID: "err-2"})
	if err == nil {
		t.Fatal("expected worker error, got nil")
	}
}

func TestPlaywrightAdapter_CleanStateROIAndDiagnostics(t *testing.T) {
	mockResp := playwright.WorkerResponse{
		Success:       true,
		URL:           "http://localhost:3000/dashboard",
		ScreenshotB64: base64.StdEncoding.EncodeToString([]byte("roi-screenshot-bytes")),
		Accessibility: []evidence.AccessibilityNode{
			{ID: "nav", Role: "navigation", Name: "Main Nav"},
			{ID: "btn", Role: "button", Name: "Filter"},
		},
		Fonts: &evidence.FontEvidence{
			Status: "loaded",
			Faces: []evidence.FontFaceEvidence{
				{Family: "Roboto", Weight: "400", Status: "loaded"},
				{Family: "Roboto", Weight: "700", Status: "loaded"},
			},
			Total: 2,
		},
		Diagnostics: &evidence.DiagnosticsEvidence{
			Complete: true,
		},
		RuntimeIssues: []evidence.RuntimeIssue{
			{
				Code:     "CONSOLE_ERROR",
				Message:  "Uncaught TypeError: Cannot read property 'map' of undefined",
				Severity: evidence.SeverityHigh,
			},
			{
				Code:     "NETWORK_FAILURE",
				Message:  "Failed to load resource: the server responded with a status of 404 (Not Found)",
				Severity: evidence.SeverityMedium,
				Details: map[string]string{
					"url": "http://localhost:3000/api/missing",
				},
			},
		},
		Latency: evidence.RuntimeLatency{
			SnapshotMS:      12.0,
			PixelsMS:        20.0,
			AccessibilityMS: 8.0,
			FontsMS:         4.0,
			DiagnosticsMS:   3.0,
			TotalMS:         47.0,
		},
	}

	runner := &playwright.MockRunner{Response: mockResp}
	adapter := playwright.New(playwright.Config{
		DefaultBrowser: playwright.BrowserChromium,
		Runner:         runner,
	})

	roi := &evidence.Rect{
		X:      50,
		Y:      100,
		Width:  400,
		Height: 300,
	}

	req := playwright.TruthPathRequest{
		RunID:              "run-roi-clean",
		Browser:            playwright.BrowserChromium,
		URL:                "http://localhost:3000/dashboard",
		Region:             roi,
		CapturePixels:      true,
		CaptureARIA:        true,
		CaptureFonts:       true,
		CaptureDiagnostics: true,
		PauseAnimations:    true,
		FreezeClock:        true,
	}

	packet, err := adapter.Capture(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected capture error: %v", err)
	}

	// 1. Verify ROI bounds
	if packet.Pixels == nil {
		t.Fatalf("expected pixels evidence")
	}
	if packet.Pixels.Width != 400 || packet.Pixels.Height != 300 {
		t.Errorf("pixel bounds = %dx%d, want 400x300", packet.Pixels.Width, packet.Pixels.Height)
	}
	if len(packet.VisualRegions) != 1 || packet.VisualRegions[0].ID != "requested-roi" {
		t.Errorf("expected requested-roi in VisualRegions, got %+v", packet.VisualRegions)
	}

	// 2. Verify ARIA tree
	if len(packet.Accessibility) != 2 {
		t.Errorf("expected 2 accessibility nodes, got %d", len(packet.Accessibility))
	}

	// 3. Verify Font status
	if packet.Fonts == nil || packet.Fonts.Total != 2 {
		t.Errorf("expected 2 font faces, got %+v", packet.Fonts)
	}

	// 4. Verify Runtime issues (console errors and network failures)
	if len(packet.RuntimeIssues) != 2 {
		t.Fatalf("expected 2 runtime issues, got %d", len(packet.RuntimeIssues))
	}
	if packet.RuntimeIssues[0].Code != "CONSOLE_ERROR" {
		t.Errorf("issue 0 code = %s, want CONSOLE_ERROR", packet.RuntimeIssues[0].Code)
	}
	if packet.RuntimeIssues[1].Code != "NETWORK_FAILURE" {
		t.Errorf("issue 1 code = %s, want NETWORK_FAILURE", packet.RuntimeIssues[1].Code)
	}

	// 5. Verify Diagnostics
	if packet.Diagnostics == nil || !packet.Diagnostics.Complete {
		t.Errorf("expected complete diagnostics evidence")
	}

	// 6. Verify Deterministic Controls sent to runner
	if !runner.LastReq.PauseAnimations || !runner.LastReq.FreezeClock {
		t.Errorf("expected PauseAnimations and FreezeClock to be set in worker request")
	}
}

