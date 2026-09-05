package engine

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type fmea008Collector struct {
	ctx fidelity.CalibrationContext
}

func (c *fmea008Collector) Collect(_ context.Context, req ValidationRequest, _ ValidationPlan) (evidence.Packet, error) {
	return evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp", Version: "Chrome/140.0.1", FidelityID: "blink-l2"},
		Viewport: evidence.Viewport{Width: 1280, Height: 800, DeviceScale: 1, Browser: "Chrome/140.0.1"},
		Elements: []evidence.ElementRef{{ID: "main", Tag: "main", Role: "main", Visible: true, Bounds: evidence.Rect{Width: 1000, Height: 700}}},
		Diagnostics: &evidence.DiagnosticsEvidence{Complete: true},
	}, nil
}

func (c *fmea008Collector) CalibrationContext(_ context.Context, _ ValidationRequest, _ ValidationPlan, _ evidence.Packet) (fidelity.CalibrationContext, error) {
	return c.ctx, nil
}

func engineFMEA008Context() fidelity.CalibrationContext {
	return fidelity.CalibrationContext{
		Approx: fidelity.CalibrationEnvironment{
			RendererName: "fastcdp", RendererVersion: "Chrome/140.0.1", FidelityID: "blink-l2",
			BrowserFamily: "chromium", BrowserVersion: "Chrome/140.0.1", RuntimeVersion: "1.3",
			Platform: "linux/amd64", ViewportWidth: 1280, ViewportHeight: 800, DeviceScale: 1,
		},
		Truth: fidelity.CalibrationEnvironment{
			RendererName: "playwright-chromium", RendererVersion: "worker=1.0.0;playwright=1.62.1;browser=Chrome/140.0.1",
			FidelityID: "truthpath:worker=1.0.0;playwright=1.62.1;browser=Chrome/140.0.1",
			BrowserFamily: "chromium", BrowserVersion: "Chrome/140.0.1", WorkerVersion: "1.0.0", RuntimeVersion: "1.62.1",
			Platform: "linux/amd64", ViewportWidth: 1280, ViewportHeight: 800, DeviceScale: 1,
		},
	}
}

func putEngineFMEA008Record(t *testing.T, registry *fidelity.CalibrationRegistry, ctx fidelity.CalibrationContext, now time.Time) {
	t.Helper()
	key, err := ctx.Key()
	if err != nil { t.Fatal(err) }
	if err := registry.Put(fidelity.CalibrationRecord{
		Class: fidelity.EvidenceClassStaticLayout, Tier: fidelity.TierL2, Context: ctx, EnvironmentKey: key,
		CorpusDigest: "sha256:engine-fmea008", ArtifactRef: "artifacts/calibration/engine-fmea008.json",
		Samples: 100, PassedSamples: 100, CreatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(24*time.Hour),
	}); err != nil { t.Fatal(err) }
}

func TestFMEA008PipelineMissingCalibrationIsEvidenceInsufficiencyNotPass(t *testing.T) {
	ctx := engineFMEA008Context()
	pipeline := &Pipeline{
		Collector: &fmea008Collector{ctx: ctx},
		VerPolicy: verifier.DefaultPolicy(),
		Calibration: fidelity.NewStrictCalibrationAuthority(),
	}
	res, err := pipeline.Execute(context.Background(), ValidationRequest{
		RunID: "fmea008-missing", Intent: evidenceplan.IntentQuickStructural, RequireLegalPass: true,
	})
	if err != nil { t.Fatal(err) }
	if res.PassAuthority.Allowed {
		t.Fatalf("missing calibration granted PASS: %#v", res.PassAuthority)
	}
	if res.PassAuthority.RequiredEscalation != fidelity.TierL3 {
		t.Fatalf("escalation = %q, want L3", res.PassAuthority.RequiredEscalation)
	}
	if !containsMissing(res.Report.MissingEvidence, "valid runtime calibration for legal PASS") {
		t.Fatalf("missing evidence did not expose calibration insufficiency: %#v", res.Report)
	}
}

func TestFMEA008PipelineSameValidatedKeyAuthorizesPass(t *testing.T) {
	now := time.Date(2026, 9, 5, 6, 0, 0, 0, time.UTC)
	ctx := engineFMEA008Context()
	registry := fidelity.NewCalibrationRegistry()
	putEngineFMEA008Record(t, registry, ctx, now)
	authority := fidelity.NewCalibrationAuthority(registry, fidelity.DefaultCalibrationPolicy())
	authority.Now = func() time.Time { return now }
	pipeline := &Pipeline{
		Collector: &fmea008Collector{ctx: ctx}, VerPolicy: verifier.DefaultPolicy(), Calibration: authority,
	}
	res, err := pipeline.Execute(context.Background(), ValidationRequest{
		RunID: "fmea008-valid", Intent: evidenceplan.IntentQuickStructural, RequireLegalPass: true,
	})
	if err != nil { t.Fatal(err) }
	if !res.PassAuthority.Allowed || len(res.PassAuthority.CalibrationKeys) != 1 {
		t.Fatalf("valid exact calibration did not authorize PASS: %#v", res.PassAuthority)
	}
	if containsMissing(res.Report.MissingEvidence, "valid runtime calibration for legal PASS") {
		t.Fatalf("valid calibration still reported missing: %#v", res.Report)
	}
}

func TestFMEA008PipelineVersionDriftRevokesPreviouslyLegalPass(t *testing.T) {
	now := time.Date(2026, 9, 5, 6, 0, 0, 0, time.UTC)
	original := engineFMEA008Context()
	registry := fidelity.NewCalibrationRegistry()
	putEngineFMEA008Record(t, registry, original, now)
	authority := fidelity.NewCalibrationAuthority(registry, fidelity.DefaultCalibrationPolicy())
	authority.Now = func() time.Time { return now }

	drifted := original
	drifted.Truth.WorkerVersion = "1.0.1"
	pipeline := &Pipeline{Collector: &fmea008Collector{ctx: drifted}, VerPolicy: verifier.DefaultPolicy(), Calibration: authority}
	res, err := pipeline.Execute(context.Background(), ValidationRequest{
		RunID: "fmea008-drift", Intent: evidenceplan.IntentQuickStructural, RequireLegalPass: true,
	})
	if err != nil { t.Fatal(err) }
	if res.PassAuthority.Allowed {
		t.Fatalf("version drift retained legal PASS: %#v", res.PassAuthority)
	}
	if !containsReason(res.PassAuthority.Reasons, "calibration_environment_mismatch_for_static_layout") {
		t.Fatalf("drift reason missing: %#v", res.PassAuthority)
	}
}

func TestFMEA008L3TruthPathDoesNotDependOnApproximateCalibration(t *testing.T) {
	plan := ValidationPlan{Need: EvidenceNeed{Geometry: true}, EvidencePlan: evidenceplan.Plan{Structural: true}}
	req := ValidationRequest{RequireLegalPass: true}
	packet := evidence.Packet{Renderer: evidence.RendererRef{Tier: "L3", Name: "playwright-chromium", Version: "attested"}}
	result := EvaluatePassAuthority(context.Background(), req, plan, packet, nil, nil, nil)
	if !result.Allowed || result.Tier != fidelity.TierL3 {
		t.Fatalf("L3 should remain authoritative without approximate calibration: %#v", result)
	}
}

func containsMissing(values []string, want string) bool {
	for _, value := range values { if value == want { return true } }
	return false
}

func containsReason(values []string, want string) bool {
	for _, value := range values { if value == want { return true } }
	return false
}
