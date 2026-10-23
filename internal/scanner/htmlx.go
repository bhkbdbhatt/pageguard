package scanner

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

// Links holds the outbound references found in a document, grouped by kind.
type Links struct {
	Documents   []string `json:"documents"` // <a href>
	Stylesheets []string `json:"stylesheets"`
	Scripts     []string `json:"scripts"`
	Images      []string `json:"images"`
	Iframes     []string `json:"iframes"`
}

// All returns every discovered URL, de-duplicated, preserving first-seen order.
func (l Links) All() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(l.Documents)+len(l.Stylesheets)+len(l.Scripts)+len(l.Images)+len(l.Iframes))
	for _, group := range [][]string{l.Documents, l.Stylesheets, l.Scripts, l.Images, l.Iframes} {
		for _, u := range group {
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			out = append(out, u)
		}
	}
	return out
}

// attrRegexp builds a tolerant matcher for `attr` on `tag`, supporting
// double-quoted, single-quoted and bare attribute values.
func attrRegexp(tag, attr string) *regexp.Regexp {
	// (?is) = case-insensitive, dot matches newline.
	pat := `(?is)<` + tag + `\b[^>]*?\b` + attr + `\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`
	return regexp.MustCompile(pat)
}

var (
	reLinkHref   = attrRegexp("link", "href")
	reScriptSrc  = attrRegexp("script", "src")
	reImgSrc     = attrRegexp("img", "src")
	reIframeSrc  = attrRegexp("iframe", "src")
	reAnchorHref = attrRegexp("a", "href")

	reTitle    = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reMetaName = func(name string) *regexp.Regexp {
		return regexp.MustCompile(`(?is)<meta\b[^>]*\bname\s*=\s*["']` + name + `["'][^>]*\bcontent\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	}
	reMetaDescription = reMetaName("description")
)

func matchValues(re *regexp.Regexp, doc string) []string {
	ms := re.FindAllStringSubmatch(doc, -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		for i := 1; i < len(m); i++ {
			if m[i] != "" {
				out = append(out, html.UnescapeString(strings.TrimSpace(m[i])))
				break
			}
		}
	}
	return out
}

// ExtractLinks parses link-like references from HTML and resolves them against
// base. This is a deliberately small, dependency-free parser: it is good
// enough for crawling and is covered by tests; a full DOM parser can be
// swapped in behind this function without touching callers.
func ExtractLinks(base *url.URL, doc string) Links {
	resolve := func(refs []string) []string {
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			if u, ok := resolveRef(base, ref); ok {
				out = append(out, u)
			}
		}
		return out
	}
	return Links{
		Documents:   resolve(matchValues(reAnchorHref, doc)),
		Stylesheets: resolve(matchValues(reLinkHref, doc)),
		Scripts:     resolve(matchValues(reScriptSrc, doc)),
		Images:      resolve(matchValues(reImgSrc, doc)),
		Iframes:     resolve(matchValues(reIframeSrc, doc)),
	}
}

// resolveRef resolves a possibly-relative reference and filters out
// non-HTTP(S) and fragment-only targets.
func resolveRef(base *url.URL, ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "#") {
		return "", false
	}
	low := strings.ToLower(ref)
	for _, scheme := range []string{"javascript:", "mailto:", "tel:", "data:", "blob:", "about:"} {
		if strings.HasPrefix(low, scheme) {
			return "", false
		}
	}
	var abs *url.URL
	if base != nil {
		abs = base.ResolveReference(&url.URL{Path: ref})
		if u, err := url.Parse(ref); err == nil {
			abs = base.ResolveReference(u)
		}
	} else {
		u, err := url.Parse(ref)
		if err != nil {
			return "", false
		}
		abs = u
	}
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return "", false
	}
	abs.Fragment = ""
	return abs.String(), true
}

// ExtractTitle returns the trimmed <title> text, or "".
func ExtractTitle(doc string) string {
	m := reTitle.FindStringSubmatch(doc)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(m[1]))
}

// ExtractMetaDescription returns the meta description content, or "".
func ExtractMetaDescription(doc string) string {
	vals := matchValues(reMetaDescription, doc)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

// LooksLikeSPA heuristically reports whether a document is a client-rendered
// shell that should be re-scanned with a browser (used by RenderAuto).
func LooksLikeSPA(doc string) bool {
	if len(doc) == 0 {
		return false
	}
	lower := strings.ToLower(doc)
	mountPoints := []string{`id="root"`, `id="app"`, `id="__next"`, `id="__nuxt"`, `data-reactroot`, `ng-version`}
	for _, m := range mountPoints {
		if strings.Contains(lower, m) {
			return true
		}
	}
	// Very small bodies with a script bundle are typically SPA shells.
	links := ExtractLinks(nil, doc)
	return len(doc) < 4096 && len(links.Documents) == 0 && strings.Contains(lower, "<script")
}
