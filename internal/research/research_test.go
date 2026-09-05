package research

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/memory"
)

func TestResearch_SearchAndAdmission(t *testing.T) {
	researcher := NewMemoryCatalogResearcher()
	ctx := context.Background()

	bundle, err := researcher.Search(ctx, ResearchRequest{Query: "wcag contrast", Domain: "accessibility"})
	if err != nil { t.Fatalf("unexpected error searching research: %v", err) }
	if len(bundle.Sources) == 0 { t.Fatalf("expected at least 1 source in WCAG bundle") }
	if len(bundle.Claims) < 2 { t.Fatalf("expected at least 2 claims in WCAG bundle, got %d", len(bundle.Claims)) }

	researcher.SetAvailable(false)
	if _, err = researcher.Search(ctx, ResearchRequest{Query: "wcag"}); err == nil { t.Fatalf("expected error when researcher is offline") }
	researcher.SetAvailable(true)

	store := memory.NewEpMemoryStore()
	adapter := NewAdmissionAdapter(store)
	targetNamespace := memory.NewResearchGlobalNamespace()
	if err := adapter.IngestBundle(ctx, bundle, targetNamespace); err != nil { t.Fatalf("failed to ingest research bundle: %v", err) }

	qRes, err := store.Query(ctx, memory.QueryRequest{Namespace: targetNamespace, Kind: memory.NodeResearchSource})
	if err != nil || qRes.Total != 1 { t.Fatalf("expected 1 research source atom in store, got %d, err: %v", qRes.Total, err) }
	qClaims, err := store.Query(ctx, memory.QueryRequest{Namespace: targetNamespace, Kind: memory.NodeDesignRule, Tags: []string{"research_claim"}})
	if err != nil || qClaims.Total != 2 { t.Fatalf("expected 2 research claim rule atoms in store, got %d, err: %v", qClaims.Total, err) }

	edges, err := store.GetEdges(ctx, targetNamespace, "claim_claim_wcag_contrast")
	if err != nil || len(edges) == 0 { t.Fatalf("expected RelDerivedFrom edge for claim, got %d edges err=%v", len(edges), err) }
	if edges[0].Relation != memory.RelDerivedFrom { t.Fatalf("expected RelDerivedFrom, got %s", edges[0].Relation) }

	pack, err := store.RetrieveContextPack(ctx, memory.ContextPackRequest{Scope: targetNamespace, FocusAxes: []string{"accessibility"}, BudgetTokens: 1000})
	if err != nil { t.Fatalf("failed to retrieve context pack with research claims: %v", err) }
	if len(pack.AdmittedRules) != 2 { t.Fatalf("expected 2 admitted rules in context pack from research, got %d", len(pack.AdmittedRules)) }
}
