package memory

import (
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
)

// NodeKind represents the entity type in the SncSinCore design memory ontology.
type NodeKind string

const (
	NodeDesignFinding     NodeKind = "DesignFinding"
	NodeDesignRule        NodeKind = "DesignRule"
	NodeRepairPattern     NodeKind = "RepairPattern"
	NodeCounterexample    NodeKind = "Counterexample"
	NodeComponentPattern  NodeKind = "ComponentPattern"
	NodeProductProfile    NodeKind = "ProductProfile"
	NodeEvidenceArtifact  NodeKind = "EvidenceArtifact"
	NodeRenderEnvironment NodeKind = "RenderEnvironment"
	NodeEvaluationResult  NodeKind = "EvaluationResult"
	NodeResearchSource    NodeKind = "ResearchSource"
)

// RelationKind defines typed epistemic relationships between memory nodes.
type RelationKind string

const (
	RelEvidenceFor      RelationKind = "evidence_for"
	RelRefutes          RelationKind = "refutes"
	RelGeneralizes      RelationKind = "generalizes"
	RelObservedOn       RelationKind = "observed_on"
	RelCausedBy         RelationKind = "caused_by"
	RelRepairedBy       RelationKind = "repaired_by"
	RelImprovesAxis     RelationKind = "improves_axis"
	RelRegressesAxis    RelationKind = "regresses_axis"
	RelApplicableTo     RelationKind = "applicable_to"
	RelCounterexampleTo RelationKind = "counterexample_to"
	RelDerivedFrom      RelationKind = "derived_from"
)

// ProvenanceRecord retains full verifiable lineage for any memory atom.
type ProvenanceRecord struct {
	RunID          string        `json:"run_id"`
	EvidenceDigest string        `json:"evidence_digest"`
	Renderer       string        `json:"renderer"`       // "wggo", "fastcdp", "playwright"
	Tier           fidelity.Tier `json:"tier"`           // "L0", "L1", "L2", "L3"
	Environment    string        `json:"environment"`    // e.g. "chromium-blink-128", "wggo-rgba"
	RuleVersion    string        `json:"rule_version"`   // e.g. "v1.2.0"
	CriticVersion  string        `json:"critic_version"` // e.g. "local-semantic-v1"
	ProjectScope   string        `json:"project_scope"`  // e.g. "project-alpha" or "global"
	Timestamp      time.Time     `json:"timestamp"`
	Outcome        string        `json:"outcome"`        // "CONFIRMED", "REPAIRED", "REFUTED", "BENCHMARKED"
}

// MemoryAtom is the canonical node container in the epistemic graph.
type MemoryAtom struct {
	ID         string           `json:"id"`
	Kind       NodeKind         `json:"kind"`
	Namespace  Namespace        `json:"namespace"`
	Provenance ProvenanceRecord `json:"provenance"`
	Data       any              `json:"data"` // specific atom payload
	Tags       []string         `json:"tags,omitempty"`
	Confidence float64          `json:"confidence"` // 0.0 to 1.0
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// MemoryEdge represents a typed directed relationship between two memory atoms.
type MemoryEdge struct {
	FromID     string           `json:"from_id"`
	ToID       string           `json:"to_id"`
	Relation   RelationKind     `json:"relation"`
	Weight     float64          `json:"weight"`
	Provenance ProvenanceRecord `json:"provenance"`
	CreatedAt  time.Time        `json:"created_at"`
}

// Specific atom payload structures

// DesignFindingAtom represents an admitted design finding or defect.
type DesignFindingAtom struct {
	FindingID      string            `json:"finding_id"`
	Axis           string            `json:"axis"`
	Category       string            `json:"category"`
	RuleID         string            `json:"rule_id,omitempty"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Severity       evidence.Severity `json:"severity"`
	HardConstraint bool              `json:"hard_constraint"`
	RegionID       string            `json:"region_id,omitempty"`
	ElementIDs     []string          `json:"element_ids,omitempty"`
	Suggestion     string            `json:"suggestion,omitempty"`
}

// DesignRuleAtom represents an active or candidate design rule.
type DesignRuleAtom struct {
	RuleID         string  `json:"rule_id"`
	Axis           string  `json:"axis"`
	Category       string  `json:"category"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	HardConstraint bool    `json:"hard_constraint"`
	Weight         float64 `json:"weight"`
	Version        string  `json:"version"`
}

// RepairPatternAtom encapsulates a proven repair hypothesis and its success rate.
type RepairPatternAtom struct {
	PatternID       string   `json:"pattern_id"`
	Strategy        string   `json:"strategy"`
	FindingCategory string   `json:"finding_category"`
	TargetFiles     []string `json:"target_files,omitempty"`
	PatchSnippet    string   `json:"patch_snippet"`
	ExpectedOutcome string   `json:"expected_outcome"`
	SuccessCount    int      `json:"success_count"`
	FailureCount    int      `json:"failure_count"`
	SuccessRate     float64  `json:"success_rate"`
}

// CounterexampleAtom represents an observation refuting a proposed rule or repair.
type CounterexampleAtom struct {
	TargetEntityID string `json:"target_entity_id"`
	Reason         string `json:"reason"`
	RefutingDigest string `json:"refuting_digest"`
	Observation    string `json:"observation"`
}

// ComponentPatternAtom records UI component architectural and styling invariants.
type ComponentPatternAtom struct {
	ComponentName string   `json:"component_name"`
	Category      string   `json:"category"`
	RequiredRoles []string `json:"required_roles,omitempty"`
	MarkupPattern string   `json:"markup_pattern,omitempty"`
	CSSInvariants []string `json:"css_invariants,omitempty"`
}

// ProductProfileAtom contains brand, typography, and UX guidelines for a project.
type ProductProfileAtom struct {
	ProfileID      string             `json:"profile_id"`
	Name           string             `json:"name"`
	TargetDevices  []string           `json:"target_devices"`
	PrimaryFont    string             `json:"primary_font"`
	RubricWeights  map[string]float64 `json:"rubric_weights"`
	EditorialVoice string             `json:"editorial_voice"`
}

// EvidenceArtifactAtom references binary/DOM evidence by digest/URI.
type EvidenceArtifactAtom struct {
	Kind       string `json:"kind"` // "screenshot", "roi_crop", "dom_snapshot", "visual_diff"
	Digest     string `json:"digest"`
	URI        string `json:"uri,omitempty"`
	SizeBytes  int64  `json:"size_bytes"`
	Dimensions string `json:"dimensions,omitempty"` // e.g. "1280x800"
}

// RenderEnvironmentAtom records browser/renderer state for calibration.
type RenderEnvironmentAtom struct {
	Renderer      string  `json:"renderer"`
	BrowserFamily string  `json:"browser_family"`
	ViewportW     int     `json:"viewport_w"`
	ViewportH     int     `json:"viewport_h"`
	DeviceScale   float64 `json:"device_scale"`
	Theme         string  `json:"theme"`
	UserAgent     string  `json:"user_agent,omitempty"`
}

// EvaluationResultAtom records metric scores and benchmark results.
type EvaluationResultAtom struct {
	RunID          string             `json:"run_id"`
	Score          float64            `json:"score"`
	Passed         bool               `json:"passed"`
	HardViolations int                `json:"hard_violations"`
	AxisScores     map[string]float64 `json:"axis_scores,omitempty"`
	DurationMS     int64              `json:"duration_ms"`
}

// ResearchSourceAtom captures authoritative external design or standard references.
type ResearchSourceAtom struct {
	SourceURI       string    `json:"source_uri"`
	Title           string    `json:"title"`
	Authors         []string  `json:"authors,omitempty"`
	PublicationDate string    `json:"publication_date,omitempty"`
	Summary         string    `json:"summary"`
	ExtractedClaims []string  `json:"extracted_claims,omitempty"`
	Digest          string    `json:"digest"`
	FetchedAt       time.Time `json:"fetched_at"`
}
