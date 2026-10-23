// Package postgres implements domain.Store on PostgreSQL using pgx.
//
// It is the production persistence backend. Cores stay dependency-free; only
// this package (plus the entrypoint) pulls in pgx.
package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pageguard/pageguard/internal/domain"
)

// Store is a PostgreSQL-backed domain.Store.
type Store struct {
	pool *pgxpool.Pool
}

var _ domain.Store = (*Store)(nil)

// New opens a connection pool to dsn and verifies connectivity.
func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse dsn: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// NewWithPool wraps an existing pool (used by tests).
func NewWithPool(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Close releases the connection pool.
func (s *Store) Close() { s.pool.Close() }

// Pool exposes the underlying pool (e.g. for migrations).
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// Migrate applies every *.sql file in dir, in lexical order, using the simple
// query protocol so multi-statement files (with their own BEGIN/COMMIT) run.
func (s *Store) Migrate(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("postgres: read migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("postgres: read %s: %w", name, err)
		}
		conn, err := s.pool.Acquire(ctx)
		if err != nil {
			return fmt.Errorf("postgres: acquire: %w", err)
		}
		_, execErr := conn.Conn().PgConn().Exec(ctx, string(raw)).ReadAll()
		conn.Release()
		if execErr != nil {
			return fmt.Errorf("postgres: migrate %s: %w", name, execErr)
		}
	}
	return nil
}

// ---- helpers -------------------------------------------------------------

// toUUID encodes a Go ID into a pgtype.UUID for a uuid column. pgx only
// encodes uuid parameters from UUIDValuer values (not plain strings), so the
// hex id is decoded explicitly. Dashes are tolerated; an unparseable id yields
// an invalid (NULL) UUID.
func toUUID(id domain.ID) pgtype.UUID {
	s := strings.ReplaceAll(strings.TrimSpace(string(id)), "-", "")
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 16 {
		return pgtype.UUID{}
	}
	var arr [16]byte
	copy(arr[:], b)
	return pgtype.UUID{Bytes: arr, Valid: true}
}

// toID converts a PostgreSQL uuid text (dashed) into a domain.ID (no dashes).
func toID(s string) domain.ID { return domain.ID(strings.ReplaceAll(s, "-", "")) }

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func marshal(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func unmarshalInto(raw []byte, v any) {
	if len(raw) == 0 {
		return
	}
	_ = json.Unmarshal(raw, v)
}

func nonNil(ss []string) []string {
	if ss == nil {
		return []string{}
	}
	return ss
}

// severityRank mirrors domain.Severity ordering for SQL filtering.
const severityRankSQL = `CASE %s WHEN 'critical' THEN 5 WHEN 'high' THEN 4 WHEN 'medium' THEN 3 WHEN 'low' THEN 2 ELSE 1 END`

// ---- projects ------------------------------------------------------------

func (s *Store) CreateProject(ctx context.Context, p *domain.Project) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO projects (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE
		  SET name = EXCLUDED.name, slug = EXCLUDED.slug, updated_at = EXCLUDED.updated_at`,
		toUUID(p.ID), p.Name, p.Slug, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *Store) GetProject(ctx context.Context, id domain.ID) (*domain.Project, error) {
	var (
		pid string
		p   domain.Project
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, name, slug, created_at, updated_at FROM projects WHERE id = $1`,
		toUUID(id)).Scan(&pid, &p.Name, &p.Slug, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	p.ID = toID(pid)
	return &p, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, slug, created_at, updated_at FROM projects ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Project{}
	for rows.Next() {
		var (
			pid string
			p   domain.Project
		)
		if err := rows.Scan(&pid, &p.Name, &p.Slug, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.ID = toID(pid)
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ---- scans ---------------------------------------------------------------

func (s *Store) CreateScan(ctx context.Context, sc *domain.Scan) error {
	cfg, err := marshal(sc.Config)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO scans (id, project_id, name, start_urls, config, rule_set_ids, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING`,
		toUUID(sc.ID), toUUID(sc.ProjectID), sc.Name, nonNil(sc.StartURLs),
		cfg, nonNil(sc.RuleSetIDs), sc.Enabled, sc.CreatedAt, sc.UpdatedAt)
	return err
}

func (s *Store) UpdateScan(ctx context.Context, sc *domain.Scan) error {
	cfg, err := marshal(sc.Config)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE scans
		   SET name = $2, start_urls = $3, config = $4, rule_set_ids = $5, enabled = $6, updated_at = $7
		 WHERE id = $1`,
		toUUID(sc.ID), sc.Name, nonNil(sc.StartURLs), cfg, nonNil(sc.RuleSetIDs), sc.Enabled, sc.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) GetScan(ctx context.Context, id domain.ID) (*domain.Scan, error) {
	var (
		sid string
		pid string
		cfg []byte
		sc  domain.Scan
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, name, start_urls, config, rule_set_ids, enabled, created_at, updated_at
		  FROM scans WHERE id = $1`, toUUID(id)).
		Scan(&sid, &pid, &sc.Name, &sc.StartURLs, &cfg, &sc.RuleSetIDs, &sc.Enabled, &sc.CreatedAt, &sc.UpdatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	sc.ID = toID(sid)
	sc.ProjectID = toID(pid)
	unmarshalInto(cfg, &sc.Config)
	return &sc, nil
}

func (s *Store) ListScans(ctx context.Context, projectID domain.ID) ([]*domain.Scan, error) {
	q := `
		SELECT id::text, project_id::text, name, start_urls, config, rule_set_ids, enabled, created_at, updated_at
		  FROM scans`
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id = $1`
		args = append(args, toUUID(projectID))
	}
	q += ` ORDER BY created_at`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Scan{}
	for rows.Next() {
		var (
			sid string
			pid string
			cfg []byte
			sc  domain.Scan
		)
		if err := rows.Scan(&sid, &pid, &sc.Name, &sc.StartURLs, &cfg, &sc.RuleSetIDs, &sc.Enabled, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, err
		}
		sc.ID = toID(sid)
		sc.ProjectID = toID(pid)
		unmarshalInto(cfg, &sc.Config)
		out = append(out, &sc)
	}
	return out, rows.Err()
}

// ---- runs ----------------------------------------------------------------

func (s *Store) CreateRun(ctx context.Context, r *domain.Run) error {
	stats, err := marshal(r.Stats)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO scan_runs (id, scan_id, project_id, status, stats, error, started_at, finished_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING`,
		toUUID(r.ID), toUUID(r.ScanID), toUUID(r.ProjectID), string(r.Status),
		stats, nullString(r.Error), r.StartedAt, r.FinishedAt, r.CreatedAt)
	return err
}

func (s *Store) UpdateRun(ctx context.Context, r *domain.Run) error {
	stats, err := marshal(r.Stats)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE scan_runs
		   SET status = $2, stats = $3, error = $4, started_at = $5, finished_at = $6
		 WHERE id = $1`,
		toUUID(r.ID), string(r.Status), stats, nullString(r.Error), r.StartedAt, r.FinishedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) GetRun(ctx context.Context, id domain.ID) (*domain.Run, error) {
	var (
		rid, sid, pid string
		stats         []byte
		r             domain.Run
		errText       *string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, scan_id::text, project_id::text, status, stats, error, started_at, finished_at, created_at
		  FROM scan_runs WHERE id = $1`, toUUID(id)).
		Scan(&rid, &sid, &pid, &r.Status, &stats, &errText, &r.StartedAt, &r.FinishedAt, &r.CreatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	r.ID = toID(rid)
	r.ScanID = toID(sid)
	r.ProjectID = toID(pid)
	if errText != nil {
		r.Error = *errText
	}
	unmarshalInto(stats, &r.Stats)
	return &r, nil
}

func (s *Store) ListRuns(ctx context.Context, scanID domain.ID, limit int) ([]*domain.Run, error) {
	q := `
		SELECT id::text, scan_id::text, project_id::text, status, stats, error, started_at, finished_at, created_at
		  FROM scan_runs`
	args := []any{}
	if scanID != "" {
		q += ` WHERE scan_id = $1`
		args = append(args, toUUID(scanID))
	}
	q += ` ORDER BY created_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, limit)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Run{}
	for rows.Next() {
		var (
			rid, sid, pid string
			stats         []byte
			r             domain.Run
			errText       *string
		)
		if err := rows.Scan(&rid, &sid, &pid, &r.Status, &stats, &errText, &r.StartedAt, &r.FinishedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = toID(rid)
		r.ScanID = toID(sid)
		r.ProjectID = toID(pid)
		if errText != nil {
			r.Error = *errText
		}
		unmarshalInto(stats, &r.Stats)
		out = append(out, &r)
	}
	return out, rows.Err()
}

// ---- pages ---------------------------------------------------------------

func (s *Store) SavePage(ctx context.Context, p *domain.Page) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO run_pages (id, run_id, url, final_url, status, depth, rendered, title, content_type, duration_ms, error, scanned_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (run_id, url) DO UPDATE
		  SET final_url = EXCLUDED.final_url, status = EXCLUDED.status, depth = EXCLUDED.depth,
		      rendered = EXCLUDED.rendered, title = EXCLUDED.title, content_type = EXCLUDED.content_type,
		      duration_ms = EXCLUDED.duration_ms, error = EXCLUDED.error, scanned_at = EXCLUDED.scanned_at`,
		toUUID(p.ID), toUUID(p.RunID), p.URL, nullString(p.FinalURL), p.Status, p.Depth,
		p.Rendered, nullString(p.Title), nullString(p.ContentType), p.DurationMs, nullString(p.Error), p.ScannedAt)
	return err
}

func (s *Store) GetPage(ctx context.Context, id domain.ID) (*domain.Page, error) {
	var (
		pgid                   string
		rid                    string
		finalURL, title, ctype *string
		errText                *string
		p                      domain.Page
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, run_id::text, url, final_url, status, depth, rendered, title, content_type, duration_ms, error, scanned_at
		  FROM run_pages WHERE id = $1`, toUUID(id)).
		Scan(&pgid, &rid, &p.URL, &finalURL, &p.Status, &p.Depth, &p.Rendered, &title, &ctype, &p.DurationMs, &errText, &p.ScannedAt)
	if err != nil {
		return nil, notFound(err)
	}
	p.ID = toID(pgid)
	p.RunID = toID(rid)
	assignStr(&p.FinalURL, finalURL)
	assignStr(&p.Title, title)
	assignStr(&p.ContentType, ctype)
	assignStr(&p.Error, errText)
	return &p, nil
}

func (s *Store) ListPages(ctx context.Context, f domain.PageFilter) ([]*domain.Page, error) {
	q := `
		SELECT id::text, run_id::text, url, final_url, status, depth, rendered, title, content_type, duration_ms, error, scanned_at
		  FROM run_pages WHERE 1=1`
	args := []any{}
	if f.RunID != "" {
		args = append(args, toUUID(f.RunID))
		q += fmt.Sprintf(` AND run_id = $%d`, len(args))
	}
	if f.FailedOnly {
		q += ` AND ((error IS NOT NULL AND error <> '') OR status = 0 OR status >= 400)`
	}
	q += ` ORDER BY url`
	if f.Limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, f.Limit)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Page{}
	for rows.Next() {
		var (
			pgid                   string
			rid                    string
			finalURL, title, ctype *string
			errText                *string
			p                      domain.Page
		)
		if err := rows.Scan(&pgid, &rid, &p.URL, &finalURL, &p.Status, &p.Depth, &p.Rendered, &title, &ctype, &p.DurationMs, &errText, &p.ScannedAt); err != nil {
			return nil, err
		}
		p.ID = toID(pgid)
		p.RunID = toID(rid)
		assignStr(&p.FinalURL, finalURL)
		assignStr(&p.Title, title)
		assignStr(&p.ContentType, ctype)
		assignStr(&p.Error, errText)
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ---- scan results (per-page checkpoints) --------------------------------

func (s *Store) SaveScanResult(ctx context.Context, r *domain.ScanResult) error {
	raw, err := marshal(r)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO scan_results (run_id, url, result, created_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (run_id, url) DO UPDATE SET result = EXCLUDED.result, created_at = now()`,
		toUUID(r.RunID), r.URL, raw)
	return err
}

func (s *Store) GetScanResult(ctx context.Context, runID domain.ID, url string) (*domain.ScanResult, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `
		SELECT result FROM scan_results WHERE run_id = $1 AND url = $2`, toUUID(runID), url).Scan(&raw)
	if err != nil {
		return nil, notFound(err)
	}
	var r domain.ScanResult
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) ListScanResults(ctx context.Context, runID domain.ID, limit int) ([]*domain.ScanResult, error) {
	q := `SELECT result FROM scan_results WHERE run_id = $1 ORDER BY url`
	if limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, limit)
	}
	rows, err := s.pool.Query(ctx, q, toUUID(runID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ScanResult{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var r domain.ScanResult
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// ---- findings ------------------------------------------------------------

func (s *Store) SaveFinding(ctx context.Context, f *domain.Finding) error {
	ev, err := marshal(f.Evidence)
	if err != nil {
		return err
	}
	pageID := any(nil)
	if !f.PageID.IsEmpty() {
		pageID = toUUID(f.PageID)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO findings (id, run_id, page_id, page_url, plugin_id, rule_id, severity, title, description, evidence, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO NOTHING`,
		toUUID(f.ID), toUUID(f.RunID), pageID, f.PageURL, f.PluginID, f.RuleID,
		string(f.Severity), f.Title, nullString(f.Description), ev, f.CreatedAt)
	return err
}

func (s *Store) ListFindings(ctx context.Context, f domain.FindingFilter) ([]*domain.Finding, error) {
	q := `
		SELECT id::text, run_id::text, COALESCE(page_id::text, ''), COALESCE(page_url, ''),
		       plugin_id, rule_id, severity, title, description, evidence, created_at
		  FROM findings WHERE 1=1`
	args := []any{}
	if f.RunID != "" {
		args = append(args, toUUID(f.RunID))
		q += fmt.Sprintf(` AND run_id = $%d`, len(args))
	}
	if f.MinSeverity != "" {
		args = append(args, string(f.MinSeverity))
		rank := fmt.Sprintf(severityRankSQL, "severity")
		minRank := fmt.Sprintf(severityRankSQL, fmt.Sprintf("$%d", len(args)))
		q += fmt.Sprintf(` AND %s >= %s`, rank, minRank)
	}
	if f.RuleID != "" {
		args = append(args, f.RuleID)
		q += fmt.Sprintf(` AND rule_id = $%d`, len(args))
	}
	if f.PluginID != "" {
		args = append(args, f.PluginID)
		q += fmt.Sprintf(` AND plugin_id = $%d`, len(args))
	}
	q += ` ORDER BY ` + fmt.Sprintf(severityRankSQL, "severity") + ` DESC, created_at`
	if f.Limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, f.Limit)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Finding{}
	for rows.Next() {
		var (
			fid, rid, pid string
			ev            []byte
			desc          *string
			fo            domain.Finding
		)
		if err := rows.Scan(&fid, &rid, &pid, &fo.PageURL, &fo.PluginID, &fo.RuleID,
			&fo.Severity, &fo.Title, &desc, &ev, &fo.CreatedAt); err != nil {
			return nil, err
		}
		fo.ID = toID(fid)
		fo.RunID = toID(rid)
		if pid != "" {
			fo.PageID = toID(pid)
		}
		assignStr(&fo.Description, desc)
		unmarshalInto(ev, &fo.Evidence)
		out = append(out, &fo)
	}
	return out, rows.Err()
}

// ---- audit rules ---------------------------------------------------------

func (s *Store) CreateRule(ctx context.Context, r *domain.AuditRule) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_rules (id, project_id, name, vendor, enabled, url_pattern, target,
		                         parameter, event_name, operator, expected, severity, description,
		                         created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		toUUID(r.ID), toUUID(r.ProjectID), r.Name, r.Vendor, r.Enabled, r.URLPattern,
		string(r.Target), r.Parameter, r.Event, string(r.Operator), r.Expected,
		string(r.Severity), r.Description, r.CreatedAt, r.UpdatedAt)
	return err
}

func (s *Store) GetRule(ctx context.Context, id domain.ID) (*domain.AuditRule, error) {
	return s.scanRule(s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, name, vendor, enabled, url_pattern, target,
		       parameter, event_name, operator, expected, severity, description,
		       created_at, updated_at
		  FROM audit_rules WHERE id = $1`, toUUID(id)))
}

func (s *Store) UpdateRule(ctx context.Context, r *domain.AuditRule) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE audit_rules
		   SET name=$2, vendor=$3, enabled=$4, url_pattern=$5, target=$6, parameter=$7,
		       event_name=$8, operator=$9, expected=$10, severity=$11, description=$12,
		       updated_at=$13
		 WHERE id=$1`,
		toUUID(r.ID), r.Name, r.Vendor, r.Enabled, r.URLPattern, string(r.Target),
		r.Parameter, r.Event, string(r.Operator), r.Expected, string(r.Severity),
		r.Description, r.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteRule(ctx context.Context, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM audit_rules WHERE id=$1`, toUUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListRules(ctx context.Context, projectID domain.ID) ([]*domain.AuditRule, error) {
	q := `SELECT id::text, project_id::text, name, vendor, enabled, url_pattern, target,
	             parameter, event_name, operator, expected, severity, description,
	             created_at, updated_at
	        FROM audit_rules WHERE 1=1`
	args := []any{}
	if projectID != "" {
		args = append(args, toUUID(projectID))
		q += ` AND project_id = $1`
	}
	q += ` ORDER BY created_at, name`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.AuditRule{}
	for rows.Next() {
		r, err := s.scanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *Store) scanRule(row rowScanner) (*domain.AuditRule, error) {
	var (
		id, projectID, target, operator, severity string
		r                                         domain.AuditRule
	)
	if err := row.Scan(&id, &projectID, &r.Name, &r.Vendor, &r.Enabled, &r.URLPattern,
		&target, &r.Parameter, &r.Event, &operator, &r.Expected, &severity,
		&r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	r.ID = toID(id)
	r.ProjectID = toID(projectID)
	r.Target = domain.RuleTarget(target)
	r.Operator = domain.RuleOperator(operator)
	r.Severity = domain.Severity(severity)
	return &r, nil
}

// ---- events --------------------------------------------------------------

func (s *Store) AppendEvent(ctx context.Context, e *domain.Event) error {
	payload, err := marshal(e.Payload)
	if err != nil {
		return err
	}
	at := e.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
		INSERT INTO run_events (run_id, type, at, payload)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		toUUID(e.RunID), string(e.Type), at, payload).Scan(&id)
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

func (s *Store) ListEvents(ctx context.Context, runID domain.ID, afterID int64) ([]*domain.Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, run_id::text, type, at, payload
		  FROM run_events WHERE run_id = $1 AND id > $2 ORDER BY id`,
		toUUID(runID), afterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Event{}
	for rows.Next() {
		var (
			rid     string
			payload []byte
			e       domain.Event
		)
		if err := rows.Scan(&e.ID, &rid, &e.Type, &e.At, &payload); err != nil {
			return nil, err
		}
		e.RunID = toID(rid)
		unmarshalInto(payload, &e.Payload)
		out = append(out, &e)
	}
	return out, rows.Err()
}

// ---- small utilities -----------------------------------------------------

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func assignStr(dst *string, src *string) {
	if src != nil {
		*dst = *src
	}
}
