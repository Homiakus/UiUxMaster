package impact

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkResolverLeaf1K(b *testing.B) {
	benchmarkResolverLeaf(b, 1000)
}

func BenchmarkResolverLeaf10K(b *testing.B) {
	benchmarkResolverLeaf(b, 10000)
}

func BenchmarkResolverLeaf100K(b *testing.B) {
	benchmarkResolverLeaf(b, 100000)
}

func BenchmarkResolverFanout1K(b *testing.B) {
	benchmarkResolverFanout(b, 1000)
}

func BenchmarkResolverFanout10K(b *testing.B) {
	benchmarkResolverFanout(b, 10000)
}

func BenchmarkResolverChain1K(b *testing.B) {
	benchmarkResolverChain(b, 1000)
}

func BenchmarkResolverChain10K(b *testing.B) {
	benchmarkResolverChain(b, 10000)
}

func benchmarkResolverLeaf(b *testing.B, size int) {
	b.Helper()
	g := buildChainBenchmark(b, size)
	resolver, err := NewResolver(g)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	leafID := fmt.Sprintf("n:%d", size-1)
	change := ChangeSet{NodeIDs: []string{leafID}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := resolver.ApplyChanges(ctx, change); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkResolverFanout(b *testing.B, fanoutCount int) {
	b.Helper()
	g := NewGraph()
	if err := g.AddNode(Node{ID: "token:shared", Kind: NodeDesignToken}); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < fanoutCount; i++ {
		id := fmt.Sprintf("component:%05d", i)
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

func benchmarkResolverChain(b *testing.B, depth int) {
	b.Helper()
	g := buildChainBenchmark(b, depth)
	resolver, err := NewResolver(g)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	change := ChangeSet{NodeIDs: []string{"n:0"}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := resolver.ApplyChanges(ctx, change); err != nil {
			b.Fatal(err)
		}
	}
}

func buildChainBenchmark(tb testing.TB, count int) *Graph {
	tb.Helper()
	g := NewGraph()
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("n:%d", i)
		if err := g.AddNode(Node{ID: id, Kind: NodeComponent}); err != nil {
			tb.Fatal(err)
		}
		if i == 0 {
			continue
		}
		prev := fmt.Sprintf("n:%d", i-1)
		if err := g.AddEdge(Edge{From: prev, To: id, Kind: EdgeDependsOn}); err != nil {
			tb.Fatal(err)
		}
	}
	return g
}

// TestImpactAllocationGates verifies that queries across 1k, 10k, and 100k graphs
// satisfy strict latency and allocation ceilings to prevent memory regressions.
func TestImpactAllocationGates(t *testing.T) {
	ctx := context.Background()

	scales := []struct {
		name          string
		size          int
		maxLeafAllocs float64
		maxLeafTime   time.Duration
	}{
		{name: "1K", size: 1000, maxLeafAllocs: 25, maxLeafTime: 2 * time.Millisecond},
		{name: "10K", size: 10000, maxLeafAllocs: 30, maxLeafTime: 5 * time.Millisecond},
		{name: "100K", size: 100000, maxLeafAllocs: 35, maxLeafTime: 20 * time.Millisecond},
	}

	for _, sc := range scales {
		t.Run(sc.name, func(t *testing.T) {
			g := buildChainBenchmark(t, sc.size)
			resolver, err := NewResolver(g)
			if err != nil {
				t.Fatalf("NewResolver failed: %v", err)
			}

			leafID := fmt.Sprintf("n:%d", sc.size-1)
			change := ChangeSet{NodeIDs: []string{leafID}}

			// Warmup
			if _, err := resolver.ApplyChanges(ctx, change); err != nil {
				t.Fatalf("warmup ApplyChanges failed: %v", err)
			}

			// Measure allocations
			allocs := testing.AllocsPerRun(50, func() {
				res, err := resolver.ApplyChanges(ctx, change)
				if err != nil {
					t.Fatalf("ApplyChanges failed: %v", err)
				}
				if len(res.ComponentIDs) != 1 || res.ComponentIDs[0] != leafID {
					t.Fatalf("unexpected components: %#v", res.ComponentIDs)
				}
			})

			if allocs > sc.maxLeafAllocs {
				t.Fatalf("%s scale exceeded allocation gate: got %.1f allocs, max allowed %.1f",
					sc.name, allocs, sc.maxLeafAllocs)
			}

			// Measure single-run latency
			start := time.Now()
			for iter := 0; iter < 100; iter++ {
				_, _ = resolver.ApplyChanges(ctx, change)
			}
			avgTime := time.Since(start) / 100
			if avgTime > sc.maxLeafTime {
				t.Fatalf("%s scale exceeded latency gate: got %v, max allowed %v",
					sc.name, avgTime, sc.maxLeafTime)
			}
		})
	}
}
