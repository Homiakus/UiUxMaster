package fastcdp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCaptureAXTreeProjectsSemanticFields(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	type outcome struct {
		tree AXTree
		err  error
	}
	done := make(chan outcome, 1)
	go func() {
		tree, err := conn.CaptureAXTree(ctx, "session-a11y", 0)
		done <- outcome{tree: tree, err: err}
	}()

	request := decodeWire(t, <-transport.writes)
	if request.Method != "Accessibility.getFullAXTree" || request.SessionID != "session-a11y" {
		t.Fatalf("request = %#v", request)
	}
	result := map[string]any{"nodes": []any{
		map[string]any{
			"nodeId": "1", "ignored": false,
			"role": map[string]any{"type": "role", "value": "RootWebArea"},
			"name": map[string]any{"type": "computedString", "value": "Example"},
			"childIds": []string{"2"},
		},
		map[string]any{
			"nodeId": "2", "parentId": "1", "backendDOMNodeId": 77, "ignored": false,
			"role": map[string]any{"type": "role", "value": "button"},
			"name": map[string]any{"type": "computedString", "value": "Publish"},
			"properties": []any{map[string]any{"name": "focusable", "value": map[string]any{"type": "boolean", "value": true}}},
		},
	}}
	transport.reads <- mustJSON(t, wireMessage{ID: request.ID, Result: mustRawJSON(t, result)})

	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	if len(got.tree.Nodes) != 2 || got.tree.Nodes[1].BackendDOMNodeID != 77 || got.tree.Nodes[1].Role != "button" || got.tree.Nodes[1].Name != "Publish" {
		t.Fatalf("tree = %#v", got.tree)
	}
	if got.tree.Nodes[1].Properties["focusable"] != "true" {
		t.Fatalf("properties = %#v", got.tree.Nodes[1].Properties)
	}
	text := got.tree.TextSnapshot()
	if !strings.Contains(text, "button: Publish") {
		t.Fatalf("snapshot = %q", text)
	}
}

func TestAXTreeTextSnapshotHandlesDetachedNodes(t *testing.T) {
	tree := AXTree{Nodes: []AXNode{{ID: "detached", Role: "button", Name: "Detached"}}}
	if got := tree.TextSnapshot(); !strings.Contains(got, "button: Detached") {
		t.Fatalf("snapshot = %q", got)
	}
}

var _ = json.RawMessage{}
