package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastcdp"
)

const defaultCollectorDiscardTimeout = 2 * time.Second

// CDPCollectorConfig configures the FastCDP-backed L2 collector.
type CDPCollectorConfig struct {
	Scenario        string
	FidelityID      string
	Viewport        evidence.Viewport
	WaitForNewEpoch bool
	MaxEpochRetries int
}

type cdpPageState struct {
	Epoch       uint64
	Diagnostics fastcdp.DiagnosticMark
}

// CDPCollector adapts resident FastCDP to the L2Collector interface.
type CDPCollector struct {
	runtime *fastcdp.ResidentRuntime
	config  CDPCollectorConfig
	browser fastcdp.BrowserVersion

	mu    sync.Mutex
	pages map[fastcdp.TargetID]cdpPageState
}

// NewCDPCollector creates an L2 collector wrapping a resident FastCDP runtime.
func NewCDPCollector(ctx context.Context, runtime *fastcdp.ResidentRuntime, config CDPCollectorConfig) (*CDPCollector, error) {
	if runtime == nil || runtime.Conn == nil || runtime.Pages == nil {
		return nil, fmt.Errorf("dispatcher: resident FastCDP runtime is required")
	}
	if config.Viewport.Width <= 0 || config.Viewport.Height <= 0 {
		config.Viewport = evidence.Viewport{Width: 1280, Height: 800, DeviceScale: 1}
	}
	browser, err := runtime.Version(ctx)
	if err != nil {
		return nil, fmt.Errorf("dispatcher: read browser version: %w", err)
	}
	if config.Scenario == "" {
		config.Scenario = "engine-validation"
	}
	if config.FidelityID == "" {
		config.FidelityID = "blink-l2"
	}
	return &CDPCollector{
		runtime: runtime,
		config:  config,
		browser: browser,
		pages:   make(map[fastcdp.TargetID]cdpPageState),
	}, nil
}

// CollectL2 satisfies L2Collector using warm FastCDP pages.
func (c *CDPCollector) CollectL2(ctx context.Context, req engine.ValidationRequest, plan evidenceplan.Plan) (evidence.Packet, error) {
	if c == nil || c.runtime == nil || c.runtime.Pages == nil || c.runtime.Conn == nil {
		return evidence.Packet{}, fmt.Errorf("dispatcher: CDP collector is not initialized")
	}
	lease, err := c.runtime.Pages.Acquire(ctx)
	if err != nil {
		return evidence.Packet{}, fmt.Errorf("dispatcher: acquire warm page: %w", err)
	}
	defer lease.Release()
	page := lease.Page()
	if page == nil {
		return evidence.Packet{}, fmt.Errorf("dispatcher: acquired warm page is nil")
	}

	// Wire ImpactSet-derived region if plan did not specify one
	if plan.Region == nil {
		if req.Region != nil {
			plan.Region = req.Region
			plan.Pixels = true
		} else if len(req.Scope.Regions) > 0 {
			for _, rStr := range req.Scope.Regions {
				if parsed, ok := parseRegionBounds(rStr); ok {
					plan.Region = parsed
					plan.Pixels = true
					break
				}
			}
		}
	}

	state := c.pageState(page.Session.TargetID)
	var diagnosticMark *fastcdp.DiagnosticMark
	if plan.Diagnostics {
		if page.Diagnostics == nil {
			return evidence.Packet{}, fmt.Errorf("dispatcher: plan requires diagnostics but warm-page diagnostics are disabled")
		}
		mark := state.Diagnostics
		diagnosticMark = &mark
	}

	cdpReq := fastcdp.RequestFromPlan(plan, fastcdp.PlannedRequestOptions{
		RequireAfter:     state.Epoch,
		WaitForNewEpoch:  c.config.WaitForNewEpoch,
		DiagnosticsSince: diagnosticMark,
		MaxEpochRetries:  c.config.MaxEpochRetries,
	})

	collected, err := page.CollectEvidence(ctx, c.runtime.Conn, cdpReq)
	if err != nil {
		wrapped := fmt.Errorf("dispatcher: collect FastCDP evidence: %w", err)
		c.dropPageState(page.Session.TargetID)
		discardCtx, cancel := context.WithTimeout(context.Background(), defaultCollectorDiscardTimeout)
		discardErr := lease.Discard(discardCtx)
		cancel()
		if discardErr != nil {
			return evidence.Packet{}, errors.Join(wrapped, fmt.Errorf("dispatcher: discard failed warm page: %w", discardErr))
		}
		return evidence.Packet{}, wrapped
	}

	packet := fastcdp.ToPacket(collected, fastcdp.PacketOptions{
		Scenario:   c.config.Scenario,
		Viewport:   c.config.Viewport,
		Browser:    c.browser,
		FidelityID: c.config.FidelityID,
		Region:     cdpReq.Region,
	})
	packet.RunID = req.RunID

	// Wire ImpactSet-derived routes/pages into packet
	if len(req.Scope.Routes) > 0 {
		if packet.URL == "" {
			packet.URL = req.Scope.Routes[0]
		}
		for _, route := range req.Scope.Routes {
			exists := false
			for _, doc := range packet.Documents {
				if doc.URL == route {
					exists = true
					break
				}
			}
			if !exists {
				packet.Documents = append(packet.Documents, evidence.DocumentMetrics{
					URL: route,
				})
			}
		}
	}

	// Wire ImpactSet-derived regions into packet.VisualRegions
	for _, rStr := range req.Scope.Regions {
		regionID := rStr
		exists := false
		for _, vr := range packet.VisualRegions {
			if vr.ID == regionID {
				exists = true
				break
			}
		}
		if !exists {
			var b evidence.Rect
			if parsed, ok := parseRegionBounds(rStr); ok {
				b = evidence.Rect{
					X:      parsed.X,
					Y:      parsed.Y,
					Width:  parsed.Width,
					Height: parsed.Height,
				}
			}
			packet.VisualRegions = append(packet.VisualRegions, evidence.VisualRegion{
				ID:     regionID,
				Bounds: b,
			})
		}
	}

	if cdpReq.Region != nil && packet.Pixels != nil {
		hasRequestedROI := false
		for _, vr := range packet.VisualRegions {
			if vr.ID == "requested-roi" {
				hasRequestedROI = true
				break
			}
		}
		if !hasRequestedROI {
			packet.VisualRegions = append(packet.VisualRegions, evidence.VisualRegion{
				ID:     "requested-roi",
				Bounds: packet.Pixels.Bounds,
			})
		}
	}

	state.Epoch = collected.Epoch
	if collected.Diagnostics != nil {
		state.Diagnostics = collected.Diagnostics.Through
	}
	c.savePageState(page.Session.TargetID, state)

	return packet, nil
}

func (c *CDPCollector) pageState(targetID fastcdp.TargetID) cdpPageState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pages[targetID]
}

func (c *CDPCollector) savePageState(targetID fastcdp.TargetID, state cdpPageState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pages[targetID] = state
}

func (c *CDPCollector) dropPageState(targetID fastcdp.TargetID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pages, targetID)
}

func parseRegionBounds(raw string) (*evidenceplan.Region, bool) {
	s := strings.TrimPrefix(raw, "region:")
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return nil, false
	}
	var x, y, w, h float64
	var err error
	if x, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err != nil {
		return nil, false
	}
	if y, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err != nil {
		return nil, false
	}
	if w, err = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err != nil || w <= 0 {
		return nil, false
	}
	if h, err = strconv.ParseFloat(strings.TrimSpace(parts[3]), 64); err != nil || h <= 0 {
		return nil, false
	}
	return &evidenceplan.Region{
		X:      x,
		Y:      y,
		Width:  w,
		Height: h,
		Scale:  1,
	}, true
}

