package playwright

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// Config configures the Playwright TruthPath adapter.
type Config struct {
	DefaultBrowser BrowserFamily
	Headless       bool
	WorkerCmd      string
	WorkerArgs     []string
	WorkerScript   string
	Timeout        time.Duration
	Runner         CommandRunner
}

// Adapter implements TruthPathAdapter for Playwright execution.
type Adapter struct {
	cfg Config
}

var _ TruthPathAdapter = (*Adapter)(nil)

// New creates an initialized TruthPath Adapter.
func New(cfg Config) *Adapter {
	if cfg.DefaultBrowser == "" {
		cfg.DefaultBrowser = BrowserChromium
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	if cfg.WorkerCmd == "" {
		cfg.WorkerCmd = "node"
	}
	if cfg.Runner == nil {
		cfg.Runner = &OSCommandRunner{}
	}
	return &Adapter{
		cfg: cfg,
	}
}

// Capabilities returns the capability profile of the Playwright TruthPath adapter.
func (a *Adapter) Capabilities() TruthPathCapabilities {
	return TruthPathCapabilities{
		Name:             "playwright-truthpath",
		Version:          "1.0.0",
		Browsers:         []BrowserFamily{BrowserChromium, BrowserFirefox, BrowserWebKit},
		CleanState:       true,
		SupportsARIA:     true,
		SupportsFonts:    true,
		SupportsScenario: true,
		SupportsROI:      true,
	}
}

// Capture executes clean-state page render and evidence capture.
func (a *Adapter) Capture(ctx context.Context, req TruthPathRequest) (evidence.Packet, error) {
	if err := ctx.Err(); err != nil {
		return evidence.Packet{}, err
	}
	if req.Browser == "" {
		req.Browser = a.cfg.DefaultBrowser
	}
	if req.Viewport.Width == 0 {
		req.Viewport.Width = 1280
	}
	if req.Viewport.Height == 0 {
		req.Viewport.Height = 720
	}

	workerReq := WorkerRequest{
		Command:            "capture",
		Browser:            string(req.Browser),
		URL:                req.URL,
		HTML:               string(req.HTML),
		CSS:                string(req.CSS),
		BaseURL:            req.BaseURL,
		Viewport:           req.Viewport,
		Region:             req.Region,
		CapturePixels:      req.CapturePixels,
		CaptureARIA:        req.CaptureARIA,
		CaptureFonts:       req.CaptureFonts,
		CaptureDiagnostics: req.CaptureDiagnostics,
		CaptureLayout:      req.CaptureLayout,
		PauseAnimations:    req.PauseAnimations,
		FreezeClock:        req.FreezeClock,
	}

	return a.execute(ctx, req, workerReq)
}

// RunScenario executes a deterministic multi-step interaction playthrough.
func (a *Adapter) RunScenario(ctx context.Context, req TruthPathRequest, scenario Scenario) (evidence.Packet, error) {
	if err := ctx.Err(); err != nil {
		return evidence.Packet{}, err
	}
	if err := ValidateScenario(scenario); err != nil {
		return evidence.Packet{}, fmt.Errorf("playwright: invalid scenario: %w", err)
	}
	if req.Browser == "" {
		req.Browser = a.cfg.DefaultBrowser
	}
	if req.Viewport.Width == 0 {
		req.Viewport.Width = 1280
	}
	if req.Viewport.Height == 0 {
		req.Viewport.Height = 720
	}

	req.Scenario = &scenario
	workerReq := WorkerRequest{
		Command:            "scenario",
		Browser:            string(req.Browser),
		URL:                req.URL,
		HTML:               string(req.HTML),
		CSS:                string(req.CSS),
		BaseURL:            req.BaseURL,
		Viewport:           req.Viewport,
		Region:             req.Region,
		CapturePixels:      req.CapturePixels,
		CaptureARIA:        req.CaptureARIA,
		CaptureFonts:       req.CaptureFonts,
		CaptureDiagnostics: req.CaptureDiagnostics,
		CaptureLayout:      req.CaptureLayout,
		PauseAnimations:    req.PauseAnimations,
		FreezeClock:        req.FreezeClock,
		Scenario:           &scenario,
	}

	return a.execute(ctx, req, workerReq)
}

// Close cleans up resources if any.
func (a *Adapter) Close() error {
	return nil
}

func (a *Adapter) execute(ctx context.Context, req TruthPathRequest, workerReq WorkerRequest) (evidence.Packet, error) {
	reqData, err := json.Marshal(workerReq)
	if err != nil {
		return evidence.Packet{}, fmt.Errorf("failed to marshal worker request: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()

	start := time.Now()
	args := a.cfg.WorkerArgs
	if a.cfg.WorkerScript != "" {
		args = append(args, a.cfg.WorkerScript)
	}

	out, err := a.cfg.Runner.Run(ctxTimeout, a.cfg.WorkerCmd, args, reqData)
	if err != nil {
		return evidence.Packet{}, fmt.Errorf("playwright execution failed: %w", err)
	}

	resp, err := parseWorkerResponse(out)
	if err != nil {
		return evidence.Packet{}, err
	}

	packet := MapWorkerResponseToPacket(req, resp, time.Since(start))
	if req.Scenario != nil {
		packet.Scenario = req.Scenario.ID
	}
	return packet, nil
}

// MockRunner is a helper for unit testing TruthPath adapter without running node.
type MockRunner struct {
	Response WorkerResponse
	Err      error
	LastReq  WorkerRequest
}

func (m *MockRunner) Run(_ context.Context, _ string, _ []string, stdin []byte) ([]byte, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if len(stdin) > 0 {
		_ = json.Unmarshal(stdin, &m.LastReq)
	}
	return json.Marshal(m.Response)
}

var ErrBrowserUnavailable = errors.New("playwright browser unavailable")
