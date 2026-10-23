package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/proxy"
)

// PageResult bundles the observation with crawl-relevant metadata extracted
// from the page (links, meta description) plus optional browser evidence
// (HAR archive, client-side data layer).
type PageResult struct {
	Observation     *domain.Observation
	Links           Links
	MetaDescription string
	HAR             json.RawMessage
	DataLayer       []domain.DataLayerEntry
}

// ScanOptions carries per-page scan parameters beyond the scan-level config:
// the crawler depth, the geolocation/proxy to observe from, and which browser
// artifacts to capture.
type ScanOptions struct {
	Depth       int
	Geolocation string
	// ProxyServer is the proxy for the browser render (scheme://host:port,
	// credentials stripped). ProxyUser/ProxyPass are supplied separately.
	ProxyServer string
	ProxyUser   string
	ProxyPass   string
	CaptureHAR  bool
	// CaptureDataLayer records window.dataLayer / digitalData-like payloads.
	CaptureDataLayer bool
}

// Scanner turns a URL into an Observation using a Fetcher and optional Renderer.
type Scanner struct {
	Fetcher  Fetcher
	Renderer Renderer
	Detector *Detector
}

// New constructs a Scanner. renderer may be nil, in which case browser modes
// degrade to HTTP fetching.
func New(fetcher Fetcher, renderer Renderer) *Scanner {
	if fetcher == nil {
		fetcher = NewHTTPFetcher()
	}
	return &Scanner{Fetcher: fetcher, Renderer: renderer, Detector: NewDetector()}
}

// Scan visits rawURL at the given depth. It is a convenience wrapper around
// ScanPage with default options.
func (s *Scanner) Scan(ctx context.Context, rawURL string, cfg domain.ScanConfig, depth int) (*PageResult, error) {
	return s.ScanPage(ctx, rawURL, cfg, ScanOptions{Depth: depth})
}

// ScanPage visits rawURL according to cfg and opts and returns the observation
// plus derived page metadata.
func (s *Scanner) ScanPage(ctx context.Context, rawURL string, cfg domain.ScanConfig, opts ScanOptions) (*PageResult, error) {
	cfg.Normalize()

	// In explicit browser mode, render directly and only fall back to HTTP if
	// the browser path fails.
	if cfg.RenderMode == domain.RenderBrowser && s.Renderer != nil {
		if obs, rerr := s.render(ctx, rawURL, cfg, opts); rerr == nil {
			return s.finalize(obs), nil
		}
	}

	fr, err := s.Fetcher.Fetch(ctx, rawURL, FetchOptions{
		UserAgent:    cfg.UserAgent,
		Timeout:      timeout(cfg),
		MaxBodyBytes: cfg.MaxBodyBytes,
		Geolocation:  opts.Geolocation,
	})
	if err != nil {
		return nil, err
	}

	obs := &domain.Observation{
		URL:           rawURL,
		FinalURL:      fr.URL,
		StatusCode:    fr.Status,
		ContentType:   fr.Headers.Get("Content-Type"),
		Headers:       headerToMap(fr.Headers),
		HTML:          string(fr.Body),
		HTMLTruncated: fr.Truncated,
		Geolocation:   opts.Geolocation,
		DurationMs:    fr.Duration.Milliseconds(),
		Depth:         opts.Depth,
	}
	if isHTML(fr.Headers) {
		obs.Title = ExtractTitle(obs.HTML)
	}
	obs.Cookies = cookiesFromHeaders(fr.Headers, hostOf(fr.URL))
	obs.Requests = []domain.NetworkRequest{{
		Method: "GET", URL: fr.URL, Host: hostOf(fr.URL), Status: fr.Status,
		ResourceType: "document", DurationMs: fr.Duration.Milliseconds(),
		SizeBytes: int64(len(fr.Body)),
	}}

	if cfg.RenderMode == domain.RenderAuto && s.Renderer != nil && LooksLikeSPA(obs.HTML) {
		if rendered, rerr := s.render(ctx, rawURL, cfg, opts); rerr == nil {
			obs = rendered
		}
		// A browser failure leaves the HTTP observation in place.
	}

	return s.finalize(obs), nil
}

// finalize detects tags and extracts crawl-relevant metadata from the
// observation.
func (s *Scanner) finalize(obs *domain.Observation) *PageResult {
	if isHTMLFromObservation(obs) {
		obs.Tags = s.Detector.Detect(obs)
	}
	res := &PageResult{
		Observation:     obs,
		HAR:             obs.HAR,
		DataLayer:       obs.DataLayer,
		MetaDescription: "",
	}
	if isHTMLFromObservation(obs) {
		if base, err := url.Parse(obs.FinalURL); err == nil {
			res.Links = ExtractLinks(base, obs.HTML)
		}
		res.MetaDescription = ExtractMetaDescription(obs.HTML)
	}
	return res
}

func (s *Scanner) render(ctx context.Context, rawURL string, cfg domain.ScanConfig, opts ScanOptions) (*domain.Observation, error) {
	ro := RenderOptions{
		UserAgent:        cfg.UserAgent,
		Timeout:          timeout(cfg),
		WaitMs:           cfg.RenderWaitMs,
		Viewport:         cfg.Viewport,
		Geolocation:      opts.Geolocation,
		ProxyServer:      opts.ProxyServer,
		ProxyUser:        opts.ProxyUser,
		ProxyPass:        opts.ProxyPass,
		CaptureHAR:       opts.CaptureHAR,
		CaptureDataLayer: opts.CaptureDataLayer,
	}
	if opts.Geolocation != "" {
		if r, ok := proxy.LookupRegion(opts.Geolocation); ok {
			ro.GeoLatitude, ro.GeoLongitude, ro.GeoAccuracy = r.Latitude, r.Longitude, r.Accuracy
			ro.Locale, ro.Timezone = r.Locale, r.Timezone
		}
	}
	rr, err := s.Renderer.Render(ctx, rawURL, ro)
	if err != nil {
		return nil, err
	}
	obs := &domain.Observation{
		URL:         rawURL,
		FinalURL:    rr.FinalURL,
		StatusCode:  rr.Status,
		ContentType: rr.ContentType,
		Title:       rr.Title,
		HTML:        rr.HTML,
		Requests:    rr.Requests,
		Cookies:     rr.Cookies,
		Console:     rr.Console,
		Geolocation: opts.Geolocation,
		DataLayer:   rr.DataLayer,
		HAR:         rr.HAR,
		DurationMs:  rr.Duration.Milliseconds(),
		Rendered:    true,
		Depth:       opts.Depth,
	}
	if obs.Title == "" {
		obs.Title = ExtractTitle(rr.HTML)
	}
	if obs.FinalURL == "" {
		obs.FinalURL = rawURL
	}
	return obs, nil
}

// timeout converts the millisecond config value to a duration.
func timeout(cfg domain.ScanConfig) time.Duration {
	return time.Duration(cfg.TimeoutMs) * time.Millisecond
}

func headerToMap(h http.Header) map[string]string {
	if h == nil {
		return nil
	}
	m := make(map[string]string, len(h))
	for k, v := range h {
		m[k] = strings.Join(v, ", ")
	}
	return m
}

func cookiesFromHeaders(h http.Header, pageHost string) []domain.Cookie {
	if h == nil {
		return nil
	}
	resp := http.Response{Header: h}
	parsed := resp.Cookies()
	out := make([]domain.Cookie, 0, len(parsed))
	for _, c := range parsed {
		out = append(out, domain.Cookie{
			Name:        c.Name,
			Domain:      c.Domain,
			Path:        c.Path,
			ValueHash:   hashValue(c.Value),
			SameSite:    sameSiteName(c.SameSite),
			Secure:      c.Secure,
			HTTPOnly:    c.HttpOnly,
			ExpiresUnix: expiryUnix(c),
			Session:     c.MaxAge == 0 && c.Expires.IsZero(),
			FirstParty:  sameParty(c.Domain, pageHost),
		})
	}
	return out
}

// hashValue returns a stable, non-reversible digest of a cookie value so scans
// never persist secrets in the clear.
func hashValue(v string) string {
	if v == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:8])
}

func sameSiteName(m http.SameSite) string {
	switch m {
	case http.SameSiteDefaultMode:
		return "default"
	case http.SameSiteLaxMode:
		return "lax"
	case http.SameSiteStrictMode:
		return "strict"
	case http.SameSiteNoneMode:
		return "none"
	}
	return "unspecified"
}

func expiryUnix(c *http.Cookie) int64 {
	if c.Expires.IsZero() {
		return 0
	}
	return c.Expires.Unix()
}

func sameParty(cookieDomain, pageHost string) bool {
	if cookieDomain == "" {
		return true // host-only cookie
	}
	cd := strings.TrimPrefix(strings.ToLower(cookieDomain), ".")
	ph := strings.ToLower(hostOf("//" + pageHost))
	return ph == cd || strings.HasSuffix(ph, "."+cd)
}

func isHTML(h http.Header) bool {
	return strings.Contains(strings.ToLower(h.Get("Content-Type")), "html")
}

func isHTMLFromObservation(o *domain.Observation) bool {
	if o.ContentType == "" {
		return strings.Contains(strings.ToLower(o.HTML), "<html") || o.Rendered
	}
	return strings.Contains(strings.ToLower(o.ContentType), "html") || o.Rendered
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}
