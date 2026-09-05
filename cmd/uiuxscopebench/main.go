package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
)

type stageStats struct {
	P50US  float64 `json:"p50_us"`
	P95US  float64 `json:"p95_us"`
	P99US  float64 `json:"p99_us"`
	MeanUS float64 `json:"mean_us"`
}

type artifact struct {
	SchemaVersion int        `json:"schema_version"`
	Iterations    int        `json:"iterations"`
	Impact        stageStats `json:"impact"`
	Invalidation  stageStats `json:"invalidation"`
	ImpactNodes   int        `json:"impact_nodes"`
	ScopeSize     int        `json:"scope_size"`
}

func main() {
	iterations := flag.Int("iterations", 200, "measured scope-planning iterations")
	warmup := flag.Int("warmup", 20, "warmup iterations")
	flag.Parse()
	if *iterations < 1 || *warmup < 0 {
		fmt.Fprintln(os.Stderr, "iterations>=1 and warmup>=0 are required")
		os.Exit(2)
	}

	builder := impact.NewBuilder()
	must(builder.TokenAffects("token:brand", "component:card", impact.NodeComponent))
	must(builder.ComponentInstance("component:card", "instance:card"))
	must(builder.PlaceInstance("instance:card", "route:home", "region:0,0,640,360"))
	resolver, err := impact.NewResolver(builder.Graph())
	must(err)
	policy := invalidation.DefaultPolicy()
	ctx := context.Background()
	req := engine.ValidationRequest{ChangedTokens: []string{"brand"}}

	for i := 0; i < *warmup; i++ {
		normalized, set, err := engine.ResolveImpact(ctx, req, resolver)
		must(err)
		_, _, err = engine.InvalidateImpact(ctx, normalized, set, policy)
		must(err)
	}

	impactSamples := make([]time.Duration, *iterations)
	invalidationSamples := make([]time.Duration, *iterations)
	var lastImpact impact.ImpactSet
	var lastScope invalidation.ValidationScope
	for i := 0; i < *iterations; i++ {
		startImpact := time.Now()
		normalized, set, err := engine.ResolveImpact(ctx, req, resolver)
		must(err)
		impactSamples[i] = time.Since(startImpact)

		startInvalidation := time.Now()
		_, scope, err := engine.InvalidateImpact(ctx, normalized, set, policy)
		must(err)
		invalidationSamples[i] = time.Since(startInvalidation)
		lastImpact = set
		lastScope = scope
	}

	out := artifact{
		SchemaVersion: 1,
		Iterations: *iterations,
		Impact: summarize(impactSamples),
		Invalidation: summarize(invalidationSamples),
		ImpactNodes: len(lastImpact.NodeIDs),
		ScopeSize: len(lastScope.Components) + len(lastScope.Routes) + len(lastScope.Regions),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	must(enc.Encode(out))
}

func summarize(samples []time.Duration) stageStats {
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var total time.Duration
	for _, sample := range sorted { total += sample }
	return stageStats{
		P50US: micros(percentile(sorted, 0.50)),
		P95US: micros(percentile(sorted, 0.95)),
		P99US: micros(percentile(sorted, 0.99)),
		MeanUS: micros(total / time.Duration(len(sorted))),
	}
}

func percentile(samples []time.Duration, p float64) time.Duration {
	if len(samples) == 0 { return 0 }
	idx := int(float64(len(samples)-1) * p)
	if idx < 0 { idx = 0 }
	if idx >= len(samples) { idx = len(samples)-1 }
	return samples[idx]
}

func micros(d time.Duration) float64 { return float64(d.Nanoseconds()) / 1000 }

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
