package domain

import (
	"context"
	"errors"
	"time"
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

	// Schedules are cron-driven scan definitions.
	CreateSchedule(ctx context.Context, s *Schedule) error
	GetSchedule(ctx context.Context, id ID) (*Schedule, error)
	UpdateSchedule(ctx context.Context, s *Schedule) error
	DeleteSchedule(ctx context.Context, id ID) error
	ListSchedules(ctx context.Context, projectID ID) ([]*Schedule, error)
	ListDueSchedules(ctx context.Context, now time.Time, limit int) ([]*Schedule, error)

	// Alert rules, channels, incidents and delivery attempts.
	CreateAlertRule(ctx context.Context, r *AlertRule) error
	GetAlertRule(ctx context.Context, id ID) (*AlertRule, error)
	UpdateAlertRule(ctx context.Context, r *AlertRule) error
	DeleteAlertRule(ctx context.Context, id ID) error
	ListAlertRules(ctx context.Context, projectID ID) ([]*AlertRule, error)

	CreateAlertChannel(ctx context.Context, c *AlertChannel) error
	GetAlertChannel(ctx context.Context, id ID) (*AlertChannel, error)
	UpdateAlertChannel(ctx context.Context, c *AlertChannel) error
	DeleteAlertChannel(ctx context.Context, id ID) error
	ListAlertChannels(ctx context.Context, projectID ID) ([]*AlertChannel, error)

	CreateAlertIncident(ctx context.Context, i *AlertIncident) error
	GetAlertIncident(ctx context.Context, id ID) (*AlertIncident, error)
	UpdateAlertIncident(ctx context.Context, i *AlertIncident) error
	ListAlertIncidents(ctx context.Context, projectID ID, limit int) ([]*AlertIncident, error)

	CreateAlertDelivery(ctx context.Context, d *AlertDelivery) error
	ListAlertDeliveries(ctx context.Context, incidentID ID) ([]*AlertDelivery, error)

	// Journey definitions and runs.
	CreateJourney(ctx context.Context, j *Journey) error
	GetJourney(ctx context.Context, id ID) (*Journey, error)
	UpdateJourney(ctx context.Context, j *Journey) error
	DeleteJourney(ctx context.Context, id ID) error
	ListJourneys(ctx context.Context, projectID ID) ([]*Journey, error)

	CreateJourneyRun(ctx context.Context, r *JourneyRun) error
	GetJourneyRun(ctx context.Context, id ID) (*JourneyRun, error)
	UpdateJourneyRun(ctx context.Context, r *JourneyRun) error
	ListJourneyRuns(ctx context.Context, journeyID ID, limit int) ([]*JourneyRun, error)

	// Privacy profiles, batches and per-page evaluations.
	CreatePrivacyProfile(ctx context.Context, p *PrivacyProfile) error
	GetPrivacyProfile(ctx context.Context, id ID) (*PrivacyProfile, error)
	UpdatePrivacyProfile(ctx context.Context, p *PrivacyProfile) error
	DeletePrivacyProfile(ctx context.Context, id ID) error
	ListPrivacyProfiles(ctx context.Context, projectID ID) ([]*PrivacyProfile, error)

	CreatePrivacyBatch(ctx context.Context, b *PrivacyBatch) error
	GetPrivacyBatch(ctx context.Context, id ID) (*PrivacyBatch, error)
	UpdatePrivacyBatch(ctx context.Context, b *PrivacyBatch) error
	ListPrivacyBatches(ctx context.Context, projectID ID, limit int) ([]*PrivacyBatch, error)
	ListRunnablePrivacyBatches(ctx context.Context, limit int) ([]*PrivacyBatch, error)

	CreatePrivacyEvaluation(ctx context.Context, e *PrivacyEvaluation) error
	GetPrivacyEvaluation(ctx context.Context, id ID) (*PrivacyEvaluation, error)
	UpdatePrivacyEvaluation(ctx context.Context, e *PrivacyEvaluation) error
	ListPrivacyEvaluations(ctx context.Context, batchID ID, limit int) ([]*PrivacyEvaluation, error)
	ListQueuedPrivacyEvaluations(ctx context.Context, batchID ID, limit int) ([]*PrivacyEvaluation, error)

	// Generated reports and report schedules.
	CreateReport(ctx context.Context, r *Report) error
	GetReport(ctx context.Context, id ID) (*Report, error)
	UpdateReport(ctx context.Context, r *Report) error
	DeleteReport(ctx context.Context, id ID) error
	ListReports(ctx context.Context, projectID ID, limit int) ([]*Report, error)

	CreateReportSchedule(ctx context.Context, s *ReportSchedule) error
	GetReportSchedule(ctx context.Context, id ID) (*ReportSchedule, error)
	UpdateReportSchedule(ctx context.Context, s *ReportSchedule) error
	DeleteReportSchedule(ctx context.Context, id ID) error
	ListReportSchedules(ctx context.Context, projectID ID) ([]*ReportSchedule, error)
	ListDueReportSchedules(ctx context.Context, now time.Time, limit int) ([]*ReportSchedule, error)
}
