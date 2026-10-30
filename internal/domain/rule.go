package domain

import (
	"strings"
	"time"
)

// RuleTarget selects the evidence an AuditRule is evaluated against.
type RuleTarget string

const (
	// RuleTargetRequest evaluates a query parameter on matching network requests.
	RuleTargetRequest RuleTarget = "request"
	// RuleTargetEvent evaluates a value in the captured client-side data layer.
	RuleTargetEvent RuleTarget = "event"
	// RuleTargetTag evaluates a detected tag.
	RuleTargetTag RuleTarget = "tag"
	// RuleTargetCookie evaluates an observed cookie.
	RuleTargetCookie RuleTarget = "cookie"
	// RuleTargetBeacon evaluates tracker-beacon requests only (martech-classified
	// pixels/xhr/fetch/beacon traffic), optionally a query parameter on them.
	RuleTargetBeacon RuleTarget = "beacon"
	// RuleTargetNetwork evaluates all captured network requests, including the
	// page document itself.
	RuleTargetNetwork RuleTarget = "network"
	// RuleTargetPage evaluates the page URL itself (redirect-aware final URL).
	RuleTargetPage RuleTarget = "page"
)

// RuleTargetAliases maps dashboard/compatibility spellings onto canonical
// targets. Applied when rules are created or updated.
var RuleTargetAliases = map[RuleTarget]RuleTarget{
	"datalayer": RuleTargetEvent,
	"script":    RuleTargetTag,
}

// CanonicalizeTarget resolves an alias and reports whether the result is a
// supported target.
func CanonicalizeTarget(t RuleTarget) (RuleTarget, bool) {
	normalized := RuleTarget(strings.ToLower(strings.TrimSpace(string(t))))
	if aliased, ok := RuleTargetAliases[normalized]; ok {
		normalized = aliased
	}
	switch normalized {
	case RuleTargetRequest, RuleTargetEvent, RuleTargetTag, RuleTargetCookie,
		RuleTargetBeacon, RuleTargetNetwork, RuleTargetPage:
		return normalized, true
	}
	return normalized, false
}

// RuleOperator is the assertion applied by an AuditRule.
type RuleOperator string

const (
	RuleOpExists     RuleOperator = "exists"
	RuleOpNotExists  RuleOperator = "not_exists"
	RuleOpEquals     RuleOperator = "equals"
	RuleOpNotEquals  RuleOperator = "not_equals"
	RuleOpMatches    RuleOperator = "matches"
	RuleOpNotMatches RuleOperator = "not_matches"
)

// AuditRule is a persisted, user-authored tag-governance rule. Rules are
// evaluated against every fresh page scan during a run and can be edited without
// restarting the server.
type AuditRule struct {
	ID          ID           `json:"id"`
	ProjectID   ID           `json:"projectId"`
	Name        string       `json:"name"`
	Vendor      string       `json:"vendor"`
	Enabled     bool         `json:"enabled"`
	URLPattern  string       `json:"urlPattern"`
	Target      RuleTarget   `json:"target"`
	Parameter   string       `json:"parameter"`
	Event       string       `json:"event,omitempty"`
	Operator    RuleOperator `json:"operator"`
	Expected    string       `json:"expected,omitempty"`
	Severity    Severity     `json:"severity"`
	Description string       `json:"description,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}
