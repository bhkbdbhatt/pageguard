package plugin

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/pkg/pluginsdk"
)

// Registry holds the set of active validators keyed by plugin ID.
type Registry struct {
	mu         sync.RWMutex
	validators map[string]pluginsdk.Validator
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{validators: map[string]pluginsdk.Validator{}}
}

// Register adds or replaces a validator.
func (r *Registry) Register(v pluginsdk.Validator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validators[v.ID()] = v
}

// Get returns a validator by plugin ID.
func (r *Registry) Get(id string) (pluginsdk.Validator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.validators[id]
	return v, ok
}

// Validators returns all validators sorted by ID for deterministic output.
func (r *Registry) Validators() []pluginsdk.Validator {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]pluginsdk.Validator, 0, len(r.validators))
	for _, v := range r.validators {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// Host evaluates a set of validators against observations.
type Host struct {
	registry *Registry
}

// NewHost creates a host over the given registry.
func NewHost(reg *Registry) *Host { return &Host{registry: reg} }

// Registry exposes the underlying registry (e.g. for loader registration).
func (h *Host) Registry() *Registry { return h.registry }

// ValidateInput is one page's worth of validation work.
type ValidateInput struct {
	RunID       domain.ID
	ProjectID   domain.ID
	ScanID      domain.ID
	Page        *domain.Page
	Observation *domain.Observation
	RuleSetIDs  []string
	Config      map[string]any
}

// Validate runs every applicable validator and converts results to domain
// findings. A failing plugin is isolated: its error does not abort the others.
func (h *Host) Validate(ctx context.Context, in ValidateInput) ([]*domain.Finding, error) {
	if in.Observation == nil {
		return nil, nil
	}
	sdkObs := ToSDK(in.Observation)
	scope := pluginsdk.Scope{
		RunID:     in.RunID.String(),
		ProjectID: in.ProjectID.String(),
		ScanID:    in.ScanID.String(),
		Config:    in.Config,
	}

	pageURL := in.Observation.FinalURL
	if pageURL == "" {
		pageURL = in.Observation.URL
	}

	var out []*domain.Finding
	for _, v := range h.registry.Validators() {
		if !ruleSetAllows(in.RuleSetIDs, v.ID()) {
			continue
		}
		findings, err := v.Validate(ctx, sdkObs, scope)
		if err != nil {
			continue
		}
		for _, f := range findings {
			out = append(out, &domain.Finding{
				ID:          domain.NewID(),
				RunID:       in.RunID,
				PageID:      pageID(in.Page),
				PageURL:     pageURL,
				PluginID:    v.ID(),
				RuleID:      f.RuleID,
				Severity:    parseSeverity(f.Severity),
				Title:       f.Title,
				Description: f.Description,
				Evidence:    f.Evidence,
				CreatedAt:   time.Now().UTC(),
			})
		}
	}
	return out, nil
}

func pageID(p *domain.Page) domain.ID {
	if p == nil {
		return ""
	}
	return p.ID
}

func ruleSetAllows(sets []string, pluginID string) bool {
	if len(sets) == 0 {
		return true
	}
	for _, s := range sets {
		if s == pluginID || s == "all" {
			return true
		}
	}
	return false
}

func parseSeverity(s string) domain.Severity {
	switch domain.Severity(s) {
	case domain.SevCritical:
		return domain.SevCritical
	case domain.SevHigh:
		return domain.SevHigh
	case domain.SevMedium:
		return domain.SevMedium
	case domain.SevLow:
		return domain.SevLow
	case domain.SevInfo:
		return domain.SevInfo
	}
	return domain.SevInfo
}
