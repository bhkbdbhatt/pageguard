package domain

import (
	"encoding/json"
	"time"
)

// NetworkRequest is one HTTP exchange observed while loading a page. It is the
// primary evidence source for tag, cookie and privacy validation.
type NetworkRequest struct {
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Host            string            `json:"host"`
	Status          int               `json:"status"`
	ResourceType    string            `json:"resourceType"` // document|xhr|fetch|script|image|...
	Initiator       string            `json:"initiator,omitempty"`
	RequestHeaders  map[string]string `json:"requestHeaders,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	StartedAtMs     int64             `json:"startedAtMs"`
	DurationMs      int64             `json:"durationMs"`
	SizeBytes       int64             `json:"sizeBytes"`
	ThirdParty      bool              `json:"thirdParty"`
}

// Cookie is a cookie observed in the browser context or response headers.
type Cookie struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	ValueHash   string `json:"valueHash"` // hash only; raw values are never stored
	Secure      bool   `json:"secure"`
	HTTPOnly    bool   `json:"httpOnly"`
	SameSite    string `json:"sameSite"`
	ExpiresUnix int64  `json:"expiresUnix,omitempty"`
	Session     bool   `json:"session"`
	FirstParty  bool   `json:"firstParty"`
}

// Tag is a detected martech tag, pixel or library.
type Tag struct {
	ID         string      `json:"id"`
	Vendor     string      `json:"vendor"`
	Name       string      `json:"name"`
	Category   TagCategory `json:"category"`
	Version    string      `json:"version,omitempty"`
	Evidence   string      `json:"evidence"`
	Confidence float64     `json:"confidence"` // 0..1
}

// Finding is a single rule violation or observation produced by a validator.
type Finding struct {
	ID          ID             `json:"id"`
	RunID       ID             `json:"runId"`
	PageID      ID             `json:"pageId,omitempty"`
	PageURL     string         `json:"pageUrl"`
	PluginID    string         `json:"pluginId"`
	RuleID      string         `json:"ruleId"`
	Severity    Severity       `json:"severity"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Evidence    map[string]any `json:"evidence,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
}

// ConsoleMessage captures a browser console entry (browser-rendered scans).
type ConsoleMessage struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// Observation is the immutable snapshot handed to validators. It is the
// contract between the scanner (producer) and the plugin host (consumer).
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
	Tags          []Tag             `json:"tags"`
	Cookies       []Cookie          `json:"cookies"`
	Console       []ConsoleMessage  `json:"console,omitempty"`
	// Geolocation is the logical region the page was observed from.
	Geolocation string `json:"geolocation,omitempty"`
	// DataLayer holds captured client-side data layer payloads.
	DataLayer []DataLayerEntry `json:"dataLayer,omitempty"`
	// HAR is the raw network archive (browser-rendered observations only).
	HAR        json.RawMessage `json:"har,omitempty"`
	DurationMs int64           `json:"durationMs"`
	Rendered   bool            `json:"rendered"`
	Depth      int             `json:"depth"`
	PageID     ID              `json:"pageId"`
}

// Event is a persisted progress/telemetry record for a run.
type Event struct {
	ID      int64          `json:"id"`
	RunID   ID             `json:"runId"`
	Type    EventType      `json:"type"`
	At      time.Time      `json:"at"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Progress is the derived, always-consistent view of a run's state.
type Progress struct {
	RunID      ID         `json:"runId"`
	Status     RunStatus  `json:"status"`
	Stats      RunStats   `json:"stats"`
	Percent    float64    `json:"percent"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}
