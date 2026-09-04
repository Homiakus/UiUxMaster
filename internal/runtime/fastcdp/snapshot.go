package fastcdp

import (
	"context"
	"fmt"
	"strings"
)

// Rect is a document-coordinate layout rectangle returned by Blink.
type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// SnapshotNode is the compact verifier-oriented projection of a CDP
// DOMSnapshot layout node. BackendNodeID is stable enough to address the node
// through other DOM-domain commands within the resident document lifetime.
type SnapshotNode struct {
	DocumentIndex int
	NodeIndex     int
	ParentIndex   int
	NodeType      int
	BackendNodeID int64
	Name          string
	Value         string
	Text          string
	Bounds        Rect
	Styles        map[string]string
	Attributes    map[string]string
	Clickable     bool
}

// DocumentSnapshot contains only fields needed by impact correlation and
// deterministic geometry/style verifiers.
type DocumentSnapshot struct {
	FrameID       string
	DocumentURL   string
	ContentWidth  float64
	ContentHeight float64
	Nodes         []SnapshotNode
}

type Snapshot struct {
	ComputedStyles []string
	Documents      []DocumentSnapshot
}

type SnapshotOptions struct {
	ComputedStyles  []string
	IncludeDOMRects bool
	IncludePaint    bool
}

type captureSnapshotParams struct {
	ComputedStyles                 []string `json:"computedStyles"`
	IncludePaintOrder              bool     `json:"includePaintOrder"`
	IncludeDOMRects                bool     `json:"includeDOMRects"`
	IncludeBlendedBackgroundColors bool     `json:"includeBlendedBackgroundColors,omitempty"`
	IncludeTextColorOpacities      bool     `json:"includeTextColorOpacities,omitempty"`
}

type captureSnapshotResult struct {
	Documents []captureDocument `json:"documents"`
	Strings   []string          `json:"strings"`
}

type captureDocument struct {
	DocumentURL   int              `json:"documentURL"`
	FrameID       int              `json:"frameId"`
	ContentWidth  float64          `json:"contentWidth"`
	ContentHeight float64          `json:"contentHeight"`
	Nodes         captureNodeTable `json:"nodes"`
	Layout        captureLayout    `json:"layout"`
}

type rareBooleanData struct {
	Index []int `json:"index"`
}

type captureNodeTable struct {
	ParentIndex   []int           `json:"parentIndex"`
	NodeType      []int           `json:"nodeType"`
	NodeName      []int           `json:"nodeName"`
	NodeValue     []int           `json:"nodeValue"`
	BackendNodeID []int64         `json:"backendNodeId"`
	Attributes    [][]int         `json:"attributes"`
	IsClickable   rareBooleanData `json:"isClickable"`
}

type captureLayout struct {
	NodeIndex []int       `json:"nodeIndex"`
	Styles    [][]int     `json:"styles"`
	Bounds    [][]float64 `json:"bounds"`
	Text      []int       `json:"text"`
}

// CaptureSnapshot calls DOMSnapshot.captureSnapshot with a strict computed-style
// whitelist and projects the protocol's columnar tables into verifier-friendly
// nodes. Full raw DOM/CSS state is deliberately not retained.
func (c *Connection) CaptureSnapshot(ctx context.Context, sessionID string, options SnapshotOptions) (Snapshot, error) {
	styles := normalizeStyleWhitelist(options.ComputedStyles)
	params := captureSnapshotParams{
		ComputedStyles:    styles,
		IncludePaintOrder: options.IncludePaint,
		IncludeDOMRects:   options.IncludeDOMRects,
	}
	var raw captureSnapshotResult
	if err := c.Call(ctx, sessionID, "DOMSnapshot.captureSnapshot", params, &raw); err != nil {
		return Snapshot{}, err
	}
	return projectSnapshot(raw, styles)
}

func projectSnapshot(raw captureSnapshotResult, styleNames []string) (Snapshot, error) {
	out := Snapshot{ComputedStyles: append([]string(nil), styleNames...), Documents: make([]DocumentSnapshot, 0, len(raw.Documents))}
	for documentIndex, doc := range raw.Documents {
		clickable := indexSet(doc.Nodes.IsClickable.Index)
		projected := DocumentSnapshot{
			FrameID:       stringAt(raw.Strings, doc.FrameID),
			DocumentURL:   stringAt(raw.Strings, doc.DocumentURL),
			ContentWidth:  doc.ContentWidth,
			ContentHeight: doc.ContentHeight,
			Nodes:         make([]SnapshotNode, 0, len(doc.Layout.NodeIndex)),
		}
		for layoutIndex, nodeIndex := range doc.Layout.NodeIndex {
			if nodeIndex < 0 || nodeIndex >= len(doc.Nodes.NodeName) {
				return Snapshot{}, fmt.Errorf("fastcdp: document %d layout %d references invalid node index %d", documentIndex, layoutIndex, nodeIndex)
			}
			bounds, err := rectAt(doc.Layout.Bounds, layoutIndex)
			if err != nil {
				return Snapshot{}, fmt.Errorf("fastcdp: document %d layout %d: %w", documentIndex, layoutIndex, err)
			}
			node := SnapshotNode{
				DocumentIndex: documentIndex,
				NodeIndex:     nodeIndex,
				ParentIndex:   intAt(doc.Nodes.ParentIndex, nodeIndex, -1),
				NodeType:      intAt(doc.Nodes.NodeType, nodeIndex, 0),
				BackendNodeID: int64At(doc.Nodes.BackendNodeID, nodeIndex, 0),
				Name:          stringAt(raw.Strings, intAt(doc.Nodes.NodeName, nodeIndex, -1)),
				Value:         stringAt(raw.Strings, intAt(doc.Nodes.NodeValue, nodeIndex, -1)),
				Text:          stringAt(raw.Strings, intAt(doc.Layout.Text, layoutIndex, -1)),
				Bounds:        bounds,
				Attributes:    projectAttributes(raw.Strings, intSliceAt(doc.Nodes.Attributes, nodeIndex)),
				Clickable:     clickable[nodeIndex],
			}
			if len(styleNames) > 0 {
				node.Styles = make(map[string]string, len(styleNames))
				styleIndexes := intSliceAt(doc.Layout.Styles, layoutIndex)
				for i, property := range styleNames {
					if i < len(styleIndexes) {
						node.Styles[property] = stringAt(raw.Strings, styleIndexes[i])
					}
			}
			projected.Nodes = append(projected.Nodes, node)
		}
		out.Documents = append(out.Documents, projected)
	}
	return out, nil
}

func projectAttributes(stringsTable []string, indexes []int) map[string]string {
	if len(indexes) < 2 {
		return nil
	}
	attrs := make(map[string]string, len(indexes)/2)
	for i := 0; i+1 < len(indexes); i += 2 {
		name := strings.ToLower(strings.TrimSpace(stringAt(stringsTable, indexes[i])))
		if name == "" {
			continue
		}
		attrs[name] = stringAt(stringsTable, indexes[i+1])
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func indexSet(indexes []int) map[int]bool {
	if len(indexes) == 0 {
		return nil
	}
	out := make(map[int]bool, len(indexes))
	for _, index := range indexes {
		if index >= 0 {
			out[index] = true
		}
	}
	return out
}

func normalizeStyleWhitelist(styles []string) []string {
	seen := make(map[string]struct{}, len(styles))
	out := make([]string, 0, len(styles))
	for _, style := range styles {
		style = strings.TrimSpace(style)
		if style == "" {
			continue
		}
		if _, ok := seen[style]; ok {
			continue
		}
		seen[style] = struct{}{}
		out = append(out, style)
	}
	return out
}

func rectAt(bounds [][]float64, index int) (Rect, error) {
	if index < 0 || index >= len(bounds) {
		return Rect{}, fmt.Errorf("missing bounds")
	}
	values := bounds[index]
	if len(values) != 4 {
		return Rect{}, fmt.Errorf("bounds length %d, want 4", len(values))
	}
	return Rect{X: values[0], Y: values[1], Width: values[2], Height: values[3]}, nil
}

func stringAt(stringsTable []string, index int) string {
	if index < 0 || index >= len(stringsTable) {
		return ""
	}
	return stringsTable[index]
}

func intAt(values []int, index, fallback int) int {
	if index < 0 || index >= len(values) {
		return fallback
	}
	return values[index]
}

func int64At(values []int64, index int, fallback int64) int64 {
	if index < 0 || index >= len(values) {
		return fallback
	}
	return values[index]
}

func intSliceAt(values [][]int, index int) []int {
	if index < 0 || index >= len(values) {
		return nil
	}
	return values[index]
}
