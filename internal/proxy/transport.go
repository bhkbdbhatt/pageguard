package proxy

import (
	"net/http"
	"net/url"
	"time"
)

// RoundTripper is HTTP middleware that routes each request through a proxy
// selected by region (taken from the request context, see WithRegion). When no
// rotator or no matching proxy is available, it falls back to Base so callers
// can always make direct requests.
//
// On transport-level errors it reports the failure to the rotator and retries
// with the next proxy, up to MaxAttempts. Requests with a non-replayable body
// are not retried.
type RoundTripper struct {
	Base        http.RoundTripper
	Rotator     *Rotator
	MaxAttempts int
	// Timeout bounds a single attempt's dial/response. Zero uses defaults.
	Timeout time.Duration
}

func (rt *RoundTripper) base() http.RoundTripper {
	if rt.Base != nil {
		return rt.Base
	}
	return http.DefaultTransport
}

// RoundTrip implements http.RoundTripper.
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.Rotator == nil || rt.Rotator.Empty() {
		return rt.base().RoundTrip(req)
	}
	region := RegionFromContext(req.Context())
	attempts := rt.MaxAttempts
	if attempts <= 0 {
		attempts = 2
	}

	replayable := req.Body == nil || req.GetBody != nil
	var lastErr error
	for i := 0; i < attempts; i++ {
		p, ok := rt.Rotator.Next(region)
		if !ok {
			break
		}
		attemptReq := req
		if i > 0 {
			clone, err := cloneRequest(req)
			if err != nil {
				return nil, err
			}
			attemptReq = clone
		}
		tr, err := rt.transportFor(p)
		if err != nil {
			rt.Rotator.Report(p, err)
			lastErr = err
			if !replayable {
				break
			}
			continue
		}
		resp, err := tr.RoundTrip(attemptReq)
		rt.Rotator.Report(p, err)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !replayable {
			break
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return rt.base().RoundTrip(req)
}

func (rt *RoundTripper) transportFor(p Proxy) (http.RoundTripper, error) {
	u, err := url.Parse(p.URL)
	if err != nil {
		return nil, err
	}
	return &http.Transport{
		Proxy:                 http.ProxyURL(u),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          16,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: rt.Timeout,
		ExpectContinueTimeout: time.Second,
	}, nil
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	clone := req.Clone(req.Context())
	if req.Body == nil || req.GetBody == nil {
		return clone, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	clone.Body = body
	return clone, nil
}
