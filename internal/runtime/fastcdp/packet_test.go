package fastcdp

import (
	"image"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestToPacketProjectsSemanticIdentityParentsMetricsAndPixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	copy(img.Pix, []byte{
		255, 0, 0, 255, 0, 255, 0, 255,
		0, 0, 255, 255, 255, 255, 255, 255,
	})
	snapshot := Snapshot{Documents: []DocumentSnapshot{{
		FrameID:       "frame-1",
		DocumentURL:   "https://example.test/app",
		ContentWidth:  1024,
		ContentHeight: 768,
		Nodes: []SnapshotNode{
			{
				DocumentIndex: 0, NodeIndex: 1, ParentIndex: -1, NodeType: 1,
				BackendNodeID: 10, Name: "DIV", Bounds: Rect{X: 0, Y: 0, Width: 300, Height: 100},
				Styles: map[string]string{"display": "block", "visibility": "visible", "opacity": "1"},
				Attributes: map[string]string{"id": "toolbar"},
			},
			{
				DocumentIndex: 0, NodeIndex: 2, ParentIndex: 1, NodeType: 1,
				BackendNodeID: 11, Name: "BUTTON", Text: "Publish", Bounds: Rect{X: 10, Y: 10, Width: 100, Height: 40},
				Styles: map[string]string{"display": "inline-block", "visibility": "visible", "opacity": "1", "pointer-events": "auto"},
				Attributes: map[string]string{"data-testid": "publish", "aria-label": "Publish changes"},
			},
		},
	}}}
	region := CaptureRegionOptions{X: 10, Y: 20, Width: 2, Height: 2}
	packet := ToPacket(CollectedEvidence{
		Epoch:        7,
		Snapshot:     &snapshot,
		RGBA:         img,
		CaptureStats: CaptureStats{EncodedBytes: 123},
		Timing: EvidenceTiming{
			WaitEpoch: time.Millisecond,
			Snapshot:  2 * time.Millisecond,
			Pixels:    3 * time.Millisecond,
			Total:     5 * time.Millisecond,
			Retries:   1,
		},
	}, PacketOptions{
		RunID: "run-1", Scenario: "toolbar", Region: &region,
		Viewport: evidence.Viewport{Width: 1024, Height: 768, DeviceScale: 1},
		Browser: BrowserVersion{Product: "Chrome/152.0.0.0"},
		FidelityID: "blink-l2",
	})

	if packet.URL != "https://example.test/app" || packet.Epoch != 7 {
		t.Fatalf("packet identity = %#v", packet)
	}
	if packet.Renderer.Tier != "L2" || packet.Renderer.Name != "fastcdp" || packet.Renderer.Version != "Chrome/152.0.0.0" {
		t.Fatalf("renderer = %#v", packet.Renderer)
	}
	if len(packet.Documents) != 1 || packet.Documents[0].ContentWidth != 1024 {
		t.Fatalf("documents = %#v", packet.Documents)
	}
	if len(packet.Elements) != 2 {
		t.Fatalf("elements = %#v", packet.Elements)
	}
	parent, button := packet.Elements[0], packet.Elements[1]
	if parent.ID != "dom:frame-1:10" || parent.Selector != "#toolbar" {
		t.Fatalf("parent = %#v", parent)
	}
	if button.ID != "dom:frame-1:11" || button.ParentID != parent.ID || button.Role != "button" || button.Name != "Publish changes" {
		t.Fatalf("button = %#v", button)
	}
	if button.Selector != `[data-testid="publish"]` || !button.Visible {
		t.Fatalf("button selector/visibility = %#v", button)
	}
	if packet.Pixels == nil || packet.Pixels.Width != 2 || packet.Pixels.Height != 2 || packet.Pixels.EncodedBytes != 123 || len(packet.Pixels.DigestSHA256) != 64 {
		t.Fatalf("pixels = %#v", packet.Pixels)
	}
	if packet.Pixels.Bounds != (evidence.Rect{X: 10, Y: 20, Width: 2, Height: 2}) {
		t.Fatalf("pixel bounds = %#v", packet.Pixels.Bounds)
	}
	if packet.Latency.WaitEpochMS != 1 || packet.Latency.SnapshotMS != 2 || packet.Latency.PixelsMS != 3 || packet.Latency.TotalMS != 5 || packet.Latency.Retries != 1 {
		t.Fatalf("latency = %#v", packet.Latency)
	}
}

func TestToPacketMarksZeroOpacityElementInvisible(t *testing.T) {
	snapshot := Snapshot{Documents: []DocumentSnapshot{{
		FrameID: "f",
		Nodes: []SnapshotNode{{
			NodeIndex: 1, NodeType: 1, BackendNodeID: 2, Name: "BUTTON",
			Bounds: Rect{Width: 40, Height: 40},
			Styles: map[string]string{"display": "block", "visibility": "visible", "opacity": "0"},
		}},
	}}}
	packet := ToPacket(CollectedEvidence{Epoch: 1, Snapshot: &snapshot}, PacketOptions{})
	if len(packet.Elements) != 1 || packet.Elements[0].Visible {
		t.Fatalf("element = %#v", packet.Elements)
	}
}
