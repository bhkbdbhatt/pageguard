// Package api exposes the Pageguard REST + streaming API.
package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/plugin"
	"github.com/pageguard/pageguard/internal/service"
	"github.com/pageguard/pageguard/internal/ws"
)

// Server holds HTTP handlers and their dependencies.
type Server struct {
	svc      *service.Service
	registry *plugin.Registry
	upgrader ws.Upgrader
	logger   *slog.Logger
}

// NewServer constructs an API server.
func NewServer(svc *service.Service, registry *plugin.Registry, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		svc:      svc,
		registry: registry,
		upgrader: ws.Upgrader{MaxMessageSize: 1 << 20},
		logger:   logger,
	}
}

// Handler returns the fully wired HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)

	mux.HandleFunc("GET /api/v1/projects", s.handleListProjects)
	mux.HandleFunc("POST /api/v1/projects", s.handleCreateProject)

	mux.HandleFunc("GET /api/v1/scans", s.handleListScans)
	mux.HandleFunc("POST /api/v1/scans", s.handleCreateScan)
	mux.HandleFunc("GET /api/v1/scans/{id}", s.handleGetScan)
	mux.HandleFunc("GET /api/v1/scans/{id}/runs", s.handleListScanRuns)
	mux.HandleFunc("POST /api/v1/scans/{id}/runs", s.handleStartRun)

	mux.HandleFunc("GET /api/v1/runs/{id}", s.handleGetRun)
	mux.HandleFunc("POST /api/v1/runs/{id}/cancel", s.handleCancelRun)
	mux.HandleFunc("POST /api/v1/runs/{id}/resume", s.handleResumeRun)
	mux.HandleFunc("GET /api/v1/runs/{id}/pages", s.handleListPages)
	mux.HandleFunc("GET /api/v1/runs/{id}/results", s.handleListResults)
	mux.HandleFunc("GET /api/v1/runs/{id}/result", s.handleGetResult)
	mux.HandleFunc("GET /api/v1/runs/{id}/findings", s.handleListFindings)
	mux.HandleFunc("GET /api/v1/runs/{id}/events", s.handleSSE)
	mux.HandleFunc("GET /api/v1/runs/{id}/stream", s.handleWS)

	mux.HandleFunc("GET /api/v1/overview", s.handleOverview)
	mux.HandleFunc("GET /api/v1/audits", s.handleListAudits)
	mux.HandleFunc("POST /api/v1/audits", s.handleCreateAudit)
	mux.HandleFunc("GET /api/v1/audits/export", s.handleAuditsExport)

	mux.HandleFunc("GET /api/v1/runs/{id}/validation", s.handleValidation)
	mux.HandleFunc("GET /api/v1/runs/{id}/beacons", s.handleBeacons)
	mux.HandleFunc("GET /api/v1/runs/{id}/consent", s.handleConsent)
	mux.HandleFunc("GET /api/v1/runs/{id}/report", s.handleReport)

	mux.HandleFunc("GET /api/v1/rules", s.handleListRules)
	mux.HandleFunc("POST /api/v1/rules", s.handleCreateRule)
	mux.HandleFunc("POST /api/v1/rules/starter", s.handleStarterRules)
	mux.HandleFunc("PATCH /api/v1/rules/{id}", s.handleUpdateRule)
	mux.HandleFunc("DELETE /api/v1/rules/{id}", s.handleDeleteRule)

	mux.HandleFunc("GET /api/v1/plugins", s.handleListPlugins)

	return s.cors(s.recover(s.logRequests(mux)))
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	raw := strings.TrimSpace(os.Getenv("PAGEGUARD_CORS_ORIGINS"))
	if raw == "" {
		raw = "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001,http://127.0.0.1:3001"
	}
	for _, origin := range strings.Split(raw, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowed[origin] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowed[origin] || allowed["*"]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---- middleware ----

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		s.logger.Info("http",
			"method", r.Method, "path", r.URL.Path, "status", sw.status,
			"durationMs", time.Since(start).Milliseconds())
	})
}

func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic", "path", r.URL.Path, "err", rec)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush forwards to the underlying writer when supported (needed for SSE).
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack forwards to the underlying writer so WebSocket upgrades work through
// the logging middleware.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("api: response writer does not support hijacking")
	}
	return hj.Hijack()
}

// ---- handlers ----

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"inflight": s.svc.Inflight(),
		"time":     time.Now().UTC(),
	})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.svc.Store().ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	p, err := s.svc.CreateProject(r.Context(), req.Name, req.Slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListScans(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ID(r.URL.Query().Get("projectId"))
	scans, err := s.svc.Store().ListScans(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scans": scans})
}

func (s *Server) handleCreateScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID  domain.ID         `json:"projectId"`
		Name       string            `json:"name"`
		StartURLs  []string          `json:"startUrls"`
		Config     domain.ScanConfig `json:"config"`
		RuleSetIDs []string          `json:"ruleSetIds"`
		Enabled    *bool             `json:"enabled"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || len(req.StartURLs) == 0 {
		writeError(w, http.StatusBadRequest, "name and at least one startUrl are required")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	scan, err := s.svc.CreateScan(r.Context(), &domain.Scan{
		ProjectID:  req.ProjectID,
		Name:       req.Name,
		StartURLs:  req.StartURLs,
		Config:     req.Config,
		RuleSetIDs: req.RuleSetIDs,
		Enabled:    enabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, scan)
}

func (s *Server) handleGetScan(w http.ResponseWriter, r *http.Request) {
	scan, err := s.svc.Store().GetScan(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scan)
}

func (s *Server) handleListScanRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.svc.Store().ListRuns(r.Context(), domain.ID(r.PathValue("id")), parseLimit(r.URL.Query().Get("limit"), 50))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s *Server) handleStartRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.svc.StartRun(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.svc.Store().GetRun(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.CancelRun(domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "canceled"})
}

func (s *Server) handleResumeRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.svc.ResumeRun(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) handleListResults(w http.ResponseWriter, r *http.Request) {
	results, err := s.svc.ScanResults(r.Context(), domain.ID(r.PathValue("id")), parseLimit(r.URL.Query().Get("limit"), 500))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handleGetResult(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if strings.TrimSpace(rawURL) == "" {
		writeError(w, http.StatusBadRequest, "url query parameter is required")
		return
	}
	result, err := s.svc.ScanResult(r.Context(), domain.ID(r.PathValue("id")), rawURL)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleListPages(w http.ResponseWriter, r *http.Request) {
	f := domain.PageFilter{
		RunID:      domain.ID(r.PathValue("id")),
		FailedOnly: r.URL.Query().Get("failedOnly") == "true",
		Limit:      parseLimit(r.URL.Query().Get("limit"), 500),
	}
	pages, err := s.svc.Store().ListPages(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pages": pages})
}

func (s *Server) handleListFindings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.FindingFilter{
		RunID:       domain.ID(r.PathValue("id")),
		MinSeverity: domain.Severity(q.Get("minSeverity")),
		RuleID:      q.Get("ruleId"),
		PluginID:    q.Get("pluginId"),
		Limit:       parseLimit(q.Get("limit"), 500),
	}
	findings, err := s.svc.Store().ListFindings(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"findings": findings})
}

func (s *Server) handleListPlugins(w http.ResponseWriter, _ *http.Request) {
	type pluginInfo struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	out := []pluginInfo{}
	for _, v := range s.registry.Validators() {
		out = append(out, pluginInfo{ID: v.ID(), Type: "validator"})
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": out})
}

// ---- helpers ----

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	defer body.Close()
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	if dec.More() {
		writeError(w, http.StatusBadRequest, "unexpected trailing data")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, domain.ErrConflict) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func parseLimit(raw string, def int) int {
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
