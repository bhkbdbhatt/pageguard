package tagpolicy

import (
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// ReportSchemaVersion identifies the ValidationReport JSON schema.
const ReportSchemaVersion = "1.0.0"

// PluginID identifies tag-policy findings on the shared PluginID field.
const PluginID = "tagpolicy"

// Status is the outcome of a single rule evaluation.
type Status string

const (
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusSkipped Status = "skipped"
)

// Report sections, one per policy concern.
const (
	SectionMustExist    = "must_exist"
	SectionMustNotExist = "must_not_exist"
	SectionValueRules   = "value_rules"
	SectionDuplicates   = "duplicates"
	SectionOrphaned     = "orphaned"
	SectionCookies      = "cookies"
	SectionConsent      = "consent"
)

// RuleResult is the pass/fail outcome of one policy rule (or one derived
// hygiene observation).
type RuleResult struct {
	ID       string         `json:"id"`
	Section  string         `json:"section"`
	Status   Status         `json:"status"`
	Severity Severity       `json:"severity"`
	Message  string         `json:"message"`
	Expected any            `json:"expected,omitempty"`
	Actual   any            `json:"actual,omitempty"`
	Evidence map[string]any `json:"evidence,omitempty"`
}

// Summary aggregates a report's results.
type Summary struct {
	Total      int              `json:"total"`
	Passed     int              `json:"passed"`
	Failed     int              `json:"failed"`
	Skipped    int              `json:"skipped"`
	BySeverity map[Severity]int `json:"bySeverity"`
}

// Report is the ValidationReport produced by validating one ScanResult against
// one policy.
type Report struct {
	SchemaVersion string       `json:"schemaVersion"`
	PolicyName    string       `json:"policyName"`
	RunID         domain.ID    `json:"runId,omitempty"`
	PageID        domain.ID    `json:"pageId,omitempty"`
	URL           string       `json:"url"`
	FinalURL      string       `json:"finalUrl,omitempty"`
	GeneratedAt   time.Time    `json:"generatedAt"`
	Passed        bool         `json:"passed"`
	Summary       Summary      `json:"summary"`
	Results       []RuleResult `json:"results"`
}

func (r *Report) add(res RuleResult) { r.Results = append(r.Results, res) }

func (r *Report) finalize() {
	by := map[Severity]int{}
	sum := Summary{}
	for _, res := range r.Results {
		sum.Total++
		switch res.Status {
		case StatusPass:
			sum.Passed++
		case StatusFail:
			sum.Failed++
			by[res.Severity]++
		case StatusSkipped:
			sum.Skipped++
		}
	}
	sum.BySeverity = by
	r.Summary = sum
	r.Passed = sum.Failed == 0
}

// Findings converts failed results into domain findings so a report can flow
// through the same persistence and streaming path as plugin validators.
func (r *Report) Findings() []domain.Finding {
	var out []domain.Finding
	pageURL := r.FinalURL
	if pageURL == "" {
		pageURL = r.URL
	}
	for _, res := range r.Results {
		if res.Status != StatusFail {
			continue
		}
		out = append(out, domain.Finding{
			ID:          domain.NewID(),
			RunID:       r.RunID,
			PageID:      r.PageID,
			PageURL:     pageURL,
			PluginID:    PluginID,
			RuleID:      PluginID + "." + res.ID,
			Severity:    res.Severity,
			Title:       res.Message,
			Description: res.Message,
			Evidence:    res.Evidence,
			CreatedAt:   r.GeneratedAt,
		})
	}
	return out
}
