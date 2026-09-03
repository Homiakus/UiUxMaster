package wggo

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"strings"
	"time"

	webengine "github.com/go-webengine/engine"

	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
)

const pinnedRevision = "5393deafa2815c2e1a55cf7587c5f1a57a3626cf"

// Config controls the deliberately small L1 WGGo surface.
// JavaScript is disabled by default because the fastest calibrated path targets
// static/component HTML+CSS. JS-dependent UIs belong to the fidelity escalation
// path until separate benchmarks prove a safe useful envelope.
type Config struct {
	EnableJS bool
}

// Renderer adapts go-webengine/engine to the vendor-neutral FastRender contract.
type Renderer struct {
	engine *webengine.Engine
	config Config
}

func New(config Config) *Renderer {
	e := webengine.New()
	e.DisableJS = !config.EnableJS
	// Meta fallback fabricates content and is inappropriate for verification.
	e.MetaFallback = false
	return &Renderer{engine: e, config: config}
}

func (r *Renderer) Capabilities() fastrender.Capabilities {
	features := []string{
		"block_layout",
		"flex_layout",
		"grid_layout",
		"table_layout",
		"positioning",
		"css_variables",
		"media_width",
		"rgba",
	}
	if r.config.EnableJS {
		features = append(features, "javascript")
	}
	return fastrender.Capabilities{
		Name:             "wggo",
		Version:          pinnedRevision,
		BrowserAccurate:  false,
		SupportsPixels:   true,
		SupportsGeometry: false,
		SupportsStyles:   false,
		SupportsScenario: false,
		FeatureNames:     features,
	}
}

func (r *Renderer) Render(ctx context.Context, req fastrender.Request) (fastrender.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return fastrender.Evidence{}, err
	}
	if req.Width <= 0 || req.Height <= 0 {
		return fastrender.Evidence{}, fmt.Errorf("wggo: positive width and height are required")
	}

	html := composeHTML(req.HTML, req.CSS)
	baseURL := req.BaseURL
	if baseURL == "" {
		baseURL = "https://uiuxmaster.invalid/"
	}
	viewport := image.Rect(0, 0, req.Width, req.Height)

	start := time.Now()
	img, _, err := r.engine.RenderHTML(ctx, html, baseURL, viewport)
	elapsed := time.Since(start)
	if err != nil {
		return fastrender.Evidence{}, err
	}

	return fastrender.Evidence{
		RGBA:       img,
		Renderer:   r.Capabilities(),
		Latency:    fastrender.Latency{Total: elapsed},
		FidelityID: "wggo-static-candidate-v1",
	}, nil
}

func (r *Renderer) Inspect(context.Context, fastrender.InspectRequest) (fastrender.StructuralEvidence, error) {
	// go-webengine currently exposes the final RGBA through RenderHTML, while the
	// settled internal layout box is not part of the public stable API used here.
	// Do not fake structural evidence; browser/WGGo API evolution can add it later.
	return fastrender.StructuralEvidence{}, fastrender.ErrUnsupported
}

func (r *Renderer) CaptureRegion(ctx context.Context, req fastrender.RegionRequest) (fastrender.Evidence, error) {
	evidence, err := r.Render(ctx, req.Render)
	if err != nil {
		return fastrender.Evidence{}, err
	}
	if evidence.RGBA == nil {
		return fastrender.Evidence{}, fmt.Errorf("wggo: render returned nil RGBA")
	}

	clip := req.Clip.Intersect(evidence.RGBA.Bounds())
	if clip.Empty() {
		return fastrender.Evidence{}, fmt.Errorf("wggo: requested clip %v is outside image bounds %v", req.Clip, evidence.RGBA.Bounds())
	}
	cropped := image.NewRGBA(image.Rect(0, 0, clip.Dx(), clip.Dy()))
	draw.Draw(cropped, cropped.Bounds(), evidence.RGBA, clip.Min, draw.Src)
	evidence.RGBA = cropped
	return evidence, nil
}

func (r *Renderer) RunScenario(context.Context, fastrender.Scenario) (fastrender.ScenarioEvidence, error) {
	return fastrender.ScenarioEvidence{}, fastrender.ErrUnsupported
}

func composeHTML(htmlSrc, css []byte) string {
	html := string(htmlSrc)
	if len(css) == 0 {
		return html
	}
	style := "<style data-uiuxmaster-injected>" + string(css) + "</style>"
	lower := strings.ToLower(html)
	if idx := strings.Index(lower, "</head>"); idx >= 0 {
		return html[:idx] + style + html[idx:]
	}
	if idx := strings.Index(lower, "<html"); idx >= 0 {
		if end := strings.Index(html[idx:], ">"); end >= 0 {
			at := idx + end + 1
			return html[:at] + "<head>" + style + "</head>" + html[at:]
		}
	}
	return "<!doctype html><html><head>" + style + "</head><body>" + html + "</body></html>"
}
