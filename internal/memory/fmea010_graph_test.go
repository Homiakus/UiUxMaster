package memory

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestFMEA010NamespaceJSONRoundTripPreservesFirewallIdentity(t *testing.T) {
	ns, err := NewProjectKnowledgeNamespace("project-json")
	if err != nil { t.Fatal(err) }
	raw, err := json.Marshal(ns)
	if err != nil { t.Fatal(err) }
	if string(raw) != `"knowledge/project/project-json"` { t.Fatalf("json=%s", raw) }
	var decoded Namespace
	if err := json.Unmarshal(raw, &decoded); err != nil { t.Fatal(err) }
	if !decoded.Equal(ns) || decoded.ProjectID() != "project-json" { t.Fatalf("decoded=%#v", decoded) }
}

func TestFMEA010AtomIDCollisionCannotRewriteAnotherNamespace(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	global := NewGlobalDesignNamespace()
	globalProv := ProvenanceRecord{RunID: "global", EvidenceDigest: "sha256:global", Renderer: "curated", ProjectScope: "global", SourceNamespace: global.String(), Timestamp: time.Now()}
	globalAtom := MemoryAtom{ID: "shared-id", Kind: NodeDesignRule, Namespace: global, Provenance: globalProv, Confidence: 1, Data: DesignRuleAtom{RuleID: "global", Axis: "layout", Category: "global", Title: "Global", Description: "Global invariant"}}
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: global, Atoms: []MemoryAtom{globalAtom}}); err != nil { t.Fatal(err) }

	sourceA, projectA := fmea010ProjectScopes(t, "project-a")
	provA := fmea010Prov(sourceA, "project-a", "run-a", "sha256:a", 1)
	colliding := fmea010Finding("shared-id", projectA, provA, 1)
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: sourceA, Atoms: []MemoryAtom{colliding}}); !errors.Is(err, ErrAdmissionRoute) { t.Fatalf("collision err=%v", err) }
	got, err := store.GetAtom(ctx, global, "shared-id")
	if err != nil || !got.Namespace.Equal(global) { t.Fatalf("global atom was rewritten: %#v err=%v", got, err) }
}

func TestFMEA010GraphEdgesCannotCrossPrivateProjectsOrMutateGlobalConflicts(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	sourceA, projectA := fmea010ProjectScopes(t, "project-a")
	sourceB, projectB := fmea010ProjectScopes(t, "project-b")
	provA := fmea010Prov(sourceA, "project-a", "run-a", "sha256:a", 1)
	provB := fmea010Prov(sourceB, "project-b", "run-b", "sha256:b", 1)
	a := fmea010Finding("a", projectA, provA, 1)
	b := fmea010Finding("b", projectB, provB, 1)
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: sourceA, Atoms: []MemoryAtom{a}}); err != nil { t.Fatal(err) }
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: sourceB, Atoms: []MemoryAtom{b}}); err != nil { t.Fatal(err) }

	cross := MemoryEdge{FromID: "a", ToID: "b", Relation: RelObservedOn, Weight: 1, Provenance: provA, CreatedAt: time.Now()}
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: sourceA, Edges: []MemoryEdge{cross}}); !errors.Is(err, ErrUnauthorizedAccess) { t.Fatalf("cross-project edge err=%v", err) }

	global := NewGlobalDesignNamespace()
	globalProv := ProvenanceRecord{RunID: "global", EvidenceDigest: "sha256:g", Renderer: "curated", ProjectScope: "global", SourceNamespace: global.String(), Timestamp: time.Now()}
	globalRule := MemoryAtom{ID: "global-rule", Kind: NodeDesignRule, Namespace: global, Provenance: globalProv, Confidence: 1, Data: DesignRuleAtom{RuleID: "g", Axis: "layout", Category: "global", Title: "Global", Description: "Global invariant"}}
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: global, Atoms: []MemoryAtom{globalRule}}); err != nil { t.Fatal(err) }
	refuteGlobal := MemoryEdge{FromID: "a", ToID: "global-rule", Relation: RelRefutes, Weight: 1, Provenance: provA, CreatedAt: time.Now()}
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: sourceA, Edges: []MemoryEdge{refuteGlobal}}); !errors.Is(err, ErrUnauthorizedAccess) { t.Fatalf("project conflict mutation err=%v", err) }
	if stored := store.atoms["global-rule"]; stored == nil || len(stored.Conflicts) != 0 { t.Fatalf("global conflict metadata mutated: %#v", stored) }
}

func TestFMEA010InactiveEdgeEndpointNeverLeaks(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	source, project := fmea010ProjectScopes(t, "project-edge")
	prov := fmea010Prov(source, "project-edge", "run-edge", "sha256:edge", 1)
	a := fmea010Finding("edge:a", project, prov, 1)
	b := fmea010Finding("edge:b", project, prov, 1)
	bundle := AdmissionBundle{SourceNamespace: source, Atoms: []MemoryAtom{a, b}, Edges: []MemoryEdge{{FromID: a.ID, ToID: b.ID, Relation: RelDerivedFrom, Weight: 1, Provenance: prov, CreatedAt: time.Now()}}}
	if err := store.Commit(ctx, bundle); err != nil { t.Fatal(err) }
	if err := store.Retract(ctx, project, b.ID, "invalidate target", prov); err != nil { t.Fatal(err) }
	edges, err := store.GetEdges(ctx, project, a.ID)
	if err != nil { t.Fatal(err) }
	if len(edges) != 0 { t.Fatalf("edge to inactive endpoint leaked: %#v", edges) }
}
