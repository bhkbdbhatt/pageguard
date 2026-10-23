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
