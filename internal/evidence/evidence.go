package evidence

// Severity is ordered from informational to blocking.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Rect is a browser-space bounding box in CSS pixels.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// DocumentMetrics captures Blink's own document-space metrics instead of
// reconstructing viewport overflow from the widest observed element.
type DocumentMetrics struct {
	FrameID       string  `json:"frame_id,omitempty"`
	URL           string  `json:"url,omitempty"`
	ContentWidth  float64 `json:"content_width"`
	ContentHeight float64 `json:"content_height"`
}

// ElementRef is a stable-enough reference to a rendered element for one
// validation run. Runtime adapters should prefer semantic role/name/test-id
// information over brittle CSS selectors where possible.
type ElementRef struct {
	ID            string            `json:"id"`
	FrameID       string            `json:"frame_id,omitempty"`
	BackendNodeID int64             `json:"backend_node_id,omitempty"`
	Tag           string            `json:"tag,omitempty"`
	Role          string            `json:"role,omitempty"`
	Name          string            `json:"name,omitempty"`
	Selector      string            `json:"selector,omitempty"`
	Bounds        Rect              `json:"bounds"`
	Visible       bool              `json:"visible"`
	Styles        map[string]string `json:"styles,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
	ParentID      string            `json:"parent_id,omitempty"`
	SectionID     string            `json:"section_id,omitempty"`
}

// RuntimeIssue is deterministic browser/runtime evidence such as a page error,
// failed request, layout overflow, missing font, or accessibility-tree anomaly.
type RuntimeIssue struct {
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	Severity   Severity `json:"severity"`
	ElementIDs []string `json:"element_ids,omitempty"`
}

// VisualRegion identifies a screenshot region that changed or requires
// semantic inspection. Pixel-diff and VLM adapters should localize before
// escalating to expensive whole-page analysis.
type VisualRegion struct {
	ID            string   `json:"id"`
	Bounds        Rect     `json:"bounds"`
	ChangedPixels int64    `json:"changed_pixels,omitempty"`
	DiffRatio     float64  `json:"diff_ratio,omitempty"`
	ElementIDs    []string `json:"element_ids,omitempty"`
}

// VisualFinding is a grounded critique result. Source identifies which
// verifier produced the finding (deterministic, pixel_diff, vlm, interaction,
// accessibility, or human/reference review).
type VisualFinding struct {
	ID          string   `json:"id"`
	Axis        string   `json:"axis"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Severity    Severity `json:"severity"`
	Confidence  float64  `json:"confidence"`
	Source      string   `json:"source"`
	RegionID    string   `json:"region_id,omitempty"`
	ElementIDs  []string `json:"element_ids,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	Suggestion  string   `json:"suggestion,omitempty"`
}

// Viewport describes the rendered environment used to collect evidence.
type Viewport struct {
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	DeviceScale float64 `json:"device_scale,omitempty"`
	ColorScheme string  `json:"color_scheme,omitempty"`
	Browser     string  `json:"browser,omitempty"`
}

// RendererRef records which evidence tier and renderer produced the packet.
type RendererRef struct {
	Tier       string `json:"tier,omitempty"`
	Name       string `json:"name,omitempty"`
	Version    string `json:"version,omitempty"`
	FidelityID string `json:"fidelity_id,omitempty"`
}

// PixelEvidence describes in-memory pixel evidence without forcing the hot path
// to persist or embed a large image in the canonical JSON packet. Digest is over
// the decoded RGBA bytes, so it can be used for provenance/deduplication.
type PixelEvidence struct {
	Bounds       Rect   `json:"bounds"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	EncodedBytes int    `json:"encoded_bytes,omitempty"`
	DigestSHA256 string `json:"digest_sha256,omitempty"`
}

// RuntimeLatency keeps the L2 hot-path breakdown visible without coupling the
// canonical packet to Go time.Duration serialization details.
type RuntimeLatency struct {
	WaitEpochMS float64 `json:"wait_epoch_ms,omitempty"`
	SnapshotMS  float64 `json:"snapshot_ms,omitempty"`
	PixelsMS    float64 `json:"pixels_ms,omitempty"`
	TotalMS     float64 `json:"total_ms,omitempty"`
	Retries     int     `json:"retries,omitempty"`
}

// Packet is the canonical evidence envelope passed between browser adapters,
// deterministic verifiers, critics, the engine, MCP tools, CI and future
// persistent design memory.
type Packet struct {
	RunID          string            `json:"run_id"`
	URL            string            `json:"url,omitempty"`
	Scenario       string            `json:"scenario,omitempty"`
	Epoch          uint64            `json:"epoch,omitempty"`
	Viewport       Viewport          `json:"viewport"`
	Renderer       RendererRef       `json:"renderer"`
	Latency        RuntimeLatency    `json:"latency"`
	Documents      []DocumentMetrics `json:"documents,omitempty"`
	Elements       []ElementRef      `json:"elements,omitempty"`
	RuntimeIssues  []RuntimeIssue    `json:"runtime_issues,omitempty"`
	VisualRegions  []VisualRegion    `json:"visual_regions,omitempty"`
	VisualFindings []VisualFinding   `json:"visual_findings,omitempty"`
	Pixels         *PixelEvidence    `json:"pixels,omitempty"`
	AriaSnapshot   string            `json:"aria_snapshot,omitempty"`
	ScreenshotPath string            `json:"screenshot_path,omitempty"`
	DiffPath       string            `json:"diff_path,omitempty"`
}
