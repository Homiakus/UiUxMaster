package design

import (
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// HierarchyLevel represents the progressive attention depth for visual critique.
type HierarchyLevel string

const (
	LevelPage      HierarchyLevel = "page"
	LevelSection   HierarchyLevel = "section"
	LevelComponent HierarchyLevel = "component"
	LevelElement   HierarchyLevel = "element"
	LevelPixel     HierarchyLevel = "pixel"
)

// EvidenceRef connects a design finding directly to verifiable artifacts.
type EvidenceRef struct {
	Kind       string `json:"kind"` // "screenshot", "roi_crop", "dom_snapshot", "ax_node", "visual_diff"
	Digest     string `json:"digest,omitempty"`
	URI        string `json:"uri,omitempty"`
	RegionID   string `json:"region_id,omitempty"`
	ElementID  string `json:"element_id,omitempty"`
}

// Finding represents a grounded semantic or deterministic defect in UI/UX quality.
type Finding struct {
	ID             string            `json:"id"`
	Axis           string            `json:"axis"` // e.g. "typography", "composition", "accessibility"
	Category       string            `json:"category"`
	RuleID         string            `json:"rule_id,omitempty"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Severity       evidence.Severity `json:"severity"`
	Confidence     float64           `json:"confidence"` // 0.0 to 1.0
	HardConstraint bool              `json:"hard_constraint"`
	Level          HierarchyLevel    `json:"level"`
	RegionID       string            `json:"region_id,omitempty"`
	ElementIDs     []string          `json:"element_ids,omitempty"`
	EvidenceRefs   []EvidenceRef     `json:"evidence_refs,omitempty"`
	Suggestion     string            `json:"suggestion,omitempty"`
}

// RepairHypothesis describes a concrete actionable fix proposed for one or more findings.
type RepairHypothesis struct {
	ID              string   `json:"id"`
	FindingIDs      []string `json:"finding_ids"`
	Strategy        string   `json:"strategy"` // "css_token_update", "layout_reflow", "semantic_role_fix", etc.
	TargetFiles     []string `json:"target_files,omitempty"`
	ProposedChanges string   `json:"proposed_changes"`
	ExpectedOutcome string   `json:"expected_outcome"`
	Confidence      float64  `json:"confidence"`
}

// CritiquePass encapsulates the results of one hierarchical inspection pass.
type CritiquePass struct {
	ID              string             `json:"id"`
	Level           HierarchyLevel     `json:"level"`
	TargetID        string             `json:"target_id,omitempty"`
	Findings        []Finding          `json:"findings"`
	Hypotheses      []RepairHypothesis `json:"hypotheses,omitempty"`
	Duration        time.Duration      `json:"duration"`
	GroundedScore   float64            `json:"grounded_score"` // 0.0 to 10.0 (relative)
	HardViolations  int                `json:"hard_violations"`
}

// AxisScore provides per-axis relative evaluation.
type AxisScore struct {
	AxisID     string  `json:"axis_id"`
	Baseline   float64 `json:"baseline"`
	Candidate  float64 `json:"candidate"`
	Preference string  `json:"preference"` // "candidate", "baseline", "neutral"
	Rationale  string  `json:"rationale"`
}

// CandidateComparison represents a pairwise A/B comparison across rubric axes.
type CandidateComparison struct {
	RunID               string      `json:"run_id"`
	BaselineID          string      `json:"baseline_id"`
	CandidateID         string      `json:"candidate_id"`
	AxisScores          []AxisScore `json:"axis_scores"`
	PreferredCandidate  string      `json:"preferred_candidate"` // ID of the winner
	Rationale           string      `json:"rationale"`
	HardViolations      int         `json:"hard_violations"`
	RegressedAxes       []string    `json:"regressed_axes,omitempty"`
	PassedConstraints   bool        `json:"passed_constraints"`
}
