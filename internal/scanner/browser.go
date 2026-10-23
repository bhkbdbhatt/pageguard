package scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// BrowserRenderer renders pages by delegating to an external headless-browser
// worker (the Playwright/Puppeteer adapter lives in tools/browser-renderer).
// The worker speaks newline-delimited JSON over stdin/stdout:
//
//	request : {"url":"...","opts":{...}}
//	response: {"ok":true,"result":{...}}  or  {"ok":false,"error":"..."}
//
// A single worker process is reused; calls are serialised with a mutex. This
// keeps the foundation dependency-free while providing a real browser path.
type BrowserRenderer struct {
	Command []string // e.g. []string{"node", "tools/browser-renderer/index.mjs"}

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
}

type browserRequest struct {
	URL  string            `json:"url"`
	Opts browserRenderOpts `json:"opts"`
}

type browserRenderOpts struct {
	UserAgent        string  `json:"userAgent,omitempty"`
	TimeoutMs        int     `json:"timeoutMs,omitempty"`
	WaitMs           int     `json:"waitMs,omitempty"`
	Width            int     `json:"width,omitempty"`
	Height           int     `json:"height,omitempty"`
	Geolocation      string  `json:"geolocation,omitempty"`
	GeoLatitude      float64 `json:"geoLatitude,omitempty"`
	GeoLongitude     float64 `json:"geoLongitude,omitempty"`
	GeoAccuracy      float64 `json:"geoAccuracy,omitempty"`
	Locale           string  `json:"locale,omitempty"`
	Timezone         string  `json:"timezone,omitempty"`
	ProxyServer      string  `json:"proxyServer,omitempty"`
	ProxyUser        string  `json:"proxyUser,omitempty"`
	ProxyPass        string  `json:"proxyPass,omitempty"`
	CaptureHAR       bool    `json:"captureHar,omitempty"`
	CaptureDataLayer bool    `json:"captureDataLayer,omitempty"`
}

type browserResponse struct {
	OK     bool           `json:"ok"`
	Error  string         `json:"error,omitempty"`
	Result *browserResult `json:"result,omitempty"`
}

type browserResult struct {
	FinalURL    string                  `json:"finalUrl"`
	Status      int                     `json:"status"`
	ContentType string                  `json:"contentType"`
	HTML        string                  `json:"html"`
	Title       string                  `json:"title"`
	Requests    []domain.NetworkRequest `json:"requests"`
	Cookies     []domain.Cookie         `json:"cookies"`
	Console     []domain.ConsoleMessage `json:"console"`
	DataLayer   []domain.DataLayerEntry `json:"dataLayer"`
	HAR         json.RawMessage         `json:"har,omitempty"`
}

// NewBrowserRenderer constructs a renderer that spawns the given command lazily.
func NewBrowserRenderer(command ...string) *BrowserRenderer {
	return &BrowserRenderer{Command: command}
}

func (b *BrowserRenderer) ensureStarted() error {
	if b.cmd != nil {
		return nil
	}
	if len(b.Command) == 0 {
		return fmt.Errorf("scanner: browser renderer command not configured")
	}
	cmd := exec.Command(b.Command[0], b.Command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return err
	}
	b.cmd = cmd
	b.stdin = stdin
	b.stdout = bufio.NewScanner(stdout)
	b.stdout.Buffer(make([]byte, 0, 1<<20), 16<<20)
	return nil
}

// Render delegates one page render to the browser worker.
func (b *BrowserRenderer) Render(ctx context.Context, rawURL string, opts RenderOptions) (*RenderResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.ensureStarted(); err != nil {
		return nil, err
	}

	req := browserRequest{URL: rawURL, Opts: browserRenderOpts{
		UserAgent:        opts.UserAgent,
		TimeoutMs:        int(opts.Timeout / time.Millisecond),
		WaitMs:           opts.WaitMs,
		Width:            opts.Viewport.Width,
		Height:           opts.Viewport.Height,
		Geolocation:      opts.Geolocation,
		GeoLatitude:      opts.GeoLatitude,
		GeoLongitude:     opts.GeoLongitude,
		GeoAccuracy:      opts.GeoAccuracy,
		Locale:           opts.Locale,
		Timezone:         opts.Timezone,
		ProxyServer:      opts.ProxyServer,
		ProxyUser:        opts.ProxyUser,
		ProxyPass:        opts.ProxyPass,
		CaptureHAR:       opts.CaptureHAR,
		CaptureDataLayer: opts.CaptureDataLayer,
	}}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	type result struct {
		res *RenderResult
		err error
	}
	done := make(chan result, 1)
	go func() {
		if _, werr := b.stdin.Write(append(payload, '\n')); werr != nil {
			done <- result{nil, werr}
			return
		}
		if !b.stdout.Scan() {
			done <- result{nil, fmt.Errorf("scanner: browser worker closed unexpectedly: %w", b.stdout.Err())}
			return
		}
		var resp browserResponse
		if err := json.Unmarshal(b.stdout.Bytes(), &resp); err != nil {
			done <- result{nil, err}
			return
		}
		if !resp.OK || resp.Result == nil {
			done <- result{nil, fmt.Errorf("scanner: browser worker error: %s", resp.Error)}
			return
		}
		r := resp.Result
		done <- result{&RenderResult{
			FinalURL: r.FinalURL, Status: r.Status, HTML: r.HTML, Title: r.Title,
			ContentType: r.ContentType, Requests: r.Requests, Cookies: r.Cookies,
			Console: r.Console, DataLayer: r.DataLayer, HAR: r.HAR,
		}, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-done:
		return res.res, res.err
	}
}

// Close terminates the worker process.
func (b *BrowserRenderer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stdin != nil {
		_ = b.stdin.Close()
	}
	if b.cmd != nil && b.cmd.Process != nil {
		_ = b.cmd.Process.Kill()
		_, _ = b.cmd.Process.Wait()
	}
	b.cmd, b.stdin, b.stdout = nil, nil, nil
	return nil
}
