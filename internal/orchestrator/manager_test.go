package orchestrator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/events"
	"github.com/pageguard/pageguard/internal/plugin"
	"github.com/pageguard/pageguard/internal/scanner"
	"github.com/pageguard/pageguard/internal/store/memory"
)

func TestGlobMatch(t *testing.T) {
	cases := []struct {
		pattern, target string
		want            bool
	}{
		{"**/blog/**", "https://x/blog/post", true},
		{"*.pdf", "https://x/a/report.pdf", true},
		{"https://example.com/*", "https://example.com/a", true},
		{"https://example.com/*", "https://example.com/a/b", true},
		{"/admin/*", "https://x/admin/users", false}, // pattern is matched against the full URL
	}
	for _, c := range cases {
		if got := globMatch(c.pattern, c.target); got != c.want {
			t.Errorf("globMatch(%q,%q)=%v want %v", c.pattern, c.target, got, c.want)
		}
	}
}

func newTestManager() (*Manager, *memory.Store, *events.Hub) {
	store := memory.New()
	hub := events.NewHub()
	reg := plugin.NewRegistry()
	for _, v := range plugin.BuiltinValidators() {
		reg.Register(v)
	}
	host := plugin.NewHost(reg)
	scn := scanner.New(scanner.NewHTTPFetcher(), nil)
	return NewManager(store, hub, scn, host), store, hub
}

func siteHandler() http.Handler {
	mux := http.NewServeMux()
	page := func(title, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html><head><title>` + title + `</title>` +
				`<meta name="description" content="d"></head><body>` + body + `</body></html>`))
		}
	}
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /private\n"))
	})
	mux.HandleFunc("/", page("Home", `<a href="/a">a</a><a href="/b">b</a><a href="/private">p</a><a href="https://external.example/x">ext</a>`))
	mux.HandleFunc("/a", page("A", `<a href="/a/child">child</a>`))
	mux.HandleFunc("/a/child", page("Child", ``))
	mux.HandleFunc("/b", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<html><head><title>Missing</title></head><body></body></html>`))
	})
	mux.HandleFunc("/private", page("Private", ``))
	return mux
}

func TestExecuteCrawlsSiteRespectingScopeAndRobots(t *testing.T) {
	srv := httptest.NewServer(siteHandler())
	defer srv.Close()

	mgr, store, hub := newTestManager()

	scan := &domain.Scan{
		ID:        domain.NewID(),
		ProjectID: domain.NewID(),
		Name:      "site",
		StartURLs: []string{srv.URL + "/"},
		Config: domain.ScanConfig{
			MaxDepth:         2,
			MaxPages:         50,
			Concurrency:      3,
			SameOriginOnly:   true,
			RespectRobotsTxt: true,
			RenderMode:       domain.RenderHTTP,
			UserAgent:        "PageguardTest/1.0",
		},
	}
	run := &domain.Run{ID: domain.NewID(), ScanID: scan.ID, ProjectID: scan.ProjectID, Status: domain.RunQueued}
	if err := store.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	ch, cancel := hub.Subscribe(run.ID)
	var mu sync.Mutex
	seen := map[domain.EventType]int{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for evt := range ch {
			mu.Lock()
			seen[evt.Type]++
			mu.Unlock()
		}
	}()

	if err := mgr.Execute(context.Background(), scan, run); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	cancel()
	<-done

	if run.Status != domain.RunSucceeded {
		t.Errorf("run status = %s (err=%q)", run.Status, run.Error)
	}
	if run.Stats.PagesScanned != 4 {
		t.Errorf("pagesScanned = %d, want 4", run.Stats.PagesScanned)
	}

	pages, _ := store.ListPages(context.Background(), domain.PageFilter{RunID: run.ID})
	gotURLs := map[string]bool{}
	for _, p := range pages {
		gotURLs[p.URL] = true
	}
	for _, want := range []string{srv.URL + "/", srv.URL + "/a", srv.URL + "/b", srv.URL + "/a/child"} {
		if !gotURLs[want] {
			t.Errorf("missing page %s (got %v)", want, keys(gotURLs))
		}
	}
	if gotURLs[srv.URL+"/private"] {
		t.Error("/private scanned despite robots exclusion")
	}
	if gotURLs["https://external.example/x"] {
		t.Error("external URL scanned despite SameOriginOnly")
	}

	findings, _ := store.ListFindings(context.Background(), domain.FindingFilter{RunID: run.ID})
	if !hasRule(findings, "core/http-error-status") {
		t.Errorf("expected core/http-error-status finding for /b, got %d findings", len(findings))
	}

	if seen[domain.EvtRunStarted] == 0 || seen[domain.EvtRunCompleted] == 0 {
		t.Errorf("event stream missing lifecycle events: %v", seen)
	}
}

func TestExecuteHonoursMaxPages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		var b strings.Builder
		b.WriteString("<html><head><title>t</title></head><body>")
		for i := 0; i < 20; i++ {
			b.WriteString(`<a href="/p/` + itoa(i) + `">x</a>`)
		}
		b.WriteString("</body></html>")
		_, _ = w.Write([]byte(b.String()))
	})
	mux.HandleFunc("/p/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><head><title>p</title></head><body></body></html>"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mgr, store, _ := newTestManager()
	scan := &domain.Scan{ID: domain.NewID(), ProjectID: domain.NewID(), StartURLs: []string{srv.URL + "/"},
		Config: domain.ScanConfig{MaxDepth: 5, MaxPages: 5, Concurrency: 2, SameOriginOnly: true}}
	run := &domain.Run{ID: domain.NewID(), ScanID: scan.ID, Status: domain.RunQueued}
	if err := store.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	if err := mgr.Execute(context.Background(), scan, run); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	pages, _ := store.ListPages(context.Background(), domain.PageFilter{RunID: run.ID})
	if len(pages) > 5 {
		t.Errorf("scanned %d pages, expected <= 5", len(pages))
	}
	if len(pages) == 0 {
		t.Error("expected at least the start page")
	}
}

func TestExecuteIncludeExcludePatterns(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>t</title></head><body>` +
			`<a href="/blog/one">1</a><a href="/shop/two">2</a></body></html>`))
	})
	mux.HandleFunc("/blog/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><head><title>b</title></head><body></body></html>"))
	})
	mux.HandleFunc("/shop/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><head><title>s</title></head><body></body></html>"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mgr, store, _ := newTestManager()
	scan := &domain.Scan{ID: domain.NewID(), ProjectID: domain.NewID(), StartURLs: []string{srv.URL + "/"},
		Config: domain.ScanConfig{
			MaxDepth: 2, MaxPages: 50, Concurrency: 2,
			IncludePatterns: []string{"*/blog/*"},
			// start URL "/" fails the include filter for *discovered* links only;
			// the start URL is always scanned.
		}}
	run := &domain.Run{ID: domain.NewID(), ScanID: scan.ID, Status: domain.RunQueued}
	if err := store.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := mgr.Execute(context.Background(), scan, run); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	pages, _ := store.ListPages(context.Background(), domain.PageFilter{RunID: run.ID})
	for _, p := range pages {
		if strings.Contains(p.URL, "/shop/") {
			t.Errorf("shop page scanned despite include filter: %s", p.URL)
		}
	}
	if len(pages) != 2 { // "/" + "/blog/one"
		t.Errorf("expected 2 pages, got %d: %v", len(pages), pageURLs(pages))
	}
}

func TestExecuteNoStartURLsIsFatal(t *testing.T) {
	mgr, store, _ := newTestManager()
	scan := &domain.Scan{ID: domain.NewID(), StartURLs: []string{"not a url"}}
	run := &domain.Run{ID: domain.NewID(), ScanID: scan.ID, Status: domain.RunQueued}
	if err := store.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := mgr.Execute(context.Background(), scan, run); err == nil {
		t.Fatal("expected fatal error for no valid start URLs")
	}
	if run.Status != domain.RunFailed {
		t.Errorf("status = %s", run.Status)
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func pageURLs(pages []*domain.Page) []string {
	out := make([]string, 0, len(pages))
	for _, p := range pages {
		out = append(out, p.URL)
	}
	return out
}

func hasRule(findings []*domain.Finding, ruleID string) bool {
	for _, f := range findings {
		if f.RuleID == ruleID {
			return true
		}
	}
	return false
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
