package fastcdp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"sync/atomic"
	"testing"
	"time"
)

func TestCollectEvidenceWaitsForFreshEpochAndCapturesStablePair(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()
	gate := NewEpochGate()
	gate.Advance(1)
	page := &WarmPage{Session: PageSession{SessionID: "session-evidence"}, Epoch: gate}
	pngData := evidencePNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	type outcome struct {
		evidence CollectedEvidence
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		evidence, err := page.CollectEvidence(ctx, conn, EvidenceRequest{
			RequireAfter:    1,
			WaitForNewEpoch: true,
			Snapshot:        &SnapshotOptions{ComputedStyles: []string{"display"}},
			Region:          &CaptureRegionOptions{Width: 2, Height: 1},
		})
		done <- outcome{evidence: evidence, err: err}
	}()

	select {
	case payload := <-transport.writes:
		t.Fatalf("capture started before fresh epoch: %s", payload)
	case <-time.After(20 * time.Millisecond):
	}
	gate.Advance(2)
	respondToEvidenceAttempt(t, transport, pngData, nil)

	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	if result.evidence.Epoch != 2 {
		t.Fatalf("epoch = %d, want 2", result.evidence.Epoch)
	}
	if result.evidence.Snapshot == nil || result.evidence.RGBA == nil {
		t.Fatalf("missing evidence: %#v", result.evidence)
	}
	if result.evidence.RGBA.Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatalf("RGBA bounds = %v", result.evidence.RGBA.Bounds())
	}
	if result.evidence.Timing.WaitEpoch <= 0 || result.evidence.Timing.Retries != 0 {
		t.Fatalf("timing = %#v", result.evidence.Timing)
	}
}

func TestCollectEvidenceRetriesWhenEpochChangesDuringCapture(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()
	gate := NewEpochGate()
	gate.Advance(1)
	page := &WarmPage{Session: PageSession{SessionID: "session-evidence"}, Epoch: gate}
	pngData := evidencePNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan struct {
		evidence CollectedEvidence
		err      error
	}, 1)
	go func() {
		evidence, err := page.CollectEvidence(ctx, conn, EvidenceRequest{
			Snapshot:        &SnapshotOptions{ComputedStyles: []string{"display"}},
			Region:          &CaptureRegionOptions{Width: 2, Height: 1},
			MaxEpochRetries: 1,
		})
		done <- struct {
			evidence CollectedEvidence
			err      error
		}{evidence, err}
	}()

	var changed atomic.Bool
	respondToEvidenceAttempt(t, transport, pngData, func(method string) {
		if method == "DOMSnapshot.captureSnapshot" && changed.CompareAndSwap(false, true) {
			gate.Advance(2)
		}
	})
	respondToEvidenceAttempt(t, transport, pngData, nil)

	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	if result.evidence.Epoch != 2 || result.evidence.Timing.Retries != 1 {
		t.Fatalf("evidence = %#v", result.evidence)
	}
}

func TestCollectEvidenceFailsConservativeWhenEpochKeepsChanging(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()
	gate := NewEpochGate()
	gate.Advance(7)
	page := &WarmPage{Session: PageSession{SessionID: "session-evidence"}, Epoch: gate}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := page.CollectEvidence(ctx, conn, EvidenceRequest{
			Snapshot:        &SnapshotOptions{},
			MaxEpochRetries: -1,
		})
		done <- err
	}()

	request := decodeWire(t, <-transport.writes)
	if request.Method != "DOMSnapshot.captureSnapshot" {
		t.Fatalf("method = %q", request.Method)
	}
	gate.Advance(8)
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: evidenceSnapshotRaw()})
	if err := <-done; !errors.Is(err, ErrEpochChanged) {
		t.Fatalf("error = %v, want ErrEpochChanged", err)
	}
}

func TestCollectEvidenceRejectsEmptyRequest(t *testing.T) {
	page := &WarmPage{Session: PageSession{SessionID: "session"}, Epoch: NewEpochGate()}
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()
	if _, err := page.CollectEvidence(context.Background(), conn, EvidenceRequest{}); err == nil {
		t.Fatal("expected empty evidence request error")
	}
}

func respondToEvidenceAttempt(t *testing.T, transport *fakeTransport, pngData string, onRequest func(string)) {
	t.Helper()
	for range 2 {
		request := decodeWire(t, <-transport.writes)
		if onRequest != nil {
			onRequest(request.Method)
		}
		switch request.Method {
		case "DOMSnapshot.captureSnapshot":
			transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: evidenceSnapshotRaw()})
		case "Page.captureScreenshot":
			result, err := json.Marshal(captureScreenshotResult{Data: pngData})
			if err != nil {
				t.Fatal(err)
			}
			transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: result})
		default:
			t.Fatalf("unexpected evidence method %q", request.Method)
		}
	}
}

func evidenceSnapshotRaw() json.RawMessage {
	raw := captureSnapshotResult{
		Strings: []string{"about:blank", "frame-1", "DIV", "", "block"},
		Documents: []captureDocument{{
			DocumentURL:   0,
			FrameID:       1,
			ContentWidth:  2,
			ContentHeight: 1,
			Nodes: captureNodeTable{
				ParentIndex:   []int{-1},
				NodeType:      []int{1},
				NodeName:      []int{2},
				NodeValue:     []int{3},
				BackendNodeID: []int64{77},
			},
			Layout: captureLayout{
				NodeIndex: []int{0},
				Styles:    [][]int{{4}},
				Bounds:    [][]float64{{0, 0, 2, 1}},
				Text:      []int{3},
			},
		}},
	}
	payload, _ := json.Marshal(raw)
	return payload
}

func evidencePNG(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, color.RGBA{R: 32, G: 64, B: 96, A: 255})
	img.SetRGBA(1, 0, color.RGBA{R: 200, G: 180, B: 160, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
