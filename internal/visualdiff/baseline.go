package visualdiff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

var (
	ErrBaselineNotFound  = errors.New("visualdiff: baseline reference not found")
	ErrProtectedBaseline = errors.New("visualdiff: cannot overwrite protected baseline")
)

// BaselineReference encapsulates versioned, protected visual baseline metadata.
type BaselineReference struct {
	ID           string            `json:"id"`
	Scenario     string            `json:"scenario,omitempty"`
	Viewport     evidence.Viewport `json:"viewport"`
	DigestSHA256 string            `json:"digest_sha256"`
	Version      int               `json:"version"`
	Protected    bool              `json:"protected"`
	CreatedAt    time.Time         `json:"created_at"`
}

// BaselineStore manages storage and retrieval of versioned baselines.
type BaselineStore interface {
	Get(ctx context.Context, id string) (*BaselineReference, *image.RGBA, error)
	Put(ctx context.Context, ref BaselineReference, img *image.RGBA) error
	List(ctx context.Context) ([]BaselineReference, error)
}

// MemoryBaselineStore provides a thread-safe in-memory baseline store.
type MemoryBaselineStore struct {
	mu        sync.RWMutex
	baselines map[string]BaselineReference
	images    map[string]*image.RGBA
}

// NewMemoryBaselineStore creates an initialized MemoryBaselineStore.
func NewMemoryBaselineStore() *MemoryBaselineStore {
	return &MemoryBaselineStore{
		baselines: make(map[string]BaselineReference),
		images:    make(map[string]*image.RGBA),
	}
}

// Get retrieves a baseline reference and image by ID.
func (s *MemoryBaselineStore) Get(_ context.Context, id string) (*BaselineReference, *image.RGBA, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ref, ok := s.baselines[id]
	if !ok {
		return nil, nil, ErrBaselineNotFound
	}
	img := s.images[id]
	return &ref, img, nil
}

// Put stores or updates a baseline reference and image. Protected baselines cannot be overwritten.
func (s *MemoryBaselineStore) Put(_ context.Context, ref BaselineReference, img *image.RGBA) error {
	if img == nil {
		return errors.New("visualdiff: nil image provided for baseline")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.baselines[ref.ID]; ok && existing.Protected {
		return fmt.Errorf("%w: baseline %q is protected", ErrProtectedBaseline, ref.ID)
	}

	hash := sha256.Sum256(img.Pix)
	ref.DigestSHA256 = hex.EncodeToString(hash[:])
	if ref.CreatedAt.IsZero() {
		ref.CreatedAt = time.Now()
	}
	if ref.Version <= 0 {
		ref.Version = 1
	}

	s.baselines[ref.ID] = ref
	s.images[ref.ID] = cloneRGBA(img)
	return nil
}

// List returns all stored baseline references.
func (s *MemoryBaselineStore) List(_ context.Context) ([]BaselineReference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]BaselineReference, 0, len(s.baselines))
	for _, ref := range s.baselines {
		out = append(out, ref)
	}
	return out, nil
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	dst := image.NewRGBA(b)
	copy(dst.Pix, src.Pix)
	return dst
}
