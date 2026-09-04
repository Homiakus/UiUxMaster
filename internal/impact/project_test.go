package impact

import (
	"context"
	"os"
	"path/filepath"
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

func TestIndexProjectRouteDiscoveryAndComponentPropagation(t *testing.T) {
	index, err := IndexProject([]ProjectFile{
		{Path: "src/components/Header.tsx", Content: []byte(`export function Header(){ return <header/> }`)},
		{Path: "src/components/Sidebar.tsx", Content: []byte(`export function Sidebar(){ return <aside/> }`)},
		{Path: "src/app/page.tsx", Content: []byte(`import {Header} from '../components/Header'; export default function HomePage(){ return <Header/> }`)},
		{Path: "src/app/dashboard/page.tsx", Content: []byte(`import {Header} from '../../components/Header'; import {Sidebar} from '../../components/Sidebar'; export default function DashPage(){ return <div><Header/><Sidebar/></div> }`)},
		{Path: "src/app/settings/page.tsx", Content: []byte(`import {Sidebar} from '../../components/Sidebar'; export default function SettingsPage(){ return <Sidebar/> }`)},
	})
	if err != nil {
		t.Fatal(err)
	}

	resolver, err := NewResolver(index.Graph)
	if err != nil {
		t.Fatal(err)
	}

	// Changing Header affects HomePage (route:/) and DashPage (route:/dashboard), but NOT SettingsPage (route:/settings)
	gotHeader, err := resolver.ApplyChanges(context.Background(), index.ChangeSetForFiles("src/components/Header.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	wantRoutesHeader := []string{"route:/", "route:/dashboard"}
	if !reflect.DeepEqual(gotHeader.RouteIDs, wantRoutesHeader) {
		t.Fatalf("routes for Header = %#v, want %#v", gotHeader.RouteIDs, wantRoutesHeader)
	}

	// Changing Sidebar affects DashPage (route:/dashboard) and SettingsPage (route:/settings), but NOT HomePage (route:/)
	gotSidebar, err := resolver.ApplyChanges(context.Background(), index.ChangeSetForFiles("src/components/Sidebar.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	wantRoutesSidebar := []string{"route:/dashboard", "route:/settings"}
	if !reflect.DeepEqual(gotSidebar.RouteIDs, wantRoutesSidebar) {
		t.Fatalf("routes for Sidebar = %#v, want %#v", gotSidebar.RouteIDs, wantRoutesSidebar)
	}
}

func TestIndexProjectHTMLAndLayoutPropagation(t *testing.T) {
	index, err := IndexProject([]ProjectFile{
		{Path: "index.html", Content: []byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<link rel="stylesheet" href="./src/styles.css">
				<style>:root { --brand-color: #0066cc; }</style>
			</head>
			<body>
				<div id="app"></div>
				<script src="./src/main.ts"></script>
			</body>
			</html>
		`)},
		{Path: "src/styles.css", Content: []byte(`body { color: var(--brand-color); }`)},
		{Path: "src/main.ts", Content: []byte(`console.log("ready");`)},
	})
	if err != nil {
		t.Fatal(err)
	}

	resolver, err := NewResolver(index.Graph)
	if err != nil {
		t.Fatal(err)
	}

	// Changing src/styles.css propagates to route:/
	gotStyle, err := resolver.ApplyChanges(context.Background(), index.ChangeSetForFiles("src/styles.css"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotStyle.RouteIDs, []string{"route:/"}) {
		t.Fatalf("style change routes = %#v, want ['route:/']", gotStyle.RouteIDs)
	}

	// Changing src/main.ts propagates to route:/
	gotScript, err := resolver.ApplyChanges(context.Background(), index.ChangeSetForFiles("src/main.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotScript.RouteIDs, []string{"route:/"}) {
		t.Fatalf("script change routes = %#v, want ['route:/']", gotScript.RouteIDs)
	}
}

func TestIndexDirectoryLiveDiskIngestion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uiux-live-proj-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Build a mock Vite / React app structure on disk
	mustWrite := func(relPath, content string) {
		full := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("index.html", `<!DOCTYPE html><html><head><link rel="stylesheet" href="./src/index.css"></head><body><script src="./src/main.tsx"></script></body></html>`)
	mustWrite("src/index.css", `:root { --primary-hue: 210; } .app { color: hsl(var(--primary-hue), 100%, 50%); }`)
	mustWrite("src/Button.tsx", `import React from 'react'; export function Button(){ return <button>Click</button>; }`)
	mustWrite("src/main.tsx", `import React from 'react'; import {Button} from './Button'; export function App(){ return <Button/>; }`)
	mustWrite("node_modules/fake/index.js", `module.exports = {}`) // should be ignored

	index, err := IndexDirectory(tmpDir)
	if err != nil {
		t.Fatalf("IndexDirectory failed: %v", err)
	}

	if index.Uncertain {
		t.Fatalf("unexpected index uncertainty: %#v", index.Reasons)
	}

	resolver, err := NewResolver(index.Graph)
	if err != nil {
		t.Fatal(err)
	}

	got, err := resolver.ApplyChanges(context.Background(), index.ChangeSetForFiles("src/Button.tsx"))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got.ComponentIDs, []string{"component:src/Button.tsx", "component:src/main.tsx"}) {
		t.Fatalf("components = %#v", got.ComponentIDs)
	}
	if !reflect.DeepEqual(got.RouteIDs, []string{"route:/"}) {
		t.Fatalf("route = %#v, want ['route:/']", got.RouteIDs)
	}
}
