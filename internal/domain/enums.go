package domain

// RenderMode selects how a page is materialised during a scan.
type RenderMode string

const (
	// RenderHTTP fetches the raw HTTP response only (fast, no JavaScript).
	RenderHTTP RenderMode = "http"
	// RenderBrowser drives a headless browser so client-side tags execute.
	RenderBrowser RenderMode = "browser"
	// RenderAuto tries HTTP first and falls back to the browser when the
	// document looks like an SPA (little static content / app mount point).
	RenderAuto RenderMode = "auto"
)

// RunStatus is the lifecycle state of a scan run.
type RunStatus string

const (
	RunQueued    RunStatus = "queued"
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
	RunCanceled  RunStatus = "canceled"
)

// Terminal reports whether the status is final.
func (s RunStatus) Terminal() bool {
	switch s {
	case RunSucceeded, RunFailed, RunCanceled:
		return true
	}
	return false
}

// Severity ranks how serious a finding is.
type Severity string

const (
	SevInfo     Severity = "info"
	SevLow      Severity = "low"
	SevMedium   Severity = "medium"
	SevHigh     Severity = "high"
	SevCritical Severity = "critical"
)

// weight is used to sort findings by importance.
func (s Severity) weight() int {
	switch s {
	case SevCritical:
		return 5
	case SevHigh:
		return 4
	case SevMedium:
		return 3
	case SevLow:
		return 2
	case SevInfo:
		return 1
	}
	return 0
}

// Less reports whether s is more severe than o.
func (s Severity) Less(o Severity) bool { return s.weight() < o.weight() }

// TagCategory classifies a detected tag/vendor.
type TagCategory string

const (
	TagAnalytics   TagCategory = "analytics"
	TagAdvertising TagCategory = "advertising"
	TagSocial      TagCategory = "social"
	TagTagManager  TagCategory = "tag_manager"
	TagCDP         TagCategory = "cdp"
	TagConsent     TagCategory = "consent"
	TagMonitoring  TagCategory = "monitoring"
	TagUnknown     TagCategory = "unknown"
)

// EventType enumerates run telemetry emitted over the event stream.
type EventType string

const (
	EvtRunQueued      EventType = "run.queued"
	EvtRunStarted     EventType = "run.started"
	EvtPageDiscovered EventType = "page.discovered"
	EvtPageStarted    EventType = "page.started"
	EvtPageCompleted  EventType = "page.completed"
	EvtPageResumed    EventType = "page.resumed"
	EvtPageFailed     EventType = "page.failed"
	EvtFinding        EventType = "finding.created"
	EvtRunCompleted   EventType = "run.completed"
	EvtRunFailed      EventType = "run.failed"
	EvtLog            EventType = "log"
)
