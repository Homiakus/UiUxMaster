package fidelity

import (
	"errors"
	"fmt"
)

// EvidenceClass categorizes the capability scope required to legally prove UI properties.
type EvidenceClass string

const (
	EvidenceClassStaticLayout        EvidenceClass = "static_layout"
	EvidenceClassTypography          EvidenceClass = "typography"
	EvidenceClassInteractive         EvidenceClass = "interactive"
	EvidenceClassPixelRegression     EvidenceClass = "pixel_regression"
	EvidenceClassCrossBrowserRelease EvidenceClass = "cross_browser_release"
)

// Tier represents an authoritative execution tier in UiUxMaster.
type Tier string

const (
	TierL0 Tier = "L0" // Static preflight
	TierL1 Tier = "L1" // WGGo FastRender
	TierL2 Tier = "L2" // FastBrowser Raw-CDP
	TierL3 Tier = "L3" // TruthPath Playwright
)

var (
	ErrIllegalPass = errors.New("fidelity: illegal pass on approximate tier")
)

// CalibrationRule defines legal pass permissions and escalation thresholds.
type CalibrationRule struct {
	Class            EvidenceClass `json:"class"`
	AllowedTiers     []Tier        `json:"allowed_tiers"`
	FinalGateAllowed []Tier        `json:"final_gate_allowed"`
	DefaultTier      Tier          `json:"default_tier"`
	EscalationTier   Tier          `json:"escalation_tier"`
	Description      string        `json:"description"`
}

// CalibrationMatrix enforces that no tier claims a PASS it cannot legally prove.
type CalibrationMatrix struct {
	rules map[EvidenceClass]CalibrationRule
}

// DefaultCalibrationMatrix returns the canonical UiUxMaster calibration matrix.
func DefaultCalibrationMatrix() *CalibrationMatrix {
	m := &CalibrationMatrix{
		rules: make(map[EvidenceClass]CalibrationRule),
	}

	m.rules[EvidenceClassStaticLayout] = CalibrationRule{
		Class:            EvidenceClassStaticLayout,
		AllowedTiers:     []Tier{TierL1, TierL2, TierL3},
		FinalGateAllowed: []Tier{TierL2, TierL3},
		DefaultTier:      TierL1,
		EscalationTier:   TierL2,
		Description:      "Simple static layout and box bounds; L1 speculative or L2 confirmed",
	}

	m.rules[EvidenceClassTypography] = CalibrationRule{
		Class:            EvidenceClassTypography,
		AllowedTiers:     []Tier{TierL2, TierL3},
		FinalGateAllowed: []Tier{TierL2, TierL3},
		DefaultTier:      TierL2,
		EscalationTier:   TierL2,
		Description:      "Custom font rendering and metrics; L1 prohibited, L2/L3 required",
	}

	m.rules[EvidenceClassInteractive] = CalibrationRule{
		Class:            EvidenceClassInteractive,
		AllowedTiers:     []Tier{TierL2, TierL3},
		FinalGateAllowed: []Tier{TierL2, TierL3},
		DefaultTier:      TierL2,
		EscalationTier:   TierL3,
		Description:      "Interactive states, click targets, ARIA trees; L1 prohibited, L2/L3 required",
	}

	m.rules[EvidenceClassPixelRegression] = CalibrationRule{
		Class:            EvidenceClassPixelRegression,
		AllowedTiers:     []Tier{TierL1, TierL2, TierL3},
		FinalGateAllowed: []Tier{TierL2, TierL3},
		DefaultTier:      TierL1,
		EscalationTier:   TierL2,
		Description:      "Pixel-level visual comparison; L1 fast diff, L2 Blink truth, L3 golden oracle",
	}

	m.rules[EvidenceClassCrossBrowserRelease] = CalibrationRule{
		Class:            EvidenceClassCrossBrowserRelease,
		AllowedTiers:     []Tier{TierL3},
		FinalGateAllowed: []Tier{TierL3},
		DefaultTier:      TierL3,
		EscalationTier:   TierL3,
		Description:      "Clean-state release gates across multi-browser matrix; L3 mandatory",
	}

	return m
}

// CanLegallyPass checks whether the specified tier is authorized to issue a final PASS
// for the given evidence class.
func (m *CalibrationMatrix) CanLegallyPass(tier Tier, class EvidenceClass, finalGate bool) bool {
	rule, ok := m.rules[class]
	if !ok {
		// Unknown evidence class fails closed: only L3 may pass
		return tier == TierL3
	}

	targetList := rule.AllowedTiers
	if finalGate {
		targetList = rule.FinalGateAllowed
	}

	for _, allowed := range targetList {
		if allowed == tier {
			return true
		}
	}
	return false
}

// RequiredEscalationTier returns the minimum tier required when a lower tier cannot legally pass.
func (m *CalibrationMatrix) RequiredEscalationTier(class EvidenceClass, finalGate bool) Tier {
	rule, ok := m.rules[class]
	if !ok {
		return TierL3
	}
	if finalGate {
		return rule.EscalationTier
	}
	return rule.DefaultTier
}

// ValidateLegalPass returns an error if a tier attempts an illegal pass.
func (m *CalibrationMatrix) ValidateLegalPass(tier Tier, class EvidenceClass, finalGate bool) error {
	if !m.CanLegallyPass(tier, class, finalGate) {
		return fmt.Errorf("%w: tier %s cannot legally PASS evidence class %q (finalGate=%v)", ErrIllegalPass, tier, class, finalGate)
	}
	return nil
}
