package plugin

import (
	"context"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func newHost() *Host {
	reg := NewRegistry()
	for _, v := range BuiltinValidators() {
		reg.Register(v)
	}
	return NewHost(reg)
}

func TestHostProducesDomainFindings(t *testing.T) {
	h := newHost()
	obs := &domain.Observation{URL: "https://x/", FinalURL: "https://x/", StatusCode: 500, Title: "T"}
	page := &domain.Page{ID: domain.NewID(), RunID: domain.NewID(), URL: "https://x/"}

	findings, err := h.Validate(context.Background(), ValidateInput{
		RunID: page.RunID, ProjectID: domain.NewID(), ScanID: domain.NewID(),
		Page: page, Observation: obs,
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !hasFinding(findings, "core/http-error-status") {
		t.Fatalf("expected http-error-status finding, got %d", len(findings))
	}
	for _, f := range findings {
		if f.RunID != page.RunID || f.PageID != page.ID {
			t.Errorf("finding not linked to run/page: %+v", f)
		}
		if f.PageURL != "https://x/" || f.PluginID != "core" {
			t.Errorf("finding metadata wrong: %+v", f)
		}
	}
}

func TestHostRuleSetFilter(t *testing.T) {
	h := newHost()
	obs := &domain.Observation{URL: "https://x/", FinalURL: "https://x/", StatusCode: 500, Title: "T"}
	findings, err := h.Validate(context.Background(), ValidateInput{
		RunID: domain.NewID(), Observation: obs, RuleSetIDs: []string{"some-other-plugin"},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("rule-set filter should exclude core, got %d findings", len(findings))
	}
}

func TestToSDKMapsConsoleErrors(t *testing.T) {
	obs := &domain.Observation{
		Console: []domain.ConsoleMessage{
			{Level: "error", Text: "boom"},
			{Level: "log", Text: "ignore"},
		},
	}
	sdk := ToSDK(obs)
	if len(sdk.ConsoleErrors) != 1 || sdk.ConsoleErrors[0] != "boom" {
		t.Errorf("console errors = %v", sdk.ConsoleErrors)
	}
}

func hasFinding(findings []*domain.Finding, ruleID string) bool {
	for _, f := range findings {
		if f.RuleID == ruleID {
			return true
		}
	}
	return false
}
