package visualdiff_test

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/visualdiff"
)

func TestBaselineStore_PutGetList(t *testing.T) {
	store := visualdiff.NewMemoryBaselineStore()
	ctx := context.Background()

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	img.Set(10, 10, color.RGBA{R: 255, A: 255})

	ref := visualdiff.BaselineReference{
		ID:          "home-desktop",
		Scenario:    "landing",
		Environment: testEnvironment(),
	}

	err := store.Put(ctx, ref, img)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	gotRef, gotImg, err := store.Get(ctx, "home-desktop")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if gotRef.ID != "home-desktop" {
		t.Errorf("ref ID = %s, want home-desktop", gotRef.ID)
	}
	if gotRef.DigestSHA256 == "" {
		t.Errorf("expected non-empty DigestSHA256")
	}
	if gotRef.EnvironmentKey == "" {
		t.Errorf("expected non-empty EnvironmentKey")
	}
	if gotRef.Viewport.Width != 1280 || gotRef.Viewport.Height != 800 {
		t.Errorf("viewport=%#v", gotRef.Viewport)
	}
	if gotImg == nil || gotImg.Bounds().Dx() != 100 {
		t.Errorf("invalid image retrieved")
	}

	list, err := store.List(ctx)
	if err != nil || len(list) != 1 {
		t.Errorf("expected 1 item in list, got %d (err: %v)", len(list), err)
	}
}

func TestBaselineStore_ProtectedBaselineRejectsOverwrite(t *testing.T) {
	store := visualdiff.NewMemoryBaselineStore()
	ctx := context.Background()

	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	ref := visualdiff.BaselineReference{
		ID:          "protected-golden",
		Protected:   true,
		Environment: testEnvironment(),
	}

	if err := store.Put(ctx, ref, img); err != nil {
		t.Fatalf("initial put failed: %v", err)
	}

	// Attempt overwrite through the create-only path.
	err := store.Put(ctx, ref, img)
	if err == nil {
		t.Fatal("expected error overwriting protected baseline, got nil")
	}
	if !errors.Is(err, visualdiff.ErrProtectedBaseline) {
		t.Fatalf("expected ErrProtectedBaseline, got %v", err)
	}
}

func TestLocalizeDifferences_ClustersAndIntersectsDOM(t *testing.T) {
	imgA := image.NewRGBA(image.Rect(0, 0, 200, 200))
	imgB := image.NewRGBA(image.Rect(0, 0, 200, 200))

	// Introduce two distinct changed clusters
	// Cluster 1: near top-left [10,10] to [30,30] (CTA button area)
	for y := 10; y < 30; y++ {
		for x := 10; x < 30; x++ {
			imgB.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	// Cluster 2: near bottom-right [150,150] to [170,170] (Footer area)
	for y := 150; y < 170; y++ {
		for x := 150; x < 170; x++ {
			imgB.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	elements := []evidence.ElementRef{
		{
			ID:      "btn-cta",
			Tag:     "button",
			Visible: true,
			Bounds: evidence.Rect{
				X:      5,
				Y:      5,
				Width:  40,
				Height: 40,
			},
		},
		{
			ID:      "footer-link",
			Tag:     "a",
			Visible: true,
			Bounds: evidence.Rect{
				X:      140,
				Y:      140,
				Width:  50,
				Height: 40,
			},
		},
		{
			ID:      "unrelated-sidebar",
			Tag:     "aside",
			Visible: true,
			Bounds: evidence.Rect{
				X:      80,
				Y:      80,
				Width:  30,
				Height: 30,
			},
		},
	}

	opts := visualdiff.DefaultLocalizationOptions()
	regions, findings, err := visualdiff.LocalizeDifferences(imgA, imgB, elements, opts)
	if err != nil {
		t.Fatalf("LocalizeDifferences failed: %v", err)
	}

	if len(regions) < 2 {
		t.Fatalf("expected at least 2 clustered regions, got %d", len(regions))
	}
	if len(findings) < 2 {
		t.Fatalf("expected at least 2 visual findings, got %d", len(findings))
	}

	// Verify element correlation
	ctaMatched := false
	footerMatched := false
	sidebarMatched := false

	for _, reg := range regions {
		for _, elID := range reg.ElementIDs {
			if elID == "btn-cta" { ctaMatched = true }
			if elID == "footer-link" { footerMatched = true }
			if elID == "unrelated-sidebar" { sidebarMatched = true }
		}
	}

	if !ctaMatched { t.Errorf("expected btn-cta to be correlated with cluster 1") }
	if !footerMatched { t.Errorf("expected footer-link to be correlated with cluster 2") }
	if sidebarMatched { t.Errorf("unrelated-sidebar should not match any diff cluster") }
}
