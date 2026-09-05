package visualdiff

import (
	"fmt"
	"image"
)

// Options controls deterministic per-channel tolerance. A pixel is considered
// changed when any RGBA channel differs by more than ChannelTolerance.
type Options struct {
	ChannelTolerance uint8
}

// Result summarizes an in-memory comparison without allocating a full diff image.
type Result struct {
	Width         int
	Height        int
	Pixels        int
	ComparedPixels int
	MaskedPixels  int
	ChangedPixels int
	ChangeRatio   float64
	Bounds        image.Rectangle
	MaxDelta      uint8
}

// CompareRGBA is a low-level primitive for images whose compatibility has already
// been established by the caller. Protected baseline callers must use
// Comparator.CompareBaseline so environment and mask ownership are gated first.
func CompareRGBA(a, b *image.RGBA, opts Options) (Result, error) {
	return compareRGBA(a, b, opts, nil)
}

func compareRGBA(a, b *image.RGBA, opts Options, masks []image.Rectangle) (Result, error) {
	if a == nil || b == nil {
		return Result{}, fmt.Errorf("visualdiff: nil RGBA input")
	}
	if a.Bounds().Dx() != b.Bounds().Dx() || a.Bounds().Dy() != b.Bounds().Dy() {
		return Result{}, fmt.Errorf("visualdiff: image dimensions differ: %v vs %v", a.Bounds(), b.Bounds())
	}

	width, height := a.Bounds().Dx(), a.Bounds().Dy()
	result := Result{Width: width, Height: height, Pixels: width * height}
	if width == 0 || height == 0 {
		return result, nil
	}

	minX, minY := width, height
	maxX, maxY := -1, -1
	for y := 0; y < height; y++ {
		ay := a.Rect.Min.Y + y
		by := b.Rect.Min.Y + y
		for x := 0; x < width; x++ {
			if pointMasked(x, y, masks) {
				result.MaskedPixels++
				continue
			}
			result.ComparedPixels++
			ax := a.Rect.Min.X + x
			bx := b.Rect.Min.X + x
			aoff := a.PixOffset(ax, ay)
			boff := b.PixOffset(bx, by)

			changed := false
			for c := 0; c < 4; c++ {
				d := absDiff(a.Pix[aoff+c], b.Pix[boff+c])
				if d > result.MaxDelta {
					result.MaxDelta = d
				}
				if d > opts.ChannelTolerance {
					changed = true
				}
			}
			if !changed {
				continue
			}

			result.ChangedPixels++
			if x < minX { minX = x }
			if y < minY { minY = y }
			if x > maxX { maxX = x }
			if y > maxY { maxY = y }
		}
	}

	if result.ChangedPixels > 0 {
		result.Bounds = image.Rect(minX, minY, maxX+1, maxY+1)
		denom := result.ComparedPixels
		if denom > 0 {
			result.ChangeRatio = float64(result.ChangedPixels) / float64(denom)
		}
	}
	return result, nil
}

func pointMasked(x, y int, masks []image.Rectangle) bool {
	p := image.Pt(x, y)
	for _, mask := range masks {
		if p.In(mask) {
			return true
		}
	}
	return false
}

func absDiff(a, b uint8) uint8 {
	if a >= b { return a - b }
	return b - a
}
