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

	// 1. Setup a live project on disk with real HTML/CSS files
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
  width: 3000px;
}
`
	if err := os.WriteFile(htmlPath, []byte(faultyHTML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cssPath, []byte(faultyCSS), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Setup SncSinCore EpMemoryStore
	memStore := memory.NewEpMemoryStore()

	// 3. Setup Pipeline & HostRepairEngine
	pipeline := &engine.Pipeline{
		Collector: &mockRepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	repairEngine := NewWithMemory(pipeline, memStore)

	// 4. Run closed-loop repair
	result, err := repairEngine.RunRepairLoop(ctx, RepairLoopRequest{
		RunID:         "live-repair-hydropilot-001",
		HTML:          faultyHTML,
		CSS:           faultyCSS,
		Profile:       design.FindProfile("saas-modern"),
		ProtectedAxes: []string{"accessibility", "responsive", "typography"},
		ProjectID:     "hydropilot",
		MaxIterations: 3,
	})
	if err != nil {
		t.Fatalf("RunRepairLoop failed: %v", err)
	}

	// 5. Apply the repaired content to the live files on disk
	if err := os.WriteFile(htmlPath, []byte(result.RepairedHTML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cssPath, []byte(result.RepairedCSS), 0644); err != nil {
		t.Fatal(err)
	}

	// 6. Assert 100% defect remediation and zero regression
	if result.InitialFindings == 0 {
		t.Fatalf("expected initial findings > 0, got %d", result.InitialFindings)
	}
	if result.FinalFindings != 0 {
		t.Fatalf("expected 0 final findings after closed-loop repair, got %d", result.FinalFindings)
	}
	if !result.Passed {
		t.Fatal("expected repair comparison to pass all constraints")
	}

	// 7. Verify SncSinCore Epistemic Memory Admission
	if !result.MemoryAdmitted || result.AdmittedAtoms == 0 {
		t.Fatalf("expected memory admission, got admitted=%v atoms=%d", result.MemoryAdmitted, result.AdmittedAtoms)
	}

	// Retrieve admitted atom from memory store
	ns, err := memory.NewProjectKnowledgeNamespace("hydropilot")
	if err != nil {
		t.Fatal(err)
	}

	res, err := memStore.Query(ctx, memory.QueryRequest{Namespace: ns})
	if err != nil {
		t.Fatalf("memStore.Query failed: %v", err)
	}
	if res.Total == 0 {
		t.Fatal("expected admitted atoms in hydropilot project namespace")
	}

	foundRepairPattern := false
	for _, a := range res.Atoms {
		if a.Kind == memory.NodeRepairPattern {
			foundRepairPattern = true
			if pat, ok := a.Data.(memory.RepairPatternAtom); ok {
				t.Logf("Admitted memory atom: pattern_id=%s, strategy=%q, success_rate=%.2f",
					pat.PatternID, pat.Strategy, pat.SuccessRate)
			}
		}
	}
	if !foundRepairPattern {
		t.Fatal("expected NodeRepairPattern atom committed to SncSinCore memory")
	}

	t.Logf("Live autonomous repair verification successful: %d initial findings remediated to 0, %d patches written to disk, %d atoms committed to SncSinCore memory",
		result.InitialFindings, len(result.PatchesApplied), res.Total)
}
