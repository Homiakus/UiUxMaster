package visualdiff

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"sort"
	"strings"
	"sync"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

var (
	ErrBaselineIncompatible = errors.New("visualdiff: baseline incompatible with candidate environment")
	ErrBaselineIntegrity    = errors.New("visualdiff: baseline pixel digest does not match reference")
	ErrMaskOwnership        = errors.New("visualdiff: dynamic mask exceeds declared semantic ownership")
)

type ComparisonRequest struct {
	Baseline             BaselineReference                   `json:"baseline"`
	BaselineImage        *image.RGBA                         `json:"-"`
	CandidateEnvironment evidence.RenderEnvironmentIdentity `json:"candidate_environment"`
	CandidateImage       *image.RGBA                         `json:"-"`
	Elements             []evidence.ElementRef               `json:"elements,omitempty"`
	Masks                []evidence.DynamicMask              `json:"masks,omitempty"`
	Options              Options                             `json:"options"`
}

type ComparisonResult struct {
	PixelResult             Result `json:"pixel_result"`
	BaselineEnvironmentKey  string `json:"baseline_environment_key"`
	CandidateEnvironmentKey string `json:"candidate_environment_key"`
	BaselineDigest          string `json:"baseline_digest"`
	CandidateDigest         string `json:"candidate_digest"`
	ComparisonDigest        string `json:"comparison_digest"`
	MasksApplied            int    `json:"masks_applied"`
}

type ComparisonMetrics struct {
	Comparisons             int `json:"comparisons"`
	IncompatibleComparisons int `json:"incompatible_comparisons"`
	OutcomeFlips            int `json:"outcome_flips"`
}

// Comparator owns protected-baseline comparison metrics. Metrics are keyed by
// baseline environment key, making incompatibility/flakiness visible per exact
// browser/runtime population instead of blending unrelated environments.
type Comparator struct {
	mu       sync.Mutex
	metrics  map[string]ComparisonMetrics
	outcomes map[string]string
}

func NewComparator() *Comparator {
	return &Comparator{
		metrics:  make(map[string]ComparisonMetrics),
		outcomes: make(map[string]string),
	}
}

// CompareBaseline establishes baseline integrity and environment compatibility
// before validating masks and before applying any tolerance/pixel comparison.
func (c *Comparator) CompareBaseline(req ComparisonRequest) (ComparisonResult, error) {
	if req.BaselineImage == nil || req.CandidateImage == nil {
		return ComparisonResult{}, fmt.Errorf("visualdiff: baseline and candidate images are required")
	}

	baselineKey, err := req.Baseline.Environment.CanonicalKey()
	if err != nil {
		return ComparisonResult{}, fmt.Errorf("visualdiff: baseline environment: %w", err)
	}
	candidateKey, err := req.CandidateEnvironment.CanonicalKey()
	if err != nil {
		c.recordIncompatible(baselineKey)
		return ComparisonResult{}, fmt.Errorf("%w: candidate identity invalid: %v", ErrBaselineIncompatible, err)
	}
	if req.Baseline.EnvironmentKey != "" && req.Baseline.EnvironmentKey != baselineKey {
		c.recordIncompatible(baselineKey)
		return ComparisonResult{}, fmt.Errorf("%w: stored environment key does not match stored identity", ErrBaselineIncompatible)
	}
	if baselineKey != candidateKey {
		c.recordIncompatible(baselineKey)
		return ComparisonResult{}, fmt.Errorf("%w: baseline=%s candidate=%s", ErrBaselineIncompatible, baselineKey, candidateKey)
	}

	actualBaselineDigest := imageDigest(req.BaselineImage)
	baselineDigest := strings.TrimSpace(req.Baseline.DigestSHA256)
	if baselineDigest != "" && !strings.EqualFold(baselineDigest, actualBaselineDigest) {
		return ComparisonResult{}, fmt.Errorf("%w: reference=%s actual=%s", ErrBaselineIntegrity, baselineDigest, actualBaselineDigest)
	}
	if baselineDigest == "" {
		baselineDigest = actualBaselineDigest
	}

	maskRects, err := validateMaskOwnership(req.Masks, req.Elements, req.CandidateImage.Bounds())
	if err != nil {
		return ComparisonResult{}, err
	}
	pixelResult, err := compareRGBA(req.BaselineImage, req.CandidateImage, req.Options, maskRects)
	if err != nil {
		return ComparisonResult{}, err
	}

	candidateDigest := imageDigest(req.CandidateImage)
	comparisonDigest := digestComparison(baselineKey, baselineDigest, candidateDigest, req.Options, req.Masks, pixelResult)
	result := ComparisonResult{
		PixelResult:             pixelResult,
		BaselineEnvironmentKey:  baselineKey,
		CandidateEnvironmentKey: candidateKey,
		BaselineDigest:          baselineDigest,
		CandidateDigest:         candidateDigest,
		ComparisonDigest:        comparisonDigest,
		MasksApplied:            len(maskRects),
	}
	c.recordOutcome(req.Baseline.ID, baselineKey, candidateDigest, comparisonDigest)
	return result, nil
}

func (c *Comparator) Metrics(environmentKey string) ComparisonMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.metrics[environmentKey]
}

func (c *Comparator) recordIncompatible(environmentKey string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	m := c.metrics[environmentKey]
	m.IncompatibleComparisons++
	c.metrics[environmentKey] = m
}

func (c *Comparator) recordOutcome(baselineID, environmentKey, candidateDigest, comparisonDigest string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	m := c.metrics[environmentKey]
	m.Comparisons++
	slot := strings.Join([]string{baselineID, environmentKey, candidateDigest}, "\x00")
	if prior, ok := c.outcomes[slot]; ok && prior != comparisonDigest {
		m.OutcomeFlips++
	}
	c.outcomes[slot] = comparisonDigest
	c.metrics[environmentKey] = m
}

func validateMaskOwnership(masks []evidence.DynamicMask, elements []evidence.ElementRef, imageBounds image.Rectangle) ([]image.Rectangle, error) {
	owners := make(map[string]evidence.Rect, len(elements))
	for _, element := range elements {
		if element.ID != "" && element.Visible {
			owners[element.ID] = element.Bounds
		}
	}

	out := make([]image.Rectangle, 0, len(masks))
	seen := make(map[string]struct{}, len(masks))
	for _, mask := range masks {
		if strings.TrimSpace(mask.ID) == "" || strings.TrimSpace(mask.OwnerElementID) == "" {
			return nil, fmt.Errorf("%w: mask id and owner are required", ErrMaskOwnership)
		}
		if _, dup := seen[mask.ID]; dup {
			return nil, fmt.Errorf("%w: duplicate mask id %q", ErrMaskOwnership, mask.ID)
		}
		seen[mask.ID] = struct{}{}

		owner, ok := owners[mask.OwnerElementID]
		if !ok {
			return nil, fmt.Errorf("%w: owner %q is not a visible evidence element", ErrMaskOwnership, mask.OwnerElementID)
		}
		if mask.Bounds.Width <= 0 || mask.Bounds.Height <= 0 || !rectContains(owner, mask.Bounds) {
			return nil, fmt.Errorf("%w: mask %q bounds are outside owner %q", ErrMaskOwnership, mask.ID, mask.OwnerElementID)
		}

		r := image.Rect(
			int(mask.Bounds.X),
			int(mask.Bounds.Y),
			int(mask.Bounds.X+mask.Bounds.Width),
			int(mask.Bounds.Y+mask.Bounds.Height),
		)
		if r.Empty() || !r.In(imageBounds) {
			return nil, fmt.Errorf("%w: mask %q is outside image bounds", ErrMaskOwnership, mask.ID)
		}
		out = append(out, r)
	}
	return out, nil
}

func rectContains(owner, child evidence.Rect) bool {
	return child.X >= owner.X && child.Y >= owner.Y &&
		child.X+child.Width <= owner.X+owner.Width &&
		child.Y+child.Height <= owner.Y+owner.Height
}

func digestComparison(environmentKey, baselineDigest, candidateDigest string, opts Options, masks []evidence.DynamicMask, result Result) string {
	maskIDs := make([]string, 0, len(masks))
	for _, mask := range masks {
		maskIDs = append(maskIDs, fmt.Sprintf(
			"%s:%s:%.3f,%.3f,%.3f,%.3f",
			mask.ID,
			mask.OwnerElementID,
			mask.Bounds.X,
			mask.Bounds.Y,
			mask.Bounds.Width,
			mask.Bounds.Height,
		))
	}
	sort.Strings(maskIDs)
	payload := fmt.Sprintf(
		"%s\x00%s\x00%s\x00tol=%d\x00masks=%s\x00changed=%d\x00ratio=%.12f\x00max=%d",
		environmentKey,
		baselineDigest,
		candidateDigest,
		opts.ChannelTolerance,
		strings.Join(maskIDs, "|"),
		result.ChangedPixels,
		result.ChangeRatio,
		result.MaxDelta,
	)
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}
