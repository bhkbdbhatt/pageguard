package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/cron"
	"github.com/pageguard/pageguard/internal/domain"
)

// ScheduleInput is the payload used to create a scan schedule.
type ScheduleInput struct {
	ProjectID domain.ID `json:"projectId"`
	ScanID    domain.ID `json:"scanId"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Cron      string    `json:"cron"`
	Timezone  string    `json:"timezone"`
	Enabled   *bool     `json:"enabled"`
	MaxPages  int       `json:"maxPages"`
	MaxDepth  int       `json:"maxDepth"`
	Consent   string    `json:"consent"`
	Owner     string    `json:"owner"`
}

// SchedulePatch is a partial update; nil fields are left unchanged.
type SchedulePatch struct {
	Name     *string `json:"name"`
	URL      *string `json:"url"`
	Cron     *string `json:"cron"`
	Timezone *string `json:"timezone"`
	Enabled  *bool   `json:"enabled"`
	MaxPages *int    `json:"maxPages"`
	MaxDepth *int    `json:"maxDepth"`
	Consent  *string `json:"consent"`
	Owner    *string `json:"owner"`
}

// Schedules lists the schedules for a project (or the first project).
func (s *Service) Schedules(ctx context.Context, projectID domain.ID) ([]*domain.Schedule, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.store.ListSchedules(ctx, project.ID)
}

// GetSchedule returns one schedule.
func (s *Service) GetSchedule(ctx context.Context, id domain.ID) (*domain.Schedule, error) {
	return s.store.GetSchedule(ctx, id)
}

// CreateSchedule validates the cron expression and creates the backing scan.
func (s *Service) CreateSchedule(ctx context.Context, in ScheduleInput) (*domain.Schedule, error) {
	project, err := s.resolveProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	rawURL := strings.TrimSpace(in.URL)
	if err := validateHTTPURL(rawURL); err != nil {
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
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}

	scan, err := s.ensureScheduleScan(ctx, project.ID, in.ScanID, name, rawURL, in.MaxPages, in.MaxDepth, in.Consent)
	if err != nil {
		return nil, err
	}

	sched := &domain.Schedule{
		ID:        domain.NewID(),
		ProjectID: project.ID,
		ScanID:    scan.ID,
		Name:      name,
		URL:       rawURL,
		Cron:      strings.TrimSpace(in.Cron),
		Timezone:  timezoneName(loc),
		Enabled:   enabled,
		MaxPages:  in.MaxPages,
		MaxDepth:  in.MaxDepth,
		Consent:   in.Consent,
		Owner:     in.Owner,
		NextRunAt: &next,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if !enabled {
		sched.NextRunAt = nil
	}
	if err := s.store.CreateSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

// UpdateSchedule applies a partial update and recomputes NextRunAt.
func (s *Service) UpdateSchedule(ctx context.Context, id domain.ID, patch SchedulePatch) (*domain.Schedule, error) {
	sched, err := s.store.GetSchedule(ctx, id)
	if err != nil {
		return nil, err
	}
	if patch.Name != nil {
		sched.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.URL != nil {
		sched.URL = strings.TrimSpace(*patch.URL)
	}
	if patch.Cron != nil {
		sched.Cron = strings.TrimSpace(*patch.Cron)
	}
	if patch.Timezone != nil {
		sched.Timezone = *patch.Timezone
	}
	if patch.Enabled != nil {
		sched.Enabled = *patch.Enabled
	}
	if patch.MaxPages != nil {
		sched.MaxPages = *patch.MaxPages
	}
	if patch.MaxDepth != nil {
		sched.MaxDepth = *patch.MaxDepth
	}
	if patch.Consent != nil {
		sched.Consent = *patch.Consent
	}
	if patch.Owner != nil {
		sched.Owner = *patch.Owner
	}
	if sched.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := validateHTTPURL(sched.URL); err != nil {
		return nil, err
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

	if err := s.store.UpdateSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

// DeleteSchedule removes a schedule. The backing scan is retained so its run
// history stays available.
func (s *Service) DeleteSchedule(ctx context.Context, id domain.ID) error {
	return s.store.DeleteSchedule(ctx, id)
}

// RunScheduleNow starts a run for the schedule's scan immediately.
func (s *Service) RunScheduleNow(ctx context.Context, id domain.ID) (*domain.Run, error) {
	sched, err := s.store.GetSchedule(ctx, id)
	if err != nil {
		return nil, err
	}
	scanID := sched.ScanID
	if scanID.IsEmpty() {
		scan, err := s.ensureScheduleScan(ctx, sched.ProjectID, "", sched.Name, sched.URL,
			sched.MaxPages, sched.MaxDepth, sched.Consent)
		if err != nil {
			return nil, err
		}
		scanID = scan.ID
		sched.ScanID = scanID
	} else if _, err := s.store.GetScan(ctx, scanID); err != nil {
		scan, cerr := s.ensureScheduleScan(ctx, sched.ProjectID, "", sched.Name, sched.URL,
			sched.MaxPages, sched.MaxDepth, sched.Consent)
		if cerr != nil {
			return nil, cerr
		}
		scanID = scan.ID
		sched.ScanID = scanID
	}

	run, err := s.StartRun(ctx, scanID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	sched.LastRunID = run.ID
	sched.LastStatus = run.Status
	sched.LastRunAt = &now
	sched.UpdatedAt = now
	if loc, lerr := scheduleLocation(sched.Timezone); lerr == nil {
		if next, nerr := cron.Next(sched.Cron, now, loc); nerr == nil {
			sched.NextRunAt = &next
		}
	}
	if err := s.store.UpdateSchedule(ctx, sched); err != nil {
		return run, err
	}
	return run, nil
}

func (s *Service) ensureScheduleScan(ctx context.Context, projectID, scanID domain.ID, name, rawURL string, maxPages, maxDepth int, consent string) (*domain.Scan, error) {
	if !scanID.IsEmpty() {
		scan, err := s.store.GetScan(ctx, scanID)
		if err == nil {
			return scan, nil
		}
		if err != domain.ErrNotFound {
			return nil, err
		}
	}
	cfg := domain.ScanConfig{
		MaxPages:          maxPages,
		MaxDepth:          maxDepth,
		RenderMode:        domain.RenderBrowser,
		CaptureDataLayer:  true,
		ConsentSimulation: consent,
	}
	cfg.Normalize()
	scan, err := s.CreateScan(ctx, &domain.Scan{
		ProjectID: projectID,
		Name:      name,
		StartURLs: []string{rawURL},
		Config:    cfg,
		Enabled:   true,
	})
	if err != nil {
		return nil, err
	}
	return scan, nil
}

func scheduleLocation(raw string) (*time.Location, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q", raw)
	}
	return loc, nil
}

func timezoneName(loc *time.Location) string {
	if loc == nil {
		return "UTC"
	}
	return loc.String()
}

func validateHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("a valid http(s) url is required")
	}
	return nil
}
