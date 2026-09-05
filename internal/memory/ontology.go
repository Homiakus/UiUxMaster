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

// ProvenanceRecord retains verifiable lineage for any memory atom. FMEA-010
// makes SourceNamespace authoritative for where evidence originated. Promotion
// fields are populated only by the explicit promotion authority; ordinary
// admission cannot forge a visibility expansion by choosing a global target.
type ProvenanceRecord struct {
	RunID          string        `json:"run_id"`
	EvidenceDigest string        `json:"evidence_digest"`
	Renderer       string        `json:"renderer"`
	Tier           fidelity.Tier `json:"tier"`
	Environment    string        `json:"environment"`
	RuleVersion    string        `json:"rule_version"`
	CriticVersion  string        `json:"critic_version"`
	ProjectScope   string        `json:"project_scope"`
	SourceNamespace string       `json:"source_namespace,omitempty"`
	SourceAtomIDs  []string      `json:"source_atom_ids,omitempty"`
	DecisionDigest string        `json:"decision_digest,omitempty"`
	VerifierID     string        `json:"verifier_id,omitempty"`
	Timestamp      time.Time     `json:"timestamp"`
	Outcome        string        `json:"outcome"`
}

// MemoryAtom is the canonical node container in the epistemic graph.
type MemoryAtom struct {
	ID         string           `json:"id"`
	Kind       NodeKind         `json:"kind"`
	Namespace  Namespace        `json:"namespace"`
	Provenance ProvenanceRecord `json:"provenance"`
	Data       any              `json:"data"`
	Tags       []string         `json:"tags,omitempty"`
	Confidence float64          `json:"confidence"`
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

type CounterexampleAtom struct {
	TargetEntityID string `json:"target_entity_id"`
	Reason         string `json:"reason"`
	RefutingDigest string `json:"refuting_digest"`
	Observation    string `json:"observation"`
}

type ComponentPatternAtom struct {
	ComponentName string   `json:"component_name"`
	Category      string   `json:"category"`
	RequiredRoles []string `json:"required_roles,omitempty"`
	MarkupPattern string   `json:"markup_pattern,omitempty"`
	CSSInvariants []string `json:"css_invariants,omitempty"`
}

type ProductProfileAtom struct {
	ProfileID      string             `json:"profile_id"`
	Name           string             `json:"name"`
	TargetDevices  []string           `json:"target_devices"`
	PrimaryFont    string             `json:"primary_font"`
	RubricWeights  map[string]float64 `json:"rubric_weights"`
	EditorialVoice string             `json:"editorial_voice"`
}

type EvidenceArtifactAtom struct {
	Kind      string `json:"kind"`
	Digest    string `json:"digest"`
	URI       string `json:"uri,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type RenderEnvironmentAtom struct {
	Renderer      string  `json:"renderer"`
	BrowserFamily string  `json:"browser_family"`
	BrowserVersion string `json:"browser_version,omitempty"`
	ViewportW     int     `json:"viewport_w"`
	ViewportH     int     `json:"viewport_h"`
	DeviceScale   float64 `json:"device_scale"`
	Theme         string  `json:"theme"`
	FontSetDigest string  `json:"font_set_digest,omitempty"`
}

type EvaluationResultAtom struct {
	RunID          string  `json:"run_id"`
	Score          float64 `json:"score"`
	Passed         bool    `json:"passed"`
	HardViolations int     `json:"hard_violations"`
	DurationMS     int64   `json:"duration_ms"`
}

type ResearchSourceAtom struct {
	SourceURI       string    `json:"source_uri"`
	Title           string    `json:"title"`
	Authors         []string  `json:"authors,omitempty"`
	PublicationDate time.Time `json:"publication_date,omitempty"`
	Summary         string    `json:"summary,omitempty"`
	Digest          string    `json:"digest"`
	FetchedAt       time.Time `json:"fetched_at"`
}
