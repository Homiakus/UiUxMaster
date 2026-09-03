package impact

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkResolverLeaf1000(b *testing.B) {
	g := buildChainBenchmark(b, 1000)
	resolver, err := NewResolver(g)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	change := ChangeSet{NodeIDs: []string{"n:999"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := resolver.ApplyChanges(ctx, change); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolverFanout1000(b *testing.B) {
	g := NewGraph()
	if err := g.AddNode(Node{ID: "token:shared", Kind: NodeDesignToken}); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("component:%04d", i)
		if err := g.AddNode(Node{ID: id, Kind: NodeComponent}); err != nil {
			b.Fatal(err)
		}
		if err := g.AddEdge(Edge{From: "token:shared", To: id, Kind: EdgeConsumesToken}); err != nil {
			b.Fatal(err)
		}
	}
	resolver, err := NewResolver(g)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	change := ChangeSet{NodeIDs: []string{"token:shared"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := resolver.ApplyChanges(ctx, change); err != nil {
			b.Fatal(err)
		}
	}
}

func buildChainBenchmark(b *testing.B, count int) *Graph {
	b.Helper()
	g := NewGraph()
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("n:%d", i)
		if err := g.AddNode(Node{ID: id, Kind: NodeComponent}); err != nil {
			b.Fatal(err)
		}
		if i == 0 {
			continue
		}
		prev := fmt.Sprintf("n:%d", i-1)
		if err := g.AddEdge(Edge{From: prev, To: id, Kind: EdgeDependsOn}); err != nil {
			b.Fatal(err)
		}
	}
	return g
}
