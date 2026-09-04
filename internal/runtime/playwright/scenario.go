package playwright

import (
	"errors"
	"fmt"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// Supported scenario action kinds.
const (
	ActionClick     = "click"
	ActionDblClick  = "dblclick"
	ActionFill      = "fill"
	ActionHover     = "hover"
	ActionScroll    = "scroll"
	ActionResize    = "resize"
	ActionWait      = "wait"
	ActionPress     = "press"
	ActionFocus     = "focus"
	ActionCheck     = "check"
	ActionUncheck   = "uncheck"
	ActionSelect    = "select"
)

// DeterministicControls configures clean-state deterministic browser behavior.
type DeterministicControls struct {
	PauseAnimations   bool    `json:"pause_animations"`
	FreezeClock       bool    `json:"freeze_clock"`
	ReducedMotion     bool    `json:"reduced_motion"`
	EmulateMedia      string  `json:"emulate_media,omitempty"` // "screen" or "print"
	DeviceScaleFactor float64 `json:"device_scale_factor,omitempty"`
	Timezone          string  `json:"timezone,omitempty"`
	Locale            string  `json:"locale,omitempty"`
}

// DefaultDeterministicControls returns reproducible baseline settings.
func DefaultDeterministicControls() DeterministicControls {
	return DeterministicControls{
		PauseAnimations:   true,
		FreezeClock:       true,
		ReducedMotion:     true,
		DeviceScaleFactor: 1.0,
		Timezone:          "UTC",
		Locale:            "en-US",
	}
}

// StepCheckpoint records intermediate state during multi-step scenario execution.
type StepCheckpoint struct {
	StepIndex   int                     `json:"step_index"`
	ActionKind  string                  `json:"action_kind"`
	Target      string                  `json:"target,omitempty"`
	Success     bool                    `json:"success"`
	DurationMS  float64                 `json:"duration_ms"`
	AriaTree    []evidence.AccessibilityNode `json:"aria_tree,omitempty"`
	VisualROI   *evidence.VisualRegion  `json:"visual_roi,omitempty"`
	Issues      []evidence.RuntimeIssue `json:"issues,omitempty"`
}

// ScenarioReport details the execution trace of a scenario.
type ScenarioReport struct {
	ScenarioID  string           `json:"scenario_id"`
	Completed   bool             `json:"completed"`
	TotalSteps  int              `json:"total_steps"`
	TotalMS     float64          `json:"total_ms"`
	Checkpoints []StepCheckpoint `json:"checkpoints,omitempty"`
	Error       string           `json:"error,omitempty"`
}

// ValidateScenario checks action sequence correctness before dispatching to the browser.
func ValidateScenario(s Scenario) error {
	if s.ID == "" {
		return errors.New("scenario ID must not be empty")
	}
	if len(s.Actions) == 0 {
		return errors.New("scenario must contain at least one action")
	}

	for i, action := range s.Actions {
		switch action.Kind {
		case ActionClick, ActionDblClick, ActionHover, ActionFocus, ActionCheck, ActionUncheck:
			if action.Selector == "" {
				return fmt.Errorf("action[%d] (%s) requires a selector", i, action.Kind)
			}
		case ActionFill:
			if action.Selector == "" {
				return fmt.Errorf("action[%d] (fill) requires a selector", i)
			}
		case ActionPress:
			if action.Value == "" {
				return fmt.Errorf("action[%d] (press) requires a key value", i)
			}
		case ActionScroll:
			// selector is optional for scrolling whole window
		case ActionResize:
			if action.Value == "" {
				return fmt.Errorf("action[%d] (resize) requires dimension value (e.g. 1024x768)", i)
			}
		case ActionWait:
			if action.Duration <= 0 && action.Value == "" && action.Selector == "" {
				return fmt.Errorf("action[%d] (wait) requires duration, selector, or condition value", i)
			}
		case ActionSelect:
			if action.Selector == "" || action.Value == "" {
				return fmt.Errorf("action[%d] (select) requires selector and option value", i)
			}
		default:
			return fmt.Errorf("action[%d] has unsupported kind %q", i, action.Kind)
		}
	}

	return nil
}
