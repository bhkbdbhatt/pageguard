package scanner

import (
	"net/url"
	"testing"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}

func TestExtractLinksResolvesAndFilters(t *testing.T) {
	doc := `
<html><head>
  <title>  Hello &amp; Welcome </title>
  <meta name="description" content="A test page">
  <link rel="stylesheet" href="/styles/main.css">
  <script src="https://cdn.example.com/app.js"></script>
</head><body>
  <a href="/about">About</a>
  <a href='contact.html'>Contact</a>
  <a href="#section">Anchor</a>
  <a href="javascript:void(0)">JS</a>
  <a href="mailto:hi@example.com">Mail</a>
  <img src="/img/logo.png">
  <iframe src="https://widgets.example.com/frame"></iframe>
</body></html>`

	base := mustURL(t, "https://example.com/blog/post?x=1")
	links := ExtractLinks(base, doc)

	if !contains(links.Documents, "https://example.com/about") {
		t.Errorf("Documents missing /about: %v", links.Documents)
	}
	if !contains(links.Documents, "https://example.com/blog/contact.html") {
		t.Errorf("Documents missing resolved contact: %v", links.Documents)
	}
	if contains(links.Documents, "https://example.com/blog/post?x=1#section") {
		t.Errorf("fragment-only link should be dropped: %v", links.Documents)
	}
	for _, d := range links.Documents {
		if len(d) > 0 && (d == "javascript:void(0)" || d == "mailto:hi@example.com") {
			t.Errorf("non-http link leaked: %s", d)
		}
	}
	if !contains(links.Stylesheets, "https://example.com/styles/main.css") {
		t.Errorf("Stylesheets wrong: %v", links.Stylesheets)
	}
	if !contains(links.Scripts, "https://cdn.example.com/app.js") {
		t.Errorf("Scripts wrong: %v", links.Scripts)
	}
	if !contains(links.Images, "https://example.com/img/logo.png") {
		t.Errorf("Images wrong: %v", links.Images)
	}
	if !contains(links.Iframes, "https://widgets.example.com/frame") {
		t.Errorf("Iframes wrong: %v", links.Iframes)
	}
}

func TestExtractTitleAndMeta(t *testing.T) {
	doc := `<html><head><title> Hello &amp; Welcome </title><meta name="description" content="desc"></head></html>`
	if got := ExtractTitle(doc); got != "Hello & Welcome" {
		t.Errorf("title = %q", got)
	}
	if got := ExtractMetaDescription(doc); got != "desc" {
		t.Errorf("description = %q", got)
	}
}

func TestLooksLikeSPA(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want bool
	}{
		{"react root", `<html><body><div id="root"></div><script src="bundle.js"></script></body></html>`, true},
		{"next", `<html><body><div id="__next"></div></body></html>`, true},
		{"static", `<html><head><title>x</title></head><body><a href="/a">a</a><a href="/b">b</a>${pad}</body></html>`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LooksLikeSPA(tc.doc); got != tc.want {
				t.Errorf("LooksLikeSPA = %v, want %v", got, tc.want)
			}
		})
	}
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
