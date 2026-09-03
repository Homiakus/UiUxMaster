package fastcdp

import (
	"errors"
	"strings"
)

// ResetLevel follows the UiUxMaster invariant "smallest reset wins". Levels are
// ordered from cheapest/local to broadest/most expensive recovery.
type ResetLevel uint8

const (
	ResetNone ResetLevel = iota
	ResetComponent
	ResetPage
	ResetContext
	ResetBrowser
)

func (r ResetLevel) String() string {
	switch r {
	case ResetNone:
		return "none"
	case ResetComponent:
		return "component"
	case ResetPage:
		return "page"
	case ResetContext:
		return "context"
	case ResetBrowser:
		return "browser"
	default:
		return "unknown"
	}
}

// FailureKind is intentionally small and based on recovery semantics rather
// than vendor-specific error strings.
type FailureKind uint8

const (
	FailureNone FailureKind = iota
	FailureStaleEpoch
	FailureCommandTimeout
	FailureTargetClosed
	FailureContextCorrupt
	FailureTransportLost
	FailureProtocolIntegrity
)

type RecoveryPolicy struct {
	// RepeatedTimeoutsToPage escalates repeated command/readiness timeouts from a
	// component refresh to a page reset.
	RepeatedTimeoutsToPage int
	// RepeatedPageFailuresToContext escalates recurring target/page failures to a
	// fresh browser context.
	RepeatedPageFailuresToContext int
}

type RecoveryState struct {
	ConsecutiveTimeouts    int
	ConsecutivePageFailures int
}

type RecoveryDecision struct {
	Kind       FailureKind
	Reset      ResetLevel
	Retry      bool
	Next       RecoveryState
	Reason     string
}

func DefaultRecoveryPolicy() RecoveryPolicy {
	return RecoveryPolicy{RepeatedTimeoutsToPage: 2, RepeatedPageFailuresToContext: 2}
}

// Decide returns the cheapest recovery level that can plausibly repair the
// failure. A successful operation should call Decide with FailureNone to clear
// transient escalation counters.
func (p RecoveryPolicy) Decide(state RecoveryState, kind FailureKind) RecoveryDecision {
	p = p.normalized()
	decision := RecoveryDecision{Kind: kind, Retry: kind != FailureNone, Next: state}

	switch kind {
	case FailureNone:
		decision.Reset = ResetNone
		decision.Retry = false
		decision.Next = RecoveryState{}
		decision.Reason = "healthy"
	case FailureStaleEpoch:
		decision.Reset = ResetComponent
		decision.Reason = "render epoch did not advance; retry local app/HMR synchronization before broader reset"
	case FailureCommandTimeout:
		decision.Next.ConsecutiveTimeouts++
		if decision.Next.ConsecutiveTimeouts >= p.RepeatedTimeoutsToPage {
			decision.Reset = ResetPage
			decision.Next.ConsecutiveTimeouts = 0
			decision.Next.ConsecutivePageFailures++
			decision.Reason = "repeated CDP/readiness timeout; recreate or reload only the affected page"
		} else {
			decision.Reset = ResetComponent
			decision.Reason = "single timeout; retry affected component/state before page reset"
		}
	case FailureTargetClosed:
		decision.Next.ConsecutiveTimeouts = 0
		decision.Next.ConsecutivePageFailures++
		if decision.Next.ConsecutivePageFailures >= p.RepeatedPageFailuresToContext {
			decision.Reset = ResetContext
			decision.Next.ConsecutivePageFailures = 0
			decision.Reason = "repeated target loss; recreate browser context"
		} else {
			decision.Reset = ResetPage
			decision.Reason = "target/page disappeared; recreate only the page/session"
		}
	case FailureContextCorrupt:
		decision.Reset = ResetContext
		decision.Next = RecoveryState{}
		decision.Reason = "context state is invalid; isolate recovery to browser context"
	case FailureTransportLost, FailureProtocolIntegrity:
		decision.Reset = ResetBrowser
		decision.Next = RecoveryState{}
		decision.Reason = "CDP transport/protocol integrity lost; browser connection must be rebuilt"
	default:
		decision.Reset = ResetBrowser
		decision.Next = RecoveryState{}
		decision.Reason = "unknown recovery class; fail conservative at browser boundary"
	}
	return decision
}

func (p RecoveryPolicy) normalized() RecoveryPolicy {
	if p.RepeatedTimeoutsToPage < 1 {
		p.RepeatedTimeoutsToPage = 2
	}
	if p.RepeatedPageFailuresToContext < 1 {
		p.RepeatedPageFailuresToContext = 2
	}
	return p
}

// ClassifyError converts common transport/protocol symptoms into recovery
// semantics. Callers should prefer explicit FailureKinds when they know the
// source; this fallback exists for adapter boundaries.
func ClassifyError(err error) FailureKind {
	if err == nil {
		return FailureNone
	}
	if errors.Is(err, ErrClosed) {
		return FailureTransportLost
	}
	var protocolErr *ProtocolError
	if errors.As(err, &protocolErr) {
		message := strings.ToLower(protocolErr.Message)
		switch {
		case strings.Contains(message, "target closed"), strings.Contains(message, "no target with given id"), strings.Contains(message, "session with given id"):
			return FailureTargetClosed
		default:
			return FailureProtocolIntegrity
		}
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "deadline exceeded"), strings.Contains(message, "timeout"):
		return FailureCommandTimeout
	case strings.Contains(message, "target closed"):
		return FailureTargetClosed
	case strings.Contains(message, "websocket"), strings.Contains(message, "broken pipe"), strings.Contains(message, "connection reset"), strings.Contains(message, "unexpected eof"):
		return FailureTransportLost
	default:
		return FailureProtocolIntegrity
	}
}
