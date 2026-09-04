package skillstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// StatePatch encapsulates a model-proposed atomic state transition.
type StatePatch struct {
	ExpectedRevision        int64    `json:"expected_revision"`
	ExpectedLastPatchDigest string   `json:"expected_last_patch_digest,omitempty"`
	PhaseUpdate             string   `json:"phase_update,omitempty"`
	AddActiveRegions        []string `json:"add_active_regions,omitempty"`
	RemoveActiveRegions     []string `json:"remove_active_regions,omitempty"`
	AddFindings             []string `json:"add_findings,omitempty"`
	ResolveFindings         []string `json:"resolve_findings,omitempty"`
	NewEvidenceDigest       string   `json:"new_evidence_digest,omitempty"`
	DeductIterations        int      `json:"deduct_iterations,omitempty"`
	DeductVLMBudget         int      `json:"deduct_vlm_budget,omitempty"`
	DeductRepairAttempts    int      `json:"deduct_repair_attempts,omitempty"`
}

type semanticPatchPayload struct {
	PhaseUpdate          string   `json:"phase_update,omitempty"`
	AddActiveRegions     []string `json:"add_active_regions,omitempty"`
	RemoveActiveRegions  []string `json:"remove_active_regions,omitempty"`
	AddFindings          []string `json:"add_findings,omitempty"`
	ResolveFindings      []string `json:"resolve_findings,omitempty"`
	NewEvidenceDigest    string   `json:"new_evidence_digest,omitempty"`
	DeductIterations     int      `json:"deduct_iterations,omitempty"`
	DeductVLMBudget      int      `json:"deduct_vlm_budget,omitempty"`
	DeductRepairAttempts int      `json:"deduct_repair_attempts,omitempty"`
}

// ComputeDigest computes a deterministic hash of the semantic patch operations.
func (p *StatePatch) ComputeDigest() string {
	payload := semanticPatchPayload{
		PhaseUpdate:          p.PhaseUpdate,
		AddActiveRegions:     p.AddActiveRegions,
		RemoveActiveRegions:  p.RemoveActiveRegions,
		AddFindings:          p.AddFindings,
		ResolveFindings:      p.ResolveFindings,
		NewEvidenceDigest:    p.NewEvidenceDigest,
		DeductIterations:     p.DeductIterations,
		DeductVLMBudget:      p.DeductVLMBudget,
		DeductRepairAttempts: p.DeductRepairAttempts,
	}
	data, _ := json.Marshal(payload)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}

// ApplyPatch executes atomic CAS mutation on SkillState.
func ApplyPatch(s *SkillState, patch StatePatch) (*SkillState, error) {
	if s == nil {
		return nil, ErrInvalidState
	}

	// 1. CAS Revision Check
	if patch.ExpectedRevision != s.Revision {
		return nil, fmt.Errorf("%w: expected revision %d, current state is %d",
			ErrStalePatch, patch.ExpectedRevision, s.Revision)
	}

	// 2. Digest Chain Check (if specified)
	if patch.ExpectedLastPatchDigest != "" && patch.ExpectedLastPatchDigest != s.LastPatchDigest {
		return nil, fmt.Errorf("%w: expected digest %s, state has %s",
			ErrDigestMismatch, patch.ExpectedLastPatchDigest, s.LastPatchDigest)
	}

	// 3. Budget Check
	if patch.DeductIterations > 0 && s.Budget.RemainingIterations < patch.DeductIterations {
		return nil, fmt.Errorf("%w: iterations budget exceeded", ErrBudgetExhausted)
	}
	if patch.DeductVLMBudget > 0 && s.Budget.RemainingVLMBudget < patch.DeductVLMBudget {
		return nil, fmt.Errorf("%w: VLM budget exceeded", ErrBudgetExhausted)
	}
	if patch.DeductRepairAttempts > 0 && s.Budget.RemainingRepairAttempts < patch.DeductRepairAttempts {
		return nil, fmt.Errorf("%w: repair attempts budget exceeded", ErrBudgetExhausted)
	}

	patchDigest := patch.ComputeDigest()

	// 4. Oscillation Detection
	// If this exact patch digest was applied recently (within last 4 steps), flag oscillation
	oscillationDetected := false
	windowSize := 4
	startIdx := len(s.PatchHistoryDigests) - windowSize
	if startIdx < 0 {
		startIdx = 0
	}
	for _, prevDigest := range s.PatchHistoryDigests[startIdx:] {
		if prevDigest == patchDigest {
			oscillationDetected = true
			break
		}
	}

	// 5. Mutate state clone
	next := s.Clone()
	next.Revision = s.Revision + 1
	next.Iteration = s.Iteration + patch.DeductIterations
	next.LastPatchDigest = patchDigest
	next.PatchHistoryDigests = append(next.PatchHistoryDigests, patchDigest)
	if len(next.PatchHistoryDigests) > 20 {
		next.PatchHistoryDigests = next.PatchHistoryDigests[len(next.PatchHistoryDigests)-20:]
	}
	next.UpdatedAt = time.Now()

	if patch.PhaseUpdate != "" {
		next.CurrentPhase = patch.PhaseUpdate
	}
	if patch.NewEvidenceDigest != "" {
		next.LastEvidenceDigest = patch.NewEvidenceDigest
	}

	// Deduct budgets
	if patch.DeductIterations > 0 {
		next.Budget.RemainingIterations -= patch.DeductIterations
	}
	if patch.DeductVLMBudget > 0 {
		next.Budget.RemainingVLMBudget -= patch.DeductVLMBudget
	}
	if patch.DeductRepairAttempts > 0 {
		next.Budget.RemainingRepairAttempts -= patch.DeductRepairAttempts
	}

	// Update active regions
	regionSet := make(map[string]bool)
	for _, r := range next.ActiveRegionIDs {
		regionSet[r] = true
	}
	for _, r := range patch.AddActiveRegions {
		regionSet[r] = true
	}
	for _, r := range patch.RemoveActiveRegions {
		delete(regionSet, r)
	}
	next.ActiveRegionIDs = make([]string, 0, len(regionSet))
	for r := range regionSet {
		next.ActiveRegionIDs = append(next.ActiveRegionIDs, r)
	}

	// Update findings
	findingSet := make(map[string]bool)
	for _, f := range next.ActiveFindingIDs {
		findingSet[f] = true
	}
	for _, f := range patch.AddFindings {
		findingSet[f] = true
	}
	for _, f := range patch.ResolveFindings {
		delete(findingSet, f)
		next.ResolvedFindingIDs = append(next.ResolvedFindingIDs, f)
	}
	next.ActiveFindingIDs = make([]string, 0, len(findingSet))
	for f := range findingSet {
		next.ActiveFindingIDs = append(next.ActiveFindingIDs, f)
	}

	if oscillationDetected {
		flag := fmt.Sprintf("OSCILLATION_PATCH_%s_AT_REV_%d", patchDigest, next.Revision)
		next.OscillationFlags = append(next.OscillationFlags, flag)
	}

	return next, nil
}
