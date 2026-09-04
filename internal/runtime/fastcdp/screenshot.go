package fastcdp

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"time"
)

// CaptureRegionOptions requests a Blink screenshot for only the required
// document-coordinate region. CDP itself returns base64-encoded image bytes, so
// this path decodes entirely in memory and never touches the filesystem.
type CaptureRegionOptions struct {
	X                     float64
	Y                     float64
	Width                 float64
	Height                float64
	Scale                 float64
	CaptureBeyondViewport bool
	OptimizeForSpeed      bool
}

type CaptureStats struct {
	EncodedBytes int
	Base64Decode time.Duration
	ImageDecode  time.Duration
	Total        time.Duration
}

type captureScreenshotParams struct {
	Format                string       `json:"format"`
	FromSurface           bool         `json:"fromSurface"`
	CaptureBeyondViewport bool         `json:"captureBeyondViewport,omitempty"`
	OptimizeForSpeed      bool         `json:"optimizeForSpeed,omitempty"`
	Clip                  viewportClip `json:"clip"`
}

type viewportClip struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Scale  float64 `json:"scale"`
}

type captureScreenshotResult struct {
	Data string `json:"data"`
}

func (o CaptureRegionOptions) validate() error {
	if o.Width <= 0 || o.Height <= 0 {
		return fmt.Errorf("fastcdp: screenshot width and height must be positive")
	}
	if o.Scale < 0 {
		return fmt.Errorf("fastcdp: screenshot scale cannot be negative")
	}
	return nil
}

// CaptureRegionRGBA returns only the requested ROI as RGBA. Unlike WGGo, CDP
// does not expose a raw pixel buffer for screenshots, so one in-memory PNG
// decode is unavoidable here. Keeping the clip small bounds both protocol bytes
// and decode cost.
func (c *Connection) CaptureRegionRGBA(ctx context.Context, sessionID string, options CaptureRegionOptions) (*image.RGBA, CaptureStats, error) {
	started := time.Now()
	if err := options.validate(); err != nil {
		return nil, CaptureStats{}, err
	}
	if options.Scale == 0 {
		options.Scale = 1
	}

	params := captureScreenshotParams{
		Format:                "png",
		FromSurface:           true,
		CaptureBeyondViewport: options.CaptureBeyondViewport,
		OptimizeForSpeed:      options.OptimizeForSpeed,
		Clip: viewportClip{
			X:      options.X,
			Y:      options.Y,
			Width:  options.Width,
			Height: options.Height,
			Scale:  options.Scale,
		},
	}
	var result captureScreenshotResult
	if err := c.Call(ctx, sessionID, "Page.captureScreenshot", params, &result); err != nil {
		return nil, CaptureStats{}, err
	}
	if result.Data == "" {
		return nil, CaptureStats{}, fmt.Errorf("fastcdp: Page.captureScreenshot returned empty data")
	}

	decodeStart := time.Now()
	encoded, err := base64.StdEncoding.DecodeString(result.Data)
	base64Duration := time.Since(decodeStart)
	if err != nil {
		return nil, CaptureStats{}, fmt.Errorf("fastcdp: decode screenshot base64: %w", err)
	}

	imageStart := time.Now()
	decoded, err := png.Decode(bytes.NewReader(encoded))
	imageDuration := time.Since(imageStart)
	if err != nil {
		return nil, CaptureStats{}, fmt.Errorf("fastcdp: decode screenshot PNG: %w", err)
	}

	rgba := toRGBA(decoded)
	total := time.Since(started)
	if total <= 0 {
		total = time.Nanosecond
	}
	return rgba, CaptureStats{
		EncodedBytes: len(encoded),
		Base64Decode: base64Duration,
		ImageDecode:  imageDuration,
		Total:        total,
	}, nil
}

func toRGBA(src image.Image) *image.RGBA {
	if rgba, ok := src.(*image.RGBA); ok && rgba.Bounds().Min == (image.Point{}) {
		return rgba
	}
	bounds := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, out.Bounds(), src, bounds.Min, draw.Src)
	return out
}
