package governance

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// Report bundles everything needed for an audit export.
type Report struct {
	GeneratedAt time.Time         `json:"generatedAt"`
	Project     *domain.Project   `json:"project,omitempty"`
	Run         *domain.Run       `json:"run"`
	Scan        *domain.Scan      `json:"scan"`
	Validation  *ValidationReport `json:"validation"`
	Consent     *ConsentMatrix    `json:"consent,omitempty"`
}

// BuildReport loads an audit and its derived validation/consent data.
func (s *Service) BuildReport(ctx context.Context, runID domain.ID) (*Report, error) {
	run, scan, _, results, err := s.runContext(ctx, runID)
	if err != nil {
		return nil, err
	}
	validation, err := s.Validation(ctx, runID)
	if err != nil {
		return nil, err
	}
	report := &Report{GeneratedAt: time.Now().UTC(), Run: run, Scan: scan, Validation: validation}
	if project, err := s.store.GetProject(ctx, run.ProjectID); err == nil {
		report.Project = project
	}
	if sr := latestResult(results); sr != nil {
		m := buildConsentMatrix(sr)
		report.Consent = m
	}
	return report, nil
}

// ReportBytes renders an audit report in json, csv or html format.
func (s *Service) ReportBytes(ctx context.Context, runID domain.ID, format string) ([]byte, string, error) {
	report, err := s.BuildReport(ctx, runID)
	if err != nil {
		return nil, "", err
	}
	switch format {
	case "", "json":
		raw, err := json.MarshalIndent(report, "", "  ")
		return raw, "application/json", err
	case "csv":
		raw, err := reportCSV(report)
		return raw, "text/csv", err
	case "html":
		return []byte(reportHTML(report)), "text/html", nil
	default:
		return nil, "", fmt.Errorf("unsupported report format %q", format)
	}
}

func reportCSV(report *Report) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"page_url", "rule", "vendor", "severity", "status", "reason"})
	for _, page := range report.Validation.Pages {
		for _, res := range page.Results {
			_ = w.Write([]string{page.URL, res.Name, res.Vendor, string(res.Severity), res.Status, res.Reason})
		}
	}
	if len(report.Validation.Pages) == 0 {
		_ = w.Write([]string{"", "", "", "", "", "no validation results"})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func reportHTML(report *Report) string {
	var b bytes.Buffer
	project := ""
	if report.Project != nil {
		project = report.Project.Name
	}
	scanName := ""
	if report.Scan != nil {
		scanName = report.Scan.Name
	}
	fmt.Fprintf(&b, `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Governance report</title>
<style>
body{font-family:Inter,Arial,sans-serif;color:#0f172a;margin:40px;line-height:1.45}
h1{font-size:26px;margin:0 0 4px}h2{font-size:17px;margin:28px 0 10px;border-bottom:1px solid #e2e8f0;padding-bottom:6px}
.muted{color:#64748b;font-size:12px}.kpis{display:flex;gap:12px;flex-wrap:wrap;margin-top:18px}
.kpi{border:1px solid #e2e8f0;border-radius:10px;padding:12px 16px;min-width:130px}
.kpi b{display:block;font-size:22px}.kpi span{font-size:11px;color:#64748b;text-transform:uppercase}
table{width:100%%;border-collapse:collapse;font-size:12px}th,td{border-bottom:1px solid #e2e8f0;padding:7px 8px;text-align:left;vertical-align:top}
th{background:#f8fafc;text-transform:uppercase;font-size:10px;color:#64748b}.fail{color:#b91c1c;font-weight:600}.pass{color:#047857}
@media print{body{margin:18mm}.no-print{display:none}}
</style></head><body>
<h1>Tag Governance Audit Report</h1>
<div class="muted">%s &middot; %s &middot; generated %s</div>`,
		html.EscapeString(project), html.EscapeString(scanName), report.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, `<div class="kpis">
<div class="kpi"><b>%.1f%%</b><span>Health score</span></div>
<div class="kpi"><b>%d</b><span>Pages scanned</span></div>
<div class="kpi"><b>%d</b><span>Beacons</span></div>
<div class="kpi"><b>%d</b><span>Findings</span></div>
<div class="kpi"><b>%.1f%%</b><span>CMP compliance</span></div>
</div>`, report.Validation.HealthScore, report.Run.Stats.PagesScanned,
		report.Validation.Beacons, len(report.Validation.Findings), report.Validation.ConsentScore)
	fmt.Fprint(&b, `<h2>Rule results</h2><table><thead><tr><th>Page</th><th>Rule</th><th>Vendor</th><th>Severity</th><th>Status</th><th>Detail</th></tr></thead><tbody>`)
	for _, page := range report.Validation.Pages {
		for _, res := range page.Results {
			class := "pass"
			if res.Status == "fail" {
				class = "fail"
			}
			fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td class="%s">%s</td><td>%s</td></tr>`,
				html.EscapeString(page.URL), html.EscapeString(res.Name), html.EscapeString(res.Vendor),
				html.EscapeString(string(res.Severity)), class, html.EscapeString(res.Status), html.EscapeString(res.Reason))
		}
	}
	fmt.Fprint(&b, `</tbody></table>`)
	if report.Consent != nil {
		fmt.Fprintf(&b, `<h2>Consent matrix</h2><div class="muted">CMP: %s &middot; pre-consent score %.1f%% &middot; %d leaks</div>`,
			html.EscapeString(report.Consent.CMPVendor), report.Consent.PreConsentScore, report.Consent.Leaks)
		fmt.Fprint(&b, `<table><thead><tr><th>Category</th><th>Tracker</th><th>Vendor</th><th>Pre-consent</th><th>Post-consent</th><th>Status</th></tr></thead><tbody>`)
		for _, e := range report.Consent.Entries {
			fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td class="%s">%s</td></tr>`,
				html.EscapeString(e.Category), html.EscapeString(e.Name), html.EscapeString(e.Vendor),
				yesNo(e.PreConsent), yesNo(e.PostConsent), statusClass(e.Status), html.EscapeString(e.Status))
		}
		fmt.Fprint(&b, `</tbody></table>`)
	}
	fmt.Fprintf(&b, `<h2>Run summary</h2><table><tbody>
<tr><th>Run ID</th><td>%s</td></tr><tr><th>Status</th><td>%s</td></tr>
<tr><th>Started</th><td>%s</td></tr><tr><th>Duration</th><td>%s</td></tr>
<tr><th>Pages failed</th><td>%d</td></tr><tr><th>Critical findings</th><td>%d</td></tr>
</tbody></table>`, html.EscapeString(string(report.Run.ID)), html.EscapeString(string(report.Run.Status)),
		startedAt(report.Run), duration(report.Run), report.Run.Stats.PagesFailed, report.Run.Stats.CriticalFindings)
	fmt.Fprint(&b, `<p class="muted" style="margin-top:30px">Generated by Pageguard OpenTagValidator. Consent results are derived from the captured pre-consent snapshot.</p></body></html>`)
	return b.String()
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func statusClass(s string) string {
	switch s {
	case "leak", "fail":
		return "fail"
	case "allowed", "pass":
		return "pass"
	}
	return ""
}

func startedAt(r *domain.Run) string {
	if r.StartedAt == nil {
		return ""
	}
	return r.StartedAt.Format(time.RFC3339)
}

func duration(r *domain.Run) string {
	if r.StartedAt == nil {
		return ""
	}
	end := time.Now()
	if r.FinishedAt != nil {
		end = *r.FinishedAt
	}
	secs := int64(end.Sub(*r.StartedAt).Seconds())
	return strconv.FormatInt(secs, 10) + "s"
}
