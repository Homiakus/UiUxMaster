package skillstate

import (
	"context"
	"fmt"

	"github.com/Homiakus/UiUxMaster/internal/memory"
)

// MemoryPort provides bounded, firewall-compliant access to epistemic memory.
type MemoryPort interface {
	RetrieveContext(ctx context.Context, state *SkillState, focusAxes []string, tokenBudget int) (*memory.ContextPack, error)
	AdmitObservation(ctx context.Context, state *SkillState, bundle memory.AdmissionBundle) error
}

// StoreMemoryPort connects SkillState execution with EpMemoryStore.
type StoreMemoryPort struct {
	store memory.Store
}

// NewStoreMemoryPort creates a MemoryPort backed by an EpMemoryStore.
func NewStoreMemoryPort(store memory.Store) *StoreMemoryPort {
	return &StoreMemoryPort{store: store}
}

// RetrieveContext queries SncSinCore memory for a bounded ContextPack obeying firewall boundaries.
func (p *StoreMemoryPort) RetrieveContext(ctx context.Context, state *SkillState, focusAxes []string, tokenBudget int) (*memory.ContextPack, error) {
	if state == nil {
		return nil, ErrInvalidState
	}
	if p.store == nil {
		return nil, fmt.Errorf("memory port: store is nil")
	}

	// Create project-level scope for the active run
	ns, err := memory.NewProjectKnowledgeNamespace(state.RunID)
	if err != nil {
		ns = memory.NewGlobalDesignNamespace()
	}

	req := memory.ContextPackRequest{
		Scope:              ns,
		FocusAxes:          focusAxes,
		BudgetTokens:       tokenBudget,
		MaxSimilarCases:    5,
		MaxCounterexamples: 3,
	}

	return p.store.RetrieveContextPack(ctx, req)
}

// AdmitObservation commits validated observation atoms and edges into epistemic memory.
func (p *StoreMemoryPort) AdmitObservation(ctx context.Context, state *SkillState, bundle memory.AdmissionBundle) error {
	if state == nil {
		return ErrInvalidState
	}
	if p.store == nil {
		return fmt.Errorf("memory port: store is nil")
	}

	return p.store.Commit(ctx, bundle)
}
