package fastcdp

import (
	"context"
	"sync"
)

// EpochGate tracks the newest application render epoch observed from the
// resident browser. It is intentionally independent from how the browser emits
// the signal (Runtime binding, console bridge, explicit poll, etc.).
type EpochGate struct {
	mu      sync.Mutex
	current uint64
	nextID  uint64
	waiters map[uint64]epochWaiter
}

type epochWaiter struct {
	after uint64
	ch    chan uint64
}

func NewEpochGate() *EpochGate {
	return &EpochGate{waiters: make(map[uint64]epochWaiter)}
}

func (g *EpochGate) Current() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.current
}

// Advance publishes a monotonically increasing render epoch. Stale or duplicate
// values are ignored. Every waiter whose threshold has been crossed is released.
func (g *EpochGate) Advance(epoch uint64) bool {
	g.mu.Lock()
	if epoch <= g.current {
		g.mu.Unlock()
		return false
	}
	g.current = epoch
	ready := make([]epochWaiter, 0)
	for id, waiter := range g.waiters {
		if epoch > waiter.after {
			ready = append(ready, waiter)
			delete(g.waiters, id)
		}
	}
	g.mu.Unlock()

	for _, waiter := range ready {
		waiter.ch <- epoch
	}
	return true
}

// WaitAfter waits until an epoch strictly newer than after is observed. It
// handles the race where Advance happens between the caller's last read and
// waiter registration by checking current under the same mutex.
func (g *EpochGate) WaitAfter(ctx context.Context, after uint64) (uint64, error) {
	g.mu.Lock()
	if g.current > after {
		current := g.current
		g.mu.Unlock()
		return current, nil
	}
	g.nextID++
	id := g.nextID
	ch := make(chan uint64, 1)
	g.waiters[id] = epochWaiter{after: after, ch: ch}
	g.mu.Unlock()

	select {
	case epoch := <-ch:
		return epoch, nil
	case <-ctx.Done():
		g.mu.Lock()
		delete(g.waiters, id)
		g.mu.Unlock()
		return 0, ctx.Err()
	}
}
