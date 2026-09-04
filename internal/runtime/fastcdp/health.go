package fastcdp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// StaleReason categorizes reasons why a leased or warm page has become stale.
type StaleReason string

const (
	StaleUnexpectedOrigin    StaleReason = "unexpected_origin"
	StaleUnexpectedURL       StaleReason = "unexpected_url"
	StaleBrokenEpochBridge   StaleReason = "broken_epoch_bridge"
	StaleInvalidSession      StaleReason = "invalid_session"
	StaleInvalidContext      StaleReason = "invalid_context"
	StaleServiceWorkerActive StaleReason = "service_worker_active"
	StaleDOMExplosion        StaleReason = "dom_nodes_exceeded"
	StaleResourceGrowth      StaleReason = "bounded_resource_growth_exceeded"
)

// PageHealthCriteria configures the stale-state validation thresholds.
type PageHealthCriteria struct {
	ExpectedOrigin       string `json:"expected_origin,omitempty"`
	ExpectedURL          string `json:"expected_url,omitempty"`
	MaxDOMNodes          int    `json:"max_dom_nodes,omitempty"`
	MaxJSHeapBytes       uint64 `json:"max_js_heap_bytes,omitempty"`
	AllowServiceWorkers  bool   `json:"allow_service_workers"`
	VerifyEpochBridge    bool   `json:"verify_epoch_bridge"`
	VerifySessionAlive   bool   `json:"verify_session_alive"`
}

// DefaultPageHealthCriteria returns safe defaults for warm validation pages.
func DefaultPageHealthCriteria() PageHealthCriteria {
	return PageHealthCriteria{
		ExpectedURL:         "about:blank",
		MaxDOMNodes:         5000,
		MaxJSHeapBytes:      100 * 1024 * 1024, // 100MB
		AllowServiceWorkers: false,
		VerifyEpochBridge:   true,
		VerifySessionAlive:  true,
	}
}

// PageHealthReport summarizes the runtime health state of a page.
type PageHealthReport struct {
	Healthy          bool          `json:"healthy"`
	CurrentURL       string        `json:"current_url,omitempty"`
	CurrentOrigin    string        `json:"current_origin,omitempty"`
	DOMNodeCount     int           `json:"dom_node_count,omitempty"`
	JSHeapBytes      uint64        `json:"js_heap_bytes,omitempty"`
	ServiceWorker    bool          `json:"service_worker_active"`
	EpochBridgeOK    bool          `json:"epoch_bridge_ok"`
	StaleReasons     []StaleReason `json:"stale_reasons,omitempty"`
	RecommendedReset ResetLevel    `json:"recommended_reset"`
}

type runtimeHealthProbe struct {
	URL           string  `json:"url"`
	Origin        string  `json:"origin"`
	DOMNodes      int     `json:"domNodes"`
	HeapUsedBytes *uint64 `json:"heapUsedBytes"`
	HasSW         bool    `json:"hasSW"`
	HasEpochFn    bool    `json:"hasEpochFn"`
}

const probeExpression = `(() => {
	let heap = 0;
	if (typeof performance !== "undefined" && performance.memory && performance.memory.usedJSHeapSize) {
		heap = performance.memory.usedJSHeapSize;
	}
	let hasSW = false;
	if (typeof navigator !== "undefined" && navigator.serviceWorker && navigator.serviceWorker.controller) {
		hasSW = true;
	}
	return {
		url: window.location ? window.location.href : "",
		origin: window.location ? window.location.origin : "",
		domNodes: document.getElementsByTagName ? document.getElementsByTagName("*").length : 0,
		heapUsedBytes: heap,
		hasSW: hasSW,
		hasEpochFn: typeof window.__UIUX_SIGNAL_RENDER__ === "function"
	};
})()`

// CheckPageHealth probes the browser runtime on a given session to detect any
// stale or unrecoverable page state.
func CheckPageHealth(ctx context.Context, conn *Connection, sessionID SessionID, bridge *EpochBridge, criteria PageHealthCriteria) (PageHealthReport, error) {
	if conn == nil {
		return PageHealthReport{}, fmt.Errorf("fastcdp: connection is nil")
	}

	report := PageHealthReport{
		Healthy:          true,
		RecommendedReset: ResetNone,
	}

	var evalResp struct {
		Result struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}

	err := conn.Call(ctx, string(sessionID), "Runtime.evaluate", map[string]any{
		"expression":    probeExpression,
		"returnByValue": true,
	}, &evalResp)

	if err != nil {
		report.Healthy = false
		report.StaleReasons = append(report.StaleReasons, StaleInvalidSession)
		// If session or target is dead, decide between page or context reset
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "target") || strings.Contains(errLower, "session") {
			report.RecommendedReset = ResetPage
		} else {
			report.RecommendedReset = ResetContext
		}
		return report, nil
	}

	var probe runtimeHealthProbe
	if err := json.Unmarshal(evalResp.Result.Value, &probe); err != nil {
		report.Healthy = false
		report.StaleReasons = append(report.StaleReasons, StaleInvalidSession)
		report.RecommendedReset = ResetPage
		return report, nil
	}

	report.CurrentURL = probe.URL
	report.CurrentOrigin = probe.Origin
	report.DOMNodeCount = probe.DOMNodes
	if probe.HeapUsedBytes != nil {
		report.JSHeapBytes = *probe.HeapUsedBytes
	}
	report.ServiceWorker = probe.HasSW
	report.EpochBridgeOK = probe.HasEpochFn

	// Check 1: Unexpected URL
	if criteria.ExpectedURL != "" && probe.URL != criteria.ExpectedURL {
		report.Healthy = false
		report.StaleReasons = append(report.StaleReasons, StaleUnexpectedURL)
		report.escalateReset(ResetPage)
	}

	// Check 2: Unexpected Origin
	if criteria.ExpectedOrigin != "" && probe.Origin != criteria.ExpectedOrigin {
		report.Healthy = false
		report.StaleReasons = append(report.StaleReasons, StaleUnexpectedOrigin)
		report.escalateReset(ResetPage)
	}

	// Check 3: Broken Epoch Bridge
	if criteria.VerifyEpochBridge {
		if !probe.HasEpochFn || (bridge != nil && bridge.IsClosed()) {
			report.Healthy = false
			report.EpochBridgeOK = false
			report.StaleReasons = append(report.StaleReasons, StaleBrokenEpochBridge)
			report.escalateReset(ResetComponent)
		}
	}

	// Check 4: Stale Active Service Worker
	if !criteria.AllowServiceWorkers && probe.HasSW {
		report.Healthy = false
		report.StaleReasons = append(report.StaleReasons, StaleServiceWorkerActive)
		// Active service worker controller must be isolated/cleared at browser context level
		report.escalateReset(ResetContext)
	}

	// Check 5: Bounded Resource Growth - DOM Nodes
	if criteria.MaxDOMNodes > 0 && probe.DOMNodes > criteria.MaxDOMNodes {
		report.Healthy = false
		report.StaleReasons = append(report.StaleReasons, StaleDOMExplosion)
		report.escalateReset(ResetPage)
	}

	// Check 6: Bounded Resource Growth - JS Heap
	if criteria.MaxJSHeapBytes > 0 && probe.HeapUsedBytes != nil && *probe.HeapUsedBytes > criteria.MaxJSHeapBytes {
		report.Healthy = false
		report.StaleReasons = append(report.StaleReasons, StaleResourceGrowth)
		report.escalateReset(ResetContext)
	}

	return report, nil
}

func (r *PageHealthReport) escalateReset(level ResetLevel) {
	if level > r.RecommendedReset {
		r.RecommendedReset = level
	}
}

// CheckHealth inspects the warm page against criteria.
func (p *WarmPage) CheckHealth(ctx context.Context, conn *Connection, criteria PageHealthCriteria) (PageHealthReport, error) {
	if p == nil {
		return PageHealthReport{Healthy: false, StaleReasons: []StaleReason{StaleInvalidSession}, RecommendedReset: ResetPage}, nil
	}
	return CheckPageHealth(ctx, conn, p.Session.SessionID, p.Bridge, criteria)
}

// ReleaseWithHealthCheck verifies page health before releasing it back to the warm pool.
// If the page is stale, it is discarded immediately rather than returned to idle.
func (l *PageLease) ReleaseWithHealthCheck(ctx context.Context, criteria PageHealthCriteria) (PageHealthReport, error) {
	if l == nil || l.pool == nil || l.page == nil {
		return PageHealthReport{Healthy: false, StaleReasons: []StaleReason{StaleInvalidSession}, RecommendedReset: ResetPage}, nil
	}
	report, err := l.page.CheckHealth(ctx, l.pool.conn, criteria)
	if err != nil || !report.Healthy {
		// Stale or unhealthy page: discard from pool
		_ = l.Discard(ctx)
		return report, err
	}
	l.Release()
	return report, nil
}
