package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/service"
)

// ---- inventory ----

func (s *Server) handleInventory(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.svc.Inventory(r.Context(),
		domain.ID(r.URL.Query().Get("projectId")), domain.ID(r.URL.Query().Get("runId")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

// ---- schedules ----

func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.Schedules(r.Context(), domain.ID(r.URL.Query().Get("projectId")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schedules": rows})
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	row, err := s.svc.GetSchedule(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req service.ScheduleInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.CreateSchedule(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var patch service.SchedulePatch
	if !readJSON(w, r, &patch) {
		return
	}
	row, err := s.svc.UpdateSchedule(r.Context(), domain.ID(r.PathValue("id")), patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteSchedule(r.Context(), domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRunSchedule(w http.ResponseWriter, r *http.Request) {
	run, err := s.svc.RunScheduleNow(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"run": run})
}

// ---- alerts ----

func (s *Server) handleListAlertRules(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.AlertRules(r.Context(), domain.ID(r.URL.Query().Get("projectId")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rows})
}

func (s *Server) handleCreateAlertRule(w http.ResponseWriter, r *http.Request) {
	var req service.AlertRuleInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.CreateAlertRule(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) handleUpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	var req service.AlertRuleInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.UpdateAlertRule(r.Context(), domain.ID(r.PathValue("id")), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleDeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteAlertRule(r.Context(), domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListAlertChannels(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.AlertChannels(r.Context(), domain.ID(r.URL.Query().Get("projectId")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channels": rows})
}

func (s *Server) handleCreateAlertChannel(w http.ResponseWriter, r *http.Request) {
	var req service.AlertChannelInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.CreateAlertChannel(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) handleUpdateAlertChannel(w http.ResponseWriter, r *http.Request) {
	var req service.AlertChannelInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.UpdateAlertChannel(r.Context(), domain.ID(r.PathValue("id")), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleDeleteAlertChannel(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteAlertChannel(r.Context(), domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTestAlertChannel(w http.ResponseWriter, r *http.Request) {
	delivery, err := s.svc.TestAlertChannel(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "delivery": delivery})
}

func (s *Server) handleListAlertIncidents(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.AlertIncidents(r.Context(),
		domain.ID(r.URL.Query().Get("projectId")), parseLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"incidents": rows})
}

func (s *Server) handleUpdateAlertIncident(w http.ResponseWriter, r *http.Request) {
	var patch service.IncidentPatch
	if !readJSON(w, r, &patch) {
		return
	}
	row, err := s.svc.UpdateIncident(r.Context(), domain.ID(r.PathValue("id")), patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAlertDeliveries(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.AlertDeliveries(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deliveries": rows})
}

// ---- journeys ----

func (s *Server) handleListJourneys(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.Journeys(r.Context(), domain.ID(r.URL.Query().Get("projectId")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"journeys": rows})
}

func (s *Server) handleGetJourney(w http.ResponseWriter, r *http.Request) {
	row, err := s.svc.GetJourney(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleCreateJourney(w http.ResponseWriter, r *http.Request) {
	var req service.JourneyInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.CreateJourney(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) handleUpdateJourney(w http.ResponseWriter, r *http.Request) {
	var req service.JourneyInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.UpdateJourney(r.Context(), domain.ID(r.PathValue("id")), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleDeleteJourney(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteJourney(r.Context(), domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListJourneyRuns(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.JourneyRuns(r.Context(),
		domain.ID(r.PathValue("id")), parseLimit(r.URL.Query().Get("limit"), 25))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": rows})
}

func (s *Server) handleRunJourney(w http.ResponseWriter, r *http.Request) {
	run, err := s.svc.RunJourney(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"run": run})
}

func (s *Server) handleGetJourneyRun(w http.ResponseWriter, r *http.Request) {
	row, err := s.svc.GetJourneyRun(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

// ---- privacy ----

func (s *Server) handleListPrivacyProfiles(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.PrivacyProfiles(r.Context(), domain.ID(r.URL.Query().Get("projectId")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": rows})
}

func (s *Server) handleCreatePrivacyProfile(w http.ResponseWriter, r *http.Request) {
	var req service.PrivacyProfileInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.CreatePrivacyProfile(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) handleUpdatePrivacyProfile(w http.ResponseWriter, r *http.Request) {
	var req service.PrivacyProfileInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.UpdatePrivacyProfile(r.Context(), domain.ID(r.PathValue("id")), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleDeletePrivacyProfile(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeletePrivacyProfile(r.Context(), domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListPrivacyBatches(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.PrivacyBatches(r.Context(),
		domain.ID(r.URL.Query().Get("projectId")), parseLimit(r.URL.Query().Get("limit"), 50))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"batches": rows})
}

func (s *Server) handleGetPrivacyBatch(w http.ResponseWriter, r *http.Request) {
	row, err := s.svc.GetPrivacyBatch(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleStartPrivacyBatch(w http.ResponseWriter, r *http.Request) {
	var req service.PrivacyBatchInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.StartPrivacyBatch(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, row)
}

func (s *Server) handleListPrivacyEvaluations(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.PrivacyEvaluations(r.Context(),
		domain.ID(r.PathValue("id")), parseLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"evaluations": rows})
}

// ---- reports ----

func (s *Server) handleReportTemplates(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"templates": service.ReportTemplates()})
}

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.Reports(r.Context(),
		domain.ID(r.URL.Query().Get("projectId")), parseLimit(r.URL.Query().Get("limit"), 50))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": rows})
}

func (s *Server) handleCreateReport(w http.ResponseWriter, r *http.Request) {
	var req service.ReportInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.GenerateReport(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	row, err := s.svc.Report(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleDownloadReport(w http.ResponseWriter, r *http.Request) {
	row, err := s.svc.Report(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	filename := reportFilename(row.Name, row.Format)
	disposition := "attachment"
	if row.Format == "html" {
		disposition = "inline"
	}
	contentType := row.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(row.Content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(row.Content)
}

func (s *Server) handleDeleteReport(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteReport(r.Context(), domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListReportSchedules(w http.ResponseWriter, r *http.Request) {
	rows, err := s.svc.ReportSchedules(r.Context(), domain.ID(r.URL.Query().Get("projectId")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schedules": rows})
}

func (s *Server) handleCreateReportSchedule(w http.ResponseWriter, r *http.Request) {
	var req service.ReportScheduleInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.CreateReportSchedule(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) handleUpdateReportSchedule(w http.ResponseWriter, r *http.Request) {
	var req service.ReportScheduleInput
	if !readJSON(w, r, &req) {
		return
	}
	row, err := s.svc.UpdateReportSchedule(r.Context(), domain.ID(r.PathValue("id")), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleDeleteReportSchedule(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteReportSchedule(r.Context(), domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRunReportSchedule(w http.ResponseWriter, r *http.Request) {
	row, err := s.svc.RunReportScheduleNow(r.Context(), domain.ID(r.PathValue("id")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func reportFilename(name, format string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "pageguard-report"
	}
	if len(slug) > 80 {
		slug = slug[:80]
	}
	if format == "" {
		format = "json"
	}
	return slug + "." + format
}
