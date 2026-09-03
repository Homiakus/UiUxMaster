package fastcdp

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEpochGateWaitAfterReleasesOnNewerEpoch(t *testing.T) {
	gate := NewEpochGate()
	gate.Advance(10)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan uint64, 1)
	errs := make(chan error, 1)
	go func() {
		epoch, err := gate.WaitAfter(ctx, 10)
		if err != nil {
			errs <- err
			return
		}
		done <- epoch
	}()

	if gate.Advance(10) {
		t.Fatal("duplicate epoch should not advance gate")
	}
	gate.Advance(11)

	select {
	case err := <-errs:
		t.Fatal(err)
	case epoch := <-done:
		if epoch != 11 {
			t.Fatalf("epoch = %d, want 11", epoch)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestEpochGateReturnsImmediatelyWhenAlreadyFresh(t *testing.T) {
	gate := NewEpochGate()
	gate.Advance(42)
	epoch, err := gate.WaitAfter(context.Background(), 41)
	if err != nil {
		t.Fatal(err)
	}
	if epoch != 42 {
		t.Fatalf("epoch = %d, want 42", epoch)
	}
}

func TestEpochGateCancellationRemovesWaiter(t *testing.T) {
	gate := NewEpochGate()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := gate.WaitAfter(ctx, 0)
		done <- err
	}()

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("WaitAfter error = %v, want context.Canceled", err)
	}
	gate.mu.Lock()
	defer gate.mu.Unlock()
	if len(gate.waiters) != 0 {
		t.Fatalf("waiters = %d, want 0", len(gate.waiters))
	}
}

func TestEpochGateReleasesMultipleThresholds(t *testing.T) {
	gate := NewEpochGate()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	results := make(chan uint64, 2)
	for _, after := range []uint64{3, 7} {
		after := after
		go func() {
			epoch, err := gate.WaitAfter(ctx, after)
			if err == nil {
				results <- epoch
			}
		}()
	}

	gate.Advance(8)
	for range 2 {
		select {
		case epoch := <-results:
			if epoch != 8 {
				t.Fatalf("epoch = %d, want 8", epoch)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}
