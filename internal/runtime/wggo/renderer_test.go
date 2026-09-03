package wggo

import (
	"context"
	"image"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
)

func TestRenderReturnsRGBA(t *testing.T) {
	r := New(Config{})
	got, err := r.Render(context.Background(), fastrender.Request{
		HTML:   []byte(`<main class="hero"><h1>UiUxMaster</h1></main>`),
		CSS:    []byte(`html,body{margin:0}.hero{width:120px;height:80px;background:#fff}`),
		Width:  120,
		Height: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.RGBA == nil {
		t.Fatal("RGBA is nil")
	}
	if got.RGBA.Bounds().Dx() != 120 {
		t.Fatalf("width = %d, want 120", got.RGBA.Bounds().Dx())
	}
	if got.Renderer.BrowserAccurate {
		t.Fatal("WGGo must not claim browser accuracy")
	}
	if got.FidelityID == "" {
		t.Fatal("missing fidelity id")
	}
}

func TestCaptureRegionNormalizesBounds(t *testing.T) {
	r := New(Config{})
	got, err := r.CaptureRegion(context.Background(), fastrender.RegionRequest{
		Render: fastrender.Request{
			HTML:   []byte(`<div>crop</div>`),
			CSS:    []byte(`html,body{margin:0}div{width:100px;height:100px}`),
			Width:  100,
			Height: 100,
		},
		Clip: image.Rect(10, 20, 60, 70),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.RGBA == nil || got.RGBA.Bounds() != image.Rect(0, 0, 50, 50) {
		t.Fatalf("crop bounds = %v", got.RGBA.Bounds())
	}
}

func TestUnsupportedOperationsAreExplicit(t *testing.T) {
	r := New(Config{})
	if _, err := r.Inspect(context.Background(), fastrender.InspectRequest{}); err != fastrender.ErrUnsupported {
		t.Fatalf("Inspect error = %v, want ErrUnsupported", err)
	}
	if _, err := r.RunScenario(context.Background(), fastrender.Scenario{}); err != fastrender.ErrUnsupported {
		t.Fatalf("RunScenario error = %v, want ErrUnsupported", err)
	}
}

func TestComposeHTMLInjectsCSSIntoHead(t *testing.T) {
	got := composeHTML([]byte(`<html><head><title>x</title></head><body>x</body></html>`), []byte(`body{margin:0}`))
	wantNeedle := `<style data-uiuxmaster-injected>body{margin:0}</style></head>`
	if !contains(got, wantNeedle) {
		t.Fatalf("injected HTML missing %q: %s", wantNeedle, got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
