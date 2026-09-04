package design

import "github.com/Homiakus/UiUxMaster/internal/evidence"

// Rule defines an explicit, versioned guideline or hard constraint.
type Rule struct {
	ID             string            `json:"id"`
	Version        string            `json:"version"`
	Axis           string            `json:"axis"`
	Category       string            `json:"category"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	HardConstraint bool              `json:"hard_constraint"`
	Severity       evidence.Severity `json:"severity"`
	Remediation    string            `json:"remediation"`
}

// CanonicalRules returns the versioned rules for premium editorial, motion, and responsive quality.
func CanonicalRules() []Rule {
	return []Rule{
		// Typography & Editorial
		{
			ID:             "RULE-TYPO-001",
			Version:        "1.0.0",
			Axis:           "typography",
			Category:       "editorial",
			Name:           "Line Length Bounded",
			Description:    "Body copy should maintain readable line length (45-80 characters per line) to prevent eye fatigue.",
			HardConstraint: false,
			Severity:       evidence.SeverityLow,
			Remediation:    "Set max-width: 65ch on body text containers.",
		},
		{
			ID:             "RULE-TYPO-002",
			Version:        "1.0.0",
			Axis:           "typography",
			Category:       "hierarchy",
			Name:           "Single Heading Level 1",
			Description:    "Every page must contain exactly one primary <h1> element for document outline clarity.",
			HardConstraint: true,
			Severity:       evidence.SeverityHigh,
			Remediation:    "Ensure only one <h1> is present and subsequent sections use <h2>-<h6>.",
		},
		{
			ID:             "RULE-TYPO-003",
			Version:        "1.0.0",
			Axis:           "typography",
			Category:       "fonts",
			Name:           "Font Settlement Required",
			Description:    "Web fonts must settle and load without fallback font shift or failed requests before final release.",
			HardConstraint: true,
			Severity:       evidence.SeverityHigh,
			Remediation:    "Use font-display: swap and verify @font-face URLs.",
		},

		// Responsive & Layout
		{
			ID:             "RULE-RESP-001",
			Version:        "1.0.0",
			Axis:           "responsive",
			Category:       "overflow",
			Name:           "No Horizontal Viewport Overflow",
			Description:    "Content must fit within the horizontal viewport boundaries without creating unwanted horizontal scrollbars.",
			HardConstraint: true,
			Severity:       evidence.SeverityCritical,
			Remediation:    "Use max-width: 100%, box-sizing: border-box, and avoid fixed-width overflow elements.",
		},
		{
			ID:             "RULE-RESP-002",
			Version:        "1.0.0",
			Axis:           "responsive",
			Category:       "fluid",
			Name:           "Fluid Container Scaling",
			Description:    "Layout grids and flex containers should scale gracefully across mobile, tablet, and wide desktop viewports.",
			HardConstraint: false,
			Severity:       evidence.SeverityMedium,
			Remediation:    "Use responsive grid (repeat(auto-fit, minmax(...))) or container queries.",
		},

		// Interaction & Accessibility
		{
			ID:             "RULE-A11Y-001",
			Version:        "1.0.0",
			Axis:           "accessibility",
			Category:       "target_size",
			Name:           "Minimum Touch Target Size",
			Description:    "Interactive elements must satisfy minimum touch target size (at least 24x24px, 44x44px recommended for mobile).",
			HardConstraint: true,
			Severity:       evidence.SeverityMedium,
			Remediation:    "Add min-width / min-height or padding to interactive targets.",
		},
		{
			ID:             "RULE-A11Y-002",
			Version:        "1.0.0",
			Axis:           "accessibility",
			Category:       "name",
			Name:           "Actionable Controls Must Have Accessible Name",
			Description:    "Buttons, links, and input elements must have clear text content, aria-label, or label association.",
			HardConstraint: true,
			Severity:       evidence.SeverityHigh,
			Remediation:    "Add visible text or aria-label to icon buttons and form controls.",
		},
		{
			ID:             "RULE-A11Y-003",
			Version:        "1.0.0",
			Axis:           "accessibility",
			Category:       "visibility",
			Name:           "Interactive Controls Must Be Available",
			Description:    "Interactive elements must not be invisible, zero-sized, or disabled with pointer-events:none while active.",
			HardConstraint: true,
			Severity:       evidence.SeverityHigh,
			Remediation:    "Ensure interactive elements are displayed and clickable.",
		},

		// Motion & Feedback
		{
			ID:             "RULE-MOT-001",
			Version:        "1.0.0",
			Axis:           "motion",
			Category:       "reduced_motion",
			Name:           "Respect Reduced Motion Preference",
			Description:    "Non-essential animations and transitions must be paused or removed when prefers-reduced-motion is active.",
			HardConstraint: true,
			Severity:       evidence.SeverityMedium,
			Remediation:    "Wrap decorative animations in @media (prefers-reduced-motion: no-preference).",
		},
		{
			ID:             "RULE-MOT-002",
			Version:        "1.0.0",
			Axis:           "motion",
			Category:       "performance",
			Name:           "Bounded Transition Duration",
			Description:    "Interactive feedback transitions should stay within 100-300ms for snappy responsiveness.",
			HardConstraint: false,
			Severity:       evidence.SeverityLow,
			Remediation:    "Use transition: transform 200ms ease, opacity 200ms ease.",
		},

		// Visual Craft & Composition
		{
			ID:             "RULE-CRAFT-001",
			Version:        "1.0.0",
			Axis:           "craft",
			Category:       "whitespace",
			Name:           "Consistent Spacing Scale",
			Description:    "Spacing, paddings, and margins should derive from a systematic scale (e.g. 4px/8px base).",
			HardConstraint: false,
			Severity:       evidence.SeverityLow,
			Remediation:    "Use CSS custom properties (--space-1, --space-2, etc.).",
		},
		{
			ID:             "RULE-CRAFT-002",
			Version:        "1.0.0",
			Axis:           "craft",
			Category:       "clipping",
			Name:           "No Unintended Ancestor Clipping",
			Description:    "Interactive elements, dropdowns, and text content must not be unintentionally clipped by overflow:hidden containers.",
			HardConstraint: true,
			Severity:       evidence.SeverityHigh,
			Remediation:    "Adjust ancestor height/padding or use position: fixed / popover for floating menus.",
		},
	}
}

// RuleIndex indexes rules by ID and axis for rapid lookup.
type RuleIndex struct {
	byID   map[string]Rule
	byAxis map[string][]Rule
}

// NewRuleIndex builds an index over a set of rules.
func NewRuleIndex(rules []Rule) *RuleIndex {
	idx := &RuleIndex{
		byID:   make(map[string]Rule, len(rules)),
		byAxis: make(map[string][]Rule),
	}
	for _, r := range rules {
		idx.byID[r.ID] = r
		idx.byAxis[r.Axis] = append(idx.byAxis[r.Axis], r)
	}
	return idx
}

// Get finds a rule by its ID.
func (idx *RuleIndex) Get(id string) (Rule, bool) {
	r, ok := idx.byID[id]
	return r, ok
}

// ByAxis returns all rules associated with a specific design axis.
func (idx *RuleIndex) ByAxis(axis string) []Rule {
	return idx.byAxis[axis]
}
