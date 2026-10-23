package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseProxiesRegionInheritance(t *testing.T) {
	got := ParseProxies([]string{"us=http://a:1", "http://b:2", " eu = http://c:3 ", "http://d:4"})
	want := []Proxy{
		{URL: "http://a:1", Region: "us"},
		{URL: "http://b:2", Region: "us"},
		{URL: "http://c:3", Region: "eu"},
		{URL: "http://d:4", Region: "eu"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d proxies, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("proxy[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestProxyHelpers(t *testing.T) {
	p := Proxy{URL: "http://user:secret@host:8080", Region: "us"}
	if p.Host() != "host:8080" {
		t.Errorf("Host() = %q", p.Host())
	}
	if p.Server() != "http://host:8080" {
		t.Errorf("Server() = %q", p.Server())
	}
	u, pw, ok := p.Credentials()
	if !ok || u != "user" || pw != "secret" {
		t.Errorf("Credentials() = %q,%q,%v", u, pw, ok)
	}
}

func TestRotatorRoundRobin(t *testing.T) {
	r := New([]Proxy{
		{URL: "http://a:1", Region: "us"},
		{URL: "http://b:2", Region: "us"},
	})
	seen := map[string]int{}
	for i := 0; i < 4; i++ {
		p, ok := r.Next("us")
		if !ok {
			t.Fatal("expected a proxy")
		}
		seen[p.URL]++
	}
	if seen["http://a:1"] != 2 || seen["http://b:2"] != 2 {
		t.Errorf("round-robin distribution wrong: %+v", seen)
	}
}

func TestRotatorFallsBackToRegionless(t *testing.T) {
	r := New([]Proxy{{URL: "http://any:1"}})
	p, ok := r.Next("us")
	if !ok || p.URL != "http://any:1" {
		t.Fatalf("Next(us) = %+v, %v", p, ok)
	}
}

func TestRotatorCooldownAndRecovery(t *testing.T) {
	now := time.Now()
	r := New(
		[]Proxy{{URL: "http://a:1", Region: "us"}},
		WithFailureThreshold(1),
		WithCooldown(time.Minute),
		WithClock(func() time.Time { return now }),
	)
	p, ok := r.Next("us")
	if !ok {
		t.Fatal("expected a proxy")
	}
	r.Report(p, errors.New("boom"))
	if _, ok := r.Next("us"); ok {
		t.Fatal("expected proxy to be in cooldown")
	}
	now = now.Add(2 * time.Minute)
	if _, ok := r.Next("us"); !ok {
		t.Fatal("expected proxy to recover after cooldown")
	}
}

func TestRotatorReportSuccessClearsFailures(t *testing.T) {
	r := New([]Proxy{{URL: "http://a:1", Region: "us"}}, WithFailureThreshold(2))
	p, _ := r.Next("us")
	r.Report(p, errors.New("transient"))
	r.Report(p, nil)
	r.Report(p, errors.New("transient"))
	if _, ok := r.Next("us"); !ok {
		t.Fatal("success should have reset the failure count")
	}
}

func TestRoundTripperDirectWhenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	rt := &RoundTripper{Rotator: New(nil)}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestRoundTripperErrorsThroughDeadProxy(t *testing.T) {
	rt := &RoundTripper{
		Rotator:     New([]Proxy{{URL: "http://127.0.0.1:1", Region: "us"}}),
		MaxAttempts: 1,
	}
	req, _ := http.NewRequestWithContext(WithRegion(context.Background(), "us"), http.MethodGet, "http://example.com/", nil)
	if _, err := rt.RoundTrip(req); err == nil {
		t.Fatal("expected error routing through a dead proxy")
	}
}
