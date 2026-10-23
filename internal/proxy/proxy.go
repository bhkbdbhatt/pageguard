// Package proxy provides proxy rotation and geolocation context for scans.
//
// Cross-border data collection detection requires observing a site from more
// than one network vantage point. A Rotator selects, per request/page, a proxy
// registered for a target region; a RoundTripper wires the same rotation into
// net/http (used by the HTTP fetcher), while the browser renderer receives the
// proxy explicitly per render.
package proxy

import (
	"context"
	"net/url"
	"strings"
)

// Proxy is a single egress endpoint. URL is a standard proxy URL, optionally
// including credentials (scheme://user:pass@host:port). Region is a logical
// geolocation label (e.g. "us", "eu", "de").
type Proxy struct {
	URL    string `json:"url"`
	Region string `json:"region"`
	Label  string `json:"label,omitempty"`
}

// Host returns the host:port of the proxy, without credentials.
func (p Proxy) Host() string {
	u, err := url.Parse(p.URL)
	if err != nil {
		return p.URL
	}
	return u.Host
}

// Server returns the proxy URL as Playwright expects it (scheme://host:port),
// with any userinfo stripped (credentials are passed separately).
func (p Proxy) Server() string {
	u, err := url.Parse(p.URL)
	if err != nil {
		return p.URL
	}
	c := *u
	c.User = nil
	return c.String()
}

// Credentials returns the proxy username/password, if present.
func (p Proxy) Credentials() (user, pass string, ok bool) {
	u, err := url.Parse(p.URL)
	if err != nil || u.User == nil {
		return "", "", false
	}
	pass, _ = u.User.Password()
	return u.User.Username(), pass, true
}

// Region describes a geolocation profile used for cross-border data collection
// detection. It is applied to the browser context (geolocation, locale,
// timezone) and recorded on every ScanResult observed from that region.
type Region struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy,omitempty"`
	Locale    string  `json:"locale,omitempty"`
	Timezone  string  `json:"timezone,omitempty"`
}

// builtinRegions is a small catalogue of common geolocations. Operators can use
// any string as a region name; unknown names yield a zero-value Region (no
// browser geolocation spoofing, but the label is still recorded).
var builtinRegions = map[string]Region{
	"us": {Name: "us", Latitude: 40.7128, Longitude: -74.0060, Accuracy: 50, Locale: "en-US", Timezone: "America/New_York"},
	"ca": {Name: "ca", Latitude: 43.6532, Longitude: -79.3832, Accuracy: 50, Locale: "en-CA", Timezone: "America/Toronto"},
	"uk": {Name: "uk", Latitude: 51.5074, Longitude: -0.1278, Accuracy: 50, Locale: "en-GB", Timezone: "Europe/London"},
	"ie": {Name: "ie", Latitude: 53.3498, Longitude: -6.2603, Accuracy: 50, Locale: "en-IE", Timezone: "Europe/Dublin"},
	"de": {Name: "de", Latitude: 52.5200, Longitude: 13.4050, Accuracy: 50, Locale: "de-DE", Timezone: "Europe/Berlin"},
	"fr": {Name: "fr", Latitude: 48.8566, Longitude: 2.3522, Accuracy: 50, Locale: "fr-FR", Timezone: "Europe/Paris"},
	"nl": {Name: "nl", Latitude: 52.3676, Longitude: 4.9041, Accuracy: 50, Locale: "nl-NL", Timezone: "Europe/Amsterdam"},
	"au": {Name: "au", Latitude: -33.8688, Longitude: 151.2093, Accuracy: 50, Locale: "en-AU", Timezone: "Australia/Sydney"},
	"jp": {Name: "jp", Latitude: 35.6762, Longitude: 139.6503, Accuracy: 50, Locale: "ja-JP", Timezone: "Asia/Tokyo"},
	"sg": {Name: "sg", Latitude: 1.3521, Longitude: 103.8198, Accuracy: 50, Locale: "en-SG", Timezone: "Asia/Singapore"},
	"in": {Name: "in", Latitude: 19.0760, Longitude: 72.8777, Accuracy: 50, Locale: "en-IN", Timezone: "Asia/Kolkata"},
	"br": {Name: "br", Latitude: -23.5505, Longitude: -46.6333, Accuracy: 50, Locale: "pt-BR", Timezone: "America/Sao_Paulo"},
}

// LookupRegion returns the catalog entry for name (case-insensitive). The second
// result is false for unknown names.
func LookupRegion(name string) (Region, bool) {
	r, ok := builtinRegions[strings.ToLower(strings.TrimSpace(name))]
	return r, ok
}

// Regions returns the names of all built-in regions.
func Regions() []string {
	out := make([]string, 0, len(builtinRegions))
	for k := range builtinRegions {
		out = append(out, k)
	}
	// deterministic order
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

type regionKey struct{}

// WithRegion attaches a region label to a context so a proxy RoundTripper can
// select a matching egress proxy without threading it through every call.
func WithRegion(ctx context.Context, region string) context.Context {
	if region == "" {
		return ctx
	}
	return context.WithValue(ctx, regionKey{}, region)
}

// RegionFromContext extracts a region label previously set with WithRegion.
func RegionFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(regionKey{}).(string); ok {
		return v
	}
	return ""
}
