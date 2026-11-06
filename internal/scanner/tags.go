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
	{ID: "microsoft-clarity", Vendor: "Microsoft", Name: "Microsoft Clarity", Category: domain.TagAnalytics,
		Domains:  []string{"clarity.ms"},
		Snippets: []string{"clarity("}},
	{ID: "heap", Vendor: "Heap", Name: "Heap", Category: domain.TagAnalytics,
		Domains:  []string{"heap.io", "heapanalytics.com"},
		Snippets: []string{"heap.load("}},
	{ID: "amplitude", Vendor: "Amplitude", Name: "Amplitude", Category: domain.TagAnalytics,
		Domains:  []string{"amplitude.com"},
		Snippets: []string{"amplitude.init("}},
	{ID: "fullstory", Vendor: "FullStory", Name: "FullStory", Category: domain.TagAnalytics,
		Domains:  []string{"fullstory.com", "fs.fullstory.com"},
		Snippets: []string{"fullstory"}},
	{ID: "logrocket", Vendor: "LogRocket", Name: "LogRocket", Category: domain.TagMonitoring,
		Domains:  []string{"logrocket.com", "lr-ingest.com"},
		Snippets: []string{"logrocket"}},
	{ID: "contentsquare", Vendor: "Contentsquare", Name: "Contentsquare", Category: domain.TagAnalytics,
		Domains:  []string{"contentsquare.net"},
		Snippets: []string{"contentsquare"}},
	{ID: "quantcast", Vendor: "Quantcast", Name: "Quantcast", Category: domain.TagAdvertising,
		Domains:  []string{"quantserve.com", "quantcast.com"},
		Snippets: []string{"quantserve"}},
	{ID: "comscore", Vendor: "Comscore", Name: "Comscore", Category: domain.TagAnalytics,
		Domains:  []string{"scorecardresearch.com", "comscore.com"},
		Snippets: []string{"scorecardresearch"}},
	{ID: "tealium", Vendor: "Tealium", Name: "Tealium iQ", Category: domain.TagTagManager,
		Domains:  []string{"tealiumiq.com", "tags.tiqcdn.com"},
		Snippets: []string{"utag.js"}},
	{ID: "optimizely", Vendor: "Optimizely", Name: "Optimizely", Category: domain.TagAnalytics,
		Domains:  []string{"optimizely.com"},
		Snippets: []string{"optimizely"}},
	{ID: "vwo", Vendor: "VWO", Name: "VWO", Category: domain.TagAnalytics,
		Domains:  []string{"visualwebsiteoptimizer.com", "vwo.com"},
		Snippets: []string{"vwo"}},
	{ID: "ab-tasty", Vendor: "AB Tasty", Name: "AB Tasty", Category: domain.TagAnalytics,
		Domains:  []string{"abtasty.com"},
		Snippets: []string{"abtasty"}},
	{ID: "braze", Vendor: "Braze", Name: "Braze", Category: domain.TagCDP,
		Domains:  []string{"braze.com", "appboy.com"},
		Snippets: []string{"appboy"}},
	{ID: "klaviyo", Vendor: "Klaviyo", Name: "Klaviyo", Category: domain.TagCDP,
		Domains:  []string{"klaviyo.com", "static.klaviyo.com"},
		Snippets: []string{"klaviyo"}},
	{ID: "mailchimp", Vendor: "Mailchimp", Name: "Mailchimp", Category: domain.TagCDP,
		Domains:  []string{"mailchimp.com", "chimpstatic.com"},
		Snippets: []string{"mailchimp"}},
	{ID: "attentive", Vendor: "Attentive", Name: "Attentive", Category: domain.TagCDP,
		Domains:  []string{"attentive.com", "attn.tv"},
		Snippets: []string{"attentive"}},
	{ID: "postscript", Vendor: "Postscript", Name: "Postscript", Category: domain.TagCDP,
		Domains:  []string{"postscript.io"},
		Snippets: []string{"postscript"}},
	{ID: "iterable", Vendor: "Iterable", Name: "Iterable", Category: domain.TagCDP,
		Domains:  []string{"iterable.com"},
		Snippets: []string{"iterable"}},
	{ID: "customer-io", Vendor: "Customer.io", Name: "Customer.io", Category: domain.TagCDP,
		Domains:  []string{"customer.io", "customerio.com"},
		Snippets: []string{"customerio"}},
	{ID: "bing-ads", Vendor: "Microsoft", Name: "Microsoft Advertising", Category: domain.TagAdvertising,
		Domains:  []string{"bat.bing.com"},
		Snippets: []string{"bat.bing.com"}},
	{ID: "pinterest-tag", Vendor: "Pinterest", Name: "Pinterest Tag", Category: domain.TagAdvertising,
		Domains:  []string{"ct.pinterest.com"},
		Snippets: []string{"pintrk("}},
	{ID: "snap-pixel", Vendor: "Snap", Name: "Snap Pixel", Category: domain.TagAdvertising,
		Domains:  []string{"sc-static.net", "tr.snapchat.com"},
		Snippets: []string{"snaptr("}},
	{ID: "reddit-pixel", Vendor: "Reddit", Name: "Reddit Pixel", Category: domain.TagAdvertising,
		Domains:  []string{"redditstatic.com", "alb.reddit.com"},
		Snippets: []string{"rdt("}},
	{ID: "twitter-pixel", Vendor: "X", Name: "X Pixel", Category: domain.TagAdvertising,
		Domains:  []string{"static.ads-twitter.com", "analytics.twitter.com"},
		Snippets: []string{"twq("}},
	{ID: "amazon-ads", Vendor: "Amazon", Name: "Amazon Ads", Category: domain.TagAdvertising,
		Domains:  []string{"amazon-adsystem.com", "assoc-amazon.com"},
		Snippets: []string{"amazon-adsystem"}},
	{ID: "criteo", Vendor: "Criteo", Name: "Criteo", Category: domain.TagAdvertising,
		Domains:  []string{"criteo.com", "criteo.net"},
		Snippets: []string{"criteo"}},
	{ID: "taboola", Vendor: "Taboola", Name: "Taboola", Category: domain.TagAdvertising,
		Domains:  []string{"taboola.com"},
		Snippets: []string{"taboola"}},
	{ID: "outbrain", Vendor: "Outbrain", Name: "Outbrain", Category: domain.TagAdvertising,
		Domains:  []string{"outbrain.com"},
		Snippets: []string{"outbrain"}},
	{ID: "yotpo", Vendor: "Yotpo", Name: "Yotpo", Category: domain.TagSocial,
		Domains:  []string{"yotpo.com"},
		Snippets: []string{"yotpo"}},
	{ID: "trustpilot", Vendor: "Trustpilot", Name: "Trustpilot", Category: domain.TagSocial,
		Domains:  []string{"trustpilot.com", "widget.trustpilot.com"},
		Snippets: []string{"trustpilot"}},
	{ID: "bazaarvoice", Vendor: "Bazaarvoice", Name: "Bazaarvoice", Category: domain.TagSocial,
		Domains:  []string{"bazaarvoice.com"},
		Snippets: []string{"bazaarvoice"}},
	{ID: "cookiebot", Vendor: "Cookiebot", Name: "Cookiebot", Category: domain.TagConsent,
		Domains:  []string{"cookiebot.com", "cookiebot.eu"},
		Snippets: []string{"cookiebot"}},
	{ID: "usercentrics", Vendor: "Usercentrics", Name: "Usercentrics", Category: domain.TagConsent,
		Domains:  []string{"usercentrics.eu", "usercentrics.com"},
		Snippets: []string{"usercentrics"}},
	{ID: "didomi", Vendor: "Didomi", Name: "Didomi", Category: domain.TagConsent,
		Domains:  []string{"didomi.io"},
		Snippets: []string{"didomi"}},
	{ID: "osano", Vendor: "Osano", Name: "Osano", Category: domain.TagConsent,
		Domains:  []string{"osano.com"},
		Snippets: []string{"osano"}},
	{ID: "trustarc", Vendor: "TrustArc", Name: "TrustArc", Category: domain.TagConsent,
		Domains:  []string{"trustarc.com"},
		Snippets: []string{"trustarc"}},
	{ID: "sourcepoint", Vendor: "Sourcepoint", Name: "Sourcepoint", Category: domain.TagConsent,
		Domains:  []string{"sourcepoint.com", "sourcepoint.mgr.consensu.org"},
		Snippets: []string{"sourcepoint"}},
	{ID: "qualtrics", Vendor: "Qualtrics", Name: "Qualtrics", Category: domain.TagAnalytics,
		Domains:  []string{"qualtrics.com"},
		Snippets: []string{"qualtrics"}},
	{ID: "drift", Vendor: "Drift", Name: "Drift", Category: domain.TagSocial,
		Domains:  []string{"drift.com", "js.driftt.com"},
		Snippets: []string{"drift"}},
	{ID: "intercom", Vendor: "Intercom", Name: "Intercom", Category: domain.TagSocial,
		Domains:  []string{"intercom.io", "intercomcdn.com", "intercom.com"},
		Snippets: []string{"intercom"}},
	{ID: "zendesk", Vendor: "Zendesk", Name: "Zendesk", Category: domain.TagSocial,
		Domains:  []string{"zendesk.com", "zdassets.com"},
		Snippets: []string{"zendesk"}},
	{ID: "gorgias", Vendor: "Gorgias", Name: "Gorgias", Category: domain.TagSocial,
		Domains:  []string{"gorgias.chat", "gorgias.com"},
		Snippets: []string{"gorgias"}},
	{ID: "nosto", Vendor: "Nosto", Name: "Nosto", Category: domain.TagAnalytics,
		Domains:  []string{"nosto.com"},
		Snippets: []string{"nosto"}},
	{ID: "dynamic-yield", Vendor: "Dynamic Yield", Name: "Dynamic Yield", Category: domain.TagAnalytics,
		Domains:  []string{"dynamicyield.com"},
		Snippets: []string{"dynamicyield"}},
	{ID: "bloomreach", Vendor: "Bloomreach", Name: "Bloomreach", Category: domain.TagCDP,
		Domains:  []string{"bloomreach.com"},
		Snippets: []string{"bloomreach"}},
	{ID: "emarsys", Vendor: "Emarsys", Name: "Emarsys", Category: domain.TagCDP,
		Domains:  []string{"emarsys.net", "scarabresearch.com"},
		Snippets: []string{"emarsys"}},
	{ID: "snowplow", Vendor: "Snowplow", Name: "Snowplow", Category: domain.TagAnalytics,
		Domains:  []string{"snowplow.io", "snowplowanalytics.com"},
		Snippets: []string{"snowplow"}},
	{ID: "rudderstack", Vendor: "RudderStack", Name: "RudderStack", Category: domain.TagCDP,
		Domains:  []string{"rudderstack.com", "rudderlabs.com"},
		Snippets: []string{"rudderanalytics"}},
	{ID: "mparticle", Vendor: "mParticle", Name: "mParticle", Category: domain.TagCDP,
		Domains:  []string{"mparticle.com"},
		Snippets: []string{"mparticle"}},
	{ID: "prebid", Vendor: "Prebid", Name: "Prebid.js", Category: domain.TagAdvertising,
		Domains:  []string{"prebid.org"},
		Snippets: []string{"pbjs"}},
}
