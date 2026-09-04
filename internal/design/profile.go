package design

// ProductProfile defines the target design system personality, typography ratios, density, and constraint strictness.
type ProductProfile struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	TypographyScale    string            `json:"typography_scale"` // e.g. "major-third" (1.25), "perfect-fourth" (1.333), "golden-ratio" (1.618)
	BaseFontSize       int               `json:"base_font_size"`   // e.g. 16px
	LineLengthMax      int               `json:"line_length_max"`  // e.g. 65 characters
	Density            string            `json:"density"`          // "compact", "normal", "spacious"
	BaseBorderRadius   string            `json:"base_border_radius"`
	SupportedColorModes []string         `json:"supported_color_modes"` // e.g. ["light", "dark", "system"]
	StrictA11y         bool              `json:"strict_a11y"`
	MotionAllowed      bool              `json:"motion_allowed"`
	CustomProperties   map[string]string `json:"custom_properties,omitempty"`
}

// CanonicalProfiles returns predefined product profiles for agent use.
func CanonicalProfiles() []ProductProfile {
	return []ProductProfile{
		{
			ID:                  "editorial-minimal",
			Name:                "Editorial & Minimalist",
			Description:         "High-contrast editorial layout with generous whitespace, disciplined typography, and restrained accent colors.",
			TypographyScale:     "perfect-fourth",
			BaseFontSize:        18,
			LineLengthMax:       65,
			Density:             "spacious",
			BaseBorderRadius:    "0px",
			SupportedColorModes: []string{"light", "dark"},
			StrictA11y:          true,
			MotionAllowed:       true,
			CustomProperties: map[string]string{
				"art_direction": "swiss-modernist",
				"contrast_ratio": "high",
			},
		},
		{
			ID:                  "saas-modern",
			Name:                "Modern SaaS Web App",
			Description:         "Clean modern interface with clear visual hierarchy, accessible touch targets, and balanced density.",
			TypographyScale:     "major-third",
			BaseFontSize:        16,
			LineLengthMax:       75,
			Density:             "normal",
			BaseBorderRadius:    "8px",
			SupportedColorModes: []string{"light", "dark", "system"},
			StrictA11y:          true,
			MotionAllowed:       true,
		},
		{
			ID:                  "dashboard-data-dense",
			Name:                "Data-Dense Professional Dashboard",
			Description:         "Compact tables, metrics, cards, and toolbars optimized for high-density information display.",
			TypographyScale:     "minor-third",
			BaseFontSize:        14,
			LineLengthMax:       90,
			Density:             "compact",
			BaseBorderRadius:    "4px",
			SupportedColorModes: []string{"dark", "light"},
			StrictA11y:          true,
			MotionAllowed:       false,
			CustomProperties: map[string]string{
				"tabular_nums": "true",
			},
		},
	}
}

// FindProfile returns the product profile by ID or falls back to "saas-modern".
func FindProfile(id string) ProductProfile {
	profiles := CanonicalProfiles()
	for _, p := range profiles {
		if p.ID == id {
			return p
		}
	}
	return profiles[1] // Default to saas-modern
}
