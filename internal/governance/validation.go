package governance

import (
	"context"
	"sort"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/rules"
)

// RuleEvaluation is the API representation of one rule result.
type RuleEvaluation struct {
	RuleID   string          `json:"ruleId"`
	Name     string          `json:"name"`
	Vendor   string          `json:"vendor"`
	Target   string          `json:"target"`
	Severity domain.Severity `json:"severity"`
	Status   string          `json:"status"`
	Reason   string          `json:"reason,omitempty"`
	Evidence map[string]any  `json:"evidence,omitempty"`
}

// PageValidation groups rule results for one page.
type PageValidation struct {
	URL        string           `json:"url"`
	FinalURL   string           `json:"finalUrl,omitempty"`
	PageID     domain.ID        `json:"pageId"`
	StatusCode int              `json:"statusCode"`
	DurationMs int64            `json:"durationMs"`
	Error      string           `json:"error,omitempty"`
	Passed     bool             `json:"passed"`
	Evaluated  int              `json:"evaluated"`
	Results    []RuleEvaluation `json:"results"`
}

// ValidationReport is the validation view of a run.
type ValidationReport struct {
	RunID             domain.ID          `json:"runId"`
	ScanID            domain.ID          `json:"scanId"`
	GeneratedAt       time.Time          `json:"generatedAt"`
	ConsentSimulation string             `json:"consentSimulation,omitempty"`
	HealthScore       float64            `json:"healthScore"`
	ConsentScore      float64            `json:"consentScore"`
	Beacons           int                `json:"beacons"`
	Pages             []PageValidation   `json:"pages"`
	Rules             []rules.RuleStat   `json:"ruleStats"`
	VendorScores      map[string]float64 `json:"vendorScores"`
	Findings          []*domain.Finding  `json:"findings"`
}

// Validation evaluates all configured rules against a run's checkpoints.
func (s *Service) Validation(ctx context.Context, runID domain.ID) (*ValidationReport, error) {
	run, scan, projectRules, results, err := s.runContext(ctx, runID)
	if err != nil {
		return nil, err
	}
	byPage, agg := evaluate(projectRules, results)
	findings, _ := s.store.ListFindings(ctx, domain.FindingFilter{RunID: runID})
	_, consent := consentFromResults(results)
	out := &ValidationReport{
		RunID: runID, ScanID: run.ScanID, GeneratedAt: time.Now().UTC(),
		HealthScore: agg.HealthScore, ConsentScore: consent.Score, Beacons: consent.Beacons,
		Pages: []PageValidation{}, Rules: agg.RuleStats, VendorScores: agg.VendorScores,
		Findings: findings,
	}
	if scan != nil {
		out.ConsentSimulation = scan.Config.ConsentSimulation
	}
	if agg.Evaluated == 0 {
		out.HealthScore = rules.HealthFromFindings(findings)
	}
	sorted := append([]*domain.ScanResult(nil), results...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].URL < sorted[j].URL })
	for _, sr := range sorted {
		rs := byPage[sr.URL]
		passed, evaluated := rules.PageHealth(rs)
		page := PageValidation{
			URL: sr.URL, FinalURL: sr.FinalURL, PageID: sr.PageID, StatusCode: sr.StatusCode,
			DurationMs: sr.DurationMs, Error: sr.Error, Passed: passed, Evaluated: evaluated,
			Results: []RuleEvaluation{},
		}
		for _, res := range rs {
			if res.Rule == nil {
				continue
			}
			page.Results = append(page.Results, RuleEvaluation{
				RuleID: string(res.Rule.ID), Name: res.Rule.Name, Vendor: res.Rule.Vendor,
				Target: string(res.Rule.Target), Severity: res.Rule.Severity,
				Status: string(res.Status), Reason: res.Reason, Evidence: res.Evidence,
			})
		}
		out.Pages = append(out.Pages, page)
	}
	sort.Slice(out.Rules, func(i, j int) bool {
		if out.Rules[i].Rule.Vendor == out.Rules[j].Rule.Vendor {
			return out.Rules[i].Rule.Name < out.Rules[j].Rule.Name
		}
		return out.Rules[i].Rule.Vendor < out.Rules[j].Rule.Vendor
	})
	return out, nil
}
