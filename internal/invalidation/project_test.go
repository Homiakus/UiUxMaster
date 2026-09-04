package invalidation

import (
	"context"
	"reflect"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/impact"
)

func TestResolveProjectScope_WiredFromProjectFiles(t *testing.T) {
	files := []impact.ProjectFile{
		{
			Path:    "src/Button.css",
			Content: []byte(`.btn { color: var(--color-primary); }`),
		},
		{
			Path: "src/Button.tsx",
			Content: []byte(`
				import "./Button.css";
				export function Button() { return <button className="btn">Click</button>; }
			`),
		},
		{
			Path: "src/Home.tsx",
			Content: []byte(`
				import { Button } from "./Button";
				export function Home() { return <div><Button /></div>; }
			`),
		},
	}

	index, err := impact.IndexProject(files)
	if err != nil {
		t.Fatalf("IndexProject error: %v", err)
	}

	policy := DefaultPolicy()
	scope, err := ResolveProjectScope(context.Background(), index, []string{"src/Button.css"}, policy, Options{})
	if err != nil {
		t.Fatalf("ResolveProjectScope error: %v", err)
	}

	// Changing Button.css must impact component:src/Button.tsx and component:src/Home.tsx
	wantComponents := []string{"component:src/Button.tsx", "component:src/Home.tsx"}
	if !reflect.DeepEqual(scope.Components, wantComponents) {
		t.Errorf("Components = %v, want %v", scope.Components, wantComponents)
	}
}

func TestResolveProjectScope_DynamicImportCausesUncertaintyWidening(t *testing.T) {
	files := []impact.ProjectFile{
		{
			Path: "src/Dynamic.tsx",
			Content: []byte(`
				const mod = import("./dynamic/" + name);
			`),
		},
	}

	index, err := impact.IndexProject(files)
	if err != nil {
		t.Fatalf("IndexProject error: %v", err)
	}

	if !index.Uncertain {
		t.Fatalf("expected index.Uncertain = true for dynamic import")
	}

	policy := DefaultPolicy()
	scope, err := ResolveProjectScope(context.Background(), index, []string{"src/Dynamic.tsx"}, policy, Options{})
	if err != nil {
		t.Fatalf("ResolveProjectScope error: %v", err)
	}

	if !scope.WholeSite {
		t.Errorf("expected WholeSite = true due to dynamic import uncertainty")
	}
	if !scope.Widened {
		t.Errorf("expected Widened = true")
	}

	hasReason := false
	for _, r := range scope.WideningReasons {
		if r == string(ReasonUnknownDependency) {
			hasReason = true
			break
		}
	}
	if !hasReason {
		t.Errorf("expected ReasonUnknownDependency in %v", scope.WideningReasons)
	}
}
