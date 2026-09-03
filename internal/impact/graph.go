package impact

import (
	"fmt"
	"sort"
	"sync"
)

// Graph is a small, race-safe directed graph optimized for deterministic
// incremental impact queries. It deliberately stays domain-specific instead of
// becoming a generic graph framework.
type Graph struct {
	mu       sync.RWMutex
	nodes    map[string]Node
	outgoing map[string]map[string]Edge
	incoming map[string]map[string]Edge
}

func NewGraph() *Graph {
	return &Graph{
		nodes:    make(map[string]Node),
		outgoing: make(map[string]map[string]Edge),
		incoming: make(map[string]map[string]Edge),
	}
}

func (g *Graph) AddNode(n Node) error {
	if n.ID == "" {
		return fmt.Errorf("impact: node id is empty")
	}
	if n.Kind == "" {
		return fmt.Errorf("impact: node %q kind is empty", n.ID)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if prev, ok := g.nodes[n.ID]; ok {
		if prev != n {
			return fmt.Errorf("impact: conflicting node id %q", n.ID)
		}
		return nil
	}
	g.nodes[n.ID] = n
	return nil
}

func (g *Graph) AddEdge(e Edge) error {
	if e.From == "" || e.To == "" || e.Kind == "" {
		return fmt.Errorf("impact: edge requires from, to and kind")
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.nodes[e.From]; !ok {
		return fmt.Errorf("impact: edge source %q does not exist", e.From)
	}
	if _, ok := g.nodes[e.To]; !ok {
		return fmt.Errorf("impact: edge target %q does not exist", e.To)
	}

	key := edgeKey(e)
	if g.outgoing[e.From] == nil {
		g.outgoing[e.From] = make(map[string]Edge)
	}
	if g.incoming[e.To] == nil {
		g.incoming[e.To] = make(map[string]Edge)
	}
	g.outgoing[e.From][key] = e
	g.incoming[e.To][key] = e
	return nil
}

func (g *Graph) Node(id string) (Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[id]
	return n, ok
}

// Reachable returns the start nodes plus all nodes reachable by outgoing impact
// edges. The result is sorted by stable node ID, independent of map order.
func (g *Graph) Reachable(startIDs ...string) []Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	seen := make(map[string]struct{}, len(startIDs))
	queue := make([]string, 0, len(startIDs))
	for _, id := range startIDs {
		if _, ok := g.nodes[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		queue = append(queue, id)
	}

	for head := 0; head < len(queue); head++ {
		id := queue[head]
		neighbors := make([]string, 0, len(g.outgoing[id]))
		for _, e := range g.outgoing[id] {
			neighbors = append(neighbors, e.To)
		}
		sort.Strings(neighbors)
		for _, next := range neighbors {
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			queue = append(queue, next)
		}
	}

	out := make([]Node, 0, len(seen))
	for id := range seen {
		out = append(out, g.nodes[id])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (g *Graph) Snapshot() Snapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	edges := make([]Edge, 0)
	for _, set := range g.outgoing {
		for _, e := range set {
			edges = append(edges, e)
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Kind < edges[j].Kind
	})
	return Snapshot{Nodes: nodes, Edges: edges}
}

func edgeKey(e Edge) string {
	return e.From + "\x00" + string(e.Kind) + "\x00" + e.To
}
