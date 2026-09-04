package playwright_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
)

func TestValidateScenario(t *testing.T) {
	tests := []struct {
		name      string
		scenario  playwright.Scenario
		expectErr string
	}{
		{
			name:      "empty id",
			scenario:  playwright.Scenario{},
			expectErr: "scenario ID must not be empty",
		},
		{
			name:      "no actions",
			scenario:  playwright.Scenario{ID: "sc-1"},
			expectErr: "scenario must contain at least one action",
		},
		{
			name: "click missing selector",
			scenario: playwright.Scenario{
				ID: "sc-2",
				Actions: []playwright.ScenarioAction{
					{Kind: playwright.ActionClick},
				},
			},
			expectErr: "requires a selector",
		},
		{
			name: "fill missing selector",
			scenario: playwright.Scenario{
				ID: "sc-3",
				Actions: []playwright.ScenarioAction{
					{Kind: playwright.ActionFill, Value: "hello"},
				},
			},
			expectErr: "requires a selector",
		},
		{
			name: "press missing key",
			scenario: playwright.Scenario{
				ID: "sc-4",
				Actions: []playwright.ScenarioAction{
					{Kind: playwright.ActionPress},
				},
			},
			expectErr: "requires a key value",
		},
		{
			name: "unsupported action kind",
			scenario: playwright.Scenario{
				ID: "sc-5",
				Actions: []playwright.ScenarioAction{
					{Kind: "teleport"},
				},
			},
			expectErr: "unsupported kind",
		},
		{
			name: "valid multi-step scenario",
			scenario: playwright.Scenario{
				ID: "sc-valid",
				Actions: []playwright.ScenarioAction{
					{Kind: playwright.ActionClick, Selector: "#nav-btn"},
					{Kind: playwright.ActionHover, Selector: ".menu-item"},
					{Kind: playwright.ActionFill, Selector: "input#search", Value: "query"},
					{Kind: playwright.ActionPress, Value: "Enter"},
					{Kind: playwright.ActionWait, Duration: 100 * time.Millisecond},
					{Kind: playwright.ActionSelect, Selector: "select#tier", Value: "pro"},
					{Kind: playwright.ActionCheck, Selector: "input#agree"},
					{Kind: playwright.ActionUncheck, Selector: "input#newsletter"},
					{Kind: playwright.ActionScroll, Value: "0,500"},
					{Kind: playwright.ActionResize, Value: "1440x900"},
				},
			},
			expectErr: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := playwright.ValidateScenario(tc.scenario)
			if tc.expectErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tc.expectErr) {
					t.Fatalf("expected error containing %q, got %v", tc.expectErr, err)
				}
			}
		})
	}
}

func TestDefaultDeterministicControls(t *testing.T) {
	dc := playwright.DefaultDeterministicControls()
	if !dc.PauseAnimations {
		t.Errorf("expected PauseAnimations to be true")
	}
	if !dc.FreezeClock {
		t.Errorf("expected FreezeClock to be true")
	}
	if !dc.ReducedMotion {
		t.Errorf("expected ReducedMotion to be true")
	}
	if dc.DeviceScaleFactor != 1.0 {
		t.Errorf("expected DeviceScaleFactor 1.0, got %f", dc.DeviceScaleFactor)
	}
	if dc.Timezone != "UTC" || dc.Locale != "en-US" {
		t.Errorf("expected UTC / en-US, got %s / %s", dc.Timezone, dc.Locale)
	}
}

func TestRunScenario_InvalidScenarioFailsFast(t *testing.T) {
	runner := &playwright.MockRunner{}
	adapter := playwright.New(playwright.Config{Runner: runner})

	invalidScenario := playwright.Scenario{
		ID: "invalid-sc",
		Actions: []playwright.ScenarioAction{
			{Kind: "unknown_action"},
		},
	}

	_, err := adapter.RunScenario(context.Background(), playwright.TruthPathRequest{}, invalidScenario)
	if err == nil {
		t.Fatal("expected error for invalid scenario, got nil")
	}
	if !strings.Contains(err.Error(), "invalid scenario") {
		t.Fatalf("expected 'invalid scenario' error, got %v", err)
	}
}

func TestRunScenario_FullPlaythroughWithEvidence(t *testing.T) {
	mockResp := playwright.WorkerResponse{
		Success:      true,
		URL:          "http://localhost:3000/checkout",
		AriaSnapshot: "- dialog \"Order Confirmation\"\n  - text \"Thank you!\"",
		Accessibility: []evidence.AccessibilityNode{
			{ID: "dlg", Role: "dialog", Name: "Order Confirmation"},
			{ID: "msg", Role: "status", Name: "Thank you!"},
		},
		Elements: []evidence.ElementRef{
			{ID: "conf-modal", Role: "dialog", Visible: true},
		},
		Latency: evidence.RuntimeLatency{
			TotalMS: 120.0,
		},
	}

	runner := &playwright.MockRunner{Response: mockResp}
	adapter := playwright.New(playwright.Config{
		DefaultBrowser: playwright.BrowserChromium,
		Runner:         runner,
	})

	scenario := playwright.Scenario{
		ID: "checkout-flow",
		Actions: []playwright.ScenarioAction{
			{Kind: playwright.ActionFill, Selector: "#cc-num", Value: "424242424242"},
			{Kind: playwright.ActionClick, Selector: "#pay-button"},
			{Kind: playwright.ActionWait, Selector: "#order-confirmation"},
		},
	}

	req := playwright.TruthPathRequest{
		RunID:           "run-scenario-playthrough",
		Browser:         playwright.BrowserChromium,
		PauseAnimations: true,
		FreezeClock:     true,
	}

	packet, err := adapter.RunScenario(context.Background(), req, scenario)
	if err != nil {
		t.Fatalf("unexpected RunScenario error: %v", err)
	}

	if packet.Scenario != "checkout-flow" {
		t.Errorf("scenario ID = %s, want checkout-flow", packet.Scenario)
	}
	if len(packet.Accessibility) != 2 {
		t.Errorf("accessibility nodes = %d, want 2", len(packet.Accessibility))
	}
	if len(packet.Elements) != 1 {
		t.Errorf("elements = %d, want 1", len(packet.Elements))
	}
	if runner.LastReq.Scenario == nil || len(runner.LastReq.Scenario.Actions) != 3 {
		t.Fatalf("expected 3 actions dispatched to runner")
	}
}
