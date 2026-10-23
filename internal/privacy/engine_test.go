package privacy

import (
	"context"
	"errors"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func testConfig() *Config {
	return &Config{
		Name:        "test",
		Regulations: []Regulation{GDPR, CCPA, EPrivacy},
		CookieRules: []CookieRule{
			{NamePattern: "^_ga", Category: domain.TagAnalytics},
			{NamePattern: "^_fbp", Category: domain.TagAdvertising},
			{NamePattern: "(?i)^sid", Essential: true},
		},
		Settings: Settings{
			CMPTags:           []string{"onetrust"},
			OptOutURLPatterns: []string{"(?i)do-not-sell"},
		},
	}
}

func goodObservation() *Observation {
	return &Observation{
		URL:             "https://acme.test/",
		FinalURL:        "https://acme.test/",
		BannerPresent:   true,
		BannerHasReject: true,
		Cookies:         []domain.Cookie{{Name: "sid"}},
		Links:           []string{"https://acme.test/do-not-sell"},
	}
}

func goodRescanner() *FakeRescanner {
	accept := goodObservation()
	accept.Tags = []domain.Tag{{ID: "ga", Name: "Google Analytics", Category: domain.TagAnalytics}}
	blocked := goodObservation()
	return &FakeRescanner{ByState: map[ConsentState]*Observation{
		StateAcceptAll:     accept,
		StateRejectAll:     blocked,
		StateNecessaryOnly: blocked,
		StateGPC:           blocked,
	}}
}

func badObservation() *Observation {
	return &Observation{
		URL:      "https://acme.test/",
		FinalURL: "https://acme.test/",
		Tags: []domain.Tag{
			{ID: "ga", Category: domain.TagAnalytics},
			{ID: "fb", Category: domain.TagAdvertising},
		},
		Cookies: []domain.Cookie{{Name: "_ga"}, {Name: "_fbp"}},
	}
}

func badRescanner() *FakeRescanner {
	o := func() *Observation { return badObservation() }
	return &FakeRescanner{ByState: map[ConsentState]*Observation{
		StateAcceptAll:     o(),
		StateRejectAll:     o(),
		StateNecessaryOnly: o(),
		StateGPC:           o(),
	}}
}

func TestEvaluateGoodPage(t *testing.T) {
	rep, err := Evaluate(context.Background(), testConfig(), goodRescanner())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !rep.Passed {
		t.Errorf("expected pass; violations: %+v", rep.Violations())
	}
	if rep.Score != 100 {
		t.Errorf("score = %d, want 100", rep.Score)
	}
	if rep.URL != "https://acme.test/" {
		t.Errorf("url = %q", rep.URL)
	}
	if rep.Summary.Regulations != 3 || rep.Summary.Failed != 0 {
		t.Errorf("summary = %+v", rep.Summary)
	}
	if len(rep.States) != 4 {
		t.Errorf("states = %d, want 4", len(rep.States))
	}
}

func TestEvaluateBadPage(t *testing.T) {
	rep, err := Evaluate(context.Background(), testConfig(), badRescanner())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if rep.Passed {
		t.Error("expected failure")
	}
	if rep.Score != 0 {
		t.Errorf("score = %d, want 0", rep.Score)
	}
	if rep.Summary.Failed != 3 {
		t.Errorf("failed = %d, want 3", rep.Summary.Failed)
	}
	if rep.Summary.Violations == 0 {
		t.Error("expected violations")
	}
	if len(rep.Findings()) != rep.Summary.Violations {
		t.Errorf("findings %d != violations %d", len(rep.Findings()), rep.Summary.Violations)
	}
}

func TestEvaluateGPCSignal(t *testing.T) {
	cfg := testConfig()
	cfg.Regulations = []Regulation{CCPA}
	sale := &Observation{
		URL:      "https://acme.test/",
		FinalURL: "https://acme.test/",
		Tags:     []domain.Tag{{ID: "fb", Category: domain.TagAdvertising}},
	}
	r := &FakeRescanner{ByState: map[ConsentState]*Observation{StateGPC: sale}}
	rep, err := Evaluate(context.Background(), cfg, r)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	check := findCheck(t, rep, CCPA, "do-not-sell-signal")
	if check.Status != CheckFail {
		t.Errorf("do-not-sell-signal = %s, want fail", check.Status)
	}
}

func TestEvaluateFlags(t *testing.T) {
	cfg := testConfig()
	cfg.Settings.DPIA = []DPIATrigger{{
		ID:          "tracking",
		Description: "behavioural tracking",
		Categories:  []domain.TagCategory{domain.TagAdvertising},
		MinTrackers: 3,
	}}
	cfg.Settings.CrossBorderHosts = []string{"adnetwork.io"}

	accept := goodObservation()
	accept.Tags = []domain.Tag{{ID: "fb", Category: domain.TagAdvertising}}
	accept.Requests = []domain.NetworkRequest{{URL: "https://adnetwork.io/px", Host: "adnetwork.io", ResourceType: "script"}}
	blocked := goodObservation()
	r := &FakeRescanner{ByState: map[ConsentState]*Observation{
		StateAcceptAll: accept, StateRejectAll: blocked, StateNecessaryOnly: blocked, StateGPC: blocked,
	}}

	rep, err := Evaluate(context.Background(), cfg, r)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	ids := map[string]bool{}
	for _, f := range rep.Flags {
		ids[f.ID] = true
	}
	if !ids["dpia:tracking"] {
		t.Errorf("missing DPIA flag; got %v", ids)
	}
	if !ids["cross-border-transfer"] {
		t.Errorf("missing cross-border flag; got %v", ids)
	}
}

func TestEvaluateErrors(t *testing.T) {
	if _, err := Evaluate(context.Background(), nil, goodRescanner()); err == nil {
		t.Error("expected error for nil config")
	}
	if _, err := Evaluate(context.Background(), testConfig(), nil); err == nil {
		t.Error("expected error for nil rescanner")
	}
	bad := &FakeRescanner{Err: errors.New("boom")}
	if _, err := Evaluate(context.Background(), testConfig(), bad); err == nil {
		t.Error("expected rescan error to propagate")
	}
}

func TestEvaluateRegulationSelection(t *testing.T) {
	cfg := testConfig()
	cfg.Regulations = []Regulation{GDPR}
	rep, err := Evaluate(context.Background(), cfg, goodRescanner())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(rep.Regulations) != 1 || rep.Regulations[0].Regulation != GDPR {
		t.Errorf("regulations = %+v", rep.Regulations)
	}
}

func TestAnalyzeStateClassification(t *testing.T) {
	cfg := testConfig()
	cfg.Settings.EssentialHosts = []string{"cdn.acme.com"}
	if err := cfg.Prepare(); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	obs := &Observation{
		URL:      "https://acme.test/",
		FinalURL: "https://acme.test/",
		Cookies:  []domain.Cookie{{Name: "sid"}},
		Tags:     []domain.Tag{{ID: "ga", Category: domain.TagAnalytics}},
		Requests: []domain.NetworkRequest{
			{URL: "https://acme.test/main.js", Host: "acme.test", ResourceType: "script"},
			{URL: "https://cdn.acme.com/lib.js", Host: "cdn.acme.com", ResourceType: "script"},
			{URL: "https://fonts.example.com/x.woff2", Host: "fonts.example.com", ResourceType: "font"},
			{URL: "https://ads.tracker.com/p.gif", Host: "ads.tracker.com", ResourceType: "image"},
			{URL: "https://px.tracker.com/beacon", Host: "px.tracker.com", ResourceType: "beacon"},
		},
	}
	sr := analyzeState(cfg, StateAcceptAll, obs)
	ids := trackerIDs(sr.TrackersFired)
	if len(ids) != 2 || ids[0] != "request:px.tracker.com" || ids[1] != "tag:ga" {
		t.Errorf("trackers fired = %v", ids)
	}
	if len(sr.CookiesSet) != 0 {
		t.Errorf("essential cookie should be excluded: %v", sr.CookiesSet)
	}
	if len(sr.ThirdPartyHosts) != 1 || sr.ThirdPartyHosts[0] != "px.tracker.com" {
		t.Errorf("third-party hosts = %v", sr.ThirdPartyHosts)
	}
}

func TestTrackThirdPartyDisabled(t *testing.T) {
	cfg := testConfig()
	off := false
	cfg.Settings.TrackThirdPartyRequests = &off
	obs := &Observation{
		URL:      "https://acme.test/",
		FinalURL: "https://acme.test/",
		Requests: []domain.NetworkRequest{{URL: "https://px.tracker.com/x", Host: "px.tracker.com", ResourceType: "script"}},
	}
	sr := analyzeState(cfg, StateAcceptAll, obs)
	if len(sr.TrackersFired) != 0 {
		t.Errorf("expected no trackers, got %v", trackerIDs(sr.TrackersFired))
	}
}

func findCheck(t *testing.T, rep *Report, reg Regulation, id string) Check {
	t.Helper()
	for _, rr := range rep.Regulations {
		if rr.Regulation != reg {
			continue
		}
		for _, c := range rr.Checks {
			if c.ID == id {
				return c
			}
		}
	}
	t.Fatalf("check %s in %s not found", id, reg)
	return Check{}
}
