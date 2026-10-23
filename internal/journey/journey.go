// Package journey implements the Pageguard journey tester (Module 4). A journey
// is an ordered list of browser steps (navigate, click, fill, wait). After each
// step the engine captures the full page state, validates it against the step's
// expectations (tags, data-layer events, cookies), and records the console and
// network changes since the previous step. The result is a step-by-step report
// with screenshots captured on failure.
//
// The package is organised around three seams:
//
//   - Journey is the declarative YAML/JSON document (this file, plus load.go).
//   - Session is a stateful headless-browser session (session.go); the real
//     implementation (playwright.go) drives the Node worker in
//     tools/browser-renderer, and a FakeSession keeps tests offline.
//   - Engine orchestrates a session against a Journey and produces a Report
//     (engine.go, report.go), which a Formatter renders.
package journey

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
)

// SchemaVersion identifies the Journey document schema. A change is a breaking
// change for journey authors.
const SchemaVersion = "1.0.0"

// Default timing used when a journey does not specify its own.
const (
	DefaultTimeoutMs = 20000
	DefaultWaitMs    = 500
)

// Journey is a scripted, multi-step user flow.
type Journey struct {
	Version     int            `json:"version" yaml:"version"`
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	BaseURL     string         `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Defaults    Defaults       `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Session     SessionOptions `json:"session,omitempty" yaml:"session,omitempty"`
	Steps       []Step         `json:"steps" yaml:"steps"`
}

// Defaults are per-journey timing defaults inherited by steps.
type Defaults struct {
	TimeoutMs int `json:"timeout_ms,omitempty" yaml:"timeout_ms,omitempty"`
	WaitMs    int `json:"wait_ms,omitempty" yaml:"wait_ms,omitempty"`
}

// SessionOptions configure the browser context a journey runs in.
type SessionOptions struct {
	UserAgent         string          `json:"user_agent,omitempty" yaml:"user_agent,omitempty"`
	Viewport          domain.Viewport `json:"viewport,omitempty" yaml:"viewport,omitempty"`
	Locale            string          `json:"locale,omitempty" yaml:"locale,omitempty"`
	Timezone          string          `json:"timezone,omitempty" yaml:"timezone,omitempty"`
	IgnoreHTTPSErrors bool            `json:"ignore_https_errors,omitempty" yaml:"ignore_https_errors,omitempty"`
	// CaptureDataLayer records window.dataLayer payloads after each step. It
	// defaults to true (journeys commonly assert on data-layer events).
	CaptureDataLayer *bool `json:"capture_data_layer,omitempty" yaml:"capture_data_layer,omitempty"`
}

// Step actions.
const (
	ActionNavigate = "navigate"
	ActionClick    = "click"
	ActionFill     = "fill"
	ActionType     = "type" // alias for fill
	ActionWait     = "wait"
)

// Step is one action within a journey plus the expectations to check after it.
type Step struct {
	// Action is one of navigate, click, fill (alias type) or wait.
	Action      string `json:"action" yaml:"action"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Selector is a Playwright (CSS) selector for click/fill/wait.
	Selector string `json:"selector,omitempty" yaml:"selector,omitempty"`
	// Value is the target URL for navigate or the input value for fill.
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
	// TimeoutMs overrides the journey/default timeout for this step.
	TimeoutMs int `json:"timeout_ms,omitempty" yaml:"timeout_ms,omitempty"`
	// WaitMs is an extra settle delay after the action, before capture.
	WaitMs int `json:"wait_ms,omitempty" yaml:"wait_ms,omitempty"`
	// Expect lists what must hold in the page state after the step.
	Expect Expect `json:"expect,omitempty" yaml:"expect,omitempty"`
}

// Expect describes the assertions checked against the page state after a step.
// All populated fields must hold.
type Expect struct {
	// URLContains asserts the final URL contains the substring.
	URLContains string `json:"url_contains,omitempty" yaml:"url_contains,omitempty"`
	// TitleContains asserts the page title contains the substring.
	TitleContains string `json:"title_contains,omitempty" yaml:"title_contains,omitempty"`
	// Status asserts the navigation status code (0 ignores it).
	Status int `json:"status,omitempty" yaml:"status,omitempty"`
	// Tags lists tag ids that must be detected on the page.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	// AbsentTags lists tag ids that must not be detected.
	AbsentTags []string `json:"absent_tags,omitempty" yaml:"absent_tags,omitempty"`
	// Events lists data-layer event names that must have fired by now.
	Events []string `json:"events,omitempty" yaml:"events,omitempty"`
	// Cookies lists cookie names that must be present.
	Cookies []string `json:"cookies,omitempty" yaml:"cookies,omitempty"`
	// AbsentCookies lists cookie names that must not be present.
	AbsentCookies []string `json:"absent_cookies,omitempty" yaml:"absent_cookies,omitempty"`
}

// empty reports whether the expectation makes no assertions.
func (e Expect) empty() bool {
	return e.URLContains == "" && e.TitleContains == "" && e.Status == 0 &&
		len(e.Tags) == 0 && len(e.AbsentTags) == 0 && len(e.Events) == 0 &&
		len(e.Cookies) == 0 && len(e.AbsentCookies) == 0
}

// Prepare normalises defaults and validates the journey. It is idempotent and
// is called automatically by the engine; callers loading a journey directly
// should also call it to surface authoring errors early.
func (j *Journey) Prepare() error {
	var errs []string
	if j.Version == 0 {
		j.Version = 1
	}
	if strings.TrimSpace(j.Name) == "" {
		errs = append(errs, "journey: name is required")
	}
	if len(j.Steps) == 0 {
		errs = append(errs, "journey: at least one step is required")
	}

	if j.Defaults.TimeoutMs <= 0 {
		j.Defaults.TimeoutMs = DefaultTimeoutMs
	}
	if j.Defaults.WaitMs < 0 {
		j.Defaults.WaitMs = 0
	}
	if j.Session.Viewport.Width == 0 {
		j.Session.Viewport.Width = 1366
	}
	if j.Session.Viewport.Height == 0 {
		j.Session.Viewport.Height = 768
	}

	base, baseErr := j.baseURL()
	if baseErr != nil {
		errs = append(errs, baseErr.Error())
	}

	for i := range j.Steps {
		s := &j.Steps[i]
		s.Action = strings.ToLower(strings.TrimSpace(s.Action))
		if s.TimeoutMs <= 0 {
			s.TimeoutMs = j.Defaults.TimeoutMs
		}
		if s.WaitMs < 0 {
			s.WaitMs = 0
		}
		switch s.Action {
		case ActionNavigate:
			if strings.TrimSpace(s.Value) == "" {
				errs = append(errs, fmt.Sprintf("journey: step %d (navigate) requires a value (URL)", i+1))
				break
			}
			if base != nil {
				if u, err := url.Parse(s.Value); err == nil && !u.IsAbs() && u.Host == "" {
					s.Value = base.ResolveReference(u).String()
				}
			}
		case ActionClick:
			if strings.TrimSpace(s.Selector) == "" {
				errs = append(errs, fmt.Sprintf("journey: step %d (click) requires a selector", i+1))
			}
		case ActionFill, ActionType:
			if strings.TrimSpace(s.Selector) == "" {
				errs = append(errs, fmt.Sprintf("journey: step %d (%s) requires a selector", i+1, s.Action))
			}
		case ActionWait:
			if strings.TrimSpace(s.Selector) == "" && s.TimeoutMs <= 0 {
				errs = append(errs, fmt.Sprintf("journey: step %d (wait) requires a selector or timeout_ms", i+1))
			}
		default:
			errs = append(errs, fmt.Sprintf("journey: step %d has unknown action %q", i+1, s.Action))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func (j *Journey) baseURL() (*url.URL, error) {
	if strings.TrimSpace(j.BaseURL) == "" {
		return nil, nil
	}
	u, err := url.Parse(j.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("journey: invalid base_url %q: %w", j.BaseURL, err)
	}
	return u, nil
}

// CaptureDataLayer reports whether data-layer capture is enabled (default true).
func (o SessionOptions) captureDataLayer() bool {
	return o.CaptureDataLayer == nil || *o.CaptureDataLayer
}

// CaptureDataLayer reports whether data-layer capture is enabled (default true).
func (j *Journey) CaptureDataLayer() bool { return j.Session.captureDataLayer() }
