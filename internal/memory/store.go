package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
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

type Store interface {
	Commit(ctx context.Context, bundle AdmissionBundle) error
	Retract(ctx context.Context, atomID string, reason string, prov ProvenanceRecord) error
	Supersede(ctx context.Context, oldAtomID string, newAtom MemoryAtom, prov ProvenanceRecord) error
	GetAtom(ctx context.Context, id string) (*MemoryAtom, error)
	GetEdges(ctx context.Context, atomID string) ([]MemoryEdge, error)
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
	// keyed by sideeffect.Operation.LogicalID, not payload-bound OperationID;
	// this makes retry payload drift a hard conflict instead of a second commit.
	committedOperations map[string]sideeffect.Receipt
}

func NewEpMemoryStore() *EpMemoryStore {
	return &EpMemoryStore{
		atoms: make(map[string]*StoredAtom), edgesFrom: make(map[string][]MemoryEdge), edgesTo: make(map[string][]MemoryEdge),
		edgeKeys: make(map[string]struct{}), conflicts: make([]ConflictRecord, 0), conflictKeys: make(map[string]struct{}),
		committedOperations: make(map[string]sideeffect.Receipt),
	}
}

func (s *EpMemoryStore) Commit(ctx context.Context, bundle AdmissionBundle) error {
	if err := ctx.Err(); err != nil { return err }
	s.mu.Lock(); defer s.mu.Unlock()
	s.applyBundleLocked(bundle)
	return nil
}

func (s *EpMemoryStore) CommitOnce(ctx context.Context, op sideeffect.Operation, bundle AdmissionBundle) (sideeffect.Receipt, error) {
	if err := ctx.Err(); err != nil { return sideeffect.Receipt{}, err }
	if err := op.Validate(); err != nil { return sideeffect.Receipt{}, err }
	logicalID, _ := op.LogicalID()
	opID, _ := op.ID()

	s.mu.Lock()
	defer s.mu.Unlock()
	if prior, ok := s.committedOperations[logicalID]; ok {
		if prior.PayloadDigest != op.PayloadDigest || prior.OperationID != opID { return sideeffect.Receipt{}, sideeffect.ErrOperationConflict }
		prior.Reused = true
		prior.Applied = false
		return prior, nil
	}

	s.applyBundleLocked(bundle)
	receipt := sideeffect.Receipt{
		LogicalID: logicalID, OperationID: opID, Kind: op.Kind, PayloadDigest: op.PayloadDigest,
		ResultDigest: memoryBundleResultDigest(bundle), RetryClass: op.RetryClass, Applied: true,
	}
	s.committedOperations[logicalID] = receipt
	return receipt, nil
}

func (s *EpMemoryStore) OperationReceipt(logicalID string) (sideeffect.Receipt, bool) {
	s.mu.RLock(); defer s.mu.RUnlock(); r, ok := s.committedOperations[logicalID]; return r, ok
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
	for _, edge := range bundle.Edges { s.addEdgeLocked(edge, now) }
}

func (s *EpMemoryStore) addEdgeLocked(edge MemoryEdge, now time.Time) {
	key := memoryEdgeKey(edge)
	if _, exists := s.edgeKeys[key]; exists { return }
	s.edgeKeys[key] = struct{}{}
	s.edgesFrom[edge.FromID] = append(s.edgesFrom[edge.FromID], edge)
	s.edgesTo[edge.ToID] = append(s.edgesTo[edge.ToID], edge)
	if edge.Relation != RelRefutes && edge.Relation != RelCounterexampleTo { return }
	conflictKey := edge.ToID + "\x00" + edge.FromID + "\x00" + string(edge.Relation) + "\x00" + edge.Provenance.RunID + "\x00" + edge.Provenance.EvidenceDigest
	if _, exists := s.conflictKeys[conflictKey]; !exists {
		s.conflictKeys[conflictKey] = struct{}{}
		s.conflicts = append(s.conflicts, ConflictRecord{PrimaryAtomID: edge.ToID, ConflictingAtomID: edge.FromID, Reason: fmt.Sprintf("Relation %s asserted from %s", edge.Relation, edge.FromID), DetectedAt: now})
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
	parts := make([]string, 0, len(bundle.Atoms)+len(bundle.Edges))
	for _, atom := range bundle.Atoms { parts = append(parts, "a:"+atom.ID) }
	for _, edge := range bundle.Edges { parts = append(parts, "e:"+memoryEdgeKey(edge)) }
	sort.Strings(parts)
	return sideeffect.DigestBytes([]byte(fmt.Sprint(parts)))
}

func containsString(values []string, target string) bool {
	for _, value := range values { if value == target { return true } }
	return false
}

func (s *EpMemoryStore) Retract(ctx context.Context, atomID string, reason string, prov ProvenanceRecord) error {
	if err := ctx.Err(); err != nil { return err }
	s.mu.Lock(); defer s.mu.Unlock()
	target, ok := s.atoms[atomID]
	if !ok { return fmt.Errorf("%w: %s", ErrAtomNotFound, atomID) }
	target.Status = StatusRetracted; target.RetractReason = reason; target.Atom.Provenance = prov; target.UpdatedAt = time.Now()
	return nil
}

func (s *EpMemoryStore) Supersede(ctx context.Context, oldAtomID string, newAtom MemoryAtom, prov ProvenanceRecord) error {
	if err := ctx.Err(); err != nil { return err }
	s.mu.Lock(); defer s.mu.Unlock()
	old, ok := s.atoms[oldAtomID]
	if !ok { return fmt.Errorf("%w: %s", ErrAtomNotFound, oldAtomID) }
	now := time.Now(); old.Status = StatusSuperseded; old.SupersededBy = newAtom.ID; old.UpdatedAt = now
	s.atoms[newAtom.ID] = &StoredAtom{Atom: newAtom, Status: StatusActive, UpdatedAt: now}
	s.addEdgeLocked(MemoryEdge{FromID: newAtom.ID, ToID: oldAtomID, Relation: RelGeneralizes, Weight: 1, Provenance: prov, CreatedAt: now}, now)
	return nil
}

func (s *EpMemoryStore) GetAtom(ctx context.Context, id string) (*MemoryAtom, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	s.mu.RLock(); defer s.mu.RUnlock()
	stored, ok := s.atoms[id]
	if !ok || stored.Status != StatusActive { return nil, fmt.Errorf("%w: %s", ErrAtomNotFound, id) }
	copy := stored.Atom; return &copy, nil
}

func (s *EpMemoryStore) GetEdges(ctx context.Context, atomID string) ([]MemoryEdge, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	s.mu.RLock(); defer s.mu.RUnlock()
	return append([]MemoryEdge(nil), s.edgesFrom[atomID]...), nil
}

func (s *EpMemoryStore) Query(ctx context.Context, q QueryRequest) (*QueryResult, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	s.mu.RLock(); defer s.mu.RUnlock()
	var matches []MemoryAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive { continue }
		atom := stored.Atom
		if q.Namespace.raw != "" && !CanAccess(q.Namespace, atom.Namespace) { continue }
		if q.Kind != "" && atom.Kind != q.Kind { continue }
		if q.MinConf > 0 && atom.Confidence < q.MinConf { continue }
		if len(q.Tags) > 0 {
			tagMap := make(map[string]bool); for _, t := range atom.Tags { tagMap[t] = true }
			matchAll := true; for _, reqTag := range q.Tags { if !tagMap[reqTag] { matchAll = false; break } }
			if !matchAll { continue }
		}
		matches = append(matches, atom)
	}
	sort.Slice(matches, func(i, j int) bool { if matches[i].Confidence != matches[j].Confidence { return matches[i].Confidence > matches[j].Confidence }; return matches[i].CreatedAt.After(matches[j].CreatedAt) })
	total := len(matches); if q.Limit > 0 && len(matches) > q.Limit { matches = matches[:q.Limit] }
	return &QueryResult{Atoms: matches, Total: total}, nil
}
