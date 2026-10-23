# Privacy Compliance

The privacy engine (`internal/privacy`) verifies that a page behaves correctly
under every consent choice a visitor can make. It **simulates consent states** —
`accept_all`, `reject_all`, `necessary_only` and a Global Privacy Control
(`gpc`) opt-out — re-scans the page under each, and checks that no non-essential
tag, cookie or third-party request fires when it should have been blocked.

The engine evaluates one or more **regulations** (GDPR, CCPA/CPRA, LGPD,
ePrivacy), produces a 0–100 score for each, and emits advisory **flags** such as
DPIA triggers or cross-border transfers. The result is a per-regulation
**ComplianceReport** that can be turned into `domain.Finding`s for the shared
persistence and streaming path.

The package is library-first: it depends only on the standard library plus
`internal/domain`. The browser is abstracted behind the `Rescanner` interface,
so the engine is fully testable offline with `FakeRescanner`.

## Quick start

```go
cfg, _ := privacy.LoadFile("examples/privacy/acme.yaml")

rescanner := privacy.RescanFunc(func(ctx context.Context, state privacy.ConsentState) (*privacy.Observation, error) {
    return scanUnder(ctx, state) // your browser-based capture
})

report, _ := privacy.Evaluate(ctx, cfg, rescanner)
findings := report.Findings() // []domain.Finding, PluginID "privacy"
```

```bash
go run ./cmd/privacy --policy examples/privacy/acme.yaml \
  --observations examples/privacy/observations.json --format text
# exits 0 when every regulation passes, 1 otherwise
```

Policies may be authored in **YAML** (recommended) or **JSON**. Both encodings
are validated against `schemas/privacy.schema.json`; the report is described by
`schemas/privacyreport.schema.json`.

## The policy document

```yaml
version: 1
name: "Acme privacy policy"
description: "GDPR / CCPA / ePrivacy compliance baseline"

regulations: [gdpr, ccpa, eprivacy]     # default: all four

cookie_rules:                            # classify cookies by name regex
  - name_pattern: "^_ga"
    category: analytics                  # analytics|advertising|social|…
  - name_pattern: "(?i)^(sid|session)"
    essential: true                      # allowed without consent

subject_rights:                          # LGPD: reachable data-subject rights
  - id: access
    name: "Right of access"
    url_pattern: "(?i)/(privacy|data)-request"

settings:
  consent_states: [accept_all, reject_all, necessary_only, gpc]
  essential_categories: [consent]
  essential_cookies: [csrf_token]
  essential_cookie_patterns: ["^__Host-"]
  essential_hosts: [cdn.acme.com]
  marketing_categories: [advertising, social]
  sale_categories: [advertising]
  cmp_tags: [onetrust, cookiebot, usercentrics]
  opt_out_url_patterns: ["(?i)do-not-sell", "(?i)/opt-out"]
  cross_border_hosts: [example-cdn.net, adnetwork.io]
  dpia:
    - id: large-scale-tracking
      description: "large-scale behavioural tracking"
      categories: [advertising, social]
      min_trackers: 5
  pass_threshold: 100
```

A complete, runnable example is `examples/privacy/acme.yaml` (and the identical
`acme.json`).

### `settings`

| Field | Meaning | Default |
| --- | --- | --- |
| `consent_states` | Which states to simulate. | all four |
| `essential_categories` | Tag categories allowed without consent. | `[consent]` |
| `essential_cookies` | Exact cookie names allowed without consent. | — |
| `essential_cookie_patterns` | Cookie-name regexes allowed without consent. | — |
| `essential_hosts` | Third-party hosts treated as necessary. | — |
| `marketing_categories` | Categories considered marketing. | `[advertising, social]` |
| `sale_categories` | Categories treated as a "sale"/"share" (CCPA). | `[advertising]` |
| `cmp_tags` | Tag ids that identify a consent platform. | — |
| `opt_out_url_patterns` | URL/link regexes for a "Do Not Sell / Share" mechanism. | — |
| `cross_border_hosts` | Host substrings implying an out-of-region transfer. | — |
| `dpia` | Data Protection Impact Assessment triggers. | — |
| `track_third_party_requests` | Treat non-essential third-party requests as trackers. | `true` |
| `pass_threshold` | Minimum per-regulation score for a pass. | `100` |

A `DPIATrigger` fires when an observed tag matches one of its `categories`, or
when at least `min_trackers` tags are present (`0` disables the count trigger).

## The report

```jsonc
{
  "schemaVersion": "1.0.0",
  "policyName": "Acme privacy policy",
  "url": "https://acme.com/",
  "generatedAt": "2026-01-01T00:00:00Z",
  "score": 100,                    // mean of the regulation scores
  "passed": true,                  // every regulation >= pass_threshold
  "summary": { "regulations": 3, "passed": 3, "failed": 0, "violations": 0, "flags": 0 },
  "states": [
    {
      "state": "reject_all",
      "blocked": true,             // no non-essential tracker fired
      "trackersFired": [],         // tags/cookies/requests that still fired
      "cookiesSet": [],
      "thirdPartyHosts": []
    }
  ],
  "regulations": [
    {
      "regulation": "gdpr",
      "score": 100,
      "passed": true,
      "checks": [
        { "id": "cmp-present", "status": "pass", "severity": "high" },
        { "id": "pre-consent-tracking", "status": "pass", "severity": "critical" }
      ]
    }
  ],
  "flags": [ /* advisory observations */ ]
}
```

- `status` on a check is `pass`, `fail` or `skipped` (a regulation whose inputs
  were not simulated, e.g. no `gpc` state).
- A regulation's score is `pass / (pass + fail) × 100`; skipped checks are
  ignored. Regulations with no evaluated checks score 100.
- The overall score is the mean of the regulation scores, and `passed` is true
  only when **every** regulation passes its threshold.

### Checks by regulation

| Regulation | Check id | Fails when |
| --- | --- | --- |
| GDPR | `cmp-present` | no banner / consent platform is detected. |
| GDPR | `pre-consent-tracking` | any non-essential tracker fires in a blocking state. |
| CCPA | `do-not-sell-signal` | a sale/share tracker fires under the GPC signal. |
| CCPA | `opt-out-mechanism` | no "Do Not Sell / Share" link, banner reject or settings option is found. |
| CCPA | `opt-out-honored` | a sale/share tracker fires after `reject_all`. |
| LGPD | `subject-right:<id>` | a declared data-subject right is not reachable. |
| LGPD | `consent-honored` | a non-essential tracker fires after `reject_all`. |
| ePrivacy | `banner-present` | no cookie banner is shown. |
| ePrivacy | `banner-reject-option` | the banner offers no reject-all option. |
| ePrivacy | `pre-consent-cookies` | a non-essential cookie is set in a blocking state. |

### Flags

Flags are advisory and do not change a score:

- `dpia:<id>` — an observation matches a declared DPIA trigger.
- `cross-border-transfer` — a request or cookie reaches a host matching
  `cross_border_hosts`.

### Formatters

```go
_ = (privacy.JSONFormatter{}).Format(w, report) // indented JSON
_ = (privacy.TextFormatter{}).Format(w, report) // human-readable summary
```

## Architecture

```
Config (YAML/JSON)
      │
      ▼
  Evaluate ──► for each consent state ──► Rescanner.Rescan(state) ──► Observation
      │                                                                  │
      │                                              analyzeState → StateResult
      │                                                                  │
      │  for each regulation: evaluateRegulation ────────────────────────┘
      │      (GDPR / CCPA / LGPD / ePrivacy checks)
      ├─ detectFlags(...)
      └─ Report.finalize ── Formatter / Findings()
```

- `Rescanner` / `RescanFunc` / `FakeRescanner` (`simulate.go`) are the seam
  between the engine and a browser; an `Observation` mirrors the fields the
  checks need (banner, tags, cookies, requests, links).
- `Observation` can be adapted from a `domain.ScanResult` with
  `FromScanResult` (banner/GPC signals are set by the browser session).
- `Evaluate` (`engine.go`) drives the simulation, evaluates each regulation and
  detects flags; `report.go` holds the report types, scoring and formatters.
- `Report.Findings()` maps every failed check to a `domain.Finding` with
  `PluginID` `privacy`.

## Wiring

Like `tagpolicy` and `journey`, the engine is intentionally decoupled from run
execution. Remaining work to make privacy first-class:

1. Implement a browser-based `Rescanner` that drives the render worker with a
   consent-injection script (reusing the Module 4 action schema) and captures
   banner/GPC signals.
2. Persist reports and stream them over the existing SSE/WS endpoints; store
   findings via the standard plugin path.
3. API: `POST /api/v1/privacy/checks`.
4. Reporter plugins (`pluginsdk.Reporter`) for HTML/PDF export.

## Testing

`go test ./internal/privacy` covers YAML/JSON parsing and equivalence, strict
decoding, defaults, invalid regexes and the size cap; the `FakeRescanner`/
`RescanFunc`/`ScanResult` adapters; consent-state simulation and every
per-regulation check (including GPC, DPIA and cross-border flags), scoring,
regulation selection, and request/essential-host classification; the findings
mapping and both formatters; and the struct ↔ schema lockstep. See
`docs/test-plan.md`.
