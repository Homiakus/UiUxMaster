package fastcdp

import (
	"context"
	"testing"
	"time"
)

func TestDiagnosticsBarrierWaitsForDeliveredEvents(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()
	observer, err := NewDiagnosticsObserver(conn, "session-1", 8)
	if err != nil { t.Fatal(err) }
	defer observer.Close()
	mark := observer.Mark()

	transport.reads <- mustJSON(t, wireMessage{
		Method: "Runtime.consoleAPICalled", SessionID: "session-1",
		Params: mustRawJSON(t, map[string]any{"type":"error","args":[]any{map[string]any{"value":"boom"}}}),
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func(){ done <- observer.Barrier(ctx, conn) }()
	request := decodeWire(t, <-transport.writes)
	if request.Method != "Runtime.evaluate" || request.SessionID != "session-1" { t.Fatalf("request = %#v", request) }
	transport.reads <- mustJSON(t, wireMessage{ID:request.ID, Result:mustRawJSON(t,map[string]any{"result":map[string]any{"type":"number","value":0}})})
	if err := <-done; err != nil { t.Fatal(err) }

	snapshot := observer.SnapshotSince(mark)
	if !snapshot.Complete || len(snapshot.Events)!=1 || snapshot.Events[0].Message!="boom" { t.Fatalf("snapshot = %#v", snapshot) }
}
