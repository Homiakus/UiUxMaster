package impact

import (
	"context"
	"reflect"
	"testing"
)

func TestIndexProjectPropagatesGlobalTokenThroughCSSAndComponents(t *testing.T) {
	index, err := IndexProject([]ProjectFile{
		{Path: "src/theme.css", Content: []byte(`:root{--radius-control:12px}`)},
		{Path: "src/Button.css", Content: []byte(`.button{border-radius:var(--radius-control)}`)},
		{Path: "src/Button.tsx", Content: []byte(`import './Button.css'; export function Button(){ return <button className="button"/> }`)},
		{Path: "src/App.tsx", Content: []byte(`import {Button} from './Button'; export function App(){ return <Button/> }`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if index.Uncertain {
		t.Fatalf("unexpected uncertainty: %#v", index.Reasons)
	}

	resolver, err := NewResolver(index.Graph)
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolver.ApplyChanges(context.Background(), index.ChangeSetForFiles("src/theme.css"))
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"component:src/App.tsx", "component:src/Button.tsx"}
	if !reflect.DeepEqual(got.ComponentIDs, want) {
		t.Fatalf("components = %#v, want %#v", got.ComponentIDs, want)
	}
	if got.Broad {
		t.Fatalf("known static graph unexpectedly broad: %#v", got.Reasons)
	}
}

func TestIndexProjectUnresolvedDynamicImportExpandsScope(t *testing.T) {
	index, err := IndexProject([]ProjectFile{
		{Path: "src/App.tsx", Content: []byte(`const page = import(resolvePage()); export function App(){ return null }`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !index.Uncertain {
		t.Fatal("expected index uncertainty")
	}
	resolver, err := NewResolver(index.Graph)
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolver.ApplyChanges(context.Background(), index.ChangeSetForFiles("src/App.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Broad {
		t.Fatal("uncertain dynamic dependency must force broad validation")
	}
	if !reflect.DeepEqual(got.Reasons, []string{"dynamic_import_unresolved:src/App.tsx"}) {
		t.Fatalf("reasons = %#v", got.Reasons)
	}
}

func TestIndexProjectMissingRelativeImportExpandsScope(t *testing.T) {
	index, err := IndexProject([]ProjectFile{
		{Path: "src/App.tsx", Content: []byte(`import {Missing} from './Missing'; export function App(){ return <Missing/> }`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !index.Uncertain {
		t.Fatal("expected unresolved relative import uncertainty")
	}
	if !reflect.DeepEqual(index.Reasons, []string{"unresolved_import:src/App.tsx:./Missing"}) {
		t.Fatalf("reasons = %#v", index.Reasons)
	}
}

func TestResolveProjectSpecifierSupportsIndexAndQuerySuffix(t *testing.T) {
	files := map[string][]byte{
		"src/widgets/index.tsx": nil,
		"src/theme.css":          nil,
	}
	got, internal, ok := resolveProjectSpecifier("src/App.tsx", "./widgets?raw", files)
	if !internal || !ok || got != "src/widgets/index.tsx" {
		t.Fatalf("resolved = %q internal=%v ok=%v", got, internal, ok)
	}
	got, internal, ok = resolveProjectSpecifier("src/App.tsx", "./theme.css#layer", files)
	if !internal || !ok || got != "src/theme.css" {
		t.Fatalf("resolved css = %q internal=%v ok=%v", got, internal, ok)
	}
}
