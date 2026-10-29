package privacy

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/journey"
	"github.com/pageguard/pageguard/internal/scanner"
)

// BrowserRescanner renders a page in a real browser once per consent state so
// the privacy engine can compare tracker behaviour pre- and post-consent.
type BrowserRescanner struct {
	Factory journey.SessionFactory
	// URL is the page under evaluation.
	URL string
	// Detector is optional; a default detector is used when nil.
	Detector *scanner.Detector
}

var (
	acceptSelectors = []string{
		"#onetrust-accept-btn-handler",
		"#truste-consent-button",
		"[aria-label*='Accept' i]",
		"[id*='accept' i]",
		"[class*='accept' i]",
		"[data-testid*='accept' i]",
		"button:has-text('Accept all')",
		"button:has-text('Accept')",
		".cc-btn.cc-allow",
	}
	rejectSelectors = []string{
		"#onetrust-reject-all-handler",
		"#onetrust-reject-all",
		"[aria-label*='Reject' i]",
		"[id*='reject' i]",
		"[class*='reject' i]",
		"[data-testid*='reject' i]",
		"button:has-text('Reject all')",
		"button:has-text('Decline')",
		"button:has-text('Refuse')",
		".cc-btn.cc-deny",
	}
	necessarySelectors = append(append([]string{}, rejectSelectors...),
		"button:has-text('Only necessary')",
		"button:has-text('Necessary only')",
		"button:has-text('Necessary')",
		"[id*='necessary' i]",
	)
)

// Rescan opens a fresh browser session, navigates to the page, applies the
// consent choice for the state, and captures the resulting observation.
func (b *BrowserRescanner) Rescan(ctx context.Context, state ConsentState) (*Observation, error) {
	if b.Factory == nil {
		return nil, fmt.Errorf("privacy: no browser session factory configured")
	}
	if strings.TrimSpace(b.URL) == "" {
		return nil, fmt.Errorf("privacy: no url configured")
	}
	detector := b.Detector
	if detector == nil {
		detector = scanner.NewDetector()
	}

	sess, err := b.Factory.Open(ctx, journey.SessionOptions{
		GlobalPrivacyControl: state == StateGPC,
	})
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	initial, err := sess.Perform(ctx, journey.Step{
		Action: journey.ActionNavigate, Value: b.URL, TimeoutMs: 25000, WaitMs: 500,
	})
	if err != nil {
		return nil, err
	}

	var selectors []string
	switch state {
	case StateAcceptAll:
		selectors = acceptSelectors
	case StateRejectAll:
		selectors = rejectSelectors
	case StateNecessaryOnly:
		selectors = necessarySelectors
	}

	final := initial
	if len(selectors) > 0 {
		if clicker, ok := sess.(journey.ConsentClicker); ok {
			if st, _, cerr := clicker.Consent(ctx, selectors, 1500, 800); cerr == nil && st != nil {
				final = st
			}
		} else {
			for _, sel := range selectors {
				st, perr := sess.Perform(ctx, journey.Step{
					Action: journey.ActionClick, Selector: sel, TimeoutMs: 1500, WaitMs: 400,
				})
				if perr == nil && st != nil {
					final = st
					break
				}
			}
		}
	}

	obs := &Observation{
		URL:               b.URL,
		FinalURL:          final.FinalURL,
		Tags:              detector.Detect(observationFrom(final)),
		Cookies:           final.Cookies,
		Requests:          final.Requests,
		DataLayer:         final.DataLayer,
		Links:             extractLinks(final),
		BannerPresent:     bannerPresent(initial.HTML),
		BannerHasReject:   containsAny(initial.HTML, "reject", "decline", "refuse", "deny"),
		BannerHasSettings: containsAny(initial.HTML, "preferences", "settings", "manage", "customi"),
		GPCRespected:      state == StateGPC,
	}
	if obs.FinalURL == "" {
		obs.FinalURL = final.URL
	}
	return obs, nil
}

func observationFrom(st *journey.PageState) *domain.Observation {
	if st == nil {
		return &domain.Observation{}
	}
	return &domain.Observation{
		URL:         st.URL,
		FinalURL:    st.FinalURL,
		StatusCode:  st.Status,
		Title:       st.Title,
		HTML:        st.HTML,
		ContentType: "text/html",
		Requests:    st.Requests,
		Cookies:     st.Cookies,
		Console:     st.Console,
		DataLayer:   st.DataLayer,
		Rendered:    true,
	}
}

var (
	hrefDouble = regexp.MustCompile(`(?i)href\s*=\s*"([^"]+)"`)
	hrefSingle = regexp.MustCompile(`(?i)href\s*=\s*'([^']+)'`)
)

func extractLinks(st *journey.PageState) []string {
	if st == nil {
		return nil
	}
	base := st.FinalURL
	if base == "" {
		base = st.URL
	}
	baseURL, _ := url.Parse(base)
	seen := map[string]bool{}
	var out []string
	for _, re := range []*regexp.Regexp{hrefDouble, hrefSingle} {
		for _, m := range re.FindAllStringSubmatch(st.HTML, 200) {
			raw := strings.TrimSpace(m[1])
			if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "javascript:") ||
				strings.HasPrefix(raw, "mailto:") || strings.HasPrefix(raw, "tel:") {
				continue
			}
			if baseURL != nil {
				if u, err := url.Parse(raw); err == nil {
					raw = baseURL.ResolveReference(u).String()
				}
			}
			if !seen[raw] {
				seen[raw] = true
				out = append(out, raw)
			}
			if len(out) >= 200 {
				return out
			}
		}
	}
	return out
}

var cmpMarkers = []string{
	"onetrust", "cookiebot", "cookieconsent", "cookie consent", "truste",
	"quantcast", "didomi", "usercentrics", "osano", "iubenda", "klaro",
	"consent-manager", "cmplz", "borlabs", "termly", "cookieyes", "gdpr",
}

func bannerPresent(html string) bool {
	h := strings.ToLower(html)
	for _, m := range cmpMarkers {
		if strings.Contains(h, m) {
			return true
		}
	}
	hasCookie := strings.Contains(h, "cookie")
	hasConsent := strings.Contains(h, "consent")
	hasChrome := strings.Contains(h, "banner") || strings.Contains(h, "modal") ||
		strings.Contains(h, "dialog") || strings.Contains(h, "notice") || strings.Contains(h, "popup")
	return hasCookie && hasConsent && hasChrome
}

func containsAny(html string, needles ...string) bool {
	h := strings.ToLower(html)
	for _, n := range needles {
		if strings.Contains(h, n) {
			return true
		}
	}
	return false
}

// compile-time interface check.
var _ Rescanner = (*BrowserRescanner)(nil)
