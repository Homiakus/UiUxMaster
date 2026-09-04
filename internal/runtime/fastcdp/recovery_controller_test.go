package fastcdp

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
)

type mockResetHandler struct {
	componentCalls int32
	pageCalls      int32
	contextCalls   int32
	browserCalls   int32

	componentErr error
	pageErr      error
	contextErr   error
	browserErr   error
}

func (m *mockResetHandler) ResetComponent(ctx context.Context) error {
	atomic.AddInt32(&m.componentCalls, 1)
	return m.componentErr
}

func (m *mockResetHandler) ResetPage(ctx context.Context) error {
	atomic.AddInt32(&m.pageCalls, 1)
	return m.pageErr
}

func (m *mockResetHandler) ResetContext(ctx context.Context) error {
	atomic.AddInt32(&m.contextCalls, 1)
	return m.contextErr
}

func (m *mockResetHandler) ResetBrowser(ctx context.Context) error {
	atomic.AddInt32(&m.browserCalls, 1)
	return m.browserErr
}

func TestRecoveryController_SmallestResetWins_Component(t *testing.T) {
	ctx := context.Background()
	handler := &mockResetHandler{}
	rc := NewRecoveryController(handler)

	// Fault: Stale Epoch -> Smallest reset is component
	decision, err := rc.HandleFailure(ctx, FailureStaleEpoch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Reset != ResetComponent {
		t.Fatalf("expected ResetComponent, got %s", decision.Reset)
	}
	if atomic.LoadInt32(&handler.componentCalls) != 1 {
		t.Fatalf("expected 1 component call, got %d", handler.componentCalls)
	}
	if atomic.LoadInt32(&handler.pageCalls) != 0 || atomic.LoadInt32(&handler.contextCalls) != 0 || atomic.LoadInt32(&handler.browserCalls) != 0 {
		t.Fatalf("unexpected broader reset calls: page=%d, ctx=%d, browser=%d",
			handler.pageCalls, handler.contextCalls, handler.browserCalls)
	}

	stats := rc.Stats()
	if stats.ComponentResets != 1 || stats.TotalRecoveries != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestRecoveryController_TimeoutEscalatesToPage(t *testing.T) {
	ctx := context.Background()
	handler := &mockResetHandler{}
	rc := NewRecoveryController(handler, RecoveryControllerConfig{
		Policy: RecoveryPolicy{RepeatedTimeoutsToPage: 2},
	})

	// 1st timeout -> component reset
	dec1, err := rc.HandleFailure(ctx, FailureCommandTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec1.Reset != ResetComponent {
		t.Fatalf("expected ResetComponent on 1st timeout, got %s", dec1.Reset)
	}
	if atomic.LoadInt32(&handler.componentCalls) != 1 {
		t.Fatalf("expected 1 component call")
	}

	// 2nd timeout -> escalates to page reset
	dec2, err := rc.HandleFailure(ctx, FailureCommandTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec2.Reset != ResetPage {
		t.Fatalf("expected ResetPage on 2nd timeout, got %s", dec2.Reset)
	}
	if atomic.LoadInt32(&handler.pageCalls) != 1 {
		t.Fatalf("expected 1 page call, got %d", handler.pageCalls)
	}
}

func TestRecoveryController_TargetClosedEscalatesToContext(t *testing.T) {
	ctx := context.Background()
	handler := &mockResetHandler{}
	rc := NewRecoveryController(handler, RecoveryControllerConfig{
		Policy: RecoveryPolicy{RepeatedPageFailuresToContext: 2},
	})

	// 1st target closed -> page reset
	dec1, err := rc.HandleFailure(ctx, FailureTargetClosed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec1.Reset != ResetPage {
		t.Fatalf("expected ResetPage on 1st target closed, got %s", dec1.Reset)
	}
	if atomic.LoadInt32(&handler.pageCalls) != 1 {
		t.Fatalf("expected 1 page call")
	}

	// 2nd target closed -> context reset
	dec2, err := rc.HandleFailure(ctx, FailureTargetClosed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec2.Reset != ResetContext {
		t.Fatalf("expected ResetContext on 2nd target closed, got %s", dec2.Reset)
	}
	if atomic.LoadInt32(&handler.contextCalls) != 1 {
		t.Fatalf("expected 1 context call, got %d", handler.contextCalls)
	}
}

func TestRecoveryController_TransportLost_BrowserReset(t *testing.T) {
	ctx := context.Background()
	handler := &mockResetHandler{}
	rc := NewRecoveryController(handler)

	// Transport lost -> directly triggers ResetBrowser
	dec, err := rc.HandleFailure(ctx, FailureTransportLost)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Reset != ResetBrowser {
		t.Fatalf("expected ResetBrowser, got %s", dec.Reset)
	}
	if atomic.LoadInt32(&handler.browserCalls) != 1 {
		t.Fatalf("expected 1 browser reset call, got %d", handler.browserCalls)
	}
}

func TestRecoveryController_FaultInjection_LadderEscalationOnHandlerFailure(t *testing.T) {
	ctx := context.Background()
	// Inject fault: Component reset fails, forcing escalation to page reset
	handler := &mockResetHandler{
		componentErr: errors.New("fault: component DOM detached"),
	}
	rc := NewRecoveryController(handler)

	decision, err := rc.HandleFailure(ctx, FailureStaleEpoch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Component was attempted and failed
	if atomic.LoadInt32(&handler.componentCalls) != 1 {
		t.Fatalf("expected 1 component call")
	}
	// Escalated to page reset, which succeeded
	if atomic.LoadInt32(&handler.pageCalls) != 1 {
		t.Fatalf("expected 1 page call due to escalation")
	}
	if decision.Reset != ResetPage {
		t.Fatalf("expected decision to escalate to ResetPage, got %s", decision.Reset)
	}

	stats := rc.Stats()
	if stats.Escalations != 1 || stats.FailedResets != 1 || stats.PageResets != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestRecoveryController_ExecuteWithRecovery_TransientFault(t *testing.T) {
	ctx := context.Background()
	handler := &mockResetHandler{}
	rc := NewRecoveryController(handler)

	var attempts int
	err := rc.ExecuteWithRecovery(ctx, func(c context.Context) error {
		attempts++
		if attempts == 1 {
			// Inject transient CDP protocol error on attempt 1
			return &ProtocolError{Code: -32000, Message: "Target closed"}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected transient recovery to succeed, got: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if atomic.LoadInt32(&handler.pageCalls) != 1 {
		t.Fatalf("expected 1 page reset call to recover target closed")
	}
}

func TestRecoveryController_ExecuteWithRecovery_Exhaustion(t *testing.T) {
	ctx := context.Background()
	handler := &mockResetHandler{}
	rc := NewRecoveryController(handler, RecoveryControllerConfig{
		MaxAttempts: 2,
	})

	// Inject fatal permanent error
	err := rc.ExecuteWithRecovery(ctx, func(c context.Context) error {
		return fmt.Errorf("persistent fatal error")
	})

	if err == nil {
		t.Fatalf("expected error on permanent failure")
	}
}
