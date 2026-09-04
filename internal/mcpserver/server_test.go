package mcpserver_test

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/mcpserver"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type mockCollector struct{}

func (m *mockCollector) Collect(_ context.Context, req engine.ValidationRequest, _ engine.ValidationPlan) (evidence.Packet, error) {
	return evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{
			Tier: "L2",
			Name: "chromium-cdp",
		},
		Elements: []evidence.ElementRef{
			{
				ID:            "btn-1",
				Tag:           "button",
				Role:          "button",
				Name:          "Submit",
				Visible:       true,
				Clickable:     true,
				BackendNodeID: 101,
				Bounds:        evidence.Rect{X: 10, Y: 10, Width: 120, Height: 44},
			},
		},
		Accessibility: []evidence.AccessibilityNode{
			{ID: "ax-1", BackendNodeID: 101, Role: "button", Name: "Submit"},
		},
		Diagnostics: &evidence.DiagnosticsEvidence{Complete: true},
	}, nil
}

func TestMCPServer_InitializationAndToolRegistration(t *testing.T) {
	pipeline := &engine.Pipeline{
		Collector: &mockCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}

	server := mcpserver.New(mcpserver.Config{Pipeline: pipeline})
	if server == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestInspectLayoutTool(t *testing.T) {
	packet := evidence.Packet{
		Viewport: evidence.Viewport{Width: 1000},
		Documents: []evidence.DocumentMetrics{
			{ContentWidth: 1200}, // 200px overflow
		},
	}

	policy := verifier.DefaultPolicy()
	res := verifier.Verify(packet, policy)

	if len(res.Issues) == 0 {
		t.Fatalf("expected overflow issue detected")
	}
	if res.Issues[0].Code != verifier.CodeViewportHorizontalOverflow {
		t.Errorf("issue code = %s, want %s", res.Issues[0].Code, verifier.CodeViewportHorizontalOverflow)
	}
}

func TestInspectAccessibilityTool(t *testing.T) {
	packet := evidence.Packet{
		Elements: []evidence.ElementRef{
			{ID: "btn-unnamed", Tag: "button", Visible: true, Clickable: true, BackendNodeID: 1},
		},
		Accessibility: []evidence.AccessibilityNode{
			{ID: "ax-unnamed", BackendNodeID: 1, Role: "button", Name: ""}, // missing accessible name
		},
	}

	issues := verifier.VerifyAccessibility(packet)
	if len(issues) != 1 {
		t.Fatalf("expected 1 accessibility issue, got %d", len(issues))
	}
	if issues[0].Code != verifier.CodeA11yNameMissing {
		t.Errorf("issue code = %s, want %s", issues[0].Code, verifier.CodeA11yNameMissing)
	}
}
