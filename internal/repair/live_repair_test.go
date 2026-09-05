package repair

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/memory"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

func TestLiveAutonomousRepairLoopAndMemoryAdmission(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "uiux-live-repair-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	htmlPath := filepath.Join(tmpDir, "index.html")
	cssPath := filepath.Join(tmpDir, "style.css")

	faultyHTML := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <link rel="stylesheet" href="./style.css">
  <title>Hydropilot Controller</title>
</head>
<body>
  <div>
    <p>Telemetry view without heading</p>
    <button id="pump-trigger"></button>
  </div>
</body>
</html>`

	faultyCSS := `
body {
  width: 2000px;
}
`
	if err := os.WriteFile(htmlPath, []byte(faultyHTML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cssPath, []byte(faultyCSS), 0644); err != nil {
		t.Fatal(err)
	}

	memStore := memory.NewEpMemoryStore()
	optimizationPipeline := &engine.Pipeline{
		Collector: &mockRepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	finalPipeline := &engine.Pipeline{
		Collector: &mockL3RepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	finalGate := NewPipelineFinalGate(finalPipeline, NewPrivateHeldOutSuite(passHeldOutProbe{}))
	repairEngine := NewWithMemoryAndFinalGate(optimizationPipeline, finalGate, memStore)

	result, err := repairEngine.RunRepairLoop(ctx, RepairLoopRequest{
		RunID:         "live-repair-hydropilot-001",
		HTML:          faultyHTML,
		CSS:           faultyCSS,
		Profile:       design.FindProfile("saas-modern"),
		ProtectedAxes: []string{"accessibility", "responsive", "typography", "interaction"},
		ProjectID:     "hydropilot",
		MaxIterations: 3,
		RiskClass:     RepairRiskCritical,
	})
	if err != nil {
		t.Fatalf("RunRepairLoop failed: %v", err)
	}

	if err := os.WriteFile(htmlPath, []byte(result.RepairedHTML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cssPath, []byte(result.RepairedCSS), 0644); err != nil {
		t.Fatal(err)
	}

	if result.InitialFindings == 0 {
		t.Fatalf("expected initial findings > 0, got %d", result.InitialFindings)
	}
	if result.FinalFindings != 0 {
		t.Fatalf("expected 0 optimization findings after repair, got %d", result.FinalFindings)
	}
	if !result.Passed || !result.FinalGate.Independent {
		t.Fatalf("expected independent final PASS, final=%#v", result.FinalGate)
	}
	if result.FinalGate.EvidenceTier != "L3" || result.Metrics.HeldOutEscapeRate != 0 {
		t.Fatalf("unexpected final evidence/metrics: final=%#v metrics=%#v", result.FinalGate, result.Metrics)
	}
	if !result.MemoryAdmitted || result.AdmittedAtoms == 0 {
		t.Fatalf("expected memory admission only after final PASS, got admitted=%v atoms=%d", result.MemoryAdmitted, result.AdmittedAtoms)
	}

	ns, err := memory.NewProjectKnowledgeNamespace("hydropilot")
	if err != nil {
		t.Fatal(err)
	}
	res, err := memStore.Query(ctx, memory.QueryRequest{Namespace: ns})
	if err != nil {
		t.Fatalf("memStore.Query failed: %v", err)
	}
	if res.Total == 0 {
		t.Fatal("expected independently proven repair atoms in hydropilot namespace")
	}

	foundRepairPattern := false
	for _, atom := range res.Atoms {
		if atom.Kind == memory.NodeRepairPattern {
			foundRepairPattern = true
			break
		}
	}
	if !foundRepairPattern {
		t.Fatal("expected NodeRepairPattern atom committed after independent PASS")
	}
}
