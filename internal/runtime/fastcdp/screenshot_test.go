package fastcdp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"
)

func TestCaptureRegionRGBASendsClipAndDecodesInMemory(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	type outcome struct {
		img   *image.RGBA
		stats CaptureStats
		err   error
	}
	done := make(chan outcome, 1)
	go func() {
		img, stats, err := conn.CaptureRegionRGBA(ctx, "session-7", CaptureRegionOptions{
			X: 12, Y: 34, Width: 2, Height: 1,
			CaptureBeyondViewport: true,
			OptimizeForSpeed:      true,
		})
		done <- outcome{img: img, stats: stats, err: err}
	}()

	request := decodeWire(t, <-transport.writes)
	if request.Method != "Page.captureScreenshot" || request.SessionID != "session-7" {
		t.Fatalf("unexpected request: %#v", request)
	}
	var params captureScreenshotParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		t.Fatal(err)
	}
	if params.Format != "png" || !params.FromSurface || !params.CaptureBeyondViewport || !params.OptimizeForSpeed {
		t.Fatalf("unexpected params: %#v", params)
	}
	if params.Clip != (viewportClip{X: 12, Y: 34, Width: 2, Height: 1, Scale: 1}) {
		t.Fatalf("clip = %#v", params.Clip)
	}

	fixture := image.NewRGBA(image.Rect(0, 0, 2, 1))
	fixture.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	fixture.SetRGBA(1, 0, color.RGBA{G: 128, B: 64, A: 255})
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, fixture); err != nil {
		t.Fatal(err)
	}
	result := captureScreenshotResult{Data: base64.StdEncoding.EncodeToString(pngBytes.Bytes())}
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: mustRawJSON(t, result)})

	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.img == nil || got.img.Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatalf("image = %#v", got.img)
	}
	if pixel := got.img.RGBAAt(0, 0); pixel.R != 255 || pixel.A != 255 {
		t.Fatalf("pixel 0 = %#v", pixel)
	}
	if pixel := got.img.RGBAAt(1, 0); pixel.G != 128 || pixel.B != 64 {
		t.Fatalf("pixel 1 = %#v", pixel)
	}
	if got.stats.EncodedBytes != pngBytes.Len() || got.stats.Total <= 0 {
		t.Fatalf("stats = %#v", got.stats)
	}
}

func TestCaptureRegionRGBARejectsInvalidDimensionsWithoutCDPCall(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	_, _, err := conn.CaptureRegionRGBA(context.Background(), "", CaptureRegionOptions{Width: 0, Height: 10})
	if err == nil {
		t.Fatal("expected invalid dimension error")
	}
	select {
	case payload := <-transport.writes:
		t.Fatalf("unexpected CDP write: %s", payload)
	default:
	}
}

func TestCaptureRegionRGBARejectsEmptyScreenshotData(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	done := make(chan error, 1)
	go func() {
		_, _, err := conn.CaptureRegionRGBA(context.Background(), "", CaptureRegionOptions{Width: 1, Height: 1})
		done <- err
	}()
	request := decodeWire(t, <-transport.writes)
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: json.RawMessage(`{"data":""}`)})
	if err := <-done; err == nil {
		t.Fatal("expected empty screenshot error")
	}
}
