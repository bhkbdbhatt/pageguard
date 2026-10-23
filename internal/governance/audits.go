package governance

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// AuditRow is one execution row in the audit history.
type AuditRow struct {
	RunID       domain.ID        `json:"runId"`
	ScanID      domain.ID        `json:"scanId"`
	ProjectID   domain.ID        `json:"projectId"`
	ScanName    string           `json:"scanName"`
	StartURL    string           `json:"startUrl"`
	Status      domain.RunStatus `json:"status"`
	StartedAt   *time.Time       `json:"startedAt,omitempty"`
	FinishedAt  *time.Time       `json:"finishedAt,omitempty"`
	DurationSec int64            `json:"durationSec"`
	Stats       domain.RunStats  `json:"stats"`
	Error       string           `json:"error,omitempty"`
	CreatedAt   time.Time        `json:"createdAt"`
}

// Audits returns project runs, newest first.
func (s *Service) Audits(ctx context.Context, projectID domain.ID, limit int) ([]AuditRow, error) {
	project, err := s.project(ctx, projectID)
	if err != nil {
		if err == domain.ErrNotFound {
			return []AuditRow{}, nil
		}
		return nil, err
	}
	runs, scans, err := s.projectRuns(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	out := make([]AuditRow, 0, len(runs))
	for _, run := range runs {
		scan := scans[run.ScanID]
		row := AuditRow{
			RunID: run.ID, ScanID: run.ScanID, ProjectID: run.ProjectID,
			Status: run.Status, StartedAt: run.StartedAt, FinishedAt: run.FinishedAt,
			Stats: run.Stats, Error: run.Error, CreatedAt: run.CreatedAt,
		}
		if scan != nil {
			row.ScanName = scan.Name
			if len(scan.StartURLs) > 0 {
				row.StartURL = scan.StartURLs[0]
			}
		}
		if run.StartedAt != nil {
			end := time.Now().UTC()
			if run.FinishedAt != nil {
				end = *run.FinishedAt
			}
			row.DurationSec = int64(end.Sub(*run.StartedAt).Seconds())
		}
		out = append(out, row)
	}
	return out, nil
}

// AuditCSV renders audit history as CSV.
func (s *Service) AuditCSV(ctx context.Context, projectID domain.ID) ([]byte, error) {
	rows, err := s.Audits(ctx, projectID, 0)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"run_id", "scan", "start_url", "status", "started_at", "duration_sec", "pages_scanned", "pages_failed", "beacons", "findings", "critical", "health_score", "consent_score"})
	for _, r := range rows {
		started := ""
		if r.StartedAt != nil {
			started = r.StartedAt.UTC().Format(time.RFC3339)
		}
		_ = w.Write([]string{
			string(r.RunID), r.ScanName, r.StartURL, string(r.Status), started,
			strconv.FormatInt(r.DurationSec, 10),
			strconv.Itoa(r.Stats.PagesScanned), strconv.Itoa(r.Stats.PagesFailed),
			strconv.Itoa(r.Stats.Beacons), strconv.Itoa(r.Stats.Findings),
			strconv.Itoa(r.Stats.CriticalFindings),
			strconv.FormatFloat(r.Stats.HealthScore, 'f', 1, 64),
			strconv.FormatFloat(r.Stats.ConsentScore, 'f', 1, 64),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
