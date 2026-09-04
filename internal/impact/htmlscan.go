package impact

import (
	"path"
	"regexp"
	"sort"
	"strings"
)

var (
	htmlCommentRE  = regexp.MustCompile(`(?s)<!--.*?-->`)
	htmlLinkCSSRE  = regexp.MustCompile(`(?i)<link[^>]+(?:rel=["']?stylesheet["']?[^>]+href=["']([^"']+)["']|href=["']([^"']+)["'][^>]+rel=["']?stylesheet["']?)`)
	htmlScriptRE   = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+)["']`)
	htmlStyleTagRE = regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`)
)

// HTMLFacts captures statically referenced assets and inline styles in an HTML file.
type HTMLFacts struct {
	LinkedStylesheets []string
	ScriptSources     []string
	InlineStyles      [][]byte
}

// ScanHTML performs a bounded lexical scan of an HTML file.
func ScanHTML(content []byte) HTMLFacts {
	clean := htmlCommentRE.ReplaceAll(content, nil)

	var styles []string
	for _, match := range htmlLinkCSSRE.FindAllSubmatch(clean, -1) {
		href := string(match[1])
		if href == "" && len(match) > 2 {
			href = string(match[2])
		}
		href = strings.TrimSpace(href)
		if href != "" && !isExternalURL(href) {
			styles = append(styles, href)
		}
	}

	var scripts []string
	for _, match := range htmlScriptRE.FindAllSubmatch(clean, -1) {
		src := strings.TrimSpace(string(match[1]))
		if src != "" && !isExternalURL(src) {
			scripts = append(scripts, src)
		}
	}

	var inlineStyles [][]byte
	for _, match := range htmlStyleTagRE.FindAllSubmatch(clean, -1) {
		if len(match) > 1 {
			styleText := strings.TrimSpace(string(match[1]))
			if styleText != "" {
				inlineStyles = append(inlineStyles, []byte(styleText))
			}
		}
	}

	sort.Strings(styles)
	sort.Strings(scripts)

	return HTMLFacts{
		LinkedStylesheets: uniqueSortedStrings(styles),
		ScriptSources:     uniqueSortedStrings(scripts),
		InlineStyles:      inlineStyles,
	}
}

// InferRouteFromPath maps conventional web framework file paths to route paths.
// Returns route URL and true if the file represents a route entrypoint.
func InferRouteFromPath(projectPath string) (route string, isRoute bool) {
	clean := canonicalProjectPath(projectPath)
	ext := strings.ToLower(path.Ext(clean))
	baseName := path.Base(clean)
	dir := path.Dir(clean)

	// Strip leading "src/" if present for framework route folders
	dirWithoutSrc := strings.TrimPrefix(dir, "src/")
	dirWithoutSrc = strings.TrimPrefix(dirWithoutSrc, "src")

	// 1. Static HTML files: index.html -> "/", about.html -> "/about"
	if ext == ".html" || ext == ".htm" {
		nameWithoutExt := strings.TrimSuffix(baseName, ext)
		if nameWithoutExt == "index" {
			if dir == "." || dir == "" {
				return "/", true
			}
			return "/" + dir, true
		}
		if dir == "." || dir == "" {
			return "/" + nameWithoutExt, true
		}
		return "/" + path.Join(dir, nameWithoutExt), true
	}

	// 2. Next.js App Router: app/**/page.{tsx,jsx,js,ts}
	if strings.HasPrefix(dirWithoutSrc, "app") || dirWithoutSrc == "app" {
		nameWithoutExt := strings.TrimSuffix(baseName, ext)
		if nameWithoutExt == "page" {
			sub := strings.TrimPrefix(dirWithoutSrc, "app")
			sub = strings.TrimPrefix(sub, "/")
			if sub == "" {
				return "/", true
			}
			return "/" + sub, true
		}
	}

	// 3. Next.js Pages Router: pages/**.{tsx,jsx,js,ts}
	if strings.HasPrefix(dirWithoutSrc, "pages") || dirWithoutSrc == "pages" {
		nameWithoutExt := strings.TrimSuffix(baseName, ext)
		if strings.HasPrefix(nameWithoutExt, "_") || strings.HasPrefix(nameWithoutExt, "api") {
			return "", false // special Next.js file or API route
		}
		sub := strings.TrimPrefix(dirWithoutSrc, "pages")
		sub = strings.TrimPrefix(sub, "/")
		if nameWithoutExt == "index" {
			if sub == "" {
				return "/", true
			}
			return "/" + sub, true
		}
		if sub == "" {
			return "/" + nameWithoutExt, true
		}
		return "/" + path.Join(sub, nameWithoutExt), true
	}

	// 4. Remix / React Router: routes/**.{tsx,jsx,js,ts}
	if strings.HasPrefix(dirWithoutSrc, "routes") || dirWithoutSrc == "routes" {
		nameWithoutExt := strings.TrimSuffix(baseName, ext)
		if nameWithoutExt == "_index" || nameWithoutExt == "index" {
			return "/", true
		}
		routeName := strings.ReplaceAll(nameWithoutExt, ".", "/")
		routeName = strings.TrimPrefix(routeName, "_")
		return "/" + routeName, true
	}

	return "", false
}

// IsGlobalLayout returns true if the file is a framework layout/wrapper that affects all routes.
func IsGlobalLayout(projectPath string) bool {
	clean := canonicalProjectPath(projectPath)
	ext := strings.ToLower(path.Ext(clean))
	baseName := path.Base(clean)
	dir := path.Dir(clean)
	dirWithoutSrc := strings.TrimPrefix(dir, "src/")
	dirWithoutSrc = strings.TrimPrefix(dirWithoutSrc, "src")

	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	if (dirWithoutSrc == "app" || dirWithoutSrc == "") && nameWithoutExt == "layout" {
		return true
	}
	if (dirWithoutSrc == "pages" || dirWithoutSrc == "") && (nameWithoutExt == "_app" || nameWithoutExt == "_document") {
		return true
	}
	return false
}

func isExternalURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "//")
}
