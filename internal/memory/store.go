package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrAtomNotFound      = errors.New("memory atom not found")
	ErrConflictExists    = errors.New("memory conflict detected")
	ErrAtomAlreadyExists = errors.New("memory atom with id already exists")
)

// AtomStatus represents the lifecycle state of a memory atom.
type AtomStatus string

const (
	StatusActive     AtomStatus = "ACTIVE"
	StatusRetracted  AtomStatus = "RETRACTED"
	StatusSuperseded AtomStatus = "SUPERSEDED"
)

// StoredAtom wraps a MemoryAtom with lifecycle state and conflict tracking.
type StoredAtom struct {
	Atom         MemoryAtom   `json:"atom"`
	Status       AtomStatus   `json:"status"`
	RetractReason string       `json:"retract_reason,omitempty"`
	SupersededBy string       `json:"superseded_by,omitempty"`
	Conflicts    []string     `json:"conflicts,omitempty"` // IDs of conflicting atoms/counterexamples
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ConflictRecord describes a detected contradiction between design assertions.
type ConflictRecord struct {
	PrimaryAtomID    string           `json:"primary_atom_id"`
	ConflictingAtomID string          `json:"conflicting_atom_id"`
	Reason           string           `json:"reason"`
	DetectedAt       time.Time        `json:"detected_at"`
}

// QueryRequest filters memory atoms.
type QueryRequest struct {
	Namespace Namespace
	Kind      NodeKind
	Tags      []string
	MinConf   float64
	Limit     int
}

// QueryResult returns matching atoms.
type QueryResult struct {
	Atoms []MemoryAtom `json:"atoms"`
	Total int          `json:"total"`
}

// Store defines the epistemic memory store contract.
type Store interface {
	Commit(ctx context.Context, bundle AdmissionBundle) error
	Retract(ctx context.Context, atomID string, reason string, prov ProvenanceRecord) error
	Supersede(ctx context.Context, oldAtomID string, newAtom MemoryAtom, prov ProvenanceRecord) error
	GetAtom(ctx context.Context, id string) (*MemoryAtom, error)
	GetEdges(ctx context.Context, atomID string) ([]MemoryEdge, error)
	Query(ctx context.Context, q QueryRequest) (*QueryResult, error)
	RetrieveContextPack(ctx context.Context, req ContextPackRequest) (*ContextPack, error)
}

// EpMemoryStore is an in-process, thread-safe, conflict-preserving epistemic memory store.
type EpMemoryStore struct {
	mu        sync.RWMutex
	atoms     map[string]*StoredAtom
	edgesFrom map[string][]MemoryEdge
	edgesTo   map[string][]MemoryEdge
	conflicts []ConflictRecord
}

// NewEpMemoryStore creates an initialized EpMemoryStore.
func NewEpMemoryStore() *EpMemoryStore {
	return &EpMemoryStore{
		atoms:     make(map[string]*StoredAtom),
		edgesFrom: make(map[string][]MemoryEdge),
		edgesTo:   make(map[string][]MemoryEdge),
		conflicts: make([]ConflictRecord, 0),
	}
}

// Commit transactionally applies an admitted bundle to the store.
func (s *EpMemoryStore) Commit(ctx context.Context, bundle AdmissionBundle) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// 1. Insert atoms
	for _, atom := range bundle.Atoms {
		existing, found := s.atoms[atom.ID]
		if found && existing.Status == StatusActive {
			// Update timestamp and merge data if identical, or register conflict if contradictory
			existing.Atom = atom
			existing.UpdatedAt = now
			continue
		}

		s.atoms[atom.ID] = &StoredAtom{
			Atom:      atom,
			Status:    StatusActive,
			UpdatedAt: now,
		}
	}

	// 2. Insert edges and detect relationships/conflicts
	for _, edge := range bundle.Edges {
		s.edgesFrom[edge.FromID] = append(s.edgesFrom[edge.FromID], edge)
		s.edgesTo[edge.ToID] = append(s.edgesTo[edge.ToID], edge)

		// If edge is a refutation or counterexample, record a conflict without deleting truth
		if edge.Relation == RelRefutes || edge.Relation == RelCounterexampleTo {
			s.conflicts = append(s.conflicts, ConflictRecord{
				PrimaryAtomID:     edge.ToID,
				ConflictingAtomID: edge.FromID,
				Reason:            fmt.Sprintf("Relation %s asserted from %s", edge.Relation, edge.FromID),
				DetectedAt:        now,
			})

			// Link conflict to stored atom
			if target, ok := s.atoms[edge.ToID]; ok {
				target.Conflicts = append(target.Conflicts, edge.FromID)
			}
		}
	}

	return nil
}

// Retract marks an atom as retracted with a documented reason and provenance.
func (s *EpMemoryStore) Retract(ctx context.Context, atomID string, reason string, prov ProvenanceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	target, ok := s.atoms[atomID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrAtomNotFound, atomID)
	}

	target.Status = StatusRetracted
	target.RetractReason = reason
	target.Atom.Provenance = prov
	target.UpdatedAt = time.Now()

	return nil
}

// Supersede atomically replaces an existing atom with a newer version.
func (s *EpMemoryStore) Supersede(ctx context.Context, oldAtomID string, newAtom MemoryAtom, prov ProvenanceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.atoms[oldAtomID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrAtomNotFound, oldAtomID)
	}

	now := time.Now()
	old.Status = StatusSuperseded
	old.SupersededBy = newAtom.ID
	old.UpdatedAt = now

	s.atoms[newAtom.ID] = &StoredAtom{
		Atom:      newAtom,
		Status:    StatusActive,
		UpdatedAt: now,
	}

	// Link with edge
	edge := MemoryEdge{
		FromID:     newAtom.ID,
		ToID:       oldAtomID,
		Relation:   RelGeneralizes,
		Weight:     1.0,
		Provenance: prov,
		CreatedAt:  now,
	}
	s.edgesFrom[newAtom.ID] = append(s.edgesFrom[newAtom.ID], edge)
	s.edgesTo[oldAtomID] = append(s.edgesTo[oldAtomID], edge)

	return nil
}

// GetAtom retrieves a single active or stored atom by ID.
func (s *EpMemoryStore) GetAtom(ctx context.Context, id string) (*MemoryAtom, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, ok := s.atoms[id]
	if !ok || stored.Status != StatusActive {
		return nil, fmt.Errorf("%w: %s", ErrAtomNotFound, id)
	}
	copy := stored.Atom
	return &copy, nil
}

// GetEdges retrieves all outgoing and incoming edges for an atom.
func (s *EpMemoryStore) GetEdges(ctx context.Context, atomID string) ([]MemoryEdge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []MemoryEdge
	if from, ok := s.edgesFrom[atomID]; ok {
		result = append(result, from...)
	}
	return result, nil
}

// Query searches stored atoms by namespace, kind, tags, and minimum confidence.
func (s *EpMemoryStore) Query(ctx context.Context, q QueryRequest) (*QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []MemoryAtom
	for _, stored := range s.atoms {
		if stored.Status != StatusActive {
			continue
		}

		atom := stored.Atom
		// Namespace check via firewall
		if q.Namespace.raw != "" && !CanAccess(q.Namespace, atom.Namespace) {
			continue
		}

		// Kind check
		if q.Kind != "" && atom.Kind != q.Kind {
			continue
		}

		// Confidence threshold
		if q.MinConf > 0 && atom.Confidence < q.MinConf {
			continue
		}

		// Tags filter (must contain all specified tags)
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

	// Sort by confidence descending, then by CreatedAt descending
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
