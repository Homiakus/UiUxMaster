package impact

import (
	"regexp"
	"sort"
)

var (
	jsBlockCommentRE = regexp.MustCompile(`(?s)/\*.*?\*/`)
	jsLineCommentRE  = regexp.MustCompile(`(?m)//[^\n\r]*`)
	jsStaticImportRE = regexp.MustCompile(`(?m)\b(?:import|export)\s+(?:[^'"\n;]*?\s+from\s+)?['"]([^'"]+)['"]`)
	jsDynamicLiteralRE = regexp.MustCompile(`\bimport\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	jsDynamicAnyRE = regexp.MustCompile(`\bimport\s*\(`)
)

// ESModuleDependencies is a lexical dependency scan result. StaticSpecifiers
// can be resolved into canonical project modules by a framework/project adapter.
// DynamicUnresolved forces conservative scope expansion until runtime/source
// analysis can identify the target.
type ESModuleDependencies struct {
	StaticSpecifiers  []string
	DynamicSpecifiers []string
	DynamicUnresolved bool
}

// ScanESModuleDependencies recognizes common import/export-from forms and
// literal dynamic imports. It intentionally does not pretend to be a full
// JavaScript parser; unresolved dynamic import expressions are surfaced.
func ScanESModuleDependencies(src []byte) ESModuleDependencies {
	clean := jsBlockCommentRE.ReplaceAll(src, nil)
	clean = jsLineCommentRE.ReplaceAll(clean, nil)

	staticSpecs := uniqueJSImportSpecifiers(jsStaticImportRE.FindAllSubmatch(clean, -1))
	dynamicSpecs := uniqueJSImportSpecifiers(jsDynamicLiteralRE.FindAllSubmatch(clean, -1))
	dynamicCount := len(jsDynamicAnyRE.FindAll(clean, -1))
	literalCount := len(jsDynamicLiteralRE.FindAll(clean, -1))

	return ESModuleDependencies{
		StaticSpecifiers:  staticSpecs,
		DynamicSpecifiers: dynamicSpecs,
		DynamicUnresolved: dynamicCount > literalCount,
	}
}

// IndexResolvedImports records already-resolved project module dependencies.
// Resolution of './Button' to a canonical module ID belongs to a project adapter.
func (b *Builder) IndexResolvedImports(importer string, dependencies []string) error {
	for _, dependency := range dependencies {
		if err := b.Import(importer, dependency); err != nil {
			return err
		}
	}
	return nil
}

func uniqueJSImportSpecifiers(matches [][][]byte) []string {
	set := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 || len(match[1]) == 0 {
			continue
		}
		set[string(match[1])] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
