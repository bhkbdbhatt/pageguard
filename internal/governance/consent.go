package governance

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/martech"
)

// ConsentEntry is one tracker/CMP row in the consent matrix.
type ConsentEntry struct {
	Key         string   `json:"key"`
	Category    string   `json:"category"`
	Name        string   `json:"name"`
	Vendor      string   `json:"vendor"`
	Type        string   `json:"type"`
	Essential   bool     `json:"essential"`
	PreConsent  bool     `json:"preConsent"`
	PostConsent bool     `json:"postConsent"`
	Leak        bool     `json:"leak"`
	Status      string   `json:"status"`
	Evidence    string   `json:"evidence,omitempty"`
	Cookies     []string `json:"cookies,omitempty"`
}

// ConsentMatrix is the per-page CMP governance view.
type ConsentMatrix struct {
	RunID             domain.ID      `json:"runId"`
	URL               string         `json:"url"`
	PageURL           string         `json:"pageUrl"`
	PageID            domain.ID      `json:"pageId"`
	GeneratedAt       time.Time      `json:"generatedAt"`
	ConsentSimulation string         `json:"consentSimulation,omitempty"`
	CMPDetected       bool           `json:"cmpDetected"`
	CMPVendor         string         `json:"cmpVendor,omitempty"`
	Trackers          int            `json:"trackers"`
	Beacons           int            `json:"beacons"`
	Leaks             int            `json:"leaks"`
	PreConsentScore   float64        `json:"preConsentScore"`
	PostConsentScore  float64        `json:"postConsentScore"`
	Entries           []ConsentEntry `json:"entries"`
}

type consentSummary struct {
	Beacons     int
	Leaks       int
	Score       float64
	CMPDetected bool
	CMPVendor   string
}

// ConsentScoreForResults exposes the derived consent score to the orchestrator.
func ConsentScoreForResults(results []*domain.ScanResult) float64 {
	_, summary := consentFromResults(results)
	return summary.Score
}

// BeaconsForResults counts tracker-like requests across pages.
func BeaconsForResults(results []*domain.ScanResult) int {
	_, summary := consentFromResults(results)
	return summary.Beacons
}

// Consent returns the CMP matrix for a run/page.
func (s *Service) Consent(ctx context.Context, runID domain.ID, rawURL string) (*ConsentMatrix, error) {
	run, scan, _, results, err := s.runContext(ctx, runID)
	if err != nil {
		return nil, err
	}
	sr := pickResult(results, rawURL)
	if sr == nil {
		return nil, domain.ErrNotFound
	}
	matrix := buildConsentMatrix(sr)
	matrix.RunID = run.ID
	if scan != nil {
		matrix.ConsentSimulation = scan.Config.ConsentSimulation
	}
	return matrix, nil
}

func consentFromResults(results []*domain.ScanResult) ([]*ConsentMatrix, consentSummary) {
	summary := consentSummary{Score: 100}
	matrices := make([]*ConsentMatrix, 0, len(results))
	leakKeys := map[string]bool{}
	beacons := 0
	cmpDetected := false
	cmpVendor := ""
	for _, sr := range results {
		m := buildConsentMatrix(sr)
		matrices = append(matrices, m)
		beacons += m.Beacons
		if m.CMPDetected {
			cmpDetected = true
			if cmpVendor == "" {
				cmpVendor = m.CMPVendor
			}
		}
		for _, e := range m.Entries {
			if e.Leak {
				leakKeys[e.Key] = true
			}
		}
	}
	if beacons > 0 || len(leakKeys) > 0 {
		leaks := len(leakKeys)
		denom := beacons
		if leaks > denom {
			denom = leaks
		}
		if denom > 0 {
			summary.Score = float64(denom-leaks) / float64(denom) * 100
		}
	}
	summary.Beacons = beacons
	summary.Leaks = len(leakKeys)
	summary.CMPDetected = cmpDetected
	summary.CMPVendor = cmpVendor
	return matrices, summary
}

func buildConsentMatrix(sr *domain.ScanResult) *ConsentMatrix {
	m := &ConsentMatrix{
		URL: sr.URL, PageURL: sr.FinalURL, PageID: sr.PageID, GeneratedAt: time.Now().UTC(),
		Entries: []ConsentEntry{},
	}
	entries := map[string]*ConsentEntry{}
	essentialKey := func(host string) bool {
		return martech.IsConsentHost(host) || strings.Contains(strings.ToLower(host), "googletagmanager.com")
	}
	ensure := func(key, category, name, vendor, typ string) *ConsentEntry {
		if e, ok := entries[key]; ok {
			if name != "" && e.Name == "" {
				e.Name = name
			}
			return e
		}
		e := &ConsentEntry{Key: key, Category: category, Name: name, Vendor: vendor, Type: typ}
		entries[key] = e
		return e
	}

	for _, tag := range sr.Tags {
		vendor := martech.DetectTag(tag)
		if strings.EqualFold(vendor, "OneTrust") || tag.Category == domain.TagConsent {
			m.CMPDetected = true
			m.CMPVendor = vendor
			e := ensure("cmp", "Consent", "CMP banner", vendor, "Tag")
			e.Essential, e.PreConsent, e.PostConsent, e.Status = true, true, true, "allowed"
			e.Evidence = tag.Evidence
			continue
		}
		key := vendor
		if key == "" {
			key = tag.ID
		}
		e := ensure(key, categoryForTag(tag), tag.Name, vendor, "Tag")
		e.Evidence = tag.Evidence
		e.PreConsent = true
		e.PostConsent = true
		if tag.Category == domain.TagTagManager {
			e.Essential = true
			e.Status = "allowed"
		}
	}

	for _, req := range sr.Requests {
		if !martech.IsTrackerRequest(req) {
			continue
		}
		m.Beacons++
		host := req.Host
		if host == "" {
			host = martech.HostURL(req.URL)
		}
		vendor := martech.DetectHost(host)
		if essentialKey(host) {
			if martech.IsConsentHost(host) {
				m.CMPDetected = true
				if m.CMPVendor == "" {
					m.CMPVendor = "OneTrust"
				}
			}
			key := "essential:" + host
			e := ensure(key, "Infrastructure", host, vendor, "Network")
			e.Essential, e.PreConsent, e.PostConsent, e.Status = true, true, true, "allowed"
			e.Evidence = req.URL
			continue
		}
		key := vendor
		if key == "" {
			key = "host:" + host
		}
		e := ensure(key, categoryForVendor(vendor), host, vendor, "Network")
		e.PreConsent = true
		e.PostConsent = true
		e.Evidence = req.URL
	}

	for _, c := range sr.Cookies {
		if martech.IsEssentialCookie(c.Name) {
			if strings.Contains(strings.ToLower(c.Name), "consent") || strings.Contains(strings.ToLower(c.Name), "optanon") {
				m.CMPDetected = true
				if m.CMPVendor == "" {
					m.CMPVendor = "OneTrust"
				}
			}
			continue
		}
		vendor := martech.DetectHost(c.Domain)
		if vendor == "" && !c.FirstParty {
			vendor = "Other"
		}
		key := vendor
		if key == "" {
			key = "firstparty"
		}
		e := ensure(key, "Cookies", c.Name, vendor, "Cookie")
		e.PreConsent = true
		e.PostConsent = true
		e.Cookies = append(e.Cookies, c.Name)
		if e.Evidence == "" {
			e.Evidence = c.Domain
		}
	}

	for _, e := range entries {
		if e.Essential {
			continue
		}
		if e.Type == "Cookie" && e.Vendor == "" {
			e.Status = "observed"
			continue
		}
		e.Leak = true
		e.Status = "leak"
	}

	for _, v := range martech.Vendors {
		if v.Display == "OneTrust" {
			continue
		}
		key := v.Display
		if _, ok := entries[key]; ok {
			continue
		}
		e := &ConsentEntry{
			Key: key, Category: categoryForVendor(v.Display), Name: v.Display + " tags",
			Vendor: v.Display, Type: "Network", PreConsent: false, PostConsent: true,
			Status: "blocked",
		}
		entries[key] = e
	}

	leaks := 0
	for _, e := range entries {
		if e.Leak {
			leaks++
		}
	}
	m.Entries = make([]ConsentEntry, 0, len(entries))
	for _, e := range entries {
		m.Entries = append(m.Entries, *e)
	}
	sort.Slice(m.Entries, func(i, j int) bool {
		if m.Entries[i].Leak != m.Entries[j].Leak {
			return m.Entries[i].Leak
		}
		if m.Entries[i].Essential != m.Entries[j].Essential {
			return m.Entries[i].Essential
		}
		if m.Entries[i].Category == m.Entries[j].Category {
			return m.Entries[i].Name < m.Entries[j].Name
		}
		return m.Entries[i].Category < m.Entries[j].Category
	})
	m.Trackers = len(m.Entries)
	m.Leaks = leaks
	if m.Beacons > 0 || leaks > 0 {
		denom := m.Beacons
		if leaks > denom {
			denom = leaks
		}
		if denom > 0 {
			m.PreConsentScore = float64(denom-leaks) / float64(denom) * 100
		}
	} else {
		m.PreConsentScore = 100
	}
	m.PostConsentScore = 100
	return m
}

func categoryForTag(tag domain.Tag) string {
	switch tag.Category {
	case domain.TagAdvertising:
		return "Marketing"
	case domain.TagAnalytics:
		return "Analytics"
	case domain.TagSocial:
		return "Social"
	case domain.TagTagManager:
		return "Tag Manager"
	case domain.TagConsent:
		return "Consent"
	case domain.TagCDP:
		return "CDP"
	case domain.TagMonitoring:
		return "Monitoring"
	}
	if tag.Vendor != "" {
		return categoryForVendor(tag.Vendor)
	}
	return "Other"
}

func categoryForVendor(vendor string) string {
	switch strings.ToLower(vendor) {
	case "meta":
		return "Marketing"
	case "adobe", "ga4":
		return "Analytics"
	case "onetrust":
		return "Consent"
	}
	if vendor == "" {
		return "Other"
	}
	return "Analytics"
}
