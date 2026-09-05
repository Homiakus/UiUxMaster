package dispatcher

import (
	"context"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
)

// mockL3Collector is shared by dispatcher regression suites (including FMEA-001).
// Keep it in a dedicated fixture file so focused rewrites of dispatcher_test.go
// cannot silently remove the stronger-tier test harness again.
type mockL3Collector struct {
	called   bool
	lastReq  engine.ValidationRequest
	lastPlan evidenceplan.Plan
	packet   evidence.Packet
	err      error
}

func (m *mockL3Collector) CollectL3(_ context.Context, req engine.ValidationRequest, plan evidenceplan.Plan) (evidence.Packet, error) {
	m.called = true
	m.lastReq = req
	m.lastPlan = plan
	p := m.packet
	p.RunID = req.RunID
	return p, m.err
}
