package privacy

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// Evaluate runs the privacy compliance engine. It re-scans the page under each
// configured consent state, evaluates every selected regulation, detects
// advisory flags, and returns a scored report.
//
// A non-nil error indicates the run could not be performed (bad config, nil
// rescanner, or a rescan failure); regulation results themselves are always
// reported.
func Evaluate(ctx context.Context, cfg *Config, r Rescanner) (*Report, error) {
	if cfg == nil {
		return nil, fmt.Errorf("privacy: nil config")
	}
	if err := cfg.Prepare(); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("privacy: no rescanner configured")
	}

	report := &Report{
		SchemaVersion: ReportSchemaVersion,
		PolicyName:    cfg.Name,
		GeneratedAt:   time.Now().UTC(),
	}

	observations := map[ConsentState]*Observation{}
	stateResults := map[ConsentState]StateResult{}
	for _, state := range cfg.Settings.ConsentStates {
		obs, err := r.Rescan(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("privacy: rescan %s: %w", state, err)
		}
		observations[state] = obs
		res := analyzeState(cfg, state, obs)
		stateResults[state] = res
		report.States = append(report.States, res)
		if report.URL == "" && obs != nil && obs.FinalURL != "" {
			report.URL = obs.FinalURL
		}
	}

	for _, reg := range cfg.Regulations {
		report.Regulations = append(report.Regulations, evaluateRegulation(cfg, reg, observations, stateResults))
	}

	report.Flags = detectFlags(cfg, observations)
	report.finalize(cfg.Settings.PassThreshold)
	return report, nil
}

// analyzeState classifies the observation into non-essential trackers and
// reports whether the page blocked them.
func analyzeState(cfg *Config, state ConsentState, obs *Observation) StateResult {
	sr := StateResult{State: state}
	if obs == nil {
		sr.Blocked = true
		return sr
	}
	pageHost := hostOf(obs.FinalURL)
	if pageHost == "" {
		pageHost = hostOf(obs.URL)
	}

	for _, t := range obs.Tags {
		if cfg.categoryEssential(t.Category) {
			continue
		}
		sr.TrackersFired = append(sr.TrackersFired, Tracker{
			Kind: "tag", ID: t.ID, Name: t.Name, Category: t.Category,
		})
	}
	for _, c := range obs.Cookies {
		if cfg.cookieEssential(c.Name) {
			continue
		}
		cat, _ := cfg.cookieCategory(c.Name)
		sr.CookiesSet = append(sr.CookiesSet, c.Name)
		sr.TrackersFired = append(sr.TrackersFired, Tracker{
			Kind: "cookie", ID: c.Name, Name: c.Name, Category: cat, Host: c.Domain,
		})
	}
	if cfg.tracksThirdPartyRequests() {
		seen := map[string]bool{}
		for _, req := range obs.Requests {
			h := strings.ToLower(req.Host)
			if h == "" {
				h = hostOf(req.URL)
			}
			if h == "" || h == pageHost || seen[h] || !isTrackerResource(req.ResourceType) {
				continue
			}
			if cfg.hostEssential(h) {
				continue
			}
			seen[h] = true
			sr.ThirdPartyHosts = append(sr.ThirdPartyHosts, h)
			sr.TrackersFired = append(sr.TrackersFired, Tracker{Kind: "request", ID: h, Name: h, Host: h})
		}
	}

	sort.Slice(sr.TrackersFired, func(i, j int) bool {
		if sr.TrackersFired[i].Kind != sr.TrackersFired[j].Kind {
			return sr.TrackersFired[i].Kind < sr.TrackersFired[j].Kind
		}
		return sr.TrackersFired[i].ID < sr.TrackersFired[j].ID
	})
	sort.Strings(sr.CookiesSet)
	sort.Strings(sr.ThirdPartyHosts)
	sr.Blocked = len(sr.TrackersFired) == 0
	return sr
}

// isTrackerResource reports whether a network request resource type is one that
// can constitute tracking (as opposed to a static asset).
func isTrackerResource(rt string) bool {
	switch strings.ToLower(rt) {
	case "image", "imageset", "font", "stylesheet", "media", "manifest", "texttrack", "favicon":
		return false
	}
	return true
}

func evaluateRegulation(cfg *Config, reg Regulation, obs map[ConsentState]*Observation, states map[ConsentState]StateResult) RegulationReport {
	rr := RegulationReport{Regulation: reg}
	switch reg {
	case GDPR:
		evalGDPR(cfg, obs, states, &rr)
	case CCPA:
		evalCCPA(cfg, obs, states, &rr)
	case LGPD:
		evalLGPD(cfg, obs, states, &rr)
	case EPrivacy:
		evalEPrivacy(cfg, obs, states, &rr)
	}
	return rr
}

func check(id string, status CheckStatus, sev domain.Severity, msg string, evidence map[string]any) Check {
	return Check{ID: id, Status: status, Severity: sev, Message: msg, Evidence: evidence}
}

func pass(id string, sev domain.Severity, msg string) Check {
	return check(id, CheckPass, sev, msg, nil)
}

func fail(id string, sev domain.Severity, msg string, evidence map[string]any) Check {
	return check(id, CheckFail, sev, msg, evidence)
}

func skip(id string, sev domain.Severity, msg string) Check {
	return check(id, CheckSkipped, sev, msg, nil)
}

// blockingFired returns, per blocking state that was simulated, the
// non-essential trackers that still fired.
func blockingFired(cfg *Config, states map[ConsentState]StateResult) map[string][]string {
	out := map[string][]string{}
	for _, s := range cfg.Settings.ConsentStates {
		if !s.expectingBlock() {
			continue
		}
		if ids := trackerIDs(states[s].TrackersFired); len(ids) > 0 {
			out[string(s)] = ids
		}
	}
	return out
}

func trackerIDs(ts []Tracker) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.Kind+":"+t.ID)
	}
	return out
}

func filterCategory(ts []Tracker, pred func(domain.TagCategory) bool) []Tracker {
	var out []Tracker
	for _, t := range ts {
		if t.Category != "" && pred(t.Category) {
			out = append(out, t)
		}
	}
	return out
}

// --- GDPR -----------------------------------------------------------------

func evalGDPR(cfg *Config, obs map[ConsentState]*Observation, states map[ConsentState]StateResult, rr *RegulationReport) {
	rr.Checks = append(rr.Checks, cmpPresentCheck(cfg, obs))

	if fired := blockingFired(cfg, states); len(fired) > 0 {
		rr.Checks = append(rr.Checks, fail("pre-consent-tracking", domain.SevCritical,
			"non-essential trackers fired after consent was refused",
			map[string]any{"states": fired}))
	} else {
		rr.Checks = append(rr.Checks, pass("pre-consent-tracking", domain.SevCritical,
			"no non-essential trackers fired in blocking consent states"))
	}
}

func cmpPresentCheck(cfg *Config, obs map[ConsentState]*Observation) Check {
	for _, o := range obs {
		if o == nil {
			continue
		}
		if o.BannerPresent {
			return pass("cmp-present", domain.SevHigh, "consent banner detected")
		}
		for _, t := range o.Tags {
			for _, id := range cfg.Settings.CMPTags {
				if strings.EqualFold(t.ID, id) {
					return pass("cmp-present", domain.SevHigh, "consent platform detected")
				}
			}
		}
	}
	return fail("cmp-present", domain.SevHigh, "no consent banner or consent platform detected", nil)
}

// --- CCPA -----------------------------------------------------------------

func evalCCPA(cfg *Config, obs map[ConsentState]*Observation, states map[ConsentState]StateResult, rr *RegulationReport) {
	if res, ok := states[StateGPC]; ok {
		sale := filterCategory(res.TrackersFired, cfg.categorySale)
		if len(sale) > 0 {
			rr.Checks = append(rr.Checks, fail("do-not-sell-signal", domain.SevHigh,
				"sale/share trackers fired despite a Global Privacy Control signal",
				map[string]any{"trackers": trackerIDs(sale)}))
		} else {
			rr.Checks = append(rr.Checks, pass("do-not-sell-signal", domain.SevHigh,
				"Global Privacy Control signal honoured"))
		}
	} else {
		rr.Checks = append(rr.Checks, skip("do-not-sell-signal", domain.SevHigh,
			"GPC state was not simulated"))
	}

	if optOutPresent(cfg, obs) {
		rr.Checks = append(rr.Checks, pass("opt-out-mechanism", domain.SevHigh, "opt-out mechanism detected"))
	} else {
		rr.Checks = append(rr.Checks, fail("opt-out-mechanism", domain.SevHigh,
			"no \"Do Not Sell or Share\" opt-out mechanism detected", nil))
	}

	if res, ok := states[StateRejectAll]; ok {
		sale := filterCategory(res.TrackersFired, cfg.categorySale)
		if len(sale) > 0 {
			rr.Checks = append(rr.Checks, fail("opt-out-honored", domain.SevHigh,
				"sale/share trackers fired after the visitor opted out",
				map[string]any{"trackers": trackerIDs(sale)}))
		} else {
			rr.Checks = append(rr.Checks, pass("opt-out-honored", domain.SevHigh, "opt-out was honoured"))
		}
	} else {
		rr.Checks = append(rr.Checks, skip("opt-out-honored", domain.SevHigh, "reject-all state was not simulated"))
	}
}

func optOutPresent(cfg *Config, obs map[ConsentState]*Observation) bool {
	for _, o := range obs {
		if o == nil {
			continue
		}
		if o.BannerHasReject || o.BannerHasSettings {
			return true
		}
		if cfg.matchesOptOut(o.FinalURL) || cfg.matchesOptOut(o.URL) {
			return true
		}
		for _, l := range o.Links {
			if cfg.matchesOptOut(l) {
				return true
			}
		}
		for _, req := range o.Requests {
			if cfg.matchesOptOut(req.URL) {
				return true
			}
		}
	}
	return false
}

// --- LGPD -----------------------------------------------------------------

func evalLGPD(cfg *Config, obs map[ConsentState]*Observation, states map[ConsentState]StateResult, rr *RegulationReport) {
	if len(cfg.SubjectRights) == 0 {
		rr.Checks = append(rr.Checks, skip("subject-rights", domain.SevMedium, "no subject_rights configured"))
	}
	for i, sr := range cfg.SubjectRights {
		re := cfg.rightsRe[i]
		if rightPresent(re, obs) {
			rr.Checks = append(rr.Checks, pass("subject-right:"+sr.ID, domain.SevMedium,
				"data-subject right \""+rightName(sr)+"\" is reachable"))
		} else {
			rr.Checks = append(rr.Checks, fail("subject-right:"+sr.ID, domain.SevMedium,
				"data-subject right \""+rightName(sr)+"\" is not reachable",
				map[string]any{"pattern": sr.URLPattern}))
		}
	}

	if res, ok := states[StateRejectAll]; ok {
		if len(res.TrackersFired) > 0 {
			rr.Checks = append(rr.Checks, fail("consent-honored", domain.SevHigh,
				"non-essential trackers fired after consent was refused",
				map[string]any{"trackers": trackerIDs(res.TrackersFired)}))
		} else {
			rr.Checks = append(rr.Checks, pass("consent-honored", domain.SevHigh, "consent refusal was honoured"))
		}
	} else {
		rr.Checks = append(rr.Checks, skip("consent-honored", domain.SevHigh, "reject-all state was not simulated"))
	}
}

func rightName(sr SubjectRight) string {
	if sr.Name != "" {
		return sr.Name
	}
	return sr.ID
}

func rightPresent(re interface{ MatchString(string) bool }, obs map[ConsentState]*Observation) bool {
	if re == nil {
		return false
	}
	for _, o := range obs {
		if o == nil {
			continue
		}
		if re.MatchString(o.URL) || re.MatchString(o.FinalURL) {
			return true
		}
		for _, l := range o.Links {
			if re.MatchString(l) {
				return true
			}
		}
		for _, req := range o.Requests {
			if re.MatchString(req.URL) {
				return true
			}
		}
	}
	return false
}

// --- ePrivacy -------------------------------------------------------------

func evalEPrivacy(cfg *Config, obs map[ConsentState]*Observation, states map[ConsentState]StateResult, rr *RegulationReport) {
	banner, reject := false, false
	for _, o := range obs {
		if o == nil {
			continue
		}
		if o.BannerPresent {
			banner = true
		}
		if o.BannerHasReject {
			reject = true
		}
	}
	if banner {
		rr.Checks = append(rr.Checks, pass("banner-present", domain.SevHigh, "cookie consent banner is shown"))
	} else {
		rr.Checks = append(rr.Checks, fail("banner-present", domain.SevHigh, "no cookie consent banner detected", nil))
	}
	if reject {
		rr.Checks = append(rr.Checks, pass("banner-reject-option", domain.SevHigh, "banner offers a reject-all option"))
	} else {
		rr.Checks = append(rr.Checks, fail("banner-reject-option", domain.SevHigh,
			"banner does not offer a reject-all option", nil))
	}

	blocked := true
	evidence := map[string]any{}
	for _, s := range cfg.Settings.ConsentStates {
		if !s.expectingBlock() {
			continue
		}
		if names := states[s].CookiesSet; len(names) > 0 {
			blocked = false
			evidence[string(s)] = names
		}
	}
	if blocked {
		rr.Checks = append(rr.Checks, pass("pre-consent-cookies", domain.SevCritical,
			"no non-essential cookies were set before consent"))
	} else {
		rr.Checks = append(rr.Checks, fail("pre-consent-cookies", domain.SevCritical,
			"non-essential cookies were set before consent", evidence))
	}
}

// --- flags ----------------------------------------------------------------

func detectFlags(cfg *Config, obs map[ConsentState]*Observation) []Flag {
	var flags []Flag

	sample := obs[StateAcceptAll]
	if sample == nil {
		for _, o := range obs {
			if o != nil {
				sample = o
				break
			}
		}
	}
	if sample != nil {
		for _, trig := range cfg.Settings.DPIA {
			var matched []domain.Tag
			if len(trig.Categories) > 0 {
				matched = filterCategoryTags(sample.Tags, trig.Categories)
			}
			countHit := trig.MinTrackers > 0 && len(sample.Tags) >= trig.MinTrackers
			if len(matched) == 0 && !countHit {
				continue
			}
			ev := map[string]any{}
			if len(matched) > 0 {
				ev["tags"] = tagIDs(matched)
				ev["categories"] = categoryStrings(trig.Categories)
			}
			if trig.MinTrackers > 0 {
				ev["trackerCount"] = len(sample.Tags)
				ev["minTrackers"] = trig.MinTrackers
			}
			flags = append(flags, Flag{
				ID:          "dpia:" + trig.ID,
				Regulation:  GDPR,
				Severity:    domain.SevMedium,
				Title:       "DPIA required" + triggerSuffix(trig),
				Description: "processing matches a declared Data Protection Impact Assessment trigger",
				Evidence:    ev,
			})
		}
	}

	hosts := map[string]bool{}
	for _, o := range obs {
		if o == nil {
			continue
		}
		for _, req := range o.Requests {
			h := strings.ToLower(req.Host)
			if h == "" {
				h = hostOf(req.URL)
			}
			if h != "" && cfg.isCrossBorder(h) {
				hosts[h] = true
			}
		}
		for _, c := range o.Cookies {
			d := strings.ToLower(strings.TrimPrefix(c.Domain, "."))
			if d != "" && cfg.isCrossBorder(d) {
				hosts[d] = true
			}
		}
	}
	if len(hosts) > 0 {
		flags = append(flags, Flag{
			ID:          "cross-border-transfer",
			Regulation:  GDPR,
			Severity:    domain.SevMedium,
			Title:       "Cross-border data transfer detected",
			Description: "requests or cookies reach hosts outside the declared regions",
			Evidence:    map[string]any{"hosts": sortedKeys(hosts)},
		})
	}

	sort.Slice(flags, func(i, j int) bool { return flags[i].ID < flags[j].ID })
	return flags
}

func triggerSuffix(t DPIATrigger) string {
	if t.Description != "" {
		return ": " + t.Description
	}
	return ""
}

func tagIDs(tags []domain.Tag) []string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = t.ID
	}
	return out
}

func categoryStrings(cats []domain.TagCategory) []string {
	out := make([]string, len(cats))
	for i, c := range cats {
		out[i] = string(c)
	}
	return out
}

func filterCategoryTags(tags []domain.Tag, cats []domain.TagCategory) []domain.Tag {
	want := map[domain.TagCategory]bool{}
	for _, c := range cats {
		want[c] = true
	}
	var out []domain.Tag
	for _, t := range tags {
		if want[t.Category] {
			out = append(out, t)
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func hostOf(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Host)
}
