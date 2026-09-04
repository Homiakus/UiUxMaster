package uiuxadapter

import (
	"context"
	"testing"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestNewFastCDPCollectorRejectsMissingRuntime(t *testing.T) {
	_, err := NewFastCDPCollector(context.Background(), nil, FastCDPCollectorConfig{
		Viewport: evidence.Viewport{Width: 320, Height: 200},
	})
	if err == nil {
		t.Fatal("expected missing runtime error")
	}
}

func TestCaptureRegionRequiresExplicitPositiveROI(t *testing.T) {
	if _, err := captureRegion(nil); err == nil {
		t.Fatal("expected nil ROI error")
	}
	if _, err := captureRegion(&controlplane.Region{Width: 0, Height: 10}); err == nil {
		t.Fatal("expected invalid size error")
	}
	if _, err := captureRegion(&controlplane.Region{Width: 10, Height: 10, Scale: -1}); err == nil {
		t.Fatal("expected invalid scale error")
	}

	region, err := captureRegion(&controlplane.Region{X: 3, Y: 4, Width: 100, Height: 50})
	if err != nil {
		t.Fatal(err)
	}
	if region.X != 3 || region.Y != 4 || region.Width != 100 || region.Height != 50 || region.Scale != 1 || !region.OptimizeForSpeed {
		t.Fatalf("region = %#v", region)
	}
}
