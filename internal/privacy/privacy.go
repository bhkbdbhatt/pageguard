// Package privacy implements the Pageguard privacy compliance module (Module 3).
//
// Given a page's cookie inventory and third-party script list, it produces a
// compliance report per regulation (GDPR, CCPA/CPRA, LGPD, ePrivacy). It works
// by simulating consent states — accept-all, reject-all, necessary-only and the
// Global Privacy Control (GPC) opt-out — re-scanning the page under each, and
// flagging any tracker that fires when it should have been blocked. Results are
// itemised as violations and rolled up into a 0–100 score per regulation.
//
// The package is library-first and dependency-light: it depends only on the
// standard library and internal/domain. The browser side is abstracted behind
// the Rescanner interface (simulate.go); the engine (engine.go) is pure. As with
// the tag-policy and journey modules, this package does not (yet) persist or
// serve reports — see docs/privacy.md for the wiring plan.
package privacy

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
)

// SchemaVersion identifies the Config document schema.
const SchemaVersion = "1.0.0"

// MaxConfigBytes caps the size of a privacy policy document accepted by Parse.
const MaxPolicyBytes = 1 << 20 // 1 MiB

// ConsentState models the visitor's consent choice that a page is re-scanned
// under.
type ConsentState string

const (
	// StateAcceptAll grants every consent category.
	StateAcceptAll ConsentState = "accept_all"
	// StateRejectAll denies every non-essential category.
	StateRejectAll ConsentState = "reject_all"
	// StateNecessaryOnly allows strictly necessary processing only.
	StateNecessaryOnly ConsentState = "necessary_only"
	// StateGPC applies a Global Privacy Control opt-out signal.
	StateGPC ConsentState = "gpc"
)

// AllStates is the default set of consent states simulated for every run.
var AllStates = []ConsentState{StateAcceptAll, StateRejectAll, StateNecessaryOnly, StateGPC}

// blockingStates are the states in which non-essential trackers must not fire.
func (s ConsentState) expectingBlock() bool {
	switch s {
	case StateRejectAll, StateNecessaryOnly, StateGPC:
		return true
	}
	return false
}

// Regulation identifies a privacy regime.
type Regulation string

const (
	GDPR     Regulation = "gdpr"
	CCPA     Regulation = "ccpa"
	LGPD     Regulation = "lgpd"
	EPrivacy Regulation = "eprivacy"
)

// AllRegulations lists every supported regime in evaluation order.
var AllRegulations = []Regulation{GDPR, CCPA, LGPD, EPrivacy}

func (r Regulation) valid() bool {
	for _, x := range AllRegulations {
		if r == x {
			return true
		}
	}
	return false
}

// Config is a user-authored privacy compliance policy.
type Config struct {
	Version     int    `json:"version" yaml:"version"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Regulations selects which regimes to evaluate. Defaults to all.
	Regulations []Regulation `json:"regulations,omitempty" yaml:"regulations,omitempty"`
	// CookieRules classify cookies by name pattern into categories and/or mark
	// them essential.
	CookieRules []CookieRule `json:"cookie_rules,omitempty" yaml:"cookie_rules,omitempty"`
	// SubjectRights are the data-subject rights LGPD expects to be reachable
	// from the page.
	SubjectRights []SubjectRight `json:"subject_rights,omitempty" yaml:"subject_rights,omitempty"`
	Settings      Settings       `json:"settings,omitempty" yaml:"settings,omitempty"`

	essentialCookieRe []*regexp.Regexp
	cookieRuleRe      []*regexp.Regexp
	optOutRe          []*regexp.Regexp
	rightsRe          []*regexp.Regexp
	crossBorderLower  []string
	essentialHosts    []string
}

// CookieRule classifies a cookie whose name matches NamePattern (a regular
// expression).
type CookieRule struct {
	NamePattern string             `json:"name_pattern" yaml:"name_pattern"`
	Category    domain.TagCategory `json:"category,omitempty" yaml:"category,omitempty"`
	Essential   bool               `json:"essential,omitempty" yaml:"essential,omitempty"`
}

// SubjectRight is a data-subject right (e.g. access, deletion) expected to be
// discoverable on the page.
type SubjectRight struct {
	ID         string `json:"id" yaml:"id"`
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	URLPattern string `json:"url_pattern" yaml:"url_pattern"`
}

// DPIATrigger flags a Data Protection Impact Assessment. It fires when a tag in
// one of Categories is observed, or when at least MinTrackers non-essential
// trackers fire (0 disables the count trigger).
type DPIATrigger struct {
	ID          string               `json:"id" yaml:"id"`
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Categories  []domain.TagCategory `json:"categories,omitempty" yaml:"categories,omitempty"`
	MinTrackers int                  `json:"min_trackers,omitempty" yaml:"min_trackers,omitempty"`
}

// Settings hold classification and threshold configuration.
type Settings struct {
	// ConsentStates restricts which consent states are simulated. Defaults to
	// all four.
	ConsentStates []ConsentState `json:"consent_states,omitempty" yaml:"consent_states,omitempty"`

	CMPTags        []string `json:"cmp_tags,omitempty" yaml:"cmp_tags,omitempty"`
	CMPCookieNames []string `json:"cmp_cookie_names,omitempty" yaml:"cmp_cookie_names,omitempty"`

	// EssentialCategories never require consent. Defaults to [consent].
	EssentialCategories []domain.TagCategory `json:"essential_categories,omitempty" yaml:"essential_categories,omitempty"`
	// EssentialCookies / EssentialCookiePatterns list cookies allowed without
	// consent (exact names and regexes respectively).
	EssentialCookies        []string `json:"essential_cookies,omitempty" yaml:"essential_cookies,omitempty"`
	EssentialCookiePatterns []string `json:"essential_cookie_patterns,omitempty" yaml:"essential_cookie_patterns,omitempty"`
	// EssentialHosts are third-party hosts treated as necessary.
	EssentialHosts []string `json:"essential_hosts,omitempty" yaml:"essential_hosts,omitempty"`

	// MarketingCategories are categories considered marketing (ePrivacy).
	// Defaults to [advertising, social].
	MarketingCategories []domain.TagCategory `json:"marketing_categories,omitempty" yaml:"marketing_categories,omitempty"`
	// SaleCategories are categories treated as a "sale"/"share" under CCPA/CPRA.
	// Defaults to [advertising].
	SaleCategories []domain.TagCategory `json:"sale_categories,omitempty" yaml:"sale_categories,omitempty"`

	// DPIA lists Data Protection Impact Assessment triggers.
	DPIA []DPIATrigger `json:"dpia,omitempty" yaml:"dpia,omitempty"`

	// CrossBorderHosts are host substrings indicating a transfer outside the
	// home region. Presence of any triggers a cross-border flag.
	CrossBorderHosts []string `json:"cross_border_hosts,omitempty" yaml:"cross_border_hosts,omitempty"`

	// OptOutURLPatterns are patterns identifying a "Do Not Sell / Share" or
	// opt-out link (CCPA/CPRA).
	OptOutURLPatterns []string `json:"opt_out_url_patterns,omitempty" yaml:"opt_out_url_patterns,omitempty"`

	// TrackThirdPartyRequests also treats non-essential third-party requests as
	// trackers. Defaults to true.
	TrackThirdPartyRequests *bool `json:"track_third_party_requests,omitempty" yaml:"track_third_party_requests,omitempty"`

	// PassThreshold is the minimum regulation score (0–100) for a pass.
	// Defaults to 100.
	PassThreshold int `json:"pass_threshold,omitempty" yaml:"pass_threshold,omitempty"`
}

// Prepare fills defaults, compiles regular expressions and validates the config.
// It is idempotent and called automatically by Evaluate.
func (c *Config) Prepare() error {
	var errs []string
	if c.Version == 0 {
		c.Version = 1
	}
	if strings.TrimSpace(c.Name) == "" {
		errs = append(errs, "config: name is required")
	}
	if len(c.Regulations) == 0 {
		c.Regulations = append([]Regulation(nil), AllRegulations...)
	}
	for _, r := range c.Regulations {
		if !r.valid() {
			errs = append(errs, fmt.Sprintf("config: unknown regulation %q", r))
		}
	}
	if len(c.Settings.ConsentStates) == 0 {
		c.Settings.ConsentStates = append([]ConsentState(nil), allStates()...)
	}
	for _, s := range c.Settings.ConsentStates {
		if !s.validState() {
			errs = append(errs, fmt.Sprintf("config: unknown consent state %q", s))
		}
	}
	if len(c.Settings.EssentialCategories) == 0 {
		c.Settings.EssentialCategories = []domain.TagCategory{domain.TagConsent}
	}
	if len(c.Settings.MarketingCategories) == 0 {
		c.Settings.MarketingCategories = []domain.TagCategory{domain.TagAdvertising, domain.TagSocial}
	}
	if len(c.Settings.SaleCategories) == 0 {
		c.Settings.SaleCategories = []domain.TagCategory{domain.TagAdvertising}
	}
	if c.Settings.PassThreshold <= 0 {
		c.Settings.PassThreshold = 100
	}
	if c.Settings.PassThreshold > 100 {
		c.Settings.PassThreshold = 100
	}

	if c.Settings.EssentialCookiePatterns != nil {
		c.essentialCookieRe = c.essentialCookieRe[:0]
		for _, p := range c.Settings.EssentialCookiePatterns {
			re, err := regexp.Compile(p)
			if err != nil {
				errs = append(errs, fmt.Sprintf("config: essential_cookie_patterns %q: %v", p, err))
				continue
			}
			c.essentialCookieRe = append(c.essentialCookieRe, re)
		}
	}
	if c.CookieRules != nil {
		c.cookieRuleRe = c.cookieRuleRe[:0]
		for _, rule := range c.CookieRules {
			re, err := regexp.Compile(rule.NamePattern)
			if err != nil {
				errs = append(errs, fmt.Sprintf("config: cookie_rule %q: %v", rule.NamePattern, err))
				continue
			}
			c.cookieRuleRe = append(c.cookieRuleRe, re)
		}
	}
	if c.Settings.OptOutURLPatterns != nil {
		c.optOutRe = c.optOutRe[:0]
		for _, p := range c.Settings.OptOutURLPatterns {
			re, err := regexp.Compile(p)
			if err != nil {
				errs = append(errs, fmt.Sprintf("config: opt_out_url_patterns %q: %v", p, err))
				continue
			}
			c.optOutRe = append(c.optOutRe, re)
		}
	}
	if c.SubjectRights != nil {
		c.rightsRe = c.rightsRe[:0]
		for _, sr := range c.SubjectRights {
			if strings.TrimSpace(sr.ID) == "" {
				errs = append(errs, "config: subject_rights entry missing id")
				continue
			}
			re, err := regexp.Compile(sr.URLPattern)
			if err != nil {
				errs = append(errs, fmt.Sprintf("config: subject_rights %q: %v", sr.ID, err))
				continue
			}
			c.rightsRe = append(c.rightsRe, re)
		}
	}

	c.crossBorderLower = lowerAll(c.Settings.CrossBorderHosts)
	c.essentialHosts = lowerAll(c.Settings.EssentialHosts)

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func allStates() []ConsentState {
	return []ConsentState{StateAcceptAll, StateRejectAll, StateNecessaryOnly, StateGPC}
}

func (s ConsentState) validState() bool {
	switch s {
	case StateAcceptAll, StateRejectAll, StateNecessaryOnly, StateGPC:
		return true
	}
	return false
}

func (c *Config) tracksThirdPartyRequests() bool {
	return c.Settings.TrackThirdPartyRequests == nil || *c.Settings.TrackThirdPartyRequests
}

func (c *Config) categoryEssential(cat domain.TagCategory) bool {
	for _, e := range c.Settings.EssentialCategories {
		if e == cat {
			return true
		}
	}
	return false
}

func (c *Config) categoryMarketing(cat domain.TagCategory) bool {
	for _, e := range c.Settings.MarketingCategories {
		if e == cat {
			return true
		}
	}
	return false
}

func (c *Config) categorySale(cat domain.TagCategory) bool {
	for _, e := range c.Settings.SaleCategories {
		if e == cat {
			return true
		}
	}
	return false
}

// cookieRuleCategory returns the category of the first matching cookie rule.
func (c *Config) cookieRuleCategory(name string) (domain.TagCategory, bool) {
	for i, re := range c.cookieRuleRe {
		if re.MatchString(name) {
			return c.CookieRules[i].Category, true
		}
	}
	return "", false
}

func (c *Config) cookieEssential(name string) bool {
	for _, n := range c.Settings.EssentialCookies {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	for _, re := range c.essentialCookieRe {
		if re.MatchString(name) {
			return true
		}
	}
	for i, re := range c.cookieRuleRe {
		if re.MatchString(name) {
			return c.CookieRules[i].Essential
		}
	}
	return false
}

func (c *Config) cookieCategory(name string) (domain.TagCategory, bool) {
	return c.cookieRuleCategory(name)
}

func (c *Config) hostEssential(host string) bool {
	host = strings.ToLower(host)
	for _, h := range c.essentialHosts {
		if h != "" && strings.Contains(host, h) {
			return true
		}
	}
	return false
}

func (c *Config) isCrossBorder(host string) bool {
	host = strings.ToLower(host)
	for _, h := range c.crossBorderLower {
		if h != "" && strings.Contains(host, h) {
			return true
		}
	}
	return false
}

func (c *Config) matchesOptOut(s string) bool {
	for _, re := range c.optOutRe {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func lowerAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, strings.ToLower(t))
		}
	}
	return out
}
