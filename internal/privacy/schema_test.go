package privacy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

func ptrBool(b bool) *bool { return &b }

func fullConfig() *Config {
	return &Config{
		Version:     1,
		Name:        "full",
		Description: "a fully populated policy",
		Regulations: []Regulation{GDPR, CCPA, LGPD, EPrivacy},
		CookieRules: []CookieRule{
			{NamePattern: "^_ga", Category: domain.TagAnalytics, Essential: true},
		},
		SubjectRights: []SubjectRight{
			{ID: "access", Name: "Right of access", URLPattern: "(?i)/access"},
		},
		Settings: Settings{
			ConsentStates:           []ConsentState{StateAcceptAll, StateRejectAll},
			EssentialCategories:     []domain.TagCategory{domain.TagConsent},
			EssentialCookies:        []string{"sid"},
			EssentialCookiePatterns: []string{"^__Host-"},
			EssentialHosts:          []string{"cdn.acme.com"},
			MarketingCategories:     []domain.TagCategory{domain.TagAdvertising},
			SaleCategories:          []domain.TagCategory{domain.TagAdvertising},
			CMPTags:                 []string{"onetrust"},
			DPIA:                    []DPIATrigger{{ID: "d", Description: "desc", Categories: []domain.TagCategory{domain.TagAdvertising}, MinTrackers: 1}},
			CrossBorderHosts:        []string{"adnetwork.io"},
			OptOutURLPatterns:       []string{"(?i)do-not-sell"},
			TrackThirdPartyRequests: ptrBool(true),
			PassThreshold:           50,
		},
	}
}

func fullReport() *Report {
	now := time.Now().UTC()
	return &Report{
		SchemaVersion: ReportSchemaVersion,
		PolicyName:    "full",
		URL:           "https://acme.test",
		GeneratedAt:   now,
		Score:         50,
		Passed:        false,
		Summary:       Summary{Regulations: 1, Passed: 0, Failed: 1, Violations: 1, Flags: 1},
		States: []StateResult{{
			State:           StateAcceptAll,
			Blocked:         false,
			TrackersFired:   []Tracker{{Kind: "tag", ID: "t", Name: "n", Category: domain.TagAnalytics, Host: "h"}},
			CookiesSet:      []string{"c"},
			ThirdPartyHosts: []string{"h"},
		}},
		Regulations: []RegulationReport{{
			Regulation: GDPR,
			Score:      0,
			Passed:     false,
			Checks: []Check{{
				ID: "c", Status: CheckFail, Severity: domain.SevHigh, Message: "m", Evidence: map[string]any{"k": "v"},
			}},
		}},
		Flags: []Flag{{
			ID: "f", Regulation: GDPR, Severity: domain.SevMedium, Title: "t", Description: "d", Evidence: map[string]any{"k": "v"},
		}},
	}
}

func checkSchemaLockstep(t *testing.T, schemaName string, v any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "schemas", schemaName))
	if err != nil {
		t.Fatalf("read schema %s: %v", schemaName, err)
	}
	var schema struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("parse schema %s: %v", schemaName, err)
	}

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	for _, req := range schema.Required {
		if _, ok := obj[req]; !ok {
			t.Errorf("%s requires %q but the struct does not emit it", schemaName, req)
		}
	}
	for k := range obj {
		if _, ok := schema.Properties[k]; !ok {
			t.Errorf("%s: struct emits %q which the schema does not declare", schemaName, k)
		}
	}
}

func TestConfigMatchesSchema(t *testing.T) {
	checkSchemaLockstep(t, "privacy.schema.json", fullConfig())
}

func TestReportMatchesSchema(t *testing.T) {
	checkSchemaLockstep(t, "privacyreport.schema.json", fullReport())
}
