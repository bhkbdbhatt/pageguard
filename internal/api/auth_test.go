package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/events"
	"github.com/pageguard/pageguard/internal/orchestrator"
	"github.com/pageguard/pageguard/internal/plugin"
	"github.com/pageguard/pageguard/internal/scanner"
	"github.com/pageguard/pageguard/internal/service"
	"github.com/pageguard/pageguard/internal/store/memory"
)

func newAuthTestAPI(t *testing.T) (*httptest.Server, *service.Service, string) {
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

	secret, err := svc.CreateAPIKey(context.Background(), service.APIKeyInput{
		Name:  "test admin",
		Roles: []string{domain.RoleAdmin},
	})
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	srv := httptest.NewServer(NewServer(svc, reg, logger, WithAuth(true)).Handler())
	return srv, svc, secret.Secret
}

func TestAuthRequiresAPIKey(t *testing.T) {
	srv, _, secret := newAuthTestAPI(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/projects")
	if err != nil {
		t.Fatalf("unauthenticated request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("authenticated request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authenticated status = %d, want 200", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200", resp.StatusCode)
	}
}

func TestAuthQueryKeyOnlyAcceptedForStreams(t *testing.T) {
	srv, _, secret := newAuthTestAPI(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/projects?api_key=" + secret)
	if err != nil {
		t.Fatalf("query key request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("query key on REST status = %d, want 401", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/api/v1/runs/run_missing/events?api_key=" + secret)
	if err != nil {
		t.Fatalf("stream query key request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("stream endpoint should accept api_key query parameter")
	}
}

func TestAuthReaderCannotWrite(t *testing.T) {
	srv, svc, _ := newAuthTestAPI(t)
	defer srv.Close()
	ctx := context.Background()
	project, err := svc.CreateProject(ctx, "Reader Project", "reader-project")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	reader, err := svc.CreateAPIKey(ctx, service.APIKeyInput{
		Name:      "reader",
		ProjectID: project.ID,
		Roles:     []string{domain.RoleReader},
	})
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer "+reader.Secret)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("reader write request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("reader write status = %d, want 403", resp.StatusCode)
	}
}
