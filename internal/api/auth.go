package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/scope"
	"github.com/pageguard/pageguard/internal/service"
)

// maxAuthBodyBytes caps buffered JSON bodies during scope validation.
const maxAuthBodyBytes = 1 << 20

// scopeMiddleware enforces authentication, role classes and project scoping
// for /api/v1/* when API-key auth is required. Read verbs accept any role,
// writes need writer/admin, and /api/v1/api-keys is admin-only. A scoped key
// cannot cross projects: mismatches in the projectId query parameter or JSON
// body are rejected. Resource ids inside paths resolve their project through
// the services (which honour the request scope).
type scopeMiddleware struct{ next http.Handler }

func (s *Server) scope(next http.Handler) http.Handler {
	if !s.authRequired {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			next.ServeHTTP(w, r)
			return
		}
		key, err := s.bearerKey(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/api-keys") {
			if !key.HasRole(domain.RoleAdmin) {
				writeError(w, http.StatusForbidden, "API key management requires the admin role")
				return
			}
		} else if isWrite(r.Method) && !key.HasRole(domain.RoleWriter) {
			writeError(w, http.StatusForbidden, "this key's role is read-only")
			return
		}
		if !key.Unrestricted() {
			if !s.withinScope(r, key) {
				writeError(w, http.StatusForbidden, "cross-project access is not permitted with a scoped API key")
				return
			}
		}
		ctx := scope.With(r.Context(), &scope.Scope{
			KeyID:     key.ID,
			ProjectID: key.ProjectID,
			Roles:     append([]string(nil), key.Roles...),
		})
		s.touchAPIKey(key.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// bearerKey resolves the Authorization header into a valid API key.
func (s *Server) bearerKey(r *http.Request) (*domain.APIKey, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return nil, errors.New("valid API key required (Authorization: Bearer pg_…)")
	}
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || len(token) < 4 {
		return nil, errors.New("authorization header must carry a bearer token")
	}
	if !strings.HasPrefix(token, "pg_") {
		return nil, errors.New("API key must start with pg_")
	}
	key, err := s.svc.Store().GetAPIKeyByHash(r.Context(), domain.HashAPIKey(token))
	if err != nil {
		return nil, errors.New("invalid API key")
	}
	if !key.Active() {
		return nil, errors.New("API key is disabled or expired")
	}
	return key, nil
}

func isWrite(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

// withinScope guards explicit cross-project access: the projectId query
// parameter and JSON create/update bodies must not point outside the key's
// project. Everything else (resource ids in paths) is validated by the
// services, which see the request scope through the context.
func (s *Server) withinScope(r *http.Request, key *domain.APIKey) bool {
	pinned := key.ProjectID
	if got := domain.ID(r.URL.Query().Get("projectId")); !got.IsEmpty() && got != pinned {
		return false
	}
	if !isWrite(r.Method) || !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		return true
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxAuthBodyBytes))
	if err != nil {
		return false
	}
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) == 0 {
		return true
	}
	var parsed struct {
		ProjectID domain.ID `json:"projectId"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && !parsed.ProjectID.IsEmpty() {
		return parsed.ProjectID == pinned
	}
	return true
}

// touchAPIKey records key usage at most once a minute per key, off the
// request path.
func (s *Server) touchAPIKey(id domain.ID) {
	s.touchMu.Lock()
	if time.Since(s.touchLog[id]) < time.Minute {
		s.touchMu.Unlock()
		return
	}
	s.touchLog[id] = time.Now()
	s.touchMu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.svc.TouchAPIKey(ctx, id)
	}()
}

// ---- API key management endpoints (admin only) ---------------------------

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var input service.APIKeyInput
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodyBytes)).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	secret, err := s.svc.CreateAPIKey(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, secret)
}

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.svc.ListAPIKeys(r.Context(), domain.ID(r.URL.Query().Get("projectId")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

func (s *Server) handleUpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	var patch service.APIKeyPatch
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodyBytes)).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	key, err := s.svc.UpdateAPIKey(r.Context(), domain.ID(r.PathValue("id")), patch)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (s *Server) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteAPIKey(r.Context(), domain.ID(r.PathValue("id"))); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}
