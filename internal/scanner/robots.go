package scanner

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"
)

// Robots is a parsed robots.txt relevant to a single user agent.
type Robots struct {
	rules      []robotsRule
	crawlDelay time.Duration
	sitemaps   []string
	userAgent  string
}

type robotsRule struct {
	allow   bool
	pattern string
	wildEnd bool
}

// Sitemaps returns sitemap URLs declared in robots.txt.
func (r *Robots) Sitemaps() []string { return r.sitemaps }

// CrawlDelay returns the declared crawl delay, or 0.
func (r *Robots) CrawlDelay() time.Duration { return r.crawlDelay }

// ParseRobots reads robots.txt and extracts the rule group best matching
// userAgent. Matching is simple substring/`*` matching which covers the vast
// majority of real-world files.
func ParseRobots(rd io.Reader, userAgent string) *Robots {
	ua := strings.ToLower(userAgent)
	r := &Robots{userAgent: ua}

	// First pass: find the most specific matching group.
	type group struct {
		agents []string
		rules  []robotsRule
		delay  time.Duration
	}
	var groups []group
	var cur *group
	flush := func() {
		if cur != nil && len(cur.agents) > 0 {
			groups = append(groups, *cur)
		}
		cur = nil
	}

	sc := bufio.NewScanner(rd)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)

		switch key {
		case "user-agent":
			// A new user-agent line after rules starts a new group.
			if cur != nil && len(cur.rules) > 0 {
				flush()
			}
			if cur == nil {
				cur = &group{}
			}
			cur.agents = append(cur.agents, strings.ToLower(val))
		case "disallow", "allow":
			if cur == nil {
				cur = &group{agents: []string{"*"}}
			}
			rule := robotsRule{allow: key == "allow", pattern: val}
			if strings.HasSuffix(val, "$") {
				rule.wildEnd = true
				rule.pattern = strings.TrimSuffix(val, "$")
			}
			cur.rules = append(cur.rules, rule)
		case "crawl-delay":
			if cur == nil {
				cur = &group{agents: []string{"*"}}
			}
			if secs, err := strconv.ParseFloat(val, 64); err == nil {
				cur.delay = time.Duration(secs * float64(time.Second))
			}
		case "sitemap":
			r.sitemaps = append(r.sitemaps, val)
		}
	}
	flush()

	// Pick the group with the longest matching agent token; "*" is a fallback.
	best := -1
	bestLen := -1
	for i, g := range groups {
		for _, a := range g.agents {
			match := a == "*"
			score := 0
			if match {
				score = 0
			} else if strings.Contains(ua, a) {
				match = true
				score = len(a)
			}
			if match && score > bestLen {
				bestLen = score
				best = i
			}
		}
	}
	if best >= 0 {
		r.rules = groups[best].rules
		r.crawlDelay = groups[best].delay
	}
	return r
}

// Allowed reports whether path may be crawled. Longest-match wins, with Allow
// taking precedence over Disallow of equal length.
func (r *Robots) Allowed(path string) bool {
	if path == "" {
		path = "/"
	}
	bestLen := -1
	allowed := true
	for _, rule := range r.rules {
		if rule.pattern == "" {
			// "Disallow:" with empty value allows everything.
			continue
		}
		if matchRobotsPattern(path, rule.pattern, rule.wildEnd) {
			if len(rule.pattern) > bestLen || (len(rule.pattern) == bestLen && rule.allow) {
				bestLen = len(rule.pattern)
				allowed = rule.allow
			}
		}
	}
	return allowed
}

func matchRobotsPattern(path, pattern string, anchoredEnd bool) bool {
	if !strings.Contains(pattern, "*") {
		if anchoredEnd {
			return path == pattern
		}
		return strings.HasPrefix(path, pattern)
	}
	// Support '*' wildcards by splitting on '*'.
	parts := strings.Split(pattern, "*")
	idx := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		j := strings.Index(path[idx:], part)
		if j < 0 {
			return false
		}
		if i == 0 && j != 0 {
			return false
		}
		idx += j + len(part)
	}
	if anchoredEnd && len(parts) > 0 && parts[len(parts)-1] != "" {
		return strings.HasSuffix(path, parts[len(parts)-1])
	}
	return true
}
