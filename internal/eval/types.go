package eval

import (
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// DefectKind classifies an injected adversarial UI/UX flaw.
type DefectKind string

const (
	DefectViewportOverflow     DefectKind = "layout.viewport_overflow"
	DefectTargetTooSmall       DefectKind = "interaction.target_too_small"
	DefectInteractiveOverlap   DefectKind = "interaction.overlap"
	DefectFocusSequenceAnomaly DefectKind = "accessibility.focus_sequence"
	DefectDuplicateDOMID       DefectKind = "dom.duplicate_id"
	DefectTextTruncation       DefectKind = "typography.text_truncation"
	DefectMissingA11yName      DefectKind = "accessibility.missing_name"
	DefectFixedObstruction     DefectKind = "layout.fixed_obstruction"
	DefectHeadingHierarchy     DefectKind = "design.heading_hierarchy"
	DefectPointerDisabled      DefectKind = "interaction.pointer_disabled"
)

// DefectInjection defines a deterministic defect mutator applied to an evidence packet.
type DefectInjection struct {
	ID              string                                  `json:"id"`
	Kind            DefectKind                              `json:"kind"`
	Description     string                                  `json:"description"`
	TargetElementID string                                  `json:"target_element_id"`
	Mutate          func(packet *evidence.Packet)          `json:"-"`
}

// EvalCase bundles a clean baseline packet with a set of adversarial defect injections.
type EvalCase struct {
	Name            string            `json:"name"`
	BasePacket      evidence.Packet   `json:"base_packet"`
	InjectedDefects []DefectInjection `json:"injected_defects"`
}

// DefectStats records recall and detection statistics for a specific defect kind.
type DefectStats struct {
	Injected  int     `json:"injected"`
	Detected  int     `json:"detected"`
	Localized int     `json:"localized"`
	Recall    float64 `json:"recall"`
}

// EvalReport summarizes the accuracy, recall, and localization fidelity of UiUxMaster's verifiers.
type EvalReport struct {
	TotalCases       int                    `json:"total_cases"`
	TotalInjected    int                    `json:"total_injected"`
	TotalDetected    int                    `json:"total_detected"`
	TotalLocalized   int                    `json:"total_localized"`
	FalsePositives   int                    `json:"false_positives"`
	Recall           float64                `json:"recall"`
	Precision        float64                `json:"precision"`
	LocalizationRate float64                `json:"localization_rate"`
	ByKind           map[DefectKind]*DefectStats `json:"by_kind"`
}
