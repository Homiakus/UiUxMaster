package fidelity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrCalibrationMissing             = errors.New("fidelity: calibration record missing")
	ErrCalibrationEnvironmentMismatch = errors.New("fidelity: calibration environment mismatch")
	ErrCalibrationExpired             = errors.New("fidelity: calibration expired")
	ErrCalibrationCoverage            = errors.New("fidelity: calibration corpus coverage insufficient")
	ErrCalibrationQuality             = errors.New("fidelity: calibration parity quality insufficient")
)

// CalibrationEnvironment identifies the runtime/profile dimensions whose drift
// can invalidate an approximate-vs-TruthPath parity result. Baseline-specific
// identity remains owned by FMEA-012; these fields describe renderer/runtime
// behavior, not a visual reference artifact.
type CalibrationEnvironment struct {
	RendererName      string  `json:"renderer_name"`
	RendererVersion   string  `json:"renderer_version"`
	FidelityID        string  `json:"fidelity_id,omitempty"`
	BrowserFamily     string  `json:"browser_family,omitempty"`
	BrowserVersion    string  `json:"browser_version,omitempty"`
	WorkerVersion     string  `json:"worker_version,omitempty"`
	RuntimeVersion    string  `json:"runtime_version,omitempty"`
	Platform          string  `json:"platform,omitempty"`
	ProfileID         string  `json:"profile_id,omitempty"`
	FontProfileDigest string  `json:"font_profile_digest,omitempty"`
	ViewportWidth     int     `json:"viewport_width,omitempty"`
	ViewportHeight    int     `json:"viewport_height,omitempty"`
	DeviceScale       float64 `json:"device_scale,omitempty"`
	ColorScheme       string  `json:"color_scheme,omitempty"`
}

func (e CalibrationEnvironment) normalized() CalibrationEnvironment {
	e.RendererName = strings.TrimSpace(strings.ToLower(e.RendererName))
	e.RendererVersion = strings.TrimSpace(e.RendererVersion)
	e.FidelityID = strings.TrimSpace(e.FidelityID)
	e.BrowserFamily = strings.TrimSpace(strings.ToLower(e.BrowserFamily))
	e.BrowserVersion = strings.TrimSpace(e.BrowserVersion)
	e.WorkerVersion = strings.TrimSpace(e.WorkerVersion)
	e.RuntimeVersion = strings.TrimSpace(e.RuntimeVersion)
	e.Platform = strings.TrimSpace(strings.ToLower(e.Platform))
	e.ProfileID = strings.TrimSpace(e.ProfileID)
	e.FontProfileDigest = strings.TrimSpace(e.FontProfileDigest)
	e.ColorScheme = strings.TrimSpace(strings.ToLower(e.ColorScheme))
	return e
}

func (e CalibrationEnvironment) Validate() error {
	e = e.normalized()
	if e.RendererName == "" || e.RendererVersion == "" {
		return fmt.Errorf("fidelity: calibration environment requires renderer name and exact version")
	}
	return nil
}

// CalibrationContext binds the approximate runtime to the exact TruthPath
// runtime used as the parity oracle. Drift on either side invalidates the record.
type CalibrationContext struct {
	Approx CalibrationEnvironment `json:"approx"`
	Truth  CalibrationEnvironment `json:"truth"`
}

func (c CalibrationContext) Key() (string, error) {
	c.Approx = c.Approx.normalized()
	c.Truth = c.Truth.normalized()
	if err := c.Approx.Validate(); err != nil {
		return "", fmt.Errorf("approx: %w", err)
	}
	if err := c.Truth.Validate(); err != nil {
		return "", fmt.Errorf("truth: %w", err)
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// CalibrationRecord is the persisted result of a parity corpus run for one
// evidence class, approximate tier and exact runtime pair.
type CalibrationRecord struct {
	Class          EvidenceClass      `json:"class"`
	Tier           Tier               `json:"tier"`
	Context        CalibrationContext `json:"context"`
	EnvironmentKey string             `json:"environment_key"`
	CorpusDigest   string             `json:"corpus_digest"`
	ArtifactRef    string             `json:"artifact_ref,omitempty"`
	Samples        int                `json:"samples"`
	PassedSamples  int                `json:"passed_samples"`
	CreatedAt      time.Time          `json:"created_at"`
	ExpiresAt      time.Time          `json:"expires_at,omitempty"`
}

func (r *CalibrationRecord) Normalize() error {
	if r == nil {
		return fmt.Errorf("fidelity: nil calibration record")
	}
	if r.Tier != TierL1 && r.Tier != TierL2 {
		return fmt.Errorf("fidelity: only L1/L2 parity calibration is stored, got %s", r.Tier)
	}
	if r.Class == "" {
		return fmt.Errorf("fidelity: calibration class is required")
	}
	key, err := r.Context.Key()
	if err != nil {
		return err
	}
	if r.EnvironmentKey != "" && r.EnvironmentKey != key {
		return fmt.Errorf("%w: record key=%s canonical=%s", ErrCalibrationEnvironmentMismatch, r.EnvironmentKey, key)
	}
	r.EnvironmentKey = key
	r.CorpusDigest = strings.TrimSpace(r.CorpusDigest)
	if r.CorpusDigest == "" {
		return fmt.Errorf("fidelity: calibration corpus digest is required")
	}
	if r.Samples < 0 || r.PassedSamples < 0 || r.PassedSamples > r.Samples {
		return fmt.Errorf("fidelity: invalid calibration sample counts %d/%d", r.PassedSamples, r.Samples)
	}
	return nil
}

func (r CalibrationRecord) PassRate() float64 {
	if r.Samples <= 0 {
		return 0
	}
	return float64(r.PassedSamples) / float64(r.Samples)
}

// CalibrationPolicy defines the minimum evidence needed before a stored parity
// result can authorize approximate-tier PASS.
type CalibrationPolicy struct {
	MinSamples  int
	MinPassRate float64
	MaxAge      time.Duration
}

func DefaultCalibrationPolicy() CalibrationPolicy {
	return CalibrationPolicy{MinSamples: 20, MinPassRate: 0.99, MaxAge: 30 * 24 * time.Hour}
}

// CalibrationRegistry is an in-process reference registry. Persistence adapters
// can mirror the same record shape without changing authority semantics.
type CalibrationRegistry struct {
	mu      sync.RWMutex
	records map[EvidenceClass]map[Tier][]CalibrationRecord
}

func NewCalibrationRegistry() *CalibrationRegistry {
	return &CalibrationRegistry{records: make(map[EvidenceClass]map[Tier][]CalibrationRecord)}
}

func (r *CalibrationRegistry) Put(record CalibrationRecord) error {
	if r == nil {
		return fmt.Errorf("fidelity: calibration registry is nil")
	}
	if err := record.Normalize(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	byTier := r.records[record.Class]
	if byTier == nil {
		byTier = make(map[Tier][]CalibrationRecord)
		r.records[record.Class] = byTier
	}
	items := append(byTier[record.Tier], record)
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	byTier[record.Tier] = items
	return nil
}

func (r *CalibrationRegistry) latest(class EvidenceClass, tier Tier) (CalibrationRecord, bool) {
	if r == nil {
		return CalibrationRecord{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	byTier := r.records[class]
	if byTier == nil || len(byTier[tier]) == 0 {
		return CalibrationRecord{}, false
	}
	return byTier[tier][0], true
}

// CalibrationAuthority is the runtime legal-pass gate. Static CalibrationMatrix
// says a tier may potentially prove a class; this authority says whether parity
// evidence for the exact current runtime pair is still valid.
type CalibrationAuthority struct {
	Registry *CalibrationRegistry
	Policy   CalibrationPolicy
	Now      func() time.Time
}

func NewCalibrationAuthority(registry *CalibrationRegistry, policy CalibrationPolicy) *CalibrationAuthority {
	if registry == nil {
		registry = NewCalibrationRegistry()
	}
	if policy.MinSamples <= 0 {
		policy = DefaultCalibrationPolicy()
	}
	return &CalibrationAuthority{Registry: registry, Policy: policy, Now: time.Now}
}

func NewStrictCalibrationAuthority() *CalibrationAuthority {
	return NewCalibrationAuthority(NewCalibrationRegistry(), DefaultCalibrationPolicy())
}

func (a *CalibrationAuthority) Validate(class EvidenceClass, tier Tier, current CalibrationContext) (CalibrationRecord, error) {
	if tier == TierL3 {
		return CalibrationRecord{Class: class, Tier: tier}, nil
	}
	if tier != TierL1 && tier != TierL2 {
		return CalibrationRecord{}, fmt.Errorf("%w: tier %s cannot be runtime-calibrated", ErrIllegalPass, tier)
	}
	if a == nil || a.Registry == nil {
		return CalibrationRecord{}, ErrCalibrationMissing
	}
	record, ok := a.Registry.latest(class, tier)
	if !ok {
		return CalibrationRecord{}, fmt.Errorf("%w: class=%s tier=%s", ErrCalibrationMissing, class, tier)
	}
	currentKey, err := current.Key()
	if err != nil {
		return CalibrationRecord{}, fmt.Errorf("%w: %v", ErrCalibrationEnvironmentMismatch, err)
	}
	if record.EnvironmentKey != currentKey {
		return CalibrationRecord{}, fmt.Errorf("%w: class=%s tier=%s stored=%s current=%s", ErrCalibrationEnvironmentMismatch, class, tier, record.EnvironmentKey, currentKey)
	}
	now := time.Now()
	if a.Now != nil {
		now = a.Now()
	}
	if (!record.ExpiresAt.IsZero() && !now.Before(record.ExpiresAt)) || (a.Policy.MaxAge > 0 && !record.CreatedAt.IsZero() && now.Sub(record.CreatedAt) > a.Policy.MaxAge) {
		return CalibrationRecord{}, fmt.Errorf("%w: class=%s tier=%s created=%s expires=%s", ErrCalibrationExpired, class, tier, record.CreatedAt, record.ExpiresAt)
	}
	if record.Samples < a.Policy.MinSamples {
		return CalibrationRecord{}, fmt.Errorf("%w: samples=%d minimum=%d", ErrCalibrationCoverage, record.Samples, a.Policy.MinSamples)
	}
	if record.PassRate() < a.Policy.MinPassRate {
		return CalibrationRecord{}, fmt.Errorf("%w: pass_rate=%.6f minimum=%.6f", ErrCalibrationQuality, record.PassRate(), a.Policy.MinPassRate)
	}
	return record, nil
}
