package critic

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

// StandardAuditViewports defines the standard responsive audit matrix.
var StandardAuditViewports = []evidence.Viewport{
	{Width: 375, Height: 667, DeviceScale: 2},   // Mobile (iPhone SE / standard phone)
	{Width: 768, Height: 1024, DeviceScale: 2},  // Tablet (iPad Mini / Portrait tablet)
	{Width: 1440, Height: 900, DeviceScale: 1},  // Desktop (Standard laptop/desktop display)
}

// MultiViewportRequest configures a multi-viewport UI/UX audit.
type MultiViewportRequest struct {
	RunID         string                  `json:"run_id"`
	Viewports     []evidence.Viewport     `json:"viewports,omitempty"`
	Profile       design.ProductProfile   `json:"profile"`
	ProtectedAxes []string                `json:"protected_axes,omitempty"`
}

// ViewportResult captures evidence, deterministic verification, and semantic critique for one viewport.
type ViewportResult struct {
	Viewport     evidence.Viewport       `json:"viewport"`
	Packet       evidence.Packet         `json:"packet"`
	Verification verifier.Result         `json:"verification"`
	Critique     design.CritiquePass     `json:"critique"`
	Duration     time.Duration           `json:"duration"`
}

// MultiViewportReport aggregates audit results across all evaluated viewports.
type MultiViewportReport struct {
	RunID          string                    `json:"run_id"`
	Viewports      []ViewportResult          `json:"viewports"`
	TotalFindings  int                       `json:"total_findings"`
	HardViolations int                       `json:"hard_violations"`
	GroundedScore  float64                   `json:"grounded_score"`
	LocalizedByEl  map[string][]design.Finding `json:"localized_by_element"`
	TotalDuration  time.Duration             `json:"total_duration"`
}

// ViewportCollector collects evidence for a specific viewport.
type ViewportCollector interface {
	CollectForViewport(ctx context.Context, vp evidence.Viewport) (evidence.Packet, error)
}

// MultiViewportAuditor executes systematic multi-viewport audits and ground defect localization.
type MultiViewportAuditor struct {
	critic    Critic
	verPolicy verifier.Policy
}

// NewMultiViewportAuditor creates an auditor with default policies.
func NewMultiViewportAuditor(critic Critic, verPolicy verifier.Policy) *MultiViewportAuditor {
	if critic == nil {
		critic = New()
	}
	if verPolicy.MinTargetWidth == 0 {
		verPolicy = verifier.DefaultPolicy()
	}
	return &MultiViewportAuditor{
		critic:    critic,
		verPolicy: verPolicy,
	}
}

// Audit runs progressive semantic critique and deterministic verification across all specified viewports.
func (a *MultiViewportAuditor) Audit(ctx context.Context, collector ViewportCollector, req MultiViewportRequest) (MultiViewportReport, error) {
	if err := ctx.Err(); err != nil {
		return MultiViewportReport{}, err
	}
	if collector == nil {
		return MultiViewportReport{}, fmt.Errorf("multiviewport: collector is required")
	}

	started := time.Now()
	viewports := req.Viewports
	if len(viewports) == 0 {
		viewports = StandardAuditViewports
	}

	report := MultiViewportReport{
		RunID:         req.RunID,
		Viewports:     make([]ViewportResult, 0, len(viewports)),
		LocalizedByEl: make(map[string][]design.Finding),
	}

	minScore := 10.0
	totalHard := 0
	totalFindings := 0

	for _, vp := range viewports {
		vpStart := time.Now()
		packet, err := collector.CollectForViewport(ctx, vp)
		if err != nil {
			return MultiViewportReport{}, fmt.Errorf("multiviewport: collect for %dx%d: %w", vp.Width, vp.Height, err)
		}
		packet.Viewport = vp

		// 1. Run deterministic verification
		vResult := verifier.Apply(&packet, a.verPolicy)

		// 2. Run progressive semantic critique
		cPass, err := a.critic.Critique(ctx, CritiqueRequest{
			RunID:         fmt.Sprintf("%s-%dx%d", req.RunID, vp.Width, vp.Height),
			Level:         design.LevelPage,
			Profile:       req.Profile,
			Packet:        packet,
			ProtectedAxes: req.ProtectedAxes,
		})
		if err != nil {
			return MultiViewportReport{}, fmt.Errorf("multiviewport: critique for %dx%d: %w", vp.Width, vp.Height, err)
		}

		vpDuration := time.Since(vpStart)
		report.Viewports = append(report.Viewports, ViewportResult{
			Viewport:     vp,
			Packet:       packet,
			Verification: vResult,
			Critique:     cPass,
			Duration:     vpDuration,
		})

		totalFindings += len(cPass.Findings) + len(vResult.Issues)
		totalHard += cPass.HardViolations
		if cPass.GroundedScore < minScore {
			minScore = cPass.GroundedScore
		}

		// Index findings with element localization
		for _, f := range cPass.Findings {
			for _, elID := range f.ElementIDs {
				report.LocalizedByEl[elID] = append(report.LocalizedByEl[elID], f)
			}
			if len(f.ElementIDs) == 0 {
				report.LocalizedByEl["global"] = append(report.LocalizedByEl["global"], f)
			}
		}
		for _, iss := range vResult.Issues {
			findingFromIssue := design.Finding{
				ID:             fmt.Sprintf("finding:det:%s", iss.Code),
				Axis:           "deterministic",
				Category:       iss.Code,
				Title:          iss.Code,
				Description:    iss.Message,
				Severity:       iss.Severity,
				Confidence:     1.0,
				HardConstraint: true,
				ElementIDs:     iss.ElementIDs,
			}
			for _, elID := range iss.ElementIDs {
				report.LocalizedByEl[elID] = append(report.LocalizedByEl[elID], findingFromIssue)
			}
			if len(iss.ElementIDs) == 0 {
				report.LocalizedByEl["global"] = append(report.LocalizedByEl["global"], findingFromIssue)
			}
		}
	}

	report.TotalFindings = totalFindings
	report.HardViolations = totalHard
	report.GroundedScore = minScore
	report.TotalDuration = time.Since(started)

	// Sort element keys for deterministic output
	for k := range report.LocalizedByEl {
		sort.Slice(report.LocalizedByEl[k], func(i, j int) bool {
			return report.LocalizedByEl[k][i].ID < report.LocalizedByEl[k][j].ID
		})
	}

	return report, nil
}
