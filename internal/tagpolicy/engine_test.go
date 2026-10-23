package tagpolicy

import (
	"encoding/json"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func ptrBool(b bool) *bool { return &b }

func tag(id, name string, cat domain.TagCategory) domain.Tag {
	return domain.Tag{ID: id, Name: name, Vendor: "V", Category: cat, Evidence: "ev", Confidence: 0.9}
}

func baseResult() *domain.ScanResult {
	return &domain.ScanResult{
		RunID:    domain.NewID(),
		PageID:   domain.NewID(),
		URL:      "https://acme.com/checkout",
		FinalURL: "https://acme.com/checkout",
		Tags: []domain.Tag{
			tag("google-tag-manager", "Google Tag Manager", domain.TagTagManager),
			tag("ga4", "Google Analytics 4", domain.TagAnalytics),
		},
		Requests: []domain.NetworkRequest{
			{URL: "https://www.googletagmanager.com/gtag/js?id=G-1", Host: "www.googletagmanager.com", Method: "GET", Status: 200, ResourceType: "script"},
		},
		Cookies: []domain.Cookie{
			{Name: "sid", Domain: ".acme.com", Secure: true, HTTPOnly: true, SameSite: "Lax", FirstParty: true},
		},
		DataLayer: []domain.DataLayerEntry{
			{Name: "dataLayer[0]", Value: json.RawMessage(`{"event":"purchase","page":{"type":"checkout"},"cart":{"total":42,"currency":"EUR"},"consent":{"analytics":true}}`)},
		},
	}
}

func mustParse(t *testing.T, doc string) *Policy {
	t.Helper()
	p, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("parse policy: %v", err)
	}
	return p
}

func resultByID(r *Report, id string) (RuleResult, bool) {
	for _, res := range r.Results {
		if res.ID == id {
			return res, true
		}
	}
	return RuleResult{}, false
}

func TestMustExist(t *testing.T) {
	p := mustParse(t, `
version: 1
name: p
must_exist:
  - id: gtm
    target: tag
    match: { tag: google-tag-manager }
  - id: missing-tag
    severity: high
    target: tag
    match: { tag: hotjar }
  - id: script
    target: request
    match: { url_pattern: "googletagmanager\\.com/gtag/js" }
  - id: purchase
    target: event
    match: { event: purchase }
  - id: sid
    target: cookie
    match: { name: sid }
`)
	r := Validate(baseResult(), p)

	for _, id := range []string{"gtm", "script", "purchase", "sid"} {
		res, ok := resultByID(r, id)
		if !ok || res.Status != StatusPass {
			t.Errorf("%s: got %+v, want pass", id, res)
		}
	}
	res, _ := resultByID(r, "missing-tag")
	if res.Status != StatusFail || res.Severity != SevHigh {
		t.Errorf("missing-tag: got %+v, want fail/high", res)
	}
	if r.Passed {
		t.Error("report should not pass with a failing rule")
	}
}

func TestMustNotExist(t *testing.T) {
	p := mustParse(t, `
version: 1
name: p
must_not_exist:
  - id: no-ads
    target: tag
    match: { category: advertising }
  - id: no-legacy
    target: cookie
    match: { name: flashsession }
`)
	r := Validate(baseResult(), p)
	if res, _ := resultByID(r, "no-ads"); res.Status != StatusPass {
		t.Errorf("no-ads: got %+v, want pass", res)
	}
	if res, _ := resultByID(r, "no-legacy"); res.Status != StatusPass {
		t.Errorf("no-legacy: got %+v, want pass", res)
	}

	sr := baseResult()
	sr.Tags = append(sr.Tags, tag("google-ads", "Google Ads", domain.TagAdvertising))
	r = Validate(sr, p)
	if res, _ := resultByID(r, "no-ads"); res.Status != StatusFail {
		t.Errorf("no-ads with advertising tag: got %+v, want fail", res)
	}
}

func TestCookieAttributes(t *testing.T) {
	p := mustParse(t, `
version: 1
name: p
must_exist:
  - id: sid
    target: cookie
    match: { name: sid }
    attributes:
      secure: true
      httpOnly: true
      sameSite: [Lax, Strict]
      category: functional
`)
	r := Validate(baseResult(), p)
	if res, _ := resultByID(r, "sid"); res.Status != StatusPass {
		t.Errorf("sid: got %+v, want pass", res)
	}
	if _, ok := resultByID(r, "sid-attributes"); ok {
		t.Error("expected no attribute violation")
	}

	sr := baseResult()
	sr.Cookies[0].Secure = false
	r = Validate(sr, p)
	res, ok := resultByID(r, "sid-attributes")
	if !ok || res.Status != StatusFail {
		t.Fatalf("expected sid-attributes failure, got %+v", res)
	}
}

func TestValueRules(t *testing.T) {
	p := mustParse(t, `
version: 1
name: p
value_rules:
  - id: page-type
    variable: page.type
    type: string
    enum: [home, product, checkout]
    required: true
  - id: cart-total
    variable: cart.total
    type: number
    min: 0
    max: 1000
    required_if_event: purchase
  - id: currency
    variable: cart.currency
    type: string
    pattern: "^[A-Z]{3}$"
    when_event: purchase
  - id: consent
    variable: consent.analytics
    type: boolean
    required: true
`)
	r := Validate(baseResult(), p)
	for _, id := range []string{"page-type", "cart-total", "currency", "consent"} {
		if res, ok := resultByID(r, id); !ok || res.Status != StatusPass {
			t.Errorf("%s: got %+v, want pass", id, res)
		}
	}

	// out-of-range and wrong enum
	sr := baseResult()
	sr.DataLayer[0].Value = json.RawMessage(`{"event":"purchase","page":{"type":"nope"},"cart":{"total":5000,"currency":"eur"},"consent":{"analytics":true}}`)
	r = Validate(sr, p)
	if res, _ := resultByID(r, "page-type"); res.Status != StatusFail {
		t.Errorf("page-type: got %+v, want fail", res)
	}
	if res, _ := resultByID(r, "cart-total"); res.Status != StatusFail {
		t.Errorf("cart-total: got %+v, want fail", res)
	}
	if res, _ := resultByID(r, "currency"); res.Status != StatusFail {
		t.Errorf("currency: got %+v, want fail", res)
	}
}

func TestValueRuleEvents(t *testing.T) {
	p := mustParse(t, `
version: 1
name: p
value_rules:
  - id: cart-total
    variable: cart.total
    type: number
    required_if_event: purchase
  - id: optional
    variable: cart.discount
    type: number
    when_event: purchase
`)
	// purchase did not fire: required_if_event is skipped, optional is skipped.
	sr := baseResult()
	sr.DataLayer[0].Value = json.RawMessage(`{"event":"view","page":{"type":"home"}}`)
	r := Validate(sr, p)
	if res, _ := resultByID(r, "cart-total"); res.Status != StatusSkipped {
		t.Errorf("cart-total (no purchase): got %+v, want skipped", res)
	}
	if res, _ := resultByID(r, "optional"); res.Status != StatusSkipped {
		t.Errorf("optional (no purchase): got %+v, want skipped", res)
	}

	// purchase fired but cart.total missing: required_if_event fails.
	sr.DataLayer[0].Value = json.RawMessage(`{"event":"purchase","cart":{}}`)
	r = Validate(sr, p)
	if res, _ := resultByID(r, "cart-total"); res.Status != StatusFail {
		t.Errorf("cart-total (purchase, missing): got %+v, want fail", res)
	}
}

func TestDuplicates(t *testing.T) {
	p := mustParse(t, `
version: 1
name: p
settings: { duplicate_tags: high, max_occurrences: 1 }
`)
	sr := baseResult()
	sr.Tags = append(sr.Tags, tag("ga4", "Google Analytics 4", domain.TagAnalytics))
	r := Validate(sr, p)
	res, ok := resultByID(r, "duplicate-tag-ga4")
	if !ok || res.Status != StatusFail || res.Severity != SevHigh {
		t.Fatalf("expected duplicate-tag-ga4 fail/high, got %+v", res)
	}
}

func TestOrphans(t *testing.T) {
	p := mustParse(t, `
version: 1
name: p
settings: { unexpected_tags: low }
must_exist:
  - id: gtm
    target: tag
    match: { tag: google-tag-manager }
`)
	r := Validate(baseResult(), p)
	res, ok := resultByID(r, "orphan-tag-ga4")
	if !ok || res.Status != StatusFail || res.Severity != SevLow {
		t.Fatalf("expected orphan-tag-ga4 fail/low, got %+v", res)
	}
	if _, ok := resultByID(r, "orphan-tag-google-tag-manager"); ok {
		t.Error("declared gtm tag should not be orphaned")
	}

	off := mustParse(t, `
version: 1
name: p
settings: { unexpected_tags: off }
`)
	r = Validate(baseResult(), off)
	for _, res := range r.Results {
		if res.Section == SectionOrphaned {
			t.Errorf("orphan checks should be disabled, got %+v", res)
		}
	}
}

func TestUnexpectedCookies(t *testing.T) {
	p := mustParse(t, `
version: 1
name: p
settings: { unexpected_cookies: low }
must_exist:
  - id: sid
    target: cookie
    match: { name: sid }
`)
	sr := baseResult()
	sr.Cookies = append(sr.Cookies, domain.Cookie{Name: "_ga", Domain: ".acme.com"})
	r := Validate(sr, p)
	res, ok := resultByID(r, "unexpected-cookie-_ga")
	if !ok || res.Status != StatusFail || res.Severity != SevLow {
		t.Fatalf("expected unexpected-cookie-_ga fail/low, got %+v", res)
	}
	if _, ok := resultByID(r, "unexpected-cookie-sid"); ok {
		t.Error("declared sid cookie should not be unexpected")
	}
}

func TestConsent(t *testing.T) {
	p := mustParse(t, `
version: 1
name: p
consent_requirements:
  require_cmp: true
  cmp_tags: [onetrust]
  gated_categories: [advertising]
  severity: critical
`)
	// No CMP, but no gated tags either: only the CMP check fails.
	r := Validate(baseResult(), p)
	res, ok := resultByID(r, "consent-cmp")
	if !ok || res.Status != StatusFail || res.Severity != SevCritical {
		t.Fatalf("expected consent-cmp fail/critical, got %+v", res)
	}

	// Gated advertising tag present without a CMP: both checks fail.
	sr := baseResult()
	sr.Tags = append(sr.Tags, tag("google-ads", "Google Ads", domain.TagAdvertising))
	r = Validate(sr, p)
	if res, _ := resultByID(r, "consent-gated-advertising"); res.Status != StatusFail {
		t.Errorf("expected consent-gated-advertising fail, got %+v", res)
	}

	// CMP present: everything passes.
	sr = baseResult()
	sr.Tags = append(sr.Tags, tag("onetrust", "OneTrust", domain.TagConsent), tag("google-ads", "Google Ads", domain.TagAdvertising))
	r = Validate(sr, p)
	if res, _ := resultByID(r, "consent-cmp"); res.Status != StatusPass {
		t.Errorf("consent-cmp with CMP: got %+v, want pass", res)
	}
	if _, ok := resultByID(r, "consent-gated-advertising"); ok {
		t.Error("gated checks should not run when a CMP is present")
	}
}

func TestSummaryAndFindings(t *testing.T) {
	p := mustParse(t, `
version: 1
name: full
settings: { unexpected_tags: off, unexpected_cookies: off }
must_exist:
  - id: gtm
    target: tag
    match: { tag: google-tag-manager }
  - id: missing
    severity: high
    target: tag
    match: { tag: hotjar }
`)
	r := Validate(baseResult(), p)
	if r.Summary.Total != 2 || r.Summary.Passed != 1 || r.Summary.Failed != 1 {
		t.Fatalf("unexpected summary: %+v", r.Summary)
	}
	if r.Summary.BySeverity[SevHigh] != 1 {
		t.Errorf("bySeverity high = %d, want 1", r.Summary.BySeverity[SevHigh])
	}
	if r.SchemaVersion != ReportSchemaVersion || r.PolicyName != "full" {
		t.Errorf("unexpected report header: %+v", r)
	}

	findings := r.Findings()
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(findings))
	}
	f := findings[0]
	if f.PluginID != PluginID || f.RuleID != PluginID+".missing" || f.Severity != SevHigh {
		t.Errorf("unexpected finding: %+v", f)
	}
	if f.RunID != baseResult().RunID && f.RunID == "" {
		t.Error("finding missing run id")
	}
}
