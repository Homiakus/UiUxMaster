package eval

import (
	"context"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/critic"
	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

// Harness runs systematic adversarial evaluations across mutated evidence packets.
type Harness struct {
	Policy verifier.Policy
	Critic *critic.LocalSemanticCritic
}

// NewHarness creates an initialized evaluation harness.
func NewHarness() *Harness {
	return &Harness{
		Policy: verifier.DefaultPolicy(),
		Critic: critic.New(),
	}
}

// Run executes the evaluation suite across all provided test cases and computes detection recall and localization rates.
func (h *Harness) Run(cases []EvalCase) EvalReport {
	report := EvalReport{
		TotalCases: len(cases),
		ByKind:     make(map[DefectKind]*DefectStats),
	}

	for _, c := range cases {
		// Clone base packet to isolate mutations
		mutated := clonePacket(c.BasePacket)

		// Apply defect injections
		for _, defect := range c.InjectedDefects {
			if defect.Mutate != nil {
				defect.Mutate(&mutated)
			}
			report.TotalInjected++
			stats, ok := report.ByKind[defect.Kind]
			if !ok {
				stats = &DefectStats{}
				report.ByKind[defect.Kind] = stats
			}
			stats.Injected++
		}

		// Run deterministic layout & interactive verifiers
		layoutRes := verifier.Verify(mutated, h.Policy)

		// Run accessibility verifiers
		a11yIssues := verifier.VerifyAccessibility(mutated)

		// Combine issues
		allIssues := append(append([]evidence.RuntimeIssue(nil), layoutRes.Issues...), a11yIssues...)

		// Run semantic critic for hierarchy inspection
		critiquePass, _ := h.Critic.Critique(context.Background(), critic.CritiqueRequest{
			Level:   design.LevelPage,
			Profile: design.FindProfile("saas-modern"),
			Packet:  mutated,
		})

		// Match detected issues against injected defects
		for _, defect := range c.InjectedDefects {
			stats := report.ByKind[defect.Kind]
			detected, localized := matchDefect(defect, allIssues, critiquePass.Findings)
			if detected {
				stats.Detected++
				report.TotalDetected++
				if localized {
					stats.Localized++
					report.TotalLocalized++
				}
			}
		}
	}

	// Calculate aggregate rates
	if report.TotalInjected > 0 {
		report.Recall = float64(report.TotalDetected) / float64(report.TotalInjected)
	}
	if report.TotalDetected > 0 {
		report.LocalizationRate = float64(report.TotalLocalized) / float64(report.TotalDetected)
		totalOutputs := report.TotalDetected + report.FalsePositives
		report.Precision = float64(report.TotalDetected) / float64(totalOutputs)
	}

	for _, stats := range report.ByKind {
		if stats.Injected > 0 {
			stats.Recall = float64(stats.Detected) / float64(stats.Injected)
		}
	}

	return report
}

func matchDefect(defect DefectInjection, issues []evidence.RuntimeIssue, findings []design.Finding) (detected bool, localized bool) {
	for _, issue := range issues {
		if issueMatchesKind(issue.Code, defect.Kind) {
			detected = true
			if defect.TargetElementID == "" {
				localized = true
			} else {
				for _, elID := range issue.ElementIDs {
					if elID == defect.TargetElementID {
						localized = true
						break
					}
				}
			}
			if detected && localized {
				return true, true
			}
		}
	}

	for _, f := range findings {
		if findingMatchesKind(f.RuleID, defect.Kind) {
			detected = true
			if defect.TargetElementID == "" {
				localized = true
			} else {
				for _, elID := range f.ElementIDs {
					if elID == defect.TargetElementID {
						localized = true
						break
					}
				}
			}
			if detected && localized {
				return true, true
			}
		}
	}

	return detected, localized
}

func issueMatchesKind(code string, kind DefectKind) bool {
	switch kind {
	case DefectViewportOverflow:
		return code == verifier.CodeViewportHorizontalOverflow
	case DefectTargetTooSmall:
		return code == verifier.CodeTargetTooSmall
	case DefectInteractiveOverlap:
		return code == verifier.CodeInteractiveOverlap
	case DefectFocusSequenceAnomaly:
		return code == verifier.CodeFocusSequenceAnomaly
	case DefectDuplicateDOMID:
		return code == verifier.CodeDuplicateDOMID
	case DefectTextTruncation:
		return code == verifier.CodeTextTruncationAnomaly
	case DefectMissingA11yName:
		return code == verifier.CodeA11yNameMissing
	case DefectFixedObstruction:
		return code == verifier.CodeFixedStickyObstruction
	case DefectPointerDisabled:
		return code == verifier.CodePointerEventsDisabled
	}
	return false
}

func findingMatchesKind(ruleID string, kind DefectKind) bool {
	switch kind {
	case DefectHeadingHierarchy:
		return strings.Contains(ruleID, "RULE-TYPO-002") || strings.Contains(ruleID, "single_h1") || strings.Contains(ruleID, "hierarchy")
	}
	return false
}

func clonePacket(p evidence.Packet) evidence.Packet {
	cloned := p
	cloned.Elements = append([]evidence.ElementRef(nil), p.Elements...)
	for i := range cloned.Elements {
		if p.Elements[i].Styles != nil {
			cloned.Elements[i].Styles = make(map[string]string, len(p.Elements[i].Styles))
			for k, v := range p.Elements[i].Styles {
				cloned.Elements[i].Styles[k] = v
			}
		}
		if p.Elements[i].Attributes != nil {
			cloned.Elements[i].Attributes = make(map[string]string, len(p.Elements[i].Attributes))
			for k, v := range p.Elements[i].Attributes {
				cloned.Elements[i].Attributes[k] = v
			}
		}
	}
	cloned.Documents = append([]evidence.DocumentMetrics(nil), p.Documents...)
	cloned.Accessibility = append([]evidence.AccessibilityNode(nil), p.Accessibility...)
	return cloned
}
