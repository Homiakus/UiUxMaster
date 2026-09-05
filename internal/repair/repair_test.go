package repair

import (
	"context"
	"strings"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type mockRepairCollector struct{}

func (m *mockRepairCollector) Collect(_ context.Context, req engine.ValidationRequest, _ engine.ValidationPlan) (evidence.Packet, error) {
	htmlStr := string(req.HTML)
	cssStr := string(req.CSS)

	packet := evidence.Packet{
		RunID: req.RunID,
		Viewport: evidence.Viewport{
			Width:  1280,
			Height: 800,
		},
		Renderer: evidence.RendererRef{
			Tier: "L2",
			Name: "mock-fastcdp",
		},
	}

	var elements []evidence.ElementRef
	if strings.Contains(strings.ToLower(htmlStr), "<h1") {
		elements = append(elements, evidence.ElementRef{
			ID:      "h1-main",
			Tag:     "h1",
			Role:    "heading",
			Visible: true,
			Name:    "Main Page Title",
		})
	}

	if strings.Contains(strings.ToLower(htmlStr), "<button") {
		btnName := ""
		if strings.Contains(htmlStr, `aria-label="Action Button"`) {
			btnName = "Action Button"
		}
		elements = append(elements, evidence.ElementRef{
			ID:        "cta-btn",
			Tag:       "button",
			Role:      "button",
			Visible:   true,
			Clickable: true,
			Name:      btnName,
		})
	}
	packet.Elements = elements

	contentWidth := 1200.0
	if strings.Contains(cssStr, "width: 2000px") && !strings.Contains(cssStr, "max-width: 100vw") {
		contentWidth = 2000.0
	}
	packet.Documents = []evidence.DocumentMetrics{{
		ContentWidth:  contentWidth,
		ContentHeight: 800.0,
	}}

	return packet, nil
}

func repairFixture() (string, string) {
	return `<!DOCTYPE html>
<html>
<head><title>Test App</title></head>
<body>
  <div>
    <button id="cta-btn"></button>
  </div>
</body>
</html>`, `
body {
  width: 2000px;
}
`
}

func TestHostRepairEngine_OptimizationCannotSelfApprove(t *testing.T) {
	pipeline := &engine.Pipeline{
		Collector: &mockRepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	repairEngine := New(pipeline)
	faultyHTML, faultyCSS := repairFixture()

	result, err := repairEngine.RunRepairLoop(context.Background(), RepairLoopRequest{
		RunID:         "repair-test-self-approval",
		HTML:          faultyHTML,
		CSS:           faultyCSS,
		Profile:       design.FindProfile("saas-modern"),
		ProtectedAxes: []string{"accessibility", "responsive", "typography"},
		MaxIterations: 3,
	})
	if err != nil {
		t.Fatalf("repair loop failed: %v", err)
	}

	if result.InitialFindings == 0 {
		t.Fatalf("expected initial findings > 0")
	}
	if len(result.PatchesApplied) == 0 {
		t.Fatalf("expected patches to be proposed")
	}
	if result.FinalFindings >= result.InitialFindings {
		t.Fatalf("expected optimization findings to improve: initial=%d final=%d", result.InitialFindings, result.FinalFindings)
	}
	if !result.CandidateImproved {
		t.Fatalf("expected candidate to improve under optimization-side scoring")
	}
	if result.Passed {
		t.Fatalf("optimization pipeline must never grant completion without an independent FinalGate")
	}
	if !result.EscalationRequired {
		t.Fatalf("missing independent FinalGate must require escalation")
	}
	if len(result.FinalGate.ReasonCodes) == 0 || result.FinalGate.ReasonCodes[0] != "independent_final_gate_unconfigured" {
		t.Fatalf("unexpected final gate reasons: %#v", result.FinalGate.ReasonCodes)
	}
}
