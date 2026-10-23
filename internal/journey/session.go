package journey

import (
	"context"
	"sync"

	"github.com/pageguard/pageguard/internal/domain"
)

// PageState is the complete, serialisable snapshot of the browser page after a
// step. Requests and Console are cumulative for the session; DataLayer is the
// current contents of the client-side data layer.
type PageState struct {
	URL        string                  `json:"url"`
	FinalURL   string                  `json:"finalUrl"`
	Status     int                     `json:"status"`
	Title      string                  `json:"title"`
	HTML       string                  `json:"html"`
	Requests   []domain.NetworkRequest `json:"requests"`
	Cookies    []domain.Cookie         `json:"cookies"`
	Console    []domain.ConsoleMessage `json:"console"`
	DataLayer  []domain.DataLayerEntry `json:"dataLayer"`
	DurationMs int64                   `json:"durationMs"`
	Screenshot string                  `json:"-"`
}

// Session is a stateful headless-browser session. A single session is reused
// across every step of a journey so cookies, storage and the DOM persist.
type Session interface {
	// Perform executes one step and returns the resulting page state.
	Perform(ctx context.Context, step Step) (*PageState, error)
	// Screenshot captures the current viewport as PNG bytes.
	Screenshot(ctx context.Context) ([]byte, error)
	// Close tears the session down.
	Close() error
}

// SessionFactory opens browser sessions with the given options.
type SessionFactory interface {
	Open(ctx context.Context, opts SessionOptions) (Session, error)
}

// FakeSession is a scriptable Session for tests. It returns States in order for
// successive Perform calls; Errors lets a specific call fail.
type FakeSession struct {
	States         []*PageState
	Errors         map[int]error
	ScreenshotData []byte

	mu        sync.Mutex
	idx       int
	performed []Step
	closed    bool
}

// Perform returns the next scripted state (or error) and records the step.
func (f *FakeSession) Perform(_ context.Context, step Step) (*PageState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.performed = append(f.performed, step)
	i := f.idx
	f.idx++
	if err, ok := f.Errors[i]; ok {
		return nil, err
	}
	if i < len(f.States) && f.States[i] != nil {
		return f.States[i], nil
	}
	return &PageState{}, nil
}

// Screenshot returns the configured screenshot bytes (default "PNG").
func (f *FakeSession) Screenshot(context.Context) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ScreenshotData != nil {
		return f.ScreenshotData, nil
	}
	return []byte("PNG"), nil
}

// Close marks the session closed.
func (f *FakeSession) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// Performed returns the steps passed to Perform, in order.
func (f *FakeSession) Performed() []Step {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Step(nil), f.performed...)
}

// Closed reports whether Close was called.
func (f *FakeSession) Closed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// FakeFactory opens a single FakeSession and records the open options.
type FakeFactory struct {
	Session *FakeSession
	// OpenErr, when set, is returned from Open.
	OpenErr error

	mu       sync.Mutex
	opened   int
	lastOpts SessionOptions
}

// Open returns the configured session.
func (f *FakeFactory) Open(_ context.Context, opts SessionOptions) (Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opened++
	f.lastOpts = opts
	if f.OpenErr != nil {
		return nil, f.OpenErr
	}
	if f.Session == nil {
		f.Session = &FakeSession{}
	}
	return f.Session, nil
}

// Opened returns how many times Open was called.
func (f *FakeFactory) Opened() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.opened
}

// LastOpts returns the options passed to the most recent Open.
func (f *FakeFactory) LastOpts() SessionOptions {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastOpts
}
