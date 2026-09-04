package fastcdp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDecodeDiagnostics(t *testing.T) {
	tests := []struct {
		method string
		json   string
		kind   DiagnosticKind
		level  string
	}{
		{"Runtime.exceptionThrown", `{"exceptionDetails":{"text":"Uncaught","url":"app.js","exception":{"description":"Error: boom"}}}`, DiagnosticRuntimeException, "error"},
		{"Runtime.consoleAPICalled", `{"type":"error","args":[{"value":"bad"},{"description":"Error object"}]}`, DiagnosticConsole, "error"},
		{"Log.entryAdded", `{"entry":{"source":"security","level":"warning","text":"mixed content","url":"https://example.test"}}`, DiagnosticLog, "warning"},
		{"Network.loadingFailed", `{"requestId":"r1","type":"Script","errorText":"net::ERR_FAILED","canceled":false}`, DiagnosticNetworkFailed, "error"},
		{"Network.responseReceived", `{"requestId":"r2","type":"Fetch","response":{"url":"https://example.test/api","status":503,"statusText":"Service Unavailable"}}`, DiagnosticHTTPError, "error"},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			event, ok := decodeDiagnostic(tt.method, json.RawMessage(tt.json))
			if !ok {
				t.Fatal("diagnostic was not decoded")
			}
			if event.Kind != tt.kind || event.Level != tt.level {
				t.Fatalf("event = %#v", event)
			}
		})
	}
}

func TestDiagnosticsSnapshotDetectsRingEviction(t *testing.T) {
	o := &DiagnosticsObserver{capacity: 2, stop: make(chan struct{})}
	mark := o.Mark()
	o.append(DiagnosticEvent{Kind: DiagnosticConsole, Message: "one"})
	o.append(DiagnosticEvent{Kind: DiagnosticConsole, Message: "two"})
	o.append(DiagnosticEvent{Kind: DiagnosticConsole, Message: "three"})

	snapshot := o.SnapshotSince(mark)
	if snapshot.Complete {
		t.Fatal("snapshot should be incomplete after ring eviction")
	}
	if !reflect.DeepEqual(snapshot.DroppedMethods, []string{"observer.ring"}) {
		t.Fatalf("dropped = %#v", snapshot.DroppedMethods)
	}
	if len(snapshot.Events) != 2 || snapshot.Events[0].Message != "two" || snapshot.Events[1].Message != "three" {
		t.Fatalf("events = %#v", snapshot.Events)
	}
}

func TestDiagnosticsSnapshotFiltersByMark(t *testing.T) {
	o := &DiagnosticsObserver{capacity: 4, stop: make(chan struct{})}
	o.append(DiagnosticEvent{Kind: DiagnosticConsole, Message: "before"})
	mark := o.Mark()
	o.append(DiagnosticEvent{Kind: DiagnosticConsole, Message: "after"})

	snapshot := o.SnapshotSince(mark)
	if !snapshot.Complete || len(snapshot.Events) != 1 || snapshot.Events[0].Message != "after" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}
