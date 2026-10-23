// Package orchestrator runs scans: it walks the URL frontier breadth-first,
// drives the scanner, applies validator plugins, persists results, and emits
// progress events.
//
// Runs are checkpointed per page: every successfully scanned page is written as
// a domain.ScanResult keyed by (run, url). Re-executing a run is therefore
// idempotent and resumable — checkpointed pages are loaded rather than
// re-fetched, and traversal continues from their stored links.
package orchestrator

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/events"
	"github.com/pageguard/pageguard/internal/governance"
	"github.com/pageguard/pageguard/internal/plugin"
	"github.com/pageguard/pageguard/internal/proxy"
	"github.com/pageguard/pageguard/internal/rules"
	"github.com/pageguard/pageguard/internal/scanner"
)

// maxStoredDOM caps the DOM retained per ScanResult to keep checkpoints bounded.
const maxStoredDOM = 2 << 20 // 2 MiB

// Manager executes scan runs. It is safe for concurrent runs.
type Manager struct {
	store   domain.Store
	hub     *events.Hub
	scanner *scanner.Scanner
	plugins *plugin.Host
	rotator *proxy.Rotator

	captureHAR       bool
	captureDataLayer bool
}

// Option configures a Manager.
type Option func(*Manager)

// WithProxyRotator enables proxy rotation for browser renders. Pages are routed
// through a proxy selected by their assigned geolocation.
func WithProxyRotator(r *proxy.Rotator) Option {
	return func(m *Manager) { m.rotator = r }
}

// WithCapture enables HAR and/or data-layer capture for every run.
func WithCapture(har, dataLayer bool) Option {
	return func(m *Manager) { m.captureHAR, m.captureDataLayer = har, dataLayer }
}

// NewManager wires the dependencies required to execute a run.
func NewManager(store domain.Store, hub *events.Hub, scn *scanner.Scanner, host *plugin.Host, opts ...Option) *Manager {
	m := &Manager{store: store, hub: hub, scanner: scn, plugins: host}
	for _, o := range opts {
		o(m)
	}
	return m
}

// geoPicker assigns geolocations to pages in round-robin order. It is safe for
// concurrent use across a run's worker pool.
type geoPicker struct {
	regions []string
	n       int64
}

func newGeoPicker(regions []string) *geoPicker {
	clean := make([]string, 0, len(regions))
	for _, r := range regions {
		if r = strings.TrimSpace(r); r != "" {
			clean = append(clean, r)
		}
	}
	return &geoPicker{regions: clean}
}

func (g *geoPicker) next() string {
	if len(g.regions) == 0 {
		return ""
	}
	i := atomic.AddInt64(&g.n, 1) - 1
	return g.regions[int(i)%len(g.regions)]
}

type frontierItem struct {
	url   string
	depth int
}

type pageResult struct {
	item        frontierItem
	page        *domain.Page
	obs         *domain.Observation
	scanResult  *domain.ScanResult
	links       scanner.Links
	resumed     bool
	geolocation string
	err         error
}

// Execute runs a single scan to completion. It returns a non-nil error only for
// fatal, run-level failures (bad configuration or cancellation); individual
// page failures are recorded on the run's stats.
func (m *Manager) Execute(ctx context.Context, scan *domain.Scan, run *domain.Run) error {
	cfg := scan.Config
	cfg.Normalize()

	// Reset counters so a resumed/re-executed run rebuilds them from its
	// checkpoints rather than compounding prior totals.
	run.Stats = domain.RunStats{}
	run.Error = ""

	em := events.NewEmitter(m.hub, m.store, run)
	started := time.Now().UTC()
	run.Status = domain.RunRunning
	run.StartedAt = &started
	run.ProjectID = scan.ProjectID
	if err := m.store.UpdateRun(ctx, run); err != nil {
		return err
	}
	em.SetStatus(domain.RunRunning)

	robots := newRobotsCache(m.scanner.Fetcher, cfg.UserAgent)
	geo := newGeoPicker(cfg.Geolocations)

	projectRules := []*domain.AuditRule{}
	if stored, err := m.store.ListRules(ctx, scan.ProjectID); err == nil {
		for _, rule := range stored {
			if rule.Enabled {
				projectRules = append(projectRules, rule)
			}
		}
	}

	hosts := map[string]bool{}
	visited := map[string]bool{}
	var current []frontierItem
	for _, s := range scan.StartURLs {
		u, err := url.Parse(s)
		if err != nil || u.Host == "" {
			continue
		}
		n := normalizeURL(u)
		if visited[n] {
			continue
		}
		visited[n] = true
		hosts[strings.ToLower(u.Hostname())] = true
		current = append(current, frontierItem{url: s, depth: 0})
		em.IncDiscovered()
		em.PageDiscovered(s, 0)
	}
	if len(current) == 0 {
		m.finalize(ctx, run, em, "no valid start URLs")
		return fmt.Errorf("orchestrator: scan %s has no valid start URLs", scan.ID)
	}

	scanned := 0
	for depth := 0; len(current) > 0 && scanned < cfg.MaxPages; depth++ {
		if err := ctx.Err(); err != nil {
			m.finalize(ctx, run, em, "")
			return err
		}
		batch := current
		if scanned+len(batch) > cfg.MaxPages {
			batch = batch[:cfg.MaxPages-scanned]
		}
		results := m.scanBatch(ctx, batch, scan, run.ID, cfg, em, geo)
		scanned += len(batch)

		var next []frontierItem
		for _, r := range results {
			if r.err != nil {
				em.IncFailed()
				p := &domain.Page{
					ID: domain.NewID(), RunID: run.ID, URL: r.item.url, Depth: r.item.depth,
					Error: r.err.Error(), ScannedAt: time.Now().UTC(),
				}
				_ = m.store.SavePage(ctx, p)
				continue
			}
			em.IncScanned()
			if r.resumed {
				em.PageResumed(r.item.url, r.item.depth, r.geolocation)
			} else {
				_ = m.store.SavePage(ctx, r.page)

				pluginFindings, _ := m.plugins.Validate(ctx, plugin.ValidateInput{
					RunID: run.ID, ProjectID: scan.ProjectID, ScanID: scan.ID,
					Page: r.page, Observation: r.obs, RuleSetIDs: scan.RuleSetIDs,
				})
				findings := append([]*domain.Finding{}, pluginFindings...)
				if r.scanResult != nil {
					ruleResults := rules.EvaluateAll(projectRules, r.scanResult)
					findings = append(findings, rules.Findings(run.ID, r.page.ID, r.page.URL, ruleResults)...)
				}
				for _, f := range findings {
					_ = m.store.SaveFinding(ctx, f)
					em.Finding(f)
				}
				em.IncFindings(len(findings))
				em.PageCompleted(r.page)
			}

			if r.item.depth < cfg.MaxDepth && isHTML(r.obs) {
				for _, link := range r.links.Documents {
					if !m.allowURL(ctx, link, hosts, cfg, robots) {
						continue
					}
					n := normalizeURLString(link)
					if visited[n] {
						continue
					}
					visited[n] = true
					next = append(next, frontierItem{url: link, depth: r.item.depth + 1})
					em.IncDiscovered()
					em.PageDiscovered(link, r.item.depth+1)
				}
			}
		}
		current = next
	}

	m.finalize(ctx, run, em, "")
	return nil
}

func (m *Manager) scanBatch(ctx context.Context, batch []frontierItem, scan *domain.Scan, runID domain.ID, cfg domain.ScanConfig, em *events.Emitter, geo *geoPicker) []pageResult {
	results := make([]pageResult, len(batch))
	jobs := make(chan int)
	var wg sync.WaitGroup

	workers := cfg.Concurrency
	if workers > len(batch) {
		workers = len(batch)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				results[idx] = m.scanOne(ctx, batch[idx], scan, runID, cfg, em, geo)
			}
		}()
	}
	for i := range batch {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return results
}

func (m *Manager) scanOne(ctx context.Context, it frontierItem, scan *domain.Scan, runID domain.ID, cfg domain.ScanConfig, em *events.Emitter, geo *geoPicker) pageResult {
	em.PageStarted(it.url, it.depth)

	// Idempotency: reuse the checkpoint if this URL was already scanned.
	if m.store != nil {
		if sr, err := m.store.GetScanResult(ctx, runID, it.url); err == nil && sr != nil {
			obs := observationFromScanResult(sr)
			page := &domain.Page{
				ID: sr.PageID, RunID: runID, URL: sr.URL, FinalURL: sr.FinalURL,
				Status: sr.StatusCode, Depth: sr.Depth, Rendered: sr.Rendered, Title: sr.Title,
				ContentType: sr.ContentType, DurationMs: sr.DurationMs, ScannedAt: sr.FinishedAt,
			}
			return pageResult{
				item: it, page: page, obs: obs, scanResult: sr, links: linksFromScanResult(sr),
				resumed: true, geolocation: sr.Geolocation,
			}
		}
	}

	geolocation := geo.next()
	opts := scanner.ScanOptions{
		Depth:            it.depth,
		Geolocation:      geolocation,
		CaptureHAR:       m.captureHAR || cfg.CaptureHAR,
		CaptureDataLayer: m.captureDataLayer || cfg.CaptureDataLayer,
	}

	var proxyUsed proxy.Proxy
	if m.rotator != nil && !m.rotator.Empty() {
		if p, ok := m.rotator.Next(geolocation); ok {
			proxyUsed = p
			opts.ProxyServer = p.Server()
			if u, pw, ok := p.Credentials(); ok {
				opts.ProxyUser, opts.ProxyPass = u, pw
			}
		}
	}

	res, err := m.scanner.ScanPage(ctx, it.url, cfg, opts)
	if err != nil {
		if proxyUsed.URL != "" {
			m.rotator.Report(proxyUsed, err)
		}
		em.PageFailed(it.url, err.Error())
		return pageResult{item: it, err: err}
	}
	if proxyUsed.URL != "" {
		m.rotator.Report(proxyUsed, nil)
	}

	obs := res.Observation
	annotateRequests(obs)
	page := &domain.Page{
		ID: domain.NewID(), RunID: runID, URL: it.url, FinalURL: obs.FinalURL,
		Status: obs.StatusCode, Depth: it.depth, Rendered: obs.Rendered, Title: obs.Title,
		ContentType: obs.ContentType, DurationMs: obs.DurationMs, ScannedAt: time.Now().UTC(),
	}
	obs.PageID = page.ID

	sr := buildScanResult(scan, runID, it, res, page, opts.Geolocation, proxyUsed.Host())
	_ = m.store.SaveScanResult(ctx, sr)

	return pageResult{item: it, page: page, obs: obs, scanResult: sr, links: res.Links, geolocation: opts.Geolocation}
}

// buildScanResult assembles the persisted, per-page evidence record from a
// fresh scan.
func buildScanResult(scan *domain.Scan, runID domain.ID, it frontierItem, res *scanner.PageResult, page *domain.Page, geolocation, proxyHost string) *domain.ScanResult {
	obs := res.Observation
	dom := obs.HTML
	truncated := obs.HTMLTruncated
	if len(dom) > maxStoredDOM {
		dom = dom[:maxStoredDOM]
		truncated = true
	}
	return &domain.ScanResult{
		SchemaVersion:   domain.ScanResultSchemaVersion,
		RunID:           runID,
		ScanID:          scan.ID,
		PageID:          page.ID,
		URL:             it.url,
		FinalURL:        obs.FinalURL,
		Depth:           it.depth,
		Attempt:         1,
		StatusCode:      obs.StatusCode,
		ContentType:     obs.ContentType,
		Title:           obs.Title,
		Rendered:        obs.Rendered,
		RenderMode:      scan.Config.RenderMode,
		Geolocation:     geolocation,
		ProxyHost:       proxyHost,
		StartedAt:       page.ScannedAt,
		FinishedAt:      page.ScannedAt,
		DurationMs:      obs.DurationMs,
		DOM:             dom,
		DOMTruncated:    truncated,
		DOMBytes:        len(obs.HTML),
		Requests:        obs.Requests,
		Cookies:         obs.Cookies,
		Console:         obs.Console,
		DataLayer:       res.DataLayer,
		HAR:             res.HAR,
		Tags:            obs.Tags,
		Links:           res.Links.Documents,
		MetaDescription: res.MetaDescription,
	}
}

// observationFromScanResult reconstructs enough of an Observation to drive
// crawl traversal from a stored checkpoint.
func observationFromScanResult(sr *domain.ScanResult) *domain.Observation {
	return &domain.Observation{
		URL:           sr.URL,
		FinalURL:      sr.FinalURL,
		StatusCode:    sr.StatusCode,
		ContentType:   sr.ContentType,
		Title:         sr.Title,
		HTML:          sr.DOM,
		HTMLTruncated: sr.DOMTruncated,
		Requests:      sr.Requests,
		Tags:          sr.Tags,
		Cookies:       sr.Cookies,
		Console:       sr.Console,
		Geolocation:   sr.Geolocation,
		DataLayer:     sr.DataLayer,
		HAR:           sr.HAR,
		DurationMs:    sr.DurationMs,
		Rendered:      sr.Rendered,
		Depth:         sr.Depth,
		PageID:        sr.PageID,
	}
}

func linksFromScanResult(sr *domain.ScanResult) scanner.Links {
	if len(sr.Links) > 0 {
		return scanner.Links{Documents: sr.Links}
	}
	if base, err := url.Parse(sr.FinalURL); err == nil {
		return scanner.ExtractLinks(base, sr.DOM)
	}
	return scanner.Links{}
}

func (m *Manager) allowURL(ctx context.Context, rawURL string, hosts map[string]bool, cfg domain.ScanConfig, robots *robotsCache) bool {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	if cfg.SameOriginOnly && !hosts[strings.ToLower(u.Hostname())] {
		return false
	}
	if len(cfg.IncludePatterns) > 0 && !matchAny(cfg.IncludePatterns, rawURL) {
		return false
	}
	if len(cfg.ExcludePatterns) > 0 && matchAny(cfg.ExcludePatterns, rawURL) {
		return false
	}
	if cfg.RespectRobotsTxt && !robots.allowed(ctx, rawURL) {
		return false
	}
	return true
}

func (m *Manager) finalize(ctx context.Context, run *domain.Run, em *events.Emitter, reason string) {
	prog := em.Progress()
	run.Stats = prog.Stats
	// Reconcile page/finding totals from persistence so a resumed run reports
	// the same figures as a fresh one, regardless of how many pages it reused
	// from checkpoints.
	if m.store != nil {
		if pages, err := m.store.ListPages(ctx, domain.PageFilter{RunID: run.ID}); err == nil {
			scanned, failed := 0, 0
			for _, p := range pages {
				if p.Error != "" {
					failed++
				} else {
					scanned++
				}
			}
			run.Stats.PagesScanned = scanned
			run.Stats.PagesFailed = failed
		}
		if findings, err := m.store.ListFindings(ctx, domain.FindingFilter{RunID: run.ID}); err == nil {
			run.Stats.Findings = len(findings)
			for _, f := range findings {
				if f.Severity == domain.SevCritical {
					run.Stats.CriticalFindings++
				}
			}
		}
		if results, err := m.store.ListScanResults(ctx, run.ID, 0); err == nil {
			run.Stats.Beacons = governance.BeaconsForResults(results)
			run.Stats.ConsentScore = governance.ConsentScoreForResults(results)
			storedRules, _ := m.store.ListRules(ctx, run.ProjectID)
			enabled := make([]*domain.AuditRule, 0, len(storedRules))
			for _, rule := range storedRules {
				if rule.Enabled {
					enabled = append(enabled, rule)
				}
			}
			byPage := make(map[string][]rules.Result, len(results))
			for _, sr := range results {
				byPage[sr.URL] = rules.EvaluateAll(enabled, sr)
			}
			agg := rules.Summarize(enabled, byPage)
			run.Stats.VendorScores = agg.VendorScores
			if run.Stats.VendorScores == nil {
				run.Stats.VendorScores = map[string]float64{}
			}
			if agg.Evaluated > 0 {
				run.Stats.HealthScore = agg.HealthScore
			} else {
				findings, _ := m.store.ListFindings(ctx, domain.FindingFilter{RunID: run.ID})
				run.Stats.HealthScore = rules.HealthFromFindings(findings)
			}
		}
	}
	now := time.Now().UTC()
	run.FinishedAt = &now
	switch {
	case reason != "":
		run.Status = domain.RunFailed
		run.Error = reason
	case ctx.Err() != nil:
		run.Status = domain.RunCanceled
		run.Error = ctx.Err().Error()
	default:
		run.Status = domain.RunSucceeded
	}
	_ = m.store.UpdateRun(ctx, run)
	em.Complete(run.Error)
}

// annotateRequests fills host and third-party flags on network requests, which
// must be computed after the final URL is known.
func annotateRequests(obs *domain.Observation) {
	if obs == nil {
		return
	}
	pageHost := hostOf(obs.FinalURL)
	if pageHost == "" {
		pageHost = hostOf(obs.URL)
	}
	for i := range obs.Requests {
		r := &obs.Requests[i]
		if r.Host == "" {
			r.Host = hostOf(r.URL)
		}
		r.ThirdParty = !sameSite(strings.ToLower(r.Host), pageHost)
	}
}

func sameSite(host, pageHost string) bool {
	return host == pageHost || strings.HasSuffix(host, "."+pageHost)
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func isHTML(obs *domain.Observation) bool {
	if obs == nil {
		return false
	}
	if obs.Rendered {
		return true
	}
	if obs.ContentType == "" {
		return strings.Contains(strings.ToLower(obs.HTML), "<html") || obs.Rendered
	}
	return strings.Contains(strings.ToLower(obs.ContentType), "html") || obs.Rendered
}

// normalizeURL returns a canonical form of u for de-duplication.
func normalizeURL(u *url.URL) string {
	c := *u
	c.Fragment = ""
	c.Scheme = strings.ToLower(c.Scheme)
	c.Host = strings.ToLower(c.Host)
	return c.String()
}

func normalizeURLString(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return normalizeURL(u)
}
