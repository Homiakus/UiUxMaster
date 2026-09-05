package visualdiff_test

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/visualdiff"
)

func TestFMEA012ComparatorRejectsBaselinePixelDigestMismatch(t *testing.T) {
	env := testEnvironment()
	key, err := env.CanonicalKey()
	if err != nil {
		t.Fatal(err)
	}
	baseline := image.NewRGBA(image.Rect(0, 0, 4, 4))
	candidate := image.NewRGBA(image.Rect(0, 0, 4, 4))
	cmp := visualdiff.NewComparator()
	_, err = cmp.CompareBaseline(visualdiff.ComparisonRequest{
		Baseline: visualdiff.BaselineReference{
			ID:             "tampered",
			Environment:    env,
			EnvironmentKey: key,
			DigestSHA256:   "deadbeef",
		},
		BaselineImage:        baseline,
		CandidateEnvironment: env,
		CandidateImage:       candidate,
	})
	if !errors.Is(err, visualdiff.ErrBaselineIntegrity) {
		t.Fatalf("err=%v want ErrBaselineIntegrity", err)
	}
}

func TestFMEA012BaselineChurnMetricsArePartitionedByEnvironmentKey(t *testing.T) {
	ctx := context.Background()
	store := visualdiff.NewMemoryBaselineStore()
	envA := testEnvironment()
	keyA, _ := envA.CanonicalKey()
	imgA := image.NewRGBA(image.Rect(0, 0, 4, 4))
	if err := store.Put(ctx, visualdiff.BaselineReference{ID: "home", Environment: envA, Protected: true}, imgA); err != nil {
		t.Fatal(err)
	}
	old, _, err := store.Get(ctx, "home")
	if err != nil {
		t.Fatal(err)
	}

	envB := envA
	envB.FixtureRevision = "fixture-b"
	keyB, _ := envB.CanonicalKey()
	imgB := image.NewRGBA(image.Rect(0, 0, 4, 4))
	imgB.Set(1, 1, color.RGBA{R: 255, A: 255})
	if _, err := store.Update(ctx, visualdiff.BaselineUpdateRequest{
		ID:              "home",
		ExpectedVersion: old.Version,
		ExpectedDigest:  old.DigestSHA256,
		Environment:     envB,
		Protected:       true,
		Reason:          "approved fixture revision",
		ReviewedBy:      "reviewer",
	}, imgB); err != nil {
		t.Fatal(err)
	}

	byEnv := store.MetricsByEnvironment()
	if got := byEnv[keyA]; got.Creates != 1 || got.Updates != 1 || got.EnvironmentChanges != 1 {
		t.Fatalf("env A metrics=%#v", got)
	}
	if got := byEnv[keyB]; got.Creates != 0 || got.Updates != 0 || got.EnvironmentChanges != 0 {
		t.Fatalf("env B must not inherit env A churn: %#v", got)
	}
}
