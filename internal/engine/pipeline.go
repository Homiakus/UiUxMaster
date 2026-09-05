package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

// PipelineTelemetry provides non-overlapping canonical stage timings plus
// provenance counters for the two scope-planning stages. MeasuredStageMS is the
// sum of the listed sequential stages; UnattributedMS is orchestration overhead
// and is never assigned to a named stage, preventing double counting.
type PipelineTelemetry struct {
	ImpactMS       float64 `json:"impact_ms"`
	InvalidationMS float64 `json:"invalidation_ms"`
	FidelityScanMS float64 `json:"fidelity_scan_ms"`
	RouteMS        float64 `json:"route_ms"`
	CollectMS      float64 `json:"collect_ms"`
	VerifyMS       float64 `json:"verify_ms"`
	SynthesisMS    float64 `json:"synthesis_ms"`
	MeasuredStageMS float64 `json:"measured_stage_ms"`
	UnattributedMS  float64 `json:"unattributed_ms"`
	TotalMS         float64 `json:"total_ms"`

	ImpactNodes        int `json:"impact_nodes"`
	ImpactUnknown      int `json:"impact_unknown"`
	ScopeComponents    int `json:"scope_components"`
	ScopeRoutes        int `json:"scope_routes"`
	ScopeRegions       int `json:"scope_regions"`
	ScopeViewports     int `json:"scope_viewports"`
	ScopeThemes        int `json:"scope_themes"`
	ScopeSize          int `json:"scope_size"`
	Tier               string `json:"tier"`
}

// PipelineResult encapsulates the complete execution trace from change to decision.
type PipelineResult struct {
	Request       ValidationRequest            `json:"request"`
	Scope         invalidation.ValidationScope `json:"scope"`
	Assessment    fidelity.Assessment          `json:"assessment"`
	Plan          ValidationPlan               `json:"plan"`
	Packet        evidence.Packet              `json:"packet"`
	Verification  verifier.Result              `json:"verification"`
	Report        Report                       `json:"report"`
	PassAuthority PassAuthority                `json:"pass_authority"`
	Telemetry     PipelineTelemetry            `json:"telemetry"`
}

// Pipeline orchestrates the canonical validation path:
// changed source/CSS token → ImpactSet → bounded ValidationScope →
// fidelity route → runtime evidence → attestation → verifier → engine decision.
type Pipeline struct {
	Resolver          *impact.Resolver
	Policy            *invalidation.Policy
	ImpactStage       ImpactStageFunc
	InvalidationStage InvalidationStageFunc
	Collector         Collector
	VerPolicy         verifier.Policy
	Capabilities      fastrender.Capabilities
	CalibrationMatrix *fidelity.CalibrationMatrix
	Calibration       *fidelity.CalibrationAuthority
}

func (p *Pipeline) Execute(ctx context.Context, req ValidationRequest) (PipelineResult, error) {
	if err := ctx.Err(); err != nil {
		return PipelineResult{}, err
	}
	if p.Collector == nil {
		return PipelineResult{}, fmt.Errorf("pipeline: collector is required")
	}

	startTotal := time.Now()

	// 1a. Impact resolution: source/change graph -> ImpactSet only.
	startImpact := time.Now()
	req, impactSet, err := p.resolveImpactStage(ctx, req)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("pipeline: resolve impact: %w", err)
	}
	elapsedImpact := pipelineDurationMS(time.Since(startImpact))

	// 1b. Invalidation: resolved ImpactSet -> bounded ValidationScope only.
	startInvalidation := time.Now()
	req, scope, err := p.invalidateStage(ctx, req, impactSet)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("pipeline: invalidate impact: %w", err)
	}
	elapsedInvalidation := pipelineDurationMS(time.Since(startInvalidation))

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
	elapsedFidelity := pipelineDurationMS(time.Since(startFidelity))

	// 3. Converged validation route/evidence plan.
	startRoute := time.Now()
	plan := PlanValidationRoute(req, assessment, caps)
	elapsedRoute := pipelineDurationMS(time.Since(startRoute))

	// 4. Runtime dispatch and evidence collection.
	startCollect := time.Now()
	packet, err := p.Collector.Collect(ctx, req, plan)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("pipeline: collect: %w", err)
	}
	if err := ValidateCollectedEvidence(plan, packet); err != nil {
		return PipelineResult{}, fmt.Errorf("pipeline: evidence attestation: %w", err)
	}
	if err := ValidateRevisionAttestation(req, packet); err != nil {
		return PipelineResult{}, fmt.Errorf("pipeline: revision attestation: %w", err)
	}
	elapsedCollect := pipelineDurationMS(time.Since(startCollect))

	// 5. Deterministic verifier. No verifier sees evidence before tier/revision
	// provenance has passed fail-closed guards.
	startVerify := time.Now()
	vPolicy := p.VerPolicy
	if vPolicy.MinTargetWidth == 0 {
		vPolicy = verifier.DefaultPolicy()
	}
	vResult := verifier.Apply(&packet, vPolicy)
	elapsedVerify := pipelineDurationMS(time.Since(startVerify))

	// 6. Synthesis + legal PASS authority.
	startSynthesis := time.Now()
	report := EvaluateForPlan(packet, plan.EvidencePlan)
	var calibrationProvider CalibrationContextProvider
	if provider, ok := p.Collector.(CalibrationContextProvider); ok {
		calibrationProvider = provider
	}
	passAuthority := EvaluatePassAuthority(ctx, req, plan, packet, p.CalibrationMatrix, p.Calibration, calibrationProvider)
	if passAuthority.Required && !passAuthority.Allowed && report.BlockingFindings == 0 && report.HighFindings == 0 && len(report.MissingEvidence) == 0 {
		report.MissingEvidence = append(report.MissingEvidence, "valid runtime calibration for legal PASS")
		sort.Strings(report.MissingEvidence)
		if passAuthority.RequiredEscalation != "" {
			report.RecommendedNext = fmt.Sprintf("recalibrate exact runtime parity or escalate to %s before PASS", passAuthority.RequiredEscalation)
		} else {
			report.RecommendedNext = "recalibrate exact runtime parity before PASS"
		}
	}
	elapsedSynthesis := pipelineDurationMS(time.Since(startSynthesis))

	totalMS := pipelineDurationMS(time.Since(startTotal))
	measuredStageMS := elapsedImpact + elapsedInvalidation + elapsedFidelity + elapsedRoute + elapsedCollect + elapsedVerify + elapsedSynthesis
	unattributedMS := totalMS - measuredStageMS
	if unattributedMS < 0 {
		// Independent intervals are sequential and should fit inside TotalMS. A
		// tiny negative value can only be floating-point conversion noise; never
		// push it into a named stage or double-count it.
		unattributedMS = 0
	}

	packet.Latency.ImpactMS = elapsedImpact
	packet.Latency.InvalidationMS = elapsedInvalidation
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
		ImpactMS:          elapsedImpact,
		InvalidationMS:    elapsedInvalidation,
		FidelityScanMS:    elapsedFidelity,
		RouteMS:           elapsedRoute,
		CollectMS:         elapsedCollect,
		VerifyMS:          elapsedVerify,
		SynthesisMS:       elapsedSynthesis,
		MeasuredStageMS:   measuredStageMS,
		UnattributedMS:    unattributedMS,
		TotalMS:           totalMS,
		ImpactNodes:       len(impactSet.NodeIDs),
		ImpactUnknown:     len(impactSet.UnknownIDs),
		ScopeComponents:   len(scope.Components),
		ScopeRoutes:       len(scope.Routes),
		ScopeRegions:      len(scope.Regions),
		ScopeViewports:    len(scope.Viewports),
		ScopeThemes:       len(scope.Themes),
		ScopeSize:         scopeSize,
		Tier:              string(plan.Route.Tier),
	}

	return PipelineResult{
		Request:       req,
		Scope:         scope,
		Assessment:    assessment,
		Plan:          plan,
		Packet:        packet,
		Verification:  vResult,
		Report:        report,
		PassAuthority: passAuthority,
		Telemetry:     telemetry,
	}, nil
}

func pipelineDurationMS(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
}
