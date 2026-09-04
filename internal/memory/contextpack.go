package memory

import (
	"context"
	"encoding/json"
	"sort"
)

// ContextPackRequest configures bounded memory retrieval for a specific reasoning/critic task.
type ContextPackRequest struct {
	Scope              Namespace `json:"scope"`
	FocusAxes          []string  `json:"focus_axes,omitempty"`
	FindingCategories  []string  `json:"finding_categories,omitempty"`
	RuleIDs            []string  `json:"rule_ids,omitempty"`
	BudgetTokens       int       `json:"budget_tokens"`       // e.g. 2000
	MaxSimilarCases    int       `json:"max_similar_cases"`   // default <= 5
	MaxCounterexamples int       `json:"max_counterexamples"` // default <= 3
}

// ContextPack contains the minimal sufficient validated memory projection for model reasoning.
type ContextPack struct {
	Scope            string               `json:"scope"`
	AdmittedRules    []DesignRuleAtom     `json:"admitted_rules,omitempty"`
	RelevantFindings []DesignFindingAtom  `json:"relevant_findings,omitempty"`
	ProvenPatterns   []RepairPatternAtom  `json:"proven_patterns,omitempty"`
	Counterexamples  []CounterexampleAtom `json:"counterexamples,omitempty"`
	ActiveConflicts  []ConflictRecord     `json:"active_conflicts,omitempty"`
	EstimatedTokens  int                  `json:"estimated_tokens"`
}

// RetrieveContextPack builds a bounded ContextPack obeying token limits and item bounds.
func (s *EpMemoryStore) RetrieveContextPack(ctx context.Context, req ContextPackRequest) (*ContextPack, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	maxSimilar := req.MaxSimilarCases
	if maxSimilar <= 0 {
		maxSimilar = 5
	}
	maxCounter := req.MaxCounterexamples
	if maxCounter <= 0 {
		maxCounter = 3
	}
	budgetTokens := req.BudgetTokens
	if budgetTokens <= 0 {
		budgetTokens = 2000
	}

	pack := &ContextPack{
		Scope:            req.Scope.String(),
		AdmittedRules:    make([]DesignRuleAtom, 0),
		RelevantFindings: make([]DesignFindingAtom, 0),
		ProvenPatterns:   make([]RepairPatternAtom, 0),
		Counterexamples:  make([]CounterexampleAtom, 0),
		ActiveConflicts:  make([]ConflictRecord, 0),
	}

	// 1. Gather Candidate Rules (prioritize hard constraints and focus axes)
	ruleMap := make(map[string]bool)
	for _, rID := range req.RuleIDs {
		ruleMap[rID] = true
	}
	axisMap := make(map[string]bool)
	for _, ax := range req.FocusAxes {
		axisMap[ax] = true
	}

	var rules []DesignRuleAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive || stored.Atom.Kind != NodeDesignRule {
			continue
		}
		if req.Scope.raw != "" && !CanAccess(req.Scope, stored.Atom.Namespace) {
			continue
		}

		if rule, ok := stored.Atom.Data.(DesignRuleAtom); ok {
			if ruleMap[rule.RuleID] || axisMap[rule.Axis] || len(req.FocusAxes) == 0 {
				rules = append(rules, rule)
			}
		}
	}

	// Sort rules: hard constraints first, then weight
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].HardConstraint != rules[j].HardConstraint {
			return rules[i].HardConstraint
		}
		return rules[i].Weight > rules[j].Weight
	})

	// 2. Gather Repair Patterns
	var patterns []RepairPatternAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive || stored.Atom.Kind != NodeRepairPattern {
			continue
		}
		if req.Scope.raw != "" && !CanAccess(req.Scope, stored.Atom.Namespace) {
			continue
		}

		if pat, ok := stored.Atom.Data.(RepairPatternAtom); ok {
			patterns = append(patterns, pat)
		}
	}
	// Sort patterns by success rate descending
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].SuccessRate > patterns[j].SuccessRate
	})

	// 3. Gather Counterexamples
	var counterexamples []CounterexampleAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive || stored.Atom.Kind != NodeCounterexample {
			continue
		}
		if req.Scope.raw != "" && !CanAccess(req.Scope, stored.Atom.Namespace) {
			continue
		}

		if ce, ok := stored.Atom.Data.(CounterexampleAtom); ok {
			counterexamples = append(counterexamples, ce)
		}
	}

	// 4. Gather Relevant Findings
	var findings []DesignFindingAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive || stored.Atom.Kind != NodeDesignFinding {
			continue
		}
		if req.Scope.raw != "" && !CanAccess(req.Scope, stored.Atom.Namespace) {
			continue
		}

		if f, ok := stored.Atom.Data.(DesignFindingAtom); ok {
			if axisMap[f.Axis] || len(req.FocusAxes) == 0 {
				findings = append(findings, f)
			}
		}
	}

	// Greedy budget allocation
	usedTokens := 0

	// Add rules within budget
	for _, r := range rules {
		cost := estimateTokens(r)
		if usedTokens+cost > budgetTokens {
			break
		}
		pack.AdmittedRules = append(pack.AdmittedRules, r)
		usedTokens += cost
	}

	// Add patterns up to maxSimilar and budget
	for _, p := range patterns {
		if len(pack.ProvenPatterns) >= maxSimilar {
			break
		}
		cost := estimateTokens(p)
		if usedTokens+cost > budgetTokens {
			break
		}
		pack.ProvenPatterns = append(pack.ProvenPatterns, p)
		usedTokens += cost
	}

	// Add counterexamples up to maxCounter and budget
	for _, ce := range counterexamples {
		if len(pack.Counterexamples) >= maxCounter {
			break
		}
		cost := estimateTokens(ce)
		if usedTokens+cost > budgetTokens {
			break
		}
		pack.Counterexamples = append(pack.Counterexamples, ce)
		usedTokens += cost
	}

	// Add findings up to maxSimilar and budget
	for _, f := range findings {
		if len(pack.RelevantFindings) >= maxSimilar {
			break
		}
		cost := estimateTokens(f)
		if usedTokens+cost > budgetTokens {
			break
		}
		pack.RelevantFindings = append(pack.RelevantFindings, f)
		usedTokens += cost
	}

	// Add active conflicts relevant to selected items
	for _, c := range s.conflicts {
		cost := estimateTokens(c)
		if usedTokens+cost > budgetTokens {
			break
		}
		pack.ActiveConflicts = append(pack.ActiveConflicts, c)
		usedTokens += cost
	}

	pack.EstimatedTokens = usedTokens
	return pack, nil
}

func estimateTokens(v any) int {
	data, err := json.Marshal(v)
	if err != nil {
		return 10
	}
	// Approximate 4 chars per token in json payload
	tokens := len(data) / 4
	if tokens < 5 {
		tokens = 5
	}
	return tokens
}
