package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/cron"
	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/privacy"
)

// ReportTemplate describes a report the application can generate.
type ReportTemplate struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Formats     []string `json:"formats"`
}

var reportTemplates = []ReportTemplate{
	{ID: "tpl-governance", Name: "Governance audit", Description: "Validation, consent and policy findings for a run.", Formats: []string{"json", "csv", "html"}},
	{ID: "tpl-consent", Name: "Consent matrix", Description: "Pre/post consent tracker behaviour per page.", Formats: []string{"json", "csv", "html"}},
	{ID: "tpl-inventory", Name: "Tag and cookie inventory", Description: "Aggregated tags, cookies and requests for the latest run.", Formats: []string{"json", "csv", "html"}},
	{ID: "tpl-journey", Name: "Journey results", Description: "Step-by-step journey outcomes and failures.", Formats: []string{"json", "csv", "html"}},
	{ID: "tpl-privacy", Name: "Privacy regulations", Description: "Per-regulation privacy scores and violations.", Formats: []string{"json", "csv", "html"}},
}

// ReportTemplates lists the supported templates.
func ReportTemplates() []ReportTemplate { return reportTemplates }

// ReportInput is the payload used to generate a report.
type ReportInput struct {
	ProjectID  domain.ID          `json:"projectId"`
	TemplateID string             `json:"templateId"`
	Name       string             `json:"name"`
	Format     string             `json:"format"`
	Scope      domain.ReportScope `json:"scope"`
	Owner      string             `json:"owner"`
}

// ReportScheduleInput is the payload used to create or update a report schedule.
type ReportScheduleInput struct {
	ProjectID  domain.ID          `json:"projectId"`
	TemplateID string             `json:"templateId"`
	Name       string             `json:"name"`
	Cadence    string             `json:"cadence"`
	Cron       string             `json:"cron"`
	Timezone   string             `json:"timezone"`
	Recipients []string           `json:"recipients"`
	Scope      domain.ReportScope `json:"scope"`
	Format     string             `json:"format"`
	Enabled    *bool              `json:"enabled"`
}

// Reports lists generated reports for a project.
func (s *Service) Reports(ctx context.Context, projectID domain.ID, limit int) ([]*domain.Report, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.store.ListReports(ctx, project.ID, limit)
}

// Report returns one report including its content.
func (s *Service) Report(ctx context.Context, id domain.ID) (*domain.Report, error) {
	return s.store.GetReport(ctx, id)
}

// DeleteReport removes a generated report.
func (s *Service) DeleteReport(ctx context.Context, id domain.ID) error {
	return s.store.DeleteReport(ctx, id)
}

// GenerateReport builds and stores a report synchronously.
func (s *Service) GenerateReport(ctx context.Context, in ReportInput) (*domain.Report, error) {
	project, err := s.resolveProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	templateID, templateName, err := normalizeReportTemplate(in.TemplateID)
	if err != nil {
		return nil, err
	}
	format, err := normalizeReportFormat(in.Format)
	if err != nil {
		return nil, err
	}
	scope, err := s.normalizeReportScope(ctx, project.ID, in.Scope)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = fmt.Sprintf("%s %s", templateName, time.Now().UTC().Format("2006-01-02"))
	}

	content, contentType, err := s.buildReportContent(ctx, project.ID, templateID, scope, format)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	r := &domain.Report{
		ID: domain.NewID(), ProjectID: project.ID, TemplateID: templateID, Name: name,
		Scope: scope, Format: format, Status: "ready", Owner: in.Owner,
		ContentType: contentType, Content: content, SizeBytes: int64(len(content)),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.CreateReport(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// ReportSchedules lists report schedules for a project.
func (s *Service) ReportSchedules(ctx context.Context, projectID domain.ID) ([]*domain.ReportSchedule, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.store.ListReportSchedules(ctx, project.ID)
}

// CreateReportSchedule validates and stores a report schedule.
func (s *Service) CreateReportSchedule(ctx context.Context, in ReportScheduleInput) (*domain.ReportSchedule, error) {
	project, err := s.resolveProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	templateID, templateName, err := normalizeReportTemplate(in.TemplateID)
	if err != nil {
		return nil, err
	}
	format, err := normalizeReportFormat(in.Format)
	if err != nil {
		return nil, err
	}
	scope, err := s.normalizeReportScope(ctx, project.ID, in.Scope)
	if err != nil {
		return nil, err
	}
	loc, err := scheduleLocation(in.Timezone)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	next, err := cron.Next(in.Cron, now, loc)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = templateName
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	sched := &domain.ReportSchedule{
		ID: domain.NewID(), ProjectID: project.ID, TemplateID: templateID, Name: name,
		Cadence: in.Cadence, Cron: strings.TrimSpace(in.Cron), Timezone: timezoneName(loc),
		Recipients: in.Recipients, Scope: scope, Format: format, Enabled: enabled,
		NextRunAt: &next, CreatedAt: now, UpdatedAt: now,
	}
	if !enabled {
		sched.NextRunAt = nil
	}
	if err := s.store.CreateReportSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

// UpdateReportSchedule applies a partial update.
func (s *Service) UpdateReportSchedule(ctx context.Context, id domain.ID, in ReportScheduleInput) (*domain.ReportSchedule, error) {
	sched, err := s.store.GetReportSchedule(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.TemplateID != "" {
		templateID, _, terr := normalizeReportTemplate(in.TemplateID)
		if terr != nil {
			return nil, terr
		}
		sched.TemplateID = templateID
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		sched.Name = name
	}
	if in.Cadence != "" {
		sched.Cadence = in.Cadence
	}
	if in.Cron != "" {
		sched.Cron = strings.TrimSpace(in.Cron)
	}
	if in.Timezone != "" {
		sched.Timezone = in.Timezone
	}
	if in.Recipients != nil {
		sched.Recipients = in.Recipients
	}
	if in.Format != "" {
		format, ferr := normalizeReportFormat(in.Format)
		if ferr != nil {
			return nil, ferr
		}
		sched.Format = format
	}
	if in.Scope.Type != "" || in.Scope.ID != "" {
		scope, serr := s.normalizeReportScope(ctx, sched.ProjectID, in.Scope)
		if serr != nil {
			return nil, serr
		}
		sched.Scope = scope
	}
	if in.Enabled != nil {
		sched.Enabled = *in.Enabled
	}
	loc, err := scheduleLocation(sched.Timezone)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	next, err := cron.Next(sched.Cron, now, loc)
	if err != nil {
		return nil, err
	}
	sched.NextRunAt = &next
	if !sched.Enabled {
		sched.NextRunAt = nil
	}
	sched.UpdatedAt = now
	if err := s.store.UpdateReportSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

// DeleteReportSchedule removes a report schedule.
func (s *Service) DeleteReportSchedule(ctx context.Context, id domain.ID) error {
	return s.store.DeleteReportSchedule(ctx, id)
}

// RunReportScheduleNow generates a report for a schedule immediately.
func (s *Service) RunReportScheduleNow(ctx context.Context, id domain.ID) (*domain.Report, error) {
	sched, err := s.store.GetReportSchedule(ctx, id)
	if err != nil {
		return nil, err
	}
	report, err := s.GenerateReport(ctx, ReportInput{
		ProjectID: sched.ProjectID, TemplateID: sched.TemplateID, Name: sched.Name,
		Format: sched.Format, Scope: sched.Scope,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	sched.LastRunAt = &now
	sched.UpdatedAt = now
	if loc, lerr := scheduleLocation(sched.Timezone); lerr == nil {
		if next, nerr := cron.Next(sched.Cron, now, loc); nerr == nil {
			sched.NextRunAt = &next
		}
	}
	_ = s.store.UpdateReportSchedule(ctx, sched)
	return report, nil
}

// ProcessReportSchedules generates every due report schedule.
func (s *Service) ProcessReportSchedules(ctx context.Context, now time.Time) {
	due, err := s.store.ListDueReportSchedules(ctx, now, 10)
	if err != nil {
		return
	}
	for _, sched := range due {
		if sched == nil || !sched.Enabled {
			continue
		}
		_, _ = s.RunReportScheduleNow(ctx, sched.ID)
	}
}

func (s *Service) buildReportContent(ctx context.Context, projectID domain.ID, templateID string, scope domain.ReportScope, format string) ([]byte, string, error) {
	run, _ := s.reportRun(ctx, projectID, scope)
	generated := time.Now().UTC()

	switch templateID {
	case "tpl-governance":
		if run == nil {
			return renderReport(format, "Governance audit", map[string]any{
				"generatedAt": generated, "message": "No scan runs are available yet.",
			}, []string{"status"}, nil)
		}
		raw, ct, err := s.gov.ReportBytes(ctx, run.ID, format)
		if err != nil {
			return nil, "", err
		}
		return raw, ct, nil

	case "tpl-consent":
		data := map[string]any{"generatedAt": generated, "runId": "", "consent": nil}
		var header []string
		var rows [][]string
		if run != nil {
			matrix, err := s.gov.Consent(ctx, run.ID, "")
			if err != nil && err != domain.ErrNotFound {
				return nil, "", err
			}
			data["runId"] = run.ID
			data["consent"] = matrix
			header = []string{"page_url", "key", "name", "vendor", "type", "category", "status", "essential", "leak"}
			if matrix != nil {
				for _, e := range matrix.Entries {
					rows = append(rows, []string{
						matrix.PageURL, e.Key, e.Name, e.Vendor, e.Type, e.Category, e.Status,
						fmt.Sprint(e.Essential), fmt.Sprint(e.Leak),
					})
				}
			}
		}
		return renderReport(format, "Consent matrix", data, header, rows)

	case "tpl-inventory":
		snap, err := s.Inventory(ctx, projectID, runIDOf(run))
		if err != nil {
			return nil, "", err
		}
		header := []string{"type", "name", "vendor", "category", "purpose", "pages", "domain_or_url"}
		var rows [][]string
		for _, t := range snap.Tags {
			rows = append(rows, []string{"tag", t.Name, t.Vendor, string(t.Category), t.Purpose, fmt.Sprint(t.Pages), ""})
		}
		for _, c := range snap.Cookies {
			rows = append(rows, []string{"cookie", c.Name, c.Vendor, "", c.Purpose, fmt.Sprint(c.Pages), c.Domain})
		}
		for _, r := range snap.Requests {
			rows = append(rows, []string{"request", r.Method + " " + r.URL, r.Vendor, "", "", fmt.Sprint(r.Pages), r.URL})
		}
		return renderReport(format, "Tag and cookie inventory", snap, header, rows)

	case "tpl-journey":
		journeys, err := s.store.ListJourneys(ctx, projectID)
		if err != nil {
			return nil, "", err
		}
		var runs []*domain.JourneyRun
		var rows [][]string
		for _, j := range journeys {
			jruns, _ := s.store.ListJourneyRuns(ctx, j.ID, 10)
			for _, run := range jruns {
				runs = append(runs, run)
				rows = append(rows, []string{
					j.Name, run.ID.String(), run.Status, fmt.Sprint(run.StepsPassed),
					fmt.Sprint(run.StepsTotal), fmt.Sprint(run.DurationMs),
					run.StartedAt.Format(time.RFC3339), run.Error,
				})
			}
		}
		data := map[string]any{"generatedAt": generated, "journeys": journeys, "runs": runs}
		return renderReport(format, "Journey results", data,
			[]string{"journey", "run_id", "status", "steps_passed", "steps_total", "duration_ms", "started_at", "error"}, rows)

	case "tpl-privacy":
		batches, err := s.store.ListPrivacyBatches(ctx, projectID, 10)
		if err != nil {
			return nil, "", err
		}
		type evalView struct {
			Evaluation *domain.PrivacyEvaluation `json:"evaluation"`
			Report     json.RawMessage           `json:"report,omitempty"`
		}
		var views []evalView
		var rows [][]string
		for _, batch := range batches {
			evals, _ := s.store.ListPrivacyEvaluations(ctx, batch.ID, 100)
			for _, eval := range evals {
				views = append(views, evalView{Evaluation: eval, Report: eval.Report})
				if len(eval.Report) == 0 {
					continue
				}
				var rep privacy.Report
				if err := json.Unmarshal(eval.Report, &rep); err != nil {
					continue
				}
				for _, reg := range rep.Regulations {
					failures := 0
					for _, check := range reg.Checks {
						if check.Status == privacy.CheckFail {
							failures++
						}
					}
					rows = append(rows, []string{
						batch.ID.String(), eval.URL, string(reg.Regulation), fmt.Sprint(reg.Score),
						fmt.Sprint(reg.Passed), fmt.Sprint(len(reg.Checks)), fmt.Sprint(failures),
					})
				}
			}
		}
		data := map[string]any{"generatedAt": generated, "batches": batches, "evaluations": views}
		return renderReport(format, "Privacy regulations", data,
			[]string{"batch_id", "url", "regulation", "score", "passed", "checks", "failures"}, rows)

	default:
		return nil, "", fmt.Errorf("unknown report template %q", templateID)
	}
}

func (s *Service) reportRun(ctx context.Context, projectID domain.ID, scope domain.ReportScope) (*domain.Run, error) {
	if scope.Type == "run" && scope.ID != "" {
		run, err := s.store.GetRun(ctx, domain.ID(scope.ID))
		if err != nil {
			return nil, err
		}
		if run.ProjectID != projectID {
			return nil, domain.ErrNotFound
		}
		return run, nil
	}
	return s.latestRun(ctx, projectID)
}

func runIDOf(run *domain.Run) domain.ID {
	if run == nil {
		return ""
	}
	return run.ID
}

func (s *Service) normalizeReportScope(ctx context.Context, projectID domain.ID, scope domain.ReportScope) (domain.ReportScope, error) {
	if scope.Type == "" {
		scope.Type = "project"
	}
	switch scope.Type {
	case "project":
		scope.ID = projectID.String()
	case "run":
		if scope.ID == "" {
			return scope, fmt.Errorf("run scope requires an id")
		}
		run, err := s.store.GetRun(ctx, domain.ID(scope.ID))
		if err != nil {
			return scope, err
		}
		if run.ProjectID != projectID {
			return scope, domain.ErrNotFound
		}
	default:
		return scope, fmt.Errorf("scope type must be project or run")
	}
	return scope, nil
}

func normalizeReportTemplate(id string) (string, string, error) {
	if id == "" {
		id = "tpl-governance"
	}
	for _, t := range reportTemplates {
		if t.ID == id {
			return t.ID, t.Name, nil
		}
	}
	return "", "", fmt.Errorf("unknown report template %q", id)
}

func normalizeReportFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return "json", nil
	case "csv":
		return "csv", nil
	case "html":
		return "html", nil
	}
	return "", fmt.Errorf("format must be json, csv or html")
}

func renderReport(format, title string, data any, header []string, rows [][]string) ([]byte, string, error) {
	switch format {
	case "json":
		raw, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, "", err
		}
		return append(raw, '\n'), "application/json", nil
	case "csv":
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		if len(header) > 0 {
			_ = w.Write(header)
		}
		for _, row := range rows {
			_ = w.Write(row)
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "text/csv", nil
	case "html":
		raw, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, "", err
		}
		var b strings.Builder
		b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>")
		b.WriteString(html.EscapeString(title))
		b.WriteString("</title></head><body><h1>")
		b.WriteString(html.EscapeString(title))
		b.WriteString("</h1><p>Generated by Pageguard at ")
		b.WriteString(time.Now().UTC().Format(time.RFC3339))
		b.WriteString("</p><pre>")
		b.WriteString(html.EscapeString(string(raw)))
		b.WriteString("</pre></body></html>")
		return []byte(b.String()), "text/html", nil
	}
	return nil, "", fmt.Errorf("unsupported format %q", format)
}
