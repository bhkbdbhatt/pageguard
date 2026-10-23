package service

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/governance"
)

// Governance returns the reporting service.
func (s *Service) Governance() *governance.Service { return s.gov }

// CreateAuditRequest is the dashboard's one-click audit input.
type CreateAuditRequest struct {
	ProjectID         domain.ID
	Name              string
	StartURL          string
	MaxPages          int
	MaxDepth          int
	RenderMode        domain.RenderMode
	ConsentSimulation string
	Geolocation       string
}

// CreateAudit creates a scan definition and immediately starts a run.
func (s *Service) CreateAudit(ctx context.Context, req CreateAuditRequest) (*domain.Scan, *domain.Run, error) {
	start := strings.TrimSpace(req.StartURL)
	u, err := url.Parse(start)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, nil, fmt.Errorf("startUrl must be an absolute http(s) URL")
	}
	projectID := req.ProjectID
	if projectID.IsEmpty() {
		projects, err := s.store.ListProjects(ctx)
		if err != nil {
			return nil, nil, err
		}
		if len(projects) == 0 {
			return nil, nil, fmt.Errorf("create a project before running an audit")
		}
		projectID = projects[0].ID
	}
	if _, err := s.store.GetProject(ctx, projectID); err != nil {
		return nil, nil, err
	}
	mode := req.RenderMode
	if mode == "" {
		mode = domain.RenderAuto
	}
	switch mode {
	case domain.RenderHTTP, domain.RenderBrowser, domain.RenderAuto:
	default:
		return nil, nil, fmt.Errorf("unsupported renderMode %q", mode)
	}
	consent := req.ConsentSimulation
	if consent == "" {
		consent = "pre_consent"
	}
	cfg := domain.ScanConfig{
		MaxPages:          req.MaxPages,
		MaxDepth:          req.MaxDepth,
		SameOriginOnly:    true,
		RenderMode:        mode,
		CaptureDataLayer:  true,
		CaptureHAR:        false,
		ConsentSimulation: consent,
	}
	if req.Geolocation != "" {
		cfg.Geolocations = []string{req.Geolocation}
	}
	cfg.Normalize()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("Audit %s %s", u.Hostname(), time.Now().UTC().Format("2006-01-02 15:04"))
	}
	scan, err := s.CreateScan(ctx, &domain.Scan{
		ProjectID: projectID,
		Name:      name,
		StartURLs: []string{start},
		Config:    cfg,
		Enabled:   true,
	})
	if err != nil {
		return nil, nil, err
	}
	run, err := s.StartRun(ctx, scan.ID)
	if err != nil {
		return nil, nil, err
	}
	return scan, run, nil
}

// CreateRule validates and stores an audit rule.
func (s *Service) CreateRule(ctx context.Context, r *domain.AuditRule) (*domain.AuditRule, error) {
	projectID := r.ProjectID
	if projectID.IsEmpty() {
		projects, err := s.store.ListProjects(ctx)
		if err != nil {
			return nil, err
		}
		if len(projects) == 0 {
			return nil, fmt.Errorf("create a project before adding rules")
		}
		projectID = projects[0].ID
		r.ProjectID = projectID
	}
	if _, err := s.store.GetProject(ctx, projectID); err != nil {
		return nil, err
	}
	if r.ID.IsEmpty() {
		r.ID = domain.NewID()
	}
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now
	if err := validateRule(r); err != nil {
		return nil, err
	}
	if err := s.store.CreateRule(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// RulePatch contains optional rule fields for partial updates.
type RulePatch struct {
	Name        *string              `json:"name"`
	Vendor      *string              `json:"vendor"`
	Enabled     *bool                `json:"enabled"`
	URLPattern  *string              `json:"urlPattern"`
	Target      *domain.RuleTarget   `json:"target"`
	Parameter   *string              `json:"parameter"`
	Event       *string              `json:"event"`
	Operator    *domain.RuleOperator `json:"operator"`
	Expected    *string              `json:"expected"`
	Severity    *domain.Severity     `json:"severity"`
	Description *string              `json:"description"`
}

// UpdateRule applies a partial update.
func (s *Service) UpdateRule(ctx context.Context, id domain.ID, patch RulePatch) (*domain.AuditRule, error) {
	r, err := s.store.GetRule(ctx, id)
	if err != nil {
		return nil, err
	}
	if patch.Name != nil {
		r.Name = *patch.Name
	}
	if patch.Vendor != nil {
		r.Vendor = *patch.Vendor
	}
	if patch.Enabled != nil {
		r.Enabled = *patch.Enabled
	}
	if patch.URLPattern != nil {
		r.URLPattern = *patch.URLPattern
	}
	if patch.Target != nil {
		r.Target = *patch.Target
	}
	if patch.Parameter != nil {
		r.Parameter = *patch.Parameter
	}
	if patch.Event != nil {
		r.Event = *patch.Event
	}
	if patch.Operator != nil {
		r.Operator = *patch.Operator
	}
	if patch.Expected != nil {
		r.Expected = *patch.Expected
	}
	if patch.Severity != nil {
		r.Severity = *patch.Severity
	}
	if patch.Description != nil {
		r.Description = *patch.Description
	}
	r.UpdatedAt = time.Now().UTC()
	if err := validateRule(r); err != nil {
		return nil, err
	}
	if err := s.store.UpdateRule(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// DeleteRule removes a rule.
func (s *Service) DeleteRule(ctx context.Context, id domain.ID) error {
	return s.store.DeleteRule(ctx, id)
}

// StarterRules installs a small baseline rule set for a project.
func (s *Service) StarterRules(ctx context.Context, projectID domain.ID) ([]*domain.AuditRule, error) {
	if projectID.IsEmpty() {
		projects, err := s.store.ListProjects(ctx)
		if err != nil {
			return nil, err
		}
		if len(projects) == 0 {
			return nil, fmt.Errorf("create a project before adding rules")
		}
		projectID = projects[0].ID
	}
	existing, err := s.store.ListRules(ctx, projectID)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(existing))
	for _, r := range existing {
		seen[r.Name] = true
	}
	starters := []*domain.AuditRule{
		{Name: "OneTrust CMP present", Vendor: "OneTrust", Target: domain.RuleTargetTag, Operator: domain.RuleOpExists, Severity: domain.SevCritical, Description: "A consent management platform tag must load on every page."},
		{Name: "Require Adobe ECID tag", Vendor: "Adobe", Target: domain.RuleTargetTag, Operator: domain.RuleOpExists, Severity: domain.SevHigh, Description: "Adobe Experience Cloud ID must initialise."},
		{Name: "GA4 collects client_id", Vendor: "GA4", Target: domain.RuleTargetRequest, Parameter: "cid", Operator: domain.RuleOpExists, Severity: domain.SevHigh, Description: "GA4 collect requests should include the client identifier."},
		{Name: "Block unconsented Meta pixel", Vendor: "Meta", Target: domain.RuleTargetRequest, Parameter: "ev", Operator: domain.RuleOpNotExists, Severity: domain.SevCritical, Description: "No Meta pixel event should fire before consent."},
		{Name: "No email addresses in analytics", Vendor: "GA4", Target: domain.RuleTargetRequest, Parameter: "email", Operator: domain.RuleOpNotMatches, Expected: `^[^@\s]+@[^@\s]+\.[^@\s]+$`, Severity: domain.SevCritical, Description: "Analytics requests must not carry raw email addresses."},
		{Name: "Purchase event has transaction id", Target: domain.RuleTargetEvent, Event: "purchase", Parameter: "ecommerce.transaction_id", Operator: domain.RuleOpExists, Severity: domain.SevMedium, Description: "Purchase data layer events must carry a transaction id."},
	}
	out := make([]*domain.AuditRule, 0, len(starters))
	for _, r := range starters {
		if seen[r.Name] {
			continue
		}
		r.ProjectID = projectID
		r.Enabled = true
		created, err := s.CreateRule(ctx, r)
		if err != nil {
			return nil, err
		}
		out = append(out, created)
	}
	return out, nil
}

func validateRule(r *domain.AuditRule) error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	switch r.Target {
	case domain.RuleTargetRequest, domain.RuleTargetEvent, domain.RuleTargetTag, domain.RuleTargetCookie:
	default:
		return fmt.Errorf("target must be request, event, tag or cookie")
	}
	switch r.Operator {
	case domain.RuleOpExists, domain.RuleOpNotExists, domain.RuleOpEquals, domain.RuleOpNotEquals, domain.RuleOpMatches, domain.RuleOpNotMatches:
	default:
		return fmt.Errorf("unsupported operator %q", r.Operator)
	}
	if r.Severity == "" {
		r.Severity = domain.SevMedium
	}
	switch r.Severity {
	case domain.SevInfo, domain.SevLow, domain.SevMedium, domain.SevHigh, domain.SevCritical:
	default:
		return fmt.Errorf("unsupported severity %q", r.Severity)
	}
	if r.URLPattern != "" {
		if _, err := regexp.Compile(r.URLPattern); err != nil {
			return fmt.Errorf("invalid urlPattern: %w", err)
		}
	}
	switch r.Operator {
	case domain.RuleOpEquals, domain.RuleOpNotEquals:
		if r.Expected == "" {
			return fmt.Errorf("expected value is required for %s", r.Operator)
		}
	case domain.RuleOpMatches, domain.RuleOpNotMatches:
		if r.Expected == "" {
			return fmt.Errorf("expected pattern is required for %s", r.Operator)
		}
		if _, err := regexp.Compile(r.Expected); err != nil {
			return fmt.Errorf("invalid expected pattern: %w", err)
		}
	}
	if r.Target == domain.RuleTargetRequest && strings.TrimSpace(r.Parameter) == "" {
		return fmt.Errorf("request rules require a parameter")
	}
	if r.Target == domain.RuleTargetEvent && strings.TrimSpace(r.Parameter) == "" && strings.TrimSpace(r.Event) == "" {
		return fmt.Errorf("event rules require an event name or parameter")
	}
	return nil
}
