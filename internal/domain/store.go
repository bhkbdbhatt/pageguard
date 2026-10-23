package domain

import (
	"context"
	"errors"
)

// ErrNotFound is returned by stores when a record does not exist.
var ErrNotFound = errors.New("domain: not found")

// ErrConflict is returned when an operation clashes with current state, such as
// resuming a run that is already in flight.
var ErrConflict = errors.New("domain: conflict")

// PageFilter narrows ListPages.
type PageFilter struct {
	RunID      ID
	FailedOnly bool
	Limit      int
}

// FindingFilter narrows ListFindings.
type FindingFilter struct {
	RunID       ID
	MinSeverity Severity
	RuleID      string
	PluginID    string
	Limit       int
}

// Store is the persistence contract used by the orchestrator and API.
// Implementations: internal/store/memory (tests) and internal/store/postgres.
type Store interface {
	CreateProject(ctx context.Context, p *Project) error
	GetProject(ctx context.Context, id ID) (*Project, error)
	ListProjects(ctx context.Context) ([]*Project, error)

	CreateScan(ctx context.Context, s *Scan) error
	UpdateScan(ctx context.Context, s *Scan) error
	GetScan(ctx context.Context, id ID) (*Scan, error)
	ListScans(ctx context.Context, projectID ID) ([]*Scan, error)

	CreateRun(ctx context.Context, r *Run) error
	UpdateRun(ctx context.Context, r *Run) error
	GetRun(ctx context.Context, id ID) (*Run, error)
	ListRuns(ctx context.Context, scanID ID, limit int) ([]*Run, error)

	SavePage(ctx context.Context, p *Page) error
	GetPage(ctx context.Context, id ID) (*Page, error)
	ListPages(ctx context.Context, f PageFilter) ([]*Page, error)

	// Scan results are the per-page checkpoints that make runs resumable and
	// idempotent. They are keyed by (runID, url).
	SaveScanResult(ctx context.Context, r *ScanResult) error
	GetScanResult(ctx context.Context, runID ID, url string) (*ScanResult, error)
	ListScanResults(ctx context.Context, runID ID, limit int) ([]*ScanResult, error)

	SaveFinding(ctx context.Context, f *Finding) error
	ListFindings(ctx context.Context, f FindingFilter) ([]*Finding, error)

	// Audit rules are project-scoped, declarative tag-governance checks.
	CreateRule(ctx context.Context, r *AuditRule) error
	GetRule(ctx context.Context, id ID) (*AuditRule, error)
	UpdateRule(ctx context.Context, r *AuditRule) error
	DeleteRule(ctx context.Context, id ID) error
	ListRules(ctx context.Context, projectID ID) ([]*AuditRule, error)

	AppendEvent(ctx context.Context, e *Event) error
	ListEvents(ctx context.Context, runID ID, afterID int64) ([]*Event, error)
}
