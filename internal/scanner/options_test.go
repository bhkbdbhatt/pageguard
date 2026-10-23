package scanner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

type captureRenderer struct {
	opts []RenderOptions
	res  *RenderResult
}

func (c *captureRenderer) Render(_ context.Context, rawURL string, o RenderOptions) (*RenderResult, error) {
	c.opts = append(c.opts, o)
	if c.res != nil {
		r := *c.res
		return &r, nil
	}
	return &RenderResult{FinalURL: rawURL, Status: 200, HTML: "<html></html>", Title: "t"}, nil
}

func (c *captureRenderer) Close() error { return nil }

func TestScanPagePropagatesGeoProxyAndCaptures(t *testing.T) {
	cr := &captureRenderer{res: &RenderResult{
		FinalURL:  "https://x/",
		Status:    200,
		HTML:      "<html><head><title>t</title></head></html>",
		Title:     "t",
		DataLayer: []domain.DataLayerEntry{{Name: "dataLayer[0]", Value: json.RawMessage(`{"event":"view"}`)}},
		HAR:       json.RawMessage(`{"log":{"version":"1.2"}}`),
	}}
	s := New(nil, cr)

	res, err := s.ScanPage(context.Background(), "https://x/", domain.ScanConfig{RenderMode: domain.RenderBrowser}, ScanOptions{
		Depth:            2,
		Geolocation:      "us",
		ProxyServer:      "http://proxy:8080",
		ProxyUser:        "u",
		ProxyPass:        "p",
		CaptureHAR:       true,
		CaptureDataLayer: true,
	})
	if err != nil {
		t.Fatalf("ScanPage: %v", err)
	}

	if len(cr.opts) != 1 {
		t.Fatalf("renderer called %d times", len(cr.opts))
	}
	o := cr.opts[0]
	if o.Geolocation != "us" {
		t.Errorf("geolocation = %q", o.Geolocation)
	}
	if o.GeoLatitude == 0 || o.GeoLongitude == 0 {
		t.Errorf("geo coordinates not resolved from region: %+v", o)
	}
	if o.Locale == "" || o.Timezone == "" {
		t.Errorf("locale/timezone not resolved from region: %+v", o)
	}
	if o.ProxyServer != "http://proxy:8080" || o.ProxyUser != "u" || o.ProxyPass != "p" {
		t.Errorf("proxy not propagated: %+v", o)
	}
	if !o.CaptureHAR || !o.CaptureDataLayer {
		t.Errorf("capture flags not propagated: %+v", o)
	}

	if res.Observation.Geolocation != "us" {
		t.Errorf("observation geolocation = %q", res.Observation.Geolocation)
	}
	if len(res.DataLayer) != 1 || len(res.Observation.DataLayer) != 1 {
		t.Errorf("data layer not propagated: %+v", res.DataLayer)
	}
	if len(res.HAR) == 0 || len(res.Observation.HAR) == 0 {
		t.Errorf("HAR not propagated")
	}
}

func TestScanWrapsScanPageDepth(t *testing.T) {
	ff := &fakeFetcher{responses: map[string]*FetchResult{
		"https://x/": {URL: "https://x/", Status: 200, Headers: map[string][]string{"Content-Type": {"text/html"}}, Body: []byte("<html></html>")},
	}}
	s := New(ff, nil)
	res, err := s.Scan(context.Background(), "https://x/", domain.ScanConfig{}, 3)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Observation.Depth != 3 {
		t.Errorf("depth = %d, want 3", res.Observation.Depth)
	}
}
