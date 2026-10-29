package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func patchJSON(t *testing.T, url string, body, out any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("PATCH %s: status %d: %s", url, resp.StatusCode, buf.String())
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode %s: %v", url, err)
		}
	}
}

func TestFeatureRoutes(t *testing.T) {
	srv := newTestAPI(t)
	defer srv.Close()

	var project domain.Project
	postJSON(t, srv.URL+"/api/v1/projects", map[string]any{"name": "Feature Co"}, &project)

	var schedule domain.Schedule
	postJSON(t, srv.URL+"/api/v1/schedules", map[string]any{
		"projectId": project.ID, "name": "Nightly", "url": "https://example.com/",
		"cron": "0 6 * * *", "timezone": "UTC",
	}, &schedule)
	if schedule.ID.IsEmpty() || schedule.NextRunAt == nil {
		t.Fatalf("unexpected schedule: %+v", schedule)
	}

	var listed struct {
		Schedules []domain.Schedule `json:"schedules"`
	}
	getJSON(t, srv.URL+"/api/v1/schedules?projectId="+project.ID.String(), &listed)
	if len(listed.Schedules) != 1 {
		t.Fatalf("schedules = %d, want 1", len(listed.Schedules))
	}

	var channel domain.AlertChannel
	postJSON(t, srv.URL+"/api/v1/alert-channels", map[string]any{
		"projectId": project.ID, "name": "Ops", "kind": "webhook", "target": "https://example.com/hook",
	}, &channel)

	var rule domain.AlertRule
	postJSON(t, srv.URL+"/api/v1/alert-rules", map[string]any{
		"projectId": project.ID, "name": "Critical", "severity": "critical",
		"condition": map[string]any{"minSeverity": "high"}, "channelIds": []string{channel.ID.String()},
	}, &rule)
	if rule.ConditionText == "" {
		t.Fatal("expected condition text")
	}

	var disabled domain.AlertRule
	patchJSON(t, srv.URL+"/api/v1/alert-rules/"+rule.ID.String(), map[string]any{"enabled": false}, &disabled)
	if disabled.Enabled {
		t.Fatal("expected rule to be disabled")
	}
	if disabled.ConditionText != rule.ConditionText || len(disabled.ChannelIDs) != 1 {
		t.Fatalf("partial update clobbered rule: %+v", disabled)
	}

	var journey domain.Journey
	postJSON(t, srv.URL+"/api/v1/journeys", map[string]any{
		"projectId": project.ID, "name": "Checkout", "baseUrl": "https://example.com/",
		"steps": []map[string]any{
			{"order": 1, "action": "navigate", "value": "https://example.com/"},
			{"order": 2, "action": "click", "selector": "#buy"},
		},
	}, &journey)
	if len(journey.Definition) == 0 {
		t.Fatal("expected journey definition")
	}

	var profiles struct {
		Profiles []domain.PrivacyProfile `json:"profiles"`
	}
	getJSON(t, srv.URL+"/api/v1/privacy/profiles?projectId="+project.ID.String(), &profiles)
	if len(profiles.Profiles) != 3 {
		t.Fatalf("profiles = %d, want 3 seeded", len(profiles.Profiles))
	}

	var report domain.Report
	postJSON(t, srv.URL+"/api/v1/reports", map[string]any{
		"projectId": project.ID, "templateId": "tpl-governance", "format": "json",
	}, &report)
	if report.ID.IsEmpty() || report.SizeBytes == 0 {
		t.Fatalf("unexpected report: %+v", report)
	}

	resp, err := http.Get(srv.URL + "/api/v1/reports/" + report.ID.String() + "/download")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("download status = %d", resp.StatusCode)
	}
	if disp := resp.Header.Get("Content-Disposition"); disp == "" {
		t.Fatal("missing content disposition")
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	if buf.Len() == 0 {
		t.Fatal("downloaded report is empty")
	}

	var reportSchedule domain.ReportSchedule
	postJSON(t, srv.URL+"/api/v1/report-schedules", map[string]any{
		"projectId": project.ID, "templateId": "tpl-privacy", "name": "Weekly",
		"cron": "0 6 * * 1", "timezone": "UTC", "format": "csv",
	}, &reportSchedule)
	if reportSchedule.NextRunAt == nil {
		t.Fatal("expected report schedule next run")
	}

	var fromSchedule domain.Report
	postJSON(t, srv.URL+"/api/v1/report-schedules/"+reportSchedule.ID.String()+"/run", nil, &fromSchedule)
	if fromSchedule.ID.IsEmpty() {
		t.Fatal("expected report from schedule")
	}
}
