package fastcdp

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestResidentRuntimeIntegration(t *testing.T) {
	if os.Getenv("UIUX_FASTCDP_INTEGRATION") != "1" {
		t.Skip("set UIUX_FASTCDP_INTEGRATION=1 to run against a real Chromium binary")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runtime, err := StartResidentRuntime(ctx, RuntimeConfig{
		Browser: BrowserConfig{Executable: os.Getenv("UIUX_CHROME_BIN")},
		Pages: PagePoolConfig{
			MaxPages: 1,
			Page: PageSpec{URL: "about:blank", Width: 320, Height: 200, DPR: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		if err := runtime.Close(closeCtx); err != nil {
			t.Errorf("close runtime: %v", err)
		}
	}()

	version, err := runtime.Version(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version.Product == "" {
		t.Fatal("browser product is empty")
	}

	lease, err := runtime.Pages.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	page := lease.Page()
	before := page.Epoch.Current()

	expression := `document.documentElement.style.background = "white";
document.body.style.margin = "0";
document.body.innerHTML = '<main id="probe" style="width:240px;height:120px;display:flex;align-items:center;justify-content:center;background:rgb(20,30,40);color:white">UiUxMaster L2</main>';
window.__UIUX_SIGNAL_RENDER__(1);`
	if err := runtime.Conn.Call(ctx, string(page.Session.SessionID), "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
	}, nil); err != nil {
		t.Fatal(err)
	}

	epoch, err := page.Epoch.WaitAfter(ctx, before)
	if err != nil {
		t.Fatal(err)
	}
	if epoch != 1 {
		t.Fatalf("epoch = %d, want 1", epoch)
	}

	snapshot, err := runtime.Conn.CaptureSnapshot(ctx, string(page.Session.SessionID), SnapshotOptions{
		ComputedStyles:  []string{"display", "color", "background-color"},
		IncludeDOMRects: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Documents) == 0 || len(snapshot.Documents[0].Nodes) == 0 {
		t.Fatalf("empty DOM snapshot: %#v", snapshot)
	}

	img, stats, err := runtime.Conn.CaptureRegionRGBA(ctx, string(page.Session.SessionID), CaptureRegionOptions{
		X: 0, Y: 0, Width: 320, Height: 200, Scale: 1,
		OptimizeForSpeed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if img == nil || img.Bounds().Dx() != 320 || img.Bounds().Dy() != 200 {
		t.Fatalf("unexpected screenshot bounds: %v", img)
	}
	if stats.EncodedBytes == 0 {
		t.Fatalf("empty screenshot stats: %#v", stats)
	}
}
