package fastcdp

import (
	"testing"
	"time"
)

func TestObservedSubscriptionCountsDroppedEvents(t *testing.T) {
	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	sub := conn.SubscribeObserved("Runtime.consoleAPICalled", 1)
	defer sub.Close()

	transport.reads <- mustJSON(t, wireMessage{Method: "Runtime.consoleAPICalled"})
	transport.reads <- mustJSON(t, wireMessage{Method: "Runtime.consoleAPICalled"})
	transport.reads <- mustJSON(t, wireMessage{Method: "Runtime.consoleAPICalled"})

	deadline := time.Now().Add(time.Second)
	for sub.Dropped() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := sub.Dropped(); got != 2 {
		t.Fatalf("dropped = %d, want 2", got)
	}

	select {
	case <-sub.Events:
	default:
		t.Fatal("expected first event to remain buffered")
	}
}
