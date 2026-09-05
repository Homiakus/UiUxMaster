package fastcdp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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
	stop   chan struct{}
	done   chan struct{}
	once   sync.Once
}

type bindingCalledParams struct {
	Name               string `json:"name"`
	Payload            string `json:"payload"`
	ExecutionContextID int64  `json:"executionContextId"`
}

type renderSignalPayload struct {
	Epoch    uint64 `json:"epoch"`
	Revision string `json:"revision,omitempty"`
}

// InstallEpochBridge installs a tiny page helper. The application/HMR harness
// should call window.__UIUX_SIGNAL_RENDER__(epoch, revision) only after the
// committed render for that revision is observable. The single-argument form is
// retained for revisionless legacy callers.
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
	if err := c.Call(ctx, string(session), "Runtime.evaluate", map[string]any{
		"expression":    script,
		"returnByValue": true,
	}, nil); err != nil {
		unsubscribe()
		return nil, fmt.Errorf("fastcdp: activate epoch bridge: %w", err)
	}

	bridge := &EpochBridge{
		Session: session,
		Gate:    gate,
		cancel:  unsubscribe,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go bridge.consume(events)
	return bridge, nil
}

func (b *EpochBridge) consume(events <-chan Event) {
	defer close(b.done)
	for {
		select {
		case <-b.stop:
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if SessionID(event.SessionID) != b.Session {
				continue
			}
			var params bindingCalledParams
			if err := json.Unmarshal(event.Params, &params); err != nil || params.Name != defaultEpochBinding {
				continue
			}
			token, err := parseRenderSignalPayload(params.Payload)
			if err != nil {
				continue
			}
			b.Gate.AdvanceToken(token)
		}
	}
}

func parseRenderSignalPayload(payload string) (RenderToken, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return RenderToken{}, fmt.Errorf("fastcdp: empty render signal")
	}
	if strings.HasPrefix(payload, "{") {
		var signal renderSignalPayload
		if err := json.Unmarshal([]byte(payload), &signal); err != nil {
			return RenderToken{}, fmt.Errorf("fastcdp: decode render signal: %w", err)
		}
		signal.Revision = strings.TrimSpace(signal.Revision)
		return RenderToken{Epoch: signal.Epoch, Revision: signal.Revision}, nil
	}
	epoch, err := strconv.ParseUint(payload, 10, 64)
	if err != nil {
		return RenderToken{}, fmt.Errorf("fastcdp: decode legacy render epoch: %w", err)
	}
	return RenderToken{Epoch: epoch}, nil
}

func (b *EpochBridge) Close() {
	b.once.Do(func() {
		if b.cancel != nil {
			b.cancel()
		}
		close(b.stop)
		<-b.done
	})
}

func (b *EpochBridge) IsClosed() bool {
	if b == nil {
		return true
	}
	select {
	case <-b.done:
		return true
	default:
		return false
	}
}

func epochBridgeScript(binding string) string {
	bindingJSON, _ := json.Marshal(binding)
	helperJSON, _ := json.Marshal(EpochSignalHelper)
	return fmt.Sprintf(`(() => {
  const binding = globalThis[%s];
  if (typeof binding !== "function") return;
  const emit = (epoch, revision) => {
    if (revision) {
      binding(JSON.stringify({epoch, revision}));
    } else {
      binding(String(epoch));
    }
  };
  globalThis[%s] = (epoch, revision = "") => {
    const value = Number(epoch);
    if (!Number.isFinite(value) || value < 0) return false;
    const normalized = Math.floor(value);
    const normalizedRevision = String(revision ?? "").trim();
    globalThis.__UIUX_RENDER_EPOCH__ = normalized;
    globalThis.__UIUX_RENDER_REVISION__ = normalizedRevision;
    emit(normalized, normalizedRevision);
    return true;
  };
  const current = Number(globalThis.__UIUX_RENDER_EPOCH__);
  if (Number.isFinite(current) && current >= 0) {
    const revision = String(globalThis.__UIUX_RENDER_REVISION__ ?? "").trim();
    emit(Math.floor(current), revision);
  }
})();`, bindingJSON, helperJSON)
}
