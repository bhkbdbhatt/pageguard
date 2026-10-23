# Pageguard Plugin ABI

Plugins let you extend Pageguard without forking it. There are two authoring
models, both governed by the types in `pkg/pluginsdk` (Go) and `sdk-ts`
(TypeScript).

1. **Declarative** (`runtime: "builtin"`) — a list of `RuleSpec`s evaluated by
   the built-in rule engine. No code, no child process. Best for most rules.
2. **Executable** (`runtime: "node"`) — a Node/TypeScript process the host
   invokes once per observation over stdio JSON. Best for logic that needs code.

Both produce `pluginsdk.Finding`s and are subject to the same isolation: a
misbehaving plugin is skipped, never fatal to a scan.

## 1. Plugin kinds

| Kind | Purpose | Status |
| --- | --- | --- |
| `validator` | Inspect one observation, return findings | ✅ wired |
| `reporter` | Render a completed run (HTML/PDF/Jira) | interface defined (M3) |
| `integration` | Deliver alert events (Slack/PagerDuty) | interface defined (M5) |
| `detector` | Recognise additional tags/cookies | interface defined (M2) |

## 2. Manifest reference

A plugin is a directory containing `manifest.json`:

```json
{
  "id": "acme-tag-rules",
  "name": "Acme Tag Rules",
  "version": "0.1.0",
  "kind": "validator",
  "runtime": "builtin",
  "description": "Required tags for acme.com",
  "author": "Acme",
  "license": "MIT",
  "permissions": [],
  "rules": [ /* RuleSpec[] — declarative plugins only */ ],
  "configSchema": { /* optional JSON schema for per-plugin config */ }
}
```

For `runtime: "node"`, set `entry` (relative to the plugin directory), e.g.
`"entry": "index.mjs"`.

| Field | Required | Notes |
| --- | --- | --- |
| `id` | yes | Unique; defaults to the directory name if omitted |
| `name` | yes | Display name |
| `version` | yes | SemVer |
| `kind` | yes | See table above |
| `runtime` | yes | `builtin` or `node` |
| `entry` | for `node` | JS entrypoint, relative to the plugin dir |
| `rules` | for declarative | Array of `RuleSpec` |
| `permissions` | no | Advisory allowlist (`network`, `filesystem`, …) |
| `configSchema` | no | JSON schema for the plugin's config block |

The host loads every immediate subdirectory of the plugins dir
(`PAGEGUARD_PLUGINS_DIR`, default `./plugins`). A missing plugins dir is not an
error. A manifest with an unknown runtime/kind is skipped.

## 3. Declarative rule model

```jsonc
{
  "id": "acme.gtm-required",
  "title": "Google Tag Manager is not installed",
  "description": "…",
  "severity": "high",              // critical|high|medium|low|info
  "category": "martech",
  "remediation": "Add the GTM snippet.",
  "condition": {
    "all":  [ /* ALL must hold */ ],
    "any":  [ /* if non-empty, at least one must hold */ ],
    "none": [ /* NONE may hold */ ]
  }
}
```

Condition semantics (see `internal/plugin/ruleengine.go`): a rule fires when
`all` all hold, `any` (if present) has a match, and no `none` holds. An entirely
empty condition never fires.

### Predicate

```jsonc
{ "field": "statusCode", "op": "gte", "value": 400 }
{ "field": "tag", "name": "google-tag-manager", "op": "exists" }
{ "field": "cookieSecure", "name": "sid", "op": "eq", "value": false }
```

### Fields

| `field` | Type | `name` sub-selector |
| --- | --- | --- |
| `statusCode` | number | — |
| `htmlLength` | number | — (length of `html`) |
| `requestCount` | number | — |
| `url` | string | — |
| `finalUrl` | string | — |
| `title` | string | — |
| `html` | string | — |
| `contentType` | string | — |
| `rendered` | bool | — |
| `header` | string | header name (case-insensitive) |
| `tag` | presence | tag id or name (case-insensitive) |
| `cookie` | presence | cookie name |
| `cookieSecure` | bool | cookie name |
| `cookieHttpOnly` | bool | cookie name |
| `cookieSameSite` | string | cookie name |
| `requestHost` | presence | substring match against request hosts |

### Operators

| `op` | Applies to | Meaning |
| --- | --- | --- |
| `exists` | any | field/sub-selector is present |
| `not_exists` | any | field/sub-selector is absent |
| `eq`, `neq` | all | equality (type-aware) |
| `gt`, `gte`, `lt`, `lte` | number | numeric comparison |
| `contains`, `not_contains` | string | case-insensitive substring |
| `in` | string | value equals one of a list |

Numeric comparison on a string field is unsupported and evaluates false.

### Example

`plugins/example-tag-rules/manifest.json` demonstrates required-tag,
consent-tool and cookie-attribute rules. The built-ins
(`internal/plugin/builtins.go`) are: `core/http-error-status`,
`core/title-missing`, `core/https-required`, `core/meta-description-missing`,
`core/large-html-payload`.

## 4. Out-of-process protocol (v1)

The host runs `node <entry>` once per observation, writes a single JSON request
to stdin, closes it, and reads a single JSON response from stdout.

**Request**

```json
{
  "type": "validate",
  "protocolVersion": 1,
  "observation": { /* Observation */ },
  "scope": { "runId": "…", "projectId": "…", "scanId": "…", "config": {} },
  "rules": [ /* manifest rules, if any */ ]
}
```

**Response**

```json
{ "ok": true, "findings": [ /* Finding[] */ ] }
{ "ok": false, "error": "message", "findings": [] }
```

`protocolVersion` is checked by the host; a major mismatch rejects the plugin.
`ok: false`, a non-zero exit, or malformed stdout all surface as an error that
the host isolates (the plugin contributes no findings for that page).

### Observation shape

```jsonc
{
  "url": "https://acme.test/",
  "finalUrl": "https://acme.test/",
  "statusCode": 200,
  "contentType": "text/html",
  "title": "Acme",
  "headers": { "content-type": "text/html" },
  "html": "<!doctype html>…",
  "htmlTruncated": false,
  "requests": [{
    "method": "GET", "url": "https://…", "host": "…",
    "status": 200, "resourceType": "script", "initiator": "…",
    "requestHeaders": {}, "responseHeaders": {},
    "durationMs": 42, "sizeBytes": 1024, "thirdParty": true
  }],
  "cookies": [{
    "name": "sid", "domain": "acme.test", "path": "/",
    "valueHash": "9f2c…", "secure": true, "httpOnly": true,
    "sameSite": "Lax", "expiresUnix": 0, "session": true, "firstParty": true
  }],
  "tags": [{
    "id": "google-tag-manager", "vendor": "Google", "name": "Google Tag Manager",
    "category": "tag_manager", "evidence": "googletagmanager.com",
    "confidence": 0.9
  }],
  "consoleErrors": ["…"],
  "rendered": true,
  "depth": 0,
  "pageId": "…"
}
```

Raw cookie values are **never** sent to plugins — only `valueHash`.

### Finding shape

```json
{
  "ruleId": "acme.gtm-required",
  "severity": "high",
  "title": "Google Tag Manager is not installed",
  "description": "…",
  "evidence": { "any": "json" }
}
```

## 5. TypeScript SDK

```ts
import { serveValidator, find, finding } from "@pageguard/plugin-sdk";

serveValidator((obs) => {
  const findings = [];
  if (find.thirdPartyRequests(obs).length > 0) {
    findings.push(finding("acme.third-parties", "info", "Third-party requests", {
      evidence: { hosts: find.thirdPartyRequests(obs).map((r) => r.host) },
    }));
  }
  if (obs.cookies.some((c) => !c.secure)) {
    findings.push(finding("acme.insecure-cookie", "medium", "Insecure cookie"));
  }
  return findings;
});
```

`serveValidator` reads stdin, decodes the request, runs your function (sync or
async), and writes the response. Helpers: `find.thirdPartyRequests`,
`find.insecureRequests`, `find.insecureCookies`, `find.nonHttpOnlyCookies`,
`find.hasTag`, and `finding(ruleId, severity, title, extra?)`.

## 6. In-process Go plugins

Satisfy `pluginsdk.Validator` and register it during wiring:

```go
type myValidator struct{}

func (myValidator) ID() string { return "acme-go" }
func (myValidator) Validate(ctx context.Context, obs *pluginsdk.Observation, s pluginsdk.Scope) ([]pluginsdk.Finding, error) {
    if obs.StatusCode >= 500 {
        return []pluginsdk.Finding{{RuleID: "acme.5xx", Severity: "high", Title: "Server error"}}, nil
    }
    return nil, nil
}

reg.Register(myValidator{})
```

## 7. Versioning

- `protocolVersion` / `PROTOCOL_VERSION` is currently **1**. The host rejects a
  different major version.
- Additive fields to the observation are non-breaking; renames/removals require
  a major bump.

## 8. Security

- Plugins run as child processes of the host with **no ambient credentials**;
  the manifest `permissions` list is advisory (sandbox enforcement is a
  roadmap item).
- Only hashed cookie values are exposed.
- The observation HTML is already size-capped and truncation-flagged.
- A plugin crash or timeout only loses that plugin's findings for one page.
