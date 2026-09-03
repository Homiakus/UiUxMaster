package visualdiff

import (
	"image"
	"color"
	"testing"
)

func TestCompareRGBAFindsChangedROI(t *testing.T) {
	a := image.NewRGBA(image.Rect(0, 0, 4, 4))
	b := image.NewRGBA(image.Rect(0, 0, 4, 4))
	b.SetRGBA(1, 2, color.RGBA{R: 255, A: 255})
	b.SetRGBA(2, 2, color.RGBA{G: 128, A: 255})

	got, err := CompareRGBA(a, b, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got.ChangedPixels != 2 {
		t.Fatalf("changed pixels = %d, want 2", got.ChangedPixels)
	}
	if got.Bounds != image.Rect(1, 2, 3, 3) {
		t.Fatalf("bounds = %v, want %v", got.Bounds, image.Rect(1, 2, 3, 3))
	}
	if got.ChangeRatio != 0.125 {
		t.Fatalf("ratio = %v, want 0.125", got.ChangeRatio)
	}
	if got.MaxDelta != 255 {
		t.Fatalf("max delta = %d, want 255", got.MaxDelta)
	}
}

func TestCompareRGBAHonorsTolerance(t *testing.T) {
	a := image.NewRGBA(image.Rect(0, 0, 1, 1))
	b := image.NewRGBA(image.Rect(0, 0, 1, 1))
	a.SetRGBA(0, 0, color.RGBA{R: 100, A: 255})
	b.SetRGBA(0, 0, color.RGBA{R: 103, A: 255})

	got, err := CompareRGBA(a, b, Options{ChannelTolerance: 3})
	if err != nil {
		t.Fatal(err)
	}
	if got.ChangedPixels != 0 {
		t.Fatalf("changed pixels = %d, want 0", got.ChangedPixels)
	}
}

func TestCompareRGBASupportsNonZeroSourceBounds(t *testing.T) {
	baseA := image.NewRGBA(image.Rect(0, 0, 10, 10))
	baseB := image.NewRGBA(image.Rect(0, 0, 10, 10))
	a := baseA.SubImage(image.Rect(2, 3, 6, 7)).(*image.RGBA)
	b := baseB.SubImage(image.Rect(4, 1, 8, 5)).(*image.RGBA)
	baseB.SetRGBA(5, 2, color.RGBA{B: 200, A: 255})

	got, err := CompareRGBA(a, b, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got.ChangedPixels != 1 {
		t.Fatalf("changed pixels = %d, want 1", got.ChangedPixels)
	}
	if got.Bounds != image.Rect(1, 1, 2, 2) {
		t.Fatalf("normalized bounds = %v", got.Bounds)
	}
}

func TestCompareRGBARejectsDifferentDimensions(t *testing.T) {
	_, err := CompareRGBA(
		image.NewRGBA(image.Rect(0, 0, 2, 2)),
		image.NewRGBA(image.Rect(0, 0, 3, 2)),
		Options{},
	)
	if err == nil {
		t.Fatal("expected dimension mismatch error")
	}
}
