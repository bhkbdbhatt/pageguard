package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/scope"
)

// resolveProject returns the requested project or, when id is empty, the
// auth scope's pinned project (API keys), falling back to the first project.
// It returns domain.ErrNotFound when no project exists.
func (s *Service) resolveProject(ctx context.Context, id domain.ID) (*domain.Project, error) {
	return s.resolveProjectIn(ctx, id, true)
}

// resolveProjectIn resolves a project with optional fallback to the first
// project when `fallback` is set. Scoped request context (API keys) is
// honoured before any fallback: an explicit id outside the scope is rejected.
func (s *Service) resolveProjectIn(ctx context.Context, id domain.ID, fallback bool) (*domain.Project, error) {
	scope, hasScope := scope.From(ctx)
	if hasScope {
		if !scope.HasRole(domain.RoleAdmin) {
			if id.IsEmpty() {
				id = scope.ProjectID
			}
			if id != scope.ProjectID {
				return nil, fmt.Errorf("project %s is outside this API key's scope", id)
			}
		} else if id.IsEmpty() && !scope.ProjectID.IsEmpty() {
			id = scope.ProjectID
		}
	}
	if !id.IsEmpty() {
		return s.store.GetProject(ctx, id)
	}
	if !fallback && hasScope {
		return nil, domain.ErrNotFound
	}
	projects, err := s.store.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return nil, domain.ErrNotFound
	}
	return projects[0], nil
}

// latestRun returns the most recent run for a project, if any.
func (s *Service) latestRun(ctx context.Context, projectID domain.ID) (*domain.Run, error) {
	scans, err := s.store.ListScans(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var latest *domain.Run
	for _, scan := range scans {
		runs, err := s.store.ListRuns(ctx, scan.ID, 0)
		if err != nil {
			return nil, err
		}
		for _, run := range runs {
			if latest == nil || run.CreatedAt.After(latest.CreatedAt) {
				latest = run
			}
		}
	}
	return latest, nil
}

// StartBackground starts the periodic worker that runs due schedules, report
// schedules and resumes privacy batches. It blocks until ctx is cancelled, so
// callers normally invoke it in a goroutine.
func (s *Service) StartBackground(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	s.Tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Tick(ctx)
		}
	}
}

// Tick runs every due background task once. Errors are intentionally ignored:
// one failing schedule must not block the rest.
func (s *Service) Tick(ctx context.Context) {
	now := time.Now().UTC()
	s.ProcessSchedules(ctx, now)
	s.ProcessReportSchedules(ctx, now)
	s.ProcessPrivacyBatches(ctx)
}

// ProcessSchedules starts every due scan schedule and advances NextRunAt.
func (s *Service) ProcessSchedules(ctx context.Context, now time.Time) {
	due, err := s.store.ListDueSchedules(ctx, now, 25)
	if err != nil {
		return
	}
	for _, sched := range due {
		if sched == nil || !sched.Enabled {
			continue
		}
		if _, err := s.RunScheduleNow(ctx, sched.ID); err != nil {
			continue
		}
	}
}

// authScope returns the request's authorization identity (API keys), if any.
func authScope(ctx context.Context) (*scope.Scope, bool) {
	return scope.From(ctx)
}

func sortByCreatedDesc[T any](items []T, created func(T) time.Time) []T {
	sort.SliceStable(items, func(i, j int) bool { return created(items[i]).After(created(items[j])) })
	return items
}
