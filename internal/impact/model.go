package impact

// NodeKind identifies a frontend-specific entity in the incremental impact graph.
type NodeKind string

const (
	NodeSourceFile       NodeKind = "source_file"
	NodeModule           NodeKind = "module"
	NodeStyleSheet       NodeKind = "style_sheet"
	NodeDesignToken      NodeKind = "design_token"
	NodeComponent        NodeKind = "component"
	NodeComponentVariant NodeKind = "component_variant"
	NodeComponentInstance NodeKind = "component_instance"
	NodeRoute            NodeKind = "route"
	NodeStory            NodeKind = "story"
	NodePage             NodeKind = "page"
	NodeSemanticElement  NodeKind = "semantic_element"
	NodeRenderRegion     NodeKind = "render_region"
	NodeViewport         NodeKind = "viewport"
	NodeTheme            NodeKind = "theme"
	NodeScenario         NodeKind = "scenario"
)

// EdgeKind identifies a dependency/impact relationship. Edges point in the
// direction in which a change can propagate.
type EdgeKind string

const (
	EdgeImports             EdgeKind = "imports"
	EdgeStyles              EdgeKind = "styles"
	EdgeConsumesToken       EdgeKind = "consumes_token"
	EdgeRenders             EdgeKind = "renders"
	EdgeInstantiates        EdgeKind = "instantiates"
	EdgeAppearsOn           EdgeKind = "appears_on"
	EdgeMapsToRuntimeRef    EdgeKind = "maps_to_runtime_ref"
	EdgeAffectsRegion       EdgeKind = "affects_region"
	EdgeParticipatesScenario EdgeKind = "participates_in_scenario"
	EdgeDependsOn           EdgeKind = "depends_on"
)

// Node is a stable graph entity. ID must be deterministic across equivalent
// analyses of the same project state.
type Node struct {
	ID    string   `json:"id"`
	Kind  NodeKind `json:"kind"`
	Label string   `json:"label,omitempty"`
}

// Edge is a directed impact relationship from From to To.
type Edge struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Kind EdgeKind `json:"kind"`
}

// Snapshot is a deterministic, serialization-friendly graph view.
type Snapshot struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// ChangeSet is the smallest input accepted by the first impact resolver.
// Source analyzers will translate file/token/component changes into these IDs.
type ChangeSet struct {
	NodeIDs []string `json:"node_ids"`
}

// ImpactSet is the conservative validation scope derived from a change.
type ImpactSet struct {
	NodeIDs      []string `json:"node_ids"`
	ComponentIDs []string `json:"component_ids,omitempty"`
	RouteIDs     []string `json:"route_ids,omitempty"`
	RegionIDs    []string `json:"region_ids,omitempty"`
	UnknownIDs   []string `json:"unknown_ids,omitempty"`
	Broad        bool     `json:"broad"`
}
