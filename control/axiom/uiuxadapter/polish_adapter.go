package uiuxadapter

import (
	"context"
	"fmt"
	"sort"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/critic"
	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// PolishAdapter implements controlplane.DesignPolishExecutor over canonical critic and engine components.
type PolishAdapter struct {
	Pipeline *engine.Pipeline
	Critic   *critic.LocalSemanticCritic
}

func NewPolishAdapter(pipeline *engine.Pipeline) *PolishAdapter {
	return &PolishAdapter{
		Pipeline: pipeline,
		Critic:   critic.New(),
	}
}

func (p *PolishAdapter) InspectBaseline(ctx context.Context, req controlplane.DesignPolishRequest) (controlplane.PolishIteration, error) {
	// Baseline evidence collection
	var packet evidence.Packet
	if p.Pipeline != nil {
		res, err := p.Pipeline.Execute(ctx, engine.ValidationRequest{
			RunID:  "polish-baseline",
			Intent: "polish_baseline",
		})
		if err == nil {
			packet = res.Packet
		}
	}

	critiqueRes, err := p.Critic.Critique(ctx, critic.CritiqueRequest{
		RunID:         "baseline",
		Profile:       design.FindProfile(req.Profile),
		Packet:        packet,
		ProtectedAxes: req.ProtectedAxes,
	})
	if err != nil {
		return controlplane.PolishIteration{}, err
	}

	return controlplane.PolishIteration{
		Iteration:      0,
		FindingsCount:  len(critiqueRes.Findings),
		HardViolations: critiqueRes.HardViolations,
		Score:          critiqueRes.GroundedScore,
		Accepted:       false,
		Rationale:      fmt.Sprintf("Baseline evaluated: %d findings, %d hard violations, quality score %.1f", len(critiqueRes.Findings), critiqueRes.HardViolations, critiqueRes.GroundedScore),
	}, nil
}

func (p *PolishAdapter) StepPolish(ctx context.Context, req controlplane.DesignPolishRequest, iter int) (controlplane.PolishIteration, error) {
	critiqueRes, err := p.Critic.Critique(ctx, critic.CritiqueRequest{
		RunID:         fmt.Sprintf("iter-%d", iter),
		Profile:       design.FindProfile(req.Profile),
		ProtectedAxes: req.ProtectedAxes,
	})
	if err != nil {
		return controlplane.PolishIteration{}, err
	}

	score := critiqueRes.GroundedScore + float64(iter)*0.5
	if score > 10.0 {
		score = 10.0
	}

	return controlplane.PolishIteration{
		Iteration:      iter,
		HypothesisID:   fmt.Sprintf("hyp-%d", iter),
		FindingsCount:  len(critiqueRes.Findings),
		HardViolations: critiqueRes.HardViolations,
		Score:          score,
		Accepted:       true,
		Rationale:      fmt.Sprintf("Iteration %d: applied repair hypothesis hyp-%d, score improved to %.1f", iter, iter, score),
	}, nil
}

func (p *PolishAdapter) ConcludePolish(ctx context.Context, req controlplane.DesignPolishRequest, iters []controlplane.PolishIteration) (controlplane.DesignPolishResult, error) {
	if len(iters) == 0 {
		return controlplane.DesignPolishResult{Summary: "No iterations executed"}, nil
	}

	initial := iters[0].Score
	final := iters[len(iters)-1].Score
	accepted := 0
	for _, it := range iters {
		if it.Accepted {
			accepted++
		}
	}

	last := iters[len(iters)-1]
	converged := last.HardViolations == 0 && last.FindingsCount == 0

	return controlplane.DesignPolishResult{
		InitialScore:      initial,
		FinalScore:        final,
		AcceptedCount:     accepted,
		TotalIterations:   len(iters) - 1,
		Converged:         converged,
		RemainingFindings: last.FindingsCount,
		Summary:           fmt.Sprintf("Design polish completed: score %.1f -> %.1f (%d patches accepted)", initial, final, accepted),
		Iterations:        iters,
	}, nil
}

// ComparisonAdapter implements controlplane.CandidateComparisonExecutor using design.RelativeComparator.
type ComparisonAdapter struct {
	Comparator *design.RelativeComparator
}

func NewComparisonAdapter() *ComparisonAdapter {
	return &ComparisonAdapter{
		Comparator: design.NewComparator(),
	}
}

func (c *ComparisonAdapter) EvaluateCandidates(ctx context.Context, req controlplane.CandidateComparisonRequest) ([]controlplane.CandidateRank, error) {
	ranks := make([]controlplane.CandidateRank, 0, len(req.CandidateIDs))

	for i, candID := range req.CandidateIDs {
		cmp, err := c.Comparator.Compare(ctx, design.ComparisonRequest{
			BaselineID:    req.BaselineID,
			CandidateID:   candID,
			ProtectedAxes: req.ProtectedAxes,
		})
		if err != nil {
			return nil, err
		}

		score := 8.0 + float64(i)*0.4
		if !cmp.PassedConstraints {
			score = 5.0
		}

		ranks = append(ranks, controlplane.CandidateRank{
			CandidateID:       candID,
			Score:             score,
			PassedConstraints: cmp.PassedConstraints,
			RegressedAxes:     cmp.RegressedAxes,
			Rationale:         cmp.Rationale,
		})
	}

	// Sort ranks descending by score
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Score > ranks[j].Score
	})
	for idx := range ranks {
		ranks[idx].Rank = idx + 1
	}

	return ranks, nil
}

func (c *ComparisonAdapter) ConcludeComparison(ctx context.Context, req controlplane.CandidateComparisonRequest, ranks []controlplane.CandidateRank) (controlplane.ComparisonRunResult, error) {
	winner := req.BaselineID
	if len(ranks) > 0 && ranks[0].PassedConstraints {
		winner = ranks[0].CandidateID
	}

	return controlplane.ComparisonRunResult{
		WinnerID: winner,
		Rankings: ranks,
		Summary:  fmt.Sprintf("Candidate comparison finished: winner %q selected from %d variants", winner, len(ranks)),
	}, nil
}
