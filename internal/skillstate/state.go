package skillstate

import (
	"errors"
	"time"
)

var (
	ErrStalePatch       = errors.New("skillstate: stale patch revision (CAS failed)")
	ErrDigestMismatch   = errors.New("skillstate: patch parent digest mismatch")
	ErrBudgetExhausted  = errors.New("skillstate: budget exhausted")
	ErrOscillation      = errors.New("skillstate: oscillation detected in patch sequence")
	ErrInvalidState     = errors.New("skillstate: invalid state parameter")
)

// BudgetState tracks finite execution resources for a skill run.
type BudgetState struct {
	MaxIterations           int `json:"max_iterations"`
	RemainingIterations      int `json:"remaining_iterations"`
	MaxVLMBudget            int `json:"max_vlm_budget"`
	RemainingVLMBudget       int `json:"remaining_vlm_budget"`
	MaxRepairAttempts       int `json:"max_repair_attempts"`
	RemainingRepairAttempts  int `json:"remaining_repair_attempts"`
}

// SkillState is the bounded, typed, model-visible working state projection.
type SkillState struct {
	RunID               string      `json:"run_id"`
	SkillID             string      `json:"skill_id"`
	GoalSummary         string      `json:"goal_summary"`
	CurrentPhase        string      `json:"current_phase"`
	Iteration           int         `json:"iteration"`
	Revision            int64       `json:"revision"`
	ActiveRegionIDs     []string    `json:"active_region_ids"`
	ActiveFindingIDs    []string    `json:"active_finding_ids"`
	ResolvedFindingIDs  []string    `json:"resolved_finding_ids"`
	ProtectedAxes       []string    `json:"protected_axes"`
	Budget              BudgetState `json:"budget"`
	LastEvidenceDigest  string      `json:"last_evidence_digest"`
	LastPatchDigest     string      `json:"last_patch_digest"`
	OscillationFlags    []string    `json:"oscillation_flags,omitempty"`
	PatchHistoryDigests []string    `json:"patch_history_digests,omitempty"`
	UpdatedAt           time.Time   `json:"updated_at"`
}

// NewSkillState creates a canonical bounded skill state.
func NewSkillState(runID, skillID, goalSummary string, budget BudgetState) *SkillState {
	return &SkillState{
		RunID:               runID,
		SkillID:             skillID,
		GoalSummary:         goalSummary,
		CurrentPhase:        "INIT",
		Iteration:           0,
		Revision:            1,
		ActiveRegionIDs:     make([]string, 0),
		ActiveFindingIDs:    make([]string, 0),
		ResolvedFindingIDs:  make([]string, 0),
		ProtectedAxes:       []string{"accessibility", "layout_stability"},
		Budget:              budget,
		LastEvidenceDigest:  "",
		LastPatchDigest:     "00000000",
		OscillationFlags:    make([]string, 0),
		PatchHistoryDigests: make([]string, 0),
		UpdatedAt:           time.Now(),
	}
}

// Clone creates a deep copy of the SkillState.
func (s *SkillState) Clone() *SkillState {
	copied := *s
	copied.ActiveRegionIDs = append([]string(nil), s.ActiveRegionIDs...)
	copied.ActiveFindingIDs = append([]string(nil), s.ActiveFindingIDs...)
	copied.ResolvedFindingIDs = append([]string(nil), s.ResolvedFindingIDs...)
	copied.ProtectedAxes = append([]string(nil), s.ProtectedAxes...)
	copied.OscillationFlags = append([]string(nil), s.OscillationFlags...)
	copied.PatchHistoryDigests = append([]string(nil), s.PatchHistoryDigests...)
	return &copied
}
