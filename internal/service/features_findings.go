package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// FindingPatch updates the triage fields of a finding.
type FindingPatch struct {
	Status *domain.FindingStatus `json:"status"`
	Owner  *string               `json:"owner"`
	Note   *string               `json:"note"`
}

// Findings lists findings for a run, enforcing API-key project scope.
func (s *Service) Findings(ctx context.Context, filter domain.FindingFilter) ([]*domain.Finding, error) {
	if err := s.authorizeRun(ctx, filter.RunID); err != nil {
		return nil, err
	}
	if filter.Status != "" && !domain.ValidFindingStatus(filter.Status) {
		return nil, fmt.Errorf("status must be open, acknowledged, in_progress, resolved or ignored")
	}
	return s.store.ListFindings(ctx, filter)
}

// UpdateFinding changes a finding's owner, note or triage status.
func (s *Service) UpdateFinding(ctx context.Context, id domain.ID, patch FindingPatch) (*domain.Finding, error) {
	finding, err := s.store.GetFinding(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.authorizeRun(ctx, finding.RunID); err != nil {
		return nil, err
	}
	if patch.Status != nil {
		status := domain.FindingStatus(strings.ToLower(strings.TrimSpace(string(*patch.Status))))
		if !domain.ValidFindingStatus(status) {
			return nil, fmt.Errorf("status must be open, acknowledged, in_progress, resolved or ignored")
		}
		finding.Status = status
	}
	if patch.Owner != nil {
		finding.Owner = strings.TrimSpace(*patch.Owner)
	}
	if patch.Note != nil {
		finding.Note = strings.TrimSpace(*patch.Note)
	}
	if finding.Status == "" {
		finding.Status = domain.FindingOpen
	}
	finding.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateFinding(ctx, finding); err != nil {
		return nil, err
	}
	return finding, nil
}

func (s *Service) authorizeRun(ctx context.Context, runID domain.ID) error {
	sc, ok := authScope(ctx)
	if !ok || sc.HasRole(domain.RoleAdmin) {
		return nil
	}
	if runID.IsEmpty() {
		return fmt.Errorf("%w: findings must be scoped to a run", domain.ErrForbidden)
	}
	run, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if run.ProjectID != sc.ProjectID {
		return fmt.Errorf("%w: run is outside this API key's scope", domain.ErrForbidden)
	}
	return nil
}
