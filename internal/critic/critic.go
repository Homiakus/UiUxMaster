package critic

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// CritiqueRequest describes an inspection target and context for the semantic critic.
type CritiqueRequest struct {
	RunID         string                `json:"run_id"`
	Level         design.HierarchyLevel `json:"level"`
	TargetID      string                `json:"target_id,omitempty"`
	Profile       design.ProductProfile `json:"profile"`
	Packet        evidence.Packet       `json:"packet"`
	RegionCrop    *image.RGBA           `json:"-"`
	ProtectedAxes []string              `json:"protected_axes,omitempty"`
}

// Critic evaluates UI evidence and produces grounded semantic findings and repair hypotheses.
type Critic interface {
	Critique(ctx context.Context, req CritiqueRequest) (design.CritiquePass, error)
}

// LocalSemanticCritic implements a progressive, rule-grounded local critic.
type LocalSemanticCritic struct {
	ruleIndex *design.RuleIndex
}

// New creates an initialized LocalSemanticCritic.
func New() *LocalSemanticCritic {
	return &LocalSemanticCritic{
		ruleIndex: design.NewRuleIndex(design.CanonicalRules()),
	}
}

// Critique executes hierarchical evaluation across structure, fonts, layout, and accessibility.
func (c *LocalSemanticCritic) Critique(ctx context.Context, req CritiqueRequest) (design.CritiquePass, error) {
	if err := ctx.Err(); err != nil {
		return design.CritiquePass{}, err
	}

	start := time.Now()
	if req.Profile.ID == "" {
		req.Profile = design.FindProfile("saas-modern")
	}
	if req.Level == "" {
		req.Level = design.LevelPage
	}

	findings := make([]design.Finding, 0)
	hypotheses := make([]design.RepairHypothesis, 0)

	// 1. Heading Hierarchy Inspection (Single H1)
	h1Count := 0
	var h1IDs []string
	for _, el := range req.Packet.Elements {
		if strings.EqualFold(el.Tag, "h1") || strings.EqualFold(el.Role, "heading") && el.Attributes["aria-level"] == "1" {
			h1Count++
			h1IDs = append(h1IDs, el.ID)
		}
	}
	if h1Count == 0 {
		rule, _ := c.ruleIndex.Get("RULE-TYPO-002")
		f := design.Finding{
			ID:             fmt.Sprintf("finding:heading_missing:%s", req.RunID),
			Axis:           "typography",
			Category:       "hierarchy",
			RuleID:         rule.ID,
			Title:          "Page is missing a primary heading (h1)",
			Description:    "Document outline requires exactly one top-level h1 element.",
			Severity:       evidence.SeverityHigh,
			Confidence:     1.0,
			HardConstraint: true,
			Level:          req.Level,
			Suggestion:     rule.Remediation,
		}
		findings = append(findings, f)
		hypotheses = append(hypotheses, design.RepairHypothesis{
			ID:              fmt.Sprintf("repair:add_h1:%s", req.RunID),
			FindingIDs:      []string{f.ID},
			Strategy:        "document_outline_fix",
			ProposedChanges: "Add a prominent <h1> element for the main section title.",
			ExpectedOutcome: "Valid single primary heading satisfies document hierarchy outline.",
			Confidence:      1.0,
		})
	} else if h1Count > 1 {
		rule, _ := c.ruleIndex.Get("RULE-TYPO-002")
		f := design.Finding{
			ID:             fmt.Sprintf("finding:multiple_h1:%s", req.RunID),
			Axis:           "typography",
			Category:       "hierarchy",
			RuleID:         rule.ID,
			Title:          fmt.Sprintf("Multiple h1 elements detected (%d)", h1Count),
			Description:    "Document outline should contain only one primary h1.",
			Severity:       evidence.SeverityMedium,
			Confidence:     1.0,
			HardConstraint: true,
			Level:          req.Level,
			ElementIDs:     h1IDs,
			Suggestion:     "Change secondary headings to <h2>-<h6>.",
		}
		findings = append(findings, f)
	}

	// 2. Viewport Overflow Inspection
	for _, doc := range req.Packet.Documents {
		if req.Packet.Viewport.Width > 0 && doc.ContentWidth > float64(req.Packet.Viewport.Width) {
			overflow := doc.ContentWidth - float64(req.Packet.Viewport.Width)
			rule, _ := c.ruleIndex.Get("RULE-RESP-001")
			f := design.Finding{
				ID:             fmt.Sprintf("finding:overflow:%s", req.RunID),
				Axis:           "responsive",
				Category:       "overflow",
				RuleID:         rule.ID,
				Title:          fmt.Sprintf("Horizontal viewport overflow by %.1fpx", overflow),
				Description:    fmt.Sprintf("Document content width (%.1fpx) exceeds viewport width (%dpx)", doc.ContentWidth, req.Packet.Viewport.Width),
				Severity:       evidence.SeverityCritical,
				Confidence:     1.0,
				HardConstraint: true,
				Level:          req.Level,
				Suggestion:     rule.Remediation,
			}
			findings = append(findings, f)
			hypotheses = append(hypotheses, design.RepairHypothesis{
				ID:              fmt.Sprintf("repair:fix_overflow:%s", req.RunID),
				FindingIDs:      []string{f.ID},
				Strategy:        "css_max_width_constraint",
				ProposedChanges: "Add max-width: 100% and overflow-x: hidden to parent container.",
				ExpectedOutcome: "Eliminates unwanted horizontal scrollbars across mobile and desktop.",
				Confidence:      0.95,
			})
		}
	}

	// 3. Accessibility & Interactive Target Inspection
	for _, el := range req.Packet.Elements {
		if !el.Visible {
			continue
		}
		if (el.Clickable || el.Role == "button" || el.Role == "link") && el.Name == "" {
			rule, _ := c.ruleIndex.Get("RULE-A11Y-002")
			f := design.Finding{
				ID:             fmt.Sprintf("finding:a11y_name:%s:%s", req.RunID, el.ID),
				Axis:           "accessibility",
				Category:       "name",
				RuleID:         rule.ID,
				Title:          fmt.Sprintf("Actionable element %q lacks accessible name", el.ID),
				Description:    "Interactive control has empty computed accessible name.",
				Severity:       evidence.SeverityHigh,
				Confidence:     1.0,
				HardConstraint: true,
				Level:          design.LevelElement,
				ElementIDs:     []string{el.ID},
				Suggestion:     "Provide visible text label or aria-label attribute.",
			}
			findings = append(findings, f)
			hypotheses = append(hypotheses, design.RepairHypothesis{
				ID:              fmt.Sprintf("repair:add_aria_label:%s", el.ID),
				FindingIDs:      []string{f.ID},
				Strategy:        "add_aria_label",
				ProposedChanges: fmt.Sprintf(`Add aria-label="..." to element #%s`, el.ID),
				ExpectedOutcome: "Screen readers can announce the purpose of the control.",
				Confidence:      1.0,
			})
		}
	}

	// 4. Font Loading Settlement
	if req.Packet.Fonts != nil {
		for _, face := range req.Packet.Fonts.Faces {
			if strings.EqualFold(face.Status, "error") {
				rule, _ := c.ruleIndex.Get("RULE-TYPO-003")
				f := design.Finding{
					ID:             fmt.Sprintf("finding:font_error:%s:%s", req.RunID, face.Family),
					Axis:           "typography",
					Category:       "fonts",
					RuleID:         rule.ID,
					Title:          fmt.Sprintf("Web font %q failed to load", face.Family),
					Description:    "Font asset could not be loaded, causing layout shift to fallback font.",
					Severity:       evidence.SeverityHigh,
					Confidence:     1.0,
					HardConstraint: true,
					Level:          req.Level,
					Suggestion:     rule.Remediation,
				}
				findings = append(findings, f)
			}
		}
	}

	// 5. Compute Grounded Score and Hard Violations
	hardViolations := 0
	scorePenalty := 0.0
	for _, f := range findings {
		if f.HardConstraint {
			hardViolations++
		}
		switch f.Severity {
		case evidence.SeverityCritical:
			scorePenalty += 3.0
		case evidence.SeverityHigh:
			scorePenalty += 1.5
		case evidence.SeverityMedium:
			scorePenalty += 0.8
		case evidence.SeverityLow:
			scorePenalty += 0.3
		}
	}

	groundedScore := 10.0 - scorePenalty
	if groundedScore < 0 {
		groundedScore = 0
	}

	return design.CritiquePass{
		ID:             fmt.Sprintf("critique:%s:%s", req.Level, req.RunID),
		Level:          req.Level,
		TargetID:       req.TargetID,
		Findings:       findings,
		Hypotheses:     hypotheses,
		Duration:       time.Since(start),
		GroundedScore:  groundedScore,
		HardViolations: hardViolations,
	}, nil
}
