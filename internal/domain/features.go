package domain

import (
	"encoding/json"
	"time"
)

// Schedule is a cron-driven scan definition owned by a project. It keeps the
// scan knobs inline so a schedule can be created before a Scan exists.
type Schedule struct {
	ID        ID         `json:"id"`
	ProjectID ID         `json:"projectId"`
	ScanID    ID         `json:"scanId,omitempty"`
	Name      string     `json:"name"`
	URL       string     `json:"url"`
	Cron      string     `json:"cron"`
	Timezone  string     `json:"timezone"`
	Enabled   bool       `json:"enabled"`
	MaxPages  int        `json:"maxPages"`
	MaxDepth  int        `json:"maxDepth"`
	Consent   string     `json:"consent"`
	Owner     string     `json:"owner"`

	LastRunID  ID         `json:"lastRunId,omitempty"`
	LastStatus RunStatus  `json:"lastStatus,omitempty"`
	LastRunAt  *time.Time `json:"lastRunAt,omitempty"`
	NextRunAt  *time.Time `json:"nextRunAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AlertChannel is a concrete notification destination. SecretRef names the
// environment variable that holds the secret (webhook URL, Slack webhook or
// PagerDuty routing key). Resolved secrets are never returned by the API.
type AlertChannel struct {
	ID             ID         `json:"id"`
	ProjectID      ID         `json:"projectId"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind"` // slack|webhook|email|pagerduty
	Target         string     `json:"target"`
	SecretRef      string     `json:"secretRef,omitempty"`
	Connected      bool       `json:"connected"`
	LastDeliveryAt *time.Time `json:"lastDeliveryAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// IncidentStatus is the triage state of an alert incident.
type IncidentStatus string

const (
	IncidentOpen         IncidentStatus = "open"
	IncidentAcknowledged IncidentStatus = "acknowledged"
	IncidentResolved     IncidentStatus = "resolved"
)

// AlertIncident is a persisted alert event with triage state.
type AlertIncident struct {
	ID             ID             `json:"id"`
	ProjectID      ID             `json:"projectId"`
	RuleID         ID             `json:"ruleId,omitempty"`
	RunID          ID             `json:"runId,omitempty"`
	Severity       Severity       `json:"severity"`
	Title          string         `json:"title"`
	Message        string         `json:"message"`
	RuleName       string         `json:"ruleName"`
	Channel        string         `json:"channel"`
	Status         IncidentStatus `json:"status"`
	AcknowledgedAt *time.Time     `json:"acknowledgedAt,omitempty"`
	AcknowledgedBy string         `json:"acknowledgedBy,omitempty"`
	ResolvedAt     *time.Time     `json:"resolvedAt,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// AlertDelivery records one attempt to notify a channel.
type AlertDelivery struct {
	ID          ID         `json:"id"`
	IncidentID  ID         `json:"incidentId"`
	ChannelID   ID         `json:"channelId"`
	Status      string     `json:"status"` // pending|delivered|failed
	Attempt     int        `json:"attempt"`
	Error       string     `json:"error,omitempty"`
	AttemptedAt time.Time  `json:"attemptedAt"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
}

// JourneyRun is one execution of a persisted journey definition.
type JourneyRun struct {
	ID          ID              `json:"id"`
	JourneyID   ID              `json:"journeyId"`
	ProjectID   ID              `json:"projectId"`
	Status      string          `json:"status"` // running|passed|failed|canceled
	StepsPassed int             `json:"stepsPassed"`
	StepsTotal  int             `json:"stepsTotal"`
	DurationMs  int64           `json:"durationMs"`
	Error       string          `json:"error,omitempty"`
	Report      json.RawMessage `json:"report,omitempty"`
	StartedAt   time.Time       `json:"startedAt"`
	FinishedAt  *time.Time      `json:"finishedAt,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
}

// PrivacyProfile is a named privacy policy evaluated over a set of regulations.
type PrivacyProfile struct {
	ID            ID              `json:"id"`
	ProjectID     ID              `json:"projectId"`
	Name          string          `json:"name"`
	Region        string          `json:"region"`
	RegulationIDs []string        `json:"regulationIds"`
	Config        json.RawMessage `json:"config,omitempty"`
	Active        bool            `json:"active"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

// PrivacyBatch is a durable, resumable page-crawl + evaluation job.
type PrivacyBatch struct {
	ID             ID         `json:"id"`
	ProjectID      ID         `json:"projectId"`
	ProfileID      ID         `json:"profileId,omitempty"`
	StartURL       string     `json:"startUrl"`
	Status         RunStatus  `json:"status"`
	TotalPages     int        `json:"totalPages"`
	CompletedPages int        `json:"completedPages"`
	FailedPages    int        `json:"failedPages"`
	Concurrency    int        `json:"concurrency"`
	Error          string     `json:"error,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
}

// PrivacyEvaluation is one page's queued/running/finished evaluation.
type PrivacyEvaluation struct {
	ID         ID              `json:"id"`
	BatchID    ID              `json:"batchId"`
	ProjectID  ID              `json:"projectId"`
	ProfileID  ID              `json:"profileId,omitempty"`
	URL        string          `json:"url"`
	Status     RunStatus       `json:"status"`
	Report     json.RawMessage `json:"report,omitempty"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	StartedAt  *time.Time      `json:"startedAt,omitempty"`
	FinishedAt *time.Time      `json:"finishedAt,omitempty"`
}

// ReportScope is a structured, machine-readable report selection.
type ReportScope struct {
	Type  string `json:"type"`
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
}

// Report is a generated, downloadable artefact.
type Report struct {
	ID          ID          `json:"id"`
	ProjectID   ID          `json:"projectId"`
	TemplateID  string      `json:"templateId"`
	Name        string      `json:"name"`
	Scope       ReportScope `json:"scope"`
	Format      string      `json:"format"` // json|csv|html
	Status      string      `json:"status"` // ready|generating|failed
	Owner       string      `json:"owner"`
	ContentType string      `json:"contentType,omitempty"`
	Content     []byte      `json:"-"`
	SizeBytes   int64       `json:"sizeBytes"`
	Error       string      `json:"error,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// ReportSchedule is a cron-driven report generation job.
type ReportSchedule struct {
	ID          ID          `json:"id"`
	ProjectID   ID          `json:"projectId"`
	TemplateID  string      `json:"templateId"`
	Name        string      `json:"name"`
	Cadence     string      `json:"cadence"`
	Cron        string      `json:"cron"`
	Timezone    string      `json:"timezone"`
	Recipients  []string    `json:"recipients"`
	Scope       ReportScope `json:"scope"`
	Format      string      `json:"format"`
	Enabled     bool        `json:"enabled"`
	NextRunAt   *time.Time  `json:"nextRunAt,omitempty"`
	LastRunAt   *time.Time  `json:"lastRunAt,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// InventoryTag is an aggregated tag observed across a run.
type InventoryTag struct {
	Tag
	Purpose   string    `json:"purpose"`
	Pages     int       `json:"pages"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}

// InventoryCookie is an aggregated cookie observed across a run.
type InventoryCookie struct {
	Cookie
	Vendor  string `json:"vendor"`
	Purpose string `json:"purpose"`
	Pages   int    `json:"pages"`
}

// InventoryRequest is an aggregated network request observed across a run.
type InventoryRequest struct {
	NetworkRequest
	Vendor string `json:"vendor"`
	Count  int    `json:"count"`
	Pages  int    `json:"pages"`
}

// InventorySnapshot is the project-level result of aggregating scan_results.
type InventorySnapshot struct {
	ProjectID    ID                 `json:"projectId"`
	RunID        ID                 `json:"runId,omitempty"`
	ScanID       ID                 `json:"scanId,omitempty"`
	PagesScanned int                `json:"pagesScanned"`
	Tags         []InventoryTag     `json:"tags"`
	Cookies      []InventoryCookie  `json:"cookies"`
	Requests     []InventoryRequest `json:"requests"`
	GeneratedAt  time.Time          `json:"generatedAt"`
}
