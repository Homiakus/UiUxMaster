package engine_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
	"github.com/Homiakus/UiUxMaster/internal/runtime/dispatcher"
	"github.com/Homiakus/UiUxMaster/internal/runtime/wggo"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

func TestLiveIncrementalEditPipelineTelemetryAudit(t *testing.T) {
	ctx := context.Background()

	// 1. Setup a live project directory structure
	tmpDir, err := os.MkdirTemp("", "uiux-live-pipeline-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mustWrite := func(relPath, content string) {
		full := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("src/theme.css", `:root { --primary-color: #2563eb; --radius: 6px; }`)
	mustWrite("src/components/Button.css", `.btn { background: var(--primary-color); border-radius: var(--radius); padding: 8px 16px; border: none; }`)
	mustWrite("src/components/Button.tsx", `import './Button.css'; export function Button(){ return <button className="btn">Execute</button>; }`)
	mustWrite("src/components/Card.tsx", `export function Card(){ return <div className="card">Telemetry</div>; }`)
	mustWrite("src/app/page.tsx", `import {Button} from '../components/Button'; export default function HomePage(){ return <main><Button/></main>; }`)
	mustWrite("src/app/dashboard/page.tsx", `import {Card} from '../../components/Card'; export default function DashboardPage(){ return <main><Card/></main>; }`)

	// 2. Ingest live project structure into ProjectIndex & Resolver
	index, err := impact.IndexDirectory(tmpDir)
	if err != nil {
		t.Fatalf("impact.IndexDirectory failed: %v", err)
	}
	if index.Uncertain {
		t.Fatalf("unexpected index uncertainty: %#v", index.Reasons)
	}

	resolver, err := impact.NewResolver(index.Graph)
	if err != nil {
		t.Fatalf("impact.NewResolver failed: %v", err)
	}

	// 3. Setup Dispatcher with L1 renderer and L2 collector
	wggoRenderer := wggo.New(wggo.Config{})
	mockL2 := &mockCDPCollector{}
	d := dispatcher.New(dispatcher.Config{
		L1Renderer:                  wggoRenderer,
		L2Collector:                 mockL2,
		EscalateL1ToL2OnUnsupported: true,
	})

	pipeline := engine.Pipeline{
		Resolver:  resolver,
		Policy:    invalidation.DefaultPolicy(),
		Collector: d,
		VerPolicy: verifier.DefaultPolicy(),
	}

	// 4. Simulate an incremental edit mutation touching only Button.tsx / Button.css
	// (Card.tsx and route:/dashboard MUST NOT be invalidated)
	req := engine.ValidationRequest{
		RunID:        "live-incremental-edit-001",
		ChangedFiles: []string{"src/components/Button.tsx"},
		HTML:         []byte(`<!DOCTYPE html><html><body><main><button style="width:100px;height:36px;background:#2563eb;color:#fff;border-radius:6px;border:none;">Execute</button></main></body></html>`),
		CSS:          []byte(`button { font-family: sans-serif; }`),
		Need:         engine.EvidenceNeed{Pixels: true},
	}

	// 5. Execute canonical pipeline & collect telemetry
	res, err := pipeline.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Pipeline.Execute failed: %v", err)
	}

	// 6. Assert incremental-only invalidation scope
	if len(res.Scope.Components) == 0 {
		t.Fatal("expected non-empty invalidated components")
	}
	for _, comp := range res.Scope.Components {
		if comp == "component:src/components/Card.tsx" {
			t.Errorf("unaffected component Card.tsx was unexpectedly invalidated: %s", comp)
		}
	}
	for _, route := range res.Scope.Routes {
		if route == "route:/dashboard" {
			t.Errorf("unaffected route /dashboard was unexpectedly invalidated: %s", route)
		}
	}
	if len(res.Scope.Routes) != 1 || res.Scope.Routes[0] != "route:/" {
		t.Fatalf("scope routes = %#v, want ['route:/']", res.Scope.Routes)
	}

	// 7. Audit Telemetry Latency Budgets
	tel := res.Telemetry
	t.Logf("Pipeline Telemetry Breakdown: TotalMS=%.3f, ImpactMS=%.3f, FidelityScanMS=%.3f, RouteMS=%.3f, CollectMS=%.3f, VerifyMS=%.3f, SynthesisMS=%.3f",
		tel.TotalMS, tel.ImpactMS, tel.FidelityScanMS, tel.RouteMS, tel.CollectMS, tel.VerifyMS, tel.SynthesisMS)

	if tel.ImpactMS > 50.0 { // loose upper bound for cold run in tests, typical is <1ms
		t.Errorf("ImpactMS %.3fms exceeded budget", tel.ImpactMS)
	}
	if tel.TotalMS > 500.0 { // test timeout budget
		t.Errorf("TotalMS %.3fms exceeded budget", tel.TotalMS)
	}
	if tel.VerifyMS > 20.0 {
		t.Errorf("VerifyMS %.3fms exceeded budget", tel.VerifyMS)
	}

	// 8. Verify Engine Report Decision & Verification Result
	if res.Report.RunID != req.RunID {
		t.Fatalf("report runID = %q, want %q", res.Report.RunID, req.RunID)
	}
	if res.Packet.Pixels == nil {
		t.Fatal("expected Pixels in canonical evidence packet")
	}
	if res.Report.RecommendedNext == "" {
		t.Fatal("expected RecommendedNext decision in report")
	}
}
