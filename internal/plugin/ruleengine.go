package plugin

import (
	"context"
	"strconv"
	"strings"

	"github.com/pageguard/pageguard/pkg/pluginsdk"
)

// DeclarativeValidator evaluates a set of data-driven RuleSpecs. Rules of this
// shape do not require code and can be authored by plugin manifests.
type DeclarativeValidator struct {
	PluginID string
	Rules    []pluginsdk.RuleSpec
}

// ID returns the owning plugin ID.
func (v *DeclarativeValidator) ID() string { return v.PluginID }

// Validate applies every rule and returns a finding for each match.
func (v *DeclarativeValidator) Validate(_ context.Context, obs *pluginsdk.Observation, _ pluginsdk.Scope) ([]pluginsdk.Finding, error) {
	var out []pluginsdk.Finding
	for _, r := range v.Rules {
		if !evalCondition(r.Condition, obs) {
			continue
		}
		out = append(out, pluginsdk.Finding{
			RuleID:      r.ID,
			Severity:    r.Severity,
			Title:       r.Title,
			Description: r.Description,
			Evidence: map[string]any{
				"url":         obs.FinalURL,
				"category":    r.Category,
				"remediation": r.Remediation,
			},
		})
	}
	return out, nil
}

func evalCondition(c pluginsdk.Condition, obs *pluginsdk.Observation) bool {
	for _, p := range c.All {
		if !evalPredicate(p, obs) {
			return false
		}
	}
	if len(c.Any) > 0 {
		any := false
		for _, p := range c.Any {
			if evalPredicate(p, obs) {
				any = true
				break
			}
		}
		if !any {
			return false
		}
	}
	for _, p := range c.None {
		if evalPredicate(p, obs) {
			return false
		}
	}
	// An entirely empty condition never fires.
	return len(c.All) > 0 || len(c.Any) > 0 || len(c.None) > 0
}

func evalPredicate(p pluginsdk.Predicate, obs *pluginsdk.Observation) bool {
	field := strings.ToLower(strings.TrimSpace(p.Field))
	op := strings.ToLower(strings.TrimSpace(p.Op))

	// Resolve the field to a typed value and an "exists" flag.
	var (
		numVal  float64
		strVal  string
		boolVal bool
		kind    string // "num" | "str" | "bool"
		exists  bool
	)
	switch field {
	case "statuscode":
		numVal, kind, exists = float64(obs.StatusCode), "num", true
	case "htmlLength", "htmllength":
		numVal, kind, exists = float64(len(obs.HTML)), "num", true
	case "requestcount":
		numVal, kind, exists = float64(len(obs.Requests)), "num", true
	case "url":
		strVal, kind, exists = obs.URL, "str", obs.URL != ""
	case "finalurl":
		strVal, kind, exists = obs.FinalURL, "str", obs.FinalURL != ""
	case "title":
		strVal, kind, exists = obs.Title, "str", obs.Title != ""
	case "html":
		strVal, kind, exists = obs.HTML, "str", true
	case "contenttype":
		strVal, kind, exists = obs.ContentType, "str", obs.ContentType != ""
	case "rendered":
		boolVal, kind, exists = obs.Rendered, "bool", true
	case "header":
		v, ok := lookupHeader(obs.Headers, p.Name)
		strVal, kind, exists = v, "str", ok
	case "tag":
		exists = hasTag(obs, p.Name)
		kind = "str"
	case "cookie":
		c, ok := findCookie(obs, p.Name)
		exists = ok
		strVal = c.Name
		kind = "str"
	case "cookiesecure":
		c, ok := findCookie(obs, p.Name)
		exists, boolVal, kind = ok, c.Secure, "bool"
	case "cookiehttponly":
		c, ok := findCookie(obs, p.Name)
		exists, boolVal, kind = ok, c.HTTPOnly, "bool"
	case "cookiesamesite":
		c, ok := findCookie(obs, p.Name)
		strVal, kind, exists = c.SameSite, "str", ok
	case "requesthost":
		exists = hasRequestHost(obs, p.Name)
		kind = "str"
	default:
		return false
	}

	switch op {
	case "exists":
		return exists
	case "not_exists":
		return !exists
	}
	if !exists {
		return false
	}

	switch op {
	case "eq":
		return compareEq(kind, numVal, strVal, boolVal, p.Value)
	case "neq":
		return !compareEq(kind, numVal, strVal, boolVal, p.Value)
	case "gt", "gte", "lt", "lte":
		return compareNum(kind, numVal, strVal, op, p.Value)
	case "contains":
		return strings.Contains(strings.ToLower(strVal), strings.ToLower(toString(p.Value)))
	case "not_contains":
		return !strings.Contains(strings.ToLower(strVal), strings.ToLower(toString(p.Value)))
	case "in":
		return containsValue(p.Value, strVal) || containsValue(p.Value, formatNum(numVal, kind))
	}
	return false
}

func compareEq(kind string, num float64, str string, b bool, v any) bool {
	switch kind {
	case "num":
		n, ok := toFloat(v)
		return ok && n == num
	case "bool":
		bv, ok := toBool(v)
		return ok && bv == b
	default:
		return toString(v) == str
	}
}

func compareNum(kind string, num float64, _ string, op string, v any) bool {
	if kind != "num" {
		return false // numeric comparison on strings is unsupported
	}
	left := num
	right, ok := toFloat(v)
	if !ok {
		return false
	}
	switch op {
	case "gt":
		return left > right
	case "gte":
		return left >= right
	case "lt":
		return left < right
	case "lte":
		return left <= right
	}
	return false
}

func lookupHeader(h map[string]string, name string) (string, bool) {
	for k, v := range h {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	return "", false
}

func hasTag(obs *pluginsdk.Observation, id string) bool {
	for _, t := range obs.Tags {
		if strings.EqualFold(t.ID, id) || strings.EqualFold(t.Name, id) {
			return true
		}
	}
	return false
}

func findCookie(obs *pluginsdk.Observation, name string) (pluginsdk.Cookie, bool) {
	for _, c := range obs.Cookies {
		if strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return pluginsdk.Cookie{}, false
}

func hasRequestHost(obs *pluginsdk.Observation, host string) bool {
	for _, r := range obs.Requests {
		if strings.Contains(strings.ToLower(r.Host), strings.ToLower(host)) {
			return true
		}
	}
	return false
}

func formatNum(n float64, kind string) string {
	if kind != "num" {
		return ""
	}
	return toString(n)
}

// ---- value coercion helpers (JSON decodes numbers as float64) ----

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// Avoid a trailing ".0" for integral values.
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	}
	return 0, false
}

func toBool(v any) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		if strings.EqualFold(t, "true") {
			return true, true
		}
		if strings.EqualFold(t, "false") {
			return false, true
		}
	}
	return false, false
}

func containsValue(v any, needle string) bool {
	list, ok := v.([]any)
	if !ok {
		// allow a single value
		return toString(v) == needle
	}
	for _, item := range list {
		if toString(item) == needle {
			return true
		}
	}
	return false
}
