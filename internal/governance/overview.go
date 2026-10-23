package governance

import (
	"context"
	"sort"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/martech"
)

// KPI is the top-line set of dashboard counters.
type KPI struct {
	PagesAudited   int     `json:"pagesAudited"`
	PagesTarget    int     `json:"pagesTarget"`
	HealthScore    float64 `json:"healthScore"`
	Beacons        int     `json:"beacons"`
	Critical       int     `json:"criticalFindings"`
	Compliance     float64 `json:"complianceScore"`
	Findings       int     `json:"findings"`
	ConsentLeaks   int     `json:"consentLeaks"`
	PagesValidated int     `json:"pagesValidated"`
}

// TrendPoint is one day of the seven-day health trend.
type TrendPoint struct {
	Date    string  `json:"date"`
	Health  float64 `json:"health"`
	Consent float64 `json:"consent"`
	Runs    int     `json:"runs"`
}

// VendorSlice is a vendor's share of observed tags and beacons.
type VendorSlice struct {
	Vendor  string  `json:"vendor"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// Violation is a recent finding rendered in the overview feed.
type Violation struct {
	ID          domain.ID       `json:"id"`
	RunID       domain.ID       `json:"runId"`
	PageURL     string          `json:"pageUrl"`
	RuleID      string          `json:"ruleId"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Vendor      string          `json:"vendor"`
	Severity    domain.Severity `json:"severity"`
	At          time.Time       `json:"at"`
}

// Overview is the complete dashboard payload.
type Overview struct {
	Project     *domain.Project `json:"project,omitempty"`
	GeneratedAt time.Time       `json:"generatedAt"`
	KPI         KPI             `json:"kpi"`
	Trend       []TrendPoint    `json:"trend"`
	Vendors     []VendorSlice   `json:"vendors"`
	Violations  []Violation     `json:"violations"`
	LatestRun   *domain.Run     `json:"latestRun,omitempty"`
	InFlight    int             `json:"inFlight"`
}

// Overview computes the dashboard for a project. When projectID is empty the
// first project is used.
func (s *Service) Overview(ctx context.Context, projectID domain.ID) (*Overview, error) {
	project, err := s.project(ctx, projectID)
	if err != nil {
		if err == domain.ErrNotFound {
			return &Overview{GeneratedAt: time.Now().UTC(), Trend: sevenDays(nil), Vendors: []VendorSlice{}, Violations: []Violation{}}, nil
		}
		return nil, err
	}
	runs, scans, err := s.projectRuns(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	out := &Overview{
		Project:     project,
		GeneratedAt: time.Now().UTC(),
		Trend:       sevenDays(runs),
		Vendors:     []VendorSlice{},
		Violations:  []Violation{},
	}
	for _, r := range runs {
		if r.Status == domain.RunRunning || r.Status == domain.RunQueued {
			out.InFlight++
		}
	}
	if len(runs) > 0 {
		out.LatestRun = runs[0]
	}

	latestByScan := map[domain.ID]*domain.Run{}
	for _, r := range runs {
		if _, ok := latestByScan[r.ScanID]; !ok {
			latestByScan[r.ScanID] = r
		}
	}
	for scanID, r := range latestByScan {
		out.KPI.PagesAudited += r.Stats.PagesScanned
		if scan := scans[scanID]; scan != nil {
			out.KPI.PagesTarget += scan.Config.MaxPages
		}
	}
	if out.LatestRun != nil {
		out.KPI.HealthScore = out.LatestRun.Stats.HealthScore
		out.KPI.Beacons = out.LatestRun.Stats.Beacons
		out.KPI.Critical = out.LatestRun.Stats.CriticalFindings
		out.KPI.Compliance = out.LatestRun.Stats.ConsentScore
		out.KPI.Findings = out.LatestRun.Stats.Findings
	}

	var results []*domain.ScanResult
	if out.LatestRun != nil {
		results, _ = s.store.ListScanResults(ctx, out.LatestRun.ID, 0)
		byPage, summary := evaluate(s.projectRules(ctx, project.ID), results)
		out.KPI.PagesValidated = len(byPage)
		if out.LatestRun.Stats.HealthScore == 0 && summary.Evaluated > 0 {
			out.KPI.HealthScore = summary.HealthScore
		}
	}
	out.Vendors = vendorSlices(results)
	if out.LatestRun != nil {
		findings, _ := s.store.ListFindings(ctx, domain.FindingFilter{RunID: out.LatestRun.ID, Limit: 100})
		for _, f := range findings {
			out.Violations = append(out.Violations, Violation{
				ID: f.ID, RunID: f.RunID, PageURL: f.PageURL, RuleID: f.RuleID,
				Title: f.Title, Description: f.Description, Vendor: vendorFor(f),
				Severity: f.Severity, At: f.CreatedAt,
			})
			if f.Severity == domain.SevCritical {
				out.KPI.Critical++
			}
		}
	}
	_, consent := consentFromResults(results)
	out.KPI.ConsentLeaks = consent.Leaks
	if out.LatestRun != nil && out.LatestRun.Stats.Beacons == 0 {
		out.KPI.Beacons = consent.Beacons
	}
	if out.LatestRun != nil && out.LatestRun.Stats.ConsentScore == 0 {
		out.KPI.Compliance = consent.Score
	}
	return out, nil
}

func (s *Service) projectRules(ctx context.Context, projectID domain.ID) []*domain.AuditRule {
	all, err := s.store.ListRules(ctx, projectID)
	if err != nil {
		return nil
	}
	return all
}

func sevenDays(runs []*domain.Run) []TrendPoint {
	now := time.Now().UTC()
	days := make([]time.Time, 0, 7)
	health := map[string][]float64{}
	consent := map[string][]float64{}
	count := map[string]int{}
	for i := 6; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		key := dayKey(d)
		days = append(days, d)
		health[key] = []float64{}
		consent[key] = []float64{}
	}
	for _, r := range runs {
		if !r.Status.Terminal() {
			continue
		}
		at := r.CreatedAt
		if r.FinishedAt != nil {
			at = *r.FinishedAt
		}
		key := dayKey(at)
		if _, ok := health[key]; !ok {
			continue
		}
		health[key] = append(health[key], r.Stats.HealthScore)
		consent[key] = append(consent[key], r.Stats.ConsentScore)
		count[key]++
	}
	out := make([]TrendPoint, 0, len(days))
	for _, d := range days {
		key := dayKey(d)
		out = append(out, TrendPoint{
			Date: key, Health: avg(health[key]), Consent: avg(consent[key]), Runs: count[key],
		})
	}
	return out
}

func vendorSlices(results []*domain.ScanResult) []VendorSlice {
	counts := map[string]int{}
	seenHosts := map[string]bool{}
	for _, sr := range results {
		for _, tag := range sr.Tags {
			v := martech.DetectTag(tag)
			if v != "" {
				counts[v]++
			}
		}
		for _, req := range sr.Requests {
			v := martech.DetectHost(req.Host)
			if v == "" || seenHosts[req.Host] {
				continue
			}
			seenHosts[req.Host] = true
			counts[v]++
		}
	}
	total := 0
	for _, n := range counts {
		total += n
	}
	out := make([]VendorSlice, 0, len(counts))
	for v, n := range counts {
		pct := 0.0
		if total > 0 {
			pct = float64(n) / float64(total) * 100
		}
		out = append(out, VendorSlice{Vendor: v, Count: n, Percent: pct})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Vendor < out[j].Vendor
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

func vendorFor(f *domain.Finding) string {
	if f == nil {
		return ""
	}
	if v, ok := f.Evidence["vendor"].(string); ok && v != "" {
		return v
	}
	if ev, ok := f.Evidence["samples"].([]map[string]any); ok {
		for _, sample := range ev {
			if source, ok := sample["source"].(string); ok {
				if v := martech.DetectHost(martech.HostURL(source)); v != "" {
					return v
				}
			}
		}
	}
	if f.PluginID == "rules" {
		return "Custom rule"
	}
	return f.PluginID
}
