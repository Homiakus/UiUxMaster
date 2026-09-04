package playwright

import (
	"context"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// BrowserFamily represents a supported browser engine in the Playwright TruthPath runtime.
type BrowserFamily string

const (
	BrowserChromium BrowserFamily = "chromium"
	BrowserFirefox  BrowserFamily = "firefox"
	BrowserWebKit   BrowserFamily = "webkit"
)

// ScenarioAction describes an interactive step in an L3 verification playthrough.
type ScenarioAction struct {
	Kind     string        `json:"kind"` // "click", "fill", "hover", "scroll", "resize", "wait", "press"
	Selector string        `json:"selector,omitempty"`
	Value    string        `json:"value,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

// Scenario encapsulates a deterministic interaction sequence.
type Scenario struct {
	ID      string           `json:"id"`
	Actions []ScenarioAction `json:"actions"`
}

// TruthPathRequest describes an independent clean-state verification request.
type TruthPathRequest struct {
	RunID              string                `json:"run_id"`
	Browser            BrowserFamily         `json:"browser"`
	URL                string                `json:"url,omitempty"`
	HTML               []byte                `json:"html,omitempty"`
	CSS                []byte                `json:"css,omitempty"`
	BaseURL            string                `json:"base_url,omitempty"`
	Viewport           evidence.Viewport     `json:"viewport"`
	Region             *evidence.Rect        `json:"region,omitempty"`
	CapturePixels      bool                  `json:"capture_pixels"`
	CaptureARIA        bool                  `json:"capture_aria"`
	CaptureFonts       bool                  `json:"capture_fonts"`
	CaptureDiagnostics bool                  `json:"capture_diagnostics"`
	CaptureLayout      bool                  `json:"capture_layout"`
	PauseAnimations    bool                  `json:"pause_animations"`
	FreezeClock        bool                  `json:"freeze_clock"`
	BaselineRGBA       []byte                `json:"baseline_rgba,omitempty"`
	Scenario           *Scenario             `json:"scenario,omitempty"`
}

// TruthPathCapabilities reports the browser support and clean-state properties of TruthPath.
type TruthPathCapabilities struct {
	Name             string          `json:"name"`
	Version          string          `json:"version"`
	Browsers         []BrowserFamily `json:"browsers"`
	CleanState       bool            `json:"clean_state"`
	SupportsARIA     bool            `json:"supports_aria"`
	SupportsFonts    bool            `json:"supports_fonts"`
	SupportsScenario bool            `json:"supports_scenario"`
	SupportsROI      bool            `json:"supports_roi"`
}

// TruthPathAdapter is the vendor-neutral contract for Tier L3 clean-state verification.
// Core domain packages consume this interface without importing Playwright or Node types.
type TruthPathAdapter interface {
	Capture(ctx context.Context, req TruthPathRequest) (evidence.Packet, error)
	RunScenario(ctx context.Context, req TruthPathRequest, scenario Scenario) (evidence.Packet, error)
	Capabilities() TruthPathCapabilities
	Close() error
}
