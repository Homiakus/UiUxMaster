package repair

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/memory"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

func TestFMEA010RepairWithoutProjectNeverGlobalizesMemory(t *testing.T) {
	ctx := context.Background()
	faultyHTML := `<!DOCTYPE html><html><body><p>No heading</p><button></button></body></html>`
	faultyCSS := `body { width: 2000px; }`
	memStore := memory.NewEpMemoryStore()
	optimizationPipeline := &engine.Pipeline{Collector: &mockRepairCollector{}, VerPolicy: verifier.DefaultPolicy()}
	finalPipeline := &engine.Pipeline{Collector: &mockL3RepairCollector{}, VerPolicy: verifier.DefaultPolicy()}
	finalGate := NewPipelineFinalGate(finalPipeline, NewPrivateHeldOutSuite(passHeldOutProbe{}))
	repairEngine := NewWithMemoryAndFinalGate(optimizationPipeline, finalGate, memStore)

	result, err := repairEngine.RunRepairLoop(ctx, RepairLoopRequest{
		RunID: "fmea010-unscoped-repair",
		HTML: faultyHTML,
		CSS: faultyCSS,
		Profile: design.FindProfile("saas-modern"),
		ProtectedAxes: []string{"accessibility", "responsive", "typography", "interaction"},
		MaxIterations: 3,
		RiskClass: RepairRiskCritical,
	})
	if err != nil { t.Fatalf("repair should not fail solely because long-term project scope is absent: %v", err) }
	if !result.Passed { t.Fatalf("expected independent repair pass: %#v", result.FinalGate) }
	if result.MemoryAdmitted || result.AdmittedAtoms != 0 { t.Fatalf("unscoped repair was admitted to long-term memory: %#v", result) }
	global, err := memStore.Query(ctx, memory.QueryRequest{Namespace: memory.NewGlobalDesignNamespace()})
	if err != nil { t.Fatal(err) }
	if global.Total != 0 { t.Fatalf("unscoped repair leaked %d atoms into global memory", global.Total) }
}
