package repair

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/runtime/dispatcher"
	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type realTruthPathRepairProbe struct{}

func (realTruthPathRepairProbe) Evaluate(_ context.Context, req HeldOutEvaluationRequest) error {
	if req.Candidate.Renderer.Tier != "L3" {
		return errors.New("held-out gate requires L3 candidate evidence")
	}
	foundHeading := false
	foundNamedButton := false
	for _, element := range req.Candidate.Elements {
		if element.Tag == "h1" || element.Role == "heading" {
			foundHeading = true
		}
		if element.Role == "button" && element.Name == "Action Button" {
			foundNamedButton = true
		}
	}
	if !foundHeading {
		return errors.New("held-out semantic probe: repaired page lacks primary heading")
	}
	if !foundNamedButton {
		return errors.New("held-out semantic probe: repaired button lacks expected accessible name")
	}
	return nil
}

func TestRepairFinalGateRealChromium(t *testing.T) {
	if os.Getenv("UIUX_TRUTHPATH_INTEGRATION") != "1" {
		t.Skip("set UIUX_TRUTHPATH_INTEGRATION=1 to run real repair final-gate verification")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	adapter := playwright.New(playwright.Config{
		WorkerScript: filepath.Join("..", "runtime", "playwright", "worker", "worker.cjs"),
		ProbeTimeout: 2 * time.Minute,
		Timeout:      1 * time.Minute,
	})
	readiness, err := adapter.Probe(ctx)
	if err != nil {
		t.Fatalf("TruthPath probe: %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("TruthPath not ready: %#v", readiness)
	}

	optimizationPipeline := &engine.Pipeline{
		Collector: &mockRepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	truthCollector := playwright.NewCollector(adapter, playwright.BrowserChromium)
	finalDispatcher := dispatcher.New(dispatcher.Config{L3Collector: truthCollector})
	finalPipeline := &engine.Pipeline{
		Collector: finalDispatcher,
		VerPolicy: verifier.DefaultPolicy(),
	}
	finalGate := NewPipelineFinalGate(finalPipeline, NewPrivateHeldOutSuite(realTruthPathRepairProbe{}))
	repairEngine := NewWithFinalGate(optimizationPipeline, finalGate)
	faultyHTML, faultyCSS := repairFixture()

	result, err := repairEngine.RunRepairLoop(ctx, RepairLoopRequest{
		RunID:         "fmea009-real-truthpath-final-gate",
		HTML:          faultyHTML,
		CSS:           faultyCSS,
		Profile:       design.FindProfile("saas-modern"),
		ProtectedAxes: []string{"accessibility", "responsive", "typography", "interaction"},
		RiskClass:     RepairRiskCritical,
		MaxIterations: 3,
	})
	if err != nil {
		t.Fatalf("RunRepairLoop: %v", err)
	}
	if !result.CandidateImproved {
		t.Fatalf("optimization did not produce an improved candidate")
	}
	if !result.Passed || !result.FinalGate.Independent {
		t.Fatalf("real independent final gate did not authorize completion: %#v", result.FinalGate)
	}
	if result.FinalGate.EvidenceTier != "L3" || result.FinalGate.BaselineEvidenceTier != "L3" {
		t.Fatalf("final gate evidence tiers = candidate %q baseline %q", result.FinalGate.EvidenceTier, result.FinalGate.BaselineEvidenceTier)
	}
	if result.FinalGate.HeldOut.Total != 1 || result.FinalGate.HeldOut.Failed != 0 || result.Metrics.HeldOutEscapeRate != 0 {
		t.Fatalf("held-out result = %#v metrics=%#v", result.FinalGate.HeldOut, result.Metrics)
	}
	if result.EscalationRequired {
		t.Fatalf("successful real TruthPath final gate must not require escalation")
	}
}
