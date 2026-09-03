package fastcdp

import (
	"errors"
	"testing"
)

func TestRecoveryPolicyUsesSmallestResetFirst(t *testing.T) {
	policy := DefaultRecoveryPolicy()
	state := RecoveryState{}

	first := policy.Decide(state, FailureCommandTimeout)
	if first.Reset != ResetComponent || !first.Retry {
		t.Fatalf("first timeout = %#v, want component retry", first)
	}
	second := policy.Decide(first.Next, FailureCommandTimeout)
	if second.Reset != ResetPage {
		t.Fatalf("second timeout reset = %v, want page", second.Reset)
	}
}

func TestRecoveryPolicyEscalatesRepeatedTargetLossToContext(t *testing.T) {
	policy := DefaultRecoveryPolicy()
	first := policy.Decide(RecoveryState{}, FailureTargetClosed)
	if first.Reset != ResetPage {
		t.Fatalf("first target loss reset = %v, want page", first.Reset)
	}
	second := policy.Decide(first.Next, FailureTargetClosed)
	if second.Reset != ResetContext {
		t.Fatalf("second target loss reset = %v, want context", second.Reset)
	}
}

func TestRecoveryPolicyTransportLossRequiresBrowserReset(t *testing.T) {
	decision := DefaultRecoveryPolicy().Decide(RecoveryState{ConsecutiveTimeouts: 1}, FailureTransportLost)
	if decision.Reset != ResetBrowser || decision.Next != (RecoveryState{}) {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestRecoveryPolicySuccessClearsTransientState(t *testing.T) {
	decision := DefaultRecoveryPolicy().Decide(RecoveryState{ConsecutiveTimeouts: 1, ConsecutivePageFailures: 1}, FailureNone)
	if decision.Reset != ResetNone || decision.Retry || decision.Next != (RecoveryState{}) {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestClassifyErrorRecognizesProtocolTargetLoss(t *testing.T) {
	err := &ProtocolError{Code: -32001, Message: "No target with given id found"}
	if got := ClassifyError(err); got != FailureTargetClosed {
		t.Fatalf("kind = %v, want FailureTargetClosed", got)
	}
}

func TestClassifyErrorRecognizesClosedTransport(t *testing.T) {
	if got := ClassifyError(ErrClosed); got != FailureTransportLost {
		t.Fatalf("kind = %v, want FailureTransportLost", got)
	}
	if got := ClassifyError(errors.New("websocket: connection reset by peer")); got != FailureTransportLost {
		t.Fatalf("kind = %v, want FailureTransportLost", got)
	}
}
