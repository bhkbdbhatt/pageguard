package journey

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/pageguard/pageguard/internal/domain"
)

// PlaywrightSessionFactory opens browser sessions by driving the Pageguard
// browser worker (tools/browser-renderer) over newline-delimited JSON. One
// worker process is spawned per session and kept alive for the whole journey so
// cookies, storage and the DOM persist across steps.
//
// Protocol (commands the worker understands):
//
//	{"cmd":"open","opts":{...}}                 -> {"ok":true}
//	{"cmd":"step","step":{...}}                 -> {"ok":true,"state":{...}}
//	{"cmd":"screenshot"}                        -> {"ok":true,"data":"<base64 png>"}
//	{"cmd":"close"}                             -> {"ok":true}
type PlaywrightSessionFactory struct {
	// Command is the worker invocation, e.g.
	// []string{"node", "tools/browser-renderer/index.mjs"}.
	Command []string
}

// NewPlaywrightSessionFactory constructs a factory for the given command.
func NewPlaywrightSessionFactory(command ...string) *PlaywrightSessionFactory {
	return &PlaywrightSessionFactory{Command: command}
}

// Open starts a worker process and opens a browser session in it.
func (f *PlaywrightSessionFactory) Open(ctx context.Context, opts SessionOptions) (Session, error) {
	if len(f.Command) == 0 {
		return nil, fmt.Errorf("journey: browser worker command not configured")
	}
	cmd := exec.Command(f.Command[0], f.Command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 1<<20), 32<<20)

	s := &playwrightSession{cmd: cmd, stdin: stdin, stdout: sc}
	if err := s.call(ctx, workerRequest{Cmd: "open", Opts: &workerSessionOpts{
		UserAgent:         opts.UserAgent,
		Width:             opts.Viewport.Width,
		Height:            opts.Viewport.Height,
		Locale:            opts.Locale,
		Timezone:          opts.Timezone,
		IgnoreHTTPSErrors: opts.IgnoreHTTPSErrors,
		CaptureDataLayer:  opts.captureDataLayer(),
	}}); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

type playwrightSession struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	closed bool
}

type workerRequest struct {
	Cmd  string             `json:"cmd"`
	Opts *workerSessionOpts `json:"opts,omitempty"`
	Step *workerStep        `json:"step,omitempty"`
}

type workerSessionOpts struct {
	UserAgent         string `json:"userAgent,omitempty"`
	Width             int    `json:"width,omitempty"`
	Height            int    `json:"height,omitempty"`
	Locale            string `json:"locale,omitempty"`
	Timezone          string `json:"timezone,omitempty"`
	IgnoreHTTPSErrors bool   `json:"ignoreHTTPSErrors,omitempty"`
	CaptureDataLayer  bool   `json:"captureDataLayer,omitempty"`
}

type workerStep struct {
	Action    string `json:"action"`
	Selector  string `json:"selector,omitempty"`
	Value     string `json:"value,omitempty"`
	TimeoutMs int    `json:"timeoutMs,omitempty"`
	WaitMs    int    `json:"waitMs,omitempty"`
}

type workerResponse struct {
	OK    bool         `json:"ok"`
	Error string       `json:"error,omitempty"`
	State *workerState `json:"state,omitempty"`
	Data  string       `json:"data,omitempty"`
}

type workerState struct {
	URL       string                  `json:"url"`
	FinalURL  string                  `json:"finalUrl"`
	Status    int                     `json:"status"`
	Title     string                  `json:"title"`
	HTML      string                  `json:"html"`
	Requests  []domain.NetworkRequest `json:"requests"`
	Cookies   []domain.Cookie         `json:"cookies"`
	Console   []domain.ConsoleMessage `json:"console"`
	DataLayer []domain.DataLayerEntry `json:"dataLayer"`
	Duration  int64                   `json:"durationMs"`
}

// Perform sends one step to the worker and decodes the resulting page state.
func (s *playwrightSession) Perform(ctx context.Context, step Step) (*PageState, error) {
	resp, err := s.roundTrip(ctx, workerRequest{Cmd: "step", Step: &workerStep{
		Action:    step.Action,
		Selector:  step.Selector,
		Value:     step.Value,
		TimeoutMs: step.TimeoutMs,
		WaitMs:    step.WaitMs,
	}})
	if err != nil {
		return nil, err
	}
	if resp.State == nil {
		return &PageState{}, nil
	}
	st := resp.State
	return &PageState{
		URL: st.URL, FinalURL: st.FinalURL, Status: st.Status, Title: st.Title,
		HTML: st.HTML, Requests: st.Requests, Cookies: st.Cookies,
		Console: st.Console, DataLayer: st.DataLayer, DurationMs: st.Duration,
	}, nil
}

// Screenshot captures the current page as PNG bytes.
func (s *playwrightSession) Screenshot(ctx context.Context) ([]byte, error) {
	resp, err := s.roundTrip(ctx, workerRequest{Cmd: "screenshot"})
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(resp.Data)
}

// Close asks the worker to release the session and then kills the process.
func (s *playwrightSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.stdin != nil {
		// Best-effort graceful close; ignore errors if the worker is gone.
		payload, _ := json.Marshal(workerRequest{Cmd: "close"})
		_, _ = s.stdin.Write(append(payload, '\n'))
		_ = s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
	return nil
}

func (s *playwrightSession) call(ctx context.Context, req workerRequest) error {
	_, err := s.roundTrip(ctx, req)
	return err
}

func (s *playwrightSession) roundTrip(ctx context.Context, req workerRequest) (*workerResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, fmt.Errorf("journey: session is closed")
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	type result struct {
		resp *workerResponse
		err  error
	}
	done := make(chan result, 1)
	go func() {
		if _, werr := s.stdin.Write(append(payload, '\n')); werr != nil {
			done <- result{nil, werr}
			return
		}
		if !s.stdout.Scan() {
			done <- result{nil, fmt.Errorf("journey: browser worker closed unexpectedly: %w", s.stdout.Err())}
			return
		}
		var resp workerResponse
		if err := json.Unmarshal(s.stdout.Bytes(), &resp); err != nil {
			done <- result{nil, err}
			return
		}
		if !resp.OK {
			done <- result{nil, fmt.Errorf("journey: browser worker error: %s", resp.Error)}
			return
		}
		done <- result{&resp, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-done:
		return r.resp, r.err
	}
}

// compile-time interface check.
var _ SessionFactory = (*PlaywrightSessionFactory)(nil)
