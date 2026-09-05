package evolution

import (
	"errors"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/memory"
)

var (
	ErrNonRegressionFailed   = errors.New("evolution: candidate regressed on protected axes or score")
	ErrInvalidCandidate      = errors.New("evolution: invalid candidate version or heuristic")
	ErrPromotionUnauthorized = errors.New("evolution: promotion authorization gate failed")
)

// CandidateHeuristic captures an empirical pattern extracted from admitted evidence.
type CandidateHeuristic struct {
	ID               string                `json:"id"`
	SkillID          string                `json:"skill_id"`
	Category         string                `json:"category"`
	ProposedRule     memory.DesignRuleAtom `json:"proposed_rule"`
	SourceFindingIDs []string              `json:"source_finding_ids"`
	Confidence       float64               `json:"confidence"`
	CreatedAt        time.Time             `json:"created_at"`
}

// SkillVersion is an immutable, versioned bundle of rules and patterns for a skill.
type SkillVersion struct {
	VersionID            string                     `json:"version_id"`
	SkillID              string                     `json:"skill_id"`
	ActiveRules          []memory.DesignRuleAtom    `json:"active_rules"`
	ActiveRepairPatterns []memory.RepairPatternAtom `json:"active_repair_patterns,omitempty"`
	Heuristics           []CandidateHeuristic       `json:"heuristics,omitempty"`
	IsActive             bool                       `json:"is_active"`
	CreatedAt            time.Time                  `json:"created_at"`
	PromotedAt           time.Time                  `json:"promoted_at,omitempty"`
}

// ReplayCase represents a deterministic historical benchmark fixture.
type ReplayCase struct {
	CaseID                 string              `json:"case_id"`
	Description            string              `json:"description"`
	InputCritique          design.CritiquePass `json:"input_critique"`
	ExpectedHardViolations int                 `json:"expected_hard_violations"`
	BaselineScore          float64             `json:"baseline_score"`
}

// EvaluationReport summarizes the replay evaluation outcomes for a version.
type EvaluationReport struct {
	VersionID      string   `json:"version_id"`
	TotalCases     int      `json:"total_cases"`
	PassedCases    int      `json:"passed_cases"`
	FailedCases    int      `json:"failed_cases"`
	AverageScore   float64  `json:"average_score"`
	RegressedAxes  []string `json:"regressed_axes,omitempty"`
	HardViolations int      `json:"hard_violations"`
	PassedGate     bool     `json:"passed_gate"`
	Rationale      string   `json:"rationale"`
}

// PromotionAuthorization binds the exact active/candidate/corpus tuple to an
// independent verifier. PromoteCandidate accepts no implicit "we ran shadow
// earlier" state: this durable-shaped token must exist and match the corpus.
type PromotionAuthorization struct {
	ActiveVersionID    string    `json:"active_version_id"`
	CandidateVersionID string    `json:"candidate_version_id"`
	CorpusDigest       string    `json:"corpus_digest"`
	VerifierID         string    `json:"verifier_id"`
	ReplayPassed       bool      `json:"replay_passed"`
	ShadowPassed       bool      `json:"shadow_passed"`
	Rationale          string    `json:"rationale"`
	IssuedAt           time.Time `json:"issued_at"`
}
