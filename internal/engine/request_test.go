package engine

import (
	"context"
	"reflect"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
)

func TestValidationRequest_Normalize(t *testing.T) {
	req := ValidationRequest{
		ChangedFiles: []string{"b.css", "a.tsx", "b.css"},
		TargetRoutes: []string{"page:settings", "page:home"},
	}
	req.Normalize()

	if req.RunID != "run:default" {
		t.Errorf("RunID = %q, want run:default", req.RunID)
	}
	if req.Intent != evidenceplan.IntentQuickStructural {
		t.Errorf("Intent = %q, want %q", req.Intent, evidenceplan.IntentQuickStructural)
	}
	wantFiles := []string{"a.tsx", "b.css"}
	if !reflect.DeepEqual(req.ChangedFiles, wantFiles) {
		t.Errorf("ChangedFiles = %v, want %v", req.ChangedFiles, wantFiles)
	}
}

func TestValidationRequest_DeriveNeed(t *testing.T) {
	req := ValidationRequest{
		Intent:    evidenceplan.IntentVisualRegion,
		FinalGate: true,
	}
	need := req.DeriveNeed()

	if !need.Geometry || !need.Styles || !need.Pixels || !need.CleanState {
		t.Errorf("Need = %+v, expected Geometry, Styles, Pixels, CleanState to be true", need)
	}
}

func TestValidationRequest_Signals(t *testing.T) {
	req := ValidationRequest{
		Intent:        evidenceplan.IntentInteraction,
		ChangedTokens: []string{"typography:base"},
		Need:          EvidenceNeed{Scenario: true},
	}
	sig := req.Signals(fidelity.RiskMedium)

	if sig.Intent != evidenceplan.IntentInteraction {
		t.Errorf("Intent = %q, want %q", sig.Intent, evidenceplan.IntentInteraction)
	}
	if !sig.CustomFontsChanged {
		t.Errorf("expected CustomFontsChanged=true")
	}
	if !sig.InteractionChanged {
		t.Errorf("expected InteractionChanged=true")
	}
}

func TestPlanScope_OrchestrationWithResolver(t *testing.T) {
	g := impact.NewGraph()
	mustAddNode := func(n impact.Node) {
		if err := g.AddNode(n); err != nil {
			t.Fatal(err)
		}
	}
	mustAddEdge := func(e impact.Edge) {
		if err := g.AddEdge(e); err != nil {
			t.Fatal(err)
		}
	}

	mustAddNode(impact.Node{ID: "file:button.tsx", Kind: impact.NodeSourceFile})
	mustAddNode(impact.Node{ID: "component:button", Kind: impact.NodeComponent})
	mustAddNode(impact.Node{ID: "page:home", Kind: impact.NodePage})
	mustAddNode(impact.Node{ID: "region:cta", Kind: impact.NodeRenderRegion})

	mustAddEdge(impact.Edge{From: "file:button.tsx", To: "component:button", Kind: impact.EdgeRenders})
	mustAddEdge(impact.Edge{From: "component:button", To: "page:home", Kind: impact.EdgeAppearsOn})
	mustAddEdge(impact.Edge{From: "component:button", To: "region:cta", Kind: impact.EdgeAffectsRegion})

	resolver, err := impact.NewResolver(g)
	if err != nil {
		t.Fatal(err)
	}

	req := ValidationRequest{
		RunID:        "run:test-1",
		ChangedFiles: []string{"button.tsx"},
		Intent:       evidenceplan.IntentQuickStructural,
	}

	policy := invalidation.DefaultPolicy()
	updatedReq, scope, err := PlanScope(context.Background(), req, resolver, policy)
	if err != nil {
		t.Fatalf("PlanScope error: %v", err)
	}

	wantComponents := []string{"component:button"}
	if !reflect.DeepEqual(scope.Components, wantComponents) {
		t.Errorf("scope.Components = %v, want %v", scope.Components, wantComponents)
	}

	wantRoutes := []string{"page:home"}
	if !reflect.DeepEqual(scope.Routes, wantRoutes) {
		t.Errorf("scope.Routes = %v, want %v", scope.Routes, wantRoutes)
	}

	wantRegions := []string{"region:cta"}
	if !reflect.DeepEqual(scope.Regions, wantRegions) {
		t.Errorf("scope.Regions = %v, want %v", scope.Regions, wantRegions)
	}

	if !reflect.DeepEqual(updatedReq.Scope, scope) {
		t.Errorf("updatedReq.Scope does not match returned scope")
	}
}

func TestPlanProjectScope_OrchestratesFromFiles(t *testing.T) {
	files := []impact.ProjectFile{
		{
			Path:    "components/Header.tsx",
			Content: []byte(`export function Header() { return <header>Header</header>; }`),
		},
		{
			Path: "pages/Home.tsx",
			Content: []byte(`
				import { Header } from "../components/Header";
				export function Home() { return <div><Header /></div>; }
			`),
		},
	}

	index, err := impact.IndexProject(files)
	if err != nil {
		t.Fatalf("IndexProject: %v", err)
	}

	req := ValidationRequest{
		RunID:        "run:project-1",
		ChangedFiles: []string{"components/Header.tsx"},
		Intent:       evidenceplan.IntentQuickStructural,
	}

	policy := invalidation.DefaultPolicy()
	updatedReq, scope, err := PlanProjectScope(context.Background(), req, index, policy)
	if err != nil {
		t.Fatalf("PlanProjectScope error: %v", err)
	}

	wantComponents := []string{"component:components/Header.tsx", "component:pages/Home.tsx"}
	if !reflect.DeepEqual(scope.Components, wantComponents) {
		t.Errorf("scope.Components = %v, want %v", scope.Components, wantComponents)
	}
	if !reflect.DeepEqual(updatedReq.Scope, scope) {
		t.Errorf("updatedReq.Scope does not match returned scope")
	}
}
