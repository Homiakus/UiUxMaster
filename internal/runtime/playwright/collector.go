package playwright

import (
	"context"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
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
