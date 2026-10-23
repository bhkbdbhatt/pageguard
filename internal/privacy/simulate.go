package privacy

import (
	"context"

	"github.com/pageguard/pageguard/internal/domain"
)

// Observation is the page state captured when a page is visited under a given
// consent state. It intentionally mirrors the fields the privacy checks need
// rather than the full scanner Observation, so it can be produced by a real
// browser session, a one-shot render, or a test fake.
type Observation struct {
	URL      string `json:"url"`
	FinalURL string `json:"finalUrl,omitempty"`

	// BannerPresent reports whether a cookie/consent banner was rendered.
	BannerPresent bool `json:"bannerPresent"`
	// BannerHasReject / BannerHasSettings report whether the banner offers a
	// reject-all and a granular settings option.
	BannerHasReject   bool `json:"bannerHasReject"`
	BannerHasSettings bool `json:"bannerHasSettings"`
	// GPCRespected reports whether the page honoured a Global Privacy Control
	// signal (e.g. via navigator.globalPrivacyControl).
	GPCRespected bool `json:"gpcRespected"`

	Tags      []domain.Tag            `json:"tags"`
	Cookies   []domain.Cookie         `json:"cookies"`
	Requests  []domain.NetworkRequest `json:"requests"`
	DataLayer []domain.DataLayerEntry `json:"dataLayer,omitempty"`
	// Links are the hrefs discovered in the document (for opt-out / rights
	// discovery).
	Links []string `json:"links,omitempty"`
}

// FromScanResult adapts a scanner result into an Observation. Banner and GPC
// signals are not part of a scan; callers that have them should set the fields
// afterwards.
func FromScanResult(sr *domain.ScanResult) *Observation {
	if sr == nil {
		return &Observation{}
	}
	return &Observation{
		URL:       sr.URL,
		FinalURL:  sr.FinalURL,
		Tags:      sr.Tags,
		Cookies:   sr.Cookies,
		Requests:  sr.Requests,
		DataLayer: sr.DataLayer,
		Links:     sr.Links,
	}
}

// Rescanner re-renders a page under a given consent state and returns the
// resulting observation. Implementations drive a real browser session and
// inject the consent choice; tests use FakeRescanner.
type Rescanner interface {
	Rescan(ctx context.Context, state ConsentState) (*Observation, error)
}

// RescanFunc adapts a function to the Rescanner interface.
type RescanFunc func(ctx context.Context, state ConsentState) (*Observation, error)

// Rescan calls f.
func (f RescanFunc) Rescan(ctx context.Context, state ConsentState) (*Observation, error) {
	return f(ctx, state)
}

// FakeRescanner returns a scripted Observation per consent state.
type FakeRescanner struct {
	ByState map[ConsentState]*Observation
	// Err, when set, is returned from every Rescan.
	Err error
}

// Rescan returns the observation configured for the state (or an empty one).
func (f *FakeRescanner) Rescan(_ context.Context, state ConsentState) (*Observation, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if o, ok := f.ByState[state]; ok && o != nil {
		return o, nil
	}
	return &Observation{}, nil
}

// compile-time interface checks.
var (
	_ Rescanner = RescanFunc(nil)
	_ Rescanner = (*FakeRescanner)(nil)
)
