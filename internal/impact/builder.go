package impact

import "fmt"

// Builder provides frontend-specific graph construction helpers. All edges are
// oriented in change-propagation direction: dependency -> dependent.
type Builder struct {
	graph *Graph
}

func NewBuilder() *Builder {
	return &Builder{graph: NewGraph()}
}

func (b *Builder) Graph() *Graph { return b.graph }

func (b *Builder) SourceFile(id string) error {
	return b.graph.AddNode(Node{ID: id, Kind: NodeSourceFile})
}

func (b *Builder) Module(id string) error {
	return b.graph.AddNode(Node{ID: id, Kind: NodeModule})
}

// Import records `importer imports dependency`. Since impact flows from a
// changed dependency toward its dependents, the stored edge is dependency -> importer.
func (b *Builder) Import(importer, dependency string) error {
	if importer == "" || dependency == "" {
		return fmt.Errorf("impact: importer and dependency are required")
	}
	if err := b.Module(importer); err != nil {
		return err
	}
	if err := b.Module(dependency); err != nil {
		return err
	}
	return b.graph.AddEdge(Edge{From: dependency, To: importer, Kind: EdgeImports})
}

// SourceBacksModule connects a source file to the module produced from it.
func (b *Builder) SourceBacksModule(sourceID, moduleID string) error {
	if err := b.SourceFile(sourceID); err != nil {
		return err
	}
	if err := b.Module(moduleID); err != nil {
		return err
	}
	return b.graph.AddEdge(Edge{From: sourceID, To: moduleID, Kind: EdgeDependsOn})
}

func (b *Builder) Component(id string) error {
	return b.graph.AddNode(Node{ID: id, Kind: NodeComponent})
}

// ModuleRendersComponent records that changes in a module may affect a component.
func (b *Builder) ModuleRendersComponent(moduleID, componentID string) error {
	if err := b.Module(moduleID); err != nil {
		return err
	}
	if err := b.Component(componentID); err != nil {
		return err
	}
	return b.graph.AddEdge(Edge{From: moduleID, To: componentID, Kind: EdgeRenders})
}

func (b *Builder) StyleSheet(id string) error {
	return b.graph.AddNode(Node{ID: id, Kind: NodeStyleSheet})
}

// SourceBacksStyle connects a source file to the stylesheet parsed from it.
func (b *Builder) SourceBacksStyle(sourceID, styleID string) error {
	if err := b.SourceFile(sourceID); err != nil {
		return err
	}
	if err := b.StyleSheet(styleID); err != nil {
		return err
	}
	return b.graph.AddEdge(Edge{From: sourceID, To: styleID, Kind: EdgeDependsOn})
}

// StyleImportedByModule records stylesheet -> importing module propagation.
func (b *Builder) StyleImportedByModule(styleID, moduleID string) error {
	if err := b.StyleSheet(styleID); err != nil {
		return err
	}
	if err := b.Module(moduleID); err != nil {
		return err
	}
	return b.graph.AddEdge(Edge{From: styleID, To: moduleID, Kind: EdgeStyles})
}

// StyleComponent records that a stylesheet can affect a component.
func (b *Builder) StyleComponent(styleID, componentID string) error {
	if err := b.StyleSheet(styleID); err != nil {
		return err
	}
	if err := b.Component(componentID); err != nil {
		return err
	}
	return b.graph.AddEdge(Edge{From: styleID, To: componentID, Kind: EdgeStyles})
}

func (b *Builder) DesignToken(id string) error {
	return b.graph.AddNode(Node{ID: id, Kind: NodeDesignToken})
}

// TokenAffects records a token -> consumer relationship. The consumer must
// already have a concrete frontend kind so callers cannot smuggle arbitrary
// generic graph semantics into the impact engine.
func (b *Builder) TokenAffects(tokenID, consumerID string, consumerKind NodeKind) error {
	if err := b.DesignToken(tokenID); err != nil {
		return err
	}
	switch consumerKind {
	case NodeStyleSheet, NodeComponent, NodeComponentVariant:
	default:
		return fmt.Errorf("impact: unsupported token consumer kind %q", consumerKind)
	}
	if err := b.graph.AddNode(Node{ID: consumerID, Kind: consumerKind}); err != nil {
		return err
	}
	return b.graph.AddEdge(Edge{From: tokenID, To: consumerID, Kind: EdgeConsumesToken})
}

// ComponentInstance records component -> rendered instance propagation.
func (b *Builder) ComponentInstance(componentID, instanceID string) error {
	if err := b.Component(componentID); err != nil {
		return err
	}
	if err := b.graph.AddNode(Node{ID: instanceID, Kind: NodeComponentInstance}); err != nil {
		return err
	}
	return b.graph.AddEdge(Edge{From: componentID, To: instanceID, Kind: EdgeInstantiates})
}

// PlaceInstance binds an instance to a page/route-like owner and concrete render region.
func (b *Builder) PlaceInstance(instanceID, pageID, regionID string) error {
	if err := b.graph.AddNode(Node{ID: instanceID, Kind: NodeComponentInstance}); err != nil {
		return err
	}
	if err := b.graph.AddNode(Node{ID: pageID, Kind: NodePage}); err != nil {
		return err
	}
	if err := b.graph.AddNode(Node{ID: regionID, Kind: NodeRenderRegion}); err != nil {
		return err
	}
	if err := b.graph.AddEdge(Edge{From: instanceID, To: pageID, Kind: EdgeAppearsOn}); err != nil {
		return err
	}
	return b.graph.AddEdge(Edge{From: instanceID, To: regionID, Kind: EdgeAffectsRegion})
}
