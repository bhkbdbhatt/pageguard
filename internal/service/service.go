// Package service is the application layer used by the API. It owns run
// orchestration lifecycle (start/cancel) and exposes a small, storage-backed
// surface to HTTP handlers.
package service

import (
	"context"
	"sync"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/events"
	"github.com/pageguard/pageguard/internal/governance"
	"github.com/pageguard/pageguard/internal/orchestrator"
)

// Service coordinates scans and runs.
type Service struct {
	store   domain.Store
	hub     *events.Hub
	manager *orchestrator.Manager
	gov     *governance.Service

	browserCmd []string

	mu       sync.Mutex
	inflight map[domain.ID]context.CancelFunc

	batchMu         sync.Mutex
	batchInflight   map[domain.ID]bool
	journeyInflight map[domain.ID]bool
	profileSeedMu   sync.Mutex
}

// Option customises a Service.
type Option func(*Service)

// WithBrowserCommand configures the browser worker command used by journeys
// and privacy evaluations.
func WithBrowserCommand(cmd []string) Option {
	return func(s *Service) { s.browserCmd = append([]string(nil), cmd...) }
}

// New constructs a Service.
func New(store domain.Store, hub *events.Hub, manager *orchestrator.Manager, opts ...Option) *Service {
	s := &Service{
		store:           store,
		hub:             hub,
		manager:         manager,
		gov:             governance.New(store),
		inflight:        map[domain.ID]context.CancelFunc{},
		batchInflight:   map[domain.ID]bool{},
		journeyInflight: map[domain.ID]bool{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Store exposes the underlying store (used by read-only handlers).
func (s *Service) Store() domain.Store { return s.store }

// Hub exposes the event hub (used by streaming handlers).
func (s *Service) Hub() *events.Hub { return s.hub }

// CreateProject creates a project.
func (s *Service) CreateProject(ctx context.Context, name, slug string) (*domain.Project, error) {
	now := time.Now().UTC()
	p := &domain.Project{ID: domain.NewID(), Name: name, Slug: slug, CreatedAt: now, UpdatedAt: now}
	if p.Slug == "" {
		p.Slug = slugify(name)
	}
	if err := s.store.CreateProject(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// CreateScan validates and stores a scan definition.
func (s *Service) CreateScan(ctx context.Context, scan *domain.Scan) (*domain.Scan, error) {
	scan.Config.Normalize()
	if scan.ID.IsEmpty() {
		scan.ID = domain.NewID()
	}
	now := time.Now().UTC()
	scan.CreatedAt = now
	scan.UpdatedAt = now
	if err := s.store.CreateScan(ctx, scan); err != nil {
		return nil, err
	}
	return scan, nil
}

// StartRun creates a queued run and executes it asynchronously, returning the
// newly created run immediately.
func (s *Service) StartRun(ctx context.Context, scanID domain.ID) (*domain.Run, error) {
	scan, err := s.store.GetScan(ctx, scanID)
	if err != nil {
		return nil, err
	}
	scan.Config.Normalize()

	run := &domain.Run{
		ID:        domain.NewID(),
		ScanID:    scan.ID,
		ProjectID: scan.ProjectID,
		Status:    domain.RunQueued,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	s.hub.Publish(&domain.Event{
		RunID:   run.ID,
		Type:    domain.EvtRunQueued,
		Payload: map[string]any{"status": string(domain.RunQueued)},
	})

	rctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.inflight[run.ID] = cancel
	s.mu.Unlock()

	execRun := *run
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.inflight, run.ID)
			s.mu.Unlock()
		}()
		_ = s.manager.Execute(rctx, scan, &execRun)
		s.EvaluateAlerts(context.Background(), run.ID)
	}()

	return run, nil
}

// CancelRun cancels an in-flight run. It returns domain.ErrNotFound if the run
// is not currently executing.
func (s *Service) CancelRun(runID domain.ID) error {
	s.mu.Lock()
	cancel, ok := s.inflight[runID]
	s.mu.Unlock()
	if !ok {
		return domain.ErrNotFound
	}
	cancel()
	return nil
}

// ResumeRun re-executes a run that is not currently in flight. Because runs are
// checkpointed per page, resume is idempotent: pages already scanned are loaded
// from their stored ScanResult and only the remaining frontier is crawled.
func (s *Service) ResumeRun(ctx context.Context, runID domain.ID) (*domain.Run, error) {
	s.mu.Lock()
	_, busy := s.inflight[runID]
	s.mu.Unlock()
	if busy {
		return nil, domain.ErrConflict
	}

	run, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	scan, err := s.store.GetScan(ctx, run.ID)
	if err != nil {
		scan, err = s.store.GetScan(ctx, run.ScanID)
		if err != nil {
			return nil, err
		}
	}
	scan.Config.Normalize()

	run.Status = domain.RunQueued
	run.Stats = domain.RunStats{}
	run.Error = ""
	run.FinishedAt = nil
	if err := s.store.UpdateRun(ctx, run); err != nil {
		return nil, err
	}

	rctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.inflight[run.ID] = cancel
	s.mu.Unlock()

	execRun := *run
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.inflight, run.ID)
			s.mu.Unlock()
		}()
		_ = s.manager.Execute(rctx, scan, &execRun)
		s.EvaluateAlerts(context.Background(), run.ID)
	}()

	return run, nil
}

// ScanResults returns the checkpointed per-page results for a run, newest
// first is not guaranteed; results are ordered by URL.
func (s *Service) ScanResults(ctx context.Context, runID domain.ID, limit int) ([]*domain.ScanResult, error) {
	return s.store.ListScanResults(ctx, runID, limit)
}

// ScanResult returns a single page's checkpointed result.
func (s *Service) ScanResult(ctx context.Context, runID domain.ID, url string) (*domain.ScanResult, error) {
	return s.store.GetScanResult(ctx, runID, url)
}

// Inflight returns the number of currently executing runs.
func (s *Service) Inflight() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.inflight)
}

func slugify(s string) string {
	out := make([]rune, 0, len(s))
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
			prevDash = false
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
			prevDash = false
		default:
			if !prevDash && len(out) > 0 {
				out = append(out, '-')
				prevDash = true
			}
		}
	}
	if len(out) > 0 && out[len(out)-1] == '-' {
		out = out[:len(out)-1]
	}
	return string(out)
}
