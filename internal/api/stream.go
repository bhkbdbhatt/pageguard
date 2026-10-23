package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

const sseHeartbeat = 15 * time.Second

// handleSSE streams run events as Server-Sent Events. It is the default
// progress transport because it needs no handshake and survives proxies.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	runID := domain.ID(r.PathValue("id"))
	if _, err := s.svc.Store().GetRun(r.Context(), runID); err != nil {
		writeStoreError(w, err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")

	ch, cancel := s.svc.Hub().Subscribe(runID)
	defer cancel()

	// Advise the client to reconnect after 3s and send the current run state.
	fmt.Fprint(w, "retry: 3000\n\n")
	if run, err := s.svc.Store().GetRun(r.Context(), runID); err == nil {
		writeSSE(w, "run.snapshot", 0, run)
	}
	flusher.Flush()

	heartbeat := time.NewTicker(sseHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, string(evt.Type), evt.ID, evt)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, event string, id int64, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if id > 0 {
		fmt.Fprintf(w, "id: %d\n", id)
	}
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// handleWS streams run events over a WebSocket. It is the low-latency
// transport; clients that cannot upgrade should fall back to /events (SSE).
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	runID := domain.ID(r.PathValue("id"))
	if _, err := s.svc.Store().GetRun(r.Context(), runID); err != nil {
		writeStoreError(w, err)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r)
	if err != nil {
		s.logger.Warn("ws upgrade failed", "err", err, "runId", runID)
		return
	}
	defer conn.Close()

	ch, cancel := s.svc.Hub().Subscribe(runID)
	defer cancel()

	// Drain inbound frames so close/ping frames are handled promptly.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case evt, ok := <-ch:
			if !ok {
				_ = conn.WriteClose()
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			if err := conn.WriteText(data); err != nil {
				return
			}
		}
	}
}
