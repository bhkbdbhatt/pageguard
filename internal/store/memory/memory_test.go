package memory

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

func TestProjectScanRunLifecycle(t *testing.T) {
	s := New()
	ctx := context.Background()

	p := &domain.Project{ID: domain.NewID(), Name: "Acme", CreatedAt: time.Now()}
	if err := s.CreateProject(ctx, p); err != nil {
		t.Fatal(err)
	}
	if got, err := s.GetProject(ctx, p.ID); err != nil || got.Name != "Acme" {
		t.Fatalf("GetProject = %+v, %v", got, err)
	}
	if _, err := s.GetProject(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	scan := &domain.Scan{ID: domain.NewID(), ProjectID: p.ID, Name: "s", CreatedAt: time.Now()}
	if err := s.CreateScan(ctx, scan); err != nil {
		t.Fatal(err)
	}
	scans, _ := s.ListScans(ctx, p.ID)
	if len(scans) != 1 {
		t.Fatalf("scans = %d", len(scans))
	}

	run := &domain.Run{ID: domain.NewID(), ScanID: scan.ID, ProjectID: p.ID, Status: domain.RunQueued, CreatedAt: time.Now()}
	if err := s.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunSucceeded
	if err := s.UpdateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateRun(ctx, &domain.Run{ID: "missing"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("update missing run: %v", err)
	}
	runs, _ := s.ListRuns(ctx, scan.ID, 10)
	if len(runs) != 1 || runs[0].Status != domain.RunSucceeded {
		t.Fatalf("runs = %+v", runs)
	}
}

func TestPagesAndFindingsFilters(t *testing.T) {
	s := New()
	ctx := context.Background()
	runID := domain.NewID()

	_ = s.SavePage(ctx, &domain.Page{ID: domain.NewID(), RunID: runID, URL: "https://x/a"})
	_ = s.SavePage(ctx, &domain.Page{ID: domain.NewID(), RunID: runID, URL: "https://x/b", Error: "boom"})
	pages, _ := s.ListPages(ctx, domain.PageFilter{RunID: runID})
	if len(pages) != 2 {
		t.Fatalf("pages = %d", len(pages))
	}

	_ = s.SaveFinding(ctx, &domain.Finding{ID: domain.NewID(), RunID: runID, RuleID: "r1", Severity: domain.SevLow, CreatedAt: time.Now()})
	_ = s.SaveFinding(ctx, &domain.Finding{ID: domain.NewID(), RunID: runID, RuleID: "r2", Severity: domain.SevCritical, CreatedAt: time.Now()})

	all, _ := s.ListFindings(ctx, domain.FindingFilter{RunID: runID})
	if len(all) != 2 {
		t.Fatalf("findings = %d", len(all))
	}
	if all[0].Severity != domain.SevCritical {
		t.Errorf("findings not sorted by severity: %+v", all)
	}

	high, _ := s.ListFindings(ctx, domain.FindingFilter{RunID: runID, MinSeverity: domain.SevHigh})
	if len(high) != 1 || high[0].RuleID != "r2" {
		t.Errorf("minSeverity filter wrong: %+v", high)
	}
}

func TestScanResultsCheckpoint(t *testing.T) {
	s := New()
	ctx := context.Background()
	runID := domain.NewID()

	sr := &domain.ScanResult{
		SchemaVersion: domain.ScanResultSchemaVersion,
		RunID:         runID,
		ScanID:        domain.NewID(),
		PageID:        domain.NewID(),
		URL:           "https://x/a",
		FinalURL:      "https://x/a",
		StatusCode:    200,
		RenderMode:    domain.RenderHTTP,
		Geolocation:   "us",
		StartedAt:     time.Now(),
		FinishedAt:    time.Now(),
		DataLayer:     []domain.DataLayerEntry{{Name: "dataLayer[0]", Value: json.RawMessage(`{"e":1}`)}},
	}
	if err := s.SaveScanResult(ctx, sr); err != nil {
		t.Fatal(err)
	}
	// Idempotent upsert: saving the same (run,url) must not create a duplicate.
	if err := s.SaveScanResult(ctx, sr); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetScanResult(ctx, runID, "https://x/a")
	if err != nil || got.Geolocation != "us" {
		t.Fatalf("GetScanResult = %+v, %v", got, err)
	}
	if len(got.DataLayer) != 1 || string(got.DataLayer[0].Value) != `{"e":1}` {
		t.Errorf("data layer round-trip wrong: %+v", got.DataLayer)
	}
	list, _ := s.ListScanResults(ctx, runID, 0)
	if len(list) != 1 {
		t.Fatalf("ListScanResults = %d, want 1", len(list))
	}
	if _, err := s.GetScanResult(ctx, runID, "https://x/missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEventsAppendAndList(t *testing.T) {
	s := New()
	ctx := context.Background()
	runID := domain.NewID()

	for i := 0; i < 3; i++ {
		_ = s.AppendEvent(ctx, &domain.Event{RunID: runID, Type: domain.EvtLog, At: time.Now()})
	}
	all, _ := s.ListEvents(ctx, runID, 0)
	if len(all) != 3 || all[0].ID != 1 || all[2].ID != 3 {
		t.Fatalf("events = %+v", all)
	}
	after, _ := s.ListEvents(ctx, runID, 2)
	if len(after) != 1 || after[0].ID != 3 {
		t.Errorf("afterID filter wrong: %+v", after)
	}
}
