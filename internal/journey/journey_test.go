package journey

import (
	"strings"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func TestPrepareDefaults(t *testing.T) {
	j := &Journey{Name: "x", Steps: []Step{{Action: ActionNavigate, Value: "https://a.test/"}}}
	if err := j.Prepare(); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if j.Version != 1 {
		t.Errorf("version = %d, want 1", j.Version)
	}
	if j.Defaults.TimeoutMs != DefaultTimeoutMs {
		t.Errorf("timeout = %d, want %d", j.Defaults.TimeoutMs, DefaultTimeoutMs)
	}
	if j.Session.Viewport != (domain.Viewport{Width: 1366, Height: 768}) {
		t.Errorf("viewport = %+v", j.Session.Viewport)
	}
	if !j.CaptureDataLayer() {
		t.Error("data-layer capture should default to true")
	}
}

func TestPrepareCaptureDataLayerOverride(t *testing.T) {
	no := false
	j := &Journey{
		Name:    "x",
		Session: SessionOptions{CaptureDataLayer: &no},
		Steps:   []Step{{Action: ActionNavigate, Value: "https://a.test/"}},
	}
	if err := j.Prepare(); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if j.CaptureDataLayer() {
		t.Error("expected capture disabled")
	}
}

func TestPrepareValidation(t *testing.T) {
	cases := map[string]Journey{
		"missing name": {Steps: []Step{{Action: ActionWait, Selector: "#a"}}},
		"no steps":     {Name: "x"},
		"unknown action": {Name: "x", Steps: []Step{
			{Action: "scroll"},
		}},
		"navigate no value": {Name: "x", Steps: []Step{
			{Action: ActionNavigate},
		}},
		"click no selector": {Name: "x", Steps: []Step{
			{Action: ActionClick},
		}},
		"fill no selector": {Name: "x", Steps: []Step{
			{Action: ActionFill, Value: "v"},
		}},
		"bad base url": {Name: "x", BaseURL: "://bad", Steps: []Step{
			{Action: ActionNavigate, Value: "https://a.test/"},
		}},
	}
	for name, j := range cases {
		t.Run(name, func(t *testing.T) {
			if err := j.Prepare(); err == nil {
				t.Fatalf("expected error for %q", name)
			}
		})
	}
}

func TestPrepareBareWaitIsValid(t *testing.T) {
	j := &Journey{Name: "x", Steps: []Step{{Action: ActionWait}}}
	if err := j.Prepare(); err != nil {
		t.Fatalf("bare wait should be valid: %v", err)
	}
}

func TestPrepareResolvesRelativeNavigate(t *testing.T) {
	j := &Journey{
		Name:    "x",
		BaseURL: "https://acme.com/store/",
		Steps:   []Step{{Action: ActionNavigate, Value: "../cart"}, {Action: ActionNavigate, Value: "https://other.test/"}},
	}
	if err := j.Prepare(); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if got := j.Steps[0].Value; got != "https://acme.com/cart" {
		t.Errorf("relative resolve = %q", got)
	}
	if got := j.Steps[1].Value; got != "https://other.test/" {
		t.Errorf("absolute unchanged = %q", got)
	}
}

func TestPrepareTypeAliasValid(t *testing.T) {
	j := &Journey{Name: "x", Steps: []Step{{Action: ActionType, Selector: "#e", Value: "v"}}}
	if err := j.Prepare(); err != nil {
		t.Fatalf("type alias should be valid: %v", err)
	}
}

func TestPrepareMultipleErrorsAggregated(t *testing.T) {
	j := &Journey{Steps: []Step{{Action: ActionClick}, {Action: "bogus"}}}
	err := j.Prepare()
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "name is required") || !strings.Contains(msg, "requires a selector") || !strings.Contains(msg, "unknown action") {
		t.Fatalf("errors not aggregated: %q", msg)
	}
}
