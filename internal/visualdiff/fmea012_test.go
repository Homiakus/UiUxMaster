package visualdiff_test

import (
	"context"
	"errors"
	"image"
	"image/color"
	"runtime"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/visualdiff"
)

func testEnvironment() evidence.RenderEnvironmentIdentity {
	return evidence.RenderEnvironmentIdentity{
		SchemaVersion: evidence.RenderEnvironmentSchemaVersion,
		RendererName: "wggo",
		RendererVersion: "renderer-v1",
		WorkerVersion: "in-process:renderer-v1",
		BrowserFamily: "synthetic",
		BrowserEngine: "wggo",
		BrowserVersion: "renderer-v1",
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
		ViewportWidth: 1280,
		ViewportHeight: 800,
		DeviceScale: 1,
		Theme: "light",
		FontSetDigest: "sha256:fonts-v1",
		Locale: "en-US",
		Timezone: "UTC",
		FixtureRevision: "fixture-a",
	}
}

func TestFMEA012EnvironmentKeyRejectsMaterialMismatch(t *testing.T) {
	base := testEnvironment()
	baseKey, err := base.CanonicalKey()
	if err != nil { t.Fatal(err) }
	mutations := []struct{
		name string
		mutate func(*evidence.RenderEnvironmentIdentity)
	}{
		{"browser-version", func(e *evidence.RenderEnvironmentIdentity) { e.BrowserVersion = "renderer-v2" }},
		{"font-set", func(e *evidence.RenderEnvironmentIdentity) { e.FontSetDigest = "sha256:fonts-v2" }},
		{"dpr", func(e *evidence.RenderEnvironmentIdentity) { e.DeviceScale = 2 }},
		{"theme", func(e *evidence.RenderEnvironmentIdentity) { e.Theme = "dark" }},
		{"fixture", func(e *evidence.RenderEnvironmentIdentity) { e.FixtureRevision = "fixture-b" }},
		{"locale", func(e *evidence.RenderEnvironmentIdentity) { e.Locale = "de-DE" }},
		{"timezone", func(e *evidence.RenderEnvironmentIdentity) { e.Timezone = "Europe/Berlin" }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			candidate := base
			tc.mutate(&candidate)
			key, err := candidate.CanonicalKey()
			if err != nil { t.Fatal(err) }
			if key == baseKey { t.Fatalf("material mutation %s did not change key", tc.name) }
		})
	}
}

func TestFMEA012ComparatorRejectsIncompatibleBeforeTolerance(t *testing.T) {
	imgA := image.NewRGBA(image.Rect(0, 0, 8, 8))
	imgB := image.NewRGBA(image.Rect(0, 0, 8, 8))
	base := testEnvironment()
	baseKey, _ := base.CanonicalKey()
	candidate := base
	candidate.FontSetDigest = "sha256:different-fonts"

	cmp := visualdiff.NewComparator()
	_, err := cmp.CompareBaseline(visualdiff.ComparisonRequest{
		Baseline: visualdiff.BaselineReference{ID: "home", Environment: base, EnvironmentKey: baseKey},
		BaselineImage: imgA,
		CandidateEnvironment: candidate,
		CandidateImage: imgB,
		Options: visualdiff.Options{ChannelTolerance: 255},
	})
	if !errors.Is(err, visualdiff.ErrBaselineIncompatible) {
		t.Fatalf("err=%v want ErrBaselineIncompatible", err)
	}
	m := cmp.Metrics(baseKey)
	if m.IncompatibleComparisons != 1 || m.Comparisons != 0 {
		t.Fatalf("metrics=%#v", m)
	}
}

func TestFMEA012ExactKeyComparisonIsDeterministic(t *testing.T) {
	imgA := image.NewRGBA(image.Rect(0, 0, 8, 8))
	imgB := image.NewRGBA(image.Rect(0, 0, 8, 8))
	imgB.Set(2, 3, color.RGBA{R: 200, A: 255})
	env := testEnvironment()
	key, _ := env.CanonicalKey()
	cmp := visualdiff.NewComparator()
	req := visualdiff.ComparisonRequest{
		Baseline: visualdiff.BaselineReference{ID: "home", Environment: env, EnvironmentKey: key},
		BaselineImage: imgA,
		CandidateEnvironment: env,
		CandidateImage: imgB,
		Options: visualdiff.Options{ChannelTolerance: 0},
	}
	first, err := cmp.CompareBaseline(req)
	if err != nil { t.Fatal(err) }
	second, err := cmp.CompareBaseline(req)
	if err != nil { t.Fatal(err) }
	if first.ComparisonDigest != second.ComparisonDigest || first.PixelResult != second.PixelResult {
		t.Fatalf("non-deterministic result: first=%#v second=%#v", first, second)
	}
	m := cmp.Metrics(key)
	if m.Comparisons != 2 || m.OutcomeFlips != 0 {
		t.Fatalf("metrics=%#v", m)
	}
}

func TestFMEA012ReviewedBaselineUpdateRecordsOldNewIdentity(t *testing.T) {
	ctx := context.Background()
	store := visualdiff.NewMemoryBaselineStore()
	oldImg := image.NewRGBA(image.Rect(0, 0, 4, 4))
	env := testEnvironment()
	if err := store.Put(ctx, visualdiff.BaselineReference{ID: "protected", Environment: env, Protected: true}, oldImg); err != nil {
		t.Fatal(err)
	}
	oldRef, _, err := store.Get(ctx, "protected")
	if err != nil { t.Fatal(err) }

	newImg := image.NewRGBA(image.Rect(0, 0, 4, 4))
	newImg.Set(1, 1, color.RGBA{G: 255, A: 255})
	newEnv := env
	newEnv.FixtureRevision = "fixture-b"
	updated, err := store.Update(ctx, visualdiff.BaselineUpdateRequest{
		ID: "protected",
		ExpectedVersion: oldRef.Version,
		ExpectedDigest: oldRef.DigestSHA256,
		Environment: newEnv,
		Protected: true,
		Reason: "approved intentional redesign",
		ReviewedBy: "reviewer-1",
	}, newImg)
	if err != nil { t.Fatal(err) }
	if updated.Version != oldRef.Version+1 || updated.DigestSHA256 == oldRef.DigestSHA256 {
		t.Fatalf("updated=%#v old=%#v", updated, oldRef)
	}
	history, err := store.History(ctx, "protected")
	if err != nil { t.Fatal(err) }
	if len(history) != 1 { t.Fatalf("history=%d want 1", len(history)) }
	r := history[0]
	if r.OldDigest != oldRef.DigestSHA256 || r.NewDigest != updated.DigestSHA256 || r.OldEnvironmentKey == r.NewEnvironmentKey || r.Reason == "" || r.ReviewedBy == "" {
		t.Fatalf("audit record=%#v", r)
	}
	m := store.Metrics(ctx)
	if m.Updates != 1 || m.EnvironmentChanges != 1 {
		t.Fatalf("metrics=%#v", m)
	}
}

func TestFMEA012DynamicMaskCannotEscapeOwner(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 20, 20))
	candidate := image.NewRGBA(image.Rect(0, 0, 20, 20))
	candidate.Set(15, 15, color.RGBA{R: 255, A: 255})
	env := testEnvironment()
	key, _ := env.CanonicalKey()
	cmp := visualdiff.NewComparator()
	_, err := cmp.CompareBaseline(visualdiff.ComparisonRequest{
		Baseline: visualdiff.BaselineReference{ID: "mask", Environment: env, EnvironmentKey: key},
		BaselineImage: base,
		CandidateEnvironment: env,
		CandidateImage: candidate,
		Elements: []evidence.ElementRef{{ID: "clock", Visible: true, Bounds: evidence.Rect{X: 0, Y: 0, Width: 10, Height: 10}}},
		Masks: []evidence.DynamicMask{{ID: "clock-dynamic", OwnerElementID: "clock", Bounds: evidence.Rect{X: 0, Y: 0, Width: 20, Height: 20}}},
	})
	if !errors.Is(err, visualdiff.ErrMaskOwnership) {
		t.Fatalf("err=%v want ErrMaskOwnership", err)
	}
}

func TestFMEA012OwnedMaskOnlyExcludesOwnedPixels(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 20, 20))
	candidate := image.NewRGBA(image.Rect(0, 0, 20, 20))
	candidate.Set(2, 2, color.RGBA{R: 255, A: 255}) // owned dynamic pixel
	candidate.Set(15, 15, color.RGBA{G: 255, A: 255}) // real regression outside owner
	env := testEnvironment()
	key, _ := env.CanonicalKey()
	cmp := visualdiff.NewComparator()
	result, err := cmp.CompareBaseline(visualdiff.ComparisonRequest{
		Baseline: visualdiff.BaselineReference{ID: "mask", Environment: env, EnvironmentKey: key},
		BaselineImage: base,
		CandidateEnvironment: env,
		CandidateImage: candidate,
		Elements: []evidence.ElementRef{{ID: "clock", Visible: true, Bounds: evidence.Rect{X: 0, Y: 0, Width: 10, Height: 10}}},
		Masks: []evidence.DynamicMask{{ID: "clock-dynamic", OwnerElementID: "clock", Bounds: evidence.Rect{X: 0, Y: 0, Width: 10, Height: 10}}},
	})
	if err != nil { t.Fatal(err) }
	if result.PixelResult.ChangedPixels != 1 || result.PixelResult.MaskedPixels != 100 {
		t.Fatalf("pixel result=%#v", result.PixelResult)
	}
}
