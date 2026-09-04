package invalidation

import (
	"context"
	"fmt"

	"github.com/Homiakus/UiUxMaster/internal/impact"
)

// ResolveProjectScope wires changed files and a ProjectIndex through the
// ImpactResolver and invalidation Policy to produce the authoritative ValidationScope.
func ResolveProjectScope(ctx context.Context, index *impact.ProjectIndex, changedFiles []string, policy *Policy, opts Options) (ValidationScope, error) {
	if err := ctx.Err(); err != nil {
		return ValidationScope{}, err
	}
	if index == nil || index.Graph == nil {
		return ValidationScope{}, fmt.Errorf("invalidation: project index or graph is nil")
	}
	if policy == nil {
		policy = DefaultPolicy()
	}

	changeSet := index.ChangeSetForFiles(changedFiles...)
	resolver, err := impact.NewResolver(index.Graph)
	if err != nil {
		return ValidationScope{}, fmt.Errorf("invalidation: create resolver: %w", err)
	}

	impactSet, err := resolver.ApplyChanges(ctx, changeSet)
	if err != nil {
		return ValidationScope{}, fmt.Errorf("invalidation: apply changes: %w", err)
	}

	return policy.Invalidate(impactSet, opts), nil
}
