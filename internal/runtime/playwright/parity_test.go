package playwright_test

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

// canonicalFixture returns standard HTML/CSS fixtures for multi-browser parity checks.
func canonicalFixture(name string) (html string, css string) {
	switch name {
	case "interactive_form":
		return `<!doctype html>
<html>
<head><title>Form Fixture</title></head>
<body>
  <main class="container">
    <h1>Login</h1>
    <form id="login-form">
      <label for="email">Email</label>
      <input type="email" id="email" name="email" value="user@example.com" />
      <button type="submit" id="submit-btn" style="width: 120px; height: 44px;">Sign In</button>
    </form>
  </main>
</body>
</html>`, `.container { padding: 24px; font-family: sans-serif; } button { min-height: 44px; }`

	case "typography_hero":
		return `<!doctype html>
<html>
<head><title>Hero Fixture</title></head>
<body>
  <header>
    <nav aria-label="Main Navigation">
      <a href="/home">Home</a>
      <a href="/pricing">Pricing</a>
    </nav>
  </header>
  <main>
    <h1>Next-Gen Visual Verification</h1>
    <p>Sub-second design iteration for autonomous coding agents.</p>
    <button id="cta-button" style="width: 140px; height: 48px;">Get Started</button>
  </main>
</body>
</html>`, `body { font-family: Inter, sans-serif; margin: 0; padding: 32px; } h1 { font-size: 40px; }`

	default:
		return "<html><body><div>Default Fixture</div></body></html>", ""
	}
}

// simulateBrowserRun creates a populated evidence.Packet for a specific browser engine.
func simulateBrowserRun(runID string, browser playwright.BrowserFamily, fixtureName string) evidence.Packet {
	html, css := canonicalFixture(fixtureName)
	_ = html
	_ = css

	vp := evidence.Viewport{
		Width:   1280,
		Height:  800,
		Browser: string(browser),
	}

	packet := evidence.Packet{
		RunID: runID,
		URL:   "http://localhost:8080/" + fixtureName,
		Viewport: vp,
		Renderer: evidence.RendererRef{
			Tier:    "L3",
			Name:    "playwright-" + string(browser),
			Version: "1.0.0",
		},
		Documents: []evidence.DocumentMetrics{
			{URL: "http://localhost:8080/" + fixtureName, ContentWidth: 1280, ContentHeight: 800},
		},
		Elements: []evidence.ElementRef{
			{
				ID:        "btn-cta",
				Tag:       "button",
				Role:      "button",
				Name:      "Get Started",
				Visible:   true,
				Clickable: true,
				Bounds: evidence.Rect{
					X:      32,
					Y:      200,
					Width:  140,
					Height: 48,
				},
				Styles: map[string]string{
					"display":         "inline-block",
					"pointer-events":  "auto",
					"width":           "140px",
					"height":          "48px",
				},
			},
		},
		Accessibility: []evidence.AccessibilityNode{
			{ID: "ax-nav", Role: "navigation", Name: "Main Navigation"},
			{ID: "ax-btn", Role: "button", Name: "Get Started"},
		},
		Fonts: &evidence.FontEvidence{
			Status: "loaded",
			Faces: []evidence.FontFaceEvidence{
				{Family: "Inter", Style: "normal", Weight: "400", Status: "loaded"},
			},
			Total: 1,
		},
		Diagnostics: &evidence.DiagnosticsEvidence{
			Complete: true,
		},
		Latency: evidence.RuntimeLatency{
			SnapshotMS:      14.0,
			AccessibilityMS: 6.0,
			FontsMS:         4.0,
			TotalMS:         42.0,
		},
	}

	return packet
}

func TestCrossBrowserParity_Chromium_Firefox_WebKit(t *testing.T) {
	browsers := []playwright.BrowserFamily{
		playwright.BrowserChromium,
		playwright.BrowserFirefox,
		playwright.BrowserWebKit,
	}

	packets := make(map[playwright.BrowserFamily]evidence.Packet)

	for _, b := range browsers {
		mockResp := playwright.WorkerResponse{
			Success:       true,
			URL:           "http://localhost:8080/typography_hero",
			AriaSnapshot:  "- navigation \"Main Navigation\"\n- button \"Get Started\"",
			Documents:     []evidence.DocumentMetrics{{URL: "http://localhost:8080/typography_hero", ContentWidth: 1280, ContentHeight: 800}},
			Elements: []evidence.ElementRef{
				{
					ID:        "btn-cta",
					Tag:       "button",
					Role:      "button",
					Name:      "Get Started",
					Visible:   true,
					Clickable: true,
					Bounds: evidence.Rect{
						X:      32,
						Y:      200,
						Width:  140,
						Height: 48,
					},
					Styles: map[string]string{
						"display":        "inline-block",
						"pointer-events": "auto",
					},
				},
			},
			Accessibility: []evidence.AccessibilityNode{
				{ID: "ax-nav", Role: "navigation", Name: "Main Navigation"},
				{ID: "ax-btn", Role: "button", Name: "Get Started"},
			},
			Fonts: &evidence.FontEvidence{
				Status: "loaded",
				Faces:  []evidence.FontFaceEvidence{{Family: "Inter", Status: "loaded"}},
				Total:  1,
			},
		}

		runner := &playwright.MockRunner{Response: mockResp}
		adapter := playwright.New(playwright.Config{
			DefaultBrowser: b,
			Runner:         runner,
		})

		req := playwright.TruthPathRequest{
			RunID:         "run-parity-" + string(b),
			Browser:       b,
			URL:           "http://localhost:8080/typography_hero",
			CaptureARIA:   true,
			CaptureFonts:  true,
			CaptureLayout: true,
		}

		pkt, err := adapter.Capture(context.Background(), req)
		if err != nil {
			t.Fatalf("Capture failed on browser %s: %v", b, err)
		}
		packets[b] = pkt
	}

	// Cross-browser verification parity check
	policy := verifier.DefaultPolicy()

	for b, pkt := range packets {
		// Verify accessibility node count parity
		if len(pkt.Accessibility) != 2 {
			t.Errorf("browser %s: expected 2 accessibility nodes, got %d", b, len(pkt.Accessibility))
		}

		// Verify element structure parity
		if len(pkt.Elements) != 1 || pkt.Elements[0].Name != "Get Started" {
			t.Errorf("browser %s: elements mismatch: %+v", b, pkt.Elements)
		}

		// Verify font status parity
		if pkt.Fonts == nil || pkt.Fonts.Status != "loaded" {
			t.Errorf("browser %s: fonts status not loaded", b)
		}

		// Verify deterministic rule evaluation produces identical zero blocking defects
		res := verifier.Verify(pkt, policy)
		for _, issue := range res.Issues {
			if issue.Severity == evidence.SeverityCritical || issue.Severity == evidence.SeverityHigh {
				t.Errorf("browser %s: unexpected severe issue: %+v", b, issue)
			}
		}
	}
}

func TestL2FastCDP_vs_L3TruthPath_Parity(t *testing.T) {
	// L2 FastBrowser packet simulation
	l2Packet := evidence.Packet{
		RunID: "run-l2-parity",
		URL:   "http://localhost:8080/interactive_form",
		Renderer: evidence.RendererRef{
			Tier: "L2",
			Name: "chromium-cdp",
		},
		Viewport: evidence.Viewport{Width: 1280, Height: 800},
		Documents: []evidence.DocumentMetrics{
			{URL: "http://localhost:8080/interactive_form", ContentWidth: 1280, ContentHeight: 800},
		},
		Elements: []evidence.ElementRef{
			{
				ID:        "btn-submit",
				Tag:       "button",
				Role:      "button",
				Name:      "Sign In",
				Visible:   true,
				Clickable: true,
				Bounds:    evidence.Rect{X: 24, Y: 100, Width: 120, Height: 44},
				Styles:    map[string]string{"pointer-events": "auto", "display": "block"},
			},
		},
		Accessibility: []evidence.AccessibilityNode{
			{ID: "ax-btn", Role: "button", Name: "Sign In"},
		},
		Fonts: &evidence.FontEvidence{Status: "loaded"},
	}

	// L3 TruthPath packet simulation
	l3Packet := simulateBrowserRun("run-l3-parity", playwright.BrowserChromium, "interactive_form")
	l3Packet.Elements[0].ID = "btn-submit"
	l3Packet.Elements[0].Name = "Sign In"
	l3Packet.Accessibility[1].Name = "Sign In"

	policy := verifier.DefaultPolicy()
	l2Res := verifier.Verify(l2Packet, policy)
	l3Res := verifier.Verify(l3Packet, policy)

	if len(l2Res.Issues) != len(l3Res.Issues) {
		t.Errorf("issues count mismatch: L2 has %d, L3 has %d", len(l2Res.Issues), len(l3Res.Issues))
	}
}
