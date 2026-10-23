package postgres

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// TestStoreRoundTrip exercises every domain.Store method against a real
// PostgreSQL instance. It is skipped unless PAGEGUARD_TEST_DATABASE_URL is set,
// e.g.
//
//	PAGEGUARD_TEST_DATABASE_URL=postgres://pageguard:pageguard@localhost:5432/pageguard?sslmode=disable go test ./internal/store/postgres
func TestStoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("PAGEGUARD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PAGEGUARD_TEST_DATABASE_URL to run postgres integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer st.Close()

	if err := st.Migrate(ctx, filepath.Join("..", "..", "..", "migrations")); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)

	// -- project --
	proj := &domain.Project{ID: domain.NewID(), Name: "Test Project", Slug: "test-" + string(domain.NewID())[:8], CreatedAt: now, UpdatedAt: now}
	if err := st.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	gotProj, err := st.GetProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if gotProj.ID != proj.ID {
		t.Fatalf("project id round-trip: got %q want %q", gotProj.ID, proj.ID)
	}
	if _, err := st.GetProject(ctx, domain.NewID()); err != domain.ErrNotFound {
		t.Fatalf("GetProject missing: got %v want ErrNotFound", err)
	}
	if projects, err := st.ListProjects(ctx); err != nil {
		t.Fatalf("ListProjects: %v", err)
	} else if len(projects) == 0 {
		t.Fatal("ListProjects: empty")
	}

	// -- scan --
	scan := &domain.Scan{
		ID: domain.NewID(), ProjectID: proj.ID, Name: "Test Scan",
		StartURLs:  []string{"https://example.com/"},
		Config:     domain.ScanConfig{MaxDepth: 2, MaxPages: 10, RenderMode: domain.RenderHTTP},
		RuleSetIDs: []string{"core"}, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := st.CreateScan(ctx, scan); err != nil {
		t.Fatalf("CreateScan: %v", err)
	}
	gotScan, err := st.GetScan(ctx, scan.ID)
	if err != nil {
		t.Fatalf("GetScan: %v", err)
	}
	if gotScan.Config.MaxDepth != 2 || len(gotScan.StartURLs) != 1 {
		t.Fatalf("scan config round-trip: %+v", gotScan)
	}
	scan.Name = "Renamed Scan"
	if err := st.UpdateScan(ctx, scan); err != nil {
		t.Fatalf("UpdateScan: %v", err)
	}
	if scans, err := st.ListScans(ctx, proj.ID); err != nil {
		t.Fatalf("ListScans: %v", err)
	} else if len(scans) == 0 {
		t.Fatal("ListScans: empty")
	}

	// -- run --
	run := &domain.Run{ID: domain.NewID(), ScanID: scan.ID, ProjectID: proj.ID, Status: domain.RunQueued, CreatedAt: now}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	run.Status = domain.RunSucceeded
	run.Stats = domain.RunStats{PagesScanned: 3, Findings: 1}
	if err := st.UpdateRun(ctx, run); err != nil {
		t.Fatalf("UpdateRun: %v", err)
	}
	gotRun, err := st.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if gotRun.Stats.PagesScanned != 3 {
		t.Fatalf("run stats round-trip: %+v", gotRun.Stats)
	}
	if runs, err := st.ListRuns(ctx, scan.ID, 10); err != nil {
		t.Fatalf("ListRuns: %v", err)
	} else if len(runs) == 0 {
		t.Fatal("ListRuns: empty")
	}

	// -- page --
	page := &domain.Page{
		ID: domain.NewID(), RunID: run.ID, URL: "https://example.com/", FinalURL: "https://example.com/",
		Status: 200, Depth: 0, Rendered: false, Title: "Example", ContentType: "text/html",
		DurationMs: 12, ScannedAt: now,
	}
	if err := st.SavePage(ctx, page); err != nil {
		t.Fatalf("SavePage: %v", err)
	}
	gotPage, err := st.GetPage(ctx, page.ID)
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if gotPage.Title != "Example" {
		t.Fatalf("page round-trip: %+v", gotPage)
	}
	if pages, err := st.ListPages(ctx, domain.PageFilter{RunID: run.ID}); err != nil {
		t.Fatalf("ListPages: %v", err)
	} else if len(pages) != 1 {
		t.Fatalf("ListPages: got %d want 1", len(pages))
	}

	// -- finding --
	finding := &domain.Finding{
		ID: domain.NewID(), RunID: run.ID, PageID: page.ID, PageURL: page.URL,
		PluginID: "core", RuleID: "core/http-error-status", Severity: domain.SevHigh,
		Title: "Bad status", Description: "desc", Evidence: map[string]any{"url": page.URL}, CreatedAt: now,
	}
	if err := st.SaveFinding(ctx, finding); err != nil {
		t.Fatalf("SaveFinding: %v", err)
	}
	if findings, err := st.ListFindings(ctx, domain.FindingFilter{RunID: run.ID, MinSeverity: domain.SevMedium}); err != nil {
		t.Fatalf("ListFindings: %v", err)
	} else if len(findings) != 1 {
		t.Fatalf("ListFindings: got %d want 1", len(findings))
	}
	// A higher minimum severity must exclude the high finding.
	if findings, err := st.ListFindings(ctx, domain.FindingFilter{RunID: run.ID, MinSeverity: domain.SevCritical}); err != nil {
		t.Fatalf("ListFindings(critical): %v", err)
	} else if len(findings) != 0 {
		t.Fatalf("ListFindings(critical): got %d want 0", len(findings))
	}

	// -- scan result (per-page checkpoint) --
	sr := &domain.ScanResult{
		SchemaVersion: domain.ScanResultSchemaVersion,
		RunID:         run.ID, ScanID: scan.ID, PageID: page.ID,
		URL: page.URL, FinalURL: page.URL, Depth: 0, Attempt: 1,
		StatusCode: 200, ContentType: "text/html", Title: "Example",
		RenderMode: domain.RenderHTTP, Geolocation: "us", ProxyHost: "proxy:8080",
		StartedAt: now, FinishedAt: now, DurationMs: 12,
		DOM: "<html></html>", DOMBytes: 13,
		Requests:  []domain.NetworkRequest{{Method: "GET", URL: page.URL, Host: "example.com", Status: 200}},
		DataLayer: []domain.DataLayerEntry{{Name: "dataLayer[0]", Value: json.RawMessage(`{"event":"x"}`)}},
		Links:     []string{"https://example.com/a"},
	}
	if err := st.SaveScanResult(ctx, sr); err != nil {
		t.Fatalf("SaveScanResult: %v", err)
	}
	// Idempotent upsert: a second save of the same (run,url) must not duplicate.
	if err := st.SaveScanResult(ctx, sr); err != nil {
		t.Fatalf("SaveScanResult (upsert): %v", err)
	}
	gotSR, err := st.GetScanResult(ctx, run.ID, page.URL)
	if err != nil {
		t.Fatalf("GetScanResult: %v", err)
	}
	if gotSR.Geolocation != "us" || gotSR.ProxyHost != "proxy:8080" {
		t.Fatalf("scan result round-trip: %+v", gotSR)
	}
	if len(gotSR.DataLayer) != 1 || string(gotSR.DataLayer[0].Value) != `{"event":"x"}` {
		t.Fatalf("scan result data layer round-trip: %+v", gotSR.DataLayer)
	}
	if list, err := st.ListScanResults(ctx, run.ID, 0); err != nil {
		t.Fatalf("ListScanResults: %v", err)
	} else if len(list) != 1 {
		t.Fatalf("ListScanResults: got %d want 1", len(list))
	}
	if _, err := st.GetScanResult(ctx, run.ID, "https://example.com/missing"); err != domain.ErrNotFound {
		t.Fatalf("GetScanResult missing: got %v want ErrNotFound", err)
	}

	// -- event --
	evt := &domain.Event{RunID: run.ID, Type: domain.EvtRunStarted, At: now, Payload: map[string]any{"pages": 3}}
	if err := st.AppendEvent(ctx, evt); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	if evt.ID == 0 {
		t.Fatal("AppendEvent did not assign an id")
	}
	if events, err := st.ListEvents(ctx, run.ID, 0); err != nil {
		t.Fatalf("ListEvents: %v", err)
	} else if len(events) != 1 {
		t.Fatalf("ListEvents: got %d want 1", len(events))
	}
}
