package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/sideeffect"
)

var (
	ErrAtomNotFound      = errors.New("memory atom not found")
	ErrConflictExists    = errors.New("memory conflict detected")
	ErrAtomAlreadyExists = errors.New("memory atom with id already exists")
)

type AtomStatus string

const (
	StatusActive     AtomStatus = "ACTIVE"
	StatusRetracted  AtomStatus = "RETRACTED"
	StatusSuperseded AtomStatus = "SUPERSEDED"
)

type StoredAtom struct {
	Atom          MemoryAtom `json:"atom"`
	Status        AtomStatus `json:"status"`
	RetractReason string     `json:"retract_reason,omitempty"`
	SupersededBy  string     `json:"superseded_by,omitempty"`
	Conflicts     []string   `json:"conflicts,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ConflictRecord struct {
	PrimaryAtomID     string    `json:"primary_atom_id"`
	ConflictingAtomID string    `json:"conflicting_atom_id"`
	Reason            string    `json:"reason"`
	DetectedAt        time.Time `json:"detected_at"`
}

type QueryRequest struct {
	Namespace Namespace
	Kind      NodeKind
	Tags      []string
	MinConf   float64
	Limit     int
}

type QueryResult struct {
	Atoms []MemoryAtom `json:"atoms"`
	Total int          `json:"total"`
}

// Store is deliberately scope-explicit for every read and mutation that can
// reveal or change an existing atom. Commit authorization is carried inside the
// AdmissionBundle so durable/retry adapters cannot drop the source scope.
type Store interface {
	Commit(ctx context.Context, bundle AdmissionBundle) error
	Retract(ctx context.Context, scope Namespace, atomID string, reason string, prov ProvenanceRecord) error
	Supersede(ctx context.Context, scope Namespace, oldAtomID string, newAtom MemoryAtom, prov ProvenanceRecord) error
	GetAtom(ctx context.Context, scope Namespace, id string) (*MemoryAtom, error)
	GetEdges(ctx context.Context, scope Namespace, atomID string) ([]MemoryEdge, error)
	Query(ctx context.Context, q QueryRequest) (*QueryResult, error)
	RetrieveContextPack(ctx context.Context, req ContextPackRequest) (*ContextPack, error)
}

type EpMemoryStore struct {
	mu                  sync.RWMutex
	atoms               map[string]*StoredAtom
	edgesFrom           map[string][]MemoryEdge
	edgesTo             map[string][]MemoryEdge
	edgeKeys            map[string]struct{}
	conflicts           []ConflictRecord
	conflictKeys        map[string]struct{}
	committedOperations map[string]sideeffect.Receipt
	promotions          map[string]*PromotionRecord
}

func NewEpMemoryStore() *EpMemoryStore {
	return &EpMemoryStore{
		atoms:               make(map[string]*StoredAtom),
		edgesFrom:           make(map[string][]MemoryEdge),
		edgesTo:             make(map[string][]MemoryEdge),
		edgeKeys:            make(map[string]struct{}),
		conflicts:           make([]ConflictRecord, 0),
		conflictKeys:        make(map[string]struct{}),
		committedOperations: make(map[string]sideeffect.Receipt),
		promotions:          make(map[string]*PromotionRecord),
	}
}

func (s *EpMemoryStore) Commit(ctx context.Context, bundle AdmissionBundle) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateAdmissionBundle(bundle); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyBundleLocked(bundle)
	return nil
}

func (s *EpMemoryStore) CommitOnce(ctx context.Context, op sideeffect.Operation, bundle AdmissionBundle) (sideeffect.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return sideeffect.Receipt{}, err
	}
	if err := op.Validate(); err != nil {
		return sideeffect.Receipt{}, err
	}
	if err := validateAdmissionBundle(bundle); err != nil {
		return sideeffect.Receipt{}, err
	}
	logicalID, _ := op.LogicalID()
	opID, _ := op.ID()

	s.mu.Lock()
	defer s.mu.Unlock()
	if prior, ok := s.committedOperations[logicalID]; ok {
		if prior.PayloadDigest != op.PayloadDigest || prior.OperationID != opID {
			return sideeffect.Receipt{}, sideeffect.ErrOperationConflict
		}
		prior.Reused = true
		prior.Applied = false
		return prior, nil
	}

	s.applyBundleLocked(bundle)
	receipt := sideeffect.Receipt{
		LogicalID:     logicalID,
		OperationID:   opID,
		Kind:          op.Kind,
		PayloadDigest: op.PayloadDigest,
		ResultDigest:  memoryBundleResultDigest(bundle),
		RetryClass:    op.RetryClass,
		Applied:       true,
	}
	s.committedOperations[logicalID] = receipt
	return receipt, nil
}

func validateAdmissionBundle(bundle AdmissionBundle) error {
	if len(bundle.Atoms) == 0 && len(bundle.Edges) == 0 {
		return nil
	}
	if !bundle.SourceNamespace.IsValid() {
		return fmt.Errorf("%w: admission bundle source namespace is required", ErrScopeRequired)
	}
	mapper := NewAdmissionMapper(nil)
	for _, atom := range bundle.Atoms {
		if strings.TrimSpace(atom.ID) == "" || !atom.Namespace.IsValid() {
			return fmt.Errorf("%w: invalid atom id/namespace", ErrInvalidAtomData)
		}
		if !CanAdmitOrdinary(bundle.SourceNamespace, atom.Namespace) {
			return fmt.Errorf("%w: bundle source %s cannot write atom %s in %s", ErrAdmissionRoute, bundle.SourceNamespace, atom.ID, atom.Namespace)
		}
		if err := validateStoredProvenance(mapper, bundle.SourceNamespace, atom.Namespace, atom.Provenance); err != nil {
			return fmt.Errorf("atom %s: %w", atom.ID, err)
		}
		if atom.Confidence < 0 || atom.Confidence > 1 {
			return fmt.Errorf("%w: atom %s confidence outside [0,1]", ErrInvalidAtomData, atom.ID)
		}
	}
	for _, edge := range bundle.Edges {
		if strings.TrimSpace(edge.FromID) == "" || strings.TrimSpace(edge.ToID) == "" {
			return fmt.Errorf("%w: edge endpoints are required", ErrInvalidAtomData)
		}
		if err := validateStoredProvenance(mapper, bundle.SourceNamespace, bundle.SourceNamespace, edge.Provenance); err != nil {
			return fmt.Errorf("edge %s->%s: %w", edge.FromID, edge.ToID, err)
		}
	}
	return nil
}

func validateStoredProvenance(mapper *AdmissionMapper, source, target Namespace, prov ProvenanceRecord) error {
	if err := mapper.ValidateProvenance(prov); err != nil {
		return err
	}
	if strings.TrimSpace(prov.SourceNamespace) != source.String() {
		return fmt.Errorf("%w: provenance source %q != bundle source %q", ErrAdmissionRoute, prov.SourceNamespace, source.String())
	}
	if source.IsProjectPrivate() {
		if prov.ProjectScope != source.ProjectID() || (target.IsProjectPrivate() && target.ProjectID() != source.ProjectID()) {
			return fmt.Errorf("%w: project provenance %q does not match source project %q", ErrAdmissionRoute, prov.ProjectScope, source.ProjectID())
		}
	} else if source.IsGlobal() && prov.ProjectScope != "global" {
		return fmt.Errorf("%w: global provenance must use project_scope=global", ErrAdmissionRoute)
	}
	return nil
}

func (s *EpMemoryStore) OperationReceipt(logicalID string) (sideeffect.Receipt, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.committedOperations[logicalID]
	return r, ok
}

func (s *EpMemoryStore) applyBundleLocked(bundle AdmissionBundle) {
	now := time.Now()
	for _, atom := range bundle.Atoms {
		existing, found := s.atoms[atom.ID]
		if found && existing.Status == StatusActive {
			existing.Atom = atom
			existing.UpdatedAt = now
			continue
		}
		s.atoms[atom.ID] = &StoredAtom{Atom: atom, Status: StatusActive, UpdatedAt: now}
	}
	for _, edge := range bundle.Edges {
		s.addEdgeLocked(edge, now)
	}
}

func (s *EpMemoryStore) addEdgeLocked(edge MemoryEdge, now time.Time) {
	key := memoryEdgeKey(edge)
	if _, exists := s.edgeKeys[key]; exists {
		return
	}
	s.edgeKeys[key] = struct{}{}
	s.edgesFrom[edge.FromID] = append(s.edgesFrom[edge.FromID], edge)
	s.edgesTo[edge.ToID] = append(s.edgesTo[edge.ToID], edge)
	if edge.Relation != RelRefutes && edge.Relation != RelCounterexampleTo {
		return
	}
	conflictKey := edge.ToID + "\x00" + edge.FromID + "\x00" + string(edge.Relation) + "\x00" + edge.Provenance.RunID + "\x00" + edge.Provenance.EvidenceDigest
	if _, exists := s.conflictKeys[conflictKey]; !exists {
		s.conflictKeys[conflictKey] = struct{}{}
		s.conflicts = append(s.conflicts, ConflictRecord{
			PrimaryAtomID:     edge.ToID,
			ConflictingAtomID: edge.FromID,
			Reason:            fmt.Sprintf("Relation %s asserted from %s", edge.Relation, edge.FromID),
			DetectedAt:        now,
		})
	}
	if target, ok := s.atoms[edge.ToID]; ok && !containsString(target.Conflicts, edge.FromID) {
		target.Conflicts = append(target.Conflicts, edge.FromID)
		sort.Strings(target.Conflicts)
	}
}

func memoryEdgeKey(edge MemoryEdge) string {
	return edge.FromID + "\x00" + edge.ToID + "\x00" + string(edge.Relation) + "\x00" + strconv.FormatFloat(edge.Weight, 'g', -1, 64) + "\x00" + edge.Provenance.RunID + "\x00" + edge.Provenance.EvidenceDigest
}

func memoryBundleResultDigest(bundle AdmissionBundle) string {
	parts := []string{"source:" + bundle.SourceNamespace.String()}
	for _, atom := range bundle.Atoms {
		parts = append(parts, "a:"+atom.ID+":"+atom.Namespace.String()+":"+atom.Provenance.SourceNamespace+":"+atom.Provenance.EvidenceDigest)
	}
	for _, edge := range bundle.Edges {
		parts = append(parts, "e:"+memoryEdgeKey(edge)+":"+edge.Provenance.SourceNamespace)
	}
	sort.Strings(parts)
	return sideeffect.DigestBytes([]byte(strings.Join(parts, "\n")))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *EpMemoryStore) Retract(ctx context.Context, scope Namespace, atomID string, reason string, prov ProvenanceRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !scope.IsValid() {
		return ErrScopeRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	target, ok := s.atoms[atomID]
	if !ok || target.Status != StatusActive {
		return fmt.Errorf("%w: %s", ErrAtomNotFound, atomID)
	}
	if !CanMutate(scope, target.Atom.Namespace) {
		return fmt.Errorf("%w: %s cannot retract %s in %s", ErrUnauthorizedAccess, scope, atomID, target.Atom.Namespace)
	}
	prov = bindMutationProvenance(scope, prov)
	target.Status = StatusRetracted
	target.RetractReason = reason
	target.Atom.Provenance = prov
	target.UpdatedAt = time.Now()
	s.revokePromotionsForSourceLocked(atomID, "source retracted: "+reason, prov)
	return nil
}

func (s *EpMemoryStore) Supersede(ctx context.Context, scope Namespace, oldAtomID string, newAtom MemoryAtom, prov ProvenanceRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !scope.IsValid() {
		return ErrScopeRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.atoms[oldAtomID]
	if !ok || old.Status != StatusActive {
		return fmt.Errorf("%w: %s", ErrAtomNotFound, oldAtomID)
	}
	if !CanMutate(scope, old.Atom.Namespace) || !CanMutate(scope, newAtom.Namespace) {
		return fmt.Errorf("%w: %s cannot supersede %s across namespace boundary", ErrUnauthorizedAccess, scope, oldAtomID)
	}
	if !CanAdmitOrdinary(old.Atom.Namespace, newAtom.Namespace) {
		return fmt.Errorf("%w: supersede %s -> %s would widen scope", ErrAdmissionRoute, old.Atom.Namespace, newAtom.Namespace)
	}
	prov = bindMutationProvenance(scope, prov)
	newAtom.Provenance = prov
	now := time.Now()
	old.Status = StatusSuperseded
	old.SupersededBy = newAtom.ID
	old.UpdatedAt = now
	s.atoms[newAtom.ID] = &StoredAtom{Atom: newAtom, Status: StatusActive, UpdatedAt: now}
	s.addEdgeLocked(MemoryEdge{FromID: newAtom.ID, ToID: oldAtomID, Relation: RelGeneralizes, Weight: 1, Provenance: prov, CreatedAt: now}, now)
	s.revokePromotionsForSourceLocked(oldAtomID, "source superseded by "+newAtom.ID, prov)
	return nil
}

func bindMutationProvenance(scope Namespace, prov ProvenanceRecord) ProvenanceRecord {
	prov.SourceNamespace = scope.String()
	if scope.IsProjectPrivate() {
		prov.ProjectScope = scope.ProjectID()
	} else if scope.IsGlobal() {
		prov.ProjectScope = "global"
	}
	if prov.Timestamp.IsZero() {
		prov.Timestamp = time.Now()
	}
	return prov
}

func (s *EpMemoryStore) GetAtom(ctx context.Context, scope Namespace, id string) (*MemoryAtom, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !scope.IsValid() {
		return nil, ErrScopeRequired
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	stored, ok := s.atoms[id]
	if !ok || stored.Status != StatusActive {
		return nil, fmt.Errorf("%w: %s", ErrAtomNotFound, id)
	}
	if !CanAccess(scope, stored.Atom.Namespace) {
		return nil, fmt.Errorf("%w: %s cannot read %s", ErrUnauthorizedAccess, scope, id)
	}
	copy := stored.Atom
	return &copy, nil
}

func (s *EpMemoryStore) GetEdges(ctx context.Context, scope Namespace, atomID string) ([]MemoryEdge, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !scope.IsValid() {
		return nil, ErrScopeRequired
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	anchor, ok := s.atoms[atomID]
	if !ok || anchor.Status != StatusActive {
		return nil, fmt.Errorf("%w: %s", ErrAtomNotFound, atomID)
	}
	if !CanAccess(scope, anchor.Atom.Namespace) {
		return nil, fmt.Errorf("%w: %s cannot read edges for %s", ErrUnauthorizedAccess, scope, atomID)
	}
	out := make([]MemoryEdge, 0, len(s.edgesFrom[atomID]))
	for _, edge := range s.edgesFrom[atomID] {
		if s.edgeVisibleLocked(scope, edge) {
			out = append(out, edge)
		}
	}
	return out, nil
}

func (s *EpMemoryStore) edgeVisibleLocked(scope Namespace, edge MemoryEdge) bool {
	for _, id := range []string{edge.FromID, edge.ToID} {
		if stored, ok := s.atoms[id]; ok && stored.Status == StatusActive && !CanAccess(scope, stored.Atom.Namespace) {
			return false
		}
	}
	return true
}

func (s *EpMemoryStore) Query(ctx context.Context, q QueryRequest) (*QueryResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !q.Namespace.IsValid() {
		return nil, ErrScopeRequired
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var matches []MemoryAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive {
			continue
		}
		atom := stored.Atom
		if !CanAccess(q.Namespace, atom.Namespace) {
			continue
		}
		if q.Kind != "" && atom.Kind != q.Kind {
			continue
		}
		if q.MinConf > 0 && atom.Confidence < q.MinConf {
			continue
		}
		if len(q.Tags) > 0 {
			tagMap := make(map[string]bool)
			for _, t := range atom.Tags {
				tagMap[t] = true
			}
			matchAll := true
			for _, reqTag := range q.Tags {
				if !tagMap[reqTag] {
					matchAll = false
					break
				}
			}
			if !matchAll {
				continue
			}
		}
		matches = append(matches, atom)
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Confidence != matches[j].Confidence {
			return matches[i].Confidence > matches[j].Confidence
		}
		return matches[i].CreatedAt.After(matches[j].CreatedAt)
	})
	total := len(matches)
	if q.Limit > 0 && len(matches) > q.Limit {
		matches = matches[:q.Limit]
	}
	return &QueryResult{Atoms: matches, Total: total}, nil
}
