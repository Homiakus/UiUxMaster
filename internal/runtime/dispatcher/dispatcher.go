package dispatcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
	"github.com/Homiakus/UiUxMaster/internal/visualdiff"
)

// StaticCollector executes Tier L0 static evidence collection without a browser or renderer.
type StaticCollector interface {
	CollectL0(ctx context.Context, req engine.ValidationRequest, plan evidenceplan.Plan) (evidence.Packet, error)
}

// L2Collector executes Tier L2 browser collection (e.g. FastCDP).
type L2Collector interface {
	CollectL2(ctx context.Context, req engine.ValidationRequest, plan evidenceplan.Plan) (evidence.Packet, error)
}

// L3Collector executes Tier L3 clean-state browser verification (e.g. Playwright TruthPath).
type L3Collector interface {
	CollectL3(ctx context.Context, req engine.ValidationRequest, plan evidenceplan.Plan) (evidence.Packet, error)
}

// DefaultStaticCollector implements L0 by recording changed files and structural metadata.
type DefaultStaticCollector struct{}

func (s *DefaultStaticCollector) CollectL0(_ context.Context, req engine.ValidationRequest, _ evidenceplan.Plan) (evidence.Packet, error) {
	docs := make([]evidence.DocumentMetrics, 0, len(req.ChangedFiles))
	for _, f := range req.ChangedFiles {
		docs = append(docs, evidence.DocumentMetrics{
			URL: "file://" + f,
		})
	}
	return evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{
			Tier: string(engine.TierStatic),
			Name: "static-collector",
		},
		Documents: docs,
	}, nil
}

// Config configures the runtime dispatcher.
type Config struct {
	L0Collector                 StaticCollector
	L1Renderer                  fastrender.Renderer
	L2Collector                 L2Collector
	L3Collector                 L3Collector
	EscalateL1ToL2OnUnsupported bool
}

// Dispatcher executes validation across L0 (Static), L1 (FastRender/WGGo), L2 (FastBrowser/FastCDP), and L3 (TruthPath/Playwright)
// according to RouteDecision without exposing vendor implementations to callers.
type Dispatcher struct {
	l0         StaticCollector
	l1         fastrender.Renderer
	l2         L2Collector
	l3         L3Collector
	escalateL1 bool
}

var _ engine.Collector = (*Dispatcher)(nil)

// New creates an initialized Dispatcher.
func New(cfg Config) *Dispatcher {
	l0 := cfg.L0Collector
	if l0 == nil {
		l0 = &DefaultStaticCollector{}
	}
	return &Dispatcher{
		l0:         l0,
		l1:         cfg.L1Renderer,
		l2:         cfg.L2Collector,
		l3:         cfg.L3Collector,
		escalateL1: cfg.EscalateL1ToL2OnUnsupported,
	}
}

// Capabilities returns the capability profile of the configured L1 FastRender candidate.
func (d *Dispatcher) Capabilities() fastrender.Capabilities {
	if d.l1 != nil {
		return d.l1.Capabilities()
	}
	return fastrender.Capabilities{
		Name:             "none",
		BrowserAccurate:  false,
		SupportsPixels:   false,
		SupportsGeometry: false,
		SupportsStyles:   false,
	}
}

// Collect executes the requested validation plan on the appropriate runtime tier.
func (d *Dispatcher) Collect(ctx context.Context, req engine.ValidationRequest, plan engine.ValidationPlan) (evidence.Packet, error) {
	if err := ctx.Err(); err != nil {
		return evidence.Packet{}, err
	}

	switch plan.Route.Tier {
	case engine.TierStatic:
		return d.collectL0(ctx, req, plan)
	case engine.TierFastRender:
		return d.collectL1(ctx, req, plan)
	case engine.TierFastBrowser, engine.TierSemantic:
		return d.collectL2(ctx, req, plan)
	case engine.TierTruthPath:
		if d.l3 != nil {
			return d.collectL3(ctx, req, plan)
		}
		return d.collectL2(ctx, req, plan)
	default:
		return d.collectL2(ctx, req, plan)
	}
}

// Execute combines route planning and collection into one unified workflow call.
func (d *Dispatcher) Execute(ctx context.Context, req engine.ValidationRequest, assessment fidelity.Assessment) (engine.ValidationPlan, evidence.Packet, error) {
	plan := engine.PlanValidationRoute(req, assessment, d.Capabilities())
	packet, err := d.Collect(ctx, req, plan)
	return plan, packet, err
}

func (d *Dispatcher) collectL0(ctx context.Context, req engine.ValidationRequest, plan engine.ValidationPlan) (evidence.Packet, error) {
	if d.l0 == nil {
		return evidence.Packet{}, fmt.Errorf("dispatcher: L0 static collector is nil")
	}
	return d.l0.CollectL0(ctx, req, plan.EvidencePlan)
}

func (d *Dispatcher) collectL1(ctx context.Context, req engine.ValidationRequest, plan engine.ValidationPlan) (evidence.Packet, error) {
	if d.l1 == nil {
		if d.escalateL1 && d.l2 != nil {
			return d.collectL2(ctx, req, plan)
		}
		return evidence.Packet{}, fmt.Errorf("dispatcher: L1 fastrender renderer not configured")
	}

	caps := d.l1.Capabilities()
	needsGeometry := plan.EvidencePlan.Structural || req.Need.Geometry
	needsStyles := plan.EvidencePlan.Fonts || req.Need.Styles
	if (needsGeometry && !caps.SupportsGeometry) || (needsStyles && !caps.SupportsStyles) {
		if d.escalateL1 && d.l2 != nil {
			packet, l2Err := d.collectL2(ctx, req, plan)
			if l2Err == nil {
				packet.RuntimeIssues = append(packet.RuntimeIssues, evidence.RuntimeIssue{
					Code:     "L1_ESCALATION",
					Message:  "L1 renderer cannot prove geometry/styles; escalated to L2",
					Severity: evidence.SeverityInfo,
				})
			}
			return packet, l2Err
		}
		return evidence.Packet{}, fmt.Errorf("dispatcher: L1 cannot verify geometry or styles (unsupported; escalation required)")
	}

	start := time.Now()
	vp := evidence.Viewport{Width: 1280, Height: 800, DeviceScale: 1}
	theme := ""
	if len(req.Themes) > 0 {
		theme = req.Themes[0]
	}

	renderReq := fastrender.Request{
		HTML:    req.HTML,
		CSS:     req.CSS,
		BaseURL: req.BaseURL,
		Width:   vp.Width,
		Height:  vp.Height,
		DPR:     vp.DeviceScale,
		Theme:   theme,
	}

	var ev fastrender.Evidence
	var err error

	if plan.EvidencePlan.Pixels && plan.EvidencePlan.Region != nil {
		r := plan.EvidencePlan.Region
		scale := r.Scale
		if scale <= 0 {
			scale = 1
		}
		clip := image.Rect(int(r.X), int(r.Y), int(r.X+r.Width), int(r.Y+r.Height))
		ev, err = d.l1.CaptureRegion(ctx, fastrender.RegionRequest{
			Render: renderReq,
			Clip:   clip,
		})
	} else {
		ev, err = d.l1.Render(ctx, renderReq)
	}

	if err != nil {
		if errors.Is(err, fastrender.ErrUnsupported) && d.escalateL1 && d.l2 != nil {
			packet, l2Err := d.collectL2(ctx, req, plan)
			if l2Err == nil {
				packet.RuntimeIssues = append(packet.RuntimeIssues, evidence.RuntimeIssue{
					Code:     "L1_ESCALATION",
					Message:  fmt.Sprintf("L1 renderer unsupported: %v; escalated to L2", err),
					Severity: evidence.SeverityInfo,
				})
			}
			return packet, l2Err
		}
		return evidence.Packet{}, fmt.Errorf("dispatcher: L1 render failed: %w", err)
	}

	totalMS := float64(time.Since(start).Milliseconds())
	caps = d.l1.Capabilities()

	packet := evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{
			Tier:       string(engine.TierFastRender),
			Name:       caps.Name,
			Version:    caps.Version,
			FidelityID: ev.FidelityID,
		},
		Viewport: vp,
		Latency: evidence.RuntimeLatency{
			PixelsMS: float64(ev.Latency.Paint.Milliseconds()),
			TotalMS:  totalMS,
		},
	}

	if ev.RGBA != nil {
		hash := sha256.Sum256(ev.RGBA.Pix)
		bounds := ev.RGBA.Bounds()
		packet.Pixels = &evidence.PixelEvidence{
			Bounds: evidence.Rect{
				X:      float64(bounds.Min.X),
				Y:      float64(bounds.Min.Y),
				Width:  float64(bounds.Dx()),
				Height: float64(bounds.Dy()),
			},
			Width:        bounds.Dx(),
			Height:       bounds.Dy(),
			EncodedBytes: len(ev.RGBA.Pix),
			DigestSHA256: hex.EncodeToString(hash[:]),
		}
		if plan.EvidencePlan.Region != nil {
			packet.VisualRegions = append(packet.VisualRegions, evidence.VisualRegion{
				ID: "planned-region",
				Bounds: evidence.Rect{
					X:      plan.EvidencePlan.Region.X,
					Y:      plan.EvidencePlan.Region.Y,
					Width:  plan.EvidencePlan.Region.Width,
					Height: plan.EvidencePlan.Region.Height,
				},
			})
		}

		if req.BaselineRGBA != nil {
			diffRes, diffErr := visualdiff.CompareRGBA(req.BaselineRGBA, ev.RGBA, visualdiff.Options{
				ChannelTolerance: req.Tolerance,
			})
			if diffErr != nil {
				packet.RuntimeIssues = append(packet.RuntimeIssues, evidence.RuntimeIssue{
					Code:     "VISUALDIFF_ERROR",
					Message:  fmt.Sprintf("visualdiff error: %v", diffErr),
					Severity: evidence.SeverityLow,
				})
			} else if diffRes.ChangedPixels > 0 {
				diffBounds := evidence.Rect{
					X:      float64(diffRes.Bounds.Min.X),
					Y:      float64(diffRes.Bounds.Min.Y),
					Width:  float64(diffRes.Bounds.Dx()),
					Height: float64(diffRes.Bounds.Dy()),
				}
				packet.VisualRegions = append(packet.VisualRegions, evidence.VisualRegion{
					ID:            "visualdiff-changed-roi",
					Bounds:        diffBounds,
					ChangedPixels: int64(diffRes.ChangedPixels),
					DiffRatio:     diffRes.ChangeRatio,
				})
				packet.VisualFindings = append(packet.VisualFindings, evidence.VisualFinding{
					ID:          fmt.Sprintf("finding:visualdiff:%s", req.RunID),
					Axis:        "visual_regression",
					Title:       "Visual difference detected against baseline",
					Description: fmt.Sprintf("%d changed pixels (ratio: %.4f, max delta: %d)", diffRes.ChangedPixels, diffRes.ChangeRatio, diffRes.MaxDelta),
					Severity:    evidence.SeverityMedium,
					Confidence:  1.0,
					Source:      "pixel_diff",
					RegionID:    "visualdiff-changed-roi",
				})
			}
		}
	}

	if len(ev.Boxes) > 0 {
		for i, box := range ev.Boxes {
			packet.Elements = append(packet.Elements, evidence.ElementRef{
				ID:   fmt.Sprintf("box-%d", i),
				Role: box.Kind,
				Bounds: evidence.Rect{
					X:      float64(box.Bounds.Min.X),
					Y:      float64(box.Bounds.Min.Y),
					Width:  float64(box.Bounds.Dx()),
					Height: float64(box.Bounds.Dy()),
				},
			})
		}
	}

	for _, w := range ev.Warnings {
		packet.RuntimeIssues = append(packet.RuntimeIssues, evidence.RuntimeIssue{
			Code:     "RENDER_WARNING",
			Message:  w,
			Severity: evidence.SeverityLow,
		})
	}

	return packet, nil
}

func (d *Dispatcher) collectL2(ctx context.Context, req engine.ValidationRequest, plan engine.ValidationPlan) (evidence.Packet, error) {
	if d.l2 == nil {
		return evidence.Packet{}, fmt.Errorf("dispatcher: L2 fastbrowser collector not configured")
	}
	ep := plan.EvidencePlan
	if ep.Region == nil {
		if req.Region != nil {
			ep.Region = req.Region
			ep.Pixels = true
		} else if len(req.Scope.Regions) > 0 {
			for _, rStr := range req.Scope.Regions {
				if parsed, ok := parseRegionBounds(rStr); ok {
					ep.Region = parsed
					ep.Pixels = true
					break
				}
			}
		}
	}
	return d.l2.CollectL2(ctx, req, ep)
}

func (d *Dispatcher) collectL3(ctx context.Context, req engine.ValidationRequest, plan engine.ValidationPlan) (evidence.Packet, error) {
	if d.l3 == nil {
		return evidence.Packet{}, fmt.Errorf("dispatcher: L3 truthpath collector not configured")
	}
	ep := plan.EvidencePlan
	if ep.Region == nil {
		if req.Region != nil {
			ep.Region = req.Region
			ep.Pixels = true
		} else if len(req.Scope.Regions) > 0 {
			for _, rStr := range req.Scope.Regions {
				if parsed, ok := parseRegionBounds(rStr); ok {
					ep.Region = parsed
					ep.Pixels = true
					break
				}
			}
		}
	}
	return d.l3.CollectL3(ctx, req, ep)
}

