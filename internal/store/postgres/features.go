package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pageguard/pageguard/internal/domain"
)

func idOrNil(id domain.ID) any {
	if id.IsEmpty() {
		return nil
	}
	return toUUID(id)
}

func uuidSlice(ids []domain.ID) []pgtype.UUID {
	out := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		u := toUUID(id)
		if u.Valid {
			out = append(out, u)
		}
	}
	return out
}

func idsFromText(vals []string) []domain.ID {
	out := make([]domain.ID, 0, len(vals))
	for _, v := range vals {
		out = append(out, toID(v))
	}
	return out
}

func limitSQL(limit int) string {
	if limit > 0 {
		return fmt.Sprintf(` LIMIT %d`, limit)
	}
	return ""
}

// ---- schedules -----------------------------------------------------------

func (s *Store) CreateSchedule(ctx context.Context, v *domain.Schedule) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO schedules (id, project_id, scan_id, name, url, cron, timezone, enabled,
			max_pages, max_depth, consent, owner, last_run_id, last_status, last_run_at, next_run_at,
			created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (id) DO NOTHING`,
		toUUID(v.ID), toUUID(v.ProjectID), idOrNil(v.ScanID), v.Name, v.URL, v.Cron, v.Timezone, v.Enabled,
		v.MaxPages, v.MaxDepth, v.Consent, v.Owner, idOrNil(v.LastRunID), nullString(string(v.LastStatus)),
		v.LastRunAt, v.NextRunAt, v.CreatedAt, v.UpdatedAt)
	return err
}

func (s *Store) GetSchedule(ctx context.Context, id domain.ID) (*domain.Schedule, error) {
	return scanSchedule(s.pool.QueryRow(ctx, scheduleSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdateSchedule(ctx context.Context, v *domain.Schedule) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE schedules SET scan_id=$2, name=$3, url=$4, cron=$5, timezone=$6, enabled=$7,
			max_pages=$8, max_depth=$9, consent=$10, owner=$11, last_run_id=$12, last_status=$13,
			last_run_at=$14, next_run_at=$15, updated_at=$16
		WHERE id=$1`,
		toUUID(v.ID), idOrNil(v.ScanID), v.Name, v.URL, v.Cron, v.Timezone, v.Enabled,
		v.MaxPages, v.MaxDepth, v.Consent, v.Owner, idOrNil(v.LastRunID), nullString(string(v.LastStatus)),
		v.LastRunAt, v.NextRunAt, v.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteSchedule(ctx context.Context, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM schedules WHERE id=$1`, toUUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListSchedules(ctx context.Context, projectID domain.ID) ([]*domain.Schedule, error) {
	q := scheduleSelect
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Schedule{}
	for rows.Next() {
		v, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) ListDueSchedules(ctx context.Context, now time.Time, limit int) ([]*domain.Schedule, error) {
	q := scheduleSelect + ` WHERE enabled AND next_run_at IS NOT NULL AND next_run_at <= $1 ORDER BY next_run_at` + limitSQL(limit)
	rows, err := s.pool.Query(ctx, q, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Schedule{}
	for rows.Next() {
		v, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

const scheduleSelect = `
	SELECT id::text, project_id::text, COALESCE(scan_id::text,''), name, url, cron, timezone, enabled,
	       max_pages, max_depth, consent, owner, COALESCE(last_run_id::text,''), last_status,
	       last_run_at, next_run_at, created_at, updated_at
	  FROM schedules`

func scanSchedule(row rowScanner) (*domain.Schedule, error) {
	var (
		id, projectID, scanID, lastRunID string
		lastStatus                       *string
		v                                domain.Schedule
	)
	if err := row.Scan(&id, &projectID, &scanID, &v.Name, &v.URL, &v.Cron, &v.Timezone, &v.Enabled,
		&v.MaxPages, &v.MaxDepth, &v.Consent, &v.Owner, &lastRunID, &lastStatus,
		&v.LastRunAt, &v.NextRunAt, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.ProjectID = toID(projectID)
	v.ScanID = toID(scanID)
	v.LastRunID = toID(lastRunID)
	if lastStatus != nil {
		v.LastStatus = domain.RunStatus(*lastStatus)
	}
	return &v, nil
}

// ---- alert rules ---------------------------------------------------------

const alertRuleSelect = `
	SELECT id::text, project_id::text, name, enabled, severity, condition, condition_text,
	       channels, channel_ids::text[], last_triggered_at, trigger_count, created_at, updated_at
	  FROM alert_rules`

func (s *Store) CreateAlertRule(ctx context.Context, v *domain.AlertRule) error {
	cond, err := marshal(v.Condition)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO alert_rules (id, project_id, name, enabled, severity, condition, condition_text,
			channels, channel_ids, last_triggered_at, trigger_count, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		toUUID(v.ID), toUUID(v.ProjectID), v.Name, v.Enabled, string(v.Severity), cond, v.ConditionText,
		nonNil(v.Channels), uuidSlice(v.ChannelIDs), v.LastTriggeredAt, v.TriggerCount, v.CreatedAt, v.UpdatedAt)
	return err
}

func (s *Store) GetAlertRule(ctx context.Context, id domain.ID) (*domain.AlertRule, error) {
	return scanAlertRule(s.pool.QueryRow(ctx, alertRuleSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdateAlertRule(ctx context.Context, v *domain.AlertRule) error {
	cond, err := marshal(v.Condition)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE alert_rules SET name=$2, enabled=$3, severity=$4, condition=$5, condition_text=$6,
			channels=$7, channel_ids=$8, last_triggered_at=$9, trigger_count=$10, updated_at=$11
		WHERE id=$1`,
		toUUID(v.ID), v.Name, v.Enabled, string(v.Severity), cond, v.ConditionText,
		nonNil(v.Channels), uuidSlice(v.ChannelIDs), v.LastTriggeredAt, v.TriggerCount, v.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteAlertRule(ctx context.Context, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM alert_rules WHERE id=$1`, toUUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListAlertRules(ctx context.Context, projectID domain.ID) ([]*domain.AlertRule, error) {
	q := alertRuleSelect
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.AlertRule{}
	for rows.Next() {
		v, err := scanAlertRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanAlertRule(row rowScanner) (*domain.AlertRule, error) {
	var (
		id, projectID, severity string
		cond                    []byte
		channelIDs              []string
		v                       domain.AlertRule
	)
	if err := row.Scan(&id, &projectID, &v.Name, &v.Enabled, &severity, &cond, &v.ConditionText,
		&v.Channels, &channelIDs, &v.LastTriggeredAt, &v.TriggerCount, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.ProjectID = toID(projectID)
	v.Severity = domain.Severity(severity)
	v.ChannelIDs = idsFromText(channelIDs)
	unmarshalInto(cond, &v.Condition)
	return &v, nil
}

// ---- alert channels ------------------------------------------------------

const alertChannelSelect = `
	SELECT id::text, project_id::text, name, kind, target, secret_ref, connected,
	       last_delivery_at, created_at, updated_at
	  FROM alert_channels`

func (s *Store) CreateAlertChannel(ctx context.Context, v *domain.AlertChannel) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO alert_channels (id, project_id, name, kind, target, secret_ref, connected,
			last_delivery_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		toUUID(v.ID), toUUID(v.ProjectID), v.Name, v.Kind, v.Target, v.SecretRef, v.Connected,
		v.LastDeliveryAt, v.CreatedAt, v.UpdatedAt)
	return err
}

func (s *Store) GetAlertChannel(ctx context.Context, id domain.ID) (*domain.AlertChannel, error) {
	return scanAlertChannel(s.pool.QueryRow(ctx, alertChannelSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdateAlertChannel(ctx context.Context, v *domain.AlertChannel) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE alert_channels SET name=$2, kind=$3, target=$4, secret_ref=$5, connected=$6,
			last_delivery_at=$7, updated_at=$8
		WHERE id=$1`,
		toUUID(v.ID), v.Name, v.Kind, v.Target, v.SecretRef, v.Connected, v.LastDeliveryAt, v.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteAlertChannel(ctx context.Context, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM alert_channels WHERE id=$1`, toUUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListAlertChannels(ctx context.Context, projectID domain.ID) ([]*domain.AlertChannel, error) {
	q := alertChannelSelect
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.AlertChannel{}
	for rows.Next() {
		v, err := scanAlertChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanAlertChannel(row rowScanner) (*domain.AlertChannel, error) {
	var (
		id, projectID string
		v             domain.AlertChannel
	)
	if err := row.Scan(&id, &projectID, &v.Name, &v.Kind, &v.Target, &v.SecretRef, &v.Connected,
		&v.LastDeliveryAt, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.ProjectID = toID(projectID)
	return &v, nil
}

// ---- alert incidents -----------------------------------------------------

const alertIncidentSelect = `
	SELECT id::text, project_id::text, COALESCE(rule_id::text,''), COALESCE(run_id::text,''),
	       severity, title, message, rule_name, channel, status, acknowledged_at, acknowledged_by,
	       resolved_at, created_at, updated_at
	  FROM alert_incidents`

func (s *Store) CreateAlertIncident(ctx context.Context, v *domain.AlertIncident) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO alert_incidents (id, project_id, rule_id, run_id, severity, title, message,
			rule_name, channel, status, acknowledged_at, acknowledged_by, resolved_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		toUUID(v.ID), toUUID(v.ProjectID), idOrNil(v.RuleID), idOrNil(v.RunID), string(v.Severity),
		v.Title, v.Message, v.RuleName, v.Channel, string(v.Status), v.AcknowledgedAt,
		v.AcknowledgedBy, v.ResolvedAt, v.CreatedAt, v.UpdatedAt)
	return err
}

func (s *Store) GetAlertIncident(ctx context.Context, id domain.ID) (*domain.AlertIncident, error) {
	return scanAlertIncident(s.pool.QueryRow(ctx, alertIncidentSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdateAlertIncident(ctx context.Context, v *domain.AlertIncident) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE alert_incidents SET severity=$2, title=$3, message=$4, rule_name=$5, channel=$6,
			status=$7, acknowledged_at=$8, acknowledged_by=$9, resolved_at=$10, updated_at=$11
		WHERE id=$1`,
		toUUID(v.ID), string(v.Severity), v.Title, v.Message, v.RuleName, v.Channel,
		string(v.Status), v.AcknowledgedAt, v.AcknowledgedBy, v.ResolvedAt, v.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListAlertIncidents(ctx context.Context, projectID domain.ID, limit int) ([]*domain.AlertIncident, error) {
	q := alertIncidentSelect
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at DESC` + limitSQL(limit)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.AlertIncident{}
	for rows.Next() {
		v, err := scanAlertIncident(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanAlertIncident(row rowScanner) (*domain.AlertIncident, error) {
	var (
		id, projectID, ruleID, runID, severity, status string
		v                                              domain.AlertIncident
	)
	if err := row.Scan(&id, &projectID, &ruleID, &runID, &severity, &v.Title, &v.Message,
		&v.RuleName, &v.Channel, &status, &v.AcknowledgedAt, &v.AcknowledgedBy, &v.ResolvedAt,
		&v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.ProjectID = toID(projectID)
	v.RuleID = toID(ruleID)
	v.RunID = toID(runID)
	v.Severity = domain.Severity(severity)
	v.Status = domain.IncidentStatus(status)
	return &v, nil
}

func (s *Store) CreateAlertDelivery(ctx context.Context, v *domain.AlertDelivery) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO alert_deliveries (id, incident_id, channel_id, status, attempt, error,
			attempted_at, delivered_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		toUUID(v.ID), toUUID(v.IncidentID), toUUID(v.ChannelID), v.Status, v.Attempt, v.Error,
		v.AttemptedAt, v.DeliveredAt)
	return err
}

func (s *Store) ListAlertDeliveries(ctx context.Context, incidentID domain.ID) ([]*domain.AlertDelivery, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, incident_id::text, channel_id::text, status, attempt, error, attempted_at, delivered_at
		  FROM alert_deliveries WHERE incident_id=$1 ORDER BY attempted_at, attempt`, toUUID(incidentID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.AlertDelivery{}
	for rows.Next() {
		var (
			id, incident, channel string
			v                     domain.AlertDelivery
		)
		if err := rows.Scan(&id, &incident, &channel, &v.Status, &v.Attempt, &v.Error, &v.AttemptedAt, &v.DeliveredAt); err != nil {
			return nil, err
		}
		v.ID = toID(id)
		v.IncidentID = toID(incident)
		v.ChannelID = toID(channel)
		out = append(out, &v)
	}
	return out, rows.Err()
}

// ---- journeys ------------------------------------------------------------

const journeySelect = `
	SELECT id::text, project_id::text, name, COALESCE(description,''), COALESCE(owner,''),
	       COALESCE(base_url,''), enabled, definition, steps, created_at, updated_at
	  FROM journeys`

func (s *Store) CreateJourney(ctx context.Context, v *domain.Journey) error {
	definition, err := marshal(v.Definition)
	if err != nil {
		return err
	}
	steps, err := marshal(v.Steps)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO journeys (id, project_id, name, description, owner, base_url, enabled,
			definition, steps, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		toUUID(v.ID), toUUID(v.ProjectID), v.Name, v.Description, v.Owner, v.BaseURL, v.Enabled,
		definition, steps, v.CreatedAt, v.UpdatedAt)
	return err
}

func (s *Store) GetJourney(ctx context.Context, id domain.ID) (*domain.Journey, error) {
	return scanJourney(s.pool.QueryRow(ctx, journeySelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdateJourney(ctx context.Context, v *domain.Journey) error {
	definition, err := marshal(v.Definition)
	if err != nil {
		return err
	}
	steps, err := marshal(v.Steps)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE journeys SET name=$2, description=$3, owner=$4, base_url=$5, enabled=$6,
			definition=$7, steps=$8, updated_at=$9
		WHERE id=$1`,
		toUUID(v.ID), v.Name, v.Description, v.Owner, v.BaseURL, v.Enabled, definition, steps, v.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteJourney(ctx context.Context, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM journeys WHERE id=$1`, toUUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListJourneys(ctx context.Context, projectID domain.ID) ([]*domain.Journey, error) {
	q := journeySelect
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Journey{}
	for rows.Next() {
		v, err := scanJourney(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanJourney(row rowScanner) (*domain.Journey, error) {
	var (
		id, projectID string
		definition    []byte
		steps         []byte
		v             domain.Journey
	)
	if err := row.Scan(&id, &projectID, &v.Name, &v.Description, &v.Owner, &v.BaseURL, &v.Enabled,
		&definition, &steps, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.ProjectID = toID(projectID)
	v.Definition = definition
	unmarshalInto(steps, &v.Steps)
	return &v, nil
}

const journeyRunSelect = `
	SELECT id::text, journey_id::text, project_id::text, status, steps_passed, steps_total,
	       duration_ms, error, report, started_at, finished_at, created_at
	  FROM journey_runs`

func (s *Store) CreateJourneyRun(ctx context.Context, v *domain.JourneyRun) error {
	report, err := marshal(v.Report)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO journey_runs (id, journey_id, project_id, status, steps_passed, steps_total,
			duration_ms, error, report, started_at, finished_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		toUUID(v.ID), toUUID(v.JourneyID), toUUID(v.ProjectID), v.Status, v.StepsPassed,
		v.StepsTotal, v.DurationMs, v.Error, report, v.StartedAt, v.FinishedAt, v.CreatedAt)
	return err
}

func (s *Store) GetJourneyRun(ctx context.Context, id domain.ID) (*domain.JourneyRun, error) {
	return scanJourneyRun(s.pool.QueryRow(ctx, journeyRunSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdateJourneyRun(ctx context.Context, v *domain.JourneyRun) error {
	report, err := marshal(v.Report)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE journey_runs SET status=$2, steps_passed=$3, steps_total=$4, duration_ms=$5,
			error=$6, report=$7, started_at=$8, finished_at=$9
		WHERE id=$1`,
		toUUID(v.ID), v.Status, v.StepsPassed, v.StepsTotal, v.DurationMs, v.Error,
		report, v.StartedAt, v.FinishedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListJourneyRuns(ctx context.Context, journeyID domain.ID, limit int) ([]*domain.JourneyRun, error) {
	q := journeyRunSelect
	args := []any{}
	if journeyID != "" {
		q += ` WHERE journey_id=$1`
		args = append(args, toUUID(journeyID))
	}
	q += ` ORDER BY created_at DESC` + limitSQL(limit)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.JourneyRun{}
	for rows.Next() {
		v, err := scanJourneyRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanJourneyRun(row rowScanner) (*domain.JourneyRun, error) {
	var (
		id, journeyID, projectID string
		report                   []byte
		v                        domain.JourneyRun
	)
	if err := row.Scan(&id, &journeyID, &projectID, &v.Status, &v.StepsPassed, &v.StepsTotal,
		&v.DurationMs, &v.Error, &report, &v.StartedAt, &v.FinishedAt, &v.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.JourneyID = toID(journeyID)
	v.ProjectID = toID(projectID)
	v.Report = report
	return &v, nil
}

// ---- privacy -------------------------------------------------------------

const privacyProfileSelect = `
	SELECT id::text, project_id::text, name, region, regulation_ids, config, active, created_at, updated_at
	  FROM privacy_profiles`

func (s *Store) CreatePrivacyProfile(ctx context.Context, v *domain.PrivacyProfile) error {
	cfg, err := marshal(v.Config)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO privacy_profiles (id, project_id, name, region, regulation_ids, config, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		toUUID(v.ID), toUUID(v.ProjectID), v.Name, v.Region, nonNil(v.RegulationIDs), cfg,
		v.Active, v.CreatedAt, v.UpdatedAt)
	return err
}

func (s *Store) GetPrivacyProfile(ctx context.Context, id domain.ID) (*domain.PrivacyProfile, error) {
	return scanPrivacyProfile(s.pool.QueryRow(ctx, privacyProfileSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdatePrivacyProfile(ctx context.Context, v *domain.PrivacyProfile) error {
	cfg, err := marshal(v.Config)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE privacy_profiles SET name=$2, region=$3, regulation_ids=$4, config=$5, active=$6, updated_at=$7
		WHERE id=$1`,
		toUUID(v.ID), v.Name, v.Region, nonNil(v.RegulationIDs), cfg, v.Active, v.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeletePrivacyProfile(ctx context.Context, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM privacy_profiles WHERE id=$1`, toUUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListPrivacyProfiles(ctx context.Context, projectID domain.ID) ([]*domain.PrivacyProfile, error) {
	q := privacyProfileSelect
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.PrivacyProfile{}
	for rows.Next() {
		v, err := scanPrivacyProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanPrivacyProfile(row rowScanner) (*domain.PrivacyProfile, error) {
	var (
		id, projectID string
		cfg           []byte
		v             domain.PrivacyProfile
	)
	if err := row.Scan(&id, &projectID, &v.Name, &v.Region, &v.RegulationIDs, &cfg, &v.Active,
		&v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.ProjectID = toID(projectID)
	v.Config = cfg
	return &v, nil
}

const privacyBatchSelect = `
	SELECT id::text, project_id::text, COALESCE(profile_id::text,''), start_url, status,
	       total_pages, completed_pages, failed_pages, concurrency, error, created_at, started_at, finished_at
	  FROM privacy_batches`

func (s *Store) CreatePrivacyBatch(ctx context.Context, v *domain.PrivacyBatch) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO privacy_batches (id, project_id, profile_id, start_url, status, total_pages,
			completed_pages, failed_pages, concurrency, error, created_at, started_at, finished_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		toUUID(v.ID), toUUID(v.ProjectID), idOrNil(v.ProfileID), v.StartURL, string(v.Status),
		v.TotalPages, v.CompletedPages, v.FailedPages, v.Concurrency, v.Error, v.CreatedAt, v.StartedAt, v.FinishedAt)
	return err
}

func (s *Store) GetPrivacyBatch(ctx context.Context, id domain.ID) (*domain.PrivacyBatch, error) {
	return scanPrivacyBatch(s.pool.QueryRow(ctx, privacyBatchSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdatePrivacyBatch(ctx context.Context, v *domain.PrivacyBatch) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE privacy_batches SET status=$2, total_pages=$3, completed_pages=$4, failed_pages=$5,
			concurrency=$6, error=$7, started_at=$8, finished_at=$9
		WHERE id=$1`,
		toUUID(v.ID), string(v.Status), v.TotalPages, v.CompletedPages, v.FailedPages,
		v.Concurrency, v.Error, v.StartedAt, v.FinishedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListPrivacyBatches(ctx context.Context, projectID domain.ID, limit int) ([]*domain.PrivacyBatch, error) {
	q := privacyBatchSelect
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at DESC` + limitSQL(limit)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.PrivacyBatch{}
	for rows.Next() {
		v, err := scanPrivacyBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) ListRunnablePrivacyBatches(ctx context.Context, limit int) ([]*domain.PrivacyBatch, error) {
	q := privacyBatchSelect + ` WHERE status IN ('queued','running') ORDER BY created_at` + limitSQL(limit)
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.PrivacyBatch{}
	for rows.Next() {
		v, err := scanPrivacyBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanPrivacyBatch(row rowScanner) (*domain.PrivacyBatch, error) {
	var (
		id, projectID, profileID, status string
		v                                domain.PrivacyBatch
	)
	if err := row.Scan(&id, &projectID, &profileID, &v.StartURL, &status, &v.TotalPages,
		&v.CompletedPages, &v.FailedPages, &v.Concurrency, &v.Error, &v.CreatedAt,
		&v.StartedAt, &v.FinishedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.ProjectID = toID(projectID)
	v.ProfileID = toID(profileID)
	v.Status = domain.RunStatus(status)
	return &v, nil
}

const privacyEvaluationSelect = `
	SELECT id::text, batch_id::text, project_id::text, COALESCE(profile_id::text,''), url, status,
	       report, error, created_at, started_at, finished_at
	  FROM privacy_evaluations`

func (s *Store) CreatePrivacyEvaluation(ctx context.Context, v *domain.PrivacyEvaluation) error {
	report, err := marshal(v.Report)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO privacy_evaluations (id, batch_id, project_id, profile_id, url, status,
			report, error, created_at, started_at, finished_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (batch_id, url) DO NOTHING`,
		toUUID(v.ID), toUUID(v.BatchID), toUUID(v.ProjectID), idOrNil(v.ProfileID), v.URL,
		string(v.Status), report, v.Error, v.CreatedAt, v.StartedAt, v.FinishedAt)
	return err
}

func (s *Store) GetPrivacyEvaluation(ctx context.Context, id domain.ID) (*domain.PrivacyEvaluation, error) {
	return scanPrivacyEvaluation(s.pool.QueryRow(ctx, privacyEvaluationSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdatePrivacyEvaluation(ctx context.Context, v *domain.PrivacyEvaluation) error {
	report, err := marshal(v.Report)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE privacy_evaluations SET status=$2, report=$3, error=$4, started_at=$5, finished_at=$6
		WHERE id=$1`,
		toUUID(v.ID), string(v.Status), report, v.Error, v.StartedAt, v.FinishedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListPrivacyEvaluations(ctx context.Context, batchID domain.ID, limit int) ([]*domain.PrivacyEvaluation, error) {
	q := privacyEvaluationSelect
	args := []any{}
	if batchID != "" {
		q += ` WHERE batch_id=$1`
		args = append(args, toUUID(batchID))
	}
	q += ` ORDER BY url` + limitSQL(limit)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.PrivacyEvaluation{}
	for rows.Next() {
		v, err := scanPrivacyEvaluation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) ListQueuedPrivacyEvaluations(ctx context.Context, batchID domain.ID, limit int) ([]*domain.PrivacyEvaluation, error) {
	q := privacyEvaluationSelect + ` WHERE batch_id=$1 AND status='queued' ORDER BY url` + limitSQL(limit)
	rows, err := s.pool.Query(ctx, q, toUUID(batchID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.PrivacyEvaluation{}
	for rows.Next() {
		v, err := scanPrivacyEvaluation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanPrivacyEvaluation(row rowScanner) (*domain.PrivacyEvaluation, error) {
	var (
		id, batchID, projectID, profileID, status string
		report                                    []byte
		v                                         domain.PrivacyEvaluation
	)
	if err := row.Scan(&id, &batchID, &projectID, &profileID, &v.URL, &status, &report,
		&v.Error, &v.CreatedAt, &v.StartedAt, &v.FinishedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.BatchID = toID(batchID)
	v.ProjectID = toID(projectID)
	v.ProfileID = toID(profileID)
	v.Status = domain.RunStatus(status)
	v.Report = report
	return &v, nil
}

// ---- reports -------------------------------------------------------------

const reportSelect = `
	SELECT id::text, project_id::text, template_id, name, scope_type, scope_id, scope_label,
	       format, status, owner, content_type, COALESCE(content,''::bytea), size_bytes, error,
	       created_at, updated_at
	  FROM reports`

func (s *Store) CreateReport(ctx context.Context, v *domain.Report) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO reports (id, project_id, template_id, name, scope_type, scope_id, scope_label,
			format, status, owner, content_type, content, size_bytes, error, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		toUUID(v.ID), toUUID(v.ProjectID), v.TemplateID, v.Name, v.Scope.Type, v.Scope.ID,
		v.Scope.Label, v.Format, v.Status, v.Owner, v.ContentType, v.Content, v.SizeBytes,
		v.Error, v.CreatedAt, v.UpdatedAt)
	return err
}

func (s *Store) GetReport(ctx context.Context, id domain.ID) (*domain.Report, error) {
	return scanReport(s.pool.QueryRow(ctx, reportSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdateReport(ctx context.Context, v *domain.Report) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE reports SET name=$2, scope_type=$3, scope_id=$4, scope_label=$5, format=$6,
			status=$7, owner=$8, content_type=$9, content=$10, size_bytes=$11, error=$12, updated_at=$13
		WHERE id=$1`,
		toUUID(v.ID), v.Name, v.Scope.Type, v.Scope.ID, v.Scope.Label, v.Format, v.Status,
		v.Owner, v.ContentType, v.Content, v.SizeBytes, v.Error, v.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteReport(ctx context.Context, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM reports WHERE id=$1`, toUUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListReports(ctx context.Context, projectID domain.ID, limit int) ([]*domain.Report, error) {
	q := `
		SELECT id::text, project_id::text, template_id, name, scope_type, scope_id, scope_label,
		       format, status, owner, content_type, NULL::bytea, size_bytes, error, created_at, updated_at
		  FROM reports`
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at DESC` + limitSQL(limit)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Report{}
	for rows.Next() {
		v, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanReport(row rowScanner) (*domain.Report, error) {
	var (
		id, projectID string
		v             domain.Report
	)
	if err := row.Scan(&id, &projectID, &v.TemplateID, &v.Name, &v.Scope.Type, &v.Scope.ID,
		&v.Scope.Label, &v.Format, &v.Status, &v.Owner, &v.ContentType, &v.Content,
		&v.SizeBytes, &v.Error, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.ProjectID = toID(projectID)
	return &v, nil
}

const reportScheduleSelect = `
	SELECT id::text, project_id::text, template_id, name, cadence, cron, timezone, recipients,
	       scope_type, scope_id, scope_label, format, enabled, next_run_at, last_run_at, created_at, updated_at
	  FROM report_schedules`

func (s *Store) CreateReportSchedule(ctx context.Context, v *domain.ReportSchedule) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO report_schedules (id, project_id, template_id, name, cadence, cron, timezone,
			recipients, scope_type, scope_id, scope_label, format, enabled, next_run_at, last_run_at,
			created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		toUUID(v.ID), toUUID(v.ProjectID), v.TemplateID, v.Name, v.Cadence, v.Cron, v.Timezone,
		nonNil(v.Recipients), v.Scope.Type, v.Scope.ID, v.Scope.Label, v.Format, v.Enabled,
		v.NextRunAt, v.LastRunAt, v.CreatedAt, v.UpdatedAt)
	return err
}

func (s *Store) GetReportSchedule(ctx context.Context, id domain.ID) (*domain.ReportSchedule, error) {
	return scanReportSchedule(s.pool.QueryRow(ctx, reportScheduleSelect+` WHERE id=$1`, toUUID(id)))
}

func (s *Store) UpdateReportSchedule(ctx context.Context, v *domain.ReportSchedule) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE report_schedules SET template_id=$2, name=$3, cadence=$4, cron=$5, timezone=$6,
			recipients=$7, scope_type=$8, scope_id=$9, scope_label=$10, format=$11, enabled=$12,
			next_run_at=$13, last_run_at=$14, updated_at=$15
		WHERE id=$1`,
		toUUID(v.ID), v.TemplateID, v.Name, v.Cadence, v.Cron, v.Timezone, nonNil(v.Recipients),
		v.Scope.Type, v.Scope.ID, v.Scope.Label, v.Format, v.Enabled, v.NextRunAt, v.LastRunAt, v.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteReportSchedule(ctx context.Context, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM report_schedules WHERE id=$1`, toUUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListReportSchedules(ctx context.Context, projectID domain.ID) ([]*domain.ReportSchedule, error) {
	q := reportScheduleSelect
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ReportSchedule{}
	for rows.Next() {
		v, err := scanReportSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) ListDueReportSchedules(ctx context.Context, now time.Time, limit int) ([]*domain.ReportSchedule, error) {
	q := reportScheduleSelect + ` WHERE enabled AND next_run_at IS NOT NULL AND next_run_at <= $1 ORDER BY next_run_at` + limitSQL(limit)
	rows, err := s.pool.Query(ctx, q, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ReportSchedule{}
	for rows.Next() {
		v, err := scanReportSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanReportSchedule(row rowScanner) (*domain.ReportSchedule, error) {
	var (
		id, projectID string
		v             domain.ReportSchedule
	)
	if err := row.Scan(&id, &projectID, &v.TemplateID, &v.Name, &v.Cadence, &v.Cron, &v.Timezone,
		&v.Recipients, &v.Scope.Type, &v.Scope.ID, &v.Scope.Label, &v.Format, &v.Enabled,
		&v.NextRunAt, &v.LastRunAt, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	v.ID = toID(id)
	v.ProjectID = toID(projectID)
	return &v, nil
}
