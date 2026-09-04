package engine

import (
	"context"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// Collector is the protocol-independent execution boundary. It takes a
// ValidationPlan and produces the canonical evidence.Packet without leaking
// vendor details (WGGo, FastCDP, Playwright) to callers.
type Collector interface {
	Collect(ctx context.Context, req ValidationRequest, plan ValidationPlan) (evidence.Packet, error)
}
