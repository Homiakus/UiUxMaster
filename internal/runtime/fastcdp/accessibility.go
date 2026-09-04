package fastcdp

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type AXNode struct {
	ID               string
	ParentID         string
	ChildIDs         []string
	BackendDOMNodeID int64
	FrameID          string
	Ignored          bool
	IgnoredReasons   []string
	Role             string
	Name             string
	Description      string
	Value            string
	Properties       map[string]string
}

type AXTree struct {
	Nodes []AXNode
}

type axValue struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type axProperty struct {
	Name  string  `json:"name"`
	Value axValue `json:"value"`
}

type rawAXNode struct {
	NodeID           string       `json:"nodeId"`
	Ignored          bool         `json:"ignored"`
	IgnoredReasons   []axProperty `json:"ignoredReasons"`
	Role             axValue      `json:"role"`
	Name             axValue      `json:"name"`
	Description      axValue      `json:"description"`
	Value            axValue      `json:"value"`
	Properties       []axProperty `json:"properties"`
	ParentID         string       `json:"parentId"`
	ChildIDs         []string     `json:"childIds"`
	BackendDOMNodeID int64        `json:"backendDOMNodeId"`
	FrameID          string       `json:"frameId"`
}

// CaptureAXTree fetches the accessibility tree without keeping the
// Accessibility domain enabled between captures. This avoids the persistent
// performance cost documented for Accessibility.enable while still providing
// deterministic pull-based evidence.
func (c *Connection) CaptureAXTree(ctx context.Context, sessionID string, depth int) (AXTree, error) {
	params := map[string]any{}
	if depth > 0 {
		params["depth"] = depth
	}
	var result struct {
		Nodes []rawAXNode `json:"nodes"`
	}
	if err := c.Call(ctx, sessionID, "Accessibility.getFullAXTree", params, &result); err != nil {
		return AXTree{}, err
	}
	out := AXTree{Nodes: make([]AXNode, 0, len(result.Nodes))}
	for _, raw := range result.Nodes {
		node := AXNode{
			ID:               raw.NodeID,
			ParentID:         raw.ParentID,
			ChildIDs:         append([]string(nil), raw.ChildIDs...),
			BackendDOMNodeID: raw.BackendDOMNodeID,
			FrameID:          raw.FrameID,
			Ignored:          raw.Ignored,
			Role:             axString(raw.Role),
			Name:             axString(raw.Name),
			Description:      axString(raw.Description),
			Value:            axString(raw.Value),
			Properties:       projectAXProperties(raw.Properties),
		}
		for _, reason := range raw.IgnoredReasons {
			text := reason.Name
			if value := axString(reason.Value); value != "" {
				text += "=" + value
			}
			if text != "" {
				node.IgnoredReasons = append(node.IgnoredReasons, text)
			}
		}
		sort.Strings(node.IgnoredReasons)
		out.Nodes = append(out.Nodes, node)
	}
	return out, nil
}

func projectAXProperties(properties []axProperty) map[string]string {
	if len(properties) == 0 {
		return nil
	}
	out := make(map[string]string, len(properties))
	for _, property := range properties {
		name := strings.TrimSpace(property.Name)
		if name == "" {
			continue
		}
		out[name] = axString(property.Value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func axString(value axValue) string {
	return strings.TrimSpace(stringify(value.Value))
}

// TextSnapshot creates a compact deterministic accessibility outline for MCP
// consumers and human diagnostics. Structured AX nodes remain the source of
// truth for machine verification.
func (tree AXTree) TextSnapshot() string {
	if len(tree.Nodes) == 0 {
		return ""
	}
	byID := make(map[string]AXNode, len(tree.Nodes))
	roots := make([]string, 0)
	for _, node := range tree.Nodes {
		byID[node.ID] = node
		if node.ParentID == "" {
			roots = append(roots, node.ID)
		}
	}
	sort.Strings(roots)
	var lines []string
	visited := make(map[string]bool, len(tree.Nodes))
	var walk func(string, int)
	walk = func(id string, depth int) {
		if visited[id] {
			return
		}
		visited[id] = true
		node, ok := byID[id]
		if !ok {
			return
		}
		if !node.Ignored {
			label := node.Role
			if label == "" {
				label = "node"
			}
			if node.Name != "" {
				label += ": " + node.Name
			}
			lines = append(lines, strings.Repeat("  ", depth)+"- "+label)
		}
		children := append([]string(nil), node.ChildIDs...)
		sort.Strings(children)
		for _, child := range children {
			walk(child, depth+1)
		}
	}
	for _, root := range roots {
		walk(root, 0)
	}
	// Some protocol snapshots can contain detached nodes. Keep them visible
	// rather than silently losing evidence.
	remaining := make([]string, 0)
	for id := range byID {
		if !visited[id] {
			remaining = append(remaining, id)
		}
	}
	sort.Strings(remaining)
	for _, id := range remaining {
		walk(id, 0)
	}
	return strings.Join(lines, "\n")
}

func (node AXNode) String() string {
	return fmt.Sprintf("AXNode{%s role=%q name=%q ignored=%t backend=%d}", node.ID, node.Role, node.Name, node.Ignored, node.BackendDOMNodeID)
}
