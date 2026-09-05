package playwright

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// CommandRunner executes subprocess commands. Pluggable for testing.
type CommandRunner interface {
	Run(ctx context.Context, cmd string, args []string, stdin []byte) ([]byte, error)
}

// OSCommandRunner executes commands via os/exec.
type OSCommandRunner struct{}

func (r *OSCommandRunner) Run(ctx context.Context, name string, args []string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("playwright worker error (%w): %s", err, stderr.String())
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

// WorkerRequest is the JSON payload sent to the Playwright worker process.
type WorkerRequest struct {
	Command            string            `json:"command"` // probe, capture or scenario
	Browser            string            `json:"browser,omitempty"`
	URL                string            `json:"url,omitempty"`
	HTML               string            `json:"html,omitempty"`
	CSS                string            `json:"css,omitempty"`
	BaseURL            string            `json:"base_url,omitempty"`
	Viewport           evidence.Viewport `json:"viewport,omitempty"`
	Region             *evidence.Rect    `json:"region,omitempty"`
	CapturePixels      bool              `json:"capture_pixels,omitempty"`
	CaptureARIA        bool              `json:"capture_aria,omitempty"`
	CaptureFonts       bool              `json:"capture_fonts,omitempty"`
	CaptureDiagnostics bool              `json:"capture_diagnostics,omitempty"`
	CaptureLayout      bool              `json:"capture_layout,omitempty"`
	PauseAnimations    bool              `json:"pause_animations,omitempty"`
	FreezeClock        bool              `json:"freeze_clock,omitempty"`
	Scenario           *Scenario         `json:"scenario,omitempty"`
}

// WorkerProbeResponse is the worker's runtime-attested readiness result.
type WorkerProbeResponse struct {
	Success           bool               `json:"success"`
	Error             string             `json:"error,omitempty"`
	WorkerVersion     string             `json:"worker_version,omitempty"`
	PlaywrightVersion string             `json:"playwright_version,omitempty"`
	Browsers          []BrowserReadiness `json:"browsers,omitempty"`
	Features          TruthPathFeatures  `json:"features"`
}

// WorkerResponse is the JSON payload returned by capture/scenario commands.
// Runtime identity is carried with every evidence packet rather than inferred
// from static build-time constants.
type WorkerResponse struct {
	Success           bool                           `json:"success"`
	Error             string                         `json:"error,omitempty"`
	WorkerVersion     string                         `json:"worker_version,omitempty"`
	PlaywrightVersion string                         `json:"playwright_version,omitempty"`
	BrowserVersion    string                         `json:"browser_version,omitempty"`
	URL               string                         `json:"url,omitempty"`
	AriaSnapshot      string                         `json:"aria_snapshot,omitempty"`
	ScreenshotB64     string                         `json:"screenshot_b64,omitempty"`
	ScreenshotPath    string                         `json:"screenshot_path,omitempty"`
	Documents         []evidence.DocumentMetrics     `json:"documents,omitempty"`
	Elements          []evidence.ElementRef          `json:"elements,omitempty"`
	Accessibility     []evidence.AccessibilityNode   `json:"accessibility,omitempty"`
	Fonts             *evidence.FontEvidence         `json:"fonts,omitempty"`
	Diagnostics       *evidence.DiagnosticsEvidence  `json:"diagnostics,omitempty"`
	RuntimeIssues     []evidence.RuntimeIssue        `json:"runtime_issues,omitempty"`
	Latency           evidence.RuntimeLatency        `json:"latency"`
}

// MapWorkerResponseToPacket translates the worker response into a canonical evidence.Packet.
func MapWorkerResponseToPacket(req TruthPathRequest, resp WorkerResponse, dur time.Duration) evidence.Packet {
	identity := runtimeIdentity(resp.WorkerVersion, resp.PlaywrightVersion, resp.BrowserVersion)
	packet := evidence.Packet{
		RunID: req.RunID,
		URL:   resp.URL,
		Renderer: evidence.RendererRef{
			Tier:       "L3",
			Name:       fmt.Sprintf("playwright-%s", req.Browser),
			Version:    identity,
			FidelityID: "truthpath:" + identity,
		},
		Viewport:       req.Viewport,
		Documents:      resp.Documents,
		Elements:       resp.Elements,
		Accessibility:  resp.Accessibility,
		Fonts:          resp.Fonts,
		Diagnostics:    resp.Diagnostics,
		RuntimeIssues:  resp.RuntimeIssues,
		AriaSnapshot:   resp.AriaSnapshot,
		ScreenshotPath: resp.ScreenshotPath,
		Latency:        resp.Latency,
	}

	if packet.Latency.TotalMS == 0 {
		packet.Latency.TotalMS = float64(dur.Milliseconds())
	}

	if resp.ScreenshotB64 != "" {
		data, err := base64.StdEncoding.DecodeString(resp.ScreenshotB64)
		if err == nil && len(data) > 0 {
			b := evidence.Rect{
				Width:  float64(req.Viewport.Width),
				Height: float64(req.Viewport.Height),
			}
			if req.Region != nil {
				b = *req.Region
			}
			packet.Pixels = &evidence.PixelEvidence{
				Bounds:       b,
				Width:        int(b.Width),
				Height:       int(b.Height),
				EncodedBytes: len(data),
			}
		}
	}

	if req.Region != nil {
		packet.VisualRegions = append(packet.VisualRegions, evidence.VisualRegion{
			ID:     "requested-roi",
			Bounds: *req.Region,
		})
	}

	return packet
}

func runtimeIdentity(workerVersion, playwrightVersion, browserVersion string) string {
	parts := make([]string, 0, 3)
	if workerVersion != "" {
		parts = append(parts, "worker="+workerVersion)
	}
	if playwrightVersion != "" {
		parts = append(parts, "playwright="+playwrightVersion)
	}
	if browserVersion != "" {
		parts = append(parts, "browser="+browserVersion)
	}
	if len(parts) == 0 {
		return "unattested"
	}
	return strings.Join(parts, ";")
}

func parseWorkerResponse(data []byte) (WorkerResponse, error) {
	var resp WorkerResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return WorkerResponse{}, fmt.Errorf("failed to unmarshal playwright worker response: %w", err)
	}
	if !resp.Success {
		if resp.Error == "" {
			resp.Error = "worker returned success=false"
		}
		return resp, fmt.Errorf("playwright worker error: %s", resp.Error)
	}
	return resp, nil
}

func parseProbeResponse(data []byte) (WorkerProbeResponse, error) {
	var resp WorkerProbeResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return WorkerProbeResponse{}, fmt.Errorf("failed to unmarshal playwright probe response: %w", err)
	}
	if !resp.Success {
		if resp.Error == "" {
			resp.Error = "worker probe returned success=false"
		}
		return resp, fmt.Errorf("playwright readiness probe failed: %s", resp.Error)
	}
	return resp, nil
}
