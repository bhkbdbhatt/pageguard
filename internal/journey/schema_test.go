package journey

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

func ptrBool(b bool) *bool { return &b }

func fullJourney() *Journey {
	return &Journey{
		Version:     1,
		Name:        "full",
		Description: "a fully populated journey",
		BaseURL:     "https://acme.test",
		Defaults:    Defaults{TimeoutMs: 1000, WaitMs: 10},
		Session: SessionOptions{
			UserAgent:         "ua",
			Viewport:          domain.Viewport{Width: 1, Height: 2},
			Locale:            "en-US",
			Timezone:          "UTC",
			IgnoreHTTPSErrors: true,
			CaptureDataLayer:  ptrBool(true),
		},
		Steps: []Step{{
			Action:      ActionClick,
			Description: "d",
			Selector:    "#x",
			Value:       "v",
			TimeoutMs:   500,
			WaitMs:      100,
			Expect: Expect{
				URLContains:   "a",
				TitleContains: "b",
				Status:        200,
				Tags:          []string{"t"},
				AbsentTags:    []string{"x"},
				Events:        []string{"e"},
				Cookies:       []string{"c"},
				AbsentCookies: []string{"d"},
			},
		}},
	}
}

func fullReport() *Report {
	now := time.Now().UTC()
	return &Report{
		SchemaVersion: ReportSchemaVersion,
		Journey:       "full",
		StartedAt:     now,
		FinishedAt:    now,
		DurationMs:    1,
		Passed:        false,
		Summary:       Summary{Steps: 1, Passed: 0, Failed: 1},
		Steps: []StepResult{{
			Index:       0,
			Action:      ActionNavigate,
			Selector:    "#x",
			Description: "d",
			Status:      StatusFail,
			Message:     "m",
			Error:       "e",
			DurationMs:  1,
			Checks: []CheckResult{{
				Name: "n", Status: CheckFail, Message: "m", Expected: 1, Actual: 2,
			}},
			Diff: Diff{
				NewRequests: []domain.NetworkRequest{{URL: "u"}},
				NewConsole:  []domain.ConsoleMessage{{Level: "l"}},
				NewCookies:  []domain.Cookie{{Name: "c"}},
				NewTags:     []domain.Tag{{ID: "t"}},
				NewEvents:   []string{"e"},
			},
			Screenshot: "/tmp/x.png",
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

func TestJourneyMatchesSchema(t *testing.T) {
	checkSchemaLockstep(t, "journey.schema.json", fullJourney())
}

func TestReportMatchesSchema(t *testing.T) {
	checkSchemaLockstep(t, "journeyreport.schema.json", fullReport())
}
