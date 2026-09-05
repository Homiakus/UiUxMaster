package engine

import (
	"context"

	"github.com/Homiakus/UiUxMaster/internal/impact"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
)

// ImpactStageFunc is the executable seam for impact resolution. Production
// pipelines normally leave it nil and use ResolveImpact with Pipeline.Resolver;
// alternate planners/tests can supply an implementation without conflating it
// with invalidation.
type ImpactStageFunc func(context.Context, ValidationRequest) (ValidationRequest, impact.ImpactSet, error)

// InvalidationStageFunc is the executable seam for converting a resolved
// ImpactSet into ValidationScope. It receives the output of the impact stage and
// cannot resolve graph impact itself.
type InvalidationStageFunc func(context.Context, ValidationRequest, impact.ImpactSet) (ValidationRequest, invalidation.ValidationScope, error)

func (p *Pipeline) resolveImpactStage(ctx context.Context, req ValidationRequest) (ValidationRequest, impact.ImpactSet, error) {
	if p.ImpactStage != nil {
		return p.ImpactStage(ctx, req)
	}
	return ResolveImpact(ctx, req, p.Resolver)
}

func (p *Pipeline) invalidateStage(ctx context.Context, req ValidationRequest, set impact.ImpactSet) (ValidationRequest, invalidation.ValidationScope, error) {
	if p.InvalidationStage != nil {
		return p.InvalidationStage(ctx, req, set)
	}
	return InvalidateImpact(ctx, req, set, p.Policy)
}
