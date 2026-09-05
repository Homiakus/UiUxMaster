package visualdiff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

var (
	ErrBaselineNotFound      = errors.New("visualdiff: baseline reference not found")
	ErrProtectedBaseline     = errors.New("visualdiff: cannot overwrite protected baseline")
	ErrBaselineUpdateRequired = errors.New("visualdiff: baseline update requires explicit reviewed operation")
	ErrBaselineUpdateConflict = errors.New("visualdiff: baseline update compare-and-swap conflict")
	ErrBaselineReviewRequired = errors.New("visualdiff: baseline update requires reviewer and rationale")
)

// BaselineReference binds visual pixels to the exact render environment under
// which those pixels are meaningful. EnvironmentKey is recomputed by the store
// and never trusted from callers.
type BaselineReference struct {
	ID             string                             `json:"id"`
	Scenario       string                             `json:"scenario,omitempty"`
	Viewport       evidence.Viewport                  `json:"viewport"`
	Environment    evidence.RenderEnvironmentIdentity `json:"environment"`
	EnvironmentKey string                             `json:"environment_key"`
	DigestSHA256   string                             `json:"digest_sha256"`
	Version        int                                `json:"version"`
	Protected      bool                               `json:"protected"`
	CreatedAt      time.Time                          `json:"created_at"`
	UpdatedAt      time.Time                          `json:"updated_at,omitempty"`
}

// BaselineUpdateRequest is an explicit, reviewed CAS operation. It is the only
// legal overwrite path, including for protected baselines.
type BaselineUpdateRequest struct {
	ID                 string                             `json:"id"`
	ExpectedVersion    int                                `json:"expected_version"`
	ExpectedDigest     string                             `json:"expected_digest"`
	Scenario           string                             `json:"scenario,omitempty"`
	Environment        evidence.RenderEnvironmentIdentity `json:"environment"`
	Protected          bool                               `json:"protected"`
	Reason             string                             `json:"reason"`
	ReviewedBy         string                             `json:"reviewed_by"`
}

type BaselineUpdateRecord struct {
	BaselineID        string    `json:"baseline_id"`
	OldVersion        int       `json:"old_version"`
	NewVersion        int       `json:"new_version"`
	OldDigest         string    `json:"old_digest"`
	NewDigest         string    `json:"new_digest"`
	OldEnvironmentKey string    `json:"old_environment_key"`
	NewEnvironmentKey string    `json:"new_environment_key"`
	Reason            string    `json:"reason"`
	ReviewedBy        string    `json:"reviewed_by"`
	At                time.Time `json:"at"`
}

type BaselineMetrics struct {
	Creates            int `json:"creates"`
	Updates            int `json:"updates"`
	EnvironmentChanges int `json:"environment_changes"`
}

// BaselineStore manages versioned baselines and their reviewed change history.
type BaselineStore interface {
	Get(ctx context.Context, id string) (*BaselineReference, *image.RGBA, error)
	Put(ctx context.Context, ref BaselineReference, img *image.RGBA) error
	Update(ctx context.Context, req BaselineUpdateRequest, img *image.RGBA) (*BaselineReference, error)
	History(ctx context.Context, id string) ([]BaselineUpdateRecord, error)
	List(ctx context.Context) ([]BaselineReference, error)
	Metrics(ctx context.Context) BaselineMetrics
}

// MemoryBaselineStore is a thread-safe reference implementation. The update
// contract is store-level so future file/DB stores must preserve the same CAS and
// provenance semantics.
type MemoryBaselineStore struct {
	mu        sync.RWMutex
	baselines map[string]BaselineReference
	images    map[string]*image.RGBA
	history   map[string][]BaselineUpdateRecord
	metrics   BaselineMetrics
}

func NewMemoryBaselineStore() *MemoryBaselineStore {
	return &MemoryBaselineStore{
		baselines: make(map[string]BaselineReference),
		images:    make(map[string]*image.RGBA),
		history:   make(map[string][]BaselineUpdateRecord),
	}
}

func (s *MemoryBaselineStore) Get(_ context.Context, id string) (*BaselineReference, *image.RGBA, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ref, ok := s.baselines[id]
	if !ok { return nil, nil, ErrBaselineNotFound }
	return &ref, cloneRGBA(s.images[id]), nil
}

// Put is create-only. Silent overwrite is forbidden even for unprotected
// baselines; callers must use Update so rationale, reviewer and old/new identity
// are permanently visible.
func (s *MemoryBaselineStore) Put(_ context.Context, ref BaselineReference, img *image.RGBA) error {
	if img == nil { return errors.New("visualdiff: nil image provided for baseline") }
	ref.ID = strings.TrimSpace(ref.ID)
	if ref.ID == "" { return errors.New("visualdiff: baseline id is required") }
	envKey, err := ref.Environment.CanonicalKey()
	if err != nil { return fmt.Errorf("visualdiff: baseline environment: %w", err) }

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.baselines[ref.ID]; ok {
		if existing.Protected { return fmt.Errorf("%w: baseline %q", ErrProtectedBaseline, ref.ID) }
		return fmt.Errorf("%w: baseline %q", ErrBaselineUpdateRequired, ref.ID)
	}

	now := time.Now().UTC()
	ref.Environment = ref.Environment.Normalize()
	ref.EnvironmentKey = envKey
	ref.Viewport = viewportFromEnvironment(ref.Environment)
	ref.DigestSHA256 = imageDigest(img)
	if ref.Version <= 0 { ref.Version = 1 }
	if ref.CreatedAt.IsZero() { ref.CreatedAt = now }
	ref.UpdatedAt = ref.CreatedAt
	s.baselines[ref.ID] = ref
	s.images[ref.ID] = cloneRGBA(img)
	s.metrics.Creates++
	return nil
}

func (s *MemoryBaselineStore) Update(_ context.Context, req BaselineUpdateRequest, img *image.RGBA) (*BaselineReference, error) {
	if img == nil { return nil, errors.New("visualdiff: nil image provided for baseline") }
	req.ID = strings.TrimSpace(req.ID)
	if strings.TrimSpace(req.Reason) == "" || strings.TrimSpace(req.ReviewedBy) == "" {
		return nil, ErrBaselineReviewRequired
	}
	newEnvKey, err := req.Environment.CanonicalKey()
	if err != nil { return nil, fmt.Errorf("visualdiff: baseline environment: %w", err) }

	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.baselines[req.ID]
	if !ok { return nil, ErrBaselineNotFound }
	if req.ExpectedVersion != old.Version || !strings.EqualFold(strings.TrimSpace(req.ExpectedDigest), old.DigestSHA256) {
		return nil, fmt.Errorf("%w: expected version=%d digest=%q; current version=%d digest=%q", ErrBaselineUpdateConflict, req.ExpectedVersion, req.ExpectedDigest, old.Version, old.DigestSHA256)
	}

	now := time.Now().UTC()
	updated := old
	updated.Version = old.Version + 1
	updated.DigestSHA256 = imageDigest(img)
	updated.Environment = req.Environment.Normalize()
	updated.EnvironmentKey = newEnvKey
	updated.Viewport = viewportFromEnvironment(updated.Environment)
	updated.Scenario = req.Scenario
	updated.Protected = req.Protected
	updated.UpdatedAt = now

	record := BaselineUpdateRecord{
		BaselineID: req.ID,
		OldVersion: old.Version,
		NewVersion: updated.Version,
		OldDigest: old.DigestSHA256,
		NewDigest: updated.DigestSHA256,
		OldEnvironmentKey: old.EnvironmentKey,
		NewEnvironmentKey: updated.EnvironmentKey,
		Reason: strings.TrimSpace(req.Reason),
		ReviewedBy: strings.TrimSpace(req.ReviewedBy),
		At: now,
	}
	s.baselines[req.ID] = updated
	s.images[req.ID] = cloneRGBA(img)
	s.history[req.ID] = append(s.history[req.ID], record)
	s.metrics.Updates++
	if old.EnvironmentKey != updated.EnvironmentKey { s.metrics.EnvironmentChanges++ }
	copy := updated
	return &copy, nil
}

func (s *MemoryBaselineStore) History(_ context.Context, id string) ([]BaselineUpdateRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.baselines[id]; !ok { return nil, ErrBaselineNotFound }
	return append([]BaselineUpdateRecord(nil), s.history[id]...), nil
}

func (s *MemoryBaselineStore) List(_ context.Context) ([]BaselineReference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]BaselineReference, 0, len(s.baselines))
	for _, ref := range s.baselines { out = append(out, ref) }
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *MemoryBaselineStore) Metrics(_ context.Context) BaselineMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics
}

func imageDigest(img *image.RGBA) string {
	hash := sha256.Sum256(img.Pix)
	return hex.EncodeToString(hash[:])
}

func viewportFromEnvironment(env evidence.RenderEnvironmentIdentity) evidence.Viewport {
	return evidence.Viewport{
		Width: env.ViewportWidth,
		Height: env.ViewportHeight,
		DeviceScale: env.DeviceScale,
		ColorScheme: env.Theme,
		Browser: env.BrowserFamily,
	}
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	if src == nil { return nil }
	b := src.Bounds()
	dst := image.NewRGBA(b)
	copy(dst.Pix, src.Pix)
	return dst
}
