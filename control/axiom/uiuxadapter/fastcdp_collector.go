package uiuxadapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastcdp"
)

type FastCDPCollectorConfig struct {
	Viewport        evidence.Viewport
	Scenario        string
	FidelityID      string
	WaitForNewEpoch bool
	MaxEpochRetries int
}

type fastCDPPageState struct {
	Epoch          uint64
	Diagnostics    fastcdp.DiagnosticMark
	Initialized    bool
	HasDiagnostics bool
}

// FastCDPCollector is the real L2 execution-plane implementation used by the
// Axiom control slice. Browser/process/page ownership stays in ResidentRuntime;
// the collector only leases a warm page, asks for the evidence shape selected
// by evidenceplan, projects one canonical packet, and advances ephemeral
// per-page watermarks after an accepted stable capture.
type FastCDPCollector struct {
	runtime *fastcdp.ResidentRuntime
	config  FastCDPCollectorConfig
	browser fastcdp.BrowserVersion

	mu    sync.Mutex
	pages map[fastcdp.TargetID]fastCDPPageState
}

func NewFastCDPCollector(ctx context.Context, runtime *fastcdp.ResidentRuntime, config FastCDPCollectorConfig) (*FastCDPCollector, error) {
	if runtime == nil || runtime.Conn == nil || runtime.Pages == nil {
		return nil, fmt.Errorf("uiuxadapter: resident FastCDP runtime is required")
	}
	if config.Viewport.Width <= 0 || config.Viewport.Height <= 0 {
		return nil, fmt.Errorf("uiuxadapter: collector viewport must be positive")
	}
	browser, err := runtime.Version(ctx)
	if err != nil {
		return nil, fmt.Errorf("uiuxadapter: read browser version: %w", err)
	}
	if config.Scenario == "" {
		config.Scenario = "axiom-design-validation"
	}
	if config.FidelityID == "" {
		config.FidelityID = "blink-l2"
	}
	return &FastCDPCollector{
		runtime: runtime,
		config:  config,
		browser: browser,
		pages:   make(map[fastcdp.TargetID]fastCDPPageState),
	}, nil
}

func (c *FastCDPCollector) Collect(ctx context.Context, change controlplane.Change, plan evidenceplan.Plan) (evidence.Packet, error) {
	if c == nil || c.runtime == nil || c.runtime.Pages == nil || c.runtime.Conn == nil {
		return evidence.Packet{}, fmt.Errorf("uiuxadapter: FastCDP collector is not initialized")
	}
	lease, err := c.runtime.Pages.Acquire(ctx)
	if err != nil {
		return evidence.Packet{}, fmt.Errorf("uiuxadapter: acquire warm page: %w", err)
	}
	defer lease.Release()
	page := lease.Page()
	if page == nil {
		return evidence.Packet{}, fmt.Errorf("uiuxadapter: acquired warm page is nil")
	}

	state := c.pageState(page.Session.TargetID)
	var diagnosticMark *fastcdp.DiagnosticMark
	if plan.Diagnostics {
		if page.Diagnostics == nil {
			return evidence.Packet{}, fmt.Errorf("uiuxadapter: plan requires diagnostics but warm-page diagnostics are disabled")
		}
		// The zero mark deliberately means "from the bounded beginning of this
		// warm page" on the first accepted cycle. Later cycles advance exactly to
		// DiagnosticSnapshot.Through, so no event range is silently skipped.
		mark := state.Diagnostics
		diagnosticMark = &mark
		state.HasDiagnostics = true
	}

	req := fastcdp.RequestFromPlan(plan, fastcdp.PlannedRequestOptions{
		RequireAfter:       state.Epoch,
		WaitForNewEpoch:    c.config.WaitForNewEpoch,
		DiagnosticsSince:   diagnosticMark,
		MaxEpochRetries:    c.config.MaxEpochRetries,
	})
	if plan.Pixels {
		region, err := captureRegion(change.Region)
		if err != nil {
			return evidence.Packet{}, err
		}
		req.Region = region
	}

	collected, err := page.CollectEvidence(ctx, c.runtime.Conn, req)
	if err != nil {
		return evidence.Packet{}, fmt.Errorf("uiuxadapter: collect FastCDP evidence: %w", err)
	}
	packet := fastcdp.ToPacket(collected, fastcdp.PacketOptions{
		Scenario:   c.config.Scenario,
		Viewport:   c.config.Viewport,
		Browser:    c.browser,
		FidelityID: c.config.FidelityID,
		Region:     req.Region,
	})
	if req.Region != nil && packet.Pixels != nil {
		packet.VisualRegions = append(packet.VisualRegions, evidence.VisualRegion{
			ID:     "requested-roi",
			Bounds: packet.Pixels.Bounds,
		})
	}

	state.Epoch = collected.Epoch
	state.Initialized = true
	if collected.Diagnostics != nil {
		state.Diagnostics = collected.Diagnostics.Through
		state.HasDiagnostics = true
	}
	c.setPageState(page.Session.TargetID, state)
	return packet, nil
}

func (c *FastCDPCollector) pageState(target fastcdp.TargetID) fastCDPPageState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pages[target]
}

func (c *FastCDPCollector) setPageState(target fastcdp.TargetID, state fastCDPPageState) {
	c.mu.Lock()
	c.pages[target] = state
	c.mu.Unlock()
}

func captureRegion(region *controlplane.Region) (*fastcdp.CaptureRegionOptions, error) {
	if region == nil {
		return nil, fmt.Errorf("uiuxadapter: pixel evidence requires an explicit region")
	}
	if region.Width <= 0 || region.Height <= 0 {
		return nil, fmt.Errorf("uiuxadapter: pixel region must have positive width and height")
	}
	scale := region.Scale
	if scale == 0 {
		scale = 1
	}
	if scale < 0 {
		return nil, fmt.Errorf("uiuxadapter: pixel region scale must be positive")
	}
	return &fastcdp.CaptureRegionOptions{
		X: region.X, Y: region.Y,
		Width: region.Width, Height: region.Height,
		Scale: scale, OptimizeForSpeed: true,
	}, nil
}
