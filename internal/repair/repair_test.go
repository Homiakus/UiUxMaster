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

	// Extract elements from HTML
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

	// Check width from CSS
	contentWidth := 1200.0
	if strings.Contains(cssStr, "width: 2000px") && !strings.Contains(cssStr, "max-width: 100vw") {
		contentWidth = 2000.0
	}
	packet.Documents = []evidence.DocumentMetrics{
		{
			ContentWidth:  contentWidth,
			ContentHeight: 800.0,
		},
	}

	return packet, nil
}

func TestHostRepairEngine_EndToEndAutonomousRepair(t *testing.T) {
	pipeline := &engine.Pipeline{
		Collector: &mockRepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	repairEngine := New(pipeline)

	faultyHTML := `<!DOCTYPE html>
<html>
<head><title>Test App</title></head>
<body>
  <div>
    <button id="cta-btn"></button>
  </div>
</body>
</html>`

	faultyCSS := `
body {
  width: 2000px;
}
`

	result, err := repairEngine.RunRepairLoop(context.Background(), RepairLoopRequest{
		RunID:         "repair-test-1",
		HTML:          faultyHTML,
		CSS:           faultyCSS,
		Profile:       design.FindProfile("saas-modern"),
		ProtectedAxes: []string{"accessibility", "responsive", "typography"},
		MaxIterations: 3,
	})
	if err != nil {
		t.Fatalf("repair loop failed: %v", err)
	}

	t.Logf("Repair loop result: %s", result.Summary)

	if result.InitialFindings == 0 {
		t.Errorf("expected initial findings > 0")
	}
	if len(result.PatchesApplied) == 0 {
		t.Errorf("expected patches to be applied")
	}
	if result.FinalFindings >= result.InitialFindings {
		t.Errorf("expected final findings (%d) < initial findings (%d)", result.FinalFindings, result.InitialFindings)
	}
	if !result.Passed {
		t.Errorf("expected repair loop to pass re-verification and constraints")
	}
}
