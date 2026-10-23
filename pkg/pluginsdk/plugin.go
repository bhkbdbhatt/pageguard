package pluginsdk

import "context"

// NetworkRequest mirrors a single HTTP exchange observed on a page.
type NetworkRequest struct {
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Host            string            `json:"host"`
	Status          int               `json:"status"`
	ResourceType    string            `json:"resourceType"`
	Initiator       string            `json:"initiator,omitempty"`
	RequestHeaders  map[string]string `json:"requestHeaders,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	DurationMs      int64             `json:"durationMs"`
	SizeBytes       int64             `json:"sizeBytes"`
	ThirdParty      bool              `json:"thirdParty"`
}

// Cookie mirrors a cookie observed on a page. Raw values are never exposed.
type Cookie struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	ValueHash   string `json:"valueHash"`
	Secure      bool   `json:"secure"`
	HTTPOnly    bool   `json:"httpOnly"`
	SameSite    string `json:"sameSite"`
	ExpiresUnix int64  `json:"expiresUnix,omitempty"`
	Session     bool   `json:"session"`
	FirstParty  bool   `json:"firstParty"`
}

// Tag mirrors a detected martech tag.
type Tag struct {
	ID         string  `json:"id"`
	Vendor     string  `json:"vendor"`
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Version    string  `json:"version,omitempty"`
	Evidence   string  `json:"evidence"`
	Confidence float64 `json:"confidence"`
}

// Observation is the immutable snapshot passed to validators.
type Observation struct {
	URL           string            `json:"url"`
	FinalURL      string            `json:"finalUrl"`
	StatusCode    int               `json:"statusCode"`
	ContentType   string            `json:"contentType"`
	Title         string            `json:"title"`
	Headers       map[string]string `json:"headers"`
	HTML          string            `json:"html"`
	HTMLTruncated bool              `json:"htmlTruncated"`
	Requests      []NetworkRequest  `json:"requests"`
	Cookies       []Cookie          `json:"cookies"`
	Tags          []Tag             `json:"tags"`
	ConsoleErrors []string          `json:"consoleErrors,omitempty"`
	// Geolocation is the logical region the page was observed from.
	Geolocation string `json:"geolocation,omitempty"`
	// DataLayer holds captured client-side data layer payloads.
	DataLayer []DataLayerEntry `json:"dataLayer,omitempty"`
	Rendered  bool             `json:"rendered"`
	Depth     int              `json:"depth"`
	PageID    string           `json:"pageId"`
}

// DataLayerEntry is a captured client-side data layer variable (e.g. a
// window.dataLayer push). Value is the raw JSON of the variable.
type DataLayerEntry struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// Finding is a validator result, prior to the host persisting it.
type Finding struct {
	RuleID      string         `json:"ruleId"`
	Severity    string         `json:"severity"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Evidence    map[string]any `json:"evidence,omitempty"`
}

// Scope carries run identifiers and per-plugin configuration into a validator.
type Scope struct {
	RunID     string         `json:"runId"`
	ProjectID string         `json:"projectId"`
	ScanID    string         `json:"scanId"`
	Config    map[string]any `json:"config,omitempty"`
}

// Validator inspects one Observation and returns zero or more findings.
type Validator interface {
	ID() string
	Validate(ctx context.Context, obs *Observation, scope Scope) ([]Finding, error)
}

// RunSummary is the input to a Reporter.
type RunSummary struct {
	RunID          string         `json:"runId"`
	ProjectID      string         `json:"projectId"`
	ScanID         string         `json:"scanId"`
	Status         string         `json:"status"`
	Pages          int            `json:"pages"`
	Findings       int            `json:"findings"`
	SeverityCounts map[string]int `json:"severityCounts"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// Reporter renders or ships a completed run's summary.
type Reporter interface {
	ID() string
	Report(ctx context.Context, summary RunSummary, scope Scope) error
}

// AlertEvent is delivered to integrations by Module 5.
type AlertEvent struct {
	RuleName string         `json:"ruleName"`
	RunID    string         `json:"runId"`
	Message  string         `json:"message"`
	Severity string         `json:"severity"`
	Payload  map[string]any `json:"payload,omitempty"`
}

// Integration delivers alert events to an external system.
type Integration interface {
	ID() string
	Send(ctx context.Context, event AlertEvent, scope Scope) error
}
