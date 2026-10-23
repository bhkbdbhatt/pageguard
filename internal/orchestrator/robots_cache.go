package orchestrator

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pageguard/pageguard/internal/scanner"
)

// robotsCache fetches and caches robots.txt per host for the duration of a run.
type robotsCache struct {
	fetcher scanner.Fetcher
	userAgt string

	mu     sync.Mutex
	byHost map[string]*scanner.Robots
}

func newRobotsCache(f scanner.Fetcher, ua string) *robotsCache {
	return &robotsCache{fetcher: f, userAgt: ua, byHost: map[string]*scanner.Robots{}}
}

func (c *robotsCache) get(ctx context.Context, rawURL string) *scanner.Robots {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	key := strings.ToLower(u.Host)
	c.mu.Lock()
	if r, ok := c.byHost[key]; ok {
		c.mu.Unlock()
		return r
	}
	c.mu.Unlock()

	robotsURL := u.Scheme + "://" + u.Host + "/robots.txt"
	var r *scanner.Robots
	res, err := c.fetcher.Fetch(ctx, robotsURL, scanner.FetchOptions{
		UserAgent: c.userAgt, Timeout: 8 * time.Second, MaxBodyBytes: 512 << 10,
	})
	if err == nil && res.Status == 200 {
		r = scanner.ParseRobots(bytes.NewReader(res.Body), c.userAgt)
	}
	if r == nil {
		r = scanner.ParseRobots(strings.NewReader(""), c.userAgt)
	}
	c.mu.Lock()
	c.byHost[key] = r
	c.mu.Unlock()
	return r
}

func (c *robotsCache) allowed(ctx context.Context, rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return c.byHost == nil || c.allowedFor(ctx, u)
}

func (c *robotsCache) allowedFor(ctx context.Context, u *url.URL) bool {
	r := c.get(ctx, u.String())
	if r == nil {
		return true
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	return r.Allowed(path)
}
