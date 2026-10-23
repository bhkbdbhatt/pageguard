package governance

import (
	"context"
	"net/url"
	"sort"
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/martech"
)

// Beacon is one tracker-like request.
type Beacon struct {
	URL          string            `json:"url"`
	Host         string            `json:"host"`
	Vendor       string            `json:"vendor"`
	Method       string            `json:"method"`
	ResourceType string            `json:"resourceType"`
	Status       int               `json:"status"`
	ThirdParty   bool              `json:"thirdParty"`
	DurationMs   int64             `json:"durationMs"`
	SizeBytes    int64             `json:"sizeBytes"`
	StartedAtMs  int64             `json:"startedAtMs"`
	Query        map[string]string `json:"query,omitempty"`
}

// BeaconReport is the inspector payload for one page.
type BeaconReport struct {
	RunID    domain.ID      `json:"runId"`
	URL      string         `json:"url"`
	PageURL  string         `json:"pageUrl"`
	PageID   domain.ID      `json:"pageId"`
	Status   int            `json:"statusCode"`
	Duration int64          `json:"durationMs"`
	Count    int            `json:"count"`
	Vendors  map[string]int `json:"vendors"`
	Beacons  []Beacon       `json:"beacons"`
}

// Beacons returns tracker requests captured for a run/page.
func (s *Service) Beacons(ctx context.Context, runID domain.ID, rawURL string) (*BeaconReport, error) {
	_, _, _, results, err := s.runContext(ctx, runID)
	if err != nil {
		return nil, err
	}
	sr := pickResult(results, rawURL)
	if sr == nil {
		return nil, domain.ErrNotFound
	}
	out := &BeaconReport{
		RunID: runID, URL: sr.URL, PageURL: sr.FinalURL, PageID: sr.PageID,
		Status: sr.StatusCode, Duration: sr.DurationMs, Vendors: map[string]int{},
		Beacons: []Beacon{},
	}
	for _, req := range sr.Requests {
		if !martech.IsTrackerRequest(req) {
			continue
		}
		vendor := martech.DetectHost(req.Host)
		if vendor == "" {
			vendor = "Other"
		}
		query := map[string]string{}
		if u, err := url.Parse(req.URL); err == nil {
			for key, vals := range u.Query() {
				if len(vals) > 0 {
					query[key] = vals[0]
				}
			}
		}
		out.Beacons = append(out.Beacons, Beacon{
			URL: req.URL, Host: req.Host, Vendor: vendor, Method: req.Method,
			ResourceType: req.ResourceType, Status: req.Status, ThirdParty: req.ThirdParty,
			DurationMs: req.DurationMs, SizeBytes: req.SizeBytes, StartedAtMs: req.StartedAtMs,
			Query: query,
		})
		out.Vendors[vendor]++
	}
	sort.Slice(out.Beacons, func(i, j int) bool {
		if out.Beacons[i].StartedAtMs == out.Beacons[j].StartedAtMs {
			return strings.Compare(out.Beacons[i].URL, out.Beacons[j].URL) < 0
		}
		return out.Beacons[i].StartedAtMs < out.Beacons[j].StartedAtMs
	})
	out.Count = len(out.Beacons)
	return out, nil
}
