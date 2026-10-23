package domain

import (
	"encoding/json"
	"time"
)

// ScanResultSchemaVersion identifies the ScanResult JSON schema. Consumers
// should treat a change of this value as a breaking schema change.
const ScanResultSchemaVersion = "1.0.0"

// DataLayerEntry is a client-side data layer (e.g. Google Tag Manager's
// window.dataLayer, Adobe's digitalData) captured from the browser. Value is the
// raw JSON of the variable, so consumers can interpret it without a fixed model.
type DataLayerEntry struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value,omitempty"`
}

// ScanResult is the structured, per-page evidence record emitted by the
// orchestrator. It is the canonical JSON consumers parse, and it doubles as the
// per-page checkpoint: once written, a run can be resumed without re-scanning
// that page (idempotency).
//
// A ScanResult is keyed by (RunID, URL).
type ScanResult struct {
	SchemaVersion string `json:"schemaVersion"`

	RunID  ID `json:"runId"`
	ScanID ID `json:"scanId"`
	PageID ID `json:"pageId"`

	URL      string `json:"url"`
	FinalURL string `json:"finalUrl"`
	Depth    int    `json:"depth"`

	// Attempt is 1 on the first scan; Resumed marks a result reconstructed from
	// a checkpoint rather than a fresh scan.
	Attempt int  `json:"attempt"`
	Resumed bool `json:"resumed"`

	StatusCode  int        `json:"statusCode"`
	ContentType string     `json:"contentType,omitempty"`
	Title       string     `json:"title,omitempty"`
	Rendered    bool       `json:"rendered"`
	RenderMode  RenderMode `json:"renderMode"`

	// Geolocation is the logical region the page was observed from (cross-border
	// evidence). ProxyHost is the proxy used, with credentials stripped.
	Geolocation string `json:"geolocation,omitempty"`
	ProxyHost   string `json:"proxyHost,omitempty"`

	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	DurationMs int64     `json:"durationMs"`

	// DOM is the serialized document (post-JavaScript when browser-rendered).
	DOM          string `json:"dom,omitempty"`
	DOMTruncated bool   `json:"domTruncated"`
	DOMBytes     int    `json:"domBytes"`

	Requests  []NetworkRequest `json:"requests"`
	Cookies   []Cookie         `json:"cookies"`
	Console   []ConsoleMessage `json:"console"`
	DataLayer []DataLayerEntry `json:"dataLayer"`
	HAR       json.RawMessage  `json:"har,omitempty"`

	Tags            []Tag    `json:"tags"`
	Links           []string `json:"links"`
	MetaDescription string   `json:"metaDescription,omitempty"`

	// Error is set when the page could not be scanned. A ScanResult is still
	// checkpointed so a resumed run does not retry a deterministically failing
	// URL forever (callers can force a rescan).
	Error string `json:"error,omitempty"`
}
