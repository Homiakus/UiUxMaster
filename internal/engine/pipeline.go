package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

// PipelineTelemetry provides comprehensive latency accounting across all pipeline stages.
type PipelineTelemetry struct {
	ImpactMS       float64 `json:"impact_ms"`
	InvalidationMS float64 `json:"invalidation_ms"`
	FidelityScanMS float64 `json:"fidelity_scan_ms"`
	RouteMS        float64 `json:"route_ms"`
	CollectMS      float64 `json:"collect_ms"`
	VerifyMS       float64 `json:"verify_ms"`
	SynthesisMS    float64 `json:"synthesis_ms"`
	TotalMS        float64 `json:"total_ms"`
	Tier           string  `json:"tier"`
	ScopeSize      int     `json:"scope_size"`
}

// PipelineResult encapsulates the complete execution trace from change to decision.
type PipelineResult struct {
	Request      ValidationRequest            `json:"request"`
	Scope        invalidation.ValidationScope `json:"scope"`
	Assessment   fidelity.Assessment          `json:"assessment"`
	Plan         ValidationPlan               `json:"plan"`
	Packet       evidence.Packet              `json:"packet"`
	Verification verifier.Result              `json:"verification"`
	Report       Report                       `json:"report"`
	Telemetry    PipelineTelemetry            `json:"telemetry"`
}

// Pipeline orchestrates the canonical validation path:
// changed source/CSS token → ImpactSet → bounded ValidationScope →
// fidelity route → WGGo or FastCDP → evidence.Packet → verifier → engine decision.
type Pipeline struct {
	Resolver     *impact.Resolver
	Policy       *invalidation.Policy
	Collector    Collector
	VerPolicy    verifier.Policy
	Capabilities fastrender.Capabilities
}

// Execute runs the complete canonical pipeline while recording end-to-end telemetry.
func (p *Pipeline) Execute(ctx context.Context, req ValidationRequest) (PipelineResult, error) {
	if err := ctx.Err(); err != nil {
		return PipelineResult{}, err
	}
	if p.Collector == nil {
		return PipelineResult{}, fmt.Errorf("pipeline: collector is required")
	}

	startTotal := time.Now()

	// 1. Impact Resolution & Validation Scope
	startImpact := time.Now()
	req, scope, err := PlanScope(ctx, req, p.Resolver, p.Policy)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("pipeline: plan scope: %w", err)
	}
	elapsedImpact := float64(time.Since(startImpact).Microseconds()) / 1000.0

	// 2. Fidelity Risk Assessment
	startFidelity := time.Now()
	features := fidelity.ScanSourceFeatures(fidelity.SourceInput{
		HTML:                        req.HTML,
		CSS:                         req.CSS,
		DynamicDependencyUnresolved: scope.Widened,
	})
	caps := p.Capabilities
	if caps.Name == "" {
		if capProvider, ok := p.Collector.(interface{ Capabilities() fastrender.Capabilities }); ok {
			caps = capProvider.Capabilities()
		}
	}
	fidelityCaps := fidelity.RendererCapabilities{
		Name:            caps.Name,
		BrowserAccurate: caps.BrowserAccurate,
		Supported:       make(map[fidelity.Feature]bool),
	}
	for _, fn := range caps.FeatureNames {
		fidelityCaps.Supported[fidelity.Feature(fn)] = true
	}
	assessment := fidelity.Assess(features, fidelityCaps)
	elapsedFidelity := float64(time.Since(startFidelity).Microseconds()) / 1000.0

	// 3. Converged Validation Plan (RouteDecision + EvidencePlan)
	startRoute := time.Now()
	plan := PlanValidationRoute(req, assessment, caps)
	elapsedRoute := float64(time.Since(startRoute).Microseconds()) / 1000.0

	// 4. Runtime Dispatch & Evidence Collection (L0 / L1 / L2 / L3)
	startCollect := time.Now()
	packet, err := p.Collector.Collect(ctx, req, plan)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("pipeline: collect: %w", err)
	}
	if err := ValidateCollectedEvidence(plan, packet); err != nil {
		return PipelineResult{}, fmt.Errorf("pipeline: evidence attestation: %w", err)
	}
	elapsedCollect := float64(time.Since(startCollect).Microseconds()) / 1000.0

	// 5. Deterministic Verifier
	startVerify := time.Now()
	vPolicy := p.VerPolicy
	if vPolicy.MinTargetWidth == 0 {
		vPolicy = verifier.DefaultPolicy()
	}
	vResult := verifier.Apply(&packet, vPolicy)
	elapsedVerify := float64(time.Since(startVerify).Microseconds()) / 1000.0

	// 6. Engine Decision & Report
	startSynthesis := time.Now()
	report := EvaluateForPlan(packet, plan.EvidencePlan)
	elapsedSynthesis := float64(time.Since(startSynthesis).Microseconds()) / 1000.0

	totalMS := float64(time.Since(startTotal).Microseconds()) / 1000.0

	// Populate canonical packet end-to-end telemetry
	packet.Latency.ImpactMS = elapsedImpact
	packet.Latency.InvalidationMS = elapsedImpact
	packet.Latency.FidelityScanMS = elapsedFidelity
	packet.Latency.RouteMS = elapsedRoute
	if plan.Route.Tier == TierFastRender {
		packet.Latency.FastRenderMS = elapsedCollect
	}
	packet.Latency.VerifyMS = elapsedVerify
	packet.Latency.SynthesisMS = elapsedSynthesis
	packet.Latency.TotalMS = totalMS

	scopeSize := len(scope.Components) + len(scope.Routes) + len(scope.Regions)

	telemetry := PipelineTelemetry{
		ImpactMS:       elapsedImpact,
		InvalidationMS: elapsedImpact,
		FidelityScanMS: elapsedFidelity,
		RouteMS:        elapsedRoute,
		CollectMS:      elapsedCollect,
		VerifyMS:       elapsedVerify,
		SynthesisMS:    elapsedSynthesis,
		TotalMS:        totalMS,
		Tier:           string(plan.Route.Tier),
		ScopeSize:      scopeSize,
	}

	return PipelineResult{
		Request:      req,
		Scope:        scope,
		Assessment:   assessment,
		Plan:         plan,
		Packet:       packet,
		Verification: vResult,
		Report:       report,
		Telemetry:    telemetry,
	}, nil
}
