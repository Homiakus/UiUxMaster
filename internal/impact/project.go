package impact

import (
	"fmt"
	"path"
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

var moduleExtensions = []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".cts", ".mjs", ".cjs"}

// IndexProject builds dependency/token/component relationships from a bounded
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

	// Pass 1 creates stable file/module/style/component/token entities.
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
		case ext == ".css":
			if err := b.SourceBacksStyle(fileNodeID(p), styleNodeID(p)); err != nil {
				return nil, err
			}
			if _, err := b.IndexStyleSheetCSS(styleNodeID(p), contents[p]); err != nil {
				return nil, err
			}
		}
	}

	// Pass 2 resolves local source dependencies against the known file set.
	for _, p := range paths {
		ext := strings.ToLower(path.Ext(p))
		if !isModuleExtension(ext) {
			continue
		}
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
		candidates = append(candidates, base+".css")
		for _, ext := range moduleExtensions {
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

func fileNodeID(p string) string      { return "file:" + p }
func moduleNodeID(p string) string    { return "module:" + p }
func styleNodeID(p string) string     { return "style:" + p }
func componentNodeID(p string) string { return "component:" + p }
