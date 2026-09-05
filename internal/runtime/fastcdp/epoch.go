package fastcdp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var ErrRevisionMismatch = errors.New("fastcdp: observed render revision does not match requested revision")

// RenderToken is the framework-neutral freshness identity emitted after a
// committed browser render. Epoch orders renders; Revision binds that render to
// the source/build/change identity requested by the caller.
type RenderToken struct {
	Epoch    uint64 `json:"epoch"`
	Revision string `json:"revision,omitempty"`
}

// RevisionMismatchError preserves the expected and actually observed token so
// recovery/telemetry can distinguish stale evidence from application defects.
type RevisionMismatchError struct {
	Expected string
	Observed RenderToken
}

func (e *RevisionMismatchError) Error() string {
	if e == nil {
		return ErrRevisionMismatch.Error()
	}
	return fmt.Sprintf("%v: expected=%q observed=%q epoch=%d", ErrRevisionMismatch, e.Expected, e.Observed.Revision, e.Observed.Epoch)
}

func (e *RevisionMismatchError) Unwrap() error { return ErrRevisionMismatch }

// EpochGate tracks the newest application render token observed from the
// resident browser. It is intentionally independent from Vite/Next/HMR/vendor
// protocol details.
type EpochGate struct {
	mu      sync.Mutex
	current RenderToken
	nextID  uint64
	waiters map[uint64]epochWaiter
}

type epochWaiter struct {
	after            uint64
	expectedRevision string
	ch               chan renderWaitResult
}

type renderWaitResult struct {
	token RenderToken
	err   error
}

func NewEpochGate() *EpochGate {
	return &EpochGate{waiters: make(map[uint64]epochWaiter)}
}

// Current preserves the original numeric API for latency/benchmark callers.
func (g *EpochGate) Current() uint64 { return g.CurrentToken().Epoch }

func (g *EpochGate) CurrentToken() RenderToken {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.current
}

// Advance is the legacy revisionless signal. It remains valid for callers that
// did not request revision attestation; a revision-bound waiter will reject it.
func (g *EpochGate) Advance(epoch uint64) bool {
	return g.AdvanceToken(RenderToken{Epoch: epoch})
}

// AdvanceToken publishes a monotonically increasing committed render. Duplicate
// or stale epochs are ignored, including attempts to rewrite the revision of an
// already observed epoch.
func (g *EpochGate) AdvanceToken(token RenderToken) bool {
	token.Revision = strings.TrimSpace(token.Revision)
	g.mu.Lock()
	if token.Epoch <= g.current.Epoch {
		g.mu.Unlock()
		return false
	}
	g.current = token
	ready := make([]renderWaitResult, 0)
	channels := make([]chan renderWaitResult, 0)
	for id, waiter := range g.waiters {
		if token.Epoch <= waiter.after {
			continue
		}
		result := renderWaitResult{token: token}
		if waiter.expectedRevision != "" && token.Revision != waiter.expectedRevision {
			result.err = &RevisionMismatchError{Expected: waiter.expectedRevision, Observed: token}
		}
		channels = append(channels, waiter.ch)
		ready = append(ready, result)
		delete(g.waiters, id)
	}
	g.mu.Unlock()

	for i, ch := range channels {
		ch <- ready[i]
	}
	return true
}

// WaitAfter preserves legacy numeric wait semantics.
func (g *EpochGate) WaitAfter(ctx context.Context, after uint64) (uint64, error) {
	token, err := g.WaitAfterRevision(ctx, after, "")
	return token.Epoch, err
}

// WaitAfterRevision waits for a render strictly newer than after and, when an
// expected revision is supplied, requires that exact revision. A newer numeric
// epoch carrying another revision fails closed instead of releasing stale UI.
func (g *EpochGate) WaitAfterRevision(ctx context.Context, after uint64, expectedRevision string) (RenderToken, error) {
	expectedRevision = strings.TrimSpace(expectedRevision)
	g.mu.Lock()
	if g.current.Epoch > after {
		current := g.current
		g.mu.Unlock()
		if expectedRevision != "" && current.Revision != expectedRevision {
			return RenderToken{}, &RevisionMismatchError{Expected: expectedRevision, Observed: current}
		}
		return current, nil
	}
	g.nextID++
	id := g.nextID
	ch := make(chan renderWaitResult, 1)
	g.waiters[id] = epochWaiter{after: after, expectedRevision: expectedRevision, ch: ch}
	g.mu.Unlock()

	select {
	case result := <-ch:
		return result.token, result.err
	case <-ctx.Done():
		g.mu.Lock()
		delete(g.waiters, id)
		g.mu.Unlock()
		return RenderToken{}, ctx.Err()
	}
}

// ValidateCurrentRevision verifies a non-waiting capture request against the
// latest render token.
func (g *EpochGate) ValidateCurrentRevision(after uint64, expectedRevision string) (RenderToken, error) {
	token := g.CurrentToken()
	if token.Epoch < after {
		return RenderToken{}, fmt.Errorf("fastcdp: current render epoch %d is older than required %d", token.Epoch, after)
	}
	expectedRevision = strings.TrimSpace(expectedRevision)
	if expectedRevision != "" && token.Revision != expectedRevision {
		return RenderToken{}, &RevisionMismatchError{Expected: expectedRevision, Observed: token}
	}
	return token, nil
}
