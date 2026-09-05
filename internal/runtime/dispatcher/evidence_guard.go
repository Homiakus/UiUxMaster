package dispatcher

import (
	"errors"
	"fmt"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

var (
	// ErrCollectorUnavailable means the policy-selected collector does not exist
	// in the configured runtime. It is an evidence insufficiency, never a signal
	// to substitute a weaker collector.
	ErrCollectorUnavailable = errors.New("dispatcher: required collector unavailable")
	// ErrInvalidRoute means a ValidationPlan contains a route the dispatcher does
	// not understand. Unknown routes fail closed instead of defaulting to L2.
	ErrInvalidRoute = errors.New("dispatcher: invalid validation route")
)

func unavailable(tier engine.EvidenceTier, collector string) error {
	return fmt.Errorf("%w: required=%s collector=%s", ErrCollectorUnavailable, tier, collector)
}

func attest(plan engine.ValidationPlan, packet evidence.Packet) error {
	if err := engine.ValidateCollectedEvidence(plan, packet); err != nil {
		return fmt.Errorf("dispatcher: evidence attestation: %w", err)
	}
	return nil
}
