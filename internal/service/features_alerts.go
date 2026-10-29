package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// AlertRuleInput is the payload used to create or update an alert rule.
type AlertRuleInput struct {
	ProjectID     domain.ID             `json:"projectId"`
	Name          string                `json:"name"`
	Enabled       *bool                 `json:"enabled"`
	Severity      domain.Severity       `json:"severity"`
	Condition     domain.AlertCondition `json:"condition"`
	ConditionText string                `json:"conditionText"`
	Channels      []string              `json:"channels"`
	ChannelIDs    []domain.ID           `json:"channelIds"`
}

// AlertChannelInput is the payload used to create or update a channel.
type AlertChannelInput struct {
	ProjectID domain.ID `json:"projectId"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Target    string    `json:"target"`
	SecretRef string    `json:"secretRef"`
}

// IncidentPatch updates the triage state of an incident.
type IncidentPatch struct {
	Status string `json:"status"`
	By     string `json:"by"`
}

// AlertRules lists a project's alert rules.
func (s *Service) AlertRules(ctx context.Context, projectID domain.ID) ([]*domain.AlertRule, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.store.ListAlertRules(ctx, project.ID)
}

// CreateAlertRule validates and creates an alert rule.
func (s *Service) CreateAlertRule(ctx context.Context, in AlertRuleInput) (*domain.AlertRule, error) {
	project, err := s.resolveProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	sev := in.Severity
	if sev == "" {
		sev = domain.SevMedium
	}
	now := time.Now().UTC()
	rule := &domain.AlertRule{
		ID:            domain.NewID(),
		ProjectID:     project.ID,
		Name:          name,
		Enabled:       enabled,
		Severity:      sev,
		Condition:     in.Condition,
		ConditionText: strings.TrimSpace(in.ConditionText),
		Channels:      in.Channels,
		ChannelIDs:    in.ChannelIDs,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if rule.ConditionText == "" {
		rule.ConditionText = describeCondition(rule.Condition)
	}
	if err := s.store.CreateAlertRule(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// UpdateAlertRule replaces the editable fields of a rule.
func (s *Service) UpdateAlertRule(ctx context.Context, id domain.ID, in AlertRuleInput) (*domain.AlertRule, error) {
	rule, err := s.store.GetAlertRule(ctx, id)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		rule.Name = name
	}
	if in.Enabled != nil {
		rule.Enabled = *in.Enabled
	}
	if in.Severity != "" {
		rule.Severity = in.Severity
	}
	if in.Condition.MinSeverity != "" || len(in.Condition.RuleIDs) > 0 || in.Condition.OnRunFailure || len(in.Condition.TagMissing) > 0 {
		rule.Condition = in.Condition
	}
	if in.ConditionText != "" {
		rule.ConditionText = in.ConditionText
	} else {
		rule.ConditionText = describeCondition(rule.Condition)
	}
	if in.Channels != nil {
		rule.Channels = in.Channels
	}
	if in.ChannelIDs != nil {
		rule.ChannelIDs = in.ChannelIDs
	}
	rule.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateAlertRule(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// DeleteAlertRule removes a rule.
func (s *Service) DeleteAlertRule(ctx context.Context, id domain.ID) error {
	return s.store.DeleteAlertRule(ctx, id)
}

// AlertChannels lists a project's channels.
func (s *Service) AlertChannels(ctx context.Context, projectID domain.ID) ([]*domain.AlertChannel, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.store.ListAlertChannels(ctx, project.ID)
}

// CreateAlertChannel creates a notification destination.
func (s *Service) CreateAlertChannel(ctx context.Context, in AlertChannelInput) (*domain.AlertChannel, error) {
	project, err := s.resolveProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	kind := strings.ToLower(strings.TrimSpace(in.Kind))
	switch kind {
	case "slack", "webhook", "email", "pagerduty":
	default:
		return nil, fmt.Errorf("kind must be one of slack, webhook, email or pagerduty")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	target := strings.TrimSpace(in.Target)
	secret := strings.TrimSpace(in.SecretRef)
	if target == "" && secret == "" {
		return nil, fmt.Errorf("target is required")
	}
	now := time.Now().UTC()
	ch := &domain.AlertChannel{
		ID: domain.NewID(), ProjectID: project.ID, Name: name, Kind: kind,
		Target: target, SecretRef: secret, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.CreateAlertChannel(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// UpdateAlertChannel replaces the editable fields of a channel.
func (s *Service) UpdateAlertChannel(ctx context.Context, id domain.ID, in AlertChannelInput) (*domain.AlertChannel, error) {
	ch, err := s.store.GetAlertChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		ch.Name = name
	}
	if kind := strings.ToLower(strings.TrimSpace(in.Kind)); kind != "" {
		switch kind {
		case "slack", "webhook", "email", "pagerduty":
			ch.Kind = kind
		default:
			return nil, fmt.Errorf("kind must be one of slack, webhook, email or pagerduty")
		}
	}
	if in.Target != "" {
		ch.Target = strings.TrimSpace(in.Target)
	}
	if in.SecretRef != "" {
		ch.SecretRef = strings.TrimSpace(in.SecretRef)
	}
	ch.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateAlertChannel(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// DeleteAlertChannel removes a channel.
func (s *Service) DeleteAlertChannel(ctx context.Context, id domain.ID) error {
	return s.store.DeleteAlertChannel(ctx, id)
}

// AlertIncidents lists a project's incidents, newest first.
func (s *Service) AlertIncidents(ctx context.Context, projectID domain.ID, limit int) ([]*domain.AlertIncident, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.store.ListAlertIncidents(ctx, project.ID, limit)
}

// AlertDeliveries lists delivery attempts for an incident.
func (s *Service) AlertDeliveries(ctx context.Context, incidentID domain.ID) ([]*domain.AlertDelivery, error) {
	return s.store.ListAlertDeliveries(ctx, incidentID)
}

// UpdateIncident applies a triage state change.
func (s *Service) UpdateIncident(ctx context.Context, id domain.ID, patch IncidentPatch) (*domain.AlertIncident, error) {
	inc, err := s.store.GetAlertIncident(ctx, id)
	if err != nil {
		return nil, err
	}
	status := domain.IncidentStatus(strings.ToLower(strings.TrimSpace(patch.Status)))
	switch status {
	case domain.IncidentOpen, domain.IncidentAcknowledged, domain.IncidentResolved:
	default:
		return nil, fmt.Errorf("status must be open, acknowledged or resolved")
	}
	now := time.Now().UTC()
	inc.Status = status
	switch status {
	case domain.IncidentAcknowledged:
		inc.AcknowledgedAt = &now
		if patch.By != "" {
			inc.AcknowledgedBy = patch.By
		}
	case domain.IncidentResolved:
		inc.ResolvedAt = &now
		if inc.AcknowledgedAt == nil {
			inc.AcknowledgedAt = &now
		}
	}
	inc.UpdatedAt = now
	if err := s.store.UpdateAlertIncident(ctx, inc); err != nil {
		return nil, err
	}
	return inc, nil
}

// TestAlertChannel sends a test notification and records connectivity.
func (s *Service) TestAlertChannel(ctx context.Context, id domain.ID) (*domain.AlertDelivery, error) {
	ch, err := s.store.GetAlertChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	inc := &domain.AlertIncident{
		ID: domain.NewID(), ProjectID: ch.ProjectID, Severity: domain.SevInfo,
		Title: "Pageguard test notification", Message: "This is a test alert from Pageguard.",
		RuleName: "test", Status: domain.IncidentOpen, CreatedAt: now, UpdatedAt: now,
	}
	delivery := &domain.AlertDelivery{
		ID: domain.NewID(), IncidentID: inc.ID, ChannelID: ch.ID, Attempt: 1, AttemptedAt: now,
	}
	if err := s.sendAlert(ctx, ch, inc); err != nil {
		delivery.Status = "failed"
		delivery.Error = err.Error()
		ch.Connected = false
	} else {
		delivery.Status = "delivered"
		delivery.DeliveredAt = &now
		ch.Connected = true
		ch.LastDeliveryAt = &now
	}
	ch.UpdatedAt = now
	_ = s.store.UpdateAlertChannel(ctx, ch)
	if delivery.Status == "failed" {
		return delivery, fmt.Errorf("%s", delivery.Error)
	}
	return delivery, nil
}

// EvaluateAlerts matches a finished run against the project's enabled rules,
// creates incidents and delivers notifications. It is safe to call for any run
// status.
func (s *Service) EvaluateAlerts(ctx context.Context, runID domain.ID) {
	run, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return
	}
	s.updateScheduleRunStatus(ctx, run)

	rules, err := s.store.ListAlertRules(ctx, run.ProjectID)
	if err != nil {
		return
	}
	findings, err := s.store.ListFindings(ctx, domain.FindingFilter{RunID: run.ID, Limit: 2000})
	if err != nil {
		return
	}
	var results []*domain.ScanResult
	if results, err = s.store.ListScanResults(ctx, run.ID, 2000); err != nil {
		return
	}
	channels, _ := s.store.ListAlertChannels(ctx, run.ProjectID)
	byID := make(map[domain.ID]*domain.AlertChannel, len(channels))
	for _, ch := range channels {
		byID[ch.ID] = ch
	}

	for _, rule := range rules {
		if rule == nil || !rule.Enabled {
			continue
		}
		matched, severity, reason := matchAlertRule(rule, run, findings, results)
		if !matched {
			continue
		}
		now := time.Now().UTC()
		var names []string
		var targets []*domain.AlertChannel
		for _, cid := range rule.ChannelIDs {
			if ch, ok := byID[cid]; ok {
				targets = append(targets, ch)
				names = append(names, ch.Name)
			}
		}
		inc := &domain.AlertIncident{
			ID: domain.NewID(), ProjectID: run.ProjectID, RuleID: rule.ID, RunID: run.ID,
			Severity: severity, Title: rule.Name,
			Message: fmt.Sprintf("%s (%s)", reason, run.ID), RuleName: rule.Name,
			Channel: strings.Join(names, ", "), Status: domain.IncidentOpen,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.store.CreateAlertIncident(ctx, inc); err != nil {
			continue
		}
		rule.LastTriggeredAt = &now
		rule.TriggerCount++
		rule.UpdatedAt = now
		_ = s.store.UpdateAlertRule(ctx, rule)
		s.deliverIncident(ctx, inc, targets)
	}
}

func (s *Service) deliverIncident(ctx context.Context, inc *domain.AlertIncident, channels []*domain.AlertChannel) {
	for _, ch := range channels {
		now := time.Now().UTC()
		d := &domain.AlertDelivery{
			ID: domain.NewID(), IncidentID: inc.ID, ChannelID: ch.ID,
			Attempt: 1, AttemptedAt: now,
		}
		if err := s.sendAlert(ctx, ch, inc); err != nil {
			d.Status = "failed"
			d.Error = err.Error()
			ch.Connected = false
		} else {
			d.Status = "delivered"
			d.DeliveredAt = &now
			ch.Connected = true
			ch.LastDeliveryAt = &now
		}
		ch.UpdatedAt = now
		_ = s.store.CreateAlertDelivery(ctx, d)
		_ = s.store.UpdateAlertChannel(ctx, ch)
	}
}

func (s *Service) sendAlert(ctx context.Context, ch *domain.AlertChannel, inc *domain.AlertIncident) error {
	endpoint := strings.TrimSpace(ch.Target)
	secret := ""
	if ch.SecretRef != "" {
		secret = strings.TrimSpace(os.Getenv(ch.SecretRef))
	}
	if endpoint == "" && secret != "" && (ch.Kind == "webhook" || ch.Kind == "slack" || strings.HasPrefix(secret, "http")) {
		endpoint = secret
	}
	if endpoint == "" && ch.Kind == "pagerduty" {
		endpoint = "https://events.pagerduty.com/v2/enqueue"
	}
	if endpoint == "" {
		return fmt.Errorf("%s channel %q has no http target", ch.Kind, ch.Name)
	}
	if ch.Kind == "email" && !strings.HasPrefix(endpoint, "http") {
		return fmt.Errorf("email delivery is not configured; use an http webhook target")
	}

	var payload any
	switch ch.Kind {
	case "slack":
		payload = map[string]any{"text": fmt.Sprintf("[Pageguard] %s: %s", inc.Title, inc.Message)}
	case "pagerduty":
		payload = map[string]any{
			"routing_key":  secret,
			"event_action": "trigger",
			"payload": map[string]any{
				"summary":  inc.Title + ": " + inc.Message,
				"source":   "pageguard",
				"severity": pagerDutySeverity(inc.Severity),
				"custom_details": map[string]any{
					"incidentId": inc.ID, "runId": inc.RunID, "rule": inc.RuleName,
				},
			},
		}
	default:
		payload = map[string]any{
			"event": "pageguard.alert", "id": inc.ID, "title": inc.Title,
			"message": inc.Message, "severity": inc.Severity, "runId": inc.RunID,
			"ruleId": inc.RuleID, "projectId": inc.ProjectID, "createdAt": inc.CreatedAt,
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return nil
}

func pagerDutySeverity(s domain.Severity) string {
	switch s {
	case domain.SevCritical:
		return "critical"
	case domain.SevHigh:
		return "error"
	case domain.SevMedium:
		return "warning"
	default:
		return "info"
	}
}

func matchAlertRule(rule *domain.AlertRule, run *domain.Run, findings []*domain.Finding, results []*domain.ScanResult) (bool, domain.Severity, string) {
	sev := rule.Severity
	if sev == "" {
		sev = domain.SevMedium
	}
	var reasons []string
	matched := false
	highest := sev

	if rule.Condition.OnRunFailure && run.Status == domain.RunFailed {
		matched = true
		reasons = append(reasons, "run failed")
	}
	if min := rule.Condition.MinSeverity; min != "" {
		count := 0
		for _, f := range findings {
			if f == nil {
				continue
			}
			if severityRank(f.Severity) >= severityRank(min) {
				count++
				if severityRank(f.Severity) > severityRank(highest) {
					highest = f.Severity
				}
			}
		}
		if count > 0 {
			matched = true
			reasons = append(reasons, fmt.Sprintf("%d findings at or above %s", count, min))
		}
	}
	if len(rule.Condition.RuleIDs) > 0 {
		want := map[string]bool{}
		for _, id := range rule.Condition.RuleIDs {
			want[strings.ToLower(id)] = true
		}
		count := 0
		for _, f := range findings {
			if f != nil && want[strings.ToLower(f.RuleID)] {
				count++
				if severityRank(f.Severity) > severityRank(highest) {
					highest = f.Severity
				}
			}
		}
		if count > 0 {
			matched = true
			reasons = append(reasons, fmt.Sprintf("%d findings matched rule filters", count))
		}
	}
	if len(rule.Condition.TagMissing) > 0 {
		missing := map[string]int{}
		for _, res := range results {
			if res == nil {
				continue
			}
			present := map[string]bool{}
			for _, t := range res.Tags {
				present[strings.ToLower(t.ID)] = true
			}
			for _, want := range rule.Condition.TagMissing {
				if !present[strings.ToLower(want)] {
					missing[want]++
				}
			}
		}
		if len(missing) > 0 {
			matched = true
			var parts []string
			for tag, pages := range missing {
				parts = append(parts, fmt.Sprintf("%s missing on %d pages", tag, pages))
			}
			reasons = append(reasons, strings.Join(parts, ", "))
		}
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "rule matched")
	}
	return matched, highest, strings.Join(reasons, "; ")
}

func severityRank(s domain.Severity) int {
	switch s {
	case domain.SevCritical:
		return 5
	case domain.SevHigh:
		return 4
	case domain.SevMedium:
		return 3
	case domain.SevLow:
		return 2
	case domain.SevInfo:
		return 1
	}
	return 0
}

func describeCondition(c domain.AlertCondition) string {
	var parts []string
	if c.OnRunFailure {
		parts = append(parts, "run fails")
	}
	if c.MinSeverity != "" {
		parts = append(parts, "findings >= "+string(c.MinSeverity))
	}
	if len(c.RuleIDs) > 0 {
		parts = append(parts, "rules: "+strings.Join(c.RuleIDs, ", "))
	}
	if len(c.TagMissing) > 0 {
		parts = append(parts, "missing tags: "+strings.Join(c.TagMissing, ", "))
	}
	if len(parts) == 0 {
		return "any completed run"
	}
	return strings.Join(parts, " and ")
}

func (s *Service) updateScheduleRunStatus(ctx context.Context, run *domain.Run) {
	schedules, err := s.store.ListSchedules(ctx, run.ProjectID)
	if err != nil {
		return
	}
	for _, sched := range schedules {
		if sched.LastRunID != run.ID {
			continue
		}
		now := time.Now().UTC()
		sched.LastStatus = run.Status
		sched.UpdatedAt = now
		_ = s.store.UpdateSchedule(ctx, sched)
	}
}
