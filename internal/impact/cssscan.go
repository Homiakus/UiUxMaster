package impact

import (
	"regexp"
	"sort"
	"strings"
)

var (
	cssCommentRE = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssVarDefRE  = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
	cssVarUseRE  = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)`)
)

// CSSCustomProperties describes custom-property definitions and references in
// one stylesheet. Names are returned in canonical CSS form, e.g. --space-4.
type CSSCustomProperties struct {
	Definitions []string
	References  []string
}

// ScanCSSCustomProperties performs a bounded lexical scan suitable for impact
// indexing. It is intentionally not a CSS validator/parser; unsupported or
// generated CSS must be handled conservatively by higher-level adapters.
func ScanCSSCustomProperties(css []byte) CSSCustomProperties {
	clean := cssCommentRE.ReplaceAll(css, nil)
	defs := uniqueCSSNames(cssVarDefRE.FindAllSubmatch(clean, -1))
	refs := uniqueCSSNames(cssVarUseRE.FindAllSubmatch(clean, -1))
	return CSSCustomProperties{Definitions: defs, References: refs}
}

// IndexStyleSheetCSS adds token dependency facts for one stylesheet.
// Definitions propagate stylesheet -> token; references propagate token -> stylesheet.
func (b *Builder) IndexStyleSheetCSS(styleID string, css []byte) (CSSCustomProperties, error) {
	facts := ScanCSSCustomProperties(css)
	if err := b.StyleSheet(styleID); err != nil {
		return CSSCustomProperties{}, err
	}
	for _, name := range facts.Definitions {
		tokenID := CSSCustomPropertyNodeID(name)
		if err := b.DesignToken(tokenID); err != nil {
			return CSSCustomProperties{}, err
		}
		if err := b.graph.AddEdge(Edge{From: styleID, To: tokenID, Kind: EdgeDependsOn}); err != nil {
			return CSSCustomProperties{}, err
		}
	}
	for _, name := range facts.References {
		tokenID := CSSCustomPropertyNodeID(name)
		if err := b.DesignToken(tokenID); err != nil {
			return CSSCustomProperties{}, err
		}
		if err := b.graph.AddEdge(Edge{From: tokenID, To: styleID, Kind: EdgeConsumesToken}); err != nil {
			return CSSCustomProperties{}, err
		}
	}
	return facts, nil
}

func CSSCustomPropertyNodeID(name string) string {
	name = strings.TrimSpace(name)
	return "token:" + name
}

func uniqueCSSNames(matches [][][]byte) []string {
	set := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := string(match[1])
		if name != "" {
			set[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
