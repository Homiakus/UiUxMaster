package playwright

import (
	"context"
	"fmt"
	"runtime"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
)

// PlaywrightCollector adapts a TruthPathAdapter for dispatcher L3 collection.
type PlaywrightCollector struct {
	adapter TruthPathAdapter
	browser BrowserFamily
}

// NewCollector returns an initialized PlaywrightCollector.
func NewCollector(adapter TruthPathAdapter, browser BrowserFamily) *PlaywrightCollector {
	if browser == "" {
		browser = BrowserChromium
	}
	return &PlaywrightCollector{
		adapter: adapter,
		browser: browser,
	}
}

func (p *PlaywrightCollector) CalibrationEnvironment(ctx context.Context) (fidelity.CalibrationEnvironment, error) {
	if p == nil || p.adapter == nil {
		return fidelity.CalibrationEnvironment{}, fmt.Errorf("playwright: calibration identity requires adapter")
	}
	caps := p.adapter.Capabilities()
	if !caps.Ready {
		if _, err := p.adapter.Probe(ctx); err != nil {
			return fidelity.CalibrationEnvironment{}, err
		}
		caps = p.adapter.Capabilities()
	}
	browserVersion := caps.BrowserVersions[p.browser]
	if browserVersion == "" {
		return fidelity.CalibrationEnvironment{}, fmt.Errorf("playwright: browser %s has no attested calibration version", p.browser)
	}
	env := fidelity.CalibrationEnvironment{
		RendererName:    "playwright-" + string(p.browser),
		RendererVersion: runtimeIdentity(caps.WorkerVersion, caps.PlaywrightVersion, browserVersion),
		FidelityID:      "truthpath:" + runtimeIdentity(caps.WorkerVersion, caps.PlaywrightVersion, browserVersion),
		BrowserFamily:   string(p.browser),
		BrowserVersion:  browserVersion,
		WorkerVersion:   caps.WorkerVersion,
		RuntimeVersion:  caps.PlaywrightVersion,
		Platform:        runtime.GOOS + "/" + runtime.GOARCH,
	}
	if err := env.Validate(); err != nil {
		return fidelity.CalibrationEnvironment{}, err
	}
	return env, nil
}

// CollectL3 satisfies dispatcher.L3Collector.
func (p *PlaywrightCollector) CollectL3(ctx context.Context, req engine.ValidationRequest, plan evidenceplan.Plan) (evidence.Packet, error) {
	vp := evidence.Viewport{Width: 1280, Height: 720, Browser: string(p.browser)}
	if len(req.Themes) > 0 {
		vp.ColorScheme = req.Themes[0]
	}

	var reg *evidence.Rect
	if plan.Region != nil {
		reg = &evidence.Rect{
			X:      plan.Region.X,
			Y:      plan.Region.Y,
			Width:  plan.Region.Width,
			Height: plan.Region.Height,
		}
	} else if req.Region != nil {
		reg = &evidence.Rect{
			X:      req.Region.X,
			Y:      req.Region.Y,
			Width:  req.Region.Width,
			Height: req.Region.Height,
		}
	}

	url := req.BaseURL
	if len(req.TargetRoutes) > 0 {
		url = req.TargetRoutes[0]
	} else if len(req.Scope.Routes) > 0 {
		url = req.Scope.Routes[0]
	}

	tpReq := TruthPathRequest{
		RunID:              req.RunID,
		Browser:            p.browser,
		URL:                url,
		HTML:               req.HTML,
		CSS:                req.CSS,
		BaseURL:            req.BaseURL,
		Viewport:           vp,
		Region:             reg,
		CapturePixels:      plan.Pixels,
		CaptureARIA:        plan.Accessibility,
		CaptureFonts:       plan.Fonts,
		CaptureDiagnostics: plan.Diagnostics,
		CaptureLayout:      plan.Structural,
	}

	return p.adapter.Capture(ctx, tpReq)
}
