package fastcdp

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
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

	// Deliberately contains three deterministic defects so the real-browser gate
	// validates not just protocol plumbing but the complete L2 evidence pipeline:
	// horizontal document overflow, a clipped interactive target, and a target
	// below the 24x24 CSS-pixel minimum policy.
	expression := `document.documentElement.style.background = "white";
document.body.style.margin = "0";
document.body.innerHTML = '<main id="wide" style="width:360px;height:120px;background:#eee"><div id="clip" style="width:100px;height:50px;overflow:hidden"><button id="tiny" aria-label="Tiny action" style="display:block;width:20px;height:20px;margin-left:90px">Go</button></div></main>';
window.__UIUX_SIGNAL_RENDER__(1);`
	if err := runtime.Conn.Call(ctx, string(page.Session.SessionID), "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
	}, nil); err != nil {
		t.Fatal(err)
	}

	region := CaptureRegionOptions{
		X: 0, Y: 0, Width: 320, Height: 200, Scale: 1,
		OptimizeForSpeed: true,
	}
	collected, err := page.CollectEvidence(ctx, runtime.Conn, EvidenceRequest{
		RequireAfter:    before,
		WaitForNewEpoch: true,
		Snapshot:        ptrSnapshotOptions(DefaultSnapshotOptions()),
		Region:          &region,
	})
	if err != nil {
		t.Fatal(err)
	}
	if collected.Epoch != 1 {
		t.Fatalf("epoch = %d, want 1", collected.Epoch)
	}
	if collected.Snapshot == nil || len(collected.Snapshot.Documents) == 0 || len(collected.Snapshot.Documents[0].Nodes) == 0 {
		t.Fatalf("empty DOM snapshot: %#v", collected.Snapshot)
	}
	if collected.RGBA == nil || collected.RGBA.Bounds().Dx() != 320 || collected.RGBA.Bounds().Dy() != 200 {
		t.Fatalf("unexpected screenshot bounds: %v", collected.RGBA)
	}
	if collected.CaptureStats.EncodedBytes == 0 {
		t.Fatalf("empty screenshot stats: %#v", collected.CaptureStats)
	}

	packet := ToPacket(collected, PacketOptions{
		RunID: "chromium-integration", Scenario: "deterministic-defects",
		Viewport: evidence.Viewport{Width: 320, Height: 200, DeviceScale: 1},
		Browser: version, FidelityID: "blink-l2",
		Region: &region,
	})
	if packet.Epoch != 1 || packet.Renderer.Tier != "L2" || packet.Pixels == nil || len(packet.Elements) == 0 {
		t.Fatalf("incomplete canonical packet: %#v", packet)
	}

	verifier.Apply(&packet, verifier.DefaultPolicy())
	assertIssueCode(t, packet.RuntimeIssues, verifier.CodeViewportHorizontalOverflow)
	assertIssueCode(t, packet.RuntimeIssues, verifier.CodeInteractiveClipped)
	assertIssueCode(t, packet.RuntimeIssues, verifier.CodeTargetTooSmall)
}

func ptrSnapshotOptions(value SnapshotOptions) *SnapshotOptions { return &value }

func assertIssueCode(t *testing.T, issues []evidence.RuntimeIssue, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("missing issue %q: %#v", code, issues)
}
