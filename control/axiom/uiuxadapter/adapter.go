package uiuxadapter

import (
	"context"
	"strings"

	"github.com/Homiakus/UiUxMaster/control/axiom/controlplane"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

// Collector is the narrow execution-plane seam. A production implementation can
// own resident FastCDP/WGGo resources while Axiom sees only one coarse-grained
// CollectVerify activity.
type Collector interface {
	Collect(context.Context, controlplane.Change, evidenceplan.Plan) (evidence.Packet, error)
}

type Adapter struct {
	Collector Collector
	Pipeline  *engine.Pipeline
	Policy    verifier.Policy
}

// New is retained for collection/compatibility workflows, but the legacy path
// cannot issue DecisionPass because it does not carry canonical PassAuthority.
func New(collector Collector) *Adapter {
	return &Adapter{Collector: collector, Policy: verifier.DefaultPolicy()}
}

func NewPipelineAdapter(pipeline *engine.Pipeline) *Adapter {
	return &Adapter{Pipeline: pipeline, Policy: verifier.DefaultPolicy()}
}

func (a *Adapter) PlanEvidence(_ context.Context, change controlplane.Change) (controlplane.EvidencePlan, error) {
	plan := evidenceplan.Build(canonicalSignals(change))
	return fromPlan(plan), nil
}

func (a *Adapter) CollectVerify(ctx context.Context, change controlplane.Change, requested controlplane.EvidencePlan) (controlplane.ValidationResult, error) {
	plan := toPlan(requested)

	var packet evidence.Packet
	var report engine.Report

	if a.Pipeline != nil {
		req := toValidationRequest(change)
		res, err := a.Pipeline.Execute(ctx, req)
		if err != nil {
			return controlplane.ValidationResult{}, err
		}
		packet = res.Packet
		report = res.Report
	} else if a.Collector != nil {
		var err error
		packet, err = a.Collector.Collect(ctx, change, plan)
		if err != nil {
			return controlplane.ValidationResult{}, err
		}
		policy := a.Policy
		verifier.ApplyDeterministic(&packet, policy)
		report = engine.EvaluateForPlan(packet, plan)
	}

	diagnosticsComplete := packet.Diagnostics != nil && packet.Diagnostics.Complete
	return controlplane.ValidationResult{
		BlockingFindings:     report.BlockingFindings,
		HighFindings:         report.HighFindings,
		MissingEvidence:      append([]string(nil), report.MissingEvidence...),
		VisualRegions:        len(packet.VisualRegions),
		VisualFindings:       len(packet.VisualFindings),
		PixelEvidence:        packet.Pixels != nil || packet.ScreenshotPath != "",
		DiagnosticsComplete: diagnosticsComplete,
		Summary:             report.RecommendedNext,
	}, nil
}

func (a *Adapter) Decide(_ context.Context, _ controlplane.Change, _ controlplane.EvidencePlan, result controlplane.ValidationResult) (controlplane.Decision, error) {
	switch {
	case result.BlockingFindings > 0 || result.HighFindings > 0:
		return controlplane.DecisionRepair, nil
	case len(result.MissingEvidence) > 0:
		if missingOnlyPixels(result.MissingEvidence) {
			return controlplane.DecisionPixels, nil
		}
		return controlplane.DecisionRecollect, nil
	case result.VisualRegions > 0 && result.VisualFindings == 0 && !result.PixelEvidence:
		return controlplane.DecisionPixels, nil
	case result.VisualRegions > 0 && result.VisualFindings == 0 && result.PixelEvidence:
		return controlplane.DecisionSemantic, nil
	case a == nil || a.Pipeline == nil:
		// Legacy collector mode has no canonical CalibrationAuthority/PassAuthority.
		// It remains useful for diagnostics but is never an authority for PASS.
		return controlplane.DecisionRecollect, nil
	default:
		return controlplane.DecisionPass, nil
	}
}

func canonicalSignals(change controlplane.Change) evidenceplan.Signals {
	req := toValidationRequest(change)
	req.Normalize()
	req.Need = req.DeriveNeed()
	signals := req.Signals(riskLevel(change.Risk))
	signals.CustomFontsChanged = signals.CustomFontsChanged || change.CustomFontsChanged
	signals.SemanticsChanged = signals.SemanticsChanged || change.SemanticsChanged
	signals.InteractionChanged = signals.InteractionChanged || change.InteractionChanged
	signals.RuntimeChanged = signals.RuntimeChanged || change.RuntimeChanged
	return signals
}

func riskLevel(value string) fidelity.RiskLevel {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(fidelity.RiskHigh):
		return fidelity.RiskHigh
	case string(fidelity.RiskMedium):
		return fidelity.RiskMedium
	default:
		return fidelity.RiskLow
	}
}

func fromPlan(plan evidenceplan.Plan) controlplane.EvidencePlan {
	return controlplane.EvidencePlan{
		Structural: plan.Structural, Diagnostics: plan.Diagnostics,
		Accessibility: plan.Accessibility, Fonts: plan.Fonts,
		Pixels: plan.Pixels, BrowserTruth: plan.BrowserTruth,
		Reasons: append([]string(nil), plan.Reasons...),
	}
}

func toPlan(plan controlplane.EvidencePlan) evidenceplan.Plan {
	return evidenceplan.Plan{
		Structural: plan.Structural, Diagnostics: plan.Diagnostics,
		Accessibility: plan.Accessibility, Fonts: plan.Fonts,
		Pixels: plan.Pixels, BrowserTruth: plan.BrowserTruth,
		Reasons: append([]string(nil), plan.Reasons...),
	}
}

func missingOnlyPixels(missing []string) bool {
	if len(missing) == 0 {
		return false
	}
	for _, item := range missing {
		if item != "rendered region pixels" {
			return false
		}
	}
	return true
}

func toValidationRequest(change controlplane.Change) engine.ValidationRequest {
	need := engine.EvidenceNeed{
		Geometry:     change.Need.Geometry,
		Styles:       change.Need.Styles,
		Pixels:       change.Need.Pixels,
		Scenario:     change.Need.Scenario,
		CleanState:   change.Need.CleanState,
		CrossBrowser: change.Need.CrossBrowser,
		Semantic:     change.Need.Semantic,
	}

	need.Geometry = need.Geometry || change.SemanticsChanged
	need.Styles = need.Styles || change.CustomFontsChanged
	need.Scenario = need.Scenario || change.InteractionChanged
	need.CleanState = need.CleanState || change.RuntimeChanged

	req := engine.ValidationRequest{
		RunID:            change.RunID,
		ProjectID:        change.ProjectID,
		SourceDigest:     change.SourceDigest,
		ChangedFiles:     append([]string(nil), change.ChangedFiles...),
		ChangedTokens:    append([]string(nil), change.ChangedTokens...),
		ChangedNodes:     append([]string(nil), change.ChangedNodes...),
		Intent:           evidenceplan.Intent(change.Intent),
		FinalGate:        change.FinalGate,
		RequireLegalPass: true,
		ForceWholeSite:   change.ForceWholeSite,
		TargetRoutes:     append([]string(nil), change.TargetRoutes...),
		Viewports:        append([]string(nil), change.Viewports...),
		Themes:           append([]string(nil), change.Themes...),
		Need:             need,
		BaseURL:          change.BaseURL,
	}
	if change.Region != nil {
		req.Region = &evidenceplan.Region{
			X:      change.Region.X,
			Y:      change.Region.Y,
			Width:  change.Region.Width,
			Height: change.Region.Height,
			Scale:  change.Region.Scale,
		}
	}
	return req
}
