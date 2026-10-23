package journey

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/scanner"
)

// RunOptions tune a journey run.
type RunOptions struct {
	// ScreenshotDir is where failure screenshots are written. When empty a
	// "pageguard-journey" directory under the OS temp directory is used.
	ScreenshotDir string
}

// Engine executes journeys against a browser session and validates each step.
type Engine struct {
	Factory  SessionFactory
	Detector *scanner.Detector
}

// NewEngine constructs an engine over the given session factory.
func NewEngine(factory SessionFactory) *Engine {
	return &Engine{Factory: factory, Detector: scanner.NewDetector()}
}

// stepState is the validated page state retained for diffing the next step.
type stepState struct {
	state  *PageState
	tags   []domain.Tag
	events []string
}

// Run executes every step of the journey sequentially in a single browser
// session, capturing and validating the page state after each step. It returns
// a Report even when steps fail; a non-nil error indicates the journey could
// not be started (bad definition or session open failure).
func (e *Engine) Run(ctx context.Context, j *Journey, opts RunOptions) (*Report, error) {
	if err := j.Prepare(); err != nil {
		return nil, err
	}
	if e.Factory == nil {
		return nil, fmt.Errorf("journey: no session factory configured")
	}
	detector := e.Detector
	if detector == nil {
		detector = scanner.NewDetector()
	}

	sess, err := e.Factory.Open(ctx, j.Session)
	if err != nil {
		return nil, fmt.Errorf("journey: open session: %w", err)
	}
	defer sess.Close()

	report := &Report{
		SchemaVersion: ReportSchemaVersion,
		Journey:       j.Name,
		StartedAt:     time.Now().UTC(),
	}

	var prev stepState
	for i := range j.Steps {
		step := j.Steps[i]
		sr := StepResult{
			Index:       i,
			Action:      step.Action,
			Selector:    step.Selector,
			Description: step.Description,
		}

		started := time.Now()
		st, perr := sess.Perform(ctx, step)
		sr.DurationMs = time.Since(started).Milliseconds()

		if perr != nil {
			sr.Status = StatusError
			sr.Error = perr.Error()
			sr.Message = fmt.Sprintf("%s failed: %v", step.Action, perr)
			captureScreenshot(ctx, sess, opts, &sr, i)
			report.Steps = append(report.Steps, sr)
			break // the session is no longer in a known-good state
		}

		tags := detector.Detect(observationFrom(st))
		events := dataLayerEvents(st.DataLayer)
		sr.Diff = diffState(prev, st, tags, events)
		sr.Checks = checkExpectations(step.Expect, st, tags, events)

		if msgs := failedChecks(sr.Checks); len(msgs) > 0 {
			sr.Status = StatusFail
			sr.Message = strings.Join(msgs, "; ")
			captureScreenshot(ctx, sess, opts, &sr, i)
		} else {
			sr.Status = StatusPass
			sr.Message = step.Action + " ok"
		}

		report.Steps = append(report.Steps, sr)
		prev = stepState{state: st, tags: tags, events: events}
	}

	report.FinishedAt = time.Now().UTC()
	report.DurationMs = report.FinishedAt.Sub(report.StartedAt).Milliseconds()
	report.finalize()
	return report, nil
}

func observationFrom(st *PageState) *domain.Observation {
	if st == nil {
		return &domain.Observation{}
	}
	return &domain.Observation{
		URL:         st.URL,
		FinalURL:    st.FinalURL,
		StatusCode:  st.Status,
		Title:       st.Title,
		HTML:        st.HTML,
		ContentType: "text/html",
		Requests:    st.Requests,
		Cookies:     st.Cookies,
		Console:     st.Console,
		DataLayer:   st.DataLayer,
		Rendered:    true,
	}
}

// dataLayerEvents extracts the top-level "event" name from every data-layer
// entry, de-duplicated and in order.
func dataLayerEvents(entries []domain.DataLayerEntry) []string {
	var out []string
	seen := map[string]bool{}
	for _, entry := range entries {
		v, ok := dataLayerObject(entry)
		if !ok {
			continue
		}
		ev, _ := v["event"].(string)
		ev = strings.TrimSpace(ev)
		if ev == "" || seen[strings.ToLower(ev)] {
			continue
		}
		seen[strings.ToLower(ev)] = true
		out = append(out, ev)
	}
	return out
}

// dataLayerObject decodes a data-layer entry into an object. The browser worker
// emits value as a JSON string (via JSON.stringify to guard against circular
// references), so a single unwrap is attempted when the decoded value is a
// string.
func dataLayerObject(entry domain.DataLayerEntry) (map[string]any, bool) {
	if len(entry.Value) == 0 {
		return nil, false
	}
	var v any
	if err := json.Unmarshal(entry.Value, &v); err != nil {
		return nil, false
	}
	if s, ok := v.(string); ok {
		var inner any
		if err := json.Unmarshal([]byte(s), &inner); err == nil {
			v = inner
		}
	}
	m, ok := v.(map[string]any)
	return m, ok
}

func diffState(prev stepState, cur *PageState, tags []domain.Tag, events []string) Diff {
	var d Diff

	seenReq := map[string]bool{}
	if prev.state != nil {
		for _, r := range prev.state.Requests {
			seenReq[r.Method+" "+r.URL] = true
		}
	}
	for _, r := range cur.Requests {
		if !seenReq[r.Method+" "+r.URL] {
			d.NewRequests = append(d.NewRequests, r)
		}
	}

	prevConsole := 0
	if prev.state != nil {
		prevConsole = len(prev.state.Console)
	}
	if len(cur.Console) > prevConsole {
		d.NewConsole = append([]domain.ConsoleMessage(nil), cur.Console[prevConsole:]...)
	}

	seenCookie := map[string]bool{}
	if prev.state != nil {
		for _, c := range prev.state.Cookies {
			seenCookie[c.Name] = true
		}
	}
	for _, c := range cur.Cookies {
		if !seenCookie[c.Name] {
			d.NewCookies = append(d.NewCookies, c)
		}
	}

	seenTag := map[string]bool{}
	for _, t := range prev.tags {
		seenTag[t.ID] = true
	}
	for _, t := range tags {
		if !seenTag[t.ID] {
			d.NewTags = append(d.NewTags, t)
		}
	}

	seenEvent := map[string]bool{}
	for _, e := range prev.events {
		seenEvent[strings.ToLower(e)] = true
	}
	for _, e := range events {
		if !seenEvent[strings.ToLower(e)] {
			d.NewEvents = append(d.NewEvents, e)
		}
	}

	return d
}

func checkExpectations(exp Expect, st *PageState, tags []domain.Tag, events []string) []CheckResult {
	var checks []CheckResult
	add := func(name string, ok bool, expected, actual any, msg string) {
		c := CheckResult{Name: name, Expected: expected, Actual: actual}
		if ok {
			c.Status = CheckPass
		} else {
			c.Status = CheckFail
			c.Message = msg
		}
		checks = append(checks, c)
	}

	tagIDs := map[string]bool{}
	for _, t := range tags {
		tagIDs[strings.ToLower(t.ID)] = true
	}
	cookieNames := map[string]bool{}
	for _, c := range st.Cookies {
		cookieNames[strings.ToLower(c.Name)] = true
	}
	eventSet := map[string]bool{}
	for _, e := range events {
		eventSet[strings.ToLower(e)] = true
	}

	if exp.URLContains != "" {
		actual := st.FinalURL
		if actual == "" {
			actual = st.URL
		}
		ok := strings.Contains(actual, exp.URLContains)
		add("url_contains", ok, exp.URLContains, actual,
			fmt.Sprintf("url %q does not contain %q", actual, exp.URLContains))
	}
	if exp.TitleContains != "" {
		ok := strings.Contains(st.Title, exp.TitleContains)
		add("title_contains", ok, exp.TitleContains, st.Title,
			fmt.Sprintf("title %q does not contain %q", st.Title, exp.TitleContains))
	}
	if exp.Status != 0 {
		ok := st.Status == exp.Status
		add("status", ok, exp.Status, st.Status,
			fmt.Sprintf("status %d, want %d", st.Status, exp.Status))
	}
	for _, id := range exp.Tags {
		ok := tagIDs[strings.ToLower(id)]
		add("tag:"+id, ok, "present", boolWord(ok), fmt.Sprintf("expected tag %q was not detected", id))
	}
	for _, id := range exp.AbsentTags {
		ok := !tagIDs[strings.ToLower(id)]
		add("absent_tag:"+id, ok, "absent", boolWord(!ok), fmt.Sprintf("tag %q is present but must not be", id))
	}
	for _, e := range exp.Events {
		ok := eventSet[strings.ToLower(e)]
		add("event:"+e, ok, "fired", boolWord(ok), fmt.Sprintf("expected data-layer event %q did not fire", e))
	}
	for _, name := range exp.Cookies {
		ok := cookieNames[strings.ToLower(name)]
		add("cookie:"+name, ok, "present", boolWord(ok), fmt.Sprintf("expected cookie %q was not set", name))
	}
	for _, name := range exp.AbsentCookies {
		ok := !cookieNames[strings.ToLower(name)]
		add("absent_cookie:"+name, ok, "absent", boolWord(!ok), fmt.Sprintf("cookie %q is present but must not be", name))
	}
	return checks
}

func failedChecks(checks []CheckResult) []string {
	var msgs []string
	for _, c := range checks {
		if c.Status == CheckFail {
			msgs = append(msgs, c.Message)
		}
	}
	return msgs
}

func boolWord(b bool) string {
	if b {
		return "present"
	}
	return "absent"
}

func captureScreenshot(ctx context.Context, sess Session, opts RunOptions, sr *StepResult, index int) {
	data, err := sess.Screenshot(ctx)
	if err != nil || len(data) == 0 {
		return
	}
	dir := opts.ScreenshotDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "pageguard-journey")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	name := fmt.Sprintf("step-%02d-%s.png", index+1, sanitize(sr.Action))
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		return
	}
	sr.Screenshot = filepath.Join(dir, name)
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '-'
		}
	}, s)
}
