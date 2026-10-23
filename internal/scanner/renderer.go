package scanner

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// RenderOptions tune a headless-browser render.
type RenderOptions struct {
	UserAgent string
	Timeout   time.Duration
	WaitMs    int
	Viewport  domain.Viewport
	// Geolocation is the logical region the browser context should imitate.
	Geolocation string
	// Geo* carry the concrete coordinates/locale/timezone for the region.
	GeoLatitude  float64
	GeoLongitude float64
	GeoAccuracy  float64
	Locale       string
	Timezone     string
	// ProxyServer is the browser-side proxy (scheme://host:port); credentials
	// are passed separately so they never appear in logs.
	ProxyServer string
	ProxyUser   string
	ProxyPass   string
	// CaptureHAR requests a full network archive.
	CaptureHAR bool
	// CaptureDataLayer requests client-side data layer payloads.
	CaptureDataLayer bool
}

// RenderResult is the outcome of rendering a page in a browser context.
type RenderResult struct {
	FinalURL    string
	Status      int
	ContentType string
	HTML        string
	Title       string
	Requests    []domain.NetworkRequest
	Cookies     []domain.Cookie
	Console     []domain.ConsoleMessage
	DataLayer   []domain.DataLayerEntry
	HAR         json.RawMessage
	Duration    time.Duration
}

// Renderer abstracts a headless browser so tests can substitute a fake and so
// Playwright/Puppeteer adapters remain pluggable.
type Renderer interface {
	Render(ctx context.Context, rawURL string, opts RenderOptions) (*RenderResult, error)
	Close() error
}

// FakeRenderer is a scriptable Renderer for tests.
type FakeRenderer struct {
	// Responses maps a URL to the result to return. A missing key yields a
	// default 200 result containing the URL in a minimal HTML shell.
	Responses map[string]*RenderResult
	// Err, when set, is returned for every call.
	Err error

	mu    sync.Mutex
	calls []string
}

// Render records the call and returns the configured result.
func (f *FakeRenderer) Render(_ context.Context, rawURL string, _ RenderOptions) (*RenderResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, rawURL)
	f.mu.Unlock()

	if f.Err != nil {
		return nil, f.Err
	}
	if r, ok := f.Responses[rawURL]; ok {
		cp := *r
		return &cp, nil
	}
	return &RenderResult{
		FinalURL: rawURL,
		Status:   200,
		HTML:     "<html><head><title>fake</title></head><body id=\"root\"></body></html>",
		Title:    "fake",
	}, nil
}

// Calls returns the URLs passed to Render, in order.
func (f *FakeRenderer) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// Close is a no-op.
func (f *FakeRenderer) Close() error { return nil }
