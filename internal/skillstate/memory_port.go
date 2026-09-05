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

type StoreMemoryPort struct {
	store memory.Store
}

func NewStoreMemoryPort(store memory.Store) *StoreMemoryPort {
	return &StoreMemoryPort{store: store}
}

func stateProjectNamespace(state *SkillState) (memory.Namespace, error) {
	if state == nil {
		return memory.Namespace{}, ErrInvalidState
	}
	return memory.NewProjectKnowledgeNamespace(state.RunID)
}

// RetrieveContext queries memory for a bounded ContextPack. SkillState currently
// scopes epistemic memory by RunID; the same scope is used for admission so a
// caller cannot smuggle a global observation through a project-local state.
func (p *StoreMemoryPort) RetrieveContext(ctx context.Context, state *SkillState, focusAxes []string, tokenBudget int) (*memory.ContextPack, error) {
	if state == nil {
		return nil, ErrInvalidState
	}
	if p.store == nil {
		return nil, fmt.Errorf("memory port: store is nil")
	}
	ns, err := stateProjectNamespace(state)
	if err != nil {
		return nil, err
	}
	return p.store.RetrieveContextPack(ctx, memory.ContextPackRequest{
		Scope:              ns,
		FocusAxes:          focusAxes,
		BudgetTokens:       tokenBudget,
		MaxSimilarCases:    5,
		MaxCounterexamples: 3,
	})
}

// AdmitObservation commits only observations whose complete bundle remains in
// the active state's project. Globalization requires memory.Promote.
func (p *StoreMemoryPort) AdmitObservation(ctx context.Context, state *SkillState, bundle memory.AdmissionBundle) error {
	if state == nil {
		return ErrInvalidState
	}
	if p.store == nil {
		return fmt.Errorf("memory port: store is nil")
	}
	target, err := stateProjectNamespace(state)
	if err != nil {
		return err
	}
	if !bundle.SourceNamespace.IsValid() {
		source, sourceErr := memory.NewProjectEvidenceNamespace(state.RunID)
		if sourceErr != nil {
			return sourceErr
		}
		bundle.SourceNamespace = source
	}
	for i := range bundle.Atoms {
		if !bundle.Atoms[i].Namespace.IsValid() {
			bundle.Atoms[i].Namespace = target
		}
		if bundle.Atoms[i].Namespace.ProjectID() != state.RunID {
			return fmt.Errorf("memory port: atom %s target %s escapes active project %s", bundle.Atoms[i].ID, bundle.Atoms[i].Namespace, state.RunID)
		}
		bundle.Atoms[i].Provenance.SourceNamespace = bundle.SourceNamespace.String()
		bundle.Atoms[i].Provenance.ProjectScope = state.RunID
	}
	for i := range bundle.Edges {
		bundle.Edges[i].Provenance.SourceNamespace = bundle.SourceNamespace.String()
		bundle.Edges[i].Provenance.ProjectScope = state.RunID
	}
	return p.store.Commit(ctx, bundle)
}
