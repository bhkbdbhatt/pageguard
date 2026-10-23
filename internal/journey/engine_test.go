package journey

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func req(url, host string) domain.NetworkRequest {
	return domain.NetworkRequest{Method: "GET", URL: url, Host: host, Status: 200, ResourceType: "script"}
}

func dlEvent(event string) domain.DataLayerEntry {
	return domain.DataLayerEntry{Name: "dataLayer[0]", Value: json.RawMessage(`{"event":"` + event + `"}`)}
}

func baseJourney() *Journey {
	return &Journey{
		Name: "checkout",
		Steps: []Step{
			{Action: ActionNavigate, Value: "https://acme.test/", Expect: Expect{
				Status: 200, Tags: []string{"meta-pixel"},
				Events: []string{"page_view"}, Cookies: []string{"sid"},
			}},
			{Action: ActionClick, Selector: "#buy", Expect: Expect{
				URLContains: "/product/", Events: []string{"add_to_cart"},
			}},
		},
	}
}

func TestEngineRunPass(t *testing.T) {
	factory := &FakeFactory{Session: &FakeSession{States: []*PageState{
		{
			URL: "https://acme.test/", FinalURL: "https://acme.test/", Status: 200,
			Requests:  []domain.NetworkRequest{req("https://connect.facebook.net/en_US/fbevents.js", "connect.facebook.net")},
			Cookies:   []domain.Cookie{{Name: "sid"}},
			DataLayer: []domain.DataLayerEntry{dlEvent("page_view")},
		},
		{
			URL: "https://acme.test/", FinalURL: "https://acme.test/product/1", Status: 200,
			Requests:  []domain.NetworkRequest{req("https://connect.facebook.net/en_US/fbevents.js", "connect.facebook.net")},
			Cookies:   []domain.Cookie{{Name: "sid"}},
			DataLayer: []domain.DataLayerEntry{dlEvent("page_view"), dlEvent("add_to_cart")},
		},
	}}}

	rep, err := NewEngine(factory).Run(context.Background(), baseJourney(), RunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.Passed {
		t.Fatalf("expected pass, report=%+v", rep)
	}
	if rep.Summary.Steps != 2 || rep.Summary.Passed != 2 || rep.Summary.Failed != 0 {
		t.Errorf("summary = %+v", rep.Summary)
	}
	if !factory.Session.Closed() {
		t.Error("session not closed")
	}
	if factory.Opened() != 1 {
		t.Errorf("opened %d times", factory.Opened())
	}
}

func TestEngineRunFailChecks(t *testing.T) {
	dir := t.TempDir()
	factory := &FakeFactory{Session: &FakeSession{States: []*PageState{
		{URL: "https://acme.test/", FinalURL: "https://acme.test/", Status: 200}, // nothing matches
	}}}
	j := baseJourney()
	j.Steps = j.Steps[:1]

	rep, err := NewEngine(factory).Run(context.Background(), j, RunOptions{ScreenshotDir: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Passed {
		t.Fatal("expected failure")
	}
	sr := rep.Steps[0]
	if sr.Status != StatusFail {
		t.Fatalf("status = %s", sr.Status)
	}
	if len(sr.Checks) == 0 {
		t.Fatal("expected checks")
	}
	// A screenshot should have been captured on failure.
	if sr.Screenshot == "" {
		t.Fatal("expected screenshot path")
	}
	if _, err := os.Stat(sr.Screenshot); err != nil {
		t.Errorf("screenshot not written: %v", err)
	}
}

func TestEngineErrorStops(t *testing.T) {
	factory := &FakeFactory{Session: &FakeSession{
		Errors: map[int]error{0: errors.New("navigation timeout")},
	}}
	rep, err := NewEngine(factory).Run(context.Background(), baseJourney(), RunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rep.Steps) != 1 {
		t.Fatalf("expected execution to stop after error, steps=%d", len(rep.Steps))
	}
	if rep.Steps[0].Status != StatusError {
		t.Errorf("status = %s, want error", rep.Steps[0].Status)
	}
	if rep.Passed {
		t.Error("report should not pass")
	}
}

func TestEngineOpenError(t *testing.T) {
	factory := &FakeFactory{OpenErr: errors.New("no browser")}
	_, err := NewEngine(factory).Run(context.Background(), baseJourney(), RunOptions{})
	if err == nil {
		t.Fatal("expected open error")
	}
}

func TestEngineDiff(t *testing.T) {
	factory := &FakeFactory{Session: &FakeSession{States: []*PageState{
		{
			URL: "https://acme.test/", FinalURL: "https://acme.test/", Status: 200,
			Requests:  []domain.NetworkRequest{req("https://connect.facebook.net/a.js", "connect.facebook.net")},
			Cookies:   []domain.Cookie{{Name: "sid"}},
			DataLayer: []domain.DataLayerEntry{dlEvent("page_view")},
		},
		{
			URL: "https://acme.test/", FinalURL: "https://acme.test/product/1", Status: 200,
			Requests: []domain.NetworkRequest{
				req("https://connect.facebook.net/a.js", "connect.facebook.net"),
				req("https://www.googletagmanager.com/gtm.js", "www.googletagmanager.com"),
			},
			Cookies:   []domain.Cookie{{Name: "sid"}, {Name: "cart"}},
			Console:   []domain.ConsoleMessage{{Level: "log", Text: "hi"}},
			DataLayer: []domain.DataLayerEntry{dlEvent("page_view"), dlEvent("add_to_cart")},
		},
	}}}

	rep, err := NewEngine(factory).Run(context.Background(), baseJourney(), RunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	d := rep.Steps[1].Diff
	if len(d.NewRequests) != 1 || d.NewRequests[0].Host != "www.googletagmanager.com" {
		t.Errorf("newRequests = %+v", d.NewRequests)
	}
	if len(d.NewCookies) != 1 || d.NewCookies[0].Name != "cart" {
		t.Errorf("newCookies = %+v", d.NewCookies)
	}
	if len(d.NewConsole) != 1 {
		t.Errorf("newConsole = %+v", d.NewConsole)
	}
	if len(d.NewEvents) != 1 || d.NewEvents[0] != "add_to_cart" {
		t.Errorf("newEvents = %+v", d.NewEvents)
	}
	foundGTM := false
	for _, tag := range d.NewTags {
		if tag.ID == "google-tag-manager" {
			foundGTM = true
		}
	}
	if !foundGTM {
		t.Errorf("expected new tag google-tag-manager, got %+v", d.NewTags)
	}
}

func TestDataLayerEvents(t *testing.T) {
	entries := []domain.DataLayerEntry{
		{Name: "a", Value: json.RawMessage(`{"event":"page_view"}`)},
		{Name: "b", Value: json.RawMessage(`{"event":"page_view"}`)},
		{Name: "c", Value: json.RawMessage(`{"foo":1}`)},
		{Name: "d", Value: json.RawMessage(`not-json`)},
		{Name: "e", Value: json.RawMessage(`{"event":"purchase"}`)},
		// The browser worker emits the payload as a JSON string; it must unwrap.
		{Name: "f", Value: json.RawMessage(`"{\"event\":\"add_to_cart\"}"`)},
	}
	got := dataLayerEvents(entries)
	want := []string{"page_view", "purchase", "add_to_cart"}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}

func TestCheckExpectations(t *testing.T) {
	st := &PageState{
		URL: "https://acme.test/", FinalURL: "https://acme.test/cart", Status: 201, Title: "Acme Cart",
		Cookies: []domain.Cookie{{Name: "sid"}, {Name: "legacy"}},
	}
	tags := []domain.Tag{{ID: "meta-pixel"}, {ID: "google-tag-manager"}}
	events := []string{"page_view", "add_to_cart"}

	checks := checkExpectations(Expect{
		URLContains: "/cart", TitleContains: "Acme", Status: 201,
		Tags: []string{"meta-pixel"}, AbsentTags: []string{"hotjar"},
		Events: []string{"add_to_cart"}, Cookies: []string{"sid"}, AbsentCookies: []string{"legacy"},
	}, st, tags, events)

	// absent_cookie:legacy should fail (cookie is present); everything else passes.
	for _, c := range checks {
		want := CheckPass
		if c.Name == "absent_cookie:legacy" {
			want = CheckFail
		}
		if c.Status != want {
			t.Errorf("check %s = %s, want %s (%s)", c.Name, c.Status, want, c.Message)
		}
	}
	if len(failedChecks(checks)) != 1 {
		t.Errorf("failed = %v", failedChecks(checks))
	}
}

func TestNoFactoryConfigured(t *testing.T) {
	if _, err := (&Engine{}).Run(context.Background(), baseJourney(), RunOptions{}); err == nil {
		t.Fatal("expected error when factory is nil")
	}
}
