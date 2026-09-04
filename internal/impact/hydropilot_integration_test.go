package impact

import (
	"context"
	"os"
	"testing"
)

func TestHydropilotProjectIngestion(t *testing.T) {
	hydropilotPath := `D:\Programms\hydropilot`
	if _, err := os.Stat(hydropilotPath); os.IsNotExist(err) {
		t.Skip("hydropilot directory not found at " + hydropilotPath)
	}

	index, err := IndexDirectory(hydropilotPath)
	if err != nil {
		t.Fatalf("failed to index hydropilot project: %v", err)
	}

	if index == nil || index.Graph == nil {
		t.Fatal("expected non-nil index and graph")
	}

	snap := index.Graph.Snapshot()
	t.Logf("Hydropilot index built: nodes=%d, edges=%d, uncertain=%v, reasons=%v",
		len(snap.Nodes), len(snap.Edges), index.Uncertain, index.Reasons)

	resolver, err := NewResolver(index.Graph)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	// Test resolving a sample file change
	cs := index.ChangeSetForFiles("README.md", "architecture.md")
	impactSet, err := resolver.ApplyChanges(context.Background(), cs)
	if err != nil {
		t.Fatalf("resolver failed on hydropilot changes: %v", err)
	}

	t.Logf("Hydropilot impact resolution: component_count=%d, route_count=%d, broad=%v",
		len(impactSet.ComponentIDs), len(impactSet.RouteIDs), impactSet.Broad)
}
