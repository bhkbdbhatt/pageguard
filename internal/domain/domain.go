package domain

import (
	"encoding/json"
	"time"
)

// Project is the top-level tenant boundary. Scans, journeys and alert rules
// all belong to exactly one project.
type Project struct {
	ID        ID        `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Viewport describes the emulated browser window for browser-rendered scans.
type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ScanConfig captures every tunable that affects crawling and rendering.
// Zero values are interpreted by Normalize().
type ScanConfig struct {
	MaxDepth         int        `json:"maxDepth"`
	MaxPages         int        `json:"maxPages"`
	Concurrency      int        `json:"concurrency"`
	SameOriginOnly   bool       `json:"sameOriginOnly"`
	RespectRobotsTxt bool       `json:"respectRobotsTxt"`
	IncludePatterns  []string   `json:"includePatterns"`
	ExcludePatterns  []string   `json:"excludePatterns"`
	RenderMode       RenderMode `json:"renderMode"`
	RenderWaitMs     int        `json:"renderWaitMs"`
	UserAgent        string     `json:"userAgent"`
	TimeoutMs        int        `json:"timeoutMs"`
	Viewport         Viewport   `json:"viewport"`
	MaxBodyBytes     int64      `json:"maxBodyBytes"`
	// Geolocations lists logical regions to observe pages from. When non-empty,
	// pages are assigned a region in rotation so cross-border data collection
	// can be compared. Each region is paired with a proxy at run time.
	Geolocations []string `json:"geolocations,omitempty"`
	// CaptureHAR records a full browser network archive per page.
	CaptureHAR bool `json:"captureHar,omitempty"`
	// CaptureDataLayer records client-side data layer payloads per page.
	CaptureDataLayer bool `json:"captureDataLayer,omitempty"`
	// ConsentSimulation records the consent state requested for an audit
	// (accept_all, reject_all, necessary_only or pre_consent). The scanner
	// currently captures a pre-consent snapshot; the value is retained as audit
	// metadata and drives the derived consent matrix.
	ConsentSimulation string `json:"consentSimulation,omitempty"`
}

// Normalize fills unset fields with safe defaults. It is idempotent.
func (c *ScanConfig) Normalize() {
	if c.MaxDepth <= 0 {
		c.MaxDepth = 3
	}
	if c.MaxPages <= 0 {
		c.MaxPages = 500
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 4
	}
	if c.RenderMode == "" {
		c.RenderMode = RenderHTTP
	}
	if c.UserAgent == "" {
		c.UserAgent = "PageguardBot/0.1 (+https://github.com/pageguard/pageguard)"
	}
	if c.TimeoutMs <= 0 {
		c.TimeoutMs = 20000
	}
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = 4 << 20 // 4 MiB
	}
	if c.Viewport.Width == 0 {
		c.Viewport.Width = 1366
	}
	if c.Viewport.Height == 0 {
		c.Viewport.Height = 768
	}
	if c.RenderWaitMs < 0 {
		c.RenderWaitMs = 0
	}
}

// Scan is a reusable audit definition: what to crawl and which rules to apply.
type Scan struct {
	ID         ID         `json:"id"`
	ProjectID  ID         `json:"projectId"`
	Name       string     `json:"name"`
	StartURLs  []string   `json:"startUrls"`
	Config     ScanConfig `json:"config"`
	RuleSetIDs []string   `json:"ruleSetIds"`
	Enabled    bool       `json:"enabled"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// RunStats are the aggregate counters surfaced in progress updates.
type RunStats struct {
	PagesDiscovered int `json:"pagesDiscovered"`
	PagesScanned    int `json:"pagesScanned"`
	PagesFailed     int `json:"pagesFailed"`
	Findings        int `json:"findings"`

	// Governance rollups computed when a run finishes.
	Beacons          int                `json:"beacons"`
	CriticalFindings int                `json:"criticalFindings"`
	HealthScore      float64            `json:"healthScore"`
	ConsentScore     float64            `json:"consentScore"`
	VendorScores     map[string]float64 `json:"vendorScores,omitempty"`
}

// Run is a single execution of a Scan.
type Run struct {
	ID         ID         `json:"id"`
	ScanID     ID         `json:"scanId"`
	ProjectID  ID         `json:"projectId"`
	Status     RunStatus  `json:"status"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Stats      RunStats   `json:"stats"`
	Error      string     `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// Page is the result of visiting a single URL within a Run.
type Page struct {
	ID          ID        `json:"id"`
	RunID       ID        `json:"runId"`
	URL         string    `json:"url"`
	FinalURL    string    `json:"finalUrl"`
	Status      int       `json:"status"`
	Depth       int       `json:"depth"`
	Rendered    bool      `json:"rendered"`
	Title       string    `json:"title"`
	ContentType string    `json:"contentType"`
	DurationMs  int64     `json:"durationMs"`
	Error       string    `json:"error,omitempty"`
	ScannedAt   time.Time `json:"scannedAt"`
}

// Journey is a scripted, multi-step user flow used by Module 4.
type Journey struct {
	ID          ID              `json:"id"`
	ProjectID   ID              `json:"projectId"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Owner       string          `json:"owner,omitempty"`
	BaseURL     string          `json:"baseUrl,omitempty"`
	Enabled     bool            `json:"enabled"`
	// Definition is the canonical journey.Journey document.
	Definition json.RawMessage `json:"definition,omitempty"`
	// Steps is retained for legacy records; new writes populate Definition.
	Steps     []JourneyStep `json:"steps,omitempty"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// JourneyStep is one action within a Journey.
type JourneyStep struct {
	Order     int            `json:"order"`
	Action    string         `json:"action"` // navigate|click|type|wait|assert|extract
	Selector  string         `json:"selector,omitempty"`
	Value     string         `json:"value,omitempty"`
	TimeoutMs int            `json:"timeoutMs,omitempty"`
	Expect    map[string]any `json:"expect,omitempty"`
}

// AlertRule describes a condition over findings/run state that triggers events.
type AlertRule struct {
	ID       ID     `json:"id"`
	ProjectID ID    `json:"projectId"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Severity Severity `json:"severity"`
	// Condition is the structured predicate evaluated after a run.
	Condition AlertCondition `json:"condition"`
	// ConditionText is a human-readable summary for the UI.
	ConditionText string   `json:"conditionText,omitempty"`
	// Channels is the legacy list of channel kinds (slack, email, ...).
	Channels []string `json:"channels,omitempty"`
	// ChannelIDs is the authoritative list of configured destinations.
	ChannelIDs []ID `json:"channelIds"`

	LastTriggeredAt *time.Time `json:"lastTriggeredAt,omitempty"`
	TriggerCount    int        `json:"triggerCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AlertCondition is a small, serialisable predicate evaluated after a run.
type AlertCondition struct {
	MinSeverity  Severity `json:"minSeverity,omitempty"`
	RuleIDs      []string `json:"ruleIds,omitempty"`
	OnRunFailure bool     `json:"onRunFailure,omitempty"`
	TagMissing   []string `json:"tagMissing,omitempty"`
}

// AlertEvent is an emitted notification for an AlertRule.
type AlertEvent struct {
	ID        ID             `json:"id"`
	RuleID    ID             `json:"ruleId"`
	RunID     ID             `json:"runId"`
	Message   string         `json:"message"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}
