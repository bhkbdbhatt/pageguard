package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/events"
	"github.com/pageguard/pageguard/internal/orchestrator"
	"github.com/pageguard/pageguard/internal/plugin"
	"github.com/pageguard/pageguard/internal/scanner"
	"github.com/pageguard/pageguard/internal/service"
	"github.com/pageguard/pageguard/internal/store/memory"
)

func targetSite() http.Handler {
	mux := http.NewServeMux()
	page := func(title, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head><title>` + title + `</title><meta name="description" content="d"></head><body>` + body + `</body></html>`))
		}
	}
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
	})
	mux.HandleFunc("/", page("Home", `<a href="/a">a</a><a href="/missing">m</a>`))
	mux.HandleFunc("/a", page("A", ``))
	mux.HandleFunc("/missing", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<html><head><title>missing</title></head><body></body></html>"))
	})
	return mux
}

func newTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	store := memory.New()
	hub := events.NewHub()
	reg := plugin.NewRegistry()
	for _, v := range plugin.BuiltinValidators() {
		reg.Register(v)
	}
	scn := scanner.New(scanner.NewHTTPFetcher(), nil)
	host := plugin.NewHost(reg)
	mgr := orchestrator.NewManager(store, hub, scn, host)
	svc := service.New(store, hub, mgr)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httptest.NewServer(NewServer(svc, reg, logger).Handler())
}

func TestEndToEndScanLifecycle(t *testing.T) {
	target := httptest.NewServer(targetSite())
	defer target.Close()
	apiSrv := newTestAPI(t)
	defer apiSrv.Close()

	var proj domain.Project
	postJSON(t, apiSrv.URL+"/api/v1/projects", map[string]any{"name": "Acme"}, &proj)
	if proj.ID.IsEmpty() {
		t.Fatal("project id empty")
	}

	var scan domain.Scan
	postJSON(t, apiSrv.URL+"/api/v1/scans", map[string]any{
		"projectId": proj.ID,
		"name":      "smoke",
		"startUrls": []string{target.URL + "/"},
		"config": map[string]any{
			"maxDepth": 1, "maxPages": 20, "concurrency": 2,
			"sameOriginOnly": true, "respectRobotsTxt": true,
			"userAgent": "PageguardTest/1.0",
		},
	}, &scan)

	var run domain.Run
	postJSON(t, apiSrv.URL+"/api/v1/scans/"+scan.ID.String()+"/runs", nil, &run)
	run = waitForRun(t, apiSrv.URL, run.ID)

	if run.Status != domain.RunSucceeded {
		t.Fatalf("run status = %s (err=%q)", run.Status, run.Error)
	}
	if run.Stats.PagesScanned != 3 {
		t.Errorf("pagesScanned = %d, want 3", run.Stats.PagesScanned)
	}

	var pagesResp struct {
		Pages []domain.Page `json:"pages"`
	}
	getJSON(t, apiSrv.URL+"/api/v1/runs/"+run.ID.String()+"/pages", &pagesResp)
	if len(pagesResp.Pages) != 3 {
		t.Errorf("pages = %d, want 3", len(pagesResp.Pages))
	}

	var findResp struct {
		Findings []domain.Finding `json:"findings"`
	}
	getJSON(t, apiSrv.URL+"/api/v1/runs/"+run.ID.String()+"/findings?minSeverity=high", &findResp)
	if !hasRule(findResp.Findings, "core/http-error-status") {
		t.Errorf("expected http-error-status finding, got %+v", findResp.Findings)
	}
}

func TestListScanRuns(t *testing.T) {
	target := httptest.NewServer(targetSite())
	defer target.Close()
	apiSrv := newTestAPI(t)
	defer apiSrv.Close()

	var proj domain.Project
	postJSON(t, apiSrv.URL+"/api/v1/projects", map[string]any{"name": "Acme"}, &proj)
	var scan domain.Scan
	postJSON(t, apiSrv.URL+"/api/v1/scans", map[string]any{
		"projectId": proj.ID, "name": "history", "startUrls": []string{target.URL + "/"},
		"config": map[string]any{"maxDepth": 0, "maxPages": 1, "userAgent": "PageguardTest/1.0"},
	}, &scan)

	var first domain.Run
	postJSON(t, apiSrv.URL+"/api/v1/scans/"+scan.ID.String()+"/runs", nil, &first)
	waitForRun(t, apiSrv.URL, first.ID)

	// A second scan (same project) must not leak into the first scan's history.
	var other domain.Scan
	postJSON(t, apiSrv.URL+"/api/v1/scans", map[string]any{
		"projectId": proj.ID, "name": "other", "startUrls": []string{target.URL + "/"},
		"config": map[string]any{"maxDepth": 0, "maxPages": 1, "userAgent": "PageguardTest/1.0"},
	}, &other)
	var second domain.Run
	postJSON(t, apiSrv.URL+"/api/v1/scans/"+other.ID.String()+"/runs", nil, &second)
	waitForRun(t, apiSrv.URL, second.ID)

	var resp struct {
		Runs []domain.Run `json:"runs"`
	}
	getJSON(t, apiSrv.URL+"/api/v1/scans/"+scan.ID.String()+"/runs", &resp)
	if len(resp.Runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(resp.Runs))
	}
	if resp.Runs[0].ID != first.ID {
		t.Errorf("run id = %s, want %s", resp.Runs[0].ID, first.ID)
	}
	if resp.Runs[0].Stats.PagesScanned != 1 {
		t.Errorf("pagesScanned = %d, want 1", resp.Runs[0].Stats.PagesScanned)
	}
}

func TestSSEStreamsSnapshot(t *testing.T) {
	target := httptest.NewServer(targetSite())
	defer target.Close()
	apiSrv := newTestAPI(t)
	defer apiSrv.Close()

	var proj domain.Project
	postJSON(t, apiSrv.URL+"/api/v1/projects", map[string]any{"name": "Acme"}, &proj)
	var scan domain.Scan
	postJSON(t, apiSrv.URL+"/api/v1/scans", map[string]any{
		"projectId": proj.ID, "name": "sse", "startUrls": []string{target.URL + "/"},
		"config": map[string]any{"maxDepth": 0, "maxPages": 1, "userAgent": "PageguardTest/1.0"},
	}, &scan)
	var run domain.Run
	postJSON(t, apiSrv.URL+"/api/v1/scans/"+scan.ID.String()+"/runs", nil, &run)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiSrv.URL+"/api/v1/runs/"+run.ID.String()+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sse request: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q", ct)
	}

	br := bufio.NewReader(resp.Body)
	sawData := false
	for i := 0; i < 20; i++ {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "data:") {
			sawData = true
			break
		}
	}
	if !sawData {
		t.Error("expected an SSE data line")
	}
}

func TestWebSocketUpgrades(t *testing.T) {
	target := httptest.NewServer(targetSite())
	defer target.Close()
	apiSrv := newTestAPI(t)
	defer apiSrv.Close()

	var proj domain.Project
	postJSON(t, apiSrv.URL+"/api/v1/projects", map[string]any{"name": "Acme"}, &proj)
	var scan domain.Scan
	postJSON(t, apiSrv.URL+"/api/v1/scans", map[string]any{
		"projectId": proj.ID, "name": "ws", "startUrls": []string{target.URL + "/"},
		"config": map[string]any{"maxDepth": 0, "maxPages": 1, "userAgent": "PageguardTest/1.0"},
	}, &scan)
	var run domain.Run
	postJSON(t, apiSrv.URL+"/api/v1/scans/"+scan.ID.String()+"/runs", nil, &run)

	addr := strings.TrimPrefix(apiSrv.URL, "http://")
	netConn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer netConn.Close()
	handshake := "GET /api/v1/runs/" + run.ID.String() + "/stream HTTP/1.1\r\n" +
		"Host: " + addr + "\r\n" +
		"Upgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := netConn.Write([]byte(handshake)); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = netConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	n, err := netConn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(buf[:n]), "101") {
		t.Errorf("expected 101 Switching Protocols, got %q", string(buf[:n]))
	}
}

// ---- helpers ----

func waitForRun(t *testing.T, base string, id domain.ID) domain.Run {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var run domain.Run
		getJSON(t, base+"/api/v1/runs/"+id.String(), &run)
		if run.Status.Terminal() {
			return run
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("run %s did not terminate", id)
	return domain.Run{}
}

func getJSON(t *testing.T, url string, out any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s: status %d: %s", url, resp.StatusCode, body)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func postJSON(t *testing.T, url string, body, out any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(http.MethodPost, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s: status %d: %s", url, resp.StatusCode, b)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode %s: %v", url, err)
		}
	}
}

func hasRule(findings []domain.Finding, ruleID string) bool {
	for _, f := range findings {
		if f.RuleID == ruleID {
			return true
		}
	}
	return false
}
