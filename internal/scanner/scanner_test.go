package scanner

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

type fakeFetcher struct {
	responses map[string]*FetchResult
	err       error
	calls     []string
}

func (f *fakeFetcher) Fetch(_ context.Context, rawURL string, _ FetchOptions) (*FetchResult, error) {
	f.calls = append(f.calls, rawURL)
	if f.err != nil {
		return nil, f.err
	}
	if r, ok := f.responses[rawURL]; ok {
		return r, nil
	}
	return &FetchResult{URL: rawURL, Status: 404, Headers: http.Header{"Content-Type": {"text/html"}}}, nil
}

func TestScanHTTPExtractsObservationTagsAndCookies(t *testing.T) {
	body := `<html><head>
		<title>Home</title>
		<meta name="description" content="welcome">
		<script src="https://www.googletagmanager.com/gtm.js?id=GTM-X"></script>
	</head><body><a href="/a">a</a></body></html>`

	ff := &fakeFetcher{responses: map[string]*FetchResult{
		"https://example.com/": {
			URL:    "https://example.com/",
			Status: 200,
			Headers: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Set-Cookie":   {"sid=topsecret; Path=/; Secure; HttpOnly; SameSite=Lax"},
			},
			Body: []byte(body),
		},
	}}

	s := New(ff, nil)
	res, err := s.Scan(context.Background(), "https://example.com/", domain.ScanConfig{}, 0)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	obs := res.Observation

	if obs.Title != "Home" {
		t.Errorf("title = %q", obs.Title)
	}
	if obs.StatusCode != 200 || obs.Rendered {
		t.Errorf("status=%d rendered=%v", obs.StatusCode, obs.Rendered)
	}
	if !hasTagID(obs.Tags, "google-tag-manager") {
		t.Errorf("expected google-tag-manager tag, got %+v", obs.Tags)
	}
	if len(obs.Cookies) != 1 {
		t.Fatalf("cookies = %+v", obs.Cookies)
	}
	c := obs.Cookies[0]
	if c.ValueHash == "topsecret" || len(c.ValueHash) != 16 {
		t.Errorf("cookie value must be hashed, got %q", c.ValueHash)
	}
	if !c.Secure || !c.HTTPOnly || c.SameSite != "lax" {
		t.Errorf("cookie flags wrong: %+v", c)
	}
	if !contains(res.Links.Documents, "https://example.com/a") {
		t.Errorf("links = %+v", res.Links.Documents)
	}
}

func TestScanPropagatesFetchError(t *testing.T) {
	s := New(&fakeFetcher{err: errors.New("boom")}, nil)
	if _, err := s.Scan(context.Background(), "https://x/", domain.ScanConfig{}, 0); err == nil {
		t.Fatal("expected error")
	}
}

func TestScanAutoUpgradesSPAToBrowser(t *testing.T) {
	shell := `<html><body><div id="root"></div><script src="/bundle.js"></script></body></html>`
	ff := &fakeFetcher{responses: map[string]*FetchResult{
		"https://spa.example/": {
			URL: "https://spa.example/", Status: 200,
			Headers: http.Header{"Content-Type": {"text/html"}}, Body: []byte(shell),
		},
	}}
	renderer := &FakeRenderer{Responses: map[string]*RenderResult{
		"https://spa.example/": {
			FinalURL: "https://spa.example/", Status: 200, Title: "SPA",
			HTML: `<html><head><title>SPA</title></head><body><div id="root">rendered</div></body></html>`,
			Requests: []domain.NetworkRequest{
				{Method: "GET", URL: "https://spa.example/api/data", Host: "spa.example", Status: 200, ResourceType: "fetch"},
			},
		},
	}}

	s := New(ff, renderer)
	res, err := s.Scan(context.Background(), "https://spa.example/", domain.ScanConfig{RenderMode: domain.RenderAuto}, 0)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !res.Observation.Rendered {
		t.Error("expected browser render upgrade")
	}
	if len(res.Observation.Requests) != 1 || res.Observation.Requests[0].URL != "https://spa.example/api/data" {
		t.Errorf("requests = %+v", res.Observation.Requests)
	}
	if calls := renderer.Calls(); len(calls) != 1 {
		t.Errorf("renderer calls = %v", calls)
	}
}

func TestScanBrowserModeUsesRendererDirectly(t *testing.T) {
	ff := &fakeFetcher{}
	renderer := &FakeRenderer{Responses: map[string]*RenderResult{
		"https://b.example/": {FinalURL: "https://b.example/", Status: 200, Title: "B", HTML: "<html><head><title>B</title></head><body></body></html>"},
	}}
	s := New(ff, renderer)
	res, err := s.Scan(context.Background(), "https://b.example/", domain.ScanConfig{RenderMode: domain.RenderBrowser}, 0)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !res.Observation.Rendered || res.Observation.Title != "B" {
		t.Errorf("obs = %+v", res.Observation)
	}
	if len(ff.calls) != 0 {
		t.Errorf("HTTP fetcher should not be called in browser mode: %v", ff.calls)
	}
}

func TestDetectorRecognisesByRequestHost(t *testing.T) {
	d := NewDetector()
	obs := &domain.Observation{
		Requests: []domain.NetworkRequest{{URL: "https://connect.facebook.net/en_US/fbevents.js", Host: "connect.facebook.net"}},
	}
	tags := d.Detect(obs)
	if !hasTagID(tags, "meta-pixel") {
		t.Errorf("expected meta-pixel, got %+v", tags)
	}
}

func hasTagID(tags []domain.Tag, id string) bool {
	for _, t := range tags {
		if t.ID == id {
			return true
		}
	}
	return false
}
