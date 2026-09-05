package playwright_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
)

func TestTruthPathCapabilitiesFailClosedUntilProbe(t *testing.T) {
	runner := &playwright.MockRunner{}
	adapter := playwright.New(playwright.Config{Runner: runner})

	before := adapter.Capabilities()
	if before.Ready || before.CleanState || len(before.Browsers) != 0 {
		t.Fatalf("unprobed capabilities must be fail-closed, got %#v", before)
	}

	readiness, err := adapter.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("readiness = %#v, want ready", readiness)
	}

	after := adapter.Capabilities()
	if !after.Ready || !after.CleanState {
		t.Fatalf("probed capabilities = %#v, want ready clean-state", after)
	}
	if len(after.Browsers) != 3 {
		t.Fatalf("probed browsers = %v, want three mock-attested engines", after.Browsers)
	}
	if after.WorkerVersion != playwright.WorkerProtocolVersion {
		t.Fatalf("worker version = %q", after.WorkerVersion)
	}
	if after.PlaywrightVersion != playwright.PinnedPlaywrightVersion {
		t.Fatalf("playwright version = %q", after.PlaywrightVersion)
	}
}

func TestTruthPathProbeAdvertisesOnlyLaunchableVersionedBrowsers(t *testing.T) {
	runner := &playwright.MockRunner{ProbeResponse: &playwright.WorkerProbeResponse{
		Success:           true,
		WorkerVersion:     playwright.WorkerProtocolVersion,
		PlaywrightVersion: playwright.PinnedPlaywrightVersion,
		Browsers: []playwright.BrowserReadiness{
			{Browser: playwright.BrowserChromium, Ready: true, Version: "chromium-123"},
			{Browser: playwright.BrowserFirefox, Ready: false, Error: "not installed"},
			{Browser: playwright.BrowserWebKit, Ready: true},
		},
		Features: playwright.TruthPathFeatures{
			CleanState: true, SupportsARIA: true, SupportsFonts: true, SupportsScenario: true, SupportsROI: true,
		},
	}}
	adapter := playwright.New(playwright.Config{Runner: runner})

	readiness, err := adapter.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("readiness = %#v", readiness)
	}
	caps := adapter.Capabilities()
	if len(caps.Browsers) != 1 || caps.Browsers[0] != playwright.BrowserChromium {
		t.Fatalf("capability browsers = %v, want only chromium", caps.Browsers)
	}
	if caps.BrowserVersions[playwright.BrowserChromium] != "chromium-123" {
		t.Fatalf("browser versions = %#v", caps.BrowserVersions)
	}
}

func TestTruthPathProbeRejectsProtocolAndRuntimeVersionDrift(t *testing.T) {
	tests := []struct {
		name string
		probe playwright.WorkerProbeResponse
		want error
	}{
		{
			name: "worker protocol",
			probe: playwright.WorkerProbeResponse{Success: true, WorkerVersion: "0.9.0", PlaywrightVersion: playwright.PinnedPlaywrightVersion},
			want: playwright.ErrWorkerVersionMismatch,
		},
		{
			name: "playwright runtime",
			probe: playwright.WorkerProbeResponse{Success: true, WorkerVersion: playwright.WorkerProtocolVersion, PlaywrightVersion: "1.61.0"},
			want: playwright.ErrPlaywrightVersionMismatch,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runner := &playwright.MockRunner{ProbeResponse: &tc.probe}
			adapter := playwright.New(playwright.Config{Runner: runner})
			_, err := adapter.Probe(context.Background())
			if !errors.Is(err, tc.want) {
				t.Fatalf("probe err = %v, want %v", err, tc.want)
			}
			if adapter.Capabilities().Ready {
				t.Fatalf("capabilities must remain unavailable after version mismatch")
			}
		})
	}
}

func TestTruthPathCaptureRejectsIdentityChangeAfterProbe(t *testing.T) {
	runner := &playwright.MockRunner{Response: playwright.WorkerResponse{
		Success:           true,
		WorkerVersion:     playwright.WorkerProtocolVersion,
		PlaywrightVersion: playwright.PinnedPlaywrightVersion,
		BrowserVersion:    "unexpected-browser-version",
	}}
	adapter := playwright.New(playwright.Config{Runner: runner})

	_, err := adapter.Capture(context.Background(), playwright.TruthPathRequest{
		RunID:   "identity-drift",
		Browser: playwright.BrowserChromium,
	})
	if !errors.Is(err, playwright.ErrRuntimeIdentityChanged) {
		t.Fatalf("capture err = %v, want runtime identity drift", err)
	}
	if err != nil && !strings.Contains(err.Error(), "probe=") {
		t.Fatalf("identity error lacks probe/capture evidence: %v", err)
	}
}
