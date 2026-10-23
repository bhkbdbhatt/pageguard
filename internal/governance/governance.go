// Package governance derives dashboard-level reporting from persisted scan
// evidence. It deliberately contains no browser automation: metrics are
// computed from the same ScanResult records emitted by the orchestrator.
package governance

import (
	"context"
	"sort"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/rules"
)

// Service exposes governance reports over a Store.
type Service struct {
	store domain.Store
}

// New returns a governance service.
func New(store domain.Store) *Service { return &Service{store: store} }

func (s *Service) project(ctx context.Context, projectID domain.ID) (*domain.Project, error) {
	if !projectID.IsEmpty() {
		return s.store.GetProject(ctx, projectID)
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

func (s *Service) projectScans(ctx context.Context, projectID domain.ID) ([]*domain.Scan, error) {
	return s.store.ListScans(ctx, projectID)
}

func (s *Service) projectRuns(ctx context.Context, projectID domain.ID) ([]*domain.Run, map[domain.ID]*domain.Scan, error) {
	scans, err := s.projectScans(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[domain.ID]*domain.Scan, len(scans))
	for _, scan := range scans {
		byID[scan.ID] = scan
	}
	all := []*domain.Run{}
	for _, scan := range scans {
		runs, err := s.store.ListRuns(ctx, scan.ID, 0)
		if err != nil {
			return nil, nil, err
		}
		all = append(all, runs...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	return all, byID, nil
}

func (s *Service) runContext(ctx context.Context, runID domain.ID) (*domain.Run, *domain.Scan, []*domain.AuditRule, []*domain.ScanResult, error) {
	run, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	scan, err := s.store.GetScan(ctx, run.ScanID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	projectRules, err := s.store.ListRules(ctx, run.ProjectID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	results, err := s.store.ListScanResults(ctx, runID, 0)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return run, scan, projectRules, results, nil
}

func evaluate(projectRules []*domain.AuditRule, results []*domain.ScanResult) (map[string][]rules.Result, rules.Aggregate) {
	byPage := make(map[string][]rules.Result, len(results))
	for _, sr := range results {
		byPage[sr.URL] = rules.EvaluateAll(projectRules, sr)
	}
	return byPage, rules.Summarize(projectRules, byPage)
}

func latestResult(results []*domain.ScanResult) *domain.ScanResult {
	if len(results) == 0 {
		return nil
	}
	sorted := append([]*domain.ScanResult(nil), results...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].FinishedAt.After(sorted[j].FinishedAt) })
	return sorted[0]
}

func pickResult(results []*domain.ScanResult, rawURL string) *domain.ScanResult {
	if rawURL == "" {
		return latestResult(results)
	}
	for _, sr := range results {
		if sr.URL == rawURL || sr.FinalURL == rawURL {
			return sr
		}
	}
	return nil
}

func dayKey(t time.Time) string { return t.UTC().Format("2006-01-02") }

func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var total float64
	for _, v := range vals {
		total += v
	}
	return total / float64(len(vals))
}
