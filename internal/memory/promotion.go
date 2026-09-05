package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/sideeffect"
)

var (
	ErrPromotionUnauthorized = errors.New("memory promotion is not authorized")
	ErrPromotionPoisoned     = errors.New("memory promotion candidate failed poisoning policy")
	ErrPromotionNotFound     = errors.New("memory promotion record not found")
)

type PromotionStatus string

const (
	PromotionActive     PromotionStatus = "ACTIVE"
	PromotionRolledBack PromotionStatus = "ROLLED_BACK"
	PromotionRevoked    PromotionStatus = "REVOKED_SOURCE"
)

// PromotionRequest is the only legal project-private -> global visibility path.
// The candidate is a sanitized/generalized rule, never a copy of private evidence.
type PromotionRequest struct {
	SourceNamespace           Namespace  `json:"source_namespace"`
	TargetNamespace           Namespace  `json:"target_namespace"`
	SourceAtomIDs             []string   `json:"source_atom_ids"`
	IndependentEvidenceDigests []string  `json:"independent_evidence_digests"`
	Candidate                 MemoryAtom `json:"candidate"`
	VerifierID                string     `json:"verifier_id"`
	Rationale                 string     `json:"rationale"`
}

type PromotionRecord struct {
	PromotionID                string          `json:"promotion_id"`
	OperationLogicalID         string          `json:"operation_logical_id"`
	CandidateAtomID            string          `json:"candidate_atom_id"`
	SourceNamespace            Namespace       `json:"source_namespace"`
	TargetNamespace            Namespace       `json:"target_namespace"`
	SourceAtomIDs              []string        `json:"source_atom_ids"`
	IndependentEvidenceDigests []string        `json:"independent_evidence_digests"`
	VerifierID                 string          `json:"verifier_id"`
	Rationale                  string          `json:"rationale"`
	DecisionDigest             string          `json:"decision_digest"`
	Status                     PromotionStatus `json:"status"`
	StatusReason               string          `json:"status_reason,omitempty"`
	CreatedAt                  time.Time       `json:"created_at"`
	UpdatedAt                  time.Time       `json:"updated_at"`
}

type PromotionRollbackRequest struct {
	PromotionID string `json:"promotion_id"`
	ReviewerID  string `json:"reviewer_id"`
	Reason      string `json:"reason"`
}

// PromotionPayloadDigest returns the payload digest callers must bind to the
// side-effect operation. It excludes mutable timestamps/provenance and sorts all
// evidence IDs, making retry identity deterministic.
func PromotionPayloadDigest(req PromotionRequest) string {
	sources := uniqueSorted(req.SourceAtomIDs)
	evidence := uniqueSorted(req.IndependentEvidenceDigests)
	rule, _ := req.Candidate.Data.(DesignRuleAtom)
	payload := strings.Join([]string{
		req.SourceNamespace.String(),
		req.TargetNamespace.String(),
		strings.Join(sources, ","),
		strings.Join(evidence, ","),
		req.Candidate.ID,
		string(req.Candidate.Kind),
		fmt.Sprintf("%.6f", req.Candidate.Confidence),
		rule.RuleID,
		rule.Axis,
		rule.Category,
		rule.Title,
		rule.Description,
		fmt.Sprintf("hard=%t", rule.HardConstraint),
		fmt.Sprintf("weight=%.6f", rule.Weight),
		rule.Version,
		strings.Join(uniqueSorted(req.Candidate.Tags), ","),
		strings.TrimSpace(req.VerifierID),
		strings.TrimSpace(req.Rationale),
	}, "\x00")
	return sideeffect.DigestBytes([]byte(payload))
}

func PromotionRollbackPayloadDigest(req PromotionRollbackRequest) string {
	return sideeffect.DigestBytes([]byte(strings.Join([]string{strings.TrimSpace(req.PromotionID), strings.TrimSpace(req.ReviewerID), strings.TrimSpace(req.Reason)}, "\x00")))
}

func (s *EpMemoryStore) Promote(ctx context.Context, op sideeffect.Operation, req PromotionRequest) (PromotionRecord, sideeffect.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return PromotionRecord{}, sideeffect.Receipt{}, err
	}
	if err := op.Validate(); err != nil {
		return PromotionRecord{}, sideeffect.Receipt{}, err
	}
	if op.PayloadDigest != PromotionPayloadDigest(req) {
		return PromotionRecord{}, sideeffect.Receipt{}, fmt.Errorf("%w: promotion operation payload does not bind candidate", sideeffect.ErrOperationConflict)
	}
	logicalID, _ := op.LogicalID()
	opID, _ := op.ID()

	s.mu.Lock()
	defer s.mu.Unlock()
	if prior, ok := s.committedOperations[logicalID]; ok {
		if prior.PayloadDigest != op.PayloadDigest || prior.OperationID != opID {
			return PromotionRecord{}, sideeffect.Receipt{}, sideeffect.ErrOperationConflict
		}
		for _, rec := range s.promotions {
			if rec.OperationLogicalID == logicalID {
				reused := prior
				reused.Applied = false
				reused.Reused = true
				return *rec, reused, nil
			}
		}
		return PromotionRecord{}, sideeffect.Receipt{}, fmt.Errorf("%w: receipt exists without promotion record", ErrPromotionUnauthorized)
	}

	prepared, decisionDigest, err := s.validatePromotionLocked(req)
	if err != nil {
		return PromotionRecord{}, sideeffect.Receipt{}, err
	}
	if existing, ok := s.atoms[prepared.Candidate.ID]; ok && existing.Status == StatusActive {
		return PromotionRecord{}, sideeffect.Receipt{}, fmt.Errorf("%w: candidate atom %s already active", ErrAtomAlreadyExists, prepared.Candidate.ID)
	}

	now := time.Now()
	candidate := prepared.Candidate
	candidate.Namespace = prepared.TargetNamespace
	candidate.CreatedAt = now
	candidate.UpdatedAt = now
	candidate.Provenance.SourceNamespace = prepared.SourceNamespace.String()
	candidate.Provenance.SourceAtomIDs = uniqueSorted(prepared.SourceAtomIDs)
	candidate.Provenance.DecisionDigest = decisionDigest
	candidate.Provenance.VerifierID = strings.TrimSpace(prepared.VerifierID)
	candidate.Provenance.ProjectScope = "global"
	candidate.Provenance.EvidenceDigest = decisionDigest
	candidate.Provenance.Timestamp = now
	candidate.Provenance.Outcome = "PROMOTED"
	if strings.TrimSpace(candidate.Provenance.RunID) == "" {
		candidate.Provenance.RunID = op.RunID
	}
	if strings.TrimSpace(candidate.Provenance.Renderer) == "" {
		candidate.Provenance.Renderer = "memory-promotion"
	}

	s.atoms[candidate.ID] = &StoredAtom{Atom: candidate, Status: StatusActive, UpdatedAt: now}
	for _, sourceID := range uniqueSorted(prepared.SourceAtomIDs) {
		s.addEdgeLocked(MemoryEdge{
			FromID: candidate.ID,
			ToID: sourceID,
			Relation: RelGeneralizes,
			Weight: 1,
			Provenance: candidate.Provenance,
			CreatedAt: now,
		}, now)
	}

	promotionID := "promotion:" + candidate.ID + ":" + decisionDigest[len("sha256:"):len("sha256:")+16]
	record := &PromotionRecord{
		PromotionID: promotionID,
		OperationLogicalID: logicalID,
		CandidateAtomID: candidate.ID,
		SourceNamespace: prepared.SourceNamespace,
		TargetNamespace: prepared.TargetNamespace,
		SourceAtomIDs: uniqueSorted(prepared.SourceAtomIDs),
		IndependentEvidenceDigests: uniqueSorted(prepared.IndependentEvidenceDigests),
		VerifierID: strings.TrimSpace(prepared.VerifierID),
		Rationale: strings.TrimSpace(prepared.Rationale),
		DecisionDigest: decisionDigest,
		Status: PromotionActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.promotions[promotionID] = record

	receipt := sideeffect.Receipt{
		LogicalID: logicalID,
		OperationID: opID,
		Kind: op.Kind,
		PayloadDigest: op.PayloadDigest,
		ResultDigest: decisionDigest,
		RetryClass: op.RetryClass,
		Applied: true,
	}
	s.committedOperations[logicalID] = receipt
	return *record, receipt, nil
}

func (s *EpMemoryStore) validatePromotionLocked(req PromotionRequest) (PromotionRequest, string, error) {
	if !req.SourceNamespace.IsValid() || !req.SourceNamespace.IsProjectPrivate() {
		return req, "", fmt.Errorf("%w: source must be project-private", ErrPromotionUnauthorized)
	}
	if !req.TargetNamespace.Equal(NewGlobalDesignNamespace()) {
		return req, "", fmt.Errorf("%w: target must be %s", ErrPromotionUnauthorized, NewGlobalDesignNamespace())
	}
	if req.Candidate.Kind != NodeDesignRule || !req.Candidate.Namespace.Equal(req.TargetNamespace) {
		return req, "", fmt.Errorf("%w: candidate must be a global DesignRule", ErrPromotionUnauthorized)
	}
	if req.Candidate.Confidence < 0.9 {
		return req, "", fmt.Errorf("%w: candidate confidence %.3f < 0.9", ErrPromotionPoisoned, req.Candidate.Confidence)
	}
	if strings.TrimSpace(req.VerifierID) == "" || strings.TrimSpace(req.Rationale) == "" {
		return req, "", fmt.Errorf("%w: verifier and rationale are required", ErrPromotionUnauthorized)
	}
	sourceIDs := uniqueSorted(req.SourceAtomIDs)
	if len(sourceIDs) < 2 {
		return req, "", fmt.Errorf("%w: at least two source atoms are required", ErrPromotionPoisoned)
	}
	evidenceDigests := uniqueSorted(req.IndependentEvidenceDigests)
	if len(evidenceDigests) < 2 {
		return req, "", fmt.Errorf("%w: at least two independent evidence digests are required", ErrPromotionPoisoned)
	}
	evidenceSet := make(map[string]struct{}, len(evidenceDigests))
	for _, digest := range evidenceDigests {
		if !strings.HasPrefix(digest, "sha256:") {
			return req, "", fmt.Errorf("%w: malformed evidence digest %q", ErrPromotionPoisoned, digest)
		}
		evidenceSet[digest] = struct{}{}
	}
	for _, sourceID := range sourceIDs {
		stored, ok := s.atoms[sourceID]
		if !ok || stored.Status != StatusActive {
			return req, "", fmt.Errorf("%w: source atom %s is not active", ErrPromotionUnauthorized, sourceID)
		}
		if !stored.Atom.Namespace.IsProjectPrivate() || stored.Atom.Namespace.ProjectID() != req.SourceNamespace.ProjectID() {
			return req, "", fmt.Errorf("%w: source atom %s belongs to %s", ErrPromotionUnauthorized, sourceID, stored.Atom.Namespace)
		}
		if stored.Atom.Confidence < 0.8 {
			return req, "", fmt.Errorf("%w: source atom %s confidence %.3f < 0.8", ErrPromotionPoisoned, sourceID, stored.Atom.Confidence)
		}
		if len(stored.Conflicts) > 0 {
			return req, "", fmt.Errorf("%w: source atom %s has active conflicts", ErrPromotionPoisoned, sourceID)
		}
		if _, ok := evidenceSet[stored.Atom.Provenance.EvidenceDigest]; !ok {
			return req, "", fmt.Errorf("%w: source atom %s evidence digest is not in independent set", ErrPromotionPoisoned, sourceID)
		}
	}
	if err := validateGeneralizedCandidate(req.Candidate, req.SourceNamespace.ProjectID()); err != nil {
		return req, "", err
	}
	req.SourceAtomIDs = sourceIDs
	req.IndependentEvidenceDigests = evidenceDigests
	return req, PromotionPayloadDigest(req), nil
}

func validateGeneralizedCandidate(candidate MemoryAtom, projectID string) error {
	rule, ok := candidate.Data.(DesignRuleAtom)
	if !ok {
		return fmt.Errorf("%w: candidate payload must be DesignRuleAtom", ErrPromotionPoisoned)
	}
	text := strings.ToLower(strings.Join([]string{
		rule.RuleID,
		rule.Axis,
		rule.Category,
		rule.Title,
		rule.Description,
		strings.Join(candidate.Tags, " "),
	}, " "))
	projectID = strings.ToLower(strings.TrimSpace(projectID))
	if projectID != "" && strings.Contains(text, projectID) {
		return fmt.Errorf("%w: generalized rule contains source project identifier", ErrPromotionPoisoned)
	}
	for _, marker := range []string{"password", "credential", "api key", "secret", "private:", "customer-specific", "user-specific"} {
		if strings.Contains(text, marker) {
			return fmt.Errorf("%w: generalized rule contains private marker %q", ErrPromotionPoisoned, marker)
		}
	}
	if strings.TrimSpace(rule.RuleID) == "" || strings.TrimSpace(rule.Title) == "" || strings.TrimSpace(rule.Description) == "" {
		return fmt.Errorf("%w: generalized rule lacks required semantic fields", ErrPromotionPoisoned)
	}
	return nil
}

func (s *EpMemoryStore) RollbackPromotion(ctx context.Context, op sideeffect.Operation, req PromotionRollbackRequest) (PromotionRecord, sideeffect.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return PromotionRecord{}, sideeffect.Receipt{}, err
	}
	if err := op.Validate(); err != nil {
		return PromotionRecord{}, sideeffect.Receipt{}, err
	}
	if op.PayloadDigest != PromotionRollbackPayloadDigest(req) {
		return PromotionRecord{}, sideeffect.Receipt{}, sideeffect.ErrOperationConflict
	}
	if strings.TrimSpace(req.ReviewerID) == "" || strings.TrimSpace(req.Reason) == "" {
		return PromotionRecord{}, sideeffect.Receipt{}, fmt.Errorf("%w: rollback reviewer and reason are required", ErrPromotionUnauthorized)
	}
	logicalID, _ := op.LogicalID()
	opID, _ := op.ID()

	s.mu.Lock()
	defer s.mu.Unlock()
	if prior, ok := s.committedOperations[logicalID]; ok {
		if prior.PayloadDigest != op.PayloadDigest || prior.OperationID != opID {
			return PromotionRecord{}, sideeffect.Receipt{}, sideeffect.ErrOperationConflict
		}
		rec, ok := s.promotions[req.PromotionID]
		if !ok {
			return PromotionRecord{}, sideeffect.Receipt{}, ErrPromotionNotFound
		}
		reused := prior
		reused.Applied = false
		reused.Reused = true
		return *rec, reused, nil
	}

	rec, ok := s.promotions[req.PromotionID]
	if !ok {
		return PromotionRecord{}, sideeffect.Receipt{}, ErrPromotionNotFound
	}
	if rec.Status == PromotionActive {
		s.retractPromotedAtomLocked(rec.CandidateAtomID, "promotion rollback: "+req.Reason)
		rec.Status = PromotionRolledBack
		rec.StatusReason = req.Reason
		rec.UpdatedAt = time.Now()
	}
	receipt := sideeffect.Receipt{
		LogicalID: logicalID,
		OperationID: opID,
		Kind: op.Kind,
		PayloadDigest: op.PayloadDigest,
		ResultDigest: rec.DecisionDigest,
		RetryClass: op.RetryClass,
		Applied: true,
	}
	s.committedOperations[logicalID] = receipt
	return *rec, receipt, nil
}

func (s *EpMemoryStore) PromotionRecord(id string) (PromotionRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.promotions[id]
	if !ok {
		return PromotionRecord{}, false
	}
	return *rec, true
}

func (s *EpMemoryStore) revokePromotionsForSourceLocked(sourceAtomID, reason string, prov ProvenanceRecord) {
	for _, rec := range s.promotions {
		if rec.Status != PromotionActive || !containsString(rec.SourceAtomIDs, sourceAtomID) {
			continue
		}
		s.retractPromotedAtomLocked(rec.CandidateAtomID, reason)
		rec.Status = PromotionRevoked
		rec.StatusReason = reason
		rec.UpdatedAt = time.Now()
	}
}

func (s *EpMemoryStore) retractPromotedAtomLocked(atomID, reason string) {
	if stored, ok := s.atoms[atomID]; ok && stored.Status == StatusActive {
		stored.Status = StatusRetracted
		stored.RetractReason = reason
		stored.UpdatedAt = time.Now()
	}
}

func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
