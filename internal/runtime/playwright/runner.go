package playwright

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
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
	Command            string            `json:"command"` // "capture" or "scenario"
	Browser            string            `json:"browser"`
	URL                string            `json:"url,omitempty"`
	HTML               string            `json:"html,omitempty"`
	CSS                string            `json:"css,omitempty"`
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
	Scenario           *Scenario         `json:"scenario,omitempty"`
}

// WorkerResponse is the JSON payload returned by the Playwright worker process.
type WorkerResponse struct {
	Success        bool                      `json:"success"`
	Error          string                    `json:"error,omitempty"`
	URL            string                    `json:"url,omitempty"`
	AriaSnapshot   string                    `json:"aria_snapshot,omitempty"`
	ScreenshotB64  string                    `json:"screenshot_b64,omitempty"`
	ScreenshotPath string                    `json:"screenshot_path,omitempty"`
	Documents      []evidence.DocumentMetrics `json:"documents,omitempty"`
	Elements       []evidence.ElementRef     `json:"elements,omitempty"`
	Accessibility  []evidence.AccessibilityNode `json:"accessibility,omitempty"`
	Fonts          *evidence.FontEvidence    `json:"fonts,omitempty"`
	Diagnostics    *evidence.DiagnosticsEvidence `json:"diagnostics,omitempty"`
	RuntimeIssues  []evidence.RuntimeIssue   `json:"runtime_issues,omitempty"`
	Latency        evidence.RuntimeLatency   `json:"latency"`
}

// MapWorkerResponseToPacket translates the worker response into a canonical evidence.Packet.
func MapWorkerResponseToPacket(req TruthPathRequest, resp WorkerResponse, dur time.Duration) evidence.Packet {
	packet := evidence.Packet{
		RunID: req.RunID,
		URL:   resp.URL,
		Renderer: evidence.RendererRef{
			Tier:    "L3",
			Name:    fmt.Sprintf("playwright-%s", req.Browser),
			Version: "1.0.0",
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

func parseWorkerResponse(data []byte) (WorkerResponse, error) {
	var resp WorkerResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return WorkerResponse{}, fmt.Errorf("failed to unmarshal playwright worker response: %w", err)
	}
	if !resp.Success && resp.Error != "" {
		return resp, fmt.Errorf("playwright worker error: %s", resp.Error)
	}
	return resp, nil
}
