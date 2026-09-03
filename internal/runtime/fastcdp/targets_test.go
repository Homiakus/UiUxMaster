package fastcdp

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestCreatePageCreatesAndFlatAttachesTarget(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan struct {
		page PageSession
		err  error
	}, 1)
	go func() {
		page, err := conn.CreatePage(ctx, PageSpec{URL: "https://example.test/", Width: 1280, Height: 720, Context: "ctx-1"})
		done <- struct {
			page PageSession
			err  error
		}{page, err}
	}()

	createReq := decodeWire(t, <-transport.writes)
	if createReq.Method != "Target.createTarget" {
		t.Fatalf("method = %q", createReq.Method)
	}
	var createParams map[string]any
	if err := json.Unmarshal(createReq.Params, &createParams); err != nil {
		t.Fatal(err)
	}
	if createParams["browserContextId"] != "ctx-1" || createParams["url"] != "https://example.test/" {
		t.Fatalf("params = %#v", createParams)
	}
	transport.reads <- mustJSON(t, wireMessage{ID: createReq.ID, Result: json.RawMessage(`{"targetId":"target-1"}`)})

	attachReq := decodeWire(t, <-transport.writes)
	if attachReq.Method != "Target.attachToTarget" {
		t.Fatalf("method = %q", attachReq.Method)
	}
	var attachParams struct {
		TargetID string `json:"targetId"`
		Flatten  bool   `json:"flatten"`
	}
	if err := json.Unmarshal(attachReq.Params, &attachParams); err != nil {
		t.Fatal(err)
	}
	if attachParams.TargetID != "target-1" || !attachParams.Flatten {
		t.Fatalf("attach params = %#v", attachParams)
	}
	transport.reads <- mustJSON(t, wireMessage{ID: attachReq.ID, Result: json.RawMessage(`{"sessionId":"session-1"}`)})

	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	if result.page != (PageSession{TargetID: "target-1", SessionID: "session-1", ContextID: "ctx-1"}) {
		t.Fatalf("page = %#v", result.page)
	}
}

func TestCreateBrowserContextAndDispose(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx := context.Background()
	created := make(chan struct {
		id  BrowserContextID
		err error
	}, 1)
	go func() {
		id, err := conn.CreateBrowserContext(ctx)
		created <- struct {
			id  BrowserContextID
			err error
		}{id, err}
	}()
	request := decodeWire(t, <-transport.writes)
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: json.RawMessage(`{"browserContextId":"ctx-9"}`)})
	result := <-created
	if result.err != nil || result.id != "ctx-9" {
		t.Fatalf("result = %#v", result)
	}

	disposed := make(chan error, 1)
	go func() { disposed <- conn.DisposeBrowserContext(ctx, result.id) }()
	disposeReq := decodeWire(t, <-transport.writes)
	if disposeReq.Method != "Target.disposeBrowserContext" {
		t.Fatalf("method = %q", disposeReq.Method)
	}
	transport.reads <- mustJSON(t, wireMessage{ID: disposeReq.ID, Result: json.RawMessage(`{}`)})
	if err := <-disposed; err != nil {
		t.Fatal(err)
	}
}

func TestEnablePageDomainsUsesAttachedSession(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	done := make(chan error, 1)
	go func() { done <- conn.EnablePageDomains(context.Background(), "session-x") }()
	for _, wantMethod := range []string{"Runtime.enable", "Page.enable", "DOM.enable"} {
		request := decodeWire(t, <-transport.writes)
		if request.Method != wantMethod || request.SessionID != "session-x" {
			t.Fatalf("request = %#v, want method %s/session-x", request, wantMethod)
		}
		transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: json.RawMessage(`{}`)})
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
