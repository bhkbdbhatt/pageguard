package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/journey"
)

// JourneyInput is the payload used to create or update a journey.
type JourneyInput struct {
	ProjectID   domain.ID            `json:"projectId"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Owner       string               `json:"owner"`
	BaseURL     string               `json:"baseUrl"`
	Enabled     *bool                `json:"enabled"`
	Definition  json.RawMessage      `json:"definition"`
	Steps       []domain.JourneyStep `json:"steps"`
}

// Journeys lists a project's journeys.
func (s *Service) Journeys(ctx context.Context, projectID domain.ID) ([]*domain.Journey, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.store.ListJourneys(ctx, project.ID)
}

// GetJourney returns one journey.
func (s *Service) GetJourney(ctx context.Context, id domain.ID) (*domain.Journey, error) {
	return s.store.GetJourney(ctx, id)
}

// CreateJourney validates the journey document and persists it.
func (s *Service) CreateJourney(ctx context.Context, in JourneyInput) (*domain.Journey, error) {
	project, err := s.resolveProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	def, err := buildJourneyDefinition(name, in.BaseURL, in.Definition, in.Steps)
	if err != nil {
		return nil, err
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	now := time.Now().UTC()
	j := &domain.Journey{
		ID: domain.NewID(), ProjectID: project.ID, Name: name,
		Description: in.Description, Owner: in.Owner, BaseURL: in.BaseURL,
		Enabled: enabled, Definition: def, Steps: in.Steps,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.CreateJourney(ctx, j); err != nil {
		return nil, err
	}
	return j, nil
}

// UpdateJourney applies a partial update.
func (s *Service) UpdateJourney(ctx context.Context, id domain.ID, in JourneyInput) (*domain.Journey, error) {
	j, err := s.store.GetJourney(ctx, id)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		j.Name = name
	}
	if in.Description != "" {
		j.Description = in.Description
	}
	if in.Owner != "" {
		j.Owner = in.Owner
	}
	if in.BaseURL != "" {
		j.BaseURL = in.BaseURL
	}
	if in.Enabled != nil {
		j.Enabled = *in.Enabled
	}
	if len(in.Definition) > 0 || len(in.Steps) > 0 {
		def, err := buildJourneyDefinition(j.Name, j.BaseURL, in.Definition, in.Steps)
		if err != nil {
			return nil, err
		}
		j.Definition = def
		j.Steps = in.Steps
	}
	j.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateJourney(ctx, j); err != nil {
		return nil, err
	}
	return j, nil
}

// DeleteJourney removes a journey definition.
func (s *Service) DeleteJourney(ctx context.Context, id domain.ID) error {
	return s.store.DeleteJourney(ctx, id)
}

// JourneyRuns lists recent runs for a journey.
func (s *Service) JourneyRuns(ctx context.Context, journeyID domain.ID, limit int) ([]*domain.JourneyRun, error) {
	return s.store.ListJourneyRuns(ctx, journeyID, limit)
}

// GetJourneyRun returns one run.
func (s *Service) GetJourneyRun(ctx context.Context, id domain.ID) (*domain.JourneyRun, error) {
	return s.store.GetJourneyRun(ctx, id)
}

// RunJourney starts a journey execution asynchronously and returns the run.
func (s *Service) RunJourney(ctx context.Context, id domain.ID) (*domain.JourneyRun, error) {
	j, err := s.store.GetJourney(ctx, id)
	if err != nil {
		return nil, err
	}
	if !j.Enabled {
		return nil, fmt.Errorf("journey is disabled")
	}
	doc, err := journeyDoc(j)
	if err != nil {
		return nil, err
	}

	s.batchMu.Lock()
	if s.journeyInflight[j.ID] {
		s.batchMu.Unlock()
		return nil, domain.ErrConflict
	}
	s.journeyInflight[j.ID] = true
	s.batchMu.Unlock()

	now := time.Now().UTC()
	run := &domain.JourneyRun{
		ID: domain.NewID(), JourneyID: j.ID, ProjectID: j.ProjectID,
		Status: "running", StepsTotal: len(doc.Steps), StartedAt: now, CreatedAt: now,
	}
	if err := s.store.CreateJourneyRun(ctx, run); err != nil {
		s.batchMu.Lock()
		delete(s.journeyInflight, j.ID)
		s.batchMu.Unlock()
		return nil, err
	}

	go func() {
		defer func() {
			s.batchMu.Lock()
			delete(s.journeyInflight, j.ID)
			s.batchMu.Unlock()
		}()
		rctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		start := time.Now()
		engine := journey.NewEngine(journey.NewPlaywrightSessionFactory(s.browserCmd...))
		report, rerr := engine.Run(rctx, doc, journey.RunOptions{})
		finished := time.Now().UTC()

		run.FinishedAt = &finished
		run.DurationMs = time.Since(start).Milliseconds()
		if rerr != nil {
			run.Status = "failed"
			run.Error = rerr.Error()
		} else {
			run.StepsPassed = report.Summary.Passed
			run.StepsTotal = report.Summary.Steps
			run.DurationMs = report.DurationMs
			if report.Passed {
				run.Status = "passed"
			} else {
				run.Status = "failed"
				run.Error = fmt.Sprintf("%d of %d steps failed", report.Summary.Failed, report.Summary.Steps)
			}
			if raw, merr := json.Marshal(report); merr == nil {
				run.Report = raw
			}
		}
		_ = s.store.UpdateJourneyRun(context.Background(), run)
	}()

	return run, nil
}

func journeyDoc(j *domain.Journey) (*journey.Journey, error) {
	if len(j.Definition) == 0 {
		def, err := buildJourneyDefinition(j.Name, j.BaseURL, nil, j.Steps)
		if err != nil {
			return nil, err
		}
		j.Definition = def
	}
	doc, err := journey.Parse(j.Definition)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func buildJourneyDefinition(name, baseURL string, definition json.RawMessage, steps []domain.JourneyStep) (json.RawMessage, error) {
	var doc *journey.Journey
	if len(definition) > 0 && strings.TrimSpace(string(definition)) != "null" {
		parsed, err := journey.Parse(definition)
		if err != nil {
			return nil, err
		}
		doc = parsed
		if name != "" {
			doc.Name = name
		}
		if doc.BaseURL == "" {
			doc.BaseURL = baseURL
		}
	} else {
		doc = &journey.Journey{Version: 1, Name: name, BaseURL: baseURL}
		for _, step := range steps {
			js, err := legacyStep(step)
			if err != nil {
				return nil, err
			}
			doc.Steps = append(doc.Steps, js)
		}
	}
	if err := doc.Prepare(); err != nil {
		return nil, err
	}
	return json.Marshal(doc)
}

func legacyStep(step domain.JourneyStep) (journey.Step, error) {
	action := strings.ToLower(strings.TrimSpace(step.Action))
	out := journey.Step{
		Action: action, Selector: step.Selector, Value: step.Value,
		TimeoutMs: step.TimeoutMs,
	}
	switch action {
	case journey.ActionNavigate, journey.ActionClick, journey.ActionFill, journey.ActionType, journey.ActionWait:
	default:
		return journey.Step{}, fmt.Errorf("step %d: unsupported action %q (use navigate, click, fill or wait)", step.Order, step.Action)
	}
	out.Expect = legacyExpect(step.Expect)
	return out, nil
}

func legacyExpect(m map[string]any) journey.Expect {
	var e journey.Expect
	if m == nil {
		return e
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k].(string); ok {
				return v
			}
		}
		return ""
	}
	list := func(keys ...string) []string {
		for _, k := range keys {
			if raw, ok := m[k].([]any); ok {
				var out []string
				for _, v := range raw {
					if s, ok := v.(string); ok {
						out = append(out, s)
					}
				}
				return out
			}
			if raw, ok := m[k].([]string); ok {
				return raw
			}
		}
		return nil
	}
	intVal := func(keys ...string) int {
		for _, k := range keys {
			switch v := m[k].(type) {
			case float64:
				return int(v)
			case int:
				return v
			}
		}
		return 0
	}
	e.URLContains = str("urlContains", "url_contains")
	e.TitleContains = str("titleContains", "title_contains")
	e.Status = intVal("status")
	e.Tags = list("tags")
	e.AbsentTags = list("absentTags", "absent_tags")
	e.Events = list("events")
	e.Cookies = list("cookies")
	e.AbsentCookies = list("absentCookies", "absent_cookies")
	return e
}
