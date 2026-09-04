package fastcdp

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCaptureFontStateUsesRuntimeEvaluate(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	type outcome struct {
		fonts FontState
		err   error
	}
	done := make(chan outcome, 1)
	go func() {
		fonts, err := conn.CaptureFontState(ctx, "session-fonts", 2)
		done <- outcome{fonts: fonts, err: err}
	}()

	request := decodeWire(t, <-transport.writes)
	if request.Method != "Runtime.evaluate" || request.SessionID != "session-fonts" {
		t.Fatalf("request = %#v", request)
	}
	if !strings.Contains(string(request.Params), "document.fonts") {
		t.Fatalf("params = %s", request.Params)
	}
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: mustRawJSON(t, map[string]any{
		"result": map[string]any{
			"type": "object",
			"value": map[string]any{
				"status": "loaded", "total": 1, "truncated": false,
				"faces": []any{map[string]any{"family": "Inter", "style": "normal", "weight": "400", "stretch": "normal", "status": "loaded"}},
			},
		},
	})})

	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.fonts.Status != "loaded" || got.fonts.Total != 1 || len(got.fonts.Faces) != 1 || got.fonts.Faces[0].Family != "Inter" {
		t.Fatalf("fonts = %#v", got.fonts)
	}
}
