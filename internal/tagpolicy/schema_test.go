package tagpolicy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

func fullPolicy() *Policy {
	zero := 0.0
	return &Policy{
		Version:     1,
		Name:        "full",
		Description: "a fully populated policy",
		Settings: Settings{
			UnexpectedTags:    SevInfo,
			DuplicateTags:     SevMedium,
			UnexpectedCookies: SevLow,
			MaxOccurrences:    2,
		},
		MustExist: []Expectation{{
			ID:          "e1",
			Description: "d",
			Severity:    SevHigh,
			Target:      TargetCookie,
			Match: Match{
				Tag: "t", TagPattern: "a.*", Name: "n", NamePattern: "b.*",
				Category: domain.TagAnalytics, URLPattern: "c", Host: "h", Domain: "dom", Event: "ev",
			},
			Attributes: &CookieAttr{
				Secure: ptrBool(true), HTTPOnly: ptrBool(true), SameSite: []string{"Lax"},
				FirstParty: ptrBool(true), Session: ptrBool(false), Category: "functional",
			},
		}},
		MustNotExist: []Expectation{{
			ID: "e2", Severity: SevLow, Target: TargetTag, Match: Match{Tag: "x"},
		}},
		ValueRules: []ValueRule{{
			ID: "v1", Description: "d", Severity: SevMedium, Variable: "a.b",
			Type: "number", Enum: []any{1, "two"}, Min: &zero, Max: &zero,
			Pattern: "^x$", Required: true, WhenEvent: "ev", RequiredIfEvent: "ev2",
		}},
		ConsentRequirements: ConsentRequirement{
			RequireCMP:            true,
			CMPTags:               []string{"onetrust"},
			GatedCategories:       []domain.TagCategory{domain.TagAdvertising},
			GatedCookieCategories: []string{"marketing"},
			Severity:              SevCritical,
		},
	}
}

func fullReport() *Report {
	return &Report{
		SchemaVersion: ReportSchemaVersion,
		PolicyName:    "full",
		RunID:         domain.NewID(),
		PageID:        domain.NewID(),
		URL:           "https://acme.com/",
		FinalURL:      "https://acme.com/",
		GeneratedAt:   time.Now().UTC(),
		Passed:        false,
		Summary: Summary{
			Total: 1, Passed: 0, Failed: 1, Skipped: 0,
			BySeverity: map[Severity]int{SevHigh: 1},
		},
		Results: []RuleResult{{
			ID: "r1", Section: SectionMustExist, Status: StatusFail, Severity: SevHigh,
			Message: "m", Expected: map[string]any{"a": 1}, Actual: "x",
			Evidence: map[string]any{"b": 2},
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

func TestPolicyMatchesSchema(t *testing.T) {
	checkSchemaLockstep(t, "tagpolicy.schema.json", fullPolicy())
}

func TestReportMatchesSchema(t *testing.T) {
	checkSchemaLockstep(t, "validationreport.schema.json", fullReport())
}
