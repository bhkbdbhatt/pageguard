// Package memory implements domain.Store entirely in memory. It is used by
// unit tests and by `--store=memory` local runs. It is safe for concurrent use.
package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/pageguard/pageguard/internal/domain"
)

// Store is an in-memory domain.Store.
type Store struct {
	mu sync.RWMutex

	projects map[domain.ID]*domain.Project
	scans    map[domain.ID]*domain.Scan
	runs     map[domain.ID]*domain.Run
	pages    map[domain.ID]*domain.Page
	results  map[domain.ID]map[string]*domain.ScanResult
	findings map[domain.ID]*domain.Finding
	rules    map[domain.ID]*domain.AuditRule
	events   map[domain.ID][]*domain.Event
	nextEvt  int64

	schedules       map[domain.ID]*domain.Schedule
	alertRules      map[domain.ID]*domain.AlertRule
	alertChannels   map[domain.ID]*domain.AlertChannel
	alertIncidents  map[domain.ID]*domain.AlertIncident
	alertDeliveries map[domain.ID]*domain.AlertDelivery
	journeys        map[domain.ID]*domain.Journey
	journeyRuns     map[domain.ID]*domain.JourneyRun
	privacyProfiles map[domain.ID]*domain.PrivacyProfile
	privacyBatches  map[domain.ID]*domain.PrivacyBatch
	privacyEvals    map[domain.ID]*domain.PrivacyEvaluation
	reports         map[domain.ID]*domain.Report
	reportSchedules map[domain.ID]*domain.ReportSchedule
}

// New returns an empty store.
func New() *Store {
	return &Store{
		projects: map[domain.ID]*domain.Project{},
		scans:    map[domain.ID]*domain.Scan{},
		runs:     map[domain.ID]*domain.Run{},
		pages:    map[domain.ID]*domain.Page{},
		results:  map[domain.ID]map[string]*domain.ScanResult{},
		findings: map[domain.ID]*domain.Finding{},
		rules:    map[domain.ID]*domain.AuditRule{},
		events:   map[domain.ID][]*domain.Event{},

		schedules:       map[domain.ID]*domain.Schedule{},
		alertRules:      map[domain.ID]*domain.AlertRule{},
		alertChannels:   map[domain.ID]*domain.AlertChannel{},
		alertIncidents:  map[domain.ID]*domain.AlertIncident{},
		alertDeliveries: map[domain.ID]*domain.AlertDelivery{},
		journeys:        map[domain.ID]*domain.Journey{},
		journeyRuns:     map[domain.ID]*domain.JourneyRun{},
		privacyProfiles: map[domain.ID]*domain.PrivacyProfile{},
		privacyBatches:  map[domain.ID]*domain.PrivacyBatch{},
		privacyEvals:    map[domain.ID]*domain.PrivacyEvaluation{},
		reports:         map[domain.ID]*domain.Report{},
		reportSchedules: map[domain.ID]*domain.ReportSchedule{},
	}
}

var _ domain.Store = (*Store)(nil)

func (s *Store) CreateProject(_ context.Context, p *domain.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	s.projects[p.ID] = &cp
	return nil
}

func (s *Store) GetProject(_ context.Context, id domain.ID) (*domain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if p, ok := s.projects[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (s *Store) ListProjects(_ context.Context) ([]*domain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Project, 0, len(s.projects))
	for _, p := range s.projects {
		cp := *p
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) CreateScan(_ context.Context, sc *domain.Scan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *sc
	s.scans[sc.ID] = &cp
	return nil
}

func (s *Store) UpdateScan(_ context.Context, sc *domain.Scan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.scans[sc.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *sc
	s.scans[sc.ID] = &cp
	return nil
}

func (s *Store) GetScan(_ context.Context, id domain.ID) (*domain.Scan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sc, ok := s.scans[id]; ok {
		cp := *sc
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (s *Store) ListScans(_ context.Context, projectID domain.ID) ([]*domain.Scan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.Scan{}
	for _, sc := range s.scans {
		if projectID == "" || sc.ProjectID == projectID {
			cp := *sc
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) CreateRun(_ context.Context, r *domain.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *r
	s.runs[r.ID] = &cp
	return nil
}

func (s *Store) UpdateRun(_ context.Context, r *domain.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[r.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *r
	s.runs[r.ID] = &cp
	return nil
}

func (s *Store) GetRun(_ context.Context, id domain.ID) (*domain.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.runs[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (s *Store) ListRuns(_ context.Context, scanID domain.ID, limit int) ([]*domain.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.Run{}
	for _, r := range s.runs {
		if scanID == "" || r.ScanID == scanID {
			cp := *r
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) SavePage(_ context.Context, p *domain.Page) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	s.pages[p.ID] = &cp
	return nil
}

func (s *Store) GetPage(_ context.Context, id domain.ID) (*domain.Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if p, ok := s.pages[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (s *Store) ListPages(_ context.Context, f domain.PageFilter) ([]*domain.Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.Page{}
	for _, p := range s.pages {
		if f.RunID != "" && p.RunID != f.RunID {
			continue
		}
		if f.FailedOnly && p.Error == "" && p.Status > 0 && p.Status < 400 {
			continue
		}
		cp := *p
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].URL < out[j].URL })
	return applyLimitPages(out, f.Limit), nil
}

func (s *Store) SaveScanResult(_ context.Context, r *domain.ScanResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.results[r.RunID]
	if m == nil {
		m = map[string]*domain.ScanResult{}
		s.results[r.RunID] = m
	}
	cp := *r
	m[r.URL] = &cp
	return nil
}

func (s *Store) GetScanResult(_ context.Context, runID domain.ID, url string) (*domain.ScanResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.results[runID][url]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (s *Store) ListScanResults(_ context.Context, runID domain.ID, limit int) ([]*domain.ScanResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.ScanResult, 0, len(s.results[runID]))
	for _, r := range s.results[runID] {
		cp := *r
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].URL < out[j].URL })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) SaveFinding(_ context.Context, fo *domain.Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *fo
	s.findings[fo.ID] = &cp
	return nil
}

func (s *Store) ListFindings(_ context.Context, f domain.FindingFilter) ([]*domain.Finding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.Finding{}
	for _, fo := range s.findings {
		if f.RunID != "" && fo.RunID != f.RunID {
			continue
		}
		if f.MinSeverity != "" && fo.Severity.Less(f.MinSeverity) {
			continue
		}
		if f.RuleID != "" && fo.RuleID != f.RuleID {
			continue
		}
		if f.PluginID != "" && fo.PluginID != f.PluginID {
			continue
		}
		cp := *fo
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[j].Severity.Less(out[i].Severity) // more severe first
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return applyLimitFindings(out, f.Limit), nil
}

func (s *Store) CreateRule(_ context.Context, r *domain.AuditRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *r
	s.rules[r.ID] = &cp
	return nil
}

func (s *Store) GetRule(_ context.Context, id domain.ID) (*domain.AuditRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.rules[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (s *Store) UpdateRule(_ context.Context, r *domain.AuditRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[r.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *r
	s.rules[r.ID] = &cp
	return nil
}

func (s *Store) DeleteRule(_ context.Context, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.rules, id)
	return nil
}

func (s *Store) ListRules(_ context.Context, projectID domain.ID) ([]*domain.AuditRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.AuditRule{}
	for _, r := range s.rules {
		if projectID == "" || r.ProjectID == projectID {
			cp := *r
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].Name < out[j].Name
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Store) AppendEvent(_ context.Context, e *domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextEvt++
	evt := *e
	evt.ID = s.nextEvt
	s.events[e.RunID] = append(s.events[e.RunID], &evt)
	return nil
}

func (s *Store) ListEvents(_ context.Context, runID domain.ID, afterID int64) ([]*domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.Event{}
	for _, e := range s.events[runID] {
		if e.ID > afterID {
			cp := *e
			out = append(out, &cp)
		}
	}
	return out, nil
}

func applyLimitPages(in []*domain.Page, n int) []*domain.Page {
	if n > 0 && len(in) > n {
		return in[:n]
	}
	return in
}

func applyLimitFindings(in []*domain.Finding, n int) []*domain.Finding {
	if n > 0 && len(in) > n {
		return in[:n]
	}
	return in
}
