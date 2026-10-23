package scanner

import (
	"strings"
	"testing"
	"time"
)

const sampleRobots = `
# example robots
User-agent: *
Disallow: /private
Disallow: /tmp
Allow: /private/public
Crawl-delay: 2
Sitemap: https://example.com/sitemap.xml

User-agent: PageguardBot
Disallow:
Allow: /
`

func TestParseRobots(t *testing.T) {
	r := ParseRobots(strings.NewReader(sampleRobots), "PageguardBot/0.1")

	// PageguardBot has its own group (Disallow: empty => allow all).
	if !r.Allowed("/private") {
		t.Errorf("PageguardBot-specific group should allow /private")
	}
	if len(r.Sitemaps()) != 1 || r.Sitemaps()[0] != "https://example.com/sitemap.xml" {
		t.Errorf("sitemaps = %v", r.Sitemaps())
	}
}

func TestRobotsWildcardGroup(t *testing.T) {
	r := ParseRobots(strings.NewReader(sampleRobots), "SomeOtherBot/1.0")
	if r.Allowed("/private") {
		t.Errorf("/private should be disallowed for wildcard group")
	}
	if !r.Allowed("/private/public") {
		t.Errorf("Allow should override Disallow with longer match")
	}
	if !r.Allowed("/blog") {
		t.Errorf("/blog should be allowed")
	}
	if r.CrawlDelay() != 2*time.Second {
		t.Errorf("crawl delay = %v", r.CrawlDelay())
	}
}

func TestRobotsWildcardsAndAnchors(t *testing.T) {
	robots := `
User-agent: *
Disallow: /*.pdf$
Disallow: /search*
Disallow: /admin/
`
	r := ParseRobots(strings.NewReader(robots), "bot")
	if r.Allowed("/docs/report.pdf") {
		t.Errorf("*.pdf$ should block report.pdf")
	}
	if !r.Allowed("/docs/report.pdf.html") {
		t.Errorf("$.pdf anchor should allow report.pdf.html")
	}
	if r.Allowed("/search?q=1") {
		t.Errorf("/search* should block /search?q=1")
	}
	if !r.Allowed("/admin") {
		t.Errorf("/admin/ should not block /admin")
	}
	if r.Allowed("/admin/users") {
		t.Errorf("/admin/ should block /admin/users")
	}
}
