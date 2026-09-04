package fastcdp

import (
	"context"
	"fmt"
	"sync"
)

// ResetHandler defines the executable reset operations across the recovery ladder:
// component -> page -> context -> browser.
type ResetHandler interface {
	ResetComponent(ctx context.Context) error
	ResetPage(ctx context.Context) error
	ResetContext(ctx context.Context) error
	ResetBrowser(ctx context.Context) error
}

// RecoveryStats tracks cumulative execution counts of recovery actions.
type RecoveryStats struct {
	TotalRecoveries int `json:"total_recoveries"`
	ComponentResets int `json:"component_resets"`
	PageResets      int `json:"page_resets"`
	ContextResets   int `json:"context_resets"`
	BrowserResets   int `json:"browser_resets"`
	Escalations     int `json:"escalations"`
	FailedResets    int `json:"failed_resets"`
}

// RecoveryControllerConfig configures the recovery controller.
type RecoveryControllerConfig struct {
	Policy      RecoveryPolicy
	MaxAttempts int
}

// RecoveryController executes the progressive recovery ladder
// (component -> page -> context -> browser) whenever failures occur.
type RecoveryController struct {
	mu          sync.Mutex
	policy      RecoveryPolicy
	state       RecoveryState
	handler     ResetHandler
	maxAttempts int
	stats       RecoveryStats
}

// NewRecoveryController creates a new executable recovery controller.
func NewRecoveryController(handler ResetHandler, cfg ...RecoveryControllerConfig) *RecoveryController {
	c := RecoveryControllerConfig{
		Policy:      DefaultRecoveryPolicy(),
		MaxAttempts: 3,
	}
	if len(cfg) > 0 {
		if cfg[0].Policy.RepeatedTimeoutsToPage > 0 || cfg[0].Policy.RepeatedPageFailuresToContext > 0 {
			c.Policy = cfg[0].Policy
		}
		if cfg[0].MaxAttempts > 0 {
			c.MaxAttempts = cfg[0].MaxAttempts
		}
	}
	return &RecoveryController{
		policy:      c.Policy,
		handler:     handler,
		maxAttempts: c.MaxAttempts,
	}
}

// HandleFailure executes the appropriate reset action based on failure kind.
// If the selected reset level fails, it automatically escalates up the ladder.
func (rc *RecoveryController) HandleFailure(ctx context.Context, kind FailureKind) (RecoveryDecision, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	decision := rc.policy.Decide(rc.state, kind)
	rc.state = decision.Next
	rc.stats.TotalRecoveries++

	if decision.Reset == ResetNone || rc.handler == nil {
		return decision, nil
	}

	currentLevel := decision.Reset
	for currentLevel <= ResetBrowser {
		var err error
		switch currentLevel {
		case ResetComponent:
			rc.stats.ComponentResets++
			err = rc.handler.ResetComponent(ctx)
		case ResetPage:
			rc.stats.PageResets++
			err = rc.handler.ResetPage(ctx)
		case ResetContext:
			rc.stats.ContextResets++
			err = rc.handler.ResetContext(ctx)
		case ResetBrowser:
			rc.stats.BrowserResets++
			err = rc.handler.ResetBrowser(ctx)
		}

		if err == nil {
			// Recovery at currentLevel succeeded
			return decision, nil
		}

		// Reset action failed, escalate to next higher tier
		rc.stats.FailedResets++
		rc.stats.Escalations++
		currentLevel++
		if currentLevel <= ResetBrowser {
			decision.Reset = currentLevel
		}
	}

	return decision, fmt.Errorf("fastcdp: recovery ladder exhausted through browser reset")
}

// HandleError classifies the error and executes the recovery ladder.
func (rc *RecoveryController) HandleError(ctx context.Context, err error) (RecoveryDecision, error) {
	if err == nil {
		rc.mu.Lock()
		defer rc.mu.Unlock()
		decision := rc.policy.Decide(rc.state, FailureNone)
		rc.state = decision.Next
		return decision, nil
	}
	kind := ClassifyError(err)
	return rc.HandleFailure(ctx, kind)
}

// ExecuteWithRecovery executes an operation with automatic progressive recovery retries.
func (rc *RecoveryController) ExecuteWithRecovery(ctx context.Context, op func(ctx context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < rc.maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		opErr := op(ctx)
		if opErr == nil {
			_, _ = rc.HandleError(ctx, nil)
			return nil
		}

		lastErr = opErr
		decision, recErr := rc.HandleError(ctx, opErr)
		if recErr != nil {
			return fmt.Errorf("fastcdp: recovery failed: %w (original error: %v)", recErr, opErr)
		}
		if !decision.Retry {
			return opErr
		}
	}
	return fmt.Errorf("fastcdp: maximum recovery attempts (%d) exceeded: %w", rc.maxAttempts, lastErr)
}

// State returns a snapshot of the current recovery state.
func (rc *RecoveryController) State() RecoveryState {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.state
}

// Stats returns a snapshot of the recovery metrics.
func (rc *RecoveryController) Stats() RecoveryStats {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.stats
}

// ResetState clears transient failure counters.
func (rc *RecoveryController) ResetState() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.state = RecoveryState{}
}
