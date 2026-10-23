package privacy

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// ReportSchemaVersion identifies the ComplianceReport JSON schema.
const ReportSchemaVersion = "1.0.0"

// PluginID identifies privacy findings on the shared PluginID field.
const PluginID = "privacy"

// CheckStatus is the outcome of one compliance check.
type CheckStatus string

const (
	CheckPass    CheckStatus = "pass"
	CheckFail    CheckStatus = "fail"
	CheckSkipped CheckStatus = "skipped"
)

// Check is one compliance assertion within a regulation.
type Check struct {
	ID       string          `json:"id"`
	Status   CheckStatus     `json:"status"`
	Severity domain.Severity `json:"severity"`
	Message  string          `json:"message,omitempty"`
	Evidence map[string]any  `json:"evidence,omitempty"`
}

// RegulationReport is the itemised result for a single regulation.
type RegulationReport struct {
	Regulation Regulation `json:"regulation"`
	Score      int        `json:"score"`
	Passed     bool       `json:"passed"`
	Checks     []Check    `json:"checks"`
}

// Flag is an advisory observation (e.g. a DPIA or cross-border transfer) that
// does not by itself change a regulation's score.
type Flag struct {
	ID          string          `json:"id"`
	Regulation  Regulation      `json:"regulation,omitempty"`
	Severity    domain.Severity `json:"severity"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Evidence    map[string]any  `json:"evidence,omitempty"`
}

// Tracker is a non-essential tag, cookie or request observed on the page.
type Tracker struct {
	Kind     string             `json:"kind"` // tag | cookie | request
	ID       string             `json:"id"`
	Name     string             `json:"name,omitempty"`
	Category domain.TagCategory `json:"category,omitempty"`
	Host     string             `json:"host,omitempty"`
}

// StateResult is the outcome of re-scanning the page under one consent state.
type StateResult struct {
	State           ConsentState `json:"state"`
	Blocked         bool         `json:"blocked"`
	TrackersFired   []Tracker    `json:"trackersFired,omitempty"`
	CookiesSet      []string     `json:"cookiesSet,omitempty"`
	ThirdPartyHosts []string     `json:"thirdPartyHosts,omitempty"`
}

// Summary aggregates report counts.
type Summary struct {
	Regulations int `json:"regulations"`
	Passed      int `json:"passed"`
	Failed      int `json:"failed"`
	Violations  int `json:"violations"`
	Flags       int `json:"flags"`
}

// Report is the per-regulation compliance report.
type Report struct {
	SchemaVersion string             `json:"schemaVersion"`
	PolicyName    string             `json:"policyName"`
	URL           string             `json:"url"`
	GeneratedAt   time.Time          `json:"generatedAt"`
	Score         int                `json:"score"`
	Passed        bool               `json:"passed"`
	Summary       Summary            `json:"summary"`
	States        []StateResult      `json:"states"`
	Regulations   []RegulationReport `json:"regulations"`
	Flags         []Flag             `json:"flags,omitempty"`
}

// Violation is a failed check, itemised for reporting.
type Violation struct {
	Regulation Regulation      `json:"regulation"`
	RuleID     string          `json:"ruleId"`
	Severity   domain.Severity `json:"severity"`
	Message    string          `json:"message"`
	Evidence   map[string]any  `json:"evidence,omitempty"`
}

// Violations returns every failed check across all regulations.
func (r *Report) Violations() []Violation {
	var out []Violation
	for _, rr := range r.Regulations {
		for _, c := range rr.Checks {
			if c.Status != CheckFail {
				continue
			}
			out = append(out, Violation{
				Regulation: rr.Regulation,
				RuleID:     string(rr.Regulation) + "." + c.ID,
				Severity:   c.Severity,
				Message:    c.Message,
				Evidence:   c.Evidence,
			})
		}
	}
	return out
}

// Findings maps every failed check to a domain.Finding so a report can flow
// through the same persistence and SSE/WS path as plugin validators.
func (r *Report) Findings() []domain.Finding {
	violations := r.Violations()
	now := time.Now().UTC()
	out := make([]domain.Finding, 0, len(violations))
	for _, v := range violations {
		out = append(out, domain.Finding{
			ID:          domain.NewID(),
			PageURL:     r.URL,
			PluginID:    PluginID,
			RuleID:      "privacy." + v.RuleID,
			Severity:    v.Severity,
			Title:       string(v.Regulation) + ": " + v.RuleID,
			Description: v.Message,
			Evidence:    v.Evidence,
			CreatedAt:   now,
		})
	}
	return out
}

// finalize scores each regulation and rolls the results up. threshold is the
// minimum per-regulation score considered a pass.
func (r *Report) finalize(threshold int) {
	sum := Summary{}
	total := 0
	for i := range r.Regulations {
		rr := &r.Regulations[i]
		passed, failed := 0, 0
		for _, c := range rr.Checks {
			switch c.Status {
			case CheckPass:
				passed++
			case CheckFail:
				failed++
			}
		}
		evaluated := passed + failed
		if evaluated == 0 {
			rr.Score = 100
		} else {
			rr.Score = int(float64(passed)/float64(evaluated)*100 + 0.5)
		}
		rr.Passed = rr.Score >= threshold
		if rr.Passed {
			sum.Passed++
		} else {
			sum.Failed++
		}
		total += rr.Score
	}
	sum.Regulations = len(r.Regulations)
	sum.Flags = len(r.Flags)
	sum.Violations = len(r.Violations())
	if n := len(r.Regulations); n > 0 {
		r.Score = int(float64(total)/float64(n) + 0.5)
	}
	r.Passed = sum.Failed == 0
	r.Summary = sum
}

// Formatter renders a Report.
type Formatter interface {
	Format(w io.Writer, r *Report) error
}

// JSONFormatter renders the report as indented JSON.
type JSONFormatter struct{}

// Format writes the report as indented JSON.
func (JSONFormatter) Format(w io.Writer, r *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// TextFormatter renders a compact, human-readable report.
type TextFormatter struct{}

// Format writes the report as plain text.
func (TextFormatter) Format(w io.Writer, r *Report) error {
	result := "PASS"
	if !r.Passed {
		result = "FAIL"
	}
	if _, err := fmt.Fprintf(w, "Privacy compliance: %s\nResult: %s  overall score %d/100\n\n",
		r.PolicyName, result, r.Score); err != nil {
		return err
	}
	for _, rr := range r.Regulations {
		status := "FAIL"
		if rr.Passed {
			status = "PASS"
		}
		if _, err := fmt.Fprintf(w, "%-9s %3d/100  %s\n", rr.Regulation, rr.Score, status); err != nil {
			return err
		}
		for _, c := range rr.Checks {
			if c.Status != CheckFail {
				continue
			}
			if _, err := fmt.Fprintf(w, "    x %s: %s\n", c.ID, c.Message); err != nil {
				return err
			}
		}
	}
	for _, f := range r.Flags {
		if _, err := fmt.Fprintf(w, "flag [%s] %s: %s\n", f.Severity, f.ID, f.Title); err != nil {
			return err
		}
	}
	return nil
}

// compile-time interface checks.
var (
	_ Formatter = JSONFormatter{}
	_ Formatter = TextFormatter{}
)
