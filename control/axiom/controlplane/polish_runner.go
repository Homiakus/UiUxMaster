package controlplane

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Homiakus/axiom/adgo"
)

const (
	polishPlanID      = "uiux.design-polish"
	polishPlanVersion = "1"

	activityPolishInspect  = "PolishInspect"
	activityPolishIterate  = "PolishIterate"
	activityPolishConclude = "PolishConclude"

	dataPolishRequest    = "polishRequest"
	dataPolishBaseline   = "polishBaseline"
	dataPolishIterations = "polishIterations"
	dataPolishResult     = "polishResult"
)

type PolishRunner struct {
	runtime *adgo.Runtime
	store   adgo.Store
	plan    *adgo.Plan
}

func NewPolishMemory(executor DesignPolishExecutor) (*PolishRunner, error) {
	return newPolishRunner(executor, adgo.NewMemoryStore())
}

func NewPolishFile(executor DesignPolishExecutor, root string) (*PolishRunner, error) {
	if root == "" { return nil, fmt.Errorf("controlplane: file-store root is required") }
	store, err := adgo.NewFileStore(filepath.Clean(root))
	if err != nil { return nil, fmt.Errorf("controlplane: open Axiom file store: %w", err) }
	return newPolishRunner(executor, store)
}

func newPolishRunner(executor DesignPolishExecutor, store adgo.Store) (*PolishRunner, error) {
	if executor == nil { return nil, fmt.Errorf("controlplane: polish executor is required") }
	plan, err := adgo.Compile(polishDefinition())
	if err != nil { return nil, fmt.Errorf("controlplane: compile polish plan: %w", err) }
	registry := adgo.NewRegistry()
	registerPolishActivities(registry, executor)
	runtime, err := adgo.NewRuntime(plan, store, registry)
	if err != nil { return nil, fmt.Errorf("controlplane: create Axiom runtime: %w", err) }
	return &PolishRunner{runtime: runtime, store: store, plan: plan}, nil
}

func polishDefinition() adgo.Definition {
	return adgo.Definition{
		ID: polishPlanID, Version: polishPlanVersion, InitialData: []string{dataPolishRequest},
		Nodes: []adgo.Node{
			{ID: "inspect_baseline", Kind: adgo.NodeActivity, Activity: activityPolishInspect, Requires: []string{dataPolishRequest}, Produces: []string{dataPolishBaseline}, Next: []adgo.Transition{{To: "iterate_repair"}}},
			{
				ID: "iterate_repair", Kind: adgo.NodeActivity, Activity: activityPolishIterate,
				DependsOn: []string{"inspect_baseline"}, Requires: []string{dataPolishRequest, dataPolishBaseline}, Produces: []string{dataPolishIterations},
				Next: []adgo.Transition{{To: "conclude_polish"}},
				ExternalEffect: true,
				Timeout: 2 * time.Minute,
				IdempotencyKey: "{execution}:{node}",
				Retry: adgo.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond, MaxRetryDuration: time.Second},
			},
			{ID: "conclude_polish", Kind: adgo.NodeActivity, Activity: activityPolishConclude, DependsOn: []string{"iterate_repair"}, Requires: []string{dataPolishRequest, dataPolishIterations}, Produces: []string{dataPolishResult}},
		},
	}
}

func registerPolishActivities(registry *adgo.Registry, executor DesignPolishExecutor) {
	registry.Activity(activityPolishInspect, func(ctx context.Context, req adgo.ActivityRequest) (adgo.ActivityResult, error) {
		ctx = withActivityIdentity(ctx, req)
		polishReq, err := decodeRequestData[DesignPolishRequest](req, dataPolishRequest)
		if err != nil { return adgo.ActivityResult{}, err }
		iter, err := executor.InspectBaseline(ctx, polishReq)
		if err != nil { return adgo.ActivityResult{}, err }
		return adgo.ActivityResult{Facts: map[string]any{dataPolishBaseline: iter}, Outcome: adgo.OutcomeCompleted}, nil
	})

	registry.Activity(activityPolishIterate, func(ctx context.Context, req adgo.ActivityRequest) (adgo.ActivityResult, error) {
		ctx = withActivityIdentity(ctx, req)
		polishReq, err := decodeRequestData[DesignPolishRequest](req, dataPolishRequest)
		if err != nil { return adgo.ActivityResult{}, err }
		baseline, err := decodeRequestData[PolishIteration](req, dataPolishBaseline)
		if err != nil { return adgo.ActivityResult{}, err }
		maxIters := polishReq.MaxIterations
		if maxIters <= 0 { maxIters = 3 }
		iters := []PolishIteration{baseline}
		for i := 1; i <= maxIters; i++ {
			iter, err := executor.StepPolish(ctx, polishReq, i)
			if err != nil { return adgo.ActivityResult{}, err }
			iters = append(iters, iter)
			if iter.HardViolations == 0 && iter.FindingsCount == 0 { break }
		}
		return adgo.ActivityResult{Facts: map[string]any{dataPolishIterations: iters}, Outcome: adgo.OutcomeCompleted}, nil
	})

	registry.Activity(activityPolishConclude, func(ctx context.Context, req adgo.ActivityRequest) (adgo.ActivityResult, error) {
		ctx = withActivityIdentity(ctx, req)
		polishReq, err := decodeRequestData[DesignPolishRequest](req, dataPolishRequest)
		if err != nil { return adgo.ActivityResult{}, err }
		iters, err := decodeRequestData[[]PolishIteration](req, dataPolishIterations)
		if err != nil { return adgo.ActivityResult{}, err }
		res, err := executor.ConcludePolish(ctx, polishReq, iters)
		if err != nil { return adgo.ActivityResult{}, err }
		return adgo.ActivityResult{Facts: map[string]any{dataPolishResult: res}, Outcome: adgo.OutcomeCompleted}, nil
	})
}

func (r *PolishRunner) Start(ctx context.Context, id string, req DesignPolishRequest, budget Budget) (DesignPolishRun, error) {
	if r == nil || r.runtime == nil { return DesignPolishRun{}, fmt.Errorf("controlplane: polish runner is not initialized") }
	exec, err := r.runtime.Start(ctx, id, map[string]any{dataPolishRequest: req}, toAxiomBudget(budget))
	if err != nil { return DesignPolishRun{}, err }
	return projectPolishExecution(exec)
}

func (r *PolishRunner) Run(ctx context.Context, id string) (DesignPolishRun, error) {
	if r == nil || r.runtime == nil { return DesignPolishRun{}, fmt.Errorf("controlplane: polish runner is not initialized") }
	exec, runErr := r.runtime.Run(ctx, id)
	if runErr != nil && exec == nil { return DesignPolishRun{}, runErr }
	if exec == nil { return DesignPolishRun{}, runErr }
	projected, err := projectPolishExecution(exec)
	if err != nil { return DesignPolishRun{}, err }
	if runErr != nil { return projected, runErr }
	return projected, nil
}

func (r *PolishRunner) Load(ctx context.Context, id string) (DesignPolishRun, error) {
	if r == nil || r.store == nil { return DesignPolishRun{}, fmt.Errorf("controlplane: polish runner is not initialized") }
	exec, err := r.store.Load(ctx, id)
	if err != nil { return DesignPolishRun{}, err }
	return projectPolishExecution(exec)
}

func (r *PolishRunner) StartAndRun(ctx context.Context, id string, req DesignPolishRequest, budget Budget) (DesignPolishRun, error) {
	if _, err := r.Start(ctx, id, req, budget); err != nil { return DesignPolishRun{}, err }
	return r.Run(ctx, id)
}

func projectPolishExecution(execution *adgo.Execution) (DesignPolishRun, error) {
	if execution == nil { return DesignPolishRun{}, fmt.Errorf("controlplane: nil execution") }
	req, err := decodeExecutionData[DesignPolishRequest](execution, dataPolishRequest)
	if err != nil { return DesignPolishRun{}, err }
	res, err := decodeExecutionData[DesignPolishResult](execution, dataPolishResult)
	if err != nil { return DesignPolishRun{}, err }
	history := make([]HistoryEntry, 0, len(execution.History))
	for _, item := range execution.History {
		history = append(history, HistoryEntry{Sequence: item.Seq, At: item.At, Type: item.Type, NodeID: item.NodeID, Message: item.Message, Data: item.Data})
	}
	return DesignPolishRun{
		ID: execution.ID, Status: string(execution.Status), PlanID: execution.PlanID, Request: req, Result: res,
		Usage: Usage{Cost: execution.BudgetUsage.Cost, Tokens: execution.BudgetUsage.Tokens, ActiveDuration: execution.BudgetUsage.ActiveDuration, LLMCalls: execution.BudgetUsage.LLMCalls, SearchQueries: execution.BudgetUsage.SearchQueries, BrowserFetches: execution.BudgetUsage.BrowserFetches},
		Failure: execution.Failure, History: history,
	}, nil
}
