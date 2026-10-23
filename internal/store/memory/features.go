package memory

import (
	"context"
	"sort"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

func (s *Store) CreateSchedule(_ context.Context, v *domain.Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.schedules[v.ID] = &cp
	return nil
}

func (s *Store) GetSchedule(_ context.Context, id domain.ID) (*domain.Schedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.schedules[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdateSchedule(_ context.Context, v *domain.Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.schedules[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.schedules[v.ID] = &cp
	return nil
}

func (s *Store) DeleteSchedule(_ context.Context, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.schedules[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.schedules, id)
	return nil
}

func (s *Store) ListSchedules(_ context.Context, projectID domain.ID) ([]*domain.Schedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.Schedule{}
	for _, v := range s.schedules {
		if projectID == "" || v.ProjectID == projectID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) ListDueSchedules(_ context.Context, now time.Time, limit int) ([]*domain.Schedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.Schedule{}
	for _, v := range s.schedules {
		if !v.Enabled || v.NextRunAt == nil || v.NextRunAt.After(now) {
			continue
		}
		cp := *v
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NextRunAt.Before(*out[j].NextRunAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) CreateAlertRule(_ context.Context, v *domain.AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.alertRules[v.ID] = &cp
	return nil
}

func (s *Store) GetAlertRule(_ context.Context, id domain.ID) (*domain.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.alertRules[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdateAlertRule(_ context.Context, v *domain.AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.alertRules[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.alertRules[v.ID] = &cp
	return nil
}

func (s *Store) DeleteAlertRule(_ context.Context, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.alertRules[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.alertRules, id)
	return nil
}

func (s *Store) ListAlertRules(_ context.Context, projectID domain.ID) ([]*domain.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.AlertRule{}
	for _, v := range s.alertRules {
		if projectID == "" || v.ProjectID == projectID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) CreateAlertChannel(_ context.Context, v *domain.AlertChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.alertChannels[v.ID] = &cp
	return nil
}

func (s *Store) GetAlertChannel(_ context.Context, id domain.ID) (*domain.AlertChannel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.alertChannels[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdateAlertChannel(_ context.Context, v *domain.AlertChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.alertChannels[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.alertChannels[v.ID] = &cp
	return nil
}

func (s *Store) DeleteAlertChannel(_ context.Context, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.alertChannels[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.alertChannels, id)
	return nil
}

func (s *Store) ListAlertChannels(_ context.Context, projectID domain.ID) ([]*domain.AlertChannel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.AlertChannel{}
	for _, v := range s.alertChannels {
		if projectID == "" || v.ProjectID == projectID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) CreateAlertIncident(_ context.Context, v *domain.AlertIncident) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.alertIncidents[v.ID] = &cp
	return nil
}

func (s *Store) GetAlertIncident(_ context.Context, id domain.ID) (*domain.AlertIncident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.alertIncidents[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdateAlertIncident(_ context.Context, v *domain.AlertIncident) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.alertIncidents[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.alertIncidents[v.ID] = &cp
	return nil
}

func (s *Store) ListAlertIncidents(_ context.Context, projectID domain.ID, limit int) ([]*domain.AlertIncident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.AlertIncident{}
	for _, v := range s.alertIncidents {
		if projectID == "" || v.ProjectID == projectID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) CreateAlertDelivery(_ context.Context, v *domain.AlertDelivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.alertDeliveries[v.ID] = &cp
	return nil
}

func (s *Store) ListAlertDeliveries(_ context.Context, incidentID domain.ID) ([]*domain.AlertDelivery, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.AlertDelivery{}
	for _, v := range s.alertDeliveries {
		if v.IncidentID == incidentID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AttemptedAt.Before(out[j].AttemptedAt) })
	return out, nil
}

func (s *Store) CreateJourney(_ context.Context, v *domain.Journey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.journeys[v.ID] = &cp
	return nil
}

func (s *Store) GetJourney(_ context.Context, id domain.ID) (*domain.Journey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.journeys[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdateJourney(_ context.Context, v *domain.Journey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.journeys[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.journeys[v.ID] = &cp
	return nil
}

func (s *Store) DeleteJourney(_ context.Context, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.journeys[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.journeys, id)
	return nil
}

func (s *Store) ListJourneys(_ context.Context, projectID domain.ID) ([]*domain.Journey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.Journey{}
	for _, v := range s.journeys {
		if projectID == "" || v.ProjectID == projectID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) CreateJourneyRun(_ context.Context, v *domain.JourneyRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.journeyRuns[v.ID] = &cp
	return nil
}

func (s *Store) GetJourneyRun(_ context.Context, id domain.ID) (*domain.JourneyRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.journeyRuns[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdateJourneyRun(_ context.Context, v *domain.JourneyRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.journeyRuns[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.journeyRuns[v.ID] = &cp
	return nil
}

func (s *Store) ListJourneyRuns(_ context.Context, journeyID domain.ID, limit int) ([]*domain.JourneyRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.JourneyRun{}
	for _, v := range s.journeyRuns {
		if journeyID == "" || v.JourneyID == journeyID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) CreatePrivacyProfile(_ context.Context, v *domain.PrivacyProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.privacyProfiles[v.ID] = &cp
	return nil
}

func (s *Store) GetPrivacyProfile(_ context.Context, id domain.ID) (*domain.PrivacyProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.privacyProfiles[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdatePrivacyProfile(_ context.Context, v *domain.PrivacyProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.privacyProfiles[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.privacyProfiles[v.ID] = &cp
	return nil
}

func (s *Store) DeletePrivacyProfile(_ context.Context, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.privacyProfiles[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.privacyProfiles, id)
	return nil
}

func (s *Store) ListPrivacyProfiles(_ context.Context, projectID domain.ID) ([]*domain.PrivacyProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.PrivacyProfile{}
	for _, v := range s.privacyProfiles {
		if projectID == "" || v.ProjectID == projectID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) CreatePrivacyBatch(_ context.Context, v *domain.PrivacyBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.privacyBatches[v.ID] = &cp
	return nil
}

func (s *Store) GetPrivacyBatch(_ context.Context, id domain.ID) (*domain.PrivacyBatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.privacyBatches[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdatePrivacyBatch(_ context.Context, v *domain.PrivacyBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.privacyBatches[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.privacyBatches[v.ID] = &cp
	return nil
}

func (s *Store) ListPrivacyBatches(_ context.Context, projectID domain.ID, limit int) ([]*domain.PrivacyBatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.PrivacyBatch{}
	for _, v := range s.privacyBatches {
		if projectID == "" || v.ProjectID == projectID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) ListRunnablePrivacyBatches(_ context.Context, limit int) ([]*domain.PrivacyBatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.PrivacyBatch{}
	for _, v := range s.privacyBatches {
		if v.Status == domain.RunQueued || v.Status == domain.RunRunning {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) CreatePrivacyEvaluation(_ context.Context, v *domain.PrivacyEvaluation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.privacyEvals {
		if existing.BatchID == v.BatchID && existing.URL == v.URL {
			return domain.ErrConflict
		}
	}
	cp := *v
	s.privacyEvals[v.ID] = &cp
	return nil
}

func (s *Store) GetPrivacyEvaluation(_ context.Context, id domain.ID) (*domain.PrivacyEvaluation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.privacyEvals[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdatePrivacyEvaluation(_ context.Context, v *domain.PrivacyEvaluation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.privacyEvals[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.privacyEvals[v.ID] = &cp
	return nil
}

func (s *Store) ListPrivacyEvaluations(_ context.Context, batchID domain.ID, limit int) ([]*domain.PrivacyEvaluation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.PrivacyEvaluation{}
	for _, v := range s.privacyEvals {
		if batchID == "" || v.BatchID == batchID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].URL < out[j].URL })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) ListQueuedPrivacyEvaluations(_ context.Context, batchID domain.ID, limit int) ([]*domain.PrivacyEvaluation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.PrivacyEvaluation{}
	for _, v := range s.privacyEvals {
		if v.BatchID == batchID && v.Status == domain.RunQueued {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].URL < out[j].URL })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) CreateReport(_ context.Context, v *domain.Report) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.reports[v.ID] = &cp
	return nil
}

func (s *Store) GetReport(_ context.Context, id domain.ID) (*domain.Report, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.reports[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdateReport(_ context.Context, v *domain.Report) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.reports[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.reports[v.ID] = &cp
	return nil
}

func (s *Store) DeleteReport(_ context.Context, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.reports[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.reports, id)
	return nil
}

func (s *Store) ListReports(_ context.Context, projectID domain.ID, limit int) ([]*domain.Report, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.Report{}
	for _, v := range s.reports {
		if projectID == "" || v.ProjectID == projectID {
			cp := *v
			cp.Content = nil
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) CreateReportSchedule(_ context.Context, v *domain.ReportSchedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.reportSchedules[v.ID] = &cp
	return nil
}

func (s *Store) GetReportSchedule(_ context.Context, id domain.ID) (*domain.ReportSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.reportSchedules[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdateReportSchedule(_ context.Context, v *domain.ReportSchedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.reportSchedules[v.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *v
	s.reportSchedules[v.ID] = &cp
	return nil
}

func (s *Store) DeleteReportSchedule(_ context.Context, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.reportSchedules[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.reportSchedules, id)
	return nil
}

func (s *Store) ListReportSchedules(_ context.Context, projectID domain.ID) ([]*domain.ReportSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.ReportSchedule{}
	for _, v := range s.reportSchedules {
		if projectID == "" || v.ProjectID == projectID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) ListDueReportSchedules(_ context.Context, now time.Time, limit int) ([]*domain.ReportSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.ReportSchedule{}
	for _, v := range s.reportSchedules {
		if !v.Enabled || v.NextRunAt == nil || v.NextRunAt.After(now) {
			continue
		}
		cp := *v
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NextRunAt.Before(*out[j].NextRunAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
