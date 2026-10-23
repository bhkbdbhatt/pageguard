package orchestrator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/events"
	"github.com/pageguard/pageguard/internal/plugin"
	"github.com/pageguard/pageguard/internal/proxy"
	"github.com/pageguard/pageguard/internal/scanner"
	"github.com/pageguard/pageguard/internal/store/memory"
)

func TestExecuteResumeIsIdempotent(t *testing.T) {
	var hits int32
	mux := http.NewServeMux()
	page := func(title, body string, status int) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			if status != 0 {
				w.WriteHeader(status)
			}
			_, _ = w.Write([]byte("<html><head><title>" + title + "</title></head><body>" + body + "</body></html>"))
		}
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		page("Home", `<a href="/a">a</a><a href="/b">b</a>`, 0)(w, r)
	})
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		page("A", ``, 0)(w, r)
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		page("B", ``, http.StatusNotFound)(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mgr, store, hub := newTestManager()
	scan := &domain.Scan{
		ID: domain.NewID(), ProjectID: domain.NewID(), Name: "resume",
		StartURLs: []string{srv.URL + "/"},
		Config: domain.ScanConfig{
			MaxDepth: 2, MaxPages: 50, Concurrency: 2,
			SameOriginOnly: true, RespectRobotsTxt: false, RenderMode: domain.RenderHTTP,
		},
	}
	run := &domain.Run{ID: domain.NewID(), ScanID: scan.ID, ProjectID: scan.ProjectID, Status: domain.RunQueued}
	if err := store.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	if err := mgr.Execute(context.Background(), scan, run); err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	firstHits := atomic.LoadInt32(&hits)
	firstStats := run.Stats
	pages1, _ := store.ListPages(context.Background(), domain.PageFilter{RunID: run.ID})
	findings1, _ := store.ListFindings(context.Background(), domain.FindingFilter{RunID: run.ID})
	results1, _ := store.ListScanResults(context.Background(), run.ID, 0)
	if firstStats.PagesScanned != 3 {
		t.Fatalf("first pagesScanned = %d, want 3", firstStats.PagesScanned)
	}
	if firstStats.Findings == 0 {
		t.Fatalf("first findings = 0, want at least the 404 finding")
	}
	if len(results1) != 3 {
		t.Fatalf("first scan results = %d, want 3", len(results1))
	}

	// Re-execute the same run: every page must come from its checkpoint.
	ch, cancel := hub.Subscribe(run.ID)
	var mu sync.Mutex
	resumed := 0
	done := make(chan struct{})
	go func() {
		defer close(done)
		for evt := range ch {
			if evt.Type == domain.EvtPageResumed {
				mu.Lock()
				resumed++
				mu.Unlock()
			}
		}
	}()

	if err := mgr.Execute(context.Background(), scan, run); err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	cancel()
	<-done

	if got := atomic.LoadInt32(&hits); got != firstHits {
		t.Errorf("resume re-fetched pages: hits %d -> %d", firstHits, got)
	}
	pages2, _ := store.ListPages(context.Background(), domain.PageFilter{RunID: run.ID})
	findings2, _ := store.ListFindings(context.Background(), domain.FindingFilter{RunID: run.ID})
	results2, _ := store.ListScanResults(context.Background(), run.ID, 0)
	if len(pages2) != len(pages1) {
		t.Errorf("pages duplicated on resume: %d -> %d", len(pages1), len(pages2))
	}
	if len(findings2) != len(findings1) {
		t.Errorf("findings duplicated on resume: %d -> %d", len(findings1), len(findings2))
	}
	if len(results2) != len(results1) {
		t.Errorf("scan results duplicated on resume: %d -> %d", len(results1), len(results2))
	}
	if !reflect.DeepEqual(run.Stats, firstStats) {
		t.Errorf("resume stats diverged: %+v -> %+v", firstStats, run.Stats)
	}
	if resumed != 3 {
		t.Errorf("resumed events = %d, want 3", resumed)
	}
}

// recordingRenderer captures the RenderOptions each URL was scanned with and
// serves canned HTML, so geolocation/proxy wiring can be asserted without a
// real browser.
type recordingRenderer struct {
	mu   sync.Mutex
	opts []scanner.RenderOptions
	html map[string]string
}

func (r *recordingRenderer) Render(_ context.Context, rawURL string, o scanner.RenderOptions) (*scanner.RenderResult, error) {
	r.mu.Lock()
	r.opts = append(r.opts, o)
	r.mu.Unlock()
	body := r.html[rawURL]
	if body == "" {
		body = "<html><head><title>x</title></head><body></body></html>"
	}
	return &scanner.RenderResult{
		FinalURL: rawURL, Status: 200, ContentType: "text/html", Title: "x", HTML: body,
	}, nil
}

func (r *recordingRenderer) Close() error { return nil }

func TestExecuteAssignsGeolocationsAndProxies(t *testing.T) {
	const root = "https://site.test/"
	rr := &recordingRenderer{html: map[string]string{
		root: `<html><head><title>root</title></head><body><a href="/child">c</a></body></html>`,
	}}

	store := memory.New()
	hub := events.NewHub()
	reg := plugin.NewRegistry()
	for _, v := range plugin.BuiltinValidators() {
		reg.Register(v)
	}
	scn := scanner.New(nil, rr)
	rotator := proxy.New([]proxy.Proxy{
		{URL: "http://us.proxy:8080", Region: "us"},
		{URL: "http://de.proxy:8080", Region: "de"},
	})
	mgr := NewManager(store, hub, scn, plugin.NewHost(reg), WithProxyRotator(rotator))

	scan := &domain.Scan{
		ID: domain.NewID(), ProjectID: domain.NewID(), Name: "geo",
		StartURLs: []string{root},
		Config: domain.ScanConfig{
			MaxDepth: 1, MaxPages: 10, Concurrency: 1,
			SameOriginOnly: true, RespectRobotsTxt: false,
			RenderMode:   domain.RenderBrowser,
			Geolocations: []string{"us", "de"},
		},
	}
	run := &domain.Run{ID: domain.NewID(), ScanID: scan.ID, ProjectID: scan.ProjectID, Status: domain.RunQueued}
	if err := store.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := mgr.Execute(context.Background(), scan, run); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	results, _ := store.ListScanResults(context.Background(), run.ID, 0)
	geo := map[string]int{}
	proxyHost := map[string]string{}
	for _, sr := range results {
		geo[sr.Geolocation]++
		proxyHost[sr.Geolocation] = sr.ProxyHost
	}
	if geo["us"] != 1 || geo["de"] != 1 {
		t.Fatalf("geolocation assignment = %+v, want one us and one de", geo)
	}
	if proxyHost["us"] != "us.proxy:8080" || proxyHost["de"] != "de.proxy:8080" {
		t.Fatalf("proxy assignment = %+v", proxyHost)
	}

	for _, o := range rr.opts {
		want := ""
		switch o.Geolocation {
		case "us":
			want = "http://us.proxy:8080"
		case "de":
			want = "http://de.proxy:8080"
		}
		if o.ProxyServer != want {
			t.Errorf("geolocation %q routed via %q, want %q", o.Geolocation, o.ProxyServer, want)
		}
		if o.GeoLatitude == 0 || o.Locale == "" || o.Timezone == "" {
			t.Errorf("region %q not resolved to coordinates/locale/timezone: %+v", o.Geolocation, o)
		}
	}
}

func TestGeoPickerRoundRobin(t *testing.T) {
	empty := newGeoPicker(nil)
	if got := empty.next(); got != "" {
		t.Errorf("empty picker = %q", got)
	}
	g := newGeoPicker([]string{" us ", "eu", ""})
	got := []string{g.next(), g.next(), g.next(), g.next()}
	want := []string{"us", "eu", "us", "eu"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("picker sequence = %v, want %v", got, want)
	}
}
