package fastcdp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

type cdpAutoResponder struct {
	transport    *fakeTransport
	createTarget atomic.Int64
	stop         chan struct{}
}

func newCDPAutoResponder(transport *fakeTransport) *cdpAutoResponder {
	r := &cdpAutoResponder{transport: transport, stop: make(chan struct{})}
	go r.loop()
	return r
}

func (r *cdpAutoResponder) Close() { close(r.stop) }

func (r *cdpAutoResponder) loop() {
	for {
		select {
		case <-r.stop:
			return
		case payload := <-r.transport.writes:
			var request wireMessage
			if json.Unmarshal(payload, &request) != nil {
				continue
			}
			var result json.RawMessage
			switch request.Method {
			case "Target.createBrowserContext":
				result = json.RawMessage(`{"browserContextId":"pool-context"}`)
			case "Target.createTarget":
				n := r.createTarget.Add(1)
				result = json.RawMessage(fmt.Sprintf(`{"targetId":"target-%d"}`, n))
			case "Target.attachToTarget":
				var params struct {
					TargetID string `json:"targetId"`
				}
				_ = json.Unmarshal(request.Params, &params)
				result = json.RawMessage(fmt.Sprintf(`{"sessionId":"session-%s"}`, params.TargetID))
			case "Target.closeTarget":
				result = json.RawMessage(`{"success":true}`)
			case "Page.addScriptToEvaluateOnNewDocument":
				result = json.RawMessage(`{"identifier":"epoch-script"}`)
			default:
				result = json.RawMessage(`{}`)
			}
			r.transport.reads <- mustMarshalWire(wireMessage{ID: request.ID, Result: result})
		}
	}
}

func mustMarshalWire(value wireMessage) []byte {
	payload, _ := json.Marshal(value)
	return payload
}

func TestPagePoolReusesHealthyWarmPage(t *testing.T) {
	transport := newFakeTransport()
	responder := newCDPAutoResponder(transport)
	defer responder.Close()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pool, err := NewPagePool(ctx, conn, PagePoolConfig{MaxPages: 2, Page: PageSpec{URL: "https://example.test/"}})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close(ctx)

	first, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	firstID := first.Page().Session.TargetID
	first.Release()

	second, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second.Page().Session.TargetID != firstID {
		t.Fatalf("target = %q, want reused %q", second.Page().Session.TargetID, firstID)
	}
	second.Release()
	if got := responder.createTarget.Load(); got != 1 {
		t.Fatalf("created targets = %d, want 1", got)
	}
}

func TestPagePoolBoundBlocksUntilLeaseReturns(t *testing.T) {
	transport := newFakeTransport()
	responder := newCDPAutoResponder(transport)
	defer responder.Close()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pool, err := NewPagePool(ctx, conn, PagePoolConfig{MaxPages: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close(ctx)

	first, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer shortCancel()
	if _, err := pool.Acquire(shortCtx); err == nil {
		t.Fatal("expected second acquire to respect page-pool bound")
	}
	first.Release()

	second, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second.Release()
}

func TestPagePoolDiscardReplacesOnlyAffectedPage(t *testing.T) {
	transport := newFakeTransport()
	responder := newCDPAutoResponder(transport)
	defer responder.Close()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pool, err := NewPagePool(ctx, conn, PagePoolConfig{MaxPages: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close(ctx)

	first, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	firstID := first.Page().Session.TargetID
	if err := first.Discard(ctx); err != nil {
		t.Fatal(err)
	}

	second, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	secondID := second.Page().Session.TargetID
	second.Release()
	if secondID == firstID {
		t.Fatalf("discarded target %q was reused", firstID)
	}
	if got := responder.createTarget.Load(); got != 2 {
		t.Fatalf("created targets = %d, want 2", got)
	}
}
