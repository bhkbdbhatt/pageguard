// Package pluginsdk defines the public contract for Pageguard plugins.
//
// Plugins extend the platform in four ways:
//
//   - Validators  : turn a page Observation into Findings (custom rules)
//   - Reporters   : render a completed run (PDF, HTML, Jira export, ...)
//   - Integrations: deliver alert events to an external system (Slack, PagerDuty)
//   - Detectors   : recognise additional tags/cookies (Module 2+)
//
// Plugins may be implemented in-process in Go (by satisfying the interfaces in
// this package) or out-of-process in TypeScript (by speaking the newline-
// delimited JSON protocol described by ProtocolVersion over stdin/stdout).
// The wire types below are the stable, public data contract for both.
package pluginsdk

// ProtocolVersion is the version of the stdio JSON protocol. The host refuses
// to load out-of-process plugins that advertise an incompatible major version.
const ProtocolVersion = 1

// Kind classifies a plugin.
type Kind string

const (
	KindValidator   Kind = "validator"
	KindReporter    Kind = "reporter"
	KindIntegration Kind = "integration"
	KindDetector    Kind = "detector"
)

// Manifest describes a plugin. For out-of-process plugins it is read from
// `manifest.json` in the plugin directory.
type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Kind        Kind   `json:"kind"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	License     string `json:"license,omitempty"`
	// Runtime is "builtin" for in-process plugins or "node" for out-of-process.
	Runtime string `json:"runtime"`
	// Entry is the JS entrypoint, relative to the plugin directory (runtime=node).
	Entry string `json:"entry,omitempty"`
	// Rules are declarative rule specs (Kind == validator, runtime == builtin).
	Rules []RuleSpec `json:"rules,omitempty"`
	// Permissions is an allowlist hint (e.g. "network", "filesystem").
	Permissions  []string       `json:"permissions,omitempty"`
	ConfigSchema map[string]any `json:"configSchema,omitempty"`
}

// RuleSpec is a single declarative validation rule.
type RuleSpec struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category,omitempty"`
	Remediation string    `json:"remediation,omitempty"`
	Condition   Condition `json:"condition"`
}

// Condition is a serialisable predicate over an Observation.
//
// Semantics:
//   - All predicates must hold.
//   - If Any is non-empty, at least one must hold.
//   - No predicate in None may hold.
type Condition struct {
	All  []Predicate `json:"all,omitempty"`
	Any  []Predicate `json:"any,omitempty"`
	None []Predicate `json:"none,omitempty"`
}

// Predicate is one leaf comparison. Field selects the value; Name is an
// optional sub-selector (header name, tag id, cookie name, cookie attribute).
type Predicate struct {
	Field string `json:"field"`
	Name  string `json:"name,omitempty"`
	Op    string `json:"op"`
	Value any    `json:"value,omitempty"`
}
