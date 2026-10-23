// Package rules evaluates user-authored AuditRules against captured page
// evidence. Rules are declarative so governance can be edited without shipping
// code.
package rules

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/martech"
)

// Status is the outcome of evaluating one rule.
type Status string

const (
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusSkipped Status = "skipped"
)

// Result is the outcome of one rule against one page.
type Result struct {
	Rule     *domain.AuditRule
	Status   Status
	Reason   string
	Evidence map[string]any
}

// Failed reports whether the result is a violation.
func (r Result) Failed() bool { return r.Status == StatusFail }

// EvaluateAll evaluates every enabled rule against a page.
func EvaluateAll(all []*domain.AuditRule, sr *domain.ScanResult) []Result {
	out := make([]Result, 0, len(all))
	for _, r := range all {
		if r == nil || !r.Enabled {
			continue
		}
		out = append(out, Evaluate(r, sr))
	}
	return out
}

// Evaluate applies one rule to one page.
func Evaluate(r *domain.AuditRule, sr *domain.ScanResult) Result {
	res := Result{Rule: r, Evidence: map[string]any{
		"pageUrl":   sr.URL,
		"vendor":    r.Vendor,
		"target":    string(r.Target),
		"parameter": r.Parameter,
		"operator":  string(r.Operator),
		"expected":  r.Expected,
	}}
	if sr.Error != "" {
		return Result{Rule: r, Status: StatusSkipped, Reason: "page scan failed", Evidence: res.Evidence}
	}
	if !urlMatches(r.URLPattern, sr.URL) {
		res.Status = StatusSkipped
		res.Reason = "page URL does not match rule scope"
		return res
	}

	var matches []observation
	var err error
	switch r.Target {
	case domain.RuleTargetRequest:
		matches, err = requestObservations(r, sr)
	case domain.RuleTargetEvent:
		matches, err = eventObservations(r, sr)
	case domain.RuleTargetTag:
		matches = tagObservations(r, sr.Tags)
	case domain.RuleTargetCookie:
		matches = cookieObservations(r, sr.Cookies)
	default:
		err = fmt.Errorf("unsupported target %q", r.Target)
	}
	if err != nil {
		res.Status = StatusSkipped
		res.Reason = err.Error()
		return res
	}

	pass := assertCollection(r.Operator, r.Expected, matches)
	res.Evidence["matched"] = len(matches)
	res.Evidence["samples"] = samples(matches)
	if pass {
		res.Status = StatusPass
	} else {
		res.Status = StatusFail
		res.Reason = failureReason(r, matches)
	}
	return res
}

type observation struct {
	present bool
	value   string
	source  string
}

func requestObservations(r *domain.AuditRule, sr *domain.ScanResult) ([]observation, error) {
	param := strings.TrimSpace(r.Parameter)
	if param == "" {
		return nil, fmt.Errorf("request rule requires a parameter")
	}
	out := []observation{}
	for _, req := range sr.Requests {
		host := req.Host
		if host == "" {
			host = martech.HostURL(req.URL)
		}
		if r.Vendor != "" && !martech.MatchVendor(r.Vendor, host) {
			continue
		}
		u, err := url.Parse(req.URL)
		if err != nil {
			continue
		}
		values, ok := u.Query()[param]
		if !ok {
			out = append(out, observation{source: req.URL})
			continue
		}
		for _, v := range values {
			out = append(out, observation{present: true, value: v, source: req.URL})
		}
	}
	return out, nil
}

func eventObservations(r *domain.AuditRule, sr *domain.ScanResult) ([]observation, error) {
	out := []observation{}
	for _, entry := range sr.DataLayer {
		flat := map[string]any{}
		var decoded any
		if len(entry.Value) > 0 {
			if err := json.Unmarshal(entry.Value, &decoded); err == nil {
				flatten("", decoded, flat)
			}
		}
		if entry.Name != "" {
			flat["_entry"] = entry.Name
			if _, ok := flat["event"]; !ok {
				flat["event"] = entry.Name
			}
		}
		if r.Event != "" {
			event, _ := flat["event"].(string)
			if !equalFold(event, r.Event) && !equalFold(entry.Name, r.Event) {
				continue
			}
		}
		key := r.Parameter
		if key == "" {
			key = "event"
		}
		v, ok := flat[key]
		if !ok {
			out = append(out, observation{source: entry.Name})
			continue
		}
		out = append(out, observation{present: true, value: stringify(v), source: entry.Name})
	}
	return out, nil
}

func tagObservations(r *domain.AuditRule, tags []domain.Tag) []observation {
	out := []observation{}
	for _, tag := range tags {
		if r.Vendor != "" {
			vendorValue := strings.Join([]string{tag.Vendor, tag.Name, tag.ID, tag.Evidence}, " ")
			if !martech.MatchVendor(r.Vendor, vendorValue) {
				continue
			}
		}
		if r.Parameter != "" && !containsFold(tag.ID+" "+tag.Name+" "+tag.Evidence, r.Parameter) {
			continue
		}
		out = append(out, observation{present: true, value: tag.Name, source: tag.Evidence})
	}
	return out
}

func cookieObservations(r *domain.AuditRule, cookies []domain.Cookie) []observation {
	out := []observation{}
	for _, c := range cookies {
		if r.Vendor != "" && !martech.MatchVendor(r.Vendor, c.Domain) {
			continue
		}
		if r.Parameter != "" && !containsFold(c.Name, r.Parameter) {
			continue
		}
		out = append(out, observation{present: true, value: c.Name, source: c.Domain})
	}
	return out
}

func assertCollection(op domain.RuleOperator, expected string, obs []observation) bool {
	anyPresent := false
	positive := false
	re, reErr := regexp.Compile(expected)
	for _, o := range obs {
		if o.present {
			anyPresent = true
		}
		switch op {
		case domain.RuleOpExists:
			if o.present {
				positive = true
			}
		case domain.RuleOpEquals:
			if o.present && o.value == expected {
				positive = true
			}
		case domain.RuleOpMatches:
			if reErr == nil && o.present && re.MatchString(o.value) {
				positive = true
			}
		case domain.RuleOpNotExists:
			if o.present {
				positive = true
			}
		case domain.RuleOpNotEquals:
			if o.present && o.value == expected {
				positive = true
			}
		case domain.RuleOpNotMatches:
			if reErr == nil && o.present && re.MatchString(o.value) {
				positive = true
			}
		default:
			return false
		}
	}
	switch op {
	case domain.RuleOpExists, domain.RuleOpEquals, domain.RuleOpMatches:
		if op == domain.RuleOpExists {
			return anyPresent
		}
		return positive
	case domain.RuleOpNotExists, domain.RuleOpNotEquals, domain.RuleOpNotMatches:
		return !positive
	}
	return false
}

func failureReason(r *domain.AuditRule, obs []observation) string {
	if len(obs) == 0 {
		return fmt.Sprintf("no %s evidence matched the rule", r.Target)
	}
	switch r.Operator {
	case domain.RuleOpExists:
		return fmt.Sprintf("%s %q was not found", r.Target, r.Parameter)
	case domain.RuleOpEquals:
		return fmt.Sprintf("%s %q did not equal %q", r.Target, r.Parameter, r.Expected)
	case domain.RuleOpMatches:
		return fmt.Sprintf("%s %q did not match %q", r.Target, r.Parameter, r.Expected)
	case domain.RuleOpNotExists:
		return fmt.Sprintf("%s %q is present but should not be", r.Target, r.Parameter)
	case domain.RuleOpNotEquals:
		return fmt.Sprintf("%s %q equals forbidden value %q", r.Target, r.Parameter, r.Expected)
	case domain.RuleOpNotMatches:
		return fmt.Sprintf("%s %q matches forbidden pattern %q", r.Target, r.Parameter, r.Expected)
	}
	return "rule assertion failed"
}

// Findings converts failed rule results into persisted findings.
func Findings(runID, pageID domain.ID, pageURL string, results []Result) []*domain.Finding {
	now := time.Now().UTC()
	out := []*domain.Finding{}
	for _, res := range results {
		if !res.Failed() {
			continue
		}
		r := res.Rule
		evidence := map[string]any{}
		for k, v := range res.Evidence {
			evidence[k] = v
		}
		evidence["reason"] = res.Reason
		out = append(out, &domain.Finding{
			ID:          domain.NewID(),
			RunID:       runID,
			PageID:      pageID,
			PageURL:     pageURL,
			PluginID:    "rules",
			RuleID:      string(r.ID),
			Severity:    r.Severity,
			Title:       r.Name,
			Description: res.Reason,
			Evidence:    evidence,
			CreatedAt:   now,
		})
	}
	return out
}

// PageHealth reports whether a page passed all evaluated rules and how many
// rules were evaluated. Skipped rules are excluded from the denominator.
func PageHealth(results []Result) (passed bool, evaluated int) {
	passed = true
	for _, res := range results {
		if res.Status == StatusSkipped {
			continue
		}
		evaluated++
		if res.Failed() {
			passed = false
		}
	}
	return passed, evaluated
}

// Aggregate summarizes results across pages.
type Aggregate struct {
	TotalRules   int                `json:"totalRules"`
	Evaluated    int                `json:"evaluated"`
	Passed       int                `json:"passed"`
	Failed       int                `json:"failed"`
	VendorScores map[string]float64 `json:"vendorScores"`
	HealthScore  float64            `json:"healthScore"`
	RuleStats    []RuleStat         `json:"rules"`
}

// RuleStat is per-rule pass/fail counts across a run.
type RuleStat struct {
	Rule     *domain.AuditRule `json:"rule"`
	Passed   int               `json:"passed"`
	Failed   int               `json:"failed"`
	Skipped  int               `json:"skipped"`
	PassRate float64           `json:"passRate"`
}

// Summarize aggregates per-page rule results.
func Summarize(all []*domain.AuditRule, byPage map[string][]Result) Aggregate {
	stats := make(map[domain.ID]*RuleStat, len(all))
	for _, r := range all {
		stats[r.ID] = &RuleStat{Rule: r}
	}
	agg := Aggregate{VendorScores: map[string]float64{}, RuleStats: []RuleStat{}}
	pageEvaluated := 0
	pagePassed := 0
	for _, results := range byPage {
		ok, n := PageHealth(results)
		if n > 0 {
			pageEvaluated++
			if ok {
				pagePassed++
			}
		}
		for _, res := range results {
			st, ok := stats[res.Rule.ID]
			if !ok {
				continue
			}
			switch res.Status {
			case StatusPass:
				st.Passed++
			case StatusFail:
				st.Failed++
			case StatusSkipped:
				st.Skipped++
			}
		}
	}
	vendorPass := map[string]int{}
	vendorTotal := map[string]int{}
	for _, st := range stats {
		agg.Passed += st.Passed
		agg.Failed += st.Failed
		agg.Evaluated += st.Passed + st.Failed
		if st.Rule.Vendor != "" {
			vendorPass[st.Rule.Vendor] += st.Passed
			vendorTotal[st.Rule.Vendor] += st.Passed + st.Failed
		}
		if st.Passed+st.Failed > 0 {
			st.PassRate = float64(st.Passed) / float64(st.Passed+st.Failed) * 100
		}
		agg.RuleStats = append(agg.RuleStats, *st)
	}
	agg.TotalRules = len(stats)
	if pageEvaluated > 0 {
		agg.HealthScore = float64(pagePassed) / float64(pageEvaluated) * 100
	}
	for vendor, total := range vendorTotal {
		if total > 0 {
			agg.VendorScores[vendor] = float64(vendorPass[vendor]) / float64(total) * 100
		}
	}
	return agg
}

// HealthFromFindings is the fallback score when no rules are configured.
func HealthFromFindings(findings []*domain.Finding) float64 {
	penalty := 0.0
	for _, f := range findings {
		switch f.Severity {
		case domain.SevCritical:
			penalty += 10
		case domain.SevHigh:
			penalty += 5
		case domain.SevMedium:
			penalty += 2
		case domain.SevLow:
			penalty++
		}
	}
	score := 100 - penalty
	if score < 0 {
		return 0
	}
	return score
}

func urlMatches(pattern, raw string) bool {
	if pattern == "" {
		return true
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return true
	}
	return re.MatchString(raw)
}

func flatten(prefix string, v any, out map[string]any) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flatten(key, child, out)
		}
	case []any:
		for i, child := range t {
			flatten(prefix+"."+strconv.Itoa(i), child, out)
		}
	default:
		if prefix != "" {
			out[prefix] = t
		}
	}
}

func samples(obs []observation) []map[string]any {
	out := []map[string]any{}
	for i, o := range obs {
		if i >= 3 {
			break
		}
		out = append(out, map[string]any{
			"present": o.present,
			"value":   o.value,
			"source":  o.source,
		})
	}
	return out
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func equalFold(a, b string) bool { return strings.EqualFold(a, b) }
