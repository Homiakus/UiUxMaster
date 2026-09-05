package sideeffect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type SourceState struct {
	HTML string `json:"html"`
	CSS  string `json:"css"`
}

func (s SourceState) Digest() string {
	return DigestBytes([]byte(s.HTML + "\x00" + s.CSS))
}

type sourceSnapshot struct {
	SchemaVersion int                `json:"schema_version"`
	State         SourceState        `json:"state"`
	Receipts      map[string]Receipt `json:"receipts"`
}

// SourceStore is a CAS-backed semantic source target. The file-backed form keeps
// source state and effect receipts in one atomic snapshot, so a crash cannot
// persist the mutation without also persisting the receipt that makes replay a
// no-op.
type SourceStore struct {
	mu       sync.Mutex
	path     string
	state    SourceState
	receipts map[string]Receipt
}

func NewMemorySourceStore(initial SourceState) *SourceStore {
	return &SourceStore{state: initial, receipts: make(map[string]Receipt)}
}

func NewFileSourceStore(path string, initial SourceState) (*SourceStore, error) {
	if path == "" {
		return nil, fmt.Errorf("sideeffect: source snapshot path is required")
	}
	path = filepath.Clean(path)
	if raw, err := os.ReadFile(path); err == nil {
		var snap sourceSnapshot
		if err := json.Unmarshal(raw, &snap); err != nil {
			return nil, fmt.Errorf("sideeffect: decode source snapshot: %w", err)
		}
		if snap.SchemaVersion != 1 {
			return nil, fmt.Errorf("sideeffect: unsupported source snapshot schema %d", snap.SchemaVersion)
		}
		if snap.Receipts == nil {
			snap.Receipts = make(map[string]Receipt)
		}
		return &SourceStore{path: path, state: snap.State, receipts: snap.Receipts}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("sideeffect: read source snapshot: %w", err)
	}

	s := &SourceStore{path: path, state: initial, receipts: make(map[string]Receipt)}
	if err := s.persistLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SourceStore) Current(_ context.Context) SourceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *SourceStore) Receipt(_ context.Context, operationID string) (Receipt, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.receipts[operationID]
	return r, ok
}

// CompareAndSwap applies desired exactly once for a stable Operation. A repeated
// operation with the same payload returns the original receipt with Reused=true.
// A different operation may update the source only from the exact expected
// revision; otherwise ErrCASConflict prevents overwriting concurrent work.
func (s *SourceStore) CompareAndSwap(ctx context.Context, op Operation, expectedDigest string, desired SourceState) (Receipt, error) {
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	if err := op.Validate(); err != nil {
		return Receipt{}, err
	}
	desiredDigest := desired.Digest()
	if op.PayloadDigest != desiredDigest {
		return Receipt{}, fmt.Errorf("%w: operation payload=%s desired=%s", ErrOperationConflict, op.PayloadDigest, desiredDigest)
	}
	opID, _ := op.ID()

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.receipts[opID]; ok {
		if existing.PayloadDigest != op.PayloadDigest {
			return Receipt{}, ErrOperationConflict
		}
		existing.Reused = true
		existing.Applied = false
		return existing, nil
	}

	currentDigest := s.state.Digest()
	if currentDigest != expectedDigest {
		return Receipt{}, fmt.Errorf("%w: expected=%s current=%s", ErrCASConflict, expectedDigest, currentDigest)
	}

	receipt := Receipt{
		OperationID:   opID,
		Kind:          op.Kind,
		PayloadDigest: op.PayloadDigest,
		ResultDigest:  desiredDigest,
		RetryClass:    op.RetryClass,
		Applied:       true,
	}
	oldState := s.state
	s.state = desired
	s.receipts[opID] = receipt
	if err := s.persistLocked(); err != nil {
		s.state = oldState
		delete(s.receipts, opID)
		return Receipt{}, err
	}
	return receipt, nil
}

func (s *SourceStore) persistLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("sideeffect: create source snapshot directory: %w", err)
	}
	raw, err := json.MarshalIndent(sourceSnapshot{SchemaVersion: 1, State: s.state, Receipts: s.receipts}, "", "  ")
	if err != nil {
		return fmt.Errorf("sideeffect: encode source snapshot: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".source-effects-*.tmp")
	if err != nil {
		return fmt.Errorf("sideeffect: create source temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close(); cleanup(); return fmt.Errorf("sideeffect: write source snapshot: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close(); cleanup(); return fmt.Errorf("sideeffect: fsync source snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup(); return fmt.Errorf("sideeffect: close source snapshot: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		cleanup(); return fmt.Errorf("sideeffect: replace source snapshot: %w", err)
	}
	if dir, err := os.Open(filepath.Dir(s.path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}
