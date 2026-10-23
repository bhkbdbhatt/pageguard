package privacy

import (
	"context"
	"errors"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func TestFakeRescanner(t *testing.T) {
	f := &FakeRescanner{ByState: map[ConsentState]*Observation{
		StateAcceptAll: {URL: "https://a.test", Tags: []domain.Tag{{ID: "x"}}},
	}}
	got, err := f.Rescan(context.Background(), StateAcceptAll)
	if err != nil {
		t.Fatalf("Rescan: %v", err)
	}
	if got.URL != "https://a.test" || len(got.Tags) != 1 {
		t.Errorf("unexpected observation: %+v", got)
	}
	if o, err := f.Rescan(context.Background(), StateGPC); err != nil || o == nil || len(o.Tags) != 0 {
		t.Errorf("expected empty observation for unconfigured state, got %+v (err %v)", o, err)
	}
}

func TestFakeRescannerError(t *testing.T) {
	f := &FakeRescanner{Err: errors.New("boom")}
	if _, err := f.Rescan(context.Background(), StateAcceptAll); err == nil {
		t.Fatal("expected error")
	}
}

func TestRescanFunc(t *testing.T) {
	var fn RescanFunc = func(_ context.Context, s ConsentState) (*Observation, error) {
		if s != StateGPC {
			t.Errorf("state = %s", s)
		}
		return &Observation{URL: "u"}, nil
	}
	o, err := fn.Rescan(context.Background(), StateGPC)
	if err != nil {
		t.Fatalf("Rescan: %v", err)
	}
	if o.URL != "u" {
		t.Errorf("url = %q", o.URL)
	}
}

func TestFromScanResult(t *testing.T) {
	sr := &domain.ScanResult{
		URL:      "https://a.test",
		FinalURL: "https://a.test/home",
		Tags:     []domain.Tag{{ID: "t"}},
		Cookies:  []domain.Cookie{{Name: "c"}},
		Requests: []domain.NetworkRequest{{URL: "https://b.test/x"}},
		Links:    []string{"https://a.test/opt-out"},
	}
	o := FromScanResult(sr)
	if o.URL != sr.URL || o.FinalURL != sr.FinalURL || len(o.Tags) != 1 || len(o.Links) != 1 {
		t.Errorf("adapter mismatch: %+v", o)
	}
	if empty := FromScanResult(nil); empty == nil {
		t.Fatal("expected non-nil observation for nil input")
	}
}
