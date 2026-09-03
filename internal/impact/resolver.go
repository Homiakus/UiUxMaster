package impact

import (
	"context"
	"fmt"
	"sort"
)

// Resolver maps changed graph entities to the smallest conservative validation
// scope currently known to UiUxMaster.
type Resolver struct {
	graph *Graph
}

func NewResolver(graph *Graph) (*Resolver, error) {
	if graph == nil {
		return nil, fmt.Errorf("impact: graph is nil")
	}
	return &Resolver{graph: graph}, nil
}

func (r *Resolver) ApplyChanges(ctx context.Context, changes ChangeSet) (ImpactSet, error) {
	if err := ctx.Err(); err != nil {
		return ImpactSet{}, err
	}

	known := make([]string, 0, len(changes.NodeIDs))
	unknown := make([]string, 0)
	seen := make(map[string]struct{}, len(changes.NodeIDs))
	for _, id := range changes.NodeIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if _, ok := r.graph.Node(id); ok {
			known = append(known, id)
		} else {
			unknown = append(unknown, id)
		}
	}
	return r.resolve(ctx, known, unknown, changes.Uncertain, changes.Reasons)
}

func (r *Resolver) ResolveComponent(ctx context.Context, componentID string) (ImpactSet, error) {
	return r.resolveTyped(ctx, componentID, NodeComponent)
}

func (r *Resolver) ResolveToken(ctx context.Context, tokenID string) (ImpactSet, error) {
	return r.resolveTyped(ctx, tokenID, NodeDesignToken)
}

func (r *Resolver) resolveTyped(ctx context.Context, id string, expected NodeKind) (ImpactSet, error) {
	if err := ctx.Err(); err != nil {
		return ImpactSet{}, err
	}
	n, ok := r.graph.Node(id)
	if !ok {
		return r.resolve(ctx, nil, []string{id}, false, nil)
	}
	if n.Kind != expected {
		return ImpactSet{}, fmt.Errorf("impact: node %q has kind %q, want %q", id, n.Kind, expected)
	}
	return r.resolve(ctx, []string{id}, nil, false, nil)
}

func (r *Resolver) resolve(ctx context.Context, known, unknown []string, uncertain bool, reasons []string) (ImpactSet, error) {
	if err := ctx.Err(); err != nil {
		return ImpactSet{}, err
	}

	sort.Strings(known)
	sort.Strings(unknown)
	nodes := r.graph.Reachable(known...)
	out := ImpactSet{
		NodeIDs:    make([]string, 0, len(nodes)),
		UnknownIDs: append([]string(nil), unknown...),
		Reasons:    uniqueSortedStrings(reasons),
		Broad:      len(unknown) > 0 || uncertain,
	}

	for _, n := range nodes {
		out.NodeIDs = append(out.NodeIDs, n.ID)
		switch n.Kind {
		case NodeComponent, NodeComponentVariant, NodeComponentInstance:
			out.ComponentIDs = append(out.ComponentIDs, n.ID)
		case NodeRoute, NodeStory, NodePage:
			out.RouteIDs = append(out.RouteIDs, n.ID)
		case NodeRenderRegion:
			out.RegionIDs = append(out.RegionIDs, n.ID)
		}
	}

	sort.Strings(out.ComponentIDs)
	sort.Strings(out.RouteIDs)
	sort.Strings(out.RegionIDs)
	return out, nil
}

func uniqueSortedStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
