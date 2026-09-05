package playwright

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

const (
	// WorkerProtocolVersion is the Go/Node protocol contract implemented by the
	// checked-in worker. A mismatch is a capability failure, never a warning.
	WorkerProtocolVersion = "1.0.0"
	// PinnedPlaywrightVersion is the exact runtime version validated by this
	// adapter/worker pair. Runtime drift requires explicit revalidation.
	PinnedPlaywrightVersion = "1.62.1"
)

var (
	ErrTruthPathUnavailable      = errors.New("playwright truthpath unavailable")
	ErrWorkerVersionMismatch     = errors.New("playwright worker version mismatch")
	ErrPlaywrightVersionMismatch = errors.New("playwright runtime version mismatch")
	ErrBrowserUnavailable        = errors.New("playwright browser unavailable")
	ErrRuntimeIdentityChanged    = errors.New("playwright runtime identity changed after readiness probe")
)

// Config configures the Playwright TruthPath adapter.
type Config struct {
	DefaultBrowser            BrowserFamily
	Headless                  bool
	WorkerCmd                 string
	WorkerArgs                []string
	WorkerScript              string
	Timeout                   time.Duration
	ProbeTimeout              time.Duration
	ExpectedWorkerVersion     string
	ExpectedPlaywrightVersion string
	Runner                    CommandRunner
}

// Adapter implements TruthPathAdapter for Playwright execution.
type Adapter struct {
	cfg Config

	mu        sync.RWMutex
	readiness TruthPathReadiness
	probed    bool
}

var _ TruthPathAdapter = (*Adapter)(nil)

// New creates an initialized TruthPath Adapter. Construction does not imply L3
// readiness: Capabilities remains fail-closed until Probe validates the exact
// worker/runtime/browser environment.
func New(cfg Config) *Adapter {
	if cfg.DefaultBrowser == "" {
		cfg.DefaultBrowser = BrowserChromium
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.ProbeTimeout == 0 {
		cfg.ProbeTimeout = 90 * time.Second
	}
	if cfg.WorkerCmd == "" {
		cfg.WorkerCmd = "node"
	}
	if cfg.WorkerScript == "" {
		cfg.WorkerScript = defaultWorkerScriptPath()
	}
	if cfg.ExpectedWorkerVersion == "" {
		cfg.ExpectedWorkerVersion = WorkerProtocolVersion
	}
	if cfg.ExpectedPlaywrightVersion == "" {
		cfg.ExpectedPlaywrightVersion = PinnedPlaywrightVersion
	}
	if cfg.Runner == nil {
		cfg.Runner = &OSCommandRunner{}
	}
	return &Adapter{cfg: cfg}
}

// Probe validates the worker file/command, protocol identity, exact Playwright
// version, implemented feature set, and launchability/version of each installed
// browser engine. Capabilities is derived exclusively from this attestation.
func (a *Adapter) Probe(ctx context.Context) (TruthPathReadiness, error) {
	if err := ctx.Err(); err != nil {
		return TruthPathReadiness{}, err
	}
	if a == nil {
		return TruthPathReadiness{}, ErrTruthPathUnavailable
	}

	if err := a.validateWorkerEntrypoint(); err != nil {
		return a.recordUnavailable(err), err
	}

	payload, err := json.Marshal(WorkerRequest{Command: "probe"})
	if err != nil {
		return a.recordUnavailable(err), err
	}
	probeCtx, cancel := context.WithTimeout(ctx, a.cfg.ProbeTimeout)
	defer cancel()

	out, err := a.cfg.Runner.Run(probeCtx, a.cfg.WorkerCmd, a.workerArgs(), payload)
	if err != nil {
		wrapped := fmt.Errorf("%w: probe execution: %v", ErrTruthPathUnavailable, err)
		return a.recordUnavailable(wrapped), wrapped
	}
	resp, err := parseProbeResponse(out)
	if err != nil {
		wrapped := fmt.Errorf("%w: %v", ErrTruthPathUnavailable, err)
		return a.recordUnavailable(wrapped), wrapped
	}
	if resp.WorkerVersion != a.cfg.ExpectedWorkerVersion {
		err := fmt.Errorf("%w: got %q want %q", ErrWorkerVersionMismatch, resp.WorkerVersion, a.cfg.ExpectedWorkerVersion)
		return a.recordUnavailable(err), err
	}
	if resp.PlaywrightVersion != a.cfg.ExpectedPlaywrightVersion {
		err := fmt.Errorf("%w: got %q want %q", ErrPlaywrightVersionMismatch, resp.PlaywrightVersion, a.cfg.ExpectedPlaywrightVersion)
		return a.recordUnavailable(err), err
	}

	readyCount := 0
	for i := range resp.Browsers {
		if resp.Browsers[i].Ready && resp.Browsers[i].Version != "" {
			readyCount++
		} else if resp.Browsers[i].Ready {
			resp.Browsers[i].Ready = false
			resp.Browsers[i].Error = "browser launched without version attestation"
		}
	}
	if readyCount == 0 {
		err := fmt.Errorf("%w: no browser engine passed launch/version probe", ErrTruthPathUnavailable)
		readiness := TruthPathReadiness{
			WorkerVersion:     resp.WorkerVersion,
			PlaywrightVersion: resp.PlaywrightVersion,
			Browsers:           cloneBrowserReadiness(resp.Browsers),
			Features:           resp.Features,
			Error:              err.Error(),
		}
		a.setReadiness(readiness, true)
		return readiness, err
	}

	readiness := TruthPathReadiness{
		Ready:             true,
		WorkerVersion:     resp.WorkerVersion,
		PlaywrightVersion: resp.PlaywrightVersion,
		Browsers:           cloneBrowserReadiness(resp.Browsers),
		Features:           resp.Features,
	}
	a.setReadiness(readiness, true)
	return readiness, nil
}

// Capabilities returns only runtime-attested capabilities. Before a successful
// probe it deliberately advertises no browsers and no L3 proof features.
func (a *Adapter) Capabilities() TruthPathCapabilities {
	caps := TruthPathCapabilities{
		Name:            "playwright-truthpath",
		Version:         WorkerProtocolVersion,
		Browsers:        []BrowserFamily{},
		BrowserVersions: map[BrowserFamily]string{},
	}
	if a == nil {
		return caps
	}

	a.mu.RLock()
	readiness := cloneReadiness(a.readiness)
	a.mu.RUnlock()
	if !readiness.Ready {
		return caps
	}

	caps.Ready = true
	caps.WorkerVersion = readiness.WorkerVersion
	caps.PlaywrightVersion = readiness.PlaywrightVersion
	caps.CleanState = readiness.Features.CleanState
	caps.SupportsARIA = readiness.Features.SupportsARIA
	caps.SupportsFonts = readiness.Features.SupportsFonts
	caps.SupportsScenario = readiness.Features.SupportsScenario
	caps.SupportsROI = readiness.Features.SupportsROI
	for _, browser := range readiness.Browsers {
		if !browser.Ready || browser.Version == "" {
			continue
		}
		caps.Browsers = append(caps.Browsers, browser.Browser)
		caps.BrowserVersions[browser.Browser] = browser.Version
	}
	return caps
}

// Capture executes clean-state page render and evidence capture.
func (a *Adapter) Capture(ctx context.Context, req TruthPathRequest) (evidence.Packet, error) {
	if err := ctx.Err(); err != nil {
		return evidence.Packet{}, err
	}
	if req.Browser == "" {
		req.Browser = a.cfg.DefaultBrowser
	}
	if err := a.ensureReady(ctx, req.Browser); err != nil {
		return evidence.Packet{}, err
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
	if err := a.ensureReady(ctx, req.Browser); err != nil {
		return evidence.Packet{}, err
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
func (a *Adapter) Close() error { return nil }

func (a *Adapter) ensureReady(ctx context.Context, browser BrowserFamily) error {
	a.mu.RLock()
	probed := a.probed
	readiness := cloneReadiness(a.readiness)
	a.mu.RUnlock()

	if !probed || !readiness.Ready {
		var err error
		readiness, err = a.Probe(ctx)
		if err != nil {
			return err
		}
	}
	for _, current := range readiness.Browsers {
		if current.Browser == browser {
			if current.Ready && current.Version != "" {
				return nil
			}
			return fmt.Errorf("%w: %s: %s", ErrBrowserUnavailable, browser, current.Error)
		}
	}
	return fmt.Errorf("%w: %s was not returned by readiness probe", ErrBrowserUnavailable, browser)
}

func (a *Adapter) execute(ctx context.Context, req TruthPathRequest, workerReq WorkerRequest) (evidence.Packet, error) {
	reqData, err := json.Marshal(workerReq)
	if err != nil {
		return evidence.Packet{}, fmt.Errorf("failed to marshal worker request: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()

	start := time.Now()
	out, err := a.cfg.Runner.Run(ctxTimeout, a.cfg.WorkerCmd, a.workerArgs(), reqData)
	if err != nil {
		return evidence.Packet{}, fmt.Errorf("playwright execution failed: %w", err)
	}

	resp, err := parseWorkerResponse(out)
	if err != nil {
		return evidence.Packet{}, err
	}
	if err := a.validateResponseIdentity(req.Browser, resp); err != nil {
		return evidence.Packet{}, err
	}

	packet := MapWorkerResponseToPacket(req, resp, time.Since(start))
	if req.Scenario != nil {
		packet.Scenario = req.Scenario.ID
	}
	return packet, nil
}

func (a *Adapter) validateResponseIdentity(browser BrowserFamily, resp WorkerResponse) error {
	a.mu.RLock()
	readiness := cloneReadiness(a.readiness)
	a.mu.RUnlock()

	if resp.WorkerVersion != readiness.WorkerVersion || resp.PlaywrightVersion != readiness.PlaywrightVersion {
		return fmt.Errorf("%w: worker/playwright probe=%s/%s capture=%s/%s", ErrRuntimeIdentityChanged,
			readiness.WorkerVersion, readiness.PlaywrightVersion, resp.WorkerVersion, resp.PlaywrightVersion)
	}
	for _, current := range readiness.Browsers {
		if current.Browser == browser && current.Ready {
			if resp.BrowserVersion != current.Version {
				return fmt.Errorf("%w: %s probe=%q capture=%q", ErrRuntimeIdentityChanged, browser, current.Version, resp.BrowserVersion)
			}
			return nil
		}
	}
	return fmt.Errorf("%w: %s not attested", ErrBrowserUnavailable, browser)
}

func (a *Adapter) validateWorkerEntrypoint() error {
	if a.cfg.WorkerCmd == "" || a.cfg.WorkerScript == "" {
		return fmt.Errorf("%w: worker command/script is empty", ErrTruthPathUnavailable)
	}
	if _, ok := a.cfg.Runner.(*OSCommandRunner); !ok {
		return nil
	}
	if _, err := exec.LookPath(a.cfg.WorkerCmd); err != nil {
		return fmt.Errorf("%w: worker command %q: %v", ErrTruthPathUnavailable, a.cfg.WorkerCmd, err)
	}
	info, err := os.Stat(a.cfg.WorkerScript)
	if err != nil {
		return fmt.Errorf("%w: worker script %q: %v", ErrTruthPathUnavailable, a.cfg.WorkerScript, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%w: worker script %q is a directory", ErrTruthPathUnavailable, a.cfg.WorkerScript)
	}
	return nil
}

func (a *Adapter) workerArgs() []string {
	args := append([]string(nil), a.cfg.WorkerArgs...)
	return append(args, a.cfg.WorkerScript)
}

func (a *Adapter) recordUnavailable(err error) TruthPathReadiness {
	readiness := TruthPathReadiness{Error: err.Error()}
	a.setReadiness(readiness, true)
	return readiness
}

func (a *Adapter) setReadiness(readiness TruthPathReadiness, probed bool) {
	a.mu.Lock()
	a.readiness = cloneReadiness(readiness)
	a.probed = probed
	a.mu.Unlock()
}

func cloneReadiness(in TruthPathReadiness) TruthPathReadiness {
	out := in
	out.Browsers = cloneBrowserReadiness(in.Browsers)
	return out
}

func cloneBrowserReadiness(in []BrowserReadiness) []BrowserReadiness {
	return append([]BrowserReadiness(nil), in...)
}

func defaultWorkerScriptPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(file), "worker", "worker.cjs")
}

// MockRunner is a helper for unit testing TruthPath adapter without running node.
type MockRunner struct {
	Response      WorkerResponse
	ProbeResponse *WorkerProbeResponse
	Err           error
	LastReq       WorkerRequest
}

func (m *MockRunner) Run(_ context.Context, _ string, _ []string, stdin []byte) ([]byte, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if len(stdin) > 0 {
		_ = json.Unmarshal(stdin, &m.LastReq)
	}
	if m.LastReq.Command == "probe" {
		probe := m.ProbeResponse
		if probe == nil {
			defaultProbe := WorkerProbeResponse{
				Success:           true,
				WorkerVersion:     WorkerProtocolVersion,
				PlaywrightVersion: PinnedPlaywrightVersion,
				Browsers: []BrowserReadiness{
					{Browser: BrowserChromium, Ready: true, Version: "mock-chromium"},
					{Browser: BrowserFirefox, Ready: true, Version: "mock-firefox"},
					{Browser: BrowserWebKit, Ready: true, Version: "mock-webkit"},
				},
				Features: TruthPathFeatures{CleanState: true, SupportsARIA: true, SupportsFonts: true, SupportsScenario: true, SupportsROI: true},
			}
			probe = &defaultProbe
		}
		return json.Marshal(probe)
	}

	resp := m.Response
	if resp.WorkerVersion == "" {
		resp.WorkerVersion = WorkerProtocolVersion
	}
	if resp.PlaywrightVersion == "" {
		resp.PlaywrightVersion = PinnedPlaywrightVersion
	}
	if resp.BrowserVersion == "" {
		switch BrowserFamily(m.LastReq.Browser) {
		case BrowserFirefox:
			resp.BrowserVersion = "mock-firefox"
		case BrowserWebKit:
			resp.BrowserVersion = "mock-webkit"
		default:
			resp.BrowserVersion = "mock-chromium"
		}
	}
	return json.Marshal(resp)
}
