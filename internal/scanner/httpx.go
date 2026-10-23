// Package scanner contains the page-scanning engine: an HTTP fetcher, an
// optional headless-browser renderer, HTML/link analysis, robots.txt handling,
// and a lightweight tag detector. It converts a URL into a domain.Observation
// that validators consume.
package scanner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pageguard/pageguard/internal/proxy"
)

// FetchOptions tune a single HTTP fetch.
type FetchOptions struct {
	UserAgent    string
	Timeout      time.Duration
	MaxBodyBytes int64
	Headers      map[string]string
	// Geolocation, when set, routes the request through a region-specific proxy
	// (when the fetcher is configured with a proxy RoundTripper).
	Geolocation string
}

// FetchResult is the outcome of an HTTP fetch.
type FetchResult struct {
	URL       string      // final URL after redirects
	Status    int         // final HTTP status
	Headers   http.Header // final response headers
	Body      []byte      // response body, possibly truncated
	Truncated bool
	Redirects []string // ordered redirect chain (excludes the initial URL)
	Duration  time.Duration
}

// Fetcher abstracts the HTTP transport so tests can substitute a fake.
type Fetcher interface {
	Fetch(ctx context.Context, rawURL string, opts FetchOptions) (*FetchResult, error)
}

// HTTPFetcher is the production Fetcher built on net/http.
type HTTPFetcher struct {
	Transport http.RoundTripper
	// MaxRedirects caps the redirect chain; defaults to 10.
	MaxRedirects int
}

// NewHTTPFetcher returns a fetcher using the default transport.
func NewHTTPFetcher() *HTTPFetcher { return &HTTPFetcher{} }

// Fetch performs a GET, records the redirect chain, and reads at most
// MaxBodyBytes of the body.
func (f *HTTPFetcher) Fetch(ctx context.Context, rawURL string, opts FetchOptions) (*FetchResult, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 20 * time.Second
	}
	if opts.MaxBodyBytes <= 0 {
		opts.MaxBodyBytes = 4 << 20
	}
	maxRedirects := f.MaxRedirects
	if maxRedirects <= 0 {
		maxRedirects = 10
	}

	var redirects []string
	client := &http.Client{
		Transport: f.Transport,
		Timeout:   opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirects = append(redirects, req.URL.String())
			if len(via) > maxRedirects {
				return fmt.Errorf("scanner: stopped after %d redirects", maxRedirects)
			}
			return nil
		},
	}

	if opts.Geolocation != "" {
		ctx = proxy.WithRegion(ctx, opts.Geolocation)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, opts.MaxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	truncated := int64(len(body)) > opts.MaxBodyBytes
	if truncated {
		body = body[:opts.MaxBodyBytes]
	}

	return &FetchResult{
		URL:       resp.Request.URL.String(),
		Status:    resp.StatusCode,
		Headers:   resp.Header,
		Body:      body,
		Truncated: truncated,
		Redirects: redirects,
		Duration:  time.Since(start),
	}, nil
}
