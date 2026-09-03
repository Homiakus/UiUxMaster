package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
	"github.com/Homiakus/UiUxMaster/internal/runtime/wggo"
)

type result struct {
	Scenario   string  `json:"scenario"`
	Nodes      int     `json:"nodes,omitempty"`
	Iterations int     `json:"iterations"`
	P50US      float64 `json:"p50_us"`
	P95US      float64 `json:"p95_us"`
	P99US      float64 `json:"p99_us"`
	MeanUS     float64 `json:"mean_us"`
	GoVersion  string  `json:"go_version"`
}

func main() {
	nodes := flag.Int("nodes", 1000, "synthetic graph node count")
	iterations := flag.Int("iterations", 5000, "measured impact iterations per scenario")
	warmup := flag.Int("warmup", 200, "impact warmup iterations per scenario")
	projectComponents := flag.Int("project-components", 200, "synthetic frontend component count")
	projectIterations := flag.Int("project-iterations", 50, "measured project-index build iterations")
	projectWarmup := flag.Int("project-warmup", 3, "project-index build warmup iterations")
	wggoIterations := flag.Int("wggo-iterations", 30, "measured WGGo render iterations")
	wggoWarmup := flag.Int("wggo-warmup", 3, "WGGo warmup iterations")
	flag.Parse()

	if *nodes < 2 || *iterations < 1 || *warmup < 0 || *projectComponents < 1 || *projectIterations < 1 || *projectWarmup < 0 || *wggoIterations < 1 || *wggoWarmup < 0 {
		fmt.Fprintln(os.Stderr, "nodes>=2, project-components>=1, iterations>=1 and warmups>=0 are required")
		os.Exit(2)
	}

	projectFiles := projectFixture(*projectComponents)
	benchmarks := []result{
		benchmarkLeaf(*nodes, *iterations, *warmup),
		benchmarkFanout(*nodes, *iterations, *warmup),
		benchmarkProjectIndexBuild(projectFiles, *projectIterations, *projectWarmup),
		benchmarkProjectTokenChange(projectFiles, *iterations, *warmup),
		benchmarkWGGoStatic(*wggoIterations, *wggoWarmup),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(benchmarks); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func benchmarkLeaf(nodes, iterations, warmup int) result {
	g := impact.NewGraph()
	for i := 0; i < nodes; i++ {
		id := fmt.Sprintf("n:%06d", i)
		must(g.AddNode(impact.Node{ID: id, Kind: impact.NodeComponent}))
		if i > 0 {
			prev := fmt.Sprintf("n:%06d", i-1)
			must(g.AddEdge(impact.Edge{From: prev, To: id, Kind: impact.EdgeDependsOn}))
		}
	}
	resolver, err := impact.NewResolver(g)
	must(err)
	leaf := fmt.Sprintf("n:%06d", nodes-1)
	ctx := context.Background()
	return measure("impact_leaf", nodes, iterations, warmup, func() error {
		_, err := resolver.ApplyChanges(ctx, impact.ChangeSet{NodeIDs: []string{leaf}})
		return err
	})
}

func benchmarkFanout(nodes, iterations, warmup int) result {
	g := impact.NewGraph()
	must(g.AddNode(impact.Node{ID: "token:shared", Kind: impact.NodeDesignToken}))
	for i := 0; i < nodes-1; i++ {
		id := fmt.Sprintf("component:%06d", i)
		must(g.AddNode(impact.Node{ID: id, Kind: impact.NodeComponent}))
		must(g.AddEdge(impact.Edge{From: "token:shared", To: id, Kind: impact.EdgeConsumesToken}))
	}
	resolver, err := impact.NewResolver(g)
	must(err)
	ctx := context.Background()
	return measure("impact_fanout", nodes, iterations, warmup, func() error {
		_, err := resolver.ApplyChanges(ctx, impact.ChangeSet{NodeIDs: []string{"token:shared"}})
		return err
	})
}

func benchmarkProjectIndexBuild(files []impact.ProjectFile, iterations, warmup int) result {
	return measure("project_index_build", len(files), iterations, warmup, func() error {
		idx, err := impact.IndexProject(files)
		if err != nil {
			return err
		}
		if idx.Uncertain {
			return fmt.Errorf("project benchmark unexpectedly uncertain: %v", idx.Reasons)
		}
		return nil
	})
}

func benchmarkProjectTokenChange(files []impact.ProjectFile, iterations, warmup int) result {
	idx, err := impact.IndexProject(files)
	must(err)
	if idx.Uncertain {
		must(fmt.Errorf("project benchmark unexpectedly uncertain: %v", idx.Reasons))
	}
	resolver, err := impact.NewResolver(idx.Graph)
	must(err)
	changes := idx.ChangeSetForFiles("src/theme.css")
	ctx := context.Background()
	return measure("project_token_change", len(files), iterations, warmup, func() error {
		got, err := resolver.ApplyChanges(ctx, changes)
		if err != nil {
			return err
		}
		if len(got.ComponentIDs) == 0 {
			return fmt.Errorf("project token benchmark produced no affected components")
		}
		return nil
	})
}

func projectFixture(components int) []impact.ProjectFile {
	files := make([]impact.ProjectFile, 0, 2+components*2)
	files = append(files, impact.ProjectFile{Path: "src/theme.css", Content: []byte(`:root{--brand:#3b82f6;}`)})

	var app strings.Builder
	app.WriteString(`import "./theme.css";\n`)
	for i := 0; i < components; i++ {
		name := fmt.Sprintf("C%04d", i)
		base := "src/components/" + name
		files = append(files,
			impact.ProjectFile{Path: base + ".css", Content: []byte(`.root{color:var(--brand);display:flex;gap:8px;}`)},
			impact.ProjectFile{Path: base + ".tsx", Content: []byte(fmt.Sprintf("import \"./%s.css\"; export function %s(){ return <div className=\"root\">%s</div>; }", name, name, name))},
		)
		app.WriteString(fmt.Sprintf("import { %s } from \"./components/%s\";\n", name, name))
	}
	app.WriteString("export function App(){ return null; }\n")
	files = append(files, impact.ProjectFile{Path: "src/App.tsx", Content: []byte(app.String())})
	return files
}

func benchmarkWGGoStatic(iterations, warmup int) result {
	r := wggo.New(wggo.Config{})
	ctx := context.Background()
	req := fastrender.Request{
		HTML: []byte(`<!doctype html><html><body><main class="shell"><section class="hero"><p class="eyebrow">UIUXMASTER</p><h1>Fast visual engineering loop</h1><p class="copy">Render only what changed and escalate only when fidelity requires it.</p><button>Inspect</button></section><aside class="metrics"><b>12 ms</b><span>targeted evidence</span></aside></main></body></html>`),
		CSS: []byte(`html,body{margin:0;font-family:sans-serif}.shell{display:grid;grid-template-columns:2fr 1fr;gap:32px;padding:48px}.hero{display:flex;flex-direction:column;gap:16px}.eyebrow{font-size:12px;letter-spacing:.12em}.hero h1{font-size:52px;line-height:1;margin:0;max-width:700px}.copy{font-size:18px;max-width:620px}.hero button{width:120px;height:42px}.metrics{display:flex;flex-direction:column;justify-content:end}`),
		Width:  1280,
		Height: 720,
	}
	return measure("wggo_static_render", 0, iterations, warmup, func() error {
		evidence, err := r.Render(ctx, req)
		if err != nil {
			return err
		}
		if evidence.RGBA == nil {
			return fmt.Errorf("wggo benchmark returned nil RGBA")
		}
		return nil
	})
}

func measure(name string, nodes, iterations, warmup int, fn func() error) result {
	for i := 0; i < warmup; i++ {
		must(fn())
	}

	samples := make([]time.Duration, iterations)
	var total time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		must(fn())
		d := time.Since(start)
		samples[i] = d
		total += d
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })

	return result{
		Scenario:   name,
		Nodes:      nodes,
		Iterations: iterations,
		P50US:      micros(percentile(samples, 0.50)),
		P95US:      micros(percentile(samples, 0.95)),
		P99US:      micros(percentile(samples, 0.99)),
		MeanUS:     micros(total / time.Duration(iterations)),
		GoVersion:  runtime.Version(),
	}
}

func percentile(samples []time.Duration, p float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	idx := int(float64(len(samples)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(samples) {
		idx = len(samples) - 1
	}
	return samples[idx]
}

func micros(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1000
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
