package impact

import (
	"context"
	"reflect"
	"testing"
)

func TestSnapshotDeterministic(t *testing.T) {
	build := func(reverse bool) Snapshot {
		g := NewGraph()
		nodes := []Node{
			{ID: "file:a", Kind: NodeSourceFile},
			{ID: "module:a", Kind: NodeModule},
			{ID: "component:a", Kind: NodeComponent},
		}
		if reverse {
			for i := len(nodes) - 1; i >= 0; i-- {
				mustAddNode(t, g, nodes[i])
			}
		} else {
			for _, n := range nodes {
				mustAddNode(t, g, n)
			}
		}
		mustAddEdge(t, g, Edge{From: "file:a", To: "module:a", Kind: EdgeImports})
		mustAddEdge(t, g, Edge{From: "module:a", To: "component:a", Kind: EdgeRenders})
		return g.Snapshot()
	}

	if got, want := build(false), build(true); !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot depends on insertion order\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestResolverPropagatesToValidationScope(t *testing.T) {
	g := NewGraph()
	for _, n := range []Node{
		{ID: "file:button", Kind: NodeSourceFile},
		{ID: "module:button", Kind: NodeModule},
		{ID: "component:button", Kind: NodeComponent},
		{ID: "instance:hero-cta", Kind: NodeComponentInstance},
		{ID: "page:home", Kind: NodePage},
		{ID: "region:hero-actions", Kind: NodeRenderRegion},
	} {
		mustAddNode(t, g, n)
	}
	for _, e := range []Edge{
		{From: "file:button", To: "module:button", Kind: EdgeImports},
		{From: "module:button", To: "component:button", Kind: EdgeRenders},
		{From: "component:button", To: "instance:hero-cta", Kind: EdgeInstantiates},
		{From: "instance:hero-cta", To: "page:home", Kind: EdgeAppearsOn},
		{From: "instance:hero-cta", To: "region:hero-actions", Kind: EdgeAffectsRegion},
	} {
		mustAddEdge(t, g, e)
	}

	resolver, err := NewResolver(g)
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolver.ApplyChanges(context.Background(), ChangeSet{NodeIDs: []string{"file:button"}})
	if err != nil {
		t.Fatal(err)
	}

	wantComponents := []string{"component:button", "instance:hero-cta"}
	wantRoutes := []string{"page:home"}
	wantRegions := []string{"region:hero-actions"}
	if !reflect.DeepEqual(got.ComponentIDs, wantComponents) {
		t.Fatalf("components = %#v, want %#v", got.ComponentIDs, wantComponents)
	}
	if !reflect.DeepEqual(got.RouteIDs, wantRoutes) {
		t.Fatalf("routes = %#v, want %#v", got.RouteIDs, wantRoutes)
	}
	if !reflect.DeepEqual(got.RegionIDs, wantRegions) {
		t.Fatalf("regions = %#v, want %#v", got.RegionIDs, wantRegions)
	}
	if got.Broad {
		t.Fatal("known local change unexpectedly marked broad")
	}
}

func TestResolverUnknownChangeFailsConservatively(t *testing.T) {
	resolver, err := NewResolver(NewGraph())
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolver.ApplyChanges(context.Background(), ChangeSet{NodeIDs: []string{"file:unknown"}})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Broad {
		t.Fatal("unknown change must expand scope")
	}
	if !reflect.DeepEqual(got.UnknownIDs, []string{"file:unknown"}) {
		t.Fatalf("unknown IDs = %#v", got.UnknownIDs)
	}
}

func TestStronglyConnectedComponents(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c"} {
		mustAddNode(t, g, Node{ID: id, Kind: NodeModule})
	}
	mustAddEdge(t, g, Edge{From: "a", To: "b", Kind: EdgeImports})
	mustAddEdge(t, g, Edge{From: "b", To: "a", Kind: EdgeImports})
	mustAddEdge(t, g, Edge{From: "b", To: "c", Kind: EdgeImports})

	got := g.StronglyConnectedComponents()
	want := [][]string{{"a", "b"}, {"c"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SCCs = %#v, want %#v", got, want)
	}
}

func mustAddNode(t *testing.T, g *Graph, n Node) {
	t.Helper()
	if err := g.AddNode(n); err != nil {
		t.Fatal(err)
	}
}

func mustAddEdge(t *testing.T, g *Graph, e Edge) {
	t.Helper()
	if err := g.AddEdge(e); err != nil {
		t.Fatal(err)
	}
}
