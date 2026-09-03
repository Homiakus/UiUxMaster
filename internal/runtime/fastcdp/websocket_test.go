package fastcdp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestDialCarriesConcurrentCDPCallOverRealWebSocket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept websocket: %v", err)
			return
		}
		defer conn.CloseNow()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		typ, payload, err := conn.Read(ctx)
		if err != nil {
			t.Errorf("read request: %v", err)
			return
		}
		if typ != websocket.MessageText {
			t.Errorf("message type = %v, want text", typ)
			return
		}
		var request wireMessage
		if err := json.Unmarshal(payload, &request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		response := wireMessage{ID: request.ID, Result: json.RawMessage(`{"value":"ok"}`)}
		encoded, err := json.Marshal(response)
		if err != nil {
			t.Errorf("encode response: %v", err)
			return
		}
		if err := conn.Write(ctx, websocket.MessageText, encoded); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, err := Dial(ctx, endpoint, DialOptions{ReadLimit: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var out struct {
		Value string `json:"value"`
	}
	if err := conn.Call(ctx, "", "Runtime.evaluate", map[string]string{"expression": "1+1"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value != "ok" {
		t.Fatalf("value = %q, want ok", out.Value)
	}
}

func TestDialRejectsEmptyEndpoint(t *testing.T) {
	if _, err := Dial(context.Background(), "", DialOptions{}); err == nil {
		t.Fatal("expected empty endpoint error")
	}
}
