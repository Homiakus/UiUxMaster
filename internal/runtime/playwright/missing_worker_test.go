package playwright_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
)

func TestTruthPathMissingWorkerIsUnavailable(t *testing.T) {
	adapter := playwright.New(playwright.Config{
		WorkerScript: filepath.Join(t.TempDir(), "missing-worker.cjs"),
	})

	_, err := adapter.Probe(context.Background())
	if !errors.Is(err, playwright.ErrTruthPathUnavailable) {
		t.Fatalf("probe err = %v, want ErrTruthPathUnavailable", err)
	}
	caps := adapter.Capabilities()
	if caps.Ready || len(caps.Browsers) != 0 || caps.CleanState || caps.SupportsARIA || caps.SupportsFonts || caps.SupportsScenario || caps.SupportsROI {
		t.Fatalf("missing worker must expose no usable TruthPath capability, got %#v", caps)
	}

	_, err = adapter.Capture(context.Background(), playwright.TruthPathRequest{
		RunID:   "missing-worker",
		Browser: playwright.BrowserChromium,
	})
	if !errors.Is(err, playwright.ErrTruthPathUnavailable) {
		t.Fatalf("capture err = %v, want fail-closed unavailable", err)
	}
}
