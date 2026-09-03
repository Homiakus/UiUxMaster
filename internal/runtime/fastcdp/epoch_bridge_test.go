package fastcdp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestInstallEpochBridgeTracksBindingEventsAndStops(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	gate := NewEpochGate()
	type outcome struct {
		bridge *EpochBridge
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		bridge, err := conn.InstallEpochBridge(ctx, "session-epoch", gate)
		done <- outcome{bridge: bridge, err: err}
	}()

	bindingReq := decodeWire(t, <-transport.writes)
	if bindingReq.Method != "Runtime.addBinding" || bindingReq.SessionID != "session-epoch" {
		t.Fatalf("binding request = %#v", bindingReq)
	}
	transport.reads <- mustJSON(t, wireMessage{ID: bindingReq.ID, Result: json.RawMessage(`{}`)})

	preloadReq := decodeWire(t, <-transport.writes)
	if preloadReq.Method != "Page.addScriptToEvaluateOnNewDocument" {
		t.Fatalf("preload method = %q", preloadReq.Method)
	}
	var preload map[string]string
	if err := json.Unmarshal(preloadReq.Params, &preload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preload["source"], EpochSignalHelper) {
		t.Fatalf("preload script does not contain helper %q", EpochSignalHelper)
	}
	transport.reads <- mustJSON(t, wireMessage{ID: preloadReq.ID, Result: json.RawMessage(`{"identifier":"script-1"}`)})

	evalReq := decodeWire(t, <-transport.writes)
	if evalReq.Method != "Runtime.evaluate" {
		t.Fatalf("evaluate method = %q", evalReq.Method)
	}
	transport.reads <- mustJSON(t, wireMessage{ID: evalReq.ID, Result: json.RawMessage(`{}`)})

	installed := <-done
	if installed.err != nil {
		t.Fatal(installed.err)
	}
	bridge := installed.bridge

	params, err := json.Marshal(bindingCalledParams{Name: defaultEpochBinding, Payload: "17"})
	if err != nil {
		t.Fatal(err)
	}
	transport.reads <- mustJSON(t, wireMessage{Method: "Runtime.bindingCalled", SessionID: "other", Params: params})
	transport.reads <- mustJSON(t, wireMessage{Method: "Runtime.bindingCalled", SessionID: "session-epoch", Params: params})

	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
	defer waitCancel()
	epoch, err := gate.WaitAfter(waitCtx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if epoch != 17 {
		t.Fatalf("epoch = %d, want 17", epoch)
	}

	bridge.Close()
	select {
	case <-bridge.done:
	default:
		t.Fatal("epoch bridge consumer did not stop")
	}
}
