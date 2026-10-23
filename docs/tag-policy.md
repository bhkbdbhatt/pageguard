# Tag Validation Policies

The tag-policy engine (`internal/tagpolicy`) checks a scanned page against a
user-authored **TagPolicy** and produces a **ValidationReport** with a pass/fail
result per rule. It is the policy-driven half of Module 2 (Tag & Cookie
Validation): the scanner produces evidence (`domain.ScanResult`), the policy says
what "correct" means, and the engine reports the difference.

The engine is a self-contained library. It depends only on `internal/domain` and
`gopkg.in/yaml.v3` (the YAML parser is confined to this package; JSON policies
need no extra dependency). It is not yet wired into the orchestrator or API —
wiring it into runs is a follow-up (see [Wiring](#wiring)).

## Quick start

```go
import "github.com/pageguard/pageguard/internal/tagpolicy"

policy, err := tagpolicy.LoadFile("examples/tagpolicy/acme.yaml")
if err != nil {
    log.Fatal(err)
}

// sr is the domain.ScanResult for one page (from the store or a checkpoint).
report := tagpolicy.Validate(sr, policy)
if !report.Passed {
    for _, f := range report.Findings() {
        log.Printf("%s: %s", f.Severity, f.Title)
    }
}
```

Policies may be authored in **YAML** (recommended) or **JSON**. Both encodings
are validated against `schemas/tagpolicy.schema.json`; the report is described by
`schemas/validationreport.schema.json`.

## The policy document

```yaml
version: 1
name: "Acme tag policy"
description: "Baseline martech governance for acme.com"

settings:
  unexpected_tags: info       # off | info | low | medium | high | critical
  duplicate_tags: medium
  unexpected_cookies: info
  max_occurrences: 1

must_exist:      [ ... ]      # expected tags / requests / events / cookies
must_not_exist:  [ ... ]      # forbidden tags / cookies
value_rules:     [ ... ]      # data-layer variable shape
consent_requirements: { ... } # CMP presence and gated categories
```

A complete, runnable example is `examples/tagpolicy/acme.yaml` (and the identical
`acme.json`).

### `settings`

Policy-wide hygiene checks derived from the whole policy.

| Field | Default | Meaning |
| --- | --- | --- |
| `unexpected_tags` | `info` | Severity for a detected tag **no** policy entry declares ("orphaned"). `off` disables the check. |
| `duplicate_tags` | `medium` | Severity when a tag id is detected more than `max_occurrences` times. |
| `unexpected_cookies` | `info` | Severity for a detected cookie absent from the policy inventory. |
| `max_occurrences` | `1` | Allowed detections of a single tag id before it is a duplicate. |

### `must_exist` / `must_not_exist`

Each entry is an *expectation*. `must_exist` passes when at least one item
matches; `must_not_exist` passes when nothing matches.

```yaml
- id: gtm                     # required, unique across the policy
  description: GTM container  # optional
  severity: high              # default medium
  target: tag                 # tag | request | cookie | event
  match:                      # all provided fields must match (logical AND)
    tag: google-tag-manager
  attributes:                 # cookie entries only
    secure: true
    httpOnly: true
    sameSite: [Lax, Strict]
    category: functional
```

`match` fields by target:

| Target | Fields |
| --- | --- |
| `tag` | `tag` (id), `tag_pattern` (regex on id and name), `name`, `name_pattern`, `category` |
| `request` | `url_pattern` (regex), `host` |
| `cookie` | `name`, `name_pattern`, `domain` (exact or suffix) |
| `event` | `event` (data-layer event name), or `name` |

A tag counts as *declared* (see `settings.unexpected_tags`) if it matches any
`must_exist` **or** `must_not_exist` entry with `target: tag`. Cookie attribute
assertions produce an additional `<id>-attributes` failure when a matched cookie
violates them.

### `value_rules`

Assert the shape of data-layer variables, addressed by a dotted path
(`cart.total`, `items[0].sku`).

```yaml
- id: cart-total
  severity: high
  variable: cart.total
  type: number                # string | number | integer | boolean | object | array | null
  min: 0
  max: 1000000
  required_if_event: purchase
- id: currency
  variable: cart.currency
  type: string
  pattern: "^[A-Z]{3}$"
  when_event: purchase
```

- Values are resolved against every captured data-layer entry.
- `when_event` narrows the entries the rule applies to; if the event never
  fired the rule is **skipped**.
- `required` means the variable must be present in at least one entry in scope.
- `required_if_event` requires the variable only after that event has fired.
- `enum` compares with numeric tolerance (JSON numbers are floats); `min`/`max`
  require a number; `pattern` requires a string.

### `consent_requirements`

```yaml
consent_requirements:
  require_cmp: true
  cmp_tags: [onetrust, cookiebot, usercentrics, trustarc]
  gated_categories: [advertising, social]
  gated_cookie_categories: [marketing]
  severity: critical
```

- `require_cmp` fails when no consent platform is detected. `cmp_tags` lists the
  tag ids that count; when empty, any tag in the `consent` category counts.
- If a CMP is **absent**, each `gated_categories` category present on the page
  fails, and each `gated_cookie_categories` category (as declared on the cookie
  inventory via `attributes.category`) fails.
- If a CMP is present, the gated checks do not run.

> **Module 3 scope.** Pre/post-consent behaviour (did a pixel actually fire
> *before* consent?) is not modelled yet because the scanner does not capture
> consent state. This engine checks the weaker, always-available property: a CMP
> must be present before gated categories are loaded.

## The ValidationReport

```json
{
  "schemaVersion": "1.0.0",
  "policyName": "Acme tag policy",
  "runId": "…", "pageId": "…",
  "url": "https://acme.com/checkout",
  "finalUrl": "https://acme.com/checkout",
  "generatedAt": "2026-01-01T00:00:00Z",
  "passed": false,
  "summary": { "total": 12, "passed": 9, "failed": 2, "skipped": 1,
               "bySeverity": { "high": 1, "critical": 1 } },
  "results": [
    {
      "id": "gtm",
      "section": "must_exist",
      "status": "fail",
      "severity": "high",
      "message": "expected tag(id=google-tag-manager) was not found",
      "expected": { "target": "tag", "match": { "tag": "google-tag-manager" } },
      "actual": null
    }
  ]
}
```

- `section` is one of `must_exist`, `must_not_exist`, `value_rules`,
  `duplicates`, `orphaned`, `cookies`, `consent`.
- `status` is `pass`, `fail` or `skipped`. A report passes when no rule fails.
- `Report.Findings()` maps each failing result to a `domain.Finding`
  (`pluginId: "tagpolicy"`, `ruleId: "tagpolicy.<id>"`), so a report can flow
  through the same persistence and SSE/WS path as plugin validators.

## Wiring

The engine is intentionally decoupled from run execution. Remaining Module 2
work to make it automatic:

1. Add a policy source to `ScanConfig` (e.g. `tagPolicyRef`) and load it in the
   orchestrator.
2. After each page is scanned (or on a stored `ScanResult`), call
   `tagpolicy.Validate` and persist `report.Findings()`.
3. Expose `GET /api/v1/runs/{id}/tag-policy-report` (and per-page variants).

## Testing

`go test ./internal/tagpolicy` covers YAML/JSON equivalence, strict decoding,
path resolution, every rule kind, duplicates/orphans, cookie attributes, consent
gating, the report summary and finding mapping, and the struct↔schema lockstep.
See `docs/test-plan.md`.
