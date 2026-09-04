package impact

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ProjectFile is an already-available source file. Filesystem watching/git
// integration lives outside the graph kernel.
type ProjectFile struct {
	Path    string
	Content []byte
}

// ProjectIndex is the first native frontend source index. Uncertain is explicit
// so callers can force conservative validation when lexical analysis cannot
// resolve a dependency.
type ProjectIndex struct {
	Graph     *Graph
	Uncertain bool
	Reasons   []string
}

var (
	moduleExtensions = []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".cts", ".mjs", ".cjs"}
	htmlExtensions   = []string{".html", ".htm"}
	styleExtensions  = []string{".css"}
)

// IndexProject builds dependency/token/component/route relationships from a bounded
// set of project files. It intentionally starts with deterministic relative
// imports instead of pretending to be a full framework compiler.
func IndexProject(files []ProjectFile) (*ProjectIndex, error) {
	contents := make(map[string][]byte, len(files))
	paths := make([]string, 0, len(files))
	for _, file := range files {
		p := canonicalProjectPath(file.Path)
		if p == "" || p == "." {
			return nil, fmt.Errorf("impact: invalid project file path %q", file.Path)
		}
		if _, exists := contents[p]; exists {
			return nil, fmt.Errorf("impact: duplicate project file %q", p)
		}
		contents[p] = file.Content
		paths = append(paths, p)
	}
	sort.Strings(paths)

	b := NewBuilder()
	index := &ProjectIndex{Graph: b.Graph()}

	// Track discovered routes and layout files
	var discoveredRoutes []string
	var layoutFiles []string

	// Pass 1 creates stable file/module/style/component/token/route entities.
	for _, p := range paths {
		ext := strings.ToLower(path.Ext(p))
		switch {
		case isModuleExtension(ext):
			if err := b.SourceBacksModule(fileNodeID(p), moduleNodeID(p)); err != nil {
				return nil, err
			}
			if ext == ".tsx" || ext == ".jsx" {
				if err := b.ModuleRendersComponent(moduleNodeID(p), componentNodeID(p)); err != nil {
					return nil, err
				}
			}
			if route, isRoute := InferRouteFromPath(p); isRoute {
				rID := routeNodeID(route)
				if err := b.Route(rID); err != nil {
					return nil, err
				}
				discoveredRoutes = append(discoveredRoutes, rID)
				// Page module / component appears on route
				if ext == ".tsx" || ext == ".jsx" {
					if err := b.graph.AddEdge(Edge{From: componentNodeID(p), To: rID, Kind: EdgeAppearsOn}); err != nil {
						return nil, err
					}
				} else {
					if err := b.graph.AddEdge(Edge{From: moduleNodeID(p), To: rID, Kind: EdgeAppearsOn}); err != nil {
						return nil, err
					}
				}
			}
			if IsGlobalLayout(p) {
				layoutFiles = append(layoutFiles, p)
			}

		case ext == ".css":
			if err := b.SourceBacksStyle(fileNodeID(p), styleNodeID(p)); err != nil {
				return nil, err
			}
			if _, err := b.IndexStyleSheetCSS(styleNodeID(p), contents[p]); err != nil {
				return nil, err
			}

		case isHTMLExtension(ext):
			route, isRoute := InferRouteFromPath(p)
			var rID string
			if isRoute {
				rID = routeNodeID(route)
				if err := b.Route(rID); err != nil {
					return nil, err
				}
				discoveredRoutes = append(discoveredRoutes, rID)
				if err := b.SourceFile(fileNodeID(p)); err != nil {
					return nil, err
				}
				if err := b.graph.AddEdge(Edge{From: fileNodeID(p), To: rID, Kind: EdgeAppearsOn}); err != nil {
					return nil, err
				}
			} else {
				if err := b.SourceFile(fileNodeID(p)); err != nil {
					return nil, err
				}
			}

			// Scan HTML for stylesheet links, scripts, and inline styles
			facts := ScanHTML(contents[p])
			for i, inlineCSS := range facts.InlineStyles {
				inlineStyleID := fmt.Sprintf("style:%s#inline%d", p, i+1)
				if _, err := b.IndexStyleSheetCSS(inlineStyleID, inlineCSS); err != nil {
					return nil, err
				}
				if rID != "" {
					if err := b.graph.AddEdge(Edge{From: inlineStyleID, To: rID, Kind: EdgeStyles}); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	// Pass 2 resolves local source dependencies against the known file set.
	for _, p := range paths {
		ext := strings.ToLower(path.Ext(p))
		switch {
		case isModuleExtension(ext):
			facts := ScanESModuleDependencies(contents[p])
			if facts.DynamicUnresolved {
				index.Uncertain = true
				index.Reasons = append(index.Reasons, "dynamic_import_unresolved:"+p)
			}
			specifiers := append(append([]string(nil), facts.StaticSpecifiers...), facts.DynamicSpecifiers...)
			sort.Strings(specifiers)
			for _, specifier := range specifiers {
				resolved, internal, ok := resolveProjectSpecifier(p, specifier, contents)
				if !internal {
					continue // package/external dependency; not part of local validation graph yet.
				}
				if !ok {
					index.Uncertain = true
					index.Reasons = append(index.Reasons, "unresolved_import:"+p+":"+specifier)
					continue
				}
				if strings.EqualFold(path.Ext(resolved), ".css") {
					if err := b.StyleImportedByModule(styleNodeID(resolved), moduleNodeID(p)); err != nil {
						return nil, err
					}
					continue
				}
				if err := b.Import(moduleNodeID(p), moduleNodeID(resolved)); err != nil {
					return nil, err
				}
			}

		case isHTMLExtension(ext):
			route, isRoute := InferRouteFromPath(p)
			var rID string
			if isRoute {
				rID = routeNodeID(route)
			}
			facts := ScanHTML(contents[p])
			for _, cssHref := range facts.LinkedStylesheets {
				resolved, internal, ok := resolveProjectSpecifier(p, cssHref, contents)
				if internal && ok {
					if rID != "" {
						if err := b.graph.AddEdge(Edge{From: styleNodeID(resolved), To: rID, Kind: EdgeStyles}); err != nil {
							return nil, err
						}
					}
				}
			}
			for _, scriptSrc := range facts.ScriptSources {
				resolved, internal, ok := resolveProjectSpecifier(p, scriptSrc, contents)
				if internal && ok {
					if rID != "" {
						modID := moduleNodeID(resolved)
						if err := b.Module(modID); err == nil {
							_ = b.graph.AddEdge(Edge{From: modID, To: rID, Kind: EdgeAppearsOn})
						}
					}
				}
			}
		}
	}

	// Connect global layouts to all discovered routes
	for _, layout := range layoutFiles {
		layoutModID := moduleNodeID(layout)
		for _, rID := range discoveredRoutes {
			if err := b.graph.AddEdge(Edge{From: layoutModID, To: rID, Kind: EdgeAppearsOn}); err != nil {
				return nil, err
			}
		}
	}

	index.Reasons = uniqueSortedStrings(index.Reasons)
	return index, nil
}

// ChangeSetForFiles translates changed repository paths into graph node IDs and
// carries project-index uncertainty into the resolver contract.
func (p *ProjectIndex) ChangeSetForFiles(changedPaths ...string) ChangeSet {
	ids := make([]string, 0, len(changedPaths))
	for _, changed := range changedPaths {
		cp := canonicalProjectPath(changed)
		if cp == "" || cp == "." {
			continue
		}
		ids = append(ids, fileNodeID(cp))
	}
	sort.Strings(ids)
	return ChangeSet{NodeIDs: ids, Uncertain: p.Uncertain, Reasons: append([]string(nil), p.Reasons...)}
}

// LoadProjectDirectory walks rootDir and loads all frontend/source files into ProjectFile slices.
func LoadProjectDirectory(rootDir string, customIgnores ...string) ([]ProjectFile, error) {
	ignoreMap := map[string]bool{
		".git":         true,
		"node_modules": true,
		"dist":         true,
		"build":        true,
		".next":        true,
		".turbo":       true,
		".cache":       true,
		".output":      true,
		"vendor":       true,
		".gemini":      true,
		"coverage":     true,
		".nyc_output":  true,
		"bin":          true,
	}
	for _, ign := range customIgnores {
		ignoreMap[strings.ToLower(ign)] = true
	}

	var files []ProjectFile
	err := filepath.WalkDir(rootDir, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if ignoreMap[strings.ToLower(name)] || strings.HasPrefix(name, ".") && name != "." {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		if !isIndexableExtension(ext) {
			return nil
		}

		rel, err := filepath.Rel(rootDir, filePath)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		files = append(files, ProjectFile{
			Path:    filepath.ToSlash(rel),
			Content: content,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("impact: load project directory %q: %w", rootDir, err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

// IndexDirectory reads files from a filesystem directory and indexes the project.
func IndexDirectory(rootDir string, customIgnores ...string) (*ProjectIndex, error) {
	files, err := LoadProjectDirectory(rootDir, customIgnores...)
	if err != nil {
		return nil, err
	}
	return IndexProject(files)
}

func resolveProjectSpecifier(importer, specifier string, files map[string][]byte) (resolved string, internal, ok bool) {
	specifier = stripSpecifierSuffix(strings.TrimSpace(specifier))
	if !strings.HasPrefix(specifier, ".") {
		return "", false, false
	}
	base := canonicalProjectPath(path.Join(path.Dir(importer), specifier))
	candidates := []string{base}
	if path.Ext(base) == "" {
		for _, ext := range moduleExtensions {
			candidates = append(candidates, base+ext)
		}
		for _, ext := range htmlExtensions {
			candidates = append(candidates, base+ext)
		}
		candidates = append(candidates, base+".css")
		for _, ext := range moduleExtensions {
			candidates = append(candidates, path.Join(base, "index"+ext))
		}
		for _, ext := range htmlExtensions {
			candidates = append(candidates, path.Join(base, "index"+ext))
		}
		candidates = append(candidates, path.Join(base, "index.css"))
	}
	for _, candidate := range candidates {
		if _, exists := files[candidate]; exists {
			return candidate, true, true
		}
	}
	return base, true, false
}

func stripSpecifierSuffix(specifier string) string {
	if i := strings.IndexAny(specifier, "?#"); i >= 0 {
		return specifier[:i]
	}
	return specifier
}

func canonicalProjectPath(p string) string {
	p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	p = strings.TrimPrefix(p, "./")
	return path.Clean(p)
}

func isModuleExtension(ext string) bool {
	for _, candidate := range moduleExtensions {
		if ext == candidate {
			return true
		}
	}
	return false
}

func isHTMLExtension(ext string) bool {
	for _, candidate := range htmlExtensions {
		if ext == candidate {
			return true
		}
	}
	return false
}

func isIndexableExtension(ext string) bool {
	return isModuleExtension(ext) || isHTMLExtension(ext) || ext == ".css"
}

func fileNodeID(p string) string      { return "file:" + p }
func moduleNodeID(p string) string    { return "module:" + p }
func styleNodeID(p string) string     { return "style:" + p }
func componentNodeID(p string) string { return "component:" + p }
func routeNodeID(r string) string     { return "route:" + r }
