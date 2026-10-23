package journey

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

func sampleReport() *Report {
	return &Report{
		SchemaVersion: ReportSchemaVersion,
		Journey:       "checkout",
		StartedAt:     time.Unix(0, 0).UTC(),
		FinishedAt:    time.Unix(1, 0).UTC(),
		DurationMs:    1000,
		Steps: []StepResult{
			{Index: 0, Action: ActionNavigate, Status: StatusPass, Message: "navigate ok", DurationMs: 10},
			{
				Index: 1, Action: ActionClick, Selector: "#buy", Status: StatusFail,
				Message: "click failed", DurationMs: 20,
				Checks: []CheckResult{{
					Name: "tag:meta-pixel", Status: CheckFail,
					Message: `expected tag "meta-pixel" was not detected`,
				}},
				Diff: Diff{
					NewTags:    []domain.Tag{{ID: "meta-pixel"}},
					NewCookies: []domain.Cookie{{Name: "sid"}},
					NewEvents:  []string{"page_view"},
				},
				Screenshot: "/tmp/step-02-click.png",
			},
		},
	}
}

func TestFinalize(t *testing.T) {
	r := sampleReport()
	r.finalize()
	if r.Passed {
		t.Error("expected not passed")
	}
	if r.Summary != (Summary{Steps: 2, Passed: 1, Failed: 1}) {
		t.Errorf("summary = %+v", r.Summary)
	}
}

func TestFinalizeAllPass(t *testing.T) {
	r := &Report{Steps: []StepResult{{Status: StatusPass}, {Status: StatusPass}}}
	r.finalize()
	if !r.Passed || r.Summary.Failed != 0 || r.Summary.Passed != 2 {
		t.Errorf("unexpected: %+v", r.Summary)
	}
}

func TestJSONFormatter(t *testing.T) {
	r := sampleReport()
	r.finalize()
	var buf bytes.Buffer
	if err := (JSONFormatter{}).Format(&buf, r); err != nil {
		t.Fatalf("Format: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got["schemaVersion"] != ReportSchemaVersion {
		t.Errorf("schemaVersion = %v", got["schemaVersion"])
	}
	steps, _ := got["steps"].([]any)
	if len(steps) != 2 {
		t.Errorf("steps = %v", got["steps"])
	}
}

func TestTextFormatter(t *testing.T) {
	r := sampleReport()
	r.finalize()
	var buf bytes.Buffer
	if err := (TextFormatter{}).Format(&buf, r); err != nil {
		t.Fatalf("Format: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Journey: checkout",
		"FAIL",
		"step 1",
		"x tag:meta-pixel",
		"+ tags: meta-pixel",
		"+ cookies: sid",
		"+ events: page_view",
		"screenshot: /tmp/step-02-click.png",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
