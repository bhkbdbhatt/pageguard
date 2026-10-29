package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// Inventory aggregates tags, cookies and third-party requests across every
// scanned page of a run. When runID is empty the project's latest run is used.
func (s *Service) Inventory(ctx context.Context, projectID domain.ID, runID domain.ID) (*domain.InventorySnapshot, error) {
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	snap := &domain.InventorySnapshot{ProjectID: project.ID, GeneratedAt: time.Now().UTC()}

	var run *domain.Run
	if !runID.IsEmpty() {
		run, err = s.store.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		if run.ProjectID != project.ID {
			return nil, domain.ErrNotFound
		}
	} else {
		run, err = s.latestRun(ctx, project.ID)
		if err != nil {
			return nil, err
		}
	}
	if run == nil {
		return snap, nil
	}
	snap.RunID = run.ID
	snap.ScanID = run.ScanID

	results, err := s.store.ListScanResults(ctx, run.ID, 5000)
	if err != nil {
		return nil, err
	}
	snap.PagesScanned = len(results)

	tags := map[string]*domain.InventoryTag{}
	cookies := map[string]*domain.InventoryCookie{}
	requests := map[string]*domain.InventoryRequest{}
	tagPages := map[string]map[string]bool{}
	cookiePages := map[string]map[string]bool{}
	requestPages := map[string]map[string]bool{}

	for _, res := range results {
		if res == nil {
			continue
		}
		page := res.URL
		if page == "" {
			page = res.FinalURL
		}
		seen := res.StartedAt
		if seen.IsZero() {
			seen = res.FinishedAt
		}
		if seen.IsZero() {
			seen = run.CreatedAt
		}
		for _, tag := range res.Tags {
			if tag.ID == "" {
				continue
			}
			key := strings.ToLower(tag.ID)
			item, ok := tags[key]
			if !ok {
				item = &domain.InventoryTag{
					Tag: tag, Purpose: string(tag.Category), Pages: 0,
					FirstSeen: seen, LastSeen: seen,
				}
				tags[key] = item
				tagPages[key] = map[string]bool{}
			}
			if seen.Before(item.FirstSeen) {
				item.FirstSeen = seen
			}
			if seen.After(item.LastSeen) {
				item.LastSeen = seen
			}
			if !tagPages[key][page] {
				tagPages[key][page] = true
				item.Pages++
			}
		}
		for _, c := range res.Cookies {
			if c.Name == "" {
				continue
			}
			key := strings.ToLower(c.Name + "|" + c.Domain)
			item, ok := cookies[key]
			if !ok {
				item = &domain.InventoryCookie{
					Cookie: c, Vendor: cookieVendor(c), Purpose: cookiePurpose(c.Name), Pages: 0,
				}
				cookies[key] = item
				cookiePages[key] = map[string]bool{}
			}
			if !cookiePages[key][page] {
				cookiePages[key][page] = true
				item.Pages++
			}
		}
		for _, r := range res.Requests {
			if r.URL == "" {
				continue
			}
			key := strings.ToUpper(r.Method) + " " + r.URL
			item, ok := requests[key]
			if !ok {
				item = &domain.InventoryRequest{
					NetworkRequest: r, Vendor: requestVendor(r), Count: 0, Pages: 0,
				}
				requests[key] = item
				requestPages[key] = map[string]bool{}
			}
			item.Count++
			if !requestPages[key][page] {
				requestPages[key][page] = true
				item.Pages++
			}
		}
	}

	for _, v := range tags {
		snap.Tags = append(snap.Tags, *v)
	}
	for _, v := range cookies {
		snap.Cookies = append(snap.Cookies, *v)
	}
	for _, v := range requests {
		snap.Requests = append(snap.Requests, *v)
	}
	sort.Slice(snap.Tags, func(i, j int) bool {
		if snap.Tags[i].Pages != snap.Tags[j].Pages {
			return snap.Tags[i].Pages > snap.Tags[j].Pages
		}
		return snap.Tags[i].Name < snap.Tags[j].Name
	})
	sort.Slice(snap.Cookies, func(i, j int) bool {
		if snap.Cookies[i].Pages != snap.Cookies[j].Pages {
			return snap.Cookies[i].Pages > snap.Cookies[j].Pages
		}
		return snap.Cookies[i].Name < snap.Cookies[j].Name
	})
	sort.Slice(snap.Requests, func(i, j int) bool {
		if snap.Requests[i].Pages != snap.Requests[j].Pages {
			return snap.Requests[i].Pages > snap.Requests[j].Pages
		}
		return snap.Requests[i].URL < snap.Requests[j].URL
	})
	return snap, nil
}

func requestVendor(r domain.NetworkRequest) string {
	host := r.Host
	if host == "" {
		parts := strings.SplitN(r.URL, "/", 3)
		if len(parts) >= 3 {
			host = parts[2]
		}
	}
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return strings.TrimPrefix(strings.ToLower(host), "www.")
}

func cookieVendor(c domain.Cookie) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(c.Domain)), ".")
}

var cookiePurposes = map[string]string{
	"_ga": "analytics", "_gid": "analytics", "_gat": "analytics",
	"_fbp": "advertising", "_fbc": "advertising", "_gcl_au": "advertising",
	"ide": "advertising", "nid": "advertising", "test_cookie": "advertising",
	"sessionid": "necessary", "session": "necessary", "csrftoken": "necessary",
	"ph_phc": "analytics", "ajs_anonymous_id": "analytics",
	"intercom": "support", "hubspotutk": "marketing",
}

func cookiePurpose(name string) string {
	lower := strings.ToLower(name)
	if p, ok := cookiePurposes[lower]; ok {
		return p
	}
	switch {
	case strings.HasPrefix(lower, "_ga"), strings.HasPrefix(lower, "amp_"),
		strings.Contains(lower, "analytics"), strings.Contains(lower, "_hjid"):
		return "analytics"
	case strings.HasPrefix(lower, "_fb"), strings.Contains(lower, "ads"),
		strings.Contains(lower, "campaign"), strings.Contains(lower, "track"):
		return "advertising"
	case strings.Contains(lower, "csrf"), strings.Contains(lower, "session"),
		strings.Contains(lower, "auth"), strings.Contains(lower, "token"):
		return "necessary"
	}
	return "unknown"
}
