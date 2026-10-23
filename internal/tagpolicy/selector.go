package tagpolicy

import (
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
)

// matchTag reports whether a detected tag satisfies the expectation. All
// populated match fields must hold (logical AND).
func (e *Expectation) matchTag(t domain.Tag) bool {
	m := e.Match
	if m.Tag != "" && !strings.EqualFold(t.ID, m.Tag) {
		return false
	}
	if e.reTag != nil && !e.reTag.MatchString(t.ID) && !e.reTag.MatchString(t.Name) {
		return false
	}
	if m.Name != "" && !strings.EqualFold(t.Name, m.Name) {
		return false
	}
	if e.reName != nil && !e.reName.MatchString(t.Name) {
		return false
	}
	if m.Category != "" && !strings.EqualFold(string(t.Category), string(m.Category)) {
		return false
	}
	return true
}

// matchRequest reports whether an observed request satisfies the expectation.
func (e *Expectation) matchRequest(r domain.NetworkRequest) bool {
	m := e.Match
	if e.reURL != nil && !e.reURL.MatchString(r.URL) {
		return false
	}
	if m.Host != "" && !strings.EqualFold(r.Host, m.Host) {
		return false
	}
	return true
}

// matchCookie reports whether an observed cookie satisfies the expectation.
func (e *Expectation) matchCookie(c domain.Cookie) bool {
	m := e.Match
	if m.Name != "" && !strings.EqualFold(c.Name, m.Name) {
		return false
	}
	if e.reName != nil && !e.reName.MatchString(c.Name) {
		return false
	}
	if m.Domain != "" && !domainMatch(c.Domain, m.Domain) {
		return false
	}
	return true
}

// matchEvent reports whether a data-layer event name satisfies the expectation.
func (e *Expectation) matchEvent(ev string) bool {
	m := e.Match
	if m.Event != "" && !strings.EqualFold(ev, m.Event) {
		return false
	}
	if m.Name != "" && !strings.EqualFold(ev, m.Name) {
		return false
	}
	if m.Event == "" && m.Name == "" {
		return false
	}
	return true
}

// domainMatch compares a cookie domain against a policy pattern, tolerating the
// leading dot and subdomain prefixes.
func domainMatch(cookieDomain, pattern string) bool {
	c := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(cookieDomain)), ".")
	p := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(pattern)), ".")
	if p == "" {
		return true
	}
	return c == p || strings.HasSuffix(c, "."+p)
}
