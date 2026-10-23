// Package tagpolicy implements the Pageguard tag validation engine. It checks a
// scanned page (a domain.ScanResult) against a user-authored, declarative policy
// document (YAML or JSON) and produces a Report with a pass/fail result per
// rule.
//
// A policy is intentionally data, not code: it can be versioned, reviewed and
// shipped alongside a site's tag-management configuration. Policies cover four
// concerns:
//
//   - must_exist / must_not_exist: expected and forbidden tags, requests,
//     cookies and data-layer events.
//   - value_rules: the shape of data-layer variables (type, enum, range,
//     pattern).
//   - duplicates / orphaned tags: hygiene checks derived from the whole policy.
//   - consent_requirements: that a consent platform is present before gated
//     categories load.
package tagpolicy

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/pageguard/pageguard/internal/domain"
)

// PolicySchemaVersion identifies the TagPolicy document schema. Consumers should
// treat a change of this value as a breaking schema change for policy authors.
const PolicySchemaVersion = "1.0.0"

// Severity is the importance of a rule or a violation. It aliases
// domain.Severity so report results map directly onto findings. The additional
// value SevOff disables a whole category of checks and never surfaces as a
// finding.
type Severity = domain.Severity

// Severity levels accepted in a policy. SevOff is only valid for the global
// settings (unexpected/duplicate checks), not for individual rules.
const (
	SevOff      Severity = "off"
	SevInfo     Severity = domain.SevInfo
	SevLow      Severity = domain.SevLow
	SevMedium   Severity = domain.SevMedium
	SevHigh     Severity = domain.SevHigh
	SevCritical Severity = domain.SevCritical
)

func validSeverity(s Severity, allowOff bool) bool {
	switch s {
	case SevInfo, SevLow, SevMedium, SevHigh, SevCritical:
		return true
	case SevOff:
		return allowOff
	}
	return false
}

// TagCategory aliases domain.TagCategory for use in policy matching.
type TagCategory = domain.TagCategory

// Target selects the kind of evidence an Expectation is matched against.
type Target string

const (
	TargetTag     Target = "tag"
	TargetRequest Target = "request"
	TargetCookie  Target = "cookie"
	TargetEvent   Target = "event"
)

// Policy is a complete tag validation policy.
type Policy struct {
	Version             int                `json:"version" yaml:"version"`
	Name                string             `json:"name" yaml:"name"`
	Description         string             `json:"description,omitempty" yaml:"description,omitempty"`
	Settings            Settings           `json:"settings,omitempty" yaml:"settings,omitempty"`
	MustExist           []Expectation      `json:"must_exist,omitempty" yaml:"must_exist,omitempty"`
	MustNotExist        []Expectation      `json:"must_not_exist,omitempty" yaml:"must_not_exist,omitempty"`
	ValueRules          []ValueRule        `json:"value_rules,omitempty" yaml:"value_rules,omitempty"`
	ConsentRequirements ConsentRequirement `json:"consent_requirements,omitempty" yaml:"consent_requirements,omitempty"`
}

// Settings tune the derived, policy-wide hygiene checks.
type Settings struct {
	// UnexpectedTags is the severity reported for a detected tag that no policy
	// entry declares ("orphaned"). Empty means the default (info); "off"
	// disables the check.
	UnexpectedTags Severity `json:"unexpected_tags,omitempty" yaml:"unexpected_tags,omitempty"`
	// DuplicateTags is the severity reported when a tag id is detected more than
	// MaxOccurrences times. Empty means the default (medium); "off" disables it.
	DuplicateTags Severity `json:"duplicate_tags,omitempty" yaml:"duplicate_tags,omitempty"`
	// UnexpectedCookies is the severity reported for a detected cookie that the
	// policy inventory does not declare. Empty means the default (info); "off"
	// disables it.
	UnexpectedCookies Severity `json:"unexpected_cookies,omitempty" yaml:"unexpected_cookies,omitempty"`
	// MaxOccurrences is the number of times a single tag id may be detected
	// before it is reported as a duplicate. Zero means the default (1).
	MaxOccurrences int `json:"max_occurrences,omitempty" yaml:"max_occurrences,omitempty"`
}

// Expectation is a single presence/absence assertion.
type Expectation struct {
	ID          string      `json:"id" yaml:"id"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Severity    Severity    `json:"severity,omitempty" yaml:"severity,omitempty"`
	Target      Target      `json:"target" yaml:"target"`
	Match       Match       `json:"match" yaml:"match"`
	Attributes  *CookieAttr `json:"attributes,omitempty" yaml:"attributes,omitempty"`

	reTag  *regexp.Regexp
	reName *regexp.Regexp
	reURL  *regexp.Regexp
}

// Match describes what to look for. All populated fields must match (logical
// AND); leave fields empty to ignore them.
type Match struct {
	// Tag matches a tag id exactly (case-insensitive).
	Tag string `json:"tag,omitempty" yaml:"tag,omitempty"`
	// TagPattern is a regular expression matched against the tag id and name.
	TagPattern string `json:"tag_pattern,omitempty" yaml:"tag_pattern,omitempty"`
	// Name matches a tag name, cookie name or (for events) the event name.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// NamePattern is a regular expression matched against a tag/cookie name.
	NamePattern string `json:"name_pattern,omitempty" yaml:"name_pattern,omitempty"`
	// Category matches a tag's category.
	Category TagCategory `json:"category,omitempty" yaml:"category,omitempty"`
	// URLPattern is a regular expression matched against a request URL.
	URLPattern string `json:"url_pattern,omitempty" yaml:"url_pattern,omitempty"`
	// Host matches a request host exactly (case-insensitive).
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	// Domain matches a cookie domain (case-insensitive, exact or suffix).
	Domain string `json:"domain,omitempty" yaml:"domain,omitempty"`
	// Event matches a data-layer event name exactly (case-insensitive).
	Event string `json:"event,omitempty" yaml:"event,omitempty"`
}

// CookieAttr asserts attributes of a declared cookie. Nil pointers are ignored.
type CookieAttr struct {
	Secure     *bool    `json:"secure,omitempty" yaml:"secure,omitempty"`
	HTTPOnly   *bool    `json:"httpOnly,omitempty" yaml:"httpOnly,omitempty"`
	SameSite   []string `json:"sameSite,omitempty" yaml:"sameSite,omitempty"`
	FirstParty *bool    `json:"firstParty,omitempty" yaml:"firstParty,omitempty"`
	Session    *bool    `json:"session,omitempty" yaml:"session,omitempty"`
	// Category labels the cookie for gated_cookie_categories checks. It is
	// policy metadata only and is not asserted against the scan.
	Category string `json:"category,omitempty" yaml:"category,omitempty"`
}

// ValueRule asserts the shape of a data-layer variable addressed by a dotted
// path (for example "cart.total" or "items[0].sku").
type ValueRule struct {
	ID          string   `json:"id" yaml:"id"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Severity    Severity `json:"severity,omitempty" yaml:"severity,omitempty"`
	Variable    string   `json:"variable" yaml:"variable"`
	// Type is one of string, number, integer, boolean, object, array, null.
	Type    string   `json:"type,omitempty" yaml:"type,omitempty"`
	Enum    []any    `json:"enum,omitempty" yaml:"enum,omitempty"`
	Min     *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max     *float64 `json:"max,omitempty" yaml:"max,omitempty"`
	Pattern string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	// Required means the variable must be present in at least one entry in
	// scope. When WhenEvent is set, scope is narrowed to that event, so the
	// variable is only required once the event has fired.
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`
	// WhenEvent narrows the entries the rule applies to.
	WhenEvent string `json:"when_event,omitempty" yaml:"when_event,omitempty"`
	// RequiredIfEvent requires the variable only when that event has fired.
	RequiredIfEvent string `json:"required_if_event,omitempty" yaml:"required_if_event,omitempty"`

	re *regexp.Regexp
}

// ConsentRequirement asserts that a consent platform is present before tags of
// gated categories are allowed to load.
type ConsentRequirement struct {
	// RequireCMP requires a consent platform to be detected on the page.
	RequireCMP bool `json:"require_cmp,omitempty" yaml:"require_cmp,omitempty"`
	// CMPTags lists tag ids that count as a consent platform. When empty any
	// tag in the "consent" category counts.
	CMPTags []string `json:"cmp_tags,omitempty" yaml:"cmp_tags,omitempty"`
	// GatedCategories lists tag categories that must not load without a CMP.
	GatedCategories []TagCategory `json:"gated_categories,omitempty" yaml:"gated_categories,omitempty"`
	// GatedCookieCategories lists cookie categories (as declared in the cookie
	// inventory) that must not be set without a CMP.
	GatedCookieCategories []string `json:"gated_cookie_categories,omitempty" yaml:"gated_cookie_categories,omitempty"`
	Severity              Severity `json:"severity,omitempty" yaml:"severity,omitempty"`
}

// Prepare normalises defaults and compiles the policy's regular expressions. It
// is idempotent and is called automatically by the engine; callers loading a
// policy directly should also call it to surface authoring errors early.
func (p *Policy) Prepare() error {
	var errs []error

	if p.Version == 0 {
		p.Version = 1
	}
	if p.Name == "" {
		errs = append(errs, errors.New("policy: name is required"))
	}

	if p.Settings.MaxOccurrences == 0 {
		p.Settings.MaxOccurrences = 1
	}
	if p.Settings.UnexpectedTags == "" {
		p.Settings.UnexpectedTags = SevInfo
	}
	if p.Settings.DuplicateTags == "" {
		p.Settings.DuplicateTags = SevMedium
	}
	if p.Settings.UnexpectedCookies == "" {
		p.Settings.UnexpectedCookies = SevInfo
	}
	if !validSeverity(p.Settings.UnexpectedTags, true) {
		errs = append(errs, fmt.Errorf("policy: invalid settings.unexpected_tags %q", p.Settings.UnexpectedTags))
	}
	if !validSeverity(p.Settings.DuplicateTags, true) {
		errs = append(errs, fmt.Errorf("policy: invalid settings.duplicate_tags %q", p.Settings.DuplicateTags))
	}
	if !validSeverity(p.Settings.UnexpectedCookies, true) {
		errs = append(errs, fmt.Errorf("policy: invalid settings.unexpected_cookies %q", p.Settings.UnexpectedCookies))
	}

	seen := map[string]bool{}
	checkID := func(section, id string) {
		if id == "" {
			errs = append(errs, fmt.Errorf("policy: %s entry is missing an id", section))
			return
		}
		if seen[id] {
			errs = append(errs, fmt.Errorf("policy: duplicate rule id %q", id))
		}
		seen[id] = true
	}

	for i := range p.MustExist {
		errs = append(errs, p.MustExist[i].prepare("must_exist", checkID)...)
	}
	for i := range p.MustNotExist {
		errs = append(errs, p.MustNotExist[i].prepare("must_not_exist", checkID)...)
	}
	for i := range p.ValueRules {
		errs = append(errs, p.ValueRules[i].prepare(checkID)...)
	}
	if p.ConsentRequirements.Severity == "" {
		p.ConsentRequirements.Severity = SevMedium
	}
	if !validSeverity(p.ConsentRequirements.Severity, false) {
		errs = append(errs, fmt.Errorf("policy: invalid consent_requirements.severity %q", p.ConsentRequirements.Severity))
	}

	return errors.Join(errs...)
}

func (e *Expectation) prepare(section string, checkID func(string, string)) []error {
	var errs []error
	checkID(section, e.ID)
	if e.Severity == "" {
		e.Severity = SevMedium
	}
	if !validSeverity(e.Severity, false) {
		errs = append(errs, fmt.Errorf("policy: %s %q has invalid severity %q", section, e.ID, e.Severity))
	}
	switch e.Target {
	case TargetTag, TargetRequest, TargetCookie, TargetEvent:
	default:
		errs = append(errs, fmt.Errorf("policy: %s %q has invalid target %q", section, e.ID, e.Target))
	}

	compile := func(field, expr string) *regexp.Regexp {
		if expr == "" {
			return nil
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			errs = append(errs, fmt.Errorf("policy: %s %q %s: %w", section, e.ID, field, err))
			return nil
		}
		return re
	}
	e.reTag = compile("tag_pattern", e.Match.TagPattern)
	e.reName = compile("name_pattern", e.Match.NamePattern)
	e.reURL = compile("url_pattern", e.Match.URLPattern)
	return errs
}

func (v *ValueRule) prepare(checkID func(string, string)) []error {
	var errs []error
	checkID("value_rules", v.ID)
	if v.Severity == "" {
		v.Severity = SevMedium
	}
	if !validSeverity(v.Severity, false) {
		errs = append(errs, fmt.Errorf("policy: value_rules %q has invalid severity %q", v.ID, v.Severity))
	}
	if v.Variable == "" {
		errs = append(errs, fmt.Errorf("policy: value_rules %q is missing a variable", v.ID))
	}
	switch v.Type {
	case "", "string", "number", "integer", "boolean", "object", "array", "null":
	default:
		errs = append(errs, fmt.Errorf("policy: value_rules %q has invalid type %q", v.ID, v.Type))
	}
	if v.Pattern != "" {
		re, err := regexp.Compile(v.Pattern)
		if err != nil {
			errs = append(errs, fmt.Errorf("policy: value_rules %q pattern: %w", v.ID, err))
		} else {
			v.re = re
		}
	}
	return errs
}
