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
	Kind     string        `json:"kind"` // click, fill, hover, scroll, resize, wait, press, ...
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
	RunID              string            `json:"run_id"`
	Browser            BrowserFamily     `json:"browser"`
	URL                string            `json:"url,omitempty"`
	HTML               []byte            `json:"html,omitempty"`
	CSS                []byte            `json:"css,omitempty"`
	BaseURL            string            `json:"base_url,omitempty"`
	Viewport           evidence.Viewport `json:"viewport"`
	Region             *evidence.Rect    `json:"region,omitempty"`
	CapturePixels      bool              `json:"capture_pixels"`
	CaptureARIA        bool              `json:"capture_aria"`
	CaptureFonts       bool              `json:"capture_fonts"`
	CaptureDiagnostics bool              `json:"capture_diagnostics"`
	CaptureLayout      bool              `json:"capture_layout"`
	PauseAnimations    bool              `json:"pause_animations"`
	FreezeClock        bool              `json:"freeze_clock"`
	BaselineRGBA       []byte            `json:"baseline_rgba,omitempty"`
	Scenario           *Scenario         `json:"scenario,omitempty"`
}

// BrowserReadiness is an attestation for one browser engine discovered by the
// worker probe. Browsers are advertised as usable only when Ready is true.
type BrowserReadiness struct {
	Browser        BrowserFamily `json:"browser"`
	Ready          bool          `json:"ready"`
	Version        string        `json:"version,omitempty"`
	ExecutablePath string        `json:"executable_path,omitempty"`
	Error          string        `json:"error,omitempty"`
}

// TruthPathFeatures reports features implemented and attested by the exact
// worker protocol that answered the probe.
type TruthPathFeatures struct {
	CleanState       bool `json:"clean_state"`
	SupportsARIA     bool `json:"supports_aria"`
	SupportsFonts    bool `json:"supports_fonts"`
	SupportsScenario bool `json:"supports_scenario"`
	SupportsROI      bool `json:"supports_roi"`
}

// TruthPathReadiness is the runtime attestation returned by Probe. Ready is
// true only when the worker identity/version and at least one launchable browser
// engine have been validated.
type TruthPathReadiness struct {
	Ready             bool               `json:"ready"`
	WorkerVersion     string             `json:"worker_version,omitempty"`
	PlaywrightVersion string             `json:"playwright_version,omitempty"`
	Browsers           []BrowserReadiness `json:"browsers,omitempty"`
	Features           TruthPathFeatures  `json:"features"`
	Error              string             `json:"error,omitempty"`
}

// TruthPathCapabilities reports only capabilities proven by the most recent
// successful readiness probe. A newly constructed adapter is deliberately not
// L3-capable until Probe succeeds.
type TruthPathCapabilities struct {
	Name              string                    `json:"name"`
	Version           string                    `json:"version"`
	Ready             bool                      `json:"ready"`
	WorkerVersion     string                    `json:"worker_version,omitempty"`
	PlaywrightVersion string                    `json:"playwright_version,omitempty"`
	Browsers          []BrowserFamily           `json:"browsers"`
	BrowserVersions   map[BrowserFamily]string  `json:"browser_versions,omitempty"`
	CleanState        bool                      `json:"clean_state"`
	SupportsARIA      bool                      `json:"supports_aria"`
	SupportsFonts     bool                      `json:"supports_fonts"`
	SupportsScenario  bool                      `json:"supports_scenario"`
	SupportsROI       bool                      `json:"supports_roi"`
}

// TruthPathAdapter is the vendor-neutral contract for Tier L3 clean-state verification.
// Core domain packages consume this interface without importing Playwright or Node types.
type TruthPathAdapter interface {
	Probe(ctx context.Context) (TruthPathReadiness, error)
	Capture(ctx context.Context, req TruthPathRequest) (evidence.Packet, error)
	RunScenario(ctx context.Context, req TruthPathRequest, scenario Scenario) (evidence.Packet, error)
	Capabilities() TruthPathCapabilities
	Close() error
}
