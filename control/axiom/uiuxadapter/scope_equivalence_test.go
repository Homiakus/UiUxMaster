package uiuxadapter

import (
	"context"
	"reflect"
	"testing"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type scopeCaptureCollector struct {
	req  engine.ValidationRequest
	plan engine.ValidationPlan
}

func (c *scopeCaptureCollector) Collect(_ context.Context, req engine.ValidationRequest, plan engine.ValidationPlan) (evidence.Packet, error) {
	c.req = req
	c.plan = plan
	return evidence.Packet{
		Renderer:    evidence.RendererRef{Tier: string(plan.Route.Tier)},
		Elements:    []evidence.ElementRef{{ID: "root", Tag: "main", Visible: true}},
		Diagnostics: &evidence.DiagnosticsEvidence{Complete: true},
	}, nil
}

func canonicalScopeChange() controlplane.Change {
	return controlplane.Change{
		RunID:         "run:fmea-002",
		ProjectID:     "project:fixture",
		SourceDigest:  "sha256:0123456789abcdef",
		ChangedFiles:  []string{"src/Button.tsx", "src/theme.css"},
		ChangedTokens: []string{"global"},
		ChangedNodes:  []string{"component:button"},
		TargetRoutes:  []string{"route:/settings"},
		Viewports:     []string{"mobile"},
		Themes:        []string{"dark"},
		BaseURL:       "http://127.0.0.1:4173",
		Intent:        string(evidenceplan.IntentFullDeterministic),
		Need: controlplane.ValidationNeed{
			Geometry: true,
			Styles:   true,
		},
	}
}

func directScopeRequest(change controlplane.Change) engine.ValidationRequest {
	return engine.ValidationRequest{
		RunID:         change.RunID,
		ProjectID:     change.ProjectID,
		SourceDigest:  change.SourceDigest,
		ChangedFiles:  append([]string(nil), change.ChangedFiles...),
		ChangedTokens: append([]string(nil), change.ChangedTokens...),
		ChangedNodes:  append([]string(nil), change.ChangedNodes...),
		TargetRoutes:  append([]string(nil), change.TargetRoutes...),
		Viewports:     append([]string(nil), change.Viewports...),
		Themes:        append([]string(nil), change.Themes...),
		BaseURL:       change.BaseURL,
		Intent:        evidenceplan.Intent(change.Intent),
		Need: engine.EvidenceNeed{
			Geometry: change.Need.Geometry,
			Styles:   change.Need.Styles,
			Pixels:   change.Need.Pixels,
			Scenario: change.Need.Scenario,
			CleanState: change.Need.CleanState,
		},
	}
}

func TestPipelineAdapterPreservesCanonicalScopeAndRoute(t *testing.T) {
	ctx := context.Background()
	change := canonicalScopeChange()

	directCollector := &scopeCaptureCollector{}
	directPipeline := &engine.Pipeline{
		Collector: directCollector,
		VerPolicy: verifier.DefaultPolicy(),
	}
	directResult, err := directPipeline.Execute(ctx, directScopeRequest(change))
	if err != nil {
		t.Fatalf("direct Execute failed: %v", err)
	}

	axiomCollector := &scopeCaptureCollector{}
	axiomPipeline := &engine.Pipeline{
		Collector: axiomCollector,
		VerPolicy: verifier.DefaultPolicy(),
	}
	adapter := NewPipelineAdapter(axiomPipeline)
	advisory, err := adapter.PlanEvidence(ctx, change)
	if err != nil {
		t.Fatalf("PlanEvidence failed: %v", err)
	}
	if _, err := adapter.CollectVerify(ctx, change, advisory); err != nil {
		t.Fatalf("CollectVerify failed: %v", err)
	}

	if !reflect.DeepEqual(directResult.Scope, axiomCollector.req.Scope) {
		t.Fatalf("scope mismatch\ndirect=%+v\naxiom=%+v", directResult.Scope, axiomCollector.req.Scope)
	}
	if directResult.Plan.Route.Tier != axiomCollector.plan.Route.Tier {
		t.Fatalf("tier mismatch: direct=%q axiom=%q", directResult.Plan.Route.Tier, axiomCollector.plan.Route.Tier)
	}
	if !reflect.DeepEqual(directResult.Plan.EvidencePlan, axiomCollector.plan.EvidencePlan) {
		t.Fatalf("evidence-plan mismatch\ndirect=%+v\naxiom=%+v", directResult.Plan.EvidencePlan, axiomCollector.plan.EvidencePlan)
	}

	got := axiomCollector.req
	if got.RunID != change.RunID || got.ProjectID != change.ProjectID || got.SourceDigest != change.SourceDigest {
		t.Fatalf("identity projection lost: %+v", got)
	}
	if got.BaseURL != change.BaseURL {
		t.Fatalf("base URL = %q, want %q", got.BaseURL, change.BaseURL)
	}
	if !reflect.DeepEqual(got.ChangedFiles, []string{"src/Button.tsx", "src/theme.css"}) {
		t.Fatalf("changed files lost: %v", got.ChangedFiles)
	}
	if !got.Scope.WholeSite {
		t.Fatalf("global token did not reach invalidation policy: %+v", got.Scope)
	}
	if !reflect.DeepEqual(got.Scope.Viewports, []string{"mobile"}) || !reflect.DeepEqual(got.Scope.Themes, []string{"dark"}) {
		t.Fatalf("viewport/theme scope lost: %+v", got.Scope)
	}
	if !containsString(got.Scope.Routes, "route:/settings") {
		t.Fatalf("target route lost: %v", got.Scope.Routes)
	}
}

func TestPipelineAdapterDoesNotAllowAxiomPlanToNarrowCanonicalRequest(t *testing.T) {
	ctx := context.Background()
	change := canonicalScopeChange()
	collector := &scopeCaptureCollector{}
	pipeline := &engine.Pipeline{Collector: collector, VerPolicy: verifier.DefaultPolicy()}
	adapter := NewPipelineAdapter(pipeline)

	// Deliberately empty/stale Axiom-side plan. It must not be able to narrow
	// the authoritative request or select a weaker evidence tier.
	if _, err := adapter.CollectVerify(ctx, change, controlplane.EvidencePlan{}); err != nil {
		t.Fatalf("CollectVerify failed: %v", err)
	}

	if !collector.plan.EvidencePlan.Structural {
		t.Fatalf("canonical full-deterministic request was narrowed by advisory Axiom plan: %+v", collector.plan)
	}
	if !collector.req.Need.Geometry || !collector.req.Need.Styles {
		t.Fatalf("canonical evidence need lost: %+v", collector.req.Need)
	}
	if collector.req.RunID != "run:fmea-002" {
		t.Fatalf("run id = %q, want caller-provided stable id", collector.req.RunID)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
