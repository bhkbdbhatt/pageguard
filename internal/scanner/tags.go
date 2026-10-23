package scanner

import (
	"sort"
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
)

// signature describes how to recognise a martech tag.
type signature struct {
	ID       string
	Vendor   string
	Name     string
	Category domain.TagCategory
	// Domains are hostname substrings that, when present in a request, strongly
	// indicate the tag. Matches here yield high confidence.
	Domains []string
	// Snippets are lowercase substrings searched in the document HTML/scripts;
	// matches yield lower confidence.
	Snippets []string
}

// Detector recognises known martech tags from an Observation. It is
// intentionally data-driven so the signature table can move into a plugin.
type Detector struct {
	signatures []signature
}

// NewDetector returns a detector seeded with a starter signature set. Module 2
// replaces this with the full tag registry; the interface stays the same.
func NewDetector() *Detector {
	return &Detector{signatures: defaultSignatures}
}

// Detect returns de-duplicated tags observed in the page, most confident first.
func (d *Detector) Detect(obs *domain.Observation) []domain.Tag {
	if obs == nil {
		return nil
	}
	htmlLower := strings.ToLower(obs.HTML)
	hosts := make([]string, 0, len(obs.Requests))
	for _, r := range obs.Requests {
		h := r.Host
		if h == "" {
			h = hostOf(r.URL)
		}
		hosts = append(hosts, strings.ToLower(h))
	}

	byID := map[string]domain.Tag{}
	for _, sig := range d.signatures {
		var evidence string
		confidence := 0.0

		for _, dom := range sig.Domains {
			for _, h := range hosts {
				if strings.Contains(h, dom) {
					evidence = "request host " + h
					confidence = 0.95
					break
				}
			}
			if confidence > 0 {
				break
			}
		}
		if confidence == 0 {
			for _, snip := range sig.Snippets {
				if strings.Contains(htmlLower, strings.ToLower(snip)) {
					evidence = "page source contains " + snip
					confidence = 0.6
					break
				}
			}
		}
		if confidence == 0 {
			continue
		}
		t := domain.Tag{
			ID: sig.ID, Vendor: sig.Vendor, Name: sig.Name,
			Category: sig.Category, Evidence: evidence, Confidence: confidence,
		}
		if existing, ok := byID[sig.ID]; !ok || t.Confidence > existing.Confidence {
			byID[sig.ID] = t
		}
	}

	out := make([]domain.Tag, 0, len(byID))
	for _, t := range byID {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		return out[i].ID < out[j].ID
	})
	return out
}

var defaultSignatures = []signature{
	{ID: "google-analytics-4", Vendor: "Google", Name: "Google Analytics 4", Category: domain.TagAnalytics,
		Domains:  []string{"google-analytics.com", "analytics.google.com", "googletagmanager.com"},
		Snippets: []string{"gtag(", "googletagmanager.com/gtag/js", "/g/collect"}},
	{ID: "google-tag-manager", Vendor: "Google", Name: "Google Tag Manager", Category: domain.TagTagManager,
		Domains:  []string{"googletagmanager.com"},
		Snippets: []string{"googletagmanager.com/gtm.js", "dataLayer"}},
	{ID: "google-ads", Vendor: "Google", Name: "Google Ads / DoubleClick", Category: domain.TagAdvertising,
		Domains: []string{"doubleclick.net", "googleadservices.com", "googlesyndication.com"}},
	{ID: "meta-pixel", Vendor: "Meta", Name: "Meta Pixel", Category: domain.TagAdvertising,
		Domains:  []string{"connect.facebook.net", "facebook.com/tr"},
		Snippets: []string{"fbq(", "connect.facebook.net/en_US/fbevents.js"}},
	{ID: "hotjar", Vendor: "Hotjar", Name: "Hotjar", Category: domain.TagAnalytics,
		Domains:  []string{"hotjar.com", "hotjar.io"},
		Snippets: []string{"hj(", "_hjsettings"}},
	{ID: "segment", Vendor: "Twilio", Name: "Segment", Category: domain.TagCDP,
		Domains:  []string{"segment.io", "segment.com", "cdn.segment.com"},
		Snippets: []string{"analytics.load(", "analytics.track("}},
	{ID: "mixpanel", Vendor: "Mixpanel", Name: "Mixpanel", Category: domain.TagAnalytics,
		Domains:  []string{"mixpanel.com", "mxpanel.com"},
		Snippets: []string{"mixpanel.init("}},
	{ID: "linkedin-insight", Vendor: "LinkedIn", Name: "LinkedIn Insight Tag", Category: domain.TagAdvertising,
		Domains:  []string{"snap.licdn.com", "px.ads.linkedin.com"},
		Snippets: []string{"_linkedin_partner_id"}},
	{ID: "tiktok-pixel", Vendor: "TikTok", Name: "TikTok Pixel", Category: domain.TagAdvertising,
		Domains:  []string{"analytics.tiktok.com"},
		Snippets: []string{"ttq.load("}},
	{ID: "adobe-analytics", Vendor: "Adobe", Name: "Adobe Analytics", Category: domain.TagAnalytics,
		Domains:  []string{"omtrdc.net", "adobedtm.com", "2o7.net"},
		Snippets: []string{"s.t(", "adobe_datalayer"}},
	{ID: "onetrust", Vendor: "OneTrust", Name: "OneTrust Consent", Category: domain.TagConsent,
		Domains:  []string{"cookielaw.org", "onetrust.com"},
		Snippets: []string{"otconsent", "onetrust"}},
	{ID: "hubspot", Vendor: "HubSpot", Name: "HubSpot", Category: domain.TagAnalytics,
		Domains:  []string{"hs-scripts.com", "hubspot.com", "hs-analytics.net"},
		Snippets: []string{"_hsq"}},
	{ID: "cloudflare-insights", Vendor: "Cloudflare", Name: "Cloudflare Web Analytics", Category: domain.TagAnalytics,
		Domains: []string{"cloudflareinsights.com"}},
	{ID: "sentry", Vendor: "Sentry", Name: "Sentry", Category: domain.TagMonitoring,
		Domains:  []string{"sentry-cdn.com", "sentry.io"},
		Snippets: []string{"sentry.init(", "sentry_dsn"}},
	{ID: "newrelic", Vendor: "New Relic", Name: "New Relic Browser", Category: domain.TagMonitoring,
		Domains:  []string{"nr-data.net", "js-agent.newrelic.com"},
		Snippets: []string{"newrelic"}},
}
