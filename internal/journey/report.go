package journey

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// ReportSchemaVersion identifies the JourneyReport JSON schema.
const ReportSchemaVersion = "1.0.0"

// Status is the outcome of a step.
type Status string

const (
	StatusPass  Status = "pass"
	StatusFail  Status = "fail"
	StatusError Status = "error"
)

// CheckStatus is the outcome of a single expectation check.
type CheckStatus string

const (
	CheckPass CheckStatus = "pass"
	CheckFail CheckStatus = "fail"
)

// Report is the step-by-step result of running a journey.
type Report struct {
	SchemaVersion string       `json:"schemaVersion"`
	Journey       string       `json:"journey"`
	StartedAt     time.Time    `json:"startedAt"`
	FinishedAt    time.Time    `json:"finishedAt"`
	DurationMs    int64        `json:"durationMs"`
	Passed        bool         `json:"passed"`
	Summary       Summary      `json:"summary"`
	Steps         []StepResult `json:"steps"`
}

// Summary aggregates the step outcomes.
type Summary struct {
	Steps  int `json:"steps"`
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

// StepResult is the outcome of executing and validating one step.
type StepResult struct {
	Index       int           `json:"index"`
	Action      string        `json:"action"`
	Selector    string        `json:"selector,omitempty"`
	Description string        `json:"description,omitempty"`
	Status      Status        `json:"status"`
	Message     string        `json:"message,omitempty"`
	Error       string        `json:"error,omitempty"`
	DurationMs  int64         `json:"durationMs"`
	Checks      []CheckResult `json:"checks,omitempty"`
	Diff        Diff          `json:"diff"`
	// Screenshot is the path to a PNG captured when the step failed.
	Screenshot string `json:"screenshot,omitempty"`
}

// CheckResult is one expectation assertion.
type CheckResult struct {
	Name     string      `json:"name"`
	Status   CheckStatus `json:"status"`
	Message  string      `json:"message,omitempty"`
	Expected any         `json:"expected,omitempty"`
	Actual   any         `json:"actual,omitempty"`
}

// Diff describes what changed in the page between the previous step and this
// one: newly observed network requests, console messages, cookies, tags and
// data-layer events.
type Diff struct {
	NewRequests []domain.NetworkRequest `json:"newRequests,omitempty"`
	NewConsole  []domain.ConsoleMessage `json:"newConsole,omitempty"`
	NewCookies  []domain.Cookie         `json:"newCookies,omitempty"`
	NewTags     []domain.Tag            `json:"newTags,omitempty"`
	NewEvents   []string                `json:"newEvents,omitempty"`
}

func (r *Report) finalize() {
	sum := Summary{Steps: len(r.Steps)}
	for _, s := range r.Steps {
		switch s.Status {
		case StatusPass:
			sum.Passed++
		default:
			sum.Failed++
		}
	}
	r.Summary = sum
	r.Passed = sum.Failed == 0
}

// Formatter renders a Report to a writer.
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
	if _, err := fmt.Fprintf(w, "Journey: %s\nResult:  %s (%d steps: %d passed, %d failed) in %s\n\n",
		r.Journey, result, r.Summary.Steps, r.Summary.Passed, r.Summary.Failed,
		time.Duration(r.DurationMs)*time.Millisecond); err != nil {
		return err
	}
	for _, s := range r.Steps {
		label := s.Action
		if s.Selector != "" {
			label += " " + s.Selector
		}
		if _, err := fmt.Fprintf(w, "step %d  %-24s %-5s %s\n", s.Index+1, label,
			strings.ToUpper(string(s.Status)), s.Message); err != nil {
			return err
		}
		for _, c := range s.Checks {
			if c.Status == CheckPass {
				continue
			}
			if _, err := fmt.Fprintf(w, "    x %s: %s\n", c.Name, c.Message); err != nil {
				return err
			}
		}
		if len(s.Diff.NewTags) > 0 {
			fmt.Fprintf(w, "    + tags: %s\n", joinTags(s.Diff.NewTags))
		}
		if len(s.Diff.NewCookies) > 0 {
			fmt.Fprintf(w, "    + cookies: %s\n", joinCookieNames(s.Diff.NewCookies))
		}
		if len(s.Diff.NewEvents) > 0 {
			fmt.Fprintf(w, "    + events: %s\n", strings.Join(s.Diff.NewEvents, ", "))
		}
		if s.Screenshot != "" {
			fmt.Fprintf(w, "    screenshot: %s\n", s.Screenshot)
		}
	}
	return nil
}

func joinTags(tags []domain.Tag) string {
	ids := make([]string, len(tags))
	for i, t := range tags {
		ids[i] = t.ID
	}
	return strings.Join(ids, ", ")
}

func joinCookieNames(cookies []domain.Cookie) string {
	names := make([]string, len(cookies))
	for i, c := range cookies {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}

// compile-time interface checks.
var (
	_ Formatter = JSONFormatter{}
	_ Formatter = TextFormatter{}
)
