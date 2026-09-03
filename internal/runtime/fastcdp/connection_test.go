package fastcdp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type fakeTransport struct {
	reads  chan []byte
	writes chan []byte
	closed chan struct{}
	once   sync.Once
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{
		reads:  make(chan []byte, 32),
		writes: make(chan []byte, 32),
		closed: make(chan struct{}),
	}
}

func (f *fakeTransport) Read(ctx context.Context) ([]byte, error) {
	select {
	case payload := <-f.reads:
		return payload, nil
	case <-f.closed:
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *fakeTransport) Write(ctx context.Context, payload []byte) error {
	copyPayload := append([]byte(nil), payload...)
	select {
	case f.writes <- copyPayload:
		return nil
	case <-f.closed:
		return io.ErrClosedPipe
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *fakeTransport) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

func TestConnectionCorrelatesConcurrentResponsesOutOfOrder(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	type response struct {
		Value string `json:"value"`
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	results := make(chan string, 2)
	errs := make(chan error, 2)
	for _, expression := range []string{"first", "second"} {
		expression := expression
		go func() {
			var out response
			err := conn.Call(ctx, "", "Runtime.evaluate", map[string]any{"expression": expression}, &out)
			if err != nil {
				errs <- err
				return
			}
			results <- out.Value
		}()
	}

	first := decodeWire(t, <-transport.writes)
	second := decodeWire(t, <-transport.writes)
	if first.ID == second.ID || first.ID == 0 || second.ID == 0 {
		t.Fatalf("request ids must be unique and non-zero: %d %d", first.ID, second.ID)
	}

	transport.reads <- mustJSON(t, wireMessage{ID: second.ID, Result: json.RawMessage(`{"value":"second-response"}`)})
	transport.reads <- mustJSON(t, wireMessage{ID: first.ID, Result: json.RawMessage(`{"value":"first-response"}`)})

	got := map[string]bool{}
	for range 2 {
		select {
		case err := <-errs:
			t.Fatal(err)
		case value := <-results:
			got[value] = true
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if !got["first-response"] || !got["second-response"] {
		t.Fatalf("unexpected results: %#v", got)
	}
}

func TestConnectionCancellationRemovesPendingCall(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- conn.Call(ctx, "", "DOMSnapshot.captureSnapshot", nil, nil)
	}()

	request := decodeWire(t, <-transport.writes)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Call error = %v, want context.Canceled", err)
	}

	conn.mu.Lock()
	_, stillPending := conn.pending[request.ID]
	conn.mu.Unlock()
	if stillPending {
		t.Fatal("cancelled call remained in pending map")
	}

	// A late browser response after cancellation must be ignored safely.
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: json.RawMessage(`{}`)})
}

func TestConnectionPublishesSubscribedEvents(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	events, unsubscribe := conn.Subscribe("Runtime.bindingCalled", 2)
	defer unsubscribe()
	transport.reads <- mustJSON(t, wireMessage{
		Method:    "Runtime.bindingCalled",
		SessionID: "session-1",
		Params:    json.RawMessage(`{"name":"__uiuxEpoch","payload":"42"}`),
	})

	select {
	case event := <-events:
		if event.SessionID != "session-1" || event.Method != "Runtime.bindingCalled" {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for CDP event")
	}
}

func TestConnectionProtocolErrorIsReturnedToCaller(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	done := make(chan error, 1)
	go func() {
		done <- conn.Call(context.Background(), "", "Runtime.evaluate", nil, nil)
	}()
	request := decodeWire(t, <-transport.writes)
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Error: &ProtocolError{Code: -32000, Message: "boom"}})

	var protocolErr *ProtocolError
	if err := <-done; !errors.As(err, &protocolErr) || protocolErr.Code != -32000 {
		t.Fatalf("Call error = %v, want ProtocolError(-32000)", err)
	}
}

func TestConnectionMalformedFrameFailsPendingCalls(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	done := make(chan error, 1)
	go func() {
		done <- conn.Call(context.Background(), "", "Runtime.evaluate", nil, nil)
	}()
	<-transport.writes
	transport.reads <- []byte("not-json")

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected pending call to fail")
		}
	case <-time.After(time.Second):
		t.Fatal("pending call was not released after malformed frame")
	}
	select {
	case <-conn.Done():
	default:
		t.Fatal("connection did not close after malformed frame")
	}
}

func decodeWire(t *testing.T, payload []byte) wireMessage {
	t.Helper()
	var msg wireMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatal(err)
	}
	return msg
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}
