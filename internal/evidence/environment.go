package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrIncompleteRenderEnvironment = errors.New("evidence: incomplete render environment identity")

const RenderEnvironmentSchemaVersion = 1

// RenderEnvironmentIdentity is the canonical identity of a visual-rendering
// environment. Pixel baselines are only comparable when their canonical keys are
// identical. Unknown material dimensions are not wildcarded: an incomplete key
// cannot authorize a protected baseline comparison.
type RenderEnvironmentIdentity struct {
	SchemaVersion   int     `json:"schema_version"`
	RendererName    string  `json:"renderer_name"`
	RendererVersion string  `json:"renderer_version"`
	WorkerVersion   string  `json:"worker_version"`
	BrowserFamily   string  `json:"browser_family"`
	BrowserEngine   string  `json:"browser_engine"`
	BrowserVersion  string  `json:"browser_version"`
	Platform        string  `json:"platform"`
	ViewportWidth   int     `json:"viewport_width"`
	ViewportHeight  int     `json:"viewport_height"`
	DeviceScale     float64 `json:"device_scale"`
	Theme           string  `json:"theme"`
	FontSetDigest   string  `json:"font_set_digest"`
	Locale          string  `json:"locale"`
	Timezone        string  `json:"timezone"`
	FixtureRevision string  `json:"fixture_revision"`
}

func (e RenderEnvironmentIdentity) Normalize() RenderEnvironmentIdentity {
	if e.SchemaVersion == 0 {
		e.SchemaVersion = RenderEnvironmentSchemaVersion
	}
	e.RendererName = strings.TrimSpace(e.RendererName)
	e.RendererVersion = strings.TrimSpace(e.RendererVersion)
	e.WorkerVersion = strings.TrimSpace(e.WorkerVersion)
	e.BrowserFamily = strings.ToLower(strings.TrimSpace(e.BrowserFamily))
	e.BrowserEngine = strings.ToLower(strings.TrimSpace(e.BrowserEngine))
	e.BrowserVersion = strings.TrimSpace(e.BrowserVersion)
	e.Platform = strings.ToLower(strings.TrimSpace(e.Platform))
	e.Theme = strings.ToLower(strings.TrimSpace(e.Theme))
	e.FontSetDigest = strings.ToLower(strings.TrimSpace(e.FontSetDigest))
	e.Locale = strings.TrimSpace(e.Locale)
	e.Timezone = strings.TrimSpace(e.Timezone)
	e.FixtureRevision = strings.TrimSpace(e.FixtureRevision)
	return e
}

func (e RenderEnvironmentIdentity) Validate() error {
	e = e.Normalize()
	if e.SchemaVersion != RenderEnvironmentSchemaVersion {
		return fmt.Errorf("%w: schema_version=%d", ErrIncompleteRenderEnvironment, e.SchemaVersion)
	}
	missing := make([]string, 0, 12)
	for name, value := range map[string]string{
		"renderer_name": e.RendererName,
		"renderer_version": e.RendererVersion,
		"worker_version": e.WorkerVersion,
		"browser_family": e.BrowserFamily,
		"browser_engine": e.BrowserEngine,
		"browser_version": e.BrowserVersion,
		"platform": e.Platform,
		"theme": e.Theme,
		"font_set_digest": e.FontSetDigest,
		"locale": e.Locale,
		"timezone": e.Timezone,
		"fixture_revision": e.FixtureRevision,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if e.ViewportWidth <= 0 {
		missing = append(missing, "viewport_width")
	}
	if e.ViewportHeight <= 0 {
		missing = append(missing, "viewport_height")
	}
	if e.DeviceScale <= 0 {
		missing = append(missing, "device_scale")
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%w: missing/invalid %s", ErrIncompleteRenderEnvironment, strings.Join(missing, ","))
	}
	return nil
}

// CanonicalKey returns a versioned SHA-256 key over normalized JSON. The full
// normalized identity remains stored beside the key for auditability.
func (e RenderEnvironmentIdentity) CanonicalKey() (string, error) {
	e = e.Normalize()
	if err := e.Validate(); err != nil {
		return "", err
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("evidence: encode render environment: %w", err)
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("render-env-v%d:%s", e.SchemaVersion, hex.EncodeToString(sum[:])), nil
}

// FontSetDigest creates a deterministic digest over observed font faces. A
// caller must still decide whether the observation is complete enough for the
// baseline policy; an empty/unknown set intentionally returns an empty digest.
func FontSetDigest(fonts *FontEvidence) string {
	if fonts == nil || strings.TrimSpace(fonts.Status) == "" || len(fonts.Faces) == 0 {
		return ""
	}
	faces := make([]string, 0, len(fonts.Faces))
	for _, f := range fonts.Faces {
		faces = append(faces, strings.Join([]string{
			strings.TrimSpace(f.Family),
			strings.TrimSpace(f.Style),
			strings.TrimSpace(f.Weight),
			strings.TrimSpace(f.Stretch),
			strings.TrimSpace(f.Status),
		}, "\x00"))
	}
	sort.Strings(faces)
	sum := sha256.Sum256([]byte(strings.Join(faces, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// DynamicMask is a deterministic exclusion region bound to a declared semantic
// owner. Visual comparison code must verify that Bounds is contained within the
// current owner element's bounds before excluding any pixel.
type DynamicMask struct {
	ID             string `json:"id"`
	OwnerElementID string `json:"owner_element_id"`
	Bounds         Rect   `json:"bounds"`
}
