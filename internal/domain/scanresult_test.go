package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func fullScanResult() *ScanResult {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return &ScanResult{
		SchemaVersion: ScanResultSchemaVersion,
		RunID:         NewID(),
		ScanID:        NewID(),
		PageID:        NewID(),
		URL:           "https://example.com/",
		FinalURL:      "https://example.com/",
		Depth:         1,
		Attempt:       1,
		Resumed:       false,
		StatusCode:    200,
		ContentType:   "text/html",
		Title:         "Example",
		Rendered:      true,
		RenderMode:    RenderBrowser,
		Geolocation:   "de",
		ProxyHost:     "proxy.de:8080",
		StartedAt:     now,
		FinishedAt:    now.Add(2 * time.Second),
		DurationMs:    2000,
		DOM:           "<html><head><title>Example</title></head></html>",
		DOMTruncated:  false,
		DOMBytes:      48,
		Requests: []NetworkRequest{{
			Method: "GET", URL: "https://example.com/", Host: "example.com",
			Status: 200, ResourceType: "document", StartedAtMs: 1, DurationMs: 20, SizeBytes: 512,
		}},
		Cookies: []Cookie{{
			Name: "_ga", Domain: ".example.com", Path: "/", ValueHash: "0123456789abcdef",
			Secure: true, HTTPOnly: false, SameSite: "lax", ExpiresUnix: 1893456000,
			Session: false, FirstParty: true,
		}},
		Console:   []ConsoleMessage{{Level: "error", Text: "boom"}},
		DataLayer: []DataLayerEntry{{Name: "dataLayer[0]", Value: json.RawMessage(`{"event":"view"}`)}},
		HAR:       json.RawMessage(`{"log":{"version":"1.2","entries":[]}}`),
		Tags: []Tag{{
			ID: "ga4", Vendor: "Google", Name: "Google Analytics 4", Category: TagAnalytics,
			Version: "1", Evidence: "gtag/js", Confidence: 0.9,
		}},
		Links:           []string{"https://example.com/a"},
		MetaDescription: "an example",
		Error:           "",
	}
}

func TestScanResultJSONRoundTrip(t *testing.T) {
	sr := fullScanResult()
	b, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back ScanResult
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(sr, &back) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", &back, sr)
	}
}

// TestScanResultMatchesSchema keeps the Go struct and the published JSON schema
// in lockstep: every schema "required" field must be emitted by a fully
// populated ScanResult, and the struct must not emit undeclared properties.
func TestScanResultMatchesSchema(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "schemas", "scanresult.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	b, err := json.Marshal(fullScanResult())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	for _, req := range schema.Required {
		if _, ok := obj[req]; !ok {
			t.Errorf("schema requires %q but ScanResult does not emit it", req)
		}
	}
	for k := range obj {
		if _, ok := schema.Properties[k]; !ok {
			t.Errorf("ScanResult emits %q which the schema does not declare", k)
		}
	}
}
