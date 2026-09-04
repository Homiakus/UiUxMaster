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
	Policy    verifier.Policy
}

func New(collector Collector) *Adapter {
	return &Adapter{Collector: collector, Policy: verifier.DefaultPolicy()}
}

func (a *Adapter) PlanEvidence(_ context.Context, change controlplane.Change) (controlplane.EvidencePlan, error) {
	plan := evidenceplan.Build(toSignals(change))
	return fromPlan(plan), nil
}

func (a *Adapter) CollectVerify(ctx context.Context, change controlplane.Change, requested controlplane.EvidencePlan) (controlplane.ValidationResult, error) {
	plan := toPlan(requested)
	packet, err := a.Collector.Collect(ctx, change, plan)
	if err != nil {
		return controlplane.ValidationResult{}, err
	}
	policy := a.Policy
	verifier.ApplyDeterministic(&packet, policy)
	report := engine.EvaluateForPlan(packet, plan)

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
	default:
		return controlplane.DecisionPass, nil
	}
}

func toSignals(change controlplane.Change) evidenceplan.Signals {
	signals := evidenceplan.Signals{
		Intent:             evidenceplan.Intent(change.Intent),
		Risk:               riskLevel(change.Risk),
		CustomFontsChanged: change.CustomFontsChanged,
		SemanticsChanged:   change.SemanticsChanged,
		InteractionChanged: change.InteractionChanged,
		RuntimeChanged:     change.RuntimeChanged,
		FinalGate:          change.FinalGate,
	}
	if change.Region != nil {
		signals.Region = &evidenceplan.Region{
			X: change.Region.X, Y: change.Region.Y,
			Width: change.Region.Width, Height: change.Region.Height,
			Scale: change.Region.Scale,
		}
	}
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
