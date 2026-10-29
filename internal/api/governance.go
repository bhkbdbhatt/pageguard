package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/service"
)

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := s.svc.Governance().Overview(r.Context(), domain.ID(r.URL.Query().Get("projectId")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) handleListAudits(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.Governance().Audits(r.Context(), domain.ID(r.URL.Query().Get("projectId")), parseLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"audits": rows})
}

func (s *Server) handleCreateAudit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID         domain.ID         `json:"projectId"`
		Name              string            `json:"name"`
		StartURL          string            `json:"startUrl"`
		MaxPages          int               `json:"maxPages"`
		MaxDepth          int               `json:"maxDepth"`
		RenderMode        domain.RenderMode `json:"renderMode"`
		ConsentSimulation string            `json:"consentSimulation"`
		Geolocation       string            `json:"geolocation"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	scan, run, err := s.svc.CreateAudit(r.Context(), service.CreateAuditRequest{
		ProjectID: req.ProjectID, Name: req.Name, StartURL: req.StartURL,
		MaxPages: req.MaxPages, MaxDepth: req.MaxDepth, RenderMode: req.RenderMode,
		ConsentSimulation: req.ConsentSimulation, Geolocation: req.Geolocation,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"scan": scan, "run": run})
}

func (s *Server) handleAuditsExport(w http.ResponseWriter, r *http.Request) {
	raw, err := s.svc.Governance().AuditCSV(r.Context(), domain.ID(r.URL.Query().Get("projectId")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pageguard-audits.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *Server) handleValidation(w http.ResponseWriter, r *http.Request) {
	report, err := s.svc.Governance().Validation(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleBeacons(w http.ResponseWriter, r *http.Request) {
	report, err := s.svc.Governance().Beacons(r.Context(), domain.ID(r.PathValue("id")), r.URL.Query().Get("url"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleConsent(w http.ResponseWriter, r *http.Request) {
	report, err := s.svc.Governance().Consent(r.Context(), domain.ID(r.PathValue("id")), r.URL.Query().Get("url"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	raw, contentType, err := s.svc.Governance().ReportBytes(r.Context(), domain.ID(r.PathValue("id")), format)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeStoreError(w, err)
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	filename := "pageguard-report." + format
	disposition := "attachment"
	if format == "html" {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ID(r.URL.Query().Get("projectId"))
	if projectID.IsEmpty() {
		projects, err := s.svc.Store().ListProjects(r.Context())
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if len(projects) > 0 {
			projectID = projects[0].ID
		}
	}
	all, err := s.svc.Store().ListRules(r.Context(), projectID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if all == nil {
		all = []*domain.AuditRule{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": all})
}

func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID   domain.ID           `json:"projectId"`
		Name        string              `json:"name"`
		Vendor      string              `json:"vendor"`
		Enabled     *bool               `json:"enabled"`
		URLPattern  string              `json:"urlPattern"`
		Target      domain.RuleTarget   `json:"target"`
		Parameter   string              `json:"parameter"`
		Event       string              `json:"event"`
		Operator    domain.RuleOperator `json:"operator"`
		Expected    string              `json:"expected"`
		Severity    domain.Severity     `json:"severity"`
		Description string              `json:"description"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule, err := s.svc.CreateRule(r.Context(), &domain.AuditRule{
		ProjectID: req.ProjectID, Name: req.Name, Vendor: req.Vendor, Enabled: enabled,
		URLPattern: req.URLPattern, Target: req.Target, Parameter: req.Parameter,
		Event: req.Event, Operator: req.Operator, Expected: req.Expected,
		Severity: req.Severity, Description: req.Description,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleStarterRules(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID domain.ID `json:"projectId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	created, err := s.svc.StarterRules(r.Context(), req.ProjectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"rules": created})
}

func (s *Server) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	var patch service.RulePatch
	if !readJSON(w, r, &patch) {
		return
	}
	rule, err := s.svc.UpdateRule(r.Context(), domain.ID(r.PathValue("id")), patch)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeStoreError(w, err)
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteRule(r.Context(), domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrConflict) {
		writeStoreError(w, err)
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}
