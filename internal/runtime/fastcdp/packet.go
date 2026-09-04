package fastcdp

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// DefaultEvidenceStyles is deliberately small: these properties are sufficient
// for deterministic geometry/visibility/interaction checks without turning
// DOMSnapshot into a full computed-style dump.
var DefaultEvidenceStyles = []string{
	"display",
	"position",
	"overflow-x",
	"overflow-y",
	"visibility",
	"opacity",
	"pointer-events",
	"cursor",
	"z-index",
	"font-size",
	"font-weight",
	"line-height",
}

func DefaultSnapshotOptions() SnapshotOptions {
	return SnapshotOptions{
		ComputedStyles:  append([]string(nil), DefaultEvidenceStyles...),
		IncludeDOMRects: true,
	}
}

type PacketOptions struct {
	RunID      string
	Scenario   string
	URL        string
	Viewport   evidence.Viewport
	Browser    BrowserVersion
	FidelityID string
	Region     *CaptureRegionOptions
}

// ToPacket converts one stable CollectedEvidence result into the repository's
// protocol-independent evidence envelope. Large RGBA bytes remain in the local
// hot path; Packet stores only their dimensions, provenance digest and source
// ROI metadata.
func ToPacket(collected CollectedEvidence, options PacketOptions) evidence.Packet {
	packet := evidence.Packet{
		RunID:    options.RunID,
		URL:      options.URL,
		Scenario: options.Scenario,
		Epoch:    collected.Epoch,
		Viewport: options.Viewport,
		Renderer: evidence.RendererRef{
			Tier:       "L2",
			Name:       "fastcdp",
			Version:    options.Browser.Product,
			FidelityID: options.FidelityID,
		},
		Latency: evidence.RuntimeLatency{
			WaitEpochMS: durationMS(collected.Timing.WaitEpoch),
			SnapshotMS:  durationMS(collected.Timing.Snapshot),
			PixelsMS:    durationMS(collected.Timing.Pixels),
			TotalMS:     durationMS(collected.Timing.Total),
			Retries:     collected.Timing.Retries,
		},
	}
	if packet.Viewport.Browser == "" {
		packet.Viewport.Browser = options.Browser.Product
	}

	if collected.Snapshot != nil {
		projectSnapshotIntoPacket(&packet, *collected.Snapshot)
	}
	if packet.URL == "" && len(packet.Documents) > 0 {
		packet.URL = packet.Documents[0].URL
	}
	if collected.RGBA != nil {
		bounds := evidence.Rect{}
		if options.Region != nil {
			bounds = evidence.Rect{
				X: options.Region.X, Y: options.Region.Y,
				Width: options.Region.Width, Height: options.Region.Height,
			}
		}
		sum := sha256.Sum256(collected.RGBA.Pix)
		packet.Pixels = &evidence.PixelEvidence{
			Bounds:       bounds,
			Width:        collected.RGBA.Bounds().Dx(),
			Height:       collected.RGBA.Bounds().Dy(),
			EncodedBytes: collected.CaptureStats.EncodedBytes,
			DigestSHA256: fmt.Sprintf("%x", sum[:]),
		}
	}
	return packet
}

func projectSnapshotIntoPacket(packet *evidence.Packet, snapshot Snapshot) {
	for docIndex, doc := range snapshot.Documents {
		frameID := doc.FrameID
		if frameID == "" {
			frameID = fmt.Sprintf("document-%d", docIndex)
		}
		packet.Documents = append(packet.Documents, evidence.DocumentMetrics{
			FrameID:       frameID,
			URL:           doc.DocumentURL,
			ContentWidth:  doc.ContentWidth,
			ContentHeight: doc.ContentHeight,
		})

		idByNodeIndex := make(map[int]string, len(doc.Nodes))
		for _, node := range doc.Nodes {
			if node.NodeType != 1 {
				continue
			}
			idByNodeIndex[node.NodeIndex] = elementID(frameID, node)
		}
		for _, node := range doc.Nodes {
			if node.NodeType != 1 {
				continue
			}
			attrs := cloneStrings(node.Attributes)
			styles := cloneStrings(node.Styles)
			ref := evidence.ElementRef{
				ID:            idByNodeIndex[node.NodeIndex],
				FrameID:       frameID,
				BackendNodeID: node.BackendNodeID,
				Tag:           strings.ToLower(node.Name),
				Role:          semanticRole(node.Name, attrs),
				Name:          semanticName(node, attrs),
				Selector:      semanticSelector(attrs),
				Bounds: evidence.Rect{
					X: node.Bounds.X, Y: node.Bounds.Y,
					Width: node.Bounds.Width, Height: node.Bounds.Height,
				},
				Visible:    visuallyVisible(node.Bounds, styles),
				Clickable:  node.Clickable,
				Styles:     styles,
				Attributes: attrs,
				ParentID:   idByNodeIndex[node.ParentIndex],
			}
			packet.Elements = append(packet.Elements, ref)
		}
	}
}

func elementID(frameID string, node SnapshotNode) string {
	if node.BackendNodeID > 0 {
		return fmt.Sprintf("dom:%s:%d", frameID, node.BackendNodeID)
	}
	return fmt.Sprintf("dom:%s:node-%d", frameID, node.NodeIndex)
}

func semanticRole(tag string, attrs map[string]string) string {
	if role := strings.ToLower(strings.TrimSpace(attrs["role"])); role != "" {
		return role
	}
	tag = strings.ToLower(tag)
	switch tag {
	case "button":
		return "button"
	case "a":
		if attrs["href"] != "" {
			return "link"
		}
	case "textarea":
		return "textbox"
	case "select":
		return "combobox"
	case "img":
		return "img"
	case "nav":
		return "navigation"
	case "main":
		return "main"
	case "input":
		switch strings.ToLower(attrs["type"]) {
		case "button", "submit", "reset", "image":
			return "button"
		case "checkbox":
			return "checkbox"
		case "radio":
			return "radio"
		case "range":
			return "slider"
		case "hidden":
			return ""
		default:
			return "textbox"
		}
	}
	return ""
}

func semanticName(node SnapshotNode, attrs map[string]string) string {
	for _, key := range []string{"aria-label", "alt", "title", "placeholder"} {
		if value := normalizeWhitespace(attrs[key]); value != "" {
			return value
		}
	}
	if strings.EqualFold(node.Name, "input") {
		if value := normalizeWhitespace(attrs["value"]); value != "" {
			return value
		}
	}
	if value := normalizeWhitespace(node.Text); value != "" {
		return value
	}
	return normalizeWhitespace(node.Value)
}

var simpleIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

func semanticSelector(attrs map[string]string) string {
	if testID := attrs["data-testid"]; testID != "" {
		return `[data-testid="` + escapeAttributeValue(testID) + `"]`
	}
	if id := attrs["id"]; id != "" {
		if simpleIdentifier.MatchString(id) {
			return "#" + id
		}
		return `[id="` + escapeAttributeValue(id) + `"]`
	}
	return ""
}

func escapeAttributeValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func visuallyVisible(bounds Rect, styles map[string]string) bool {
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return false
	}
	if strings.EqualFold(styles["display"], "none") {
		return false
	}
	switch strings.ToLower(styles["visibility"]) {
	case "hidden", "collapse":
		return false
	}
	if opacity := strings.TrimSpace(styles["opacity"]); opacity != "" {
		if parsed, err := strconv.ParseFloat(opacity, 64); err == nil && parsed <= 0.001 {
			return false
		}
	}
	return true
}

func cloneStrings(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func durationMS(value time.Duration) float64 {
	return float64(value.Nanoseconds()) / 1e6
}
