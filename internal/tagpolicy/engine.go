package tagpolicy

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// Validate checks a scanned page against a policy and returns a Report. The
// policy is prepared (defaults applied, regular expressions compiled) before
// evaluation; a policy already loaded through LoadFile/Parse is prepared
// already and this is a no-op.
func Validate(sr *domain.ScanResult, p *Policy) *Report {
	_ = p.Prepare()
	r := &Report{
		SchemaVersion: ReportSchemaVersion,
		PolicyName:    p.Name,
		RunID:         sr.RunID,
		PageID:        sr.PageID,
		URL:           sr.URL,
		FinalURL:      sr.FinalURL,
		GeneratedAt:   time.Now().UTC(),
	}
	e := &engine{
		p:         p,
		r:         r,
		tags:      sr.Tags,
		requests:  sr.Requests,
		cookies:   sr.Cookies,
		dataLayer: sr.DataLayer,
	}
	e.runExpectations(p.MustExist, true)
	e.runExpectations(p.MustNotExist, false)
	e.runValueRules()
	e.runDuplicates()
	e.runOrphans()
	e.runCookies()
	e.runConsent()
	r.finalize()
	return r
}

type engine struct {
	p         *Policy
	r         *Report
	tags      []domain.Tag
	requests  []domain.NetworkRequest
	cookies   []domain.Cookie
	dataLayer []domain.DataLayerEntry
}

func (e *engine) runExpectations(list []Expectation, wantPresent bool) {
	section := SectionMustExist
	if !wantPresent {
		section = SectionMustNotExist
	}
	for i := range list {
		exp := &list[i]
		actuals := e.matches(exp)
		present := len(actuals) > 0
		res := RuleResult{
			ID:       exp.ID,
			Section:  section,
			Severity: exp.Severity,
			Expected: expectedRepr(exp),
		}
		switch {
		case wantPresent && present:
			res.Status = StatusPass
			res.Message = fmt.Sprintf("expected %s is present", matchSummary(exp))
			res.Actual = actuals[0]
		case wantPresent && !present:
			res.Status = StatusFail
			res.Message = fmt.Sprintf("expected %s was not found", matchSummary(exp))
		case !wantPresent && !present:
			res.Status = StatusPass
			res.Message = fmt.Sprintf("%s is absent, as required", matchSummary(exp))
		default:
			res.Status = StatusFail
			res.Message = fmt.Sprintf("%s is present but must not be", matchSummary(exp))
			res.Actual = actuals[0]
		}
		e.r.add(res)
		if wantPresent && present && exp.Target == TargetCookie {
			e.checkCookieAttributes(exp)
		}
	}
}

// matches returns JSON-friendly evidence for every item the expectation selects.
func (e *engine) matches(exp *Expectation) []any {
	switch exp.Target {
	case TargetTag:
		var out []any
		for _, t := range e.tags {
			if exp.matchTag(t) {
				out = append(out, tagEvidence(t))
			}
		}
		return out
	case TargetRequest:
		var out []any
		for _, req := range e.requests {
			if exp.matchRequest(req) {
				out = append(out, map[string]any{"url": req.URL, "host": req.Host, "status": req.Status})
			}
		}
		return out
	case TargetCookie:
		var out []any
		for _, c := range e.cookies {
			if exp.matchCookie(c) {
				out = append(out, cookieEvidence(c))
			}
		}
		return out
	case TargetEvent:
		var out []any
		for _, entry := range e.dataLayer {
			ev := entryEvent(entry)
			if ev != "" && exp.matchEvent(ev) {
				out = append(out, ev)
			}
		}
		return out
	}
	return nil
}

func (e *engine) checkCookieAttributes(exp *Expectation) {
	if exp.Attributes == nil {
		return
	}
	for _, c := range e.cookies {
		if !exp.matchCookie(c) {
			continue
		}
		if msg := cookieAttrViolation(exp.Attributes, c); msg != "" {
			e.r.add(RuleResult{
				ID:       exp.ID + "-attributes",
				Section:  SectionMustExist,
				Status:   StatusFail,
				Severity: exp.Severity,
				Message:  msg,
				Actual:   cookieEvidence(c),
			})
		}
	}
}

func (e *engine) runValueRules() {
	for i := range e.p.ValueRules {
		e.r.add(e.evalValueRule(&e.p.ValueRules[i]))
	}
}

func (e *engine) evalValueRule(v *ValueRule) RuleResult {
	res := RuleResult{ID: v.ID, Section: SectionValueRules, Severity: v.Severity, Expected: valueRuleRepr(v)}

	scopeEvent := v.WhenEvent
	if scopeEvent == "" {
		scopeEvent = v.RequiredIfEvent
	}
	var candidates []domain.DataLayerEntry
	for _, entry := range e.dataLayer {
		if scopeEvent == "" || strings.EqualFold(entryEvent(entry), scopeEvent) {
			candidates = append(candidates, entry)
		}
	}
	if scopeEvent != "" && len(candidates) == 0 {
		res.Status = StatusSkipped
		res.Message = fmt.Sprintf("variable %q not evaluated: event %q did not fire", v.Variable, scopeEvent)
		return res
	}

	type resolved struct {
		value any
		name  string
	}
	var values []resolved
	for _, entry := range candidates {
		root, ok := entryValue(entry)
		if !ok {
			continue
		}
		if got, ok := resolvePath(root, v.Variable); ok {
			values = append(values, resolved{value: got, name: entry.Name})
		}
	}
	if len(values) == 0 {
		if v.Required || v.RequiredIfEvent != "" {
			res.Status = StatusFail
			res.Message = fmt.Sprintf("required variable %q is missing", v.Variable)
			return res
		}
		res.Status = StatusSkipped
		res.Message = fmt.Sprintf("optional variable %q is not present", v.Variable)
		return res
	}
	for _, rv := range values {
		if msg := checkValue(v, rv.value); msg != "" {
			res.Status = StatusFail
			res.Message = msg
			res.Actual = rv.value
			res.Evidence = map[string]any{"dataLayerName": rv.name, "value": rv.value}
			return res
		}
	}
	res.Status = StatusPass
	res.Message = fmt.Sprintf("variable %q satisfies its constraints", v.Variable)
	res.Actual = values[0].value
	return res
}

func (e *engine) runDuplicates() {
	sev := e.p.Settings.DuplicateTags
	if sev == "" || sev == SevOff {
		return
	}
	max := e.p.Settings.MaxOccurrences
	counts := map[string]int{}
	names := map[string]string{}
	for _, t := range e.tags {
		counts[t.ID]++
		names[t.ID] = t.Name
	}
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if counts[id] <= max {
			continue
		}
		e.r.add(RuleResult{
			ID:       "duplicate-tag-" + id,
			Section:  SectionDuplicates,
			Status:   StatusFail,
			Severity: sev,
			Message:  fmt.Sprintf("tag %q (%s) detected %d times (max %d)", id, names[id], counts[id], max),
			Expected: map[string]any{"max_occurrences": max},
			Actual:   map[string]any{"count": counts[id]},
			Evidence: map[string]any{"tagId": id, "count": counts[id]},
		})
	}
}

func (e *engine) runOrphans() {
	sev := e.p.Settings.UnexpectedTags
	if sev == "" || sev == SevOff {
		return
	}
	for _, t := range e.tags {
		if e.tagDeclared(t) {
			continue
		}
		e.r.add(RuleResult{
			ID:       "orphan-tag-" + t.ID,
			Section:  SectionOrphaned,
			Status:   StatusFail,
			Severity: sev,
			Message:  fmt.Sprintf("tag %q (%s) is not declared by the policy", t.ID, t.Name),
			Evidence: tagEvidence(t),
		})
	}
}

func (e *engine) tagDeclared(t domain.Tag) bool {
	for i := range e.p.MustExist {
		if e.p.MustExist[i].Target == TargetTag && e.p.MustExist[i].matchTag(t) {
			return true
		}
	}
	for i := range e.p.MustNotExist {
		if e.p.MustNotExist[i].Target == TargetTag && e.p.MustNotExist[i].matchTag(t) {
			return true
		}
	}
	return false
}

func (e *engine) runCookies() {
	sev := e.p.Settings.UnexpectedCookies
	if sev == "" || sev == SevOff {
		return
	}
	for _, c := range e.cookies {
		if e.cookieDeclared(c) {
			continue
		}
		e.r.add(RuleResult{
			ID:       "unexpected-cookie-" + c.Name,
			Section:  SectionCookies,
			Status:   StatusFail,
			Severity: sev,
			Message:  fmt.Sprintf("cookie %q is not declared in the policy inventory", c.Name),
			Evidence: cookieEvidence(c),
		})
	}
}

func (e *engine) cookieDeclared(c domain.Cookie) bool {
	for _, list := range [][]Expectation{e.p.MustExist, e.p.MustNotExist} {
		for i := range list {
			if list[i].Target == TargetCookie && list[i].matchCookie(c) {
				return true
			}
		}
	}
	return false
}

func (e *engine) runConsent() {
	c := e.p.ConsentRequirements
	if !c.RequireCMP && len(c.GatedCategories) == 0 && len(c.GatedCookieCategories) == 0 {
		return
	}
	cmp, cmpTag := e.cmpPresent(c)
	if c.RequireCMP {
		res := RuleResult{
			ID:       "consent-cmp",
			Section:  SectionConsent,
			Severity: c.Severity,
			Expected: map[string]any{"require_cmp": true},
		}
		if cmp {
			res.Status = StatusPass
			res.Message = "consent platform (CMP) detected"
			res.Actual = cmpTag
		} else {
			res.Status = StatusFail
			res.Message = "no consent platform (CMP) detected"
		}
		e.r.add(res)
	}
	if cmp {
		return
	}
	for _, cat := range c.GatedCategories {
		var found []any
		for _, t := range e.tags {
			if strings.EqualFold(string(t.Category), string(cat)) {
				found = append(found, tagEvidence(t))
			}
		}
		if len(found) > 0 {
			e.r.add(RuleResult{
				ID:       "consent-gated-" + string(cat),
				Section:  SectionConsent,
				Status:   StatusFail,
				Severity: c.Severity,
				Message:  fmt.Sprintf("category %q tags loaded without a consent platform", cat),
				Evidence: map[string]any{"category": string(cat), "tags": found},
			})
		}
	}
	for _, cat := range c.GatedCookieCategories {
		var found []any
		for _, ck := range e.cookies {
			if strings.EqualFold(e.cookieCategory(ck), cat) {
				found = append(found, cookieEvidence(ck))
			}
		}
		if len(found) > 0 {
			e.r.add(RuleResult{
				ID:       "consent-gated-cookie-" + cat,
				Section:  SectionConsent,
				Status:   StatusFail,
				Severity: c.Severity,
				Message:  fmt.Sprintf("cookie category %q set without a consent platform", cat),
				Evidence: map[string]any{"category": cat, "cookies": found},
			})
		}
	}
}

func (e *engine) cmpPresent(c ConsentRequirement) (bool, any) {
	for _, t := range e.tags {
		if len(c.CMPTags) > 0 {
			for _, id := range c.CMPTags {
				if strings.EqualFold(t.ID, id) {
					return true, tagEvidence(t)
				}
			}
			continue
		}
		if strings.EqualFold(string(t.Category), string(domain.TagConsent)) {
			return true, tagEvidence(t)
		}
	}
	return false, nil
}

func (e *engine) cookieCategory(c domain.Cookie) string {
	for _, list := range [][]Expectation{e.p.MustExist, e.p.MustNotExist} {
		for i := range list {
			if list[i].Target == TargetCookie && list[i].matchCookie(c) && list[i].Attributes != nil {
				return list[i].Attributes.Category
			}
		}
	}
	return ""
}

// --- value checking -------------------------------------------------------

func checkValue(v *ValueRule, val any) string {
	if !checkType(v.Type, val) {
		return fmt.Sprintf("variable %q expected type %s, got %s", v.Variable, v.Type, typeName(val))
	}
	if len(v.Enum) > 0 {
		ok := false
		for _, want := range v.Enum {
			if jsonEqual(want, val) {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Sprintf("variable %q value %s is not one of the allowed values", v.Variable, render(val))
		}
	}
	if v.Min != nil || v.Max != nil {
		f, ok := toFloat(val)
		if !ok {
			return fmt.Sprintf("variable %q expected a number for range check, got %s", v.Variable, typeName(val))
		}
		if v.Min != nil && f < *v.Min {
			return fmt.Sprintf("variable %q value %s is below minimum %v", v.Variable, render(val), *v.Min)
		}
		if v.Max != nil && f > *v.Max {
			return fmt.Sprintf("variable %q value %s is above maximum %v", v.Variable, render(val), *v.Max)
		}
	}
	if v.re != nil {
		s, ok := val.(string)
		if !ok {
			return fmt.Sprintf("variable %q expected a string for pattern check, got %s", v.Variable, typeName(val))
		}
		if !v.re.MatchString(s) {
			return fmt.Sprintf("variable %q value %q does not match pattern %q", v.Variable, s, v.Pattern)
		}
	}
	return ""
}

func checkType(t string, v any) bool {
	switch t {
	case "":
		return true
	case "null":
		return v == nil
	case "string":
		_, ok := v.(string)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "number":
		_, ok := toFloat(v)
		return ok
	case "integer":
		f, ok := toFloat(v)
		return ok && f == math.Trunc(f)
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	}
	return false
}

func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	}
	if _, ok := toFloat(v); ok {
		return "number"
	}
	return fmt.Sprintf("%T", v)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func jsonEqual(a, b any) bool {
	if af, aok := toFloat(a); aok {
		if bf, bok := toFloat(b); bok {
			return af == bf
		}
	}
	return reflect.DeepEqual(a, b)
}

func render(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// --- evidence -------------------------------------------------------------

func tagEvidence(t domain.Tag) map[string]any {
	return map[string]any{
		"id":         t.ID,
		"name":       t.Name,
		"vendor":     t.Vendor,
		"category":   string(t.Category),
		"confidence": t.Confidence,
	}
}

func cookieEvidence(c domain.Cookie) map[string]any {
	return map[string]any{
		"name":       c.Name,
		"domain":     c.Domain,
		"secure":     c.Secure,
		"httpOnly":   c.HTTPOnly,
		"sameSite":   c.SameSite,
		"session":    c.Session,
		"firstParty": c.FirstParty,
	}
}

func cookieAttrViolation(a *CookieAttr, c domain.Cookie) string {
	switch {
	case a.Secure != nil && c.Secure != *a.Secure:
		return fmt.Sprintf("cookie %q secure=%v, expected %v", c.Name, c.Secure, *a.Secure)
	case a.HTTPOnly != nil && c.HTTPOnly != *a.HTTPOnly:
		return fmt.Sprintf("cookie %q httpOnly=%v, expected %v", c.Name, c.HTTPOnly, *a.HTTPOnly)
	case a.Session != nil && c.Session != *a.Session:
		return fmt.Sprintf("cookie %q session=%v, expected %v", c.Name, c.Session, *a.Session)
	case a.FirstParty != nil && c.FirstParty != *a.FirstParty:
		return fmt.Sprintf("cookie %q firstParty=%v, expected %v", c.Name, c.FirstParty, *a.FirstParty)
	}
	if len(a.SameSite) > 0 {
		ok := false
		for _, s := range a.SameSite {
			if strings.EqualFold(c.SameSite, s) {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Sprintf("cookie %q sameSite=%q, expected one of %v", c.Name, c.SameSite, a.SameSite)
		}
	}
	return ""
}

// --- descriptions ---------------------------------------------------------

func matchSummary(exp *Expectation) string {
	m := exp.Match
	var parts []string
	if m.Tag != "" {
		parts = append(parts, "id="+m.Tag)
	}
	if m.TagPattern != "" {
		parts = append(parts, "id~="+m.TagPattern)
	}
	if m.Name != "" {
		parts = append(parts, "name="+m.Name)
	}
	if m.NamePattern != "" {
		parts = append(parts, "name~="+m.NamePattern)
	}
	if m.Category != "" {
		parts = append(parts, "category="+string(m.Category))
	}
	if m.URLPattern != "" {
		parts = append(parts, "url~="+m.URLPattern)
	}
	if m.Host != "" {
		parts = append(parts, "host="+m.Host)
	}
	if m.Domain != "" {
		parts = append(parts, "domain="+m.Domain)
	}
	if m.Event != "" {
		parts = append(parts, "event="+m.Event)
	}
	if len(parts) == 0 {
		parts = append(parts, "any")
	}
	return string(exp.Target) + "(" + strings.Join(parts, ", ") + ")"
}

func expectedRepr(exp *Expectation) map[string]any {
	out := map[string]any{"target": string(exp.Target)}
	m := map[string]any{}
	if exp.Match.Tag != "" {
		m["tag"] = exp.Match.Tag
	}
	if exp.Match.TagPattern != "" {
		m["tag_pattern"] = exp.Match.TagPattern
	}
	if exp.Match.Name != "" {
		m["name"] = exp.Match.Name
	}
	if exp.Match.NamePattern != "" {
		m["name_pattern"] = exp.Match.NamePattern
	}
	if exp.Match.Category != "" {
		m["category"] = string(exp.Match.Category)
	}
	if exp.Match.URLPattern != "" {
		m["url_pattern"] = exp.Match.URLPattern
	}
	if exp.Match.Host != "" {
		m["host"] = exp.Match.Host
	}
	if exp.Match.Domain != "" {
		m["domain"] = exp.Match.Domain
	}
	if exp.Match.Event != "" {
		m["event"] = exp.Match.Event
	}
	if len(m) > 0 {
		out["match"] = m
	}
	return out
}

func valueRuleRepr(v *ValueRule) map[string]any {
	out := map[string]any{"variable": v.Variable}
	if v.Type != "" {
		out["type"] = v.Type
	}
	if len(v.Enum) > 0 {
		out["enum"] = v.Enum
	}
	if v.Min != nil {
		out["min"] = *v.Min
	}
	if v.Max != nil {
		out["max"] = *v.Max
	}
	if v.Pattern != "" {
		out["pattern"] = v.Pattern
	}
	return out
}
