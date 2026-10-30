package rules

import (
	"encoding/json"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func resultFor(t *testing.T, r *domain.AuditRule, sr *domain.ScanResult) Result {
	t.Helper()
	return Evaluate(r, sr)
}

func TestRequestRules(t *testing.T) {
	sr := &domain.ScanResult{
		URL: "https://shop.example/checkout",
		Requests: []domain.NetworkRequest{{
			URL:  "https://adobedc.net/collect?email=user@example.com&events=cart",
			Host: "adobedc.net",
		}},
	}
	rule := &domain.AuditRule{
		ID: domain.NewID(), Name: "Email must not leak", Enabled: true,
		Vendor: "Adobe", Target: domain.RuleTargetRequest, Parameter: "email",
		Operator: domain.RuleOpNotMatches, Expected: `^[^@]+@[^@]+\.[^@]+$`,
		Severity: domain.SevCritical,
	}
	res := resultFor(t, rule, sr)
	if res.Status != StatusFail {
		t.Fatalf("expected leak failure, got %s (%s)", res.Status, res.Reason)
	}
	findings := Findings("run", "page", sr.URL, []Result{res})
	if len(findings) != 1 || findings[0].Severity != domain.SevCritical {
		t.Fatalf("findings = %+v", findings)
	}

	rule.Operator = domain.RuleOpExists
	rule.Parameter = "events"
	rule.Expected = ""
	if res := resultFor(t, rule, sr); res.Status != StatusPass {
		t.Fatalf("expected cart event pass, got %s", res.Status)
	}
}

func TestEventRules(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"event": "purchase",
		"ecommerce": map[string]any{
			"transaction_id": "T-1",
		},
	})
	sr := &domain.ScanResult{URL: "https://shop.example/thanks", DataLayer: []domain.DataLayerEntry{
		{Name: "purchase", Value: payload},
	}}
	rule := &domain.AuditRule{
		ID: domain.NewID(), Name: "Transaction id", Enabled: true,
		Target: domain.RuleTargetEvent, Event: "purchase",
		Parameter: "ecommerce.transaction_id", Operator: domain.RuleOpExists,
		Severity: domain.SevMedium,
	}
	if res := resultFor(t, rule, sr); res.Status != StatusPass {
		t.Fatalf("expected event pass, got %s (%s)", res.Status, res.Reason)
	}
	rule.Parameter = "ecommerce.currency"
	if res := resultFor(t, rule, sr); res.Status != StatusFail {
		t.Fatalf("expected missing currency failure, got %s", res.Status)
	}
}

func TestTagAndCookieRules(t *testing.T) {
	sr := &domain.ScanResult{
		URL:  "https://shop.example",
		Tags: []domain.Tag{{ID: "ecid", Vendor: "Adobe", Name: "Adobe ECID"}},
		Cookies: []domain.Cookie{
			{Name: "OptanonConsent", Domain: "shop.example", FirstParty: true},
			{Name: "AMCV_123", Domain: "adobe.com", FirstParty: false},
		},
	}
	tag := &domain.AuditRule{
		ID: domain.NewID(), Name: "Adobe tag", Enabled: true, Vendor: "Adobe",
		Target: domain.RuleTargetTag, Operator: domain.RuleOpExists, Severity: domain.SevLow,
	}
	if res := resultFor(t, tag, sr); res.Status != StatusPass {
		t.Fatalf("tag exists = %s", res.Status)
	}
	cookie := &domain.AuditRule{
		ID: domain.NewID(), Name: "Adobe cookie", Enabled: true, Vendor: "Adobe",
		Target: domain.RuleTargetCookie, Parameter: "AMCV", Operator: domain.RuleOpExists,
		Severity: domain.SevMedium,
	}
	if res := resultFor(t, cookie, sr); res.Status != StatusPass {
		t.Fatalf("cookie exists = %s", res.Status)
	}
}

func TestURLScopeAndDisabled(t *testing.T) {
	sr := &domain.ScanResult{URL: "https://shop.example/blog", Tags: []domain.Tag{{ID: "x", Name: "Ads", Vendor: "Meta"}}}
	rule := &domain.AuditRule{
		ID: domain.NewID(), Name: "Checkout only", Enabled: true,
		URLPattern: `/checkout$`, Target: domain.RuleTargetTag,
		Operator: domain.RuleOpExists, Severity: domain.SevLow,
	}
	if res := resultFor(t, rule, sr); res.Status != StatusSkipped {
		t.Fatalf("expected skip, got %s", res.Status)
	}
	rule.URLPattern = ""
	rule.Enabled = false
	res := EvaluateAll([]*domain.AuditRule{rule}, sr)
	if len(res) != 0 {
		t.Fatalf("disabled rule evaluated: %+v", res)
	}
}

func TestSummarize(t *testing.T) {
	r := &domain.AuditRule{ID: "r1", Name: "A", Vendor: "Adobe", Severity: domain.SevHigh}
	byPage := map[string][]Result{
		"a": {{Rule: r, Status: StatusPass}},
		"b": {{Rule: r, Status: StatusFail}},
		"c": {{Rule: r, Status: StatusSkipped}},
	}
	agg := Summarize([]*domain.AuditRule{r}, byPage)
	if agg.HealthScore != 50 || agg.Passed != 1 || agg.Failed != 1 {
		t.Fatalf("aggregate = %+v", agg)
	}
	if agg.VendorScores["Adobe"] != 50 {
		t.Fatalf("vendor scores = %+v", agg.VendorScores)
	}
}

func TestBeaconRules(t *testing.T) {
	sr := &domain.ScanResult{
		URL: "https://shop.example/products",
		Requests: []domain.NetworkRequest{
			{URL: "https://connect.facebook.net/fbevents?fired=1", Host: "connect.facebook.net", ResourceType: "script", ThirdParty: true},
			{URL: "https://shop.example/logo.png", Host: "shop.example", ResourceType: "image"},
			{URL: "https://adobedc.net/ee/collection?ev=purchase", Host: "adobedc.net", ResourceType: "xhr", ThirdParty: true},
		},
	}
	// A Meta beacon may not fire anywhere: not_exists over all Meta beacons.
	block := &domain.AuditRule{
		ID: domain.NewID(), Name: "No Meta beacons", Enabled: true, Vendor: "Meta",
		Target: domain.RuleTargetBeacon, Operator: domain.RuleOpNotExists, Severity: domain.SevCritical,
	}
	if res := resultFor(t, block, sr); res.Status != StatusFail {
		t.Fatalf("expected beacon failure, got %s (%s)", res.Status, res.Reason)
	}
	// Adobe collect beacons must not carry the (absent here) ev parameter.
	param := &domain.AuditRule{
		ID: domain.NewID(), Name: "Adobe ev must be purchase", Enabled: true, Vendor: "Adobe",
		Target: domain.RuleTargetBeacon, Parameter: "ev", Operator: domain.RuleOpEquals, Expected: "purchase",
	}
	if res := resultFor(t, param, sr); res.Status != StatusPass {
		t.Fatalf("expected beacon param pass, got %s (%s)", res.Status, res.Reason)
	}
	// First-party asset requests are excluded from beacon evaluation.
	firstParty := &domain.AuditRule{
		ID: domain.NewID(), Name: "Own asset is not a beacon", Enabled: true,
		Target: domain.RuleTargetBeacon, Operator: domain.RuleOpNotExists,
	}
	if res := resultFor(t, firstParty, sr); res.Status != StatusFail {
		t.Fatalf("expected external beacons to fail not_exists, got %s (%s)", res.Status, res.Reason)
	}
}

func TestNetworkRules(t *testing.T) {
	sr := &domain.ScanResult{
		URL:      "https://shop.example",
		FinalURL: "https://shop.example/home",
		Requests: []domain.NetworkRequest{
			{URL: "https://shop.example/", Host: "shop.example", ResourceType: "document"},
			{URL: "https://cdn.example/analytics.js", Host: "cdn.example", ResourceType: "script", ThirdParty: true},
		},
	}
	// Unlike beacons, network rules see the document request.
	doc := &domain.AuditRule{
		ID: domain.NewID(), Name: "Document served by first party", Enabled: true,
		Target: domain.RuleTargetNetwork, Operator: domain.RuleOpExists,
	}
	if res := resultFor(t, doc, sr); res.Status != StatusPass {
		t.Fatalf("expected network exists pass, got %s (%s)", res.Status, res.Reason)
	}
	host := &domain.AuditRule{
		ID: domain.NewID(), Name: "No CDN script", Enabled: true, Vendor: "GA4",
		Target: domain.RuleTargetNetwork, Operator: domain.RuleOpNotExists,
	}
	if res := resultFor(t, host, sr); res.Status != StatusPass {
		t.Fatalf("expected vendor-filtered network pass, got %s (%s)", res.Status, res.Reason)
	}
}

func TestPageRules(t *testing.T) {
	sr := &domain.ScanResult{URL: "https://shop.example/amp/product", FinalURL: "https://shop.example/product"}
	// Canonical URL enforcement: AMP redirect must be canonicalized.
	rule := &domain.AuditRule{
		ID: domain.NewID(), Name: "Canonical product URL", Enabled: true,
		Target: domain.RuleTargetPage, Operator: domain.RuleOpNotMatches, Expected: "/amp/",
		Severity: domain.SevHigh,
	}
	if res := resultFor(t, rule, sr); res.Status != StatusPass {
		t.Fatalf("expected page pass (amp resolved away), got %s (%s)", res.Status, res.Reason)
	}
	// Redirect left the AMP suffix in place: final URL keeps /amp/ ? must fail.
	redirected := &domain.ScanResult{
		URL: "https://shop.example/amp/product", FinalURL: "https://shop.example/amp/product/canonical",
	}
	if res := resultFor(t, rule, redirected); res.Status != StatusFail {
		t.Fatalf("expected page fail, got %s (%s)", res.Status, res.Reason)
	}
}

func TestTargetAliases(t *testing.T) {
	tgt, ok := domain.CanonicalizeTarget(" datalayer ")
	if !ok || tgt != domain.RuleTargetEvent {
		t.Fatalf("datalayer alias = %q ok=%v", tgt, ok)
	}
	tgt, ok = domain.CanonicalizeTarget("Script")
	if !ok || tgt != domain.RuleTargetTag {
		t.Fatalf("script alias = %q ok=%v", tgt, ok)
	}
	if _, ok := domain.CanonicalizeTarget("any"); ok {
		t.Fatal("any target should be rejected")
	}
}
