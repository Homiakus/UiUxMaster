package design

// Axis is one independently reviewable dimension of UI/UX quality.
// Scores are intentionally not treated as an oracle; the rubric primarily
// exists to structure evidence, relative comparisons, and defect localization.
type Axis struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
}

// DefaultRubric returns the first canonical design-critique axes used by
// UiUxMaster. Keep axes independent so a visual critic cannot hide a serious
// accessibility or interaction defect behind a high aesthetic score.
func DefaultRubric() []Axis {
	return []Axis{
		{ID: "identity", Name: "Identity & distinctiveness", Description: "Recognizable visual character without generic template or reference-site copying.", Weight: 1.0},
		{ID: "composition", Name: "Composition & hierarchy", Description: "Visual weight, grouping, alignment, whitespace, density, section rhythm, and CTA hierarchy.", Weight: 1.2},
		{ID: "typography", Name: "Typography", Description: "Scale relationships, legibility, line length, line-height, weight, optical balance, and editorial rhythm.", Weight: 1.2},
		{ID: "color", Name: "Color & contrast", Description: "Palette harmony, semantic color discipline, contrast, accent restraint, and light/dark resilience.", Weight: 1.0},
		{ID: "imagery", Name: "Imagery & art direction", Description: "Image relevance, crop, focal point, quality, compositional role, and responsive art direction.", Weight: 0.8},
		{ID: "interaction", Name: "Interaction & affordance", Description: "Discoverability, clear controls, state feedback, focus, hover/touch parity, and error recovery.", Weight: 1.2},
		{ID: "responsive", Name: "Responsive composition", Description: "Intentional behavior across containers, viewports, orientation, zoom, touch, and long/localized content.", Weight: 1.2},
		{ID: "accessibility", Name: "Accessibility", Description: "Keyboard access, focus visibility, semantic structure, contrast, motion preferences, and target sizing.", Weight: 1.3},
		{ID: "motion", Name: "Motion & feedback", Description: "Purposeful transitions and progress/status feedback without distraction or unnecessary repaint cost.", Weight: 0.7},
		{ID: "craft", Name: "Micro-craft", Description: "Optical alignment, border/radius consistency, icon rhythm, spacing precision, truncation, and edge-case polish.", Weight: 0.9},
	}
}
