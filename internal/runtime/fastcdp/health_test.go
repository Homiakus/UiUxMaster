package fastcdp

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func handleProbeMock(t *testing.T, transport *fakeTransport, probe runtimeHealthProbe, isErr bool) {
	t.Helper()
	select {
	case payload := <-transport.writes:
		var req wireMessage
		if err := json.Unmarshal(payload, &req); err != nil {
			t.Fatalf("failed to decode wire: %v", err)
		}
		if isErr {
			resp := wireMessage{
				ID:    req.ID,
				Error: &ProtocolError{Code: -32000, Message: "Target closed"},
			}
			data, _ := json.Marshal(resp)
			transport.reads <- data
			return
		}

		evalResult := map[string]any{
			"result": map[string]any{
				"type":  "object",
				"value": probe,
			},
		}
		evalResultBytes, _ := json.Marshal(evalResult)
		resp := wireMessage{
			ID:     req.ID,
			Result: evalResultBytes,
		}
		data, _ := json.Marshal(resp)
		transport.reads <- data
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for request write")
	}
}

func TestCheckPageHealth_HealthyPage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	go func() {
		probe := runtimeHealthProbe{
			URL:        "about:blank",
			Origin:     "",
			DOMNodes:   15,
			HasSW:      false,
			HasEpochFn: true,
		}
		handleProbeMock(t, transport, probe, false)
	}()

	criteria := DefaultPageHealthCriteria()
	report, err := CheckPageHealth(ctx, conn, SessionID("test-sess"), nil, criteria)
	if err != nil {
		t.Fatalf("CheckPageHealth failed: %v", err)
	}

	if !report.Healthy {
		t.Fatalf("expected healthy page, got reasons: %v", report.StaleReasons)
	}
	if report.RecommendedReset != ResetNone {
		t.Fatalf("expected ResetNone, got %s", report.RecommendedReset)
	}
	if !report.EpochBridgeOK {
		t.Fatalf("expected epoch bridge ok")
	}
	if report.DOMNodeCount != 15 {
		t.Fatalf("expected 15 DOM nodes, got %d", report.DOMNodeCount)
	}
}

func TestCheckPageHealth_UnexpectedNavigationAndOrigin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	go func() {
		probe := runtimeHealthProbe{
			URL:        "https://evil.com/phish",
			Origin:     "https://evil.com",
			DOMNodes:   50,
			HasSW:      false,
			HasEpochFn: true,
		}
		handleProbeMock(t, transport, probe, false)
	}()

	criteria := PageHealthCriteria{
		ExpectedURL:        "https://app.local/dashboard",
		ExpectedOrigin:     "https://app.local",
		VerifyEpochBridge:  true,
		VerifySessionAlive: true,
	}

	report, err := CheckPageHealth(ctx, conn, SessionID("test-sess"), nil, criteria)
	if err != nil {
		t.Fatalf("CheckPageHealth failed: %v", err)
	}

	if report.Healthy {
		t.Fatalf("expected unhealthy report on unexpected origin/url")
	}
	if !containsReason(report.StaleReasons, StaleUnexpectedURL) {
		t.Fatalf("expected StaleUnexpectedURL, got %v", report.StaleReasons)
	}
	if !containsReason(report.StaleReasons, StaleUnexpectedOrigin) {
		t.Fatalf("expected StaleUnexpectedOrigin, got %v", report.StaleReasons)
	}
	if report.RecommendedReset != ResetPage {
		t.Fatalf("expected ResetPage, got %s", report.RecommendedReset)
	}
}

func TestCheckPageHealth_BrokenEpochBridge(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	go func() {
		probe := runtimeHealthProbe{
			URL:        "about:blank",
			DOMNodes:   10,
			HasSW:      false,
			HasEpochFn: false, // Bridge disappeared!
		}
		handleProbeMock(t, transport, probe, false)
	}()

	criteria := DefaultPageHealthCriteria()
	report, err := CheckPageHealth(ctx, conn, SessionID("test-sess"), nil, criteria)
	if err != nil {
		t.Fatalf("CheckPageHealth failed: %v", err)
	}

	if report.Healthy {
		t.Fatalf("expected unhealthy report on broken epoch bridge")
	}
	if !containsReason(report.StaleReasons, StaleBrokenEpochBridge) {
		t.Fatalf("expected StaleBrokenEpochBridge in %v", report.StaleReasons)
	}
	if report.RecommendedReset != ResetComponent {
		t.Fatalf("expected ResetComponent, got %s", report.RecommendedReset)
	}
}

func TestCheckPageHealth_StaleServiceWorkerAndDOMExplosion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	heap := uint64(150 * 1024 * 1024) // 150MB > 100MB max
	go func() {
		probe := runtimeHealthProbe{
			URL:           "about:blank",
			DOMNodes:      15000, // > 5000 max
			HeapUsedBytes: &heap,
			HasSW:         true, // Active SW
			HasEpochFn:    true,
		}
		handleProbeMock(t, transport, probe, false)
	}()

	criteria := DefaultPageHealthCriteria()
	criteria.MaxDOMNodes = 5000
	criteria.MaxJSHeapBytes = 100 * 1024 * 1024
	criteria.AllowServiceWorkers = false

	report, err := CheckPageHealth(ctx, conn, SessionID("test-sess"), nil, criteria)
	if err != nil {
		t.Fatalf("CheckPageHealth failed: %v", err)
	}

	if report.Healthy {
		t.Fatalf("expected unhealthy page")
	}
	if !containsReason(report.StaleReasons, StaleDOMExplosion) {
		t.Fatalf("missing StaleDOMExplosion: %v", report.StaleReasons)
	}
	if !containsReason(report.StaleReasons, StaleServiceWorkerActive) {
		t.Fatalf("missing StaleServiceWorkerActive: %v", report.StaleReasons)
	}
	if !containsReason(report.StaleReasons, StaleResourceGrowth) {
		t.Fatalf("missing StaleResourceGrowth: %v", report.StaleReasons)
	}
	// Service worker or memory growth must escalate to ResetContext
	if report.RecommendedReset != ResetContext {
		t.Fatalf("expected ResetContext for SW and memory growth, got %s", report.RecommendedReset)
	}
}

func TestCheckPageHealth_InvalidSessionTarget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	go func() {
		handleProbeMock(t, transport, runtimeHealthProbe{}, true)
	}()

	criteria := DefaultPageHealthCriteria()
	report, err := CheckPageHealth(ctx, conn, SessionID("dead-sess"), nil, criteria)
	if err != nil {
		t.Fatalf("CheckPageHealth failed: %v", err)
	}

	if report.Healthy {
		t.Fatalf("expected unhealthy report on dead session")
	}
	if !containsReason(report.StaleReasons, StaleInvalidSession) {
		t.Fatalf("missing StaleInvalidSession in %v", report.StaleReasons)
	}
	if report.RecommendedReset != ResetPage {
		t.Fatalf("expected ResetPage, got %s", report.RecommendedReset)
	}
}

func TestReleaseWithHealthCheck_DiscardOnStale(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := newFakeTransport()
	conn := NewConnection(transport)
	defer conn.Close()

	pool := &PagePool{
		conn:  conn,
		inUse: make(chan struct{}, 1),
		idle:  make(chan *WarmPage, 1),
		pages: make(map[TargetID]*WarmPage),
	}
	pool.inUse <- struct{}{}

	page := &WarmPage{
		Session: PageSession{
			TargetID:  TargetID("target-1"),
			SessionID: SessionID("sess-1"),
		},
	}
	pool.pages[page.Session.TargetID] = page

	lease := &PageLease{
		pool: pool,
		page: page,
	}

	go func() {
		// Mock probe indicating DOM explosion
		probe := runtimeHealthProbe{
			URL:        "about:blank",
			DOMNodes:   12000,
			HasSW:      false,
			HasEpochFn: true,
		}
		handleProbeMock(t, transport, probe, false)

		// Also handle CloseTarget when page is discarded
		select {
		case payload := <-transport.writes:
			var req wireMessage
			_ = json.Unmarshal(payload, &req)
			resp := wireMessage{
				ID:     req.ID,
				Result: json.RawMessage(`{}`),
			}
			respBytes, _ := json.Marshal(resp)
			transport.reads <- respBytes
		case <-time.After(2 * time.Second):
		}
	}()

	criteria := DefaultPageHealthCriteria()
	criteria.MaxDOMNodes = 1000
	report, err := lease.ReleaseWithHealthCheck(ctx, criteria)
	if err != nil {
		t.Fatalf("ReleaseWithHealthCheck returned error: %v", err)
	}
	if report.Healthy {
		t.Fatalf("expected unhealthy report")
	}

	// Verify page was NOT returned to idle queue (it was discarded!)
	select {
	case <-pool.idle:
		t.Fatalf("stale page was unexpectedly returned to idle pool!")
	default:
		// success: idle queue is empty
	}
}

func containsReason(reasons []StaleReason, target StaleReason) bool {
	for _, r := range reasons {
		if r == target {
			return true
		}
	}
	return false
}
