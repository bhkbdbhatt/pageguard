package proxy

import (
	"strings"
	"sync"
	"time"
)

// Rotator selects a proxy for a requested region, rotating across all proxies
// serving that region and quarantining proxies that repeatedly fail. It is safe
// for concurrent use.
type Rotator struct {
	mu        sync.Mutex
	byRegion  map[string][]*poolEntry
	fallback  []*poolEntry // proxies with no region, usable for any request
	rr        map[string]int
	cooldown  time.Duration
	threshold int
	now       func() time.Time
}

type poolEntry struct {
	proxy        Proxy
	failures     int
	cooldownTill time.Time
}

// RotatorOption configures a Rotator.
type RotatorOption func(*Rotator)

// WithCooldown sets how long a proxy is skipped after exceeding the failure
// threshold (default 60s).
func WithCooldown(d time.Duration) RotatorOption {
	return func(r *Rotator) { r.cooldown = d }
}

// WithFailureThreshold sets the consecutive-failure count that triggers a
// cooldown (default 3).
func WithFailureThreshold(n int) RotatorOption {
	return func(r *Rotator) { r.threshold = n }
}

// WithClock overrides the time source (used in tests).
func WithClock(now func() time.Time) RotatorOption {
	return func(r *Rotator) { r.now = now }
}

// New builds a Rotator from the given proxies.
func New(proxies []Proxy, opts ...RotatorOption) *Rotator {
	r := &Rotator{
		byRegion:  map[string][]*poolEntry{},
		rr:        map[string]int{},
		cooldown:  60 * time.Second,
		threshold: 3,
		now:       time.Now,
	}
	for _, o := range opts {
		o(r)
	}
	for _, p := range proxies {
		e := &poolEntry{proxy: p}
		if p.Region == "" {
			r.fallback = append(r.fallback, e)
		} else {
			r.byRegion[p.Region] = append(r.byRegion[p.Region], e)
		}
	}
	return r
}

// Empty reports whether the rotator has no proxies at all.
func (r *Rotator) Empty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byRegion) == 0 && len(r.fallback) == 0
}

// Regions returns the configured regions that have at least one proxy.
func (r *Rotator) Regions() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.byRegion))
	for k := range r.byRegion {
		out = append(out, k)
	}
	return out
}

// Next returns the next healthy proxy for region. It prefers region-specific
// proxies and falls back to region-agnostic ones. The bool is false when no
// proxy is available (callers should then connect directly).
func (r *Rotator) Next(region string) (Proxy, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := region
	avail := r.healthyLocked(r.byRegion[region])
	if len(avail) == 0 {
		key = ""
		avail = r.healthyLocked(r.fallback)
	}
	if len(avail) == 0 {
		return Proxy{}, false
	}
	i := r.rr[key]
	e := avail[i%len(avail)]
	r.rr[key] = i + 1
	return e.proxy, true
}

// Report records the outcome of using a proxy. Failures accrue toward a
// cooldown; a success clears the failure count.
func (r *Rotator) Report(p Proxy, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.find(p)
	if e == nil {
		return
	}
	if err == nil {
		e.failures = 0
		e.cooldownTill = time.Time{}
		return
	}
	e.failures++
	if e.failures >= r.threshold {
		e.cooldownTill = r.now().Add(r.cooldown)
	}
}

func (r *Rotator) find(p Proxy) *poolEntry {
	for _, e := range r.byRegion[p.Region] {
		if e.proxy.URL == p.URL {
			return e
		}
	}
	for _, e := range r.fallback {
		if e.proxy.URL == p.URL {
			return e
		}
	}
	return nil
}

func (r *Rotator) healthyLocked(entries []*poolEntry) []*poolEntry {
	now := r.now()
	out := make([]*poolEntry, 0, len(entries))
	for _, e := range entries {
		if e.cooldownTill.IsZero() || now.After(e.cooldownTill) {
			out = append(out, e)
		}
	}
	return out
}

// ParseProxies parses a list of "region=url" or plain "url" specs into Proxies.
// Plain specs inherit the region of the preceding regioned spec, so
// "us=http://a,http://b,eu=http://c" yields a region "us" pool of two and an
// "eu" pool of one.
func ParseProxies(specs []string) []Proxy {
	out := make([]Proxy, 0, len(specs))
	region := ""
	for _, s := range specs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if i := strings.IndexByte(s, '='); i > 0 && !strings.Contains(s[:i], "://") {
			region = strings.TrimSpace(s[:i])
			s = strings.TrimSpace(s[i+1:])
			if s == "" {
				continue
			}
		}
		out = append(out, Proxy{URL: s, Region: region})
	}
	return out
}
