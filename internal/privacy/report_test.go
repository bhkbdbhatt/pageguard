package privacy

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestReportViolationsAndFindings(t *testing.T) {
	rep, err := Evaluate(context.Background(), testConfig(), badRescanner())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	v := rep.Violations()
	if len(v) == 0 {
		t.Fatal("expected violations")
	}
	if !strings.Contains(v[0].RuleID, ".") {
		t.Errorf("rule id = %q, want regulation-prefixed", v[0].RuleID)
	}

	f := rep.Findings()
	if len(f) != len(v) {
		t.Fatalf("findings %d != violations %d", len(f), len(v))
	}
	if f[0].PluginID != PluginID {
		t.Errorf("plugin = %q, want %q", f[0].PluginID, PluginID)
	}
	if !strings.HasPrefix(f[0].RuleID, "privacy.") {
		t.Errorf("rule = %q, want privacy. prefix", f[0].RuleID)
	}
	if f[0].PageURL != rep.URL {
		t.Errorf("page url = %q", f[0].PageURL)
	}
}

func TestFindingsEmptyWhenPassing(t *testing.T) {
	rep, err := Evaluate(context.Background(), testConfig(), goodRescanner())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if got := rep.Findings(); len(got) != 0 {
		t.Errorf("expected no findings, got %d", len(got))
	}
}

func TestFinalizeScoring(t *testing.T) {
	rep := &Report{
		Regulations: []RegulationReport{
			{
				Regulation: GDPR,
				Checks: []Check{
					{ID: "a", Status: CheckPass},
					{ID: "b", Status: CheckPass},
					{ID: "c", Status: CheckFail},
					{ID: "d", Status: CheckSkipped},
				},
			},
			{
				Regulation: EPrivacy,
				Checks:     []Check{{ID: "e", Status: CheckPass}},
			},
		},
	}
	rep.finalize(100)

	gdpr := rep.Regulations[0]
	if gdpr.Score != 67 {
		t.Errorf("gdpr score = %d, want 67 (2/3)", gdpr.Score)
	}
	if gdpr.Passed {
		t.Error("gdpr should not pass (score < threshold)")
	}
	if !rep.Regulations[1].Passed {
		t.Error("eprivacy should pass")
	}
	if rep.Score != 84 {
		t.Errorf("overall score = %d, want 84 ((67+100)/2 rounded)", rep.Score)
	}
	if rep.Passed {
		t.Error("overall should fail")
	}
	if rep.Summary.Regulations != 2 || rep.Summary.Passed != 1 || rep.Summary.Failed != 1 {
		t.Errorf("summary = %+v", rep.Summary)
	}
	if rep.Summary.Violations != 1 {
		t.Errorf("violations = %d, want 1", rep.Summary.Violations)
	}
}

func TestFinalizeNoChecks(t *testing.T) {
	rep := &Report{Regulations: []RegulationReport{{Regulation: LGPD}}}
	rep.finalize(100)
	if rep.Regulations[0].Score != 100 || !rep.Regulations[0].Passed {
		t.Errorf("empty checks should score 100/pass: %+v", rep.Regulations[0])
	}
	if rep.Score != 100 {
		t.Errorf("overall = %d", rep.Score)
	}
}

func TestJSONFormatter(t *testing.T) {
	rep, _ := Evaluate(context.Background(), testConfig(), goodRescanner())
	var buf bytes.Buffer
	if err := (JSONFormatter{}).Format(&buf, rep); err != nil {
		t.Fatalf("Format: %v", err)
	}
	var decoded Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.SchemaVersion != ReportSchemaVersion || decoded.PolicyName != "test" {
		t.Errorf("round-trip mismatch: %+v", decoded)
	}
}

func TestTextFormatter(t *testing.T) {
	good, _ := Evaluate(context.Background(), testConfig(), goodRescanner())
	var ok bytes.Buffer
	if err := (TextFormatter{}).Format(&ok, good); err != nil {
		t.Fatalf("Format: %v", err)
	}
	if !strings.Contains(ok.String(), "PASS") {
		t.Errorf("expected PASS in output:\n%s", ok.String())
	}

	bad, _ := Evaluate(context.Background(), testConfig(), badRescanner())
	var fail bytes.Buffer
	if err := (TextFormatter{}).Format(&fail, bad); err != nil {
		t.Fatalf("Format: %v", err)
	}
	if !strings.Contains(fail.String(), "FAIL") {
		t.Errorf("expected FAIL in output:\n%s", fail.String())
	}
}
