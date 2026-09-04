package design_test

import (
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestCanonicalRulesAndIndex(t *testing.T) {
	rules := design.CanonicalRules()
	if len(rules) < 10 {
		t.Fatalf("expected at least 10 canonical rules, got %d", len(rules))
	}

	idx := design.NewRuleIndex(rules)

	// Test lookup by ID
	rule, ok := idx.Get("RULE-RESP-001")
	if !ok {
		t.Fatalf("expected to find RULE-RESP-001")
	}
	if !rule.HardConstraint {
		t.Errorf("RULE-RESP-001 should be a hard constraint")
	}
	if rule.Severity != evidence.SeverityCritical {
		t.Errorf("RULE-RESP-001 severity = %s, want critical", rule.Severity)
	}

	// Test lookup by Axis
	a11yRules := idx.ByAxis("accessibility")
	if len(a11yRules) < 3 {
		t.Errorf("expected at least 3 accessibility rules, got %d", len(a11yRules))
	}

	typoRules := idx.ByAxis("typography")
	if len(typoRules) < 3 {
		t.Errorf("expected at least 3 typography rules, got %d", len(typoRules))
	}
}

func TestCanonicalProfiles(t *testing.T) {
	profiles := design.CanonicalProfiles()
	if len(profiles) != 3 {
		t.Fatalf("expected 3 canonical profiles, got %d", len(profiles))
	}

	editorial := design.FindProfile("editorial-minimal")
	if editorial.Density != "spacious" {
		t.Errorf("editorial density = %s, want spacious", editorial.Density)
	}
	if editorial.BaseBorderRadius != "0px" {
		t.Errorf("editorial border radius = %s, want 0px", editorial.BaseBorderRadius)
	}

	dashboard := design.FindProfile("dashboard-data-dense")
	if dashboard.Density != "compact" {
		t.Errorf("dashboard density = %s, want compact", dashboard.Density)
	}

	fallback := design.FindProfile("non-existent")
	if fallback.ID != "saas-modern" {
		t.Errorf("fallback profile = %s, want saas-modern", fallback.ID)
	}
}

func TestDomainTypes_CritiqueAndComparison(t *testing.T) {
	finding := design.Finding{
		ID:             "finding-1",
		Axis:           "composition",
		Category:       "hierarchy",
		Title:          "Primary CTA lacks visual dominance",
		Severity:       evidence.SeverityHigh,
		Confidence:     0.95,
		HardConstraint: false,
		Level:          design.LevelSection,
		RegionID:       "hero-section",
		ElementIDs:     []string{"cta-btn"},
		EvidenceRefs: []design.EvidenceRef{
			{Kind: "roi_crop", RegionID: "hero-section"},
		},
		Suggestion: "Increase CTA padding and contrast ratio against hero background.",
	}

	pass := design.CritiquePass{
		ID:            "pass-hero-01",
		Level:         design.LevelSection,
		TargetID:      "hero-section",
		Findings:      []design.Finding{finding},
		Duration:      25 * time.Millisecond,
		GroundedScore: 8.2,
	}

	if len(pass.Findings) != 1 || pass.Findings[0].Level != design.LevelSection {
		t.Errorf("critique pass findings mismatch: %+v", pass.Findings)
	}

	comparison := design.CandidateComparison{
		RunID:       "cmp-run-1",
		BaselineID:  "candidate-A",
		CandidateID: "candidate-B",
		AxisScores: []design.AxisScore{
			{AxisID: "composition", Baseline: 6.5, Candidate: 8.5, Preference: "candidate", Rationale: "Clearer visual hierarchy"},
			{AxisID: "accessibility", Baseline: 10.0, Candidate: 10.0, Preference: "neutral", Rationale: "Both pass all WCAG criteria"},
		},
		PreferredCandidate: "candidate-B",
		PassedConstraints:  true,
	}

	if comparison.PreferredCandidate != "candidate-B" || !comparison.PassedConstraints {
		t.Errorf("candidate comparison mismatch: %+v", comparison)
	}
}
