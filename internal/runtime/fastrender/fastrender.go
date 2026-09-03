package fastrender

import (
	"context"
	"errors"
	"image"
	"time"
)

var ErrUnsupported = errors.New("fastrender: operation unsupported")

// Request describes a deterministic render target without naming a renderer vendor.
type Request struct {
	HTML       []byte
	CSS        []byte
	Width      int
	Height     int
	DPR        float64
	Theme      string
	BaseURL    string
	RegionHint *image.Rectangle
}

// InspectRequest requests structure/geometry without requiring encoded pixels.
type InspectRequest struct {
	HTML    []byte
	CSS     []byte
	Width   int
	Height  int
	DPR     float64
	Theme   string
	BaseURL string
}

// RegionRequest captures only a requested rendered region when supported.
type RegionRequest struct {
	Render Request
	Clip   image.Rectangle
}

// Box is a renderer-neutral semantic/layout box.
type Box struct {
	Ref    string
	Kind   string
	Bounds image.Rectangle
	Text   string
}

// StyleSummary intentionally contains only verifier-relevant values. Renderer
// adapters should not dump the entire computed style universe by default.
type StyleSummary struct {
	Ref        string
	Display    string
	Position   string
	OverflowX  string
	OverflowY  string
	FontSize   string
	FontWeight string
	LineHeight string
	Color      string
	Background string
}

// Capabilities is used by the validation router and fidelity layer.
type Capabilities struct {
	Name             string
	Version          string
	BrowserAccurate  bool
	SupportsPixels   bool
	SupportsGeometry bool
	SupportsStyles   bool
	SupportsScenario bool
	FeatureNames     []string
}

// Latency exposes where render time was spent.
type Latency struct {
	Prepare time.Duration
	Layout  time.Duration
	Paint   time.Duration
	Capture time.Duration
	Total   time.Duration
}

// Evidence is the common in-process render result. RGBA may be nil for
// structure-only operations. No PNG encode/decode is required for the hot path.
type Evidence struct {
	RGBA       *image.RGBA
	Boxes      []Box
	Styles     []StyleSummary
	Renderer   Capabilities
	Latency    Latency
	Warnings   []string
	FidelityID string
}

// StructuralEvidence is the cheap geometry/style subset.
type StructuralEvidence struct {
	Boxes    []Box
	Styles   []StyleSummary
	Renderer Capabilities
	Latency  Latency
	Warnings []string
}

// Scenario is intentionally minimal for L1. Interactive renderers can implement
// richer semantics behind adapters while preserving this stable boundary.
type Scenario struct {
	ID      string
	Actions []Action
}

type Action struct {
	Kind   string
	Target string
	Value  string
}

type ScenarioEvidence struct {
	Checkpoints []Evidence
	Warnings    []string
}

// Renderer is the L1/L2-neutral contract used by the engine. Unsupported
// operations return ErrUnsupported and must trigger policy escalation rather
// than silent PASS.
type Renderer interface {
	Render(context.Context, Request) (Evidence, error)
	Inspect(context.Context, InspectRequest) (StructuralEvidence, error)
	CaptureRegion(context.Context, RegionRequest) (Evidence, error)
	RunScenario(context.Context, Scenario) (ScenarioEvidence, error)
	Capabilities() Capabilities
}
