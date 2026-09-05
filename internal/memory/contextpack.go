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
	BudgetTokens       int       `json:"budget_tokens"`
	MaxSimilarCases    int       `json:"max_similar_cases"`
	MaxCounterexamples int       `json:"max_counterexamples"`
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
// Scope is mandatory; omission is never interpreted as admin/all-project access.
func (s *EpMemoryStore) RetrieveContextPack(ctx context.Context, req ContextPackRequest) (*ContextPack, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !req.Scope.IsValid() {
		return nil, ErrScopeRequired
	}
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
		if stored.Status != StatusActive || stored.Atom.Kind != NodeDesignRule || !CanAccess(req.Scope, stored.Atom.Namespace) {
			continue
		}
		if rule, ok := stored.Atom.Data.(DesignRuleAtom); ok {
			if ruleMap[rule.RuleID] || axisMap[rule.Axis] || len(req.FocusAxes) == 0 {
				rules = append(rules, rule)
			}
		}
	}
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].HardConstraint != rules[j].HardConstraint {
			return rules[i].HardConstraint
		}
		return rules[i].Weight > rules[j].Weight
	})

	var patterns []RepairPatternAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive || stored.Atom.Kind != NodeRepairPattern || !CanAccess(req.Scope, stored.Atom.Namespace) {
			continue
		}
		if pat, ok := stored.Atom.Data.(RepairPatternAtom); ok {
			patterns = append(patterns, pat)
		}
	}
	sort.Slice(patterns, func(i, j int) bool { return patterns[i].SuccessRate > patterns[j].SuccessRate })

	var counterexamples []CounterexampleAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive || stored.Atom.Kind != NodeCounterexample || !CanAccess(req.Scope, stored.Atom.Namespace) {
			continue
		}
		if ce, ok := stored.Atom.Data.(CounterexampleAtom); ok {
			counterexamples = append(counterexamples, ce)
		}
	}

	var findings []DesignFindingAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive || stored.Atom.Kind != NodeDesignFinding || !CanAccess(req.Scope, stored.Atom.Namespace) {
			continue
		}
		if f, ok := stored.Atom.Data.(DesignFindingAtom); ok {
			if axisMap[f.Axis] || len(req.FocusAxes) == 0 {
				findings = append(findings, f)
			}
		}
	}

	usedTokens := 0
	for _, r := range rules {
		cost := estimateTokens(r)
		if usedTokens+cost > budgetTokens {
			break
		}
		pack.AdmittedRules = append(pack.AdmittedRules, r)
		usedTokens += cost
	}
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

	for _, c := range s.conflicts {
		if !s.conflictVisibleLocked(req.Scope, c) {
			continue
		}
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

func (s *EpMemoryStore) conflictVisibleLocked(scope Namespace, c ConflictRecord) bool {
	for _, id := range []string{c.PrimaryAtomID, c.ConflictingAtomID} {
		stored, ok := s.atoms[id]
		if !ok || stored.Status != StatusActive || !CanAccess(scope, stored.Atom.Namespace) {
			return false
		}
	}
	return true
}

func estimateTokens(v any) int {
	data, err := json.Marshal(v)
	if err != nil {
		return 10
	}
	tokens := len(data) / 4
	if tokens < 5 {
		tokens = 5
	}
	return tokens
}
