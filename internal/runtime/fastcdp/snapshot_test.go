package fastcdp

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCaptureSnapshotProjectsWhitelistedGeometryAndStyles(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	type outcome struct {
		snapshot Snapshot
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		snapshot, err := conn.CaptureSnapshot(ctx, "session-1", SnapshotOptions{
			ComputedStyles: []string{"display", "position", "display", " "},
		})
		done <- outcome{snapshot: snapshot, err: err}
	}()

	request := decodeWire(t, <-transport.writes)
	if request.Method != "DOMSnapshot.captureSnapshot" || request.SessionID != "session-1" {
		t.Fatalf("unexpected request: %#v", request)
	}
	var params captureSnapshotParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(params.ComputedStyles, []string{"display", "position"}) {
		t.Fatalf("computedStyles = %#v", params.ComputedStyles)
	}

	raw := captureSnapshotResult{
		Strings: []string{"https://example.test/", "frame-1", "#document", "DIV", "", "Hello", "block", "relative"},
		Documents: []captureDocument{{
			DocumentURL:   0,
			FrameID:       1,
			ContentWidth:  1280,
			ContentHeight: 720,
			Nodes: captureNodeTable{
				ParentIndex:   []int{-1, 0},
				NodeType:      []int{9, 1},
				NodeName:      []int{2, 3},
				NodeValue:     []int{4, 4},
				BackendNodeID: []int64{1, 77},
			},
			Layout: captureLayout{
				NodeIndex: []int{1},
				Styles:    [][]int{{6, 7}},
				Bounds:    [][]float64{{10, 20, 100, 40}},
				Text:      []int{5},
			},
		}},
	}
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: mustRawJSON(t, raw)})

	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	if len(result.snapshot.Documents) != 1 || len(result.snapshot.Documents[0].Nodes) != 1 {
		t.Fatalf("unexpected snapshot: %#v", result.snapshot)
	}
	node := result.snapshot.Documents[0].Nodes[0]
	if node.BackendNodeID != 77 || node.Name != "DIV" || node.Text != "Hello" {
		t.Fatalf("unexpected node identity: %#v", node)
	}
	if node.Bounds != (Rect{X: 10, Y: 20, Width: 100, Height: 40}) {
		t.Fatalf("bounds = %#v", node.Bounds)
	}
	if node.Styles["display"] != "block" || node.Styles["position"] != "relative" {
		t.Fatalf("styles = %#v", node.Styles)
	}
}

func TestCaptureSnapshotRejectsInvalidLayoutNodeIndex(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	done := make(chan error, 1)
	go func() {
		_, err := conn.CaptureSnapshot(context.Background(), "", SnapshotOptions{})
		done <- err
	}()
	request := decodeWire(t, <-transport.writes)
	raw := captureSnapshotResult{
		Strings: []string{"#document"},
		Documents: []captureDocument{{
			Nodes: captureNodeTable{NodeName: []int{0}},
			Layout: captureLayout{
				NodeIndex: []int{3},
				Bounds:    [][]float64{{0, 0, 10, 10}},
			},
		}},
	}
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: mustRawJSON(t, raw)})

	if err := <-done; err == nil || !strings.Contains(err.Error(), "invalid node index") {
		t.Fatalf("error = %v, want invalid node index", err)
	}
}

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}
