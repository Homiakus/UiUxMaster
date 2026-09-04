package controlplane

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Homiakus/axiom/adgo"
)

const (
	comparisonPlanID      = "uiux.candidate-comparison"
	comparisonPlanVersion = "1"

	activityEvaluateCandidates = "EvaluateCandidates"
	activityConcludeComparison = "ConcludeComparison"

	dataComparisonRequest = "comparisonRequest"
	dataCandidateRanks    = "candidateRanks"
	dataComparisonResult  = "comparisonResult"
)

type ComparisonRunner struct {
	runtime *adgo.Runtime
	store   adgo.Store
	plan    *adgo.Plan
}

func NewComparisonMemory(executor CandidateComparisonExecutor) (*ComparisonRunner, error) {
	return newComparisonRunner(executor, adgo.NewMemoryStore())
}

func NewComparisonFile(executor CandidateComparisonExecutor, root string) (*ComparisonRunner, error) {
	if root == "" {
		return nil, fmt.Errorf("controlplane: file-store root is required")
	}
	store, err := adgo.NewFileStore(filepath.Clean(root))
	if err != nil {
		return nil, fmt.Errorf("controlplane: open Axiom file store: %w", err)
	}
	return newComparisonRunner(executor, store)
}

func newComparisonRunner(executor CandidateComparisonExecutor, store adgo.Store) (*ComparisonRunner, error) {
	if executor == nil {
		return nil, fmt.Errorf("controlplane: comparison executor is required")
	}
	plan, err := adgo.Compile(comparisonDefinition())
	if err != nil {
		return nil, fmt.Errorf("controlplane: compile comparison plan: %w", err)
	}
	registry := adgo.NewRegistry()
	registerComparisonActivities(registry, executor)
	runtime, err := adgo.NewRuntime(plan, store, registry)
	if err != nil {
		return nil, fmt.Errorf("controlplane: create Axiom runtime: %w", err)
	}
	return &ComparisonRunner{runtime: runtime, store: store, plan: plan}, nil
}

func comparisonDefinition() adgo.Definition {
	return adgo.Definition{
		ID:          comparisonPlanID,
		Version:     comparisonPlanVersion,
		InitialData: []string{dataComparisonRequest},
		Nodes: []adgo.Node{
			{
				ID:       "evaluate_candidates",
				Kind:     adgo.NodeActivity,
				Activity: activityEvaluateCandidates,
				Requires: []string{dataComparisonRequest},
				Produces: []string{dataCandidateRanks},
				Next:     []adgo.Transition{{To: "conclude_comparison"}},
			},
			{
				ID:        "conclude_comparison",
				Kind:      adgo.NodeActivity,
				Activity:  activityConcludeComparison,
				DependsOn: []string{"evaluate_candidates"},
				Requires:  []string{dataComparisonRequest, dataCandidateRanks},
				Produces:  []string{dataComparisonResult},
			},
		},
	}
}

func registerComparisonActivities(registry *adgo.Registry, executor CandidateComparisonExecutor) {
	registry.Activity(activityEvaluateCandidates, func(ctx context.Context, req adgo.ActivityRequest) (adgo.ActivityResult, error) {
		cmpReq, err := decodeRequestData[CandidateComparisonRequest](req, dataComparisonRequest)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		ranks, err := executor.EvaluateCandidates(ctx, cmpReq)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		return adgo.ActivityResult{
			Facts:   map[string]any{dataCandidateRanks: ranks},
			Outcome: adgo.OutcomeCompleted,
		}, nil
	})

	registry.Activity(activityConcludeComparison, func(ctx context.Context, req adgo.ActivityRequest) (adgo.ActivityResult, error) {
		cmpReq, err := decodeRequestData[CandidateComparisonRequest](req, dataComparisonRequest)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		ranks, err := decodeRequestData[[]CandidateRank](req, dataCandidateRanks)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		res, err := executor.ConcludeComparison(ctx, cmpReq, ranks)
		if err != nil {
			return adgo.ActivityResult{}, err
		}
		return adgo.ActivityResult{
			Facts:   map[string]any{dataComparisonResult: res},
			Outcome: adgo.OutcomeCompleted,
		}, nil
	})
}

func (r *ComparisonRunner) StartAndRun(ctx context.Context, id string, req CandidateComparisonRequest, budget Budget) (CandidateComparisonRun, error) {
	if r == nil || r.runtime == nil {
		return CandidateComparisonRun{}, fmt.Errorf("controlplane: comparison runner is not initialized")
	}
	exec, err := r.runtime.Start(ctx, id, map[string]any{dataComparisonRequest: req}, toAxiomBudget(budget))
	if err != nil {
		return CandidateComparisonRun{}, err
	}
	exec, err = r.runtime.Run(ctx, id)
	if err != nil && exec == nil {
		return CandidateComparisonRun{}, err
	}
	return projectComparisonExecution(exec)
}

func projectComparisonExecution(execution *adgo.Execution) (CandidateComparisonRun, error) {
	if execution == nil {
		return CandidateComparisonRun{}, fmt.Errorf("controlplane: nil execution")
	}
	req, err := decodeExecutionData[CandidateComparisonRequest](execution, dataComparisonRequest)
	if err != nil {
		return CandidateComparisonRun{}, err
	}
	res, err := decodeExecutionData[ComparisonRunResult](execution, dataComparisonResult)
	if err != nil {
		return CandidateComparisonRun{}, err
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
	return CandidateComparisonRun{
		ID:      execution.ID,
		Status:  string(execution.Status),
		PlanID:  execution.PlanID,
		Request: req,
		Result:  res,
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
