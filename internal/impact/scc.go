package impact

import "sort"

// StronglyConnectedComponents returns deterministic Tarjan SCCs. Each
// component is sorted by node ID; the component list is sorted by its first ID.
func (g *Graph) StronglyConnectedComponents() [][]string {
	snapshot := g.Snapshot()
	adj := make(map[string][]string, len(snapshot.Nodes))
	for _, n := range snapshot.Nodes {
		adj[n.ID] = nil
	}
	for _, e := range snapshot.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}
	for id := range adj {
		sort.Strings(adj[id])
	}

	index := 0
	indices := make(map[string]int, len(snapshot.Nodes))
	lowlink := make(map[string]int, len(snapshot.Nodes))
	onStack := make(map[string]bool, len(snapshot.Nodes))
	stack := make([]string, 0, len(snapshot.Nodes))
	for _, n := range snapshot.Nodes {
		indices[n.ID] = -1
	}

	components := make([][]string, 0)
	var visit func(string)
	visit = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range adj[v] {
			if indices[w] == -1 {
				visit(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] && indices[w] < lowlink[v] {
				lowlink[v] = indices[w]
			}
		}

		if lowlink[v] != indices[v] {
			return
		}

		component := make([]string, 0, 1)
		for {
			last := len(stack) - 1
			w := stack[last]
			stack = stack[:last]
			onStack[w] = false
			component = append(component, w)
			if w == v {
				break
			}
		}
		sort.Strings(component)
		components = append(components, component)
	}

	for _, n := range snapshot.Nodes {
		if indices[n.ID] == -1 {
			visit(n.ID)
		}
	}

	sort.Slice(components, func(i, j int) bool {
		if len(components[i]) == 0 {
			return true
		}
		if len(components[j]) == 0 {
			return false
		}
		return components[i][0] < components[j][0]
	})
	return components
}
