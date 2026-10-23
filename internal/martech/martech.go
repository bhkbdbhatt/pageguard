// Package martech contains shared vendor knowledge used by governance rules and
// reporting. It intentionally keeps a small, maintainable host/tag map rather
// than shipping a large third-party fingerprint database.
package martech

import (
	"net/url"
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
)

// Vendor describes a known marketing/analytics vendor.
type Vendor struct {
	Key      string
	Display  string
	Hosts    []string
	TagHints []string
}

// Vendors is the canonical vendor list surfaced in the dashboard.
var Vendors = []Vendor{
	{
		Key: "adobe", Display: "Adobe",
		Hosts:    []string{"adobedc.net", "demdex.net", "omtrdc.net", "2o7.net", "everesttech.net", "adobe.com"},
		TagHints: []string{"adobe", "ecid", "launch", "at.js", "appmeasurement"},
	},
	{
		Key: "ga4", Display: "GA4",
		Hosts:    []string{"google-analytics.com", "analytics.google.com", "googletagmanager.com", "google.com"},
		TagHints: []string{"google", "ga4", "gtag", "analytics", "gtm"},
	},
	{
		Key: "meta", Display: "Meta",
		Hosts:    []string{"facebook.com", "facebook.net", "fbcdn.net", "connect.facebook.net"},
		TagHints: []string{"meta", "facebook", "fbevents", "fbq"},
	},
	{
		Key: "onetrust", Display: "OneTrust",
		Hosts:    []string{"onetrust.com", "cookielaw.org", "optanon.blob.core.windows.net"},
		TagHints: []string{"onetrust", "optanon", "cookielaw"},
	},
}

// DetectHost returns the display name of the vendor owning host, or "".
func DetectHost(host string) string {
	h := strings.ToLower(strings.TrimSpace(host))
	for _, v := range Vendors {
		for _, suffix := range v.Hosts {
			if h == suffix || strings.HasSuffix(h, "."+suffix) {
				return v.Display
			}
		}
	}
	return ""
}

// DetectTag returns the display name of the vendor behind a detected tag.
func DetectTag(tag domain.Tag) string {
	haystack := strings.ToLower(strings.Join([]string{tag.Vendor, tag.Name, tag.ID, tag.Evidence}, " "))
	for _, v := range Vendors {
		if strings.EqualFold(tag.Vendor, v.Display) || strings.EqualFold(tag.Vendor, v.Key) {
			return v.Display
		}
		for _, hint := range v.TagHints {
			if strings.Contains(haystack, hint) {
				return v.Display
			}
		}
	}
	if tag.Vendor != "" {
		return tag.Vendor
	}
	return "Other"
}

// MatchVendor reports whether a host or tag belongs to the named vendor.
func MatchVendor(vendor, value string) bool {
	v, ok := lookup(vendor)
	if !ok {
		return strings.Contains(strings.ToLower(value), strings.ToLower(vendor))
	}
	haystack := strings.ToLower(value)
	for _, h := range v.Hosts {
		if haystack == h || strings.HasSuffix(haystack, "."+h) || strings.Contains(haystack, h) {
			return true
		}
	}
	for _, hint := range v.TagHints {
		if strings.Contains(haystack, hint) {
			return true
		}
	}
	return false
}

// HostURL parses a URL and returns its lower-case host.
func HostURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// IsTrackerRequest reports whether a request is likely a tag/beacon rather than
// a static asset or the page document itself.
func IsTrackerRequest(req domain.NetworkRequest) bool {
	switch strings.ToLower(req.ResourceType) {
	case "image", "script", "xhr", "fetch", "ping", "beacon", "websocket":
	default:
		if DetectHost(req.Host) == "" {
			return false
		}
	}
	if DetectHost(req.Host) != "" {
		return true
	}
	if !req.ThirdParty {
		return false
	}
	if strings.Contains(strings.ToLower(req.URL), "collect") ||
		strings.Contains(strings.ToLower(req.URL), "beacon") ||
		strings.Contains(strings.ToLower(req.URL), "track") {
		return true
	}
	return req.ResourceType == "xhr" || req.ResourceType == "fetch" || req.ResourceType == "image"
}

// IsConsentHost reports whether a request belongs to a consent management
// platform.
func IsConsentHost(host string) bool { return DetectHost(host) == "OneTrust" }

// IsEssentialCookie reports whether a cookie is typically required for core
// site functionality.
func IsEssentialCookie(name string) bool {
	n := strings.ToLower(name)
	for _, essential := range []string{"session", "csrf", "xsrf", "auth", "token", "consent", "optanon", "onetrust", "cookie"} {
		if strings.Contains(n, essential) {
			return true
		}
	}
	return false
}

// NonEssentialCookies returns cookies that look like tracking cookies.
func NonEssentialCookies(cookies []domain.Cookie) []domain.Cookie {
	out := make([]domain.Cookie, 0, len(cookies))
	for _, c := range cookies {
		if IsEssentialCookie(c.Name) {
			continue
		}
		if !c.FirstParty || DetectHost(c.Domain) != "" {
			out = append(out, c)
		}
	}
	return out
}

func lookup(vendor string) (Vendor, bool) {
	for _, v := range Vendors {
		if strings.EqualFold(vendor, v.Key) || strings.EqualFold(vendor, v.Display) {
			return v, true
		}
	}
	return Vendor{}, false
}
