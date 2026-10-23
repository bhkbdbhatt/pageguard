package events

import (
	"github.com/pageguard/pageguard/internal/domain"
)

// Emitter is a thin helper that publishes typed events for a single run and
// keeps a running Progress snapshot in memory. The orchestrator owns one
// Emitter per run.
type Emitter struct {
	hub      *Hub
	runID    domain.ID
	store    domain.Store
	progress *domain.Progress
}

// NewEmitter creates an emitter bound to a run. The initial progress is seeded
// from the supplied run.
func NewEmitter(hub *Hub, store domain.Store, run *domain.Run) *Emitter {
	return &Emitter{
		hub:   hub,
		runID: run.ID,
		store: store,
		progress: &domain.Progress{
			RunID:  run.ID,
			Status: run.Status,
			Stats:  run.Stats,
		},
	}
}

// Progress returns a copy of the current progress snapshot.
func (e *Emitter) Progress() domain.Progress { return *e.progress }

// emit persists (best-effort) and publishes an event.
func (e *Emitter) emit(t domain.EventType, payload map[string]any) {
	evt := &domain.Event{RunID: e.runID, Type: t, Payload: payload}
	e.hub.Publish(evt)
	if e.store != nil {
		// Persistence is best-effort: a storage failure must not abort a scan.
		_ = e.store.AppendEvent(ctxBackground(), evt)
	}
}

// SetStatus updates and announces the run status.
func (e *Emitter) SetStatus(s domain.RunStatus) {
	e.progress.Status = s
	e.emit(statusEvent(s), map[string]any{"status": string(s)})
}

// IncDiscovered bumps the discovered counter.
func (e *Emitter) IncDiscovered() {
	e.progress.Stats.PagesDiscovered++
}

// IncScanned records a successful page scan.
func (e *Emitter) IncScanned() {
	e.progress.Stats.PagesScanned++
}

// IncFailed records a failed page scan.
func (e *Emitter) IncFailed() {
	e.progress.Stats.PagesFailed++
}

// IncFindings adds n findings to the running total.
func (e *Emitter) IncFindings(n int) {
	e.progress.Stats.Findings += n
}

// PageStarted announces that a page fetch/render has begun.
func (e *Emitter) PageStarted(url string, depth int) {
	e.emit(domain.EvtPageStarted, map[string]any{"url": url, "depth": depth})
}

// PageDiscovered announces a newly enqueued URL.
func (e *Emitter) PageDiscovered(url string, depth int) {
	e.emit(domain.EvtPageDiscovered, map[string]any{"url": url, "depth": depth})
}

// PageCompleted announces a successfully scanned page.
func (e *Emitter) PageCompleted(p *domain.Page) {
	e.emit(domain.EvtPageCompleted, map[string]any{
		"url": p.URL, "status": p.Status, "rendered": p.Rendered,
		"durationMs": p.DurationMs,
	})
}

// PageResumed announces a page served from a checkpoint rather than a fresh
// scan, so consumers can distinguish resumed work.
func (e *Emitter) PageResumed(url string, depth int, geolocation string) {
	e.emit(domain.EvtPageResumed, map[string]any{
		"url": url, "depth": depth, "geolocation": geolocation,
	})
}

// PageFailed announces a page that could not be scanned.
func (e *Emitter) PageFailed(url string, err string) {
	e.emit(domain.EvtPageFailed, map[string]any{"url": url, "error": err})
}

// Finding announces a single finding.
func (e *Emitter) Finding(f *domain.Finding) {
	e.emit(domain.EvtFinding, map[string]any{
		"ruleId": f.RuleID, "pluginId": f.PluginID,
		"severity": string(f.Severity), "title": f.Title, "pageUrl": f.PageURL,
	})
}

// Log emits a free-form informational log line.
func (e *Emitter) Log(msg string, fields map[string]any) {
	p := map[string]any{"message": msg}
	for k, v := range fields {
		p[k] = v
	}
	e.emit(domain.EvtLog, p)
}

// Complete emits the terminal event and returns the final progress snapshot.
func (e *Emitter) Complete(err string) {
	if err != "" {
		e.SetStatus(domain.RunFailed)
		e.emit(domain.EvtRunFailed, map[string]any{"error": err, "stats": e.progress.Stats})
	} else {
		e.SetStatus(domain.RunSucceeded)
		e.emit(domain.EvtRunCompleted, map[string]any{"stats": e.progress.Stats})
	}
}

func statusEvent(s domain.RunStatus) domain.EventType {
	switch s {
	case domain.RunQueued:
		return domain.EvtRunQueued
	case domain.RunRunning:
		return domain.EvtRunStarted
	case domain.RunFailed:
		return domain.EvtRunFailed
	default:
		return domain.EvtRunCompleted
	}
}
