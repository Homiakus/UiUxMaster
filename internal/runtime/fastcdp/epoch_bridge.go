package fastcdp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
)

const (
	defaultEpochBinding = "__uiuxMasterEpochBinding"
	EpochSignalHelper   = "__UIUX_SIGNAL_RENDER__"
)

type EpochBridge struct {
	Session SessionID
	Gate    *EpochGate

	cancel func()
	done   chan struct{}
	once   sync.Once
}

type bindingCalledParams struct {
	Name               string `json:"name"`
	Payload            string `json:"payload"`
	ExecutionContextID int64  `json:"executionContextId"`
}

// InstallEpochBridge installs a tiny page helper. The application/HMR harness
// must call window.__UIUX_SIGNAL_RENDER__(epoch) only after the committed render
// is observable. This avoids networkidle/arbitrary sleeps and makes freshness an
// explicit application-level contract.
func (c *Connection) InstallEpochBridge(ctx context.Context, session SessionID, gate *EpochGate) (*EpochBridge, error) {
	if session == "" {
		return nil, fmt.Errorf("fastcdp: epoch bridge requires session id")
	}
	if gate == nil {
		gate = NewEpochGate()
	}
	events, unsubscribe := c.Subscribe("Runtime.bindingCalled", 32)

	if err := c.Call(ctx, string(session), "Runtime.addBinding", map[string]any{"name": defaultEpochBinding}, nil); err != nil {
		unsubscribe()
		return nil, fmt.Errorf("fastcdp: install epoch binding: %w", err)
	}

	script := epochBridgeScript(defaultEpochBinding)
	var scriptResult struct {
		Identifier string `json:"identifier"`
	}
	if err := c.Call(ctx, string(session), "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": script}, &scriptResult); err != nil {
		unsubscribe()
		return nil, fmt.Errorf("fastcdp: install epoch preload: %w", err)
	}
	// The page may already be loaded when the bridge is installed, so install the
	// helper in the current execution context as well.
	if err := c.Call(ctx, string(session), "Runtime.evaluate", map[string]any{
		"expression":    script,
		"returnByValue": true,
	}, nil); err != nil {
		unsubscribe()
		return nil, fmt.Errorf("fastcdp: activate epoch bridge: %w", err)
	}

	bridge := &EpochBridge{Session: session, Gate: gate, cancel: unsubscribe, done: make(chan struct{})}
	go bridge.consume(events)
	return bridge, nil
}

func (b *EpochBridge) consume(events <-chan Event) {
	defer close(b.done)
	for event := range events {
		if SessionID(event.SessionID) != b.Session {
			continue
		}
		var params bindingCalledParams
		if err := json.Unmarshal(event.Params, &params); err != nil || params.Name != defaultEpochBinding {
			continue
		}
		epoch, err := strconv.ParseUint(params.Payload, 10, 64)
		if err != nil {
			continue
		}
		b.Gate.Advance(epoch)
	}
}

func (b *EpochBridge) Close() {
	b.once.Do(func() {
		if b.cancel != nil {
			b.cancel()
		}
		// Subscribe channels are intentionally not closed by Connection because a
		// concurrent publish may hold a snapshot. The consumer is not awaited here;
		// it becomes unreachable after unsubscribe and the connection lifetime owns
		// the final event loop. This avoids a send-on-closed-channel race.
	})
}

func epochBridgeScript(binding string) string {
	bindingJSON, _ := json.Marshal(binding)
	helperJSON, _ := json.Marshal(EpochSignalHelper)
	return fmt.Sprintf(`(() => {
  const binding = globalThis[%s];
  if (typeof binding !== "function") return;
  globalThis[%s] = (epoch) => {
    const value = Number(epoch);
    if (!Number.isFinite(value) || value < 0) return false;
    const normalized = Math.floor(value);
    globalThis.__UIUX_RENDER_EPOCH__ = normalized;
    binding(String(normalized));
    return true;
  };
  const current = Number(globalThis.__UIUX_RENDER_EPOCH__);
  if (Number.isFinite(current) && current >= 0) binding(String(Math.floor(current)));
})();`, bindingJSON, helperJSON)
}
