package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Homiakus/axiom/adgo"
)

const (
	planID      = "uiux.design-validation"
	planVersion = "1"

	activityPlanEvidence = "PlanEvidence"
	activityCollectVerify = "CollectVerify"
	activityDecide        = "Decide"

	dataChange           = "change"
	dataEvidencePlan     = "evidencePlan"
	dataValidationResult = "validationResult"
	dataDecision         = "decision"
)

type Runner struct {
	runtime *adgo.Runtime
	store   adgo.Store
	plan    *adgo.Plan
}

func NewMemory(executor Executor) (*Runner, error) {
	return newRunner(executor, adgo.NewMemoryStore())
}

// NewFile keeps run history across process restarts while retaining the same
// compact embedded Runtime semantics. Production worker separation can later
// replace this constructor without changing the public Run projection.
func NewFile(executor Executor, root string) (*Runner, error) {
	if root == "" {
		return nil, fmt.Errorf("controlplane: file-store root is required")
	}
	store, err := adgo.NewFileStore(filepath.Clean(root))
	if err != nil {
		return nil, fmt.Errorf("controlplane: open Axiom file store: %w", err)
	}
	return newRunner(executor, store)
}

func newRunner(executor Executor, store adgo.Store) (*Runner, error) {
	if executor == nil {
		return nil, fmt.Errorf("controlplane: executor is required")
	}
	plan, err := adgo.Compile(definition())
	if err != nil {
		return nil, fmt.Errorf("controlplane: compile validation plan: %w", err)
	}
	registry := adgo.NewRegistry()
	registerActivities(registry, executor)
	runtime, err := adgo.NewRuntime(plan, store, registry)
	if err != nil {
		return nil, fmt.Errorf("controlplane: create Axiom runtime: %w", err)
	}
	return &Runner{runtime: runtime, store: store, plan: plan}, nil
}

func definition() adgo.Definition {
	return adgo.Definition{
		ID:          planID,
		Version:     planVersion,
		InitialData: []string{dataChange},
		Nodes: []adgo.Node{
			{
				ID:       "plan_evidence",
				Kind:     adgo.NodeActivity,
				Activity: activityPlanEvidence,
				Requires: []string{dataChange},
				Produces: []string{dataEvidencePlan},
				Next:     []adgo.Transition{{To: "collect_verify"}},
			},
			{
				ID:        "collect_verify",
				Kind:      adgo.NodeActivity,
				Activity:  activityCollectVerify,
				DependsOn: []string{"plan_evidence"},
				Requires:  []string{dataChange, dataEvidencePlan},
				Produces:  []string{dataValidationResult},
				Next:      []adgo.Transition{{To: "decide"}},
			},
			{
				ID:        "decide",
				Kind:      adgo.NodeActivity,
				Activity:  activityDecide,
				DependsOn: []string{"collect_verify"},
				Requires:  []string{dataChange, dataEvidencePlan, dataValidationResult},
				Produces:  []string{dataDecision},
			},
		},
	}
}

func registerActivities(registry *adgo.Registry, executor Executor) {
	registry.Activity(activityPlanEvidence, func(ctx context.Context, req adgo.ActivityRequest) (adgo.ActivityResult, error) {
		change, err := decodeRequestData[Change](req, dataChange)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		plan, err := executor.PlanEvidence(ctx, change)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		return adgo.ActivityResult{
			Facts:   map[string]any{dataEvidencePlan: plan},
			Outcome: adgo.OutcomeCompleted,
		}, nil
	})

	registry.Activity(activityCollectVerify, func(ctx context.Context, req adgo.ActivityRequest) (adgo.ActivityResult, error) {
		change, err := decodeRequestData[Change](req, dataChange)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		plan, err := decodeRequestData[EvidencePlan](req, dataEvidencePlan)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		result, err := executor.CollectVerify(ctx, change, plan)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		usage := adgo.BudgetUsage{BrowserFetches: 1}
		if plan.Pixels {
			usage.BrowserFetches++
		}
		return adgo.ActivityResult{
			Facts:   map[string]any{dataValidationResult: result},
			Budget:  usage,
			Outcome: adgo.OutcomeCompleted,
		}, nil
	})

	registry.Activity(activityDecide, func(ctx context.Context, req adgo.ActivityRequest) (adgo.ActivityResult, error) {
		change, err := decodeRequestData[Change](req, dataChange)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		plan, err := decodeRequestData[EvidencePlan](req, dataEvidencePlan)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		result, err := decodeRequestData[ValidationResult](req, dataValidationResult)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		decision, err := executor.Decide(ctx, change, plan, result)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		return adgo.ActivityResult{
			Facts:   map[string]any{dataDecision: decision},
			Outcome: adgo.OutcomeCompleted,
		}, nil
	})
}

func (r *Runner) Start(ctx context.Context, id string, change Change, budget Budget) (Run, error) {
	if r == nil || r.runtime == nil {
		return Run{}, fmt.Errorf("controlplane: runner is not initialized")
	}
	execution, err := r.runtime.Start(ctx, id, map[string]any{dataChange: change}, toAxiomBudget(budget))
	if err != nil {
		return Run{}, err
	}
	return projectExecution(execution)
}

func (r *Runner) Run(ctx context.Context, id string) (Run, error) {
	if r == nil || r.runtime == nil {
		return Run{}, fmt.Errorf("controlplane: runner is not initialized")
	}
	execution, err := r.runtime.Run(ctx, id)
	if err != nil && execution == nil {
		return Run{}, err
	}
	if execution == nil {
		return Run{}, err
	}
	projected, projectErr := projectExecution(execution)
	if projectErr != nil {
		return Run{}, projectErr
	}
	if err != nil {
		return projected, err
	}
	return projected, nil
}

func (r *Runner) StartAndRun(ctx context.Context, id string, change Change, budget Budget) (Run, error) {
	if _, err := r.Start(ctx, id, change, budget); err != nil {
		return Run{}, err
	}
	return r.Run(ctx, id)
}

// Cancel is durable with respect to the selected Axiom store. The event is
// ingested by the same deterministic runtime before any further activity runs.
func (r *Runner) Cancel(ctx context.Context, id string) (Run, error) {
	if r == nil || r.runtime == nil {
		return Run{}, fmt.Errorf("controlplane: runner is not initialized")
	}
	if err := r.runtime.Signal(ctx, id, adgo.Event{ID: "cancel:" + id, Type: "CancelRequested"}); err != nil {
		return Run{}, err
	}
	return r.Run(ctx, id)
}

func (r *Runner) Load(ctx context.Context, id string) (Run, error) {
	if r == nil || r.store == nil {
		return Run{}, fmt.Errorf("controlplane: runner is not initialized")
	}
	execution, err := r.store.Load(ctx, id)
	if err != nil {
		return Run{}, err
	}
	return projectExecution(execution)
}

func (r *Runner) History(ctx context.Context, id string) ([]HistoryEntry, error) {
	run, err := r.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	return append([]HistoryEntry(nil), run.History...), nil
}

func (r *Runner) PlanDigest() string {
	if r == nil || r.plan == nil {
		return ""
	}
	return r.plan.Digest
}

func decodeRequestData[T any](req adgo.ActivityRequest, key string) (T, error) {
	var zero T
	raw, ok := req.Data[key]
	if !ok {
		return zero, fmt.Errorf("controlplane: missing activity data %q", key)
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return zero, fmt.Errorf("controlplane: decode activity data %q: %w", key, err)
	}
	return value, nil
}

func decodeExecutionData[T any](execution *adgo.Execution, key string) (T, error) {
	var zero T
	if execution == nil {
		return zero, fmt.Errorf("controlplane: nil execution")
	}
	raw, ok := execution.Data[key]
	if !ok {
		return zero, nil
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return zero, fmt.Errorf("controlplane: decode execution data %q: %w", key, err)
	}
	return value, nil
}

func projectExecution(execution *adgo.Execution) (Run, error) {
	change, err := decodeExecutionData[Change](execution, dataChange)
	if err != nil {
		return Run{}, err
	}
	plan, err := decodeExecutionData[EvidencePlan](execution, dataEvidencePlan)
	if err != nil {
		return Run{}, err
	}
	result, err := decodeExecutionData[ValidationResult](execution, dataValidationResult)
	if err != nil {
		return Run{}, err
	}
	decision, err := decodeExecutionData[Decision](execution, dataDecision)
	if err != nil {
		return Run{}, err
	}
	history := make([]HistoryEntry, 0, len(execution.History))
	for _, item := range execution.History {
		history = append(history, HistoryEntry{
			Sequence: item.Seq,
			At:       item.At,
			Type:     item.Type,
			NodeID:   item.NodeID,
			Message:  item.Message,
			Data:     item.Data,
		})
	}
	return Run{
		ID:          execution.ID,
		Status:      string(execution.Status),
		PlanID:      execution.PlanID,
		PlanVersion: execution.PlanVersion,
		PlanDigest:  execution.PlanDigest,
		Change:      change,
		Evidence:    plan,
		Validation:  result,
		Decision:    decision,
		Usage: Usage{
			Cost:           execution.BudgetUsage.Cost,
			Tokens:         execution.BudgetUsage.Tokens,
			ActiveDuration: execution.BudgetUsage.ActiveDuration,
			LLMCalls:       execution.BudgetUsage.LLMCalls,
			SearchQueries:  execution.BudgetUsage.SearchQueries,
			BrowserFetches: execution.BudgetUsage.BrowserFetches,
		},
		Failure: execution.Failure,
		History: history,
	}, nil
}

func toAxiomBudget(value Budget) adgo.BudgetLimit {
	return adgo.BudgetLimit{
		MaxCost:           value.MaxCost,
		MaxTokens:         value.MaxTokens,
		MaxDuration:       value.MaxDuration,
		MaxLLMCalls:       value.MaxLLMCalls,
		MaxSearchQueries:  value.MaxSearchQueries,
		MaxBrowserFetches: value.MaxBrowserFetches,
	}
}
