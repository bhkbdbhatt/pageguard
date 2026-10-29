package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/journey"
	"github.com/pageguard/pageguard/internal/privacy"
)

// PrivacyProfileInput is the payload used to create or update a profile.
type PrivacyProfileInput struct {
	ProjectID     domain.ID       `json:"projectId"`
	Name          string          `json:"name"`
	Region        string          `json:"region"`
	RegulationIDs []string        `json:"regulationIds"`
	Config        json.RawMessage `json:"config"`
	Active        *bool           `json:"active"`
}

// PrivacyBatchInput starts a privacy evaluation for one page and profile.
type PrivacyBatchInput struct {
	ProjectID domain.ID `json:"projectId"`
	ProfileID domain.ID `json:"profileId"`
	StartURL  string    `json:"startUrl"`
}

var defaultPrivacyProfiles = []struct {
	Name        string
	Region      string
	Regulations []string
}{
	{Name: "EU Storefront", Region: "EU", Regulations: []string{"gdpr", "eprivacy"}},
	{Name: "US Operations", Region: "US", Regulations: []string{"ccpa"}},
	{Name: "Brazil", Region: "BR", Regulations: []string{"lgpd"}},
}

// PrivacyProfiles lists profiles, seeding a sensible default set the first
// time a project is used.
func (s *Service) PrivacyProfiles(ctx context.Context, projectID domain.ID) ([]*domain.PrivacyProfile, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	s.profileSeedMu.Lock()
	defer s.profileSeedMu.Unlock()
	list, err := s.store.ListPrivacyProfiles(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		return list, nil
	}
	now := time.Now().UTC()
	for _, seed := range defaultPrivacyProfiles {
		cfg := privacy.Config{Version: 1, Name: seed.Name}
		for _, r := range seed.Regulations {
			cfg.Regulations = append(cfg.Regulations, privacy.Regulation(r))
		}
		_ = cfg.Prepare()
		raw, _ := json.Marshal(cfg)
		p := &domain.PrivacyProfile{
			ID: domain.NewID(), ProjectID: project.ID, Name: seed.Name, Region: seed.Region,
			RegulationIDs: seed.Regulations, Config: raw, Active: true,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.store.CreatePrivacyProfile(ctx, p); err != nil {
			return nil, err
		}
	}
	return s.store.ListPrivacyProfiles(ctx, project.ID)
}

// CreatePrivacyProfile validates and stores a profile.
func (s *Service) CreatePrivacyProfile(ctx context.Context, in PrivacyProfileInput) (*domain.PrivacyProfile, error) {
	project, err := s.resolveProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	regs, err := normalizeRegulations(in.RegulationIDs)
	if err != nil {
		return nil, err
	}
	raw, err := buildPrivacyConfig(name, regs, in.Config)
	if err != nil {
		return nil, err
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	now := time.Now().UTC()
	p := &domain.PrivacyProfile{
		ID: domain.NewID(), ProjectID: project.ID, Name: name, Region: in.Region,
		RegulationIDs: regs, Config: raw, Active: active, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.CreatePrivacyProfile(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdatePrivacyProfile applies a partial update.
func (s *Service) UpdatePrivacyProfile(ctx context.Context, id domain.ID, in PrivacyProfileInput) (*domain.PrivacyProfile, error) {
	p, err := s.store.GetPrivacyProfile(ctx, id)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		p.Name = name
	}
	if in.Region != "" {
		p.Region = in.Region
	}
	if in.Active != nil {
		p.Active = *in.Active
	}
	regs := p.RegulationIDs
	if in.RegulationIDs != nil {
		if regs, err = normalizeRegulations(in.RegulationIDs); err != nil {
			return nil, err
		}
	}
	if len(in.Config) > 0 || in.RegulationIDs != nil {
		raw, cerr := buildPrivacyConfig(p.Name, regs, in.Config)
		if cerr != nil {
			return nil, cerr
		}
		p.Config = raw
		p.RegulationIDs = regs
	}
	p.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdatePrivacyProfile(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// DeletePrivacyProfile removes a profile.
func (s *Service) DeletePrivacyProfile(ctx context.Context, id domain.ID) error {
	return s.store.DeletePrivacyProfile(ctx, id)
}

// PrivacyBatches lists recent batches for a project.
func (s *Service) PrivacyBatches(ctx context.Context, projectID domain.ID, limit int) ([]*domain.PrivacyBatch, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.store.ListPrivacyBatches(ctx, project.ID, limit)
}

// GetPrivacyBatch returns one batch.
func (s *Service) GetPrivacyBatch(ctx context.Context, id domain.ID) (*domain.PrivacyBatch, error) {
	return s.store.GetPrivacyBatch(ctx, id)
}

// PrivacyEvaluations lists the evaluations of a batch.
func (s *Service) PrivacyEvaluations(ctx context.Context, batchID domain.ID, limit int) ([]*domain.PrivacyEvaluation, error) {
	return s.store.ListPrivacyEvaluations(ctx, batchID, limit)
}

// StartPrivacyBatch queues and asynchronously evaluates one page under the
// selected profile.
func (s *Service) StartPrivacyBatch(ctx context.Context, in PrivacyBatchInput) (*domain.PrivacyBatch, error) {
	project, err := s.resolveProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	profile, err := s.store.GetPrivacyProfile(ctx, in.ProfileID)
	if err != nil {
		return nil, err
	}
	if profile.ProjectID != project.ID {
		return nil, domain.ErrNotFound
	}
	rawURL := strings.TrimSpace(in.StartURL)
	if err := validateHTTPURL(rawURL); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	batch := &domain.PrivacyBatch{
		ID: domain.NewID(), ProjectID: project.ID, ProfileID: profile.ID,
		StartURL: rawURL, Status: domain.RunQueued, TotalPages: 1, Concurrency: 1,
		CreatedAt: now,
	}
	if err := s.store.CreatePrivacyBatch(ctx, batch); err != nil {
		return nil, err
	}
	eval := &domain.PrivacyEvaluation{
		ID: domain.NewID(), BatchID: batch.ID, ProjectID: project.ID,
		ProfileID: profile.ID, URL: rawURL, Status: domain.RunQueued, CreatedAt: now,
	}
	if err := s.store.CreatePrivacyEvaluation(ctx, eval); err != nil {
		return nil, err
	}
	go s.runPrivacyBatch(batch.ID)
	return batch, nil
}

// ProcessPrivacyBatches resumes queued or interrupted batches.
func (s *Service) ProcessPrivacyBatches(ctx context.Context) {
	batches, err := s.store.ListRunnablePrivacyBatches(ctx, 25)
	if err != nil {
		return
	}
	for _, batch := range batches {
		if batch == nil {
			continue
		}
		s.batchMu.Lock()
		busy := s.batchInflight[batch.ID]
		s.batchMu.Unlock()
		if busy {
			continue
		}
		go s.runPrivacyBatch(batch.ID)
	}
}

func (s *Service) runPrivacyBatch(batchID domain.ID) {
	s.batchMu.Lock()
	if s.batchInflight[batchID] {
		s.batchMu.Unlock()
		return
	}
	s.batchInflight[batchID] = true
	s.batchMu.Unlock()
	defer func() {
		s.batchMu.Lock()
		delete(s.batchInflight, batchID)
		s.batchMu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	batch, err := s.store.GetPrivacyBatch(ctx, batchID)
	if err != nil || batch.Status.Terminal() {
		return
	}
	now := time.Now().UTC()
	batch.Status = domain.RunRunning
	if batch.StartedAt == nil {
		batch.StartedAt = &now
	}
	batch.Error = ""
	_ = s.store.UpdatePrivacyBatch(ctx, batch)

	var profile *domain.PrivacyProfile
	if !batch.ProfileID.IsEmpty() {
		profile, _ = s.store.GetPrivacyProfile(ctx, batch.ProfileID)
	}
	cfg, cfgErr := privacyConfig(profile)

	queued, err := s.store.ListQueuedPrivacyEvaluations(ctx, batch.ID, 100)
	if err == nil {
		for _, eval := range queued {
			s.runPrivacyEvaluation(ctx, eval, cfg, cfgErr)
		}
	}

	all, err := s.store.ListPrivacyEvaluations(ctx, batch.ID, 0)
	if err != nil {
		return
	}
	completed, failed, pending := 0, 0, 0
	for _, eval := range all {
		switch eval.Status {
		case domain.RunSucceeded:
			completed++
		case domain.RunFailed:
			failed++
		case domain.RunQueued, domain.RunRunning:
			pending++
		}
	}
	finished := time.Now().UTC()
	batch.TotalPages = len(all)
	batch.CompletedPages = completed
	batch.FailedPages = failed
	if pending == 0 {
		batch.FinishedAt = &finished
		if failed > 0 {
			batch.Status = domain.RunFailed
			if batch.Error == "" {
				batch.Error = fmt.Sprintf("%d of %d evaluations failed", failed, len(all))
			}
		} else {
			batch.Status = domain.RunSucceeded
		}
	}
	_ = s.store.UpdatePrivacyBatch(ctx, batch)
}

func (s *Service) runPrivacyEvaluation(ctx context.Context, eval *domain.PrivacyEvaluation, cfg *privacy.Config, cfgErr error) {
	started := time.Now().UTC()
	eval.Status = domain.RunRunning
	eval.StartedAt = &started
	eval.Error = ""
	_ = s.store.UpdatePrivacyEvaluation(ctx, eval)

	fail := func(msg string) {
		finished := time.Now().UTC()
		eval.Status = domain.RunFailed
		eval.Error = msg
		eval.FinishedAt = &finished
		_ = s.store.UpdatePrivacyEvaluation(context.Background(), eval)
	}
	if cfgErr != nil {
		fail(cfgErr.Error())
		return
	}
	if len(s.browserCmd) == 0 {
		fail("browser worker is not configured; set --browser-cmd or PAGEGUARD_BROWSER_CMD")
		return
	}
	rescanner := &privacy.BrowserRescanner{
		Factory: journey.NewPlaywrightSessionFactory(s.browserCmd...),
		URL:     eval.URL,
	}
	report, err := privacy.Evaluate(ctx, cfg, rescanner)
	if err != nil {
		fail(err.Error())
		return
	}
	raw, err := json.Marshal(report)
	if err != nil {
		fail(err.Error())
		return
	}
	finished := time.Now().UTC()
	eval.Status = domain.RunSucceeded
	eval.Report = raw
	eval.FinishedAt = &finished
	_ = s.store.UpdatePrivacyEvaluation(ctx, eval)
}

func privacyConfig(p *domain.PrivacyProfile) (*privacy.Config, error) {
	if p == nil {
		return nil, fmt.Errorf("privacy profile not found")
	}
	if len(p.Config) > 0 {
		return privacy.Parse(p.Config)
	}
	regs, err := normalizeRegulations(p.RegulationIDs)
	if err != nil {
		return nil, err
	}
	cfg := &privacy.Config{Version: 1, Name: p.Name}
	for _, r := range regs {
		cfg.Regulations = append(cfg.Regulations, privacy.Regulation(r))
	}
	if err := cfg.Prepare(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func buildPrivacyConfig(name string, regs []string, raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) > 0 && strings.TrimSpace(string(raw)) != "null" {
		cfg, err := privacy.Parse(raw)
		if err != nil {
			return nil, err
		}
		if name != "" {
			cfg.Name = name
		}
		if len(regs) > 0 {
			cfg.Regulations = nil
			for _, r := range regs {
				cfg.Regulations = append(cfg.Regulations, privacy.Regulation(r))
			}
		}
		if err := cfg.Prepare(); err != nil {
			return nil, err
		}
		return json.Marshal(cfg)
	}
	cfg := &privacy.Config{Version: 1, Name: name}
	for _, r := range regs {
		cfg.Regulations = append(cfg.Regulations, privacy.Regulation(r))
	}
	if err := cfg.Prepare(); err != nil {
		return nil, err
	}
	return json.Marshal(cfg)
}

func normalizeRegulations(raw []string) ([]string, error) {
	if len(raw) == 0 {
		raw = []string{"gdpr", "ccpa", "lgpd", "eprivacy"}
	}
	known := map[string]bool{"gdpr": true, "ccpa": true, "lgpd": true, "eprivacy": true}
	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		r = strings.ToLower(strings.TrimSpace(r))
		if r == "" {
			continue
		}
		if !known[r] {
			return nil, fmt.Errorf("unknown regulation %q", r)
		}
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one regulation is required")
	}
	return out, nil
}
