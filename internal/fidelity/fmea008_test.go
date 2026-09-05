package fidelity

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fmea008Context() CalibrationContext {
	return CalibrationContext{
		Approx: CalibrationEnvironment{
			RendererName: "fastcdp", RendererVersion: "Chrome/140.0.1", FidelityID: "blink-l2",
			BrowserFamily: "chromium", BrowserVersion: "Chrome/140.0.1", RuntimeVersion: "1.3",
			Platform: "linux/amd64", ViewportWidth: 1280, ViewportHeight: 800, DeviceScale: 1,
		},
		Truth: CalibrationEnvironment{
			RendererName: "playwright-chromium", RendererVersion: "worker=1.0.0;playwright=1.62.1;browser=Chrome/140.0.1",
			FidelityID: "truthpath:worker=1.0.0;playwright=1.62.1;browser=Chrome/140.0.1",
			BrowserFamily: "chromium", BrowserVersion: "Chrome/140.0.1", WorkerVersion: "1.0.0", RuntimeVersion: "1.62.1",
			Platform: "linux/amd64", ViewportWidth: 1280, ViewportHeight: 800, DeviceScale: 1,
		},
	}
}

func fmea008Record(t *testing.T, ctx CalibrationContext, created time.Time) CalibrationRecord {
	t.Helper()
	key, err := ctx.Key()
	if err != nil {
		t.Fatal(err)
	}
	return CalibrationRecord{
		Class: EvidenceClassStaticLayout, Tier: TierL2, Context: ctx, EnvironmentKey: key,
		CorpusDigest: "sha256:parity-corpus-v1", ArtifactRef: "artifacts/calibration/static-layout-l2.json",
		Samples: 100, PassedSamples: 100, CreatedAt: created, ExpiresAt: created.Add(7 * 24 * time.Hour),
	}
}

func TestFMEA008SameValidatedKeyRetainsLegalPass(t *testing.T) {
	now := time.Date(2026, 9, 5, 6, 0, 0, 0, time.UTC)
	ctx := fmea008Context()
	registry := NewCalibrationRegistry()
	if err := registry.Put(fmea008Record(t, ctx, now.Add(-time.Hour))); err != nil {
		t.Fatal(err)
	}
	authority := NewCalibrationAuthority(registry, DefaultCalibrationPolicy())
	authority.Now = func() time.Time { return now }
	record, err := authority.Validate(EvidenceClassStaticLayout, TierL2, ctx)
	if err != nil {
		t.Fatalf("valid exact key rejected: %v", err)
	}
	if record.EnvironmentKey == "" || record.ArtifactRef == "" || record.CorpusDigest == "" {
		t.Fatalf("record lost audit identity: %#v", record)
	}
}

func TestFMEA008VersionMutationInvalidatesPreviouslyLegalPass(t *testing.T) {
	now := time.Date(2026, 9, 5, 6, 0, 0, 0, time.UTC)
	original := fmea008Context()
	registry := NewCalibrationRegistry()
	if err := registry.Put(fmea008Record(t, original, now.Add(-time.Hour))); err != nil {
		t.Fatal(err)
	}
	authority := NewCalibrationAuthority(registry, DefaultCalibrationPolicy())
	authority.Now = func() time.Time { return now }

	mutations := []struct {
		name string
		edit func(*CalibrationContext)
	}{
		{"approx renderer", func(c *CalibrationContext) { c.Approx.RendererVersion = "Chrome/141.0.0" }},
		{"approx browser", func(c *CalibrationContext) { c.Approx.BrowserVersion = "Chrome/141.0.0" }},
		{"truth browser", func(c *CalibrationContext) { c.Truth.BrowserVersion = "Chrome/141.0.0" }},
		{"truth worker", func(c *CalibrationContext) { c.Truth.WorkerVersion = "1.0.1" }},
		{"truth playwright", func(c *CalibrationContext) { c.Truth.RuntimeVersion = "1.63.0" }},
		{"platform", func(c *CalibrationContext) { c.Approx.Platform = "linux/arm64" }},
		{"viewport", func(c *CalibrationContext) { c.Approx.ViewportWidth = 1440 }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			changed := original
			tc.edit(&changed)
			_, err := authority.Validate(EvidenceClassStaticLayout, TierL2, changed)
			if !errors.Is(err, ErrCalibrationEnvironmentMismatch) {
				t.Fatalf("mutation must invalidate calibration, got %v", err)
			}
		})
	}
}

func TestFMEA008ExpiredMissingAndWeakCorpusFailClosed(t *testing.T) {
	now := time.Date(2026, 9, 5, 6, 0, 0, 0, time.UTC)
	ctx := fmea008Context()

	t.Run("missing", func(t *testing.T) {
		a := NewCalibrationAuthority(NewCalibrationRegistry(), DefaultCalibrationPolicy())
		a.Now = func() time.Time { return now }
		_, err := a.Validate(EvidenceClassStaticLayout, TierL2, ctx)
		if !errors.Is(err, ErrCalibrationMissing) {
			t.Fatalf("missing err = %v", err)
		}
	})

	t.Run("expired", func(t *testing.T) {
		r := fmea008Record(t, ctx, now.Add(-10*24*time.Hour))
		r.ExpiresAt = now.Add(-time.Minute)
		registry := NewCalibrationRegistry()
		if err := registry.Put(r); err != nil { t.Fatal(err) }
		a := NewCalibrationAuthority(registry, DefaultCalibrationPolicy())
		a.Now = func() time.Time { return now }
		_, err := a.Validate(EvidenceClassStaticLayout, TierL2, ctx)
		if !errors.Is(err, ErrCalibrationExpired) { t.Fatalf("expired err = %v", err) }
	})

	t.Run("coverage", func(t *testing.T) {
		r := fmea008Record(t, ctx, now.Add(-time.Hour)); r.Samples = 5; r.PassedSamples = 5
		registry := NewCalibrationRegistry(); if err := registry.Put(r); err != nil { t.Fatal(err) }
		a := NewCalibrationAuthority(registry, DefaultCalibrationPolicy()); a.Now = func() time.Time { return now }
		_, err := a.Validate(EvidenceClassStaticLayout, TierL2, ctx)
		if !errors.Is(err, ErrCalibrationCoverage) { t.Fatalf("coverage err = %v", err) }
	})

	t.Run("quality", func(t *testing.T) {
		r := fmea008Record(t, ctx, now.Add(-time.Hour)); r.PassedSamples = 95
		registry := NewCalibrationRegistry(); if err := registry.Put(r); err != nil { t.Fatal(err) }
		a := NewCalibrationAuthority(registry, DefaultCalibrationPolicy()); a.Now = func() time.Time { return now }
		_, err := a.Validate(EvidenceClassStaticLayout, TierL2, ctx)
		if !errors.Is(err, ErrCalibrationQuality) { t.Fatalf("quality err = %v", err) }
	})
}

func TestFMEA008CalibrationSnapshotPersistsExactKeyAndArtifact(t *testing.T) {
	now := time.Date(2026, 9, 5, 6, 0, 0, 0, time.UTC)
	ctx := fmea008Context()
	registry := NewCalibrationRegistry()
	record := fmea008Record(t, ctx, now.Add(-time.Hour))
	if err := registry.Put(record); err != nil { t.Fatal(err) }

	path := filepath.Join(t.TempDir(), "calibration.json")
	if err := registry.SaveFile(path); err != nil { t.Fatal(err) }
	if info, err := os.Stat(path); err != nil || info.Size() == 0 { t.Fatalf("snapshot not written: info=%v err=%v", info, err) }

	restored, err := LoadCalibrationRegistry(path)
	if err != nil { t.Fatal(err) }
	got, ok := restored.latest(EvidenceClassStaticLayout, TierL2)
	if !ok { t.Fatal("restored record missing") }
	if got.EnvironmentKey != record.EnvironmentKey || got.CorpusDigest != record.CorpusDigest || got.ArtifactRef != record.ArtifactRef {
		t.Fatalf("durable calibration identity changed: got=%#v want=%#v", got, record)
	}
}
