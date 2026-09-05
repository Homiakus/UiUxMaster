package controlplane

import (
	"context"

	"github.com/Homiakus/axiom/adgo"
)

type ActivityIdentity struct {
	ExecutionID    string `json:"execution_id"`
	NodeID         string `json:"node_id"`
	Attempt        int    `json:"attempt"`
	IdempotencyKey string `json:"idempotency_key"`
}

type activityIdentityKey struct{}

func withActivityIdentity(ctx context.Context, req adgo.ActivityRequest) context.Context {
	return context.WithValue(ctx, activityIdentityKey{}, ActivityIdentity{
		ExecutionID: req.ExecutionID,
		NodeID: req.NodeID,
		Attempt: req.Attempt,
		IdempotencyKey: req.IdempotencyKey,
	})
}

// ActivityIdentityFromContext lets an execution adapter bind provider/source/
// memory effects to Axiom's stable retry identity without importing adgo outside
// the control-plane boundary.
func ActivityIdentityFromContext(ctx context.Context) (ActivityIdentity, bool) {
	value, ok := ctx.Value(activityIdentityKey{}).(ActivityIdentity)
	return value, ok
}
