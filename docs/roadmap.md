# Pageguard Roadmap

The foundation and **Module 1 (Page Scanning)** are implemented and tested.
This document specifies the remaining modules and the cross-cutting work. Each
module already has its database tables in `migrations/0001_init.sql` and its
plugin interface stubbed in `pkg/pluginsdk`, so delivery is primarily
implementation.

Status legend: ✅ done · 🚧 specified (below) · ⬜ not started.

---

## Cross-cutting enablers

| Item | Status | Notes |
| --- | --- | --- |
| **PostgreSQL store adapter** | ✅ done | `internal/store/postgres` (pgx) implements `domain.Store` against `migrations/0001_init.sql`; selected via `DATABASE_URL`; integration tests gated on `PAGEGUARD_TEST_DATABASE_URL`. |
| Shared event bus | ⬜ | The in-memory hub binds runs to one node. Add a Redis/NATS-backed `events.Bus` so any replica can stream any run. Until then, the Helm chart runs 1 replica. |
| Authentication & tenancy | ⬜ | API keys per project; middleware sets `projectId`; every store query is already project-scoped. |
| Scheduling | ⬜ | A cron evaluator in `service` that starts runs for scans with a schedule; shared-lock via Postgres advisory locks. |

---

## Module 2 — Tag & Cookie Validation 🚧

**Objective.** Prove that each page loads exactly the tags (and sets exactly the
cookies) it is supposed to — no more, no less.

**Data already captured.** `page_requests`, `page_cookies`, `page_tags` (Module
1 populates the document request plus detected tags; the browser renderer
already supplies the full request/cookie set).

> **Delivered: policy-based validation engine** (`internal/tagpolicy`). A
> user-authored YAML/JSON `TagPolicy` declares `must_exist`, `must_not_exist`,
> `value_rules` and `consent_requirements`; `tagpolicy.Validate(scanResult,
> policy)` returns a `ValidationReport` with a pass/fail result per rule
> (including duplicate/orphaned-tag and cookie-inventory hygiene). Schemas:
> `schemas/tagpolicy.schema.json`, `schemas/validationreport.schema.json`;
> examples: `examples/tagpolicy/`; docs: `docs/tag-policy.md`. Wiring it into
> runs (below) is still open.

**New work**

1. **Detector plugins.** Add `pluginsdk.Detector` implementations:
   ```go
   type Detector interface {
       ID() string
       Detect(obs *Observation) ([]Tag, error)
   }
   ```
   Ship a larger signature catalogue (GTM, GA4, Meta Pixel, TikTok, LinkedIn,
   Hotjar, OneTrust, etc.) keyed by script URL/global/known cookie name.
2. **Inventory model.** Add to `ScanConfig`:
   - `requiredTags[]`, `forbiddenTags[]`
   - `requiredCookies[]`, `forbiddenCookies[]`
   - `allowedThirdPartyHosts[]`
3. **Cookie classification.** First/third party, session/persistent, secure
   flags, and a purpose label (`analytics|ads|functional|consent`) from the
   detector catalogue.
4. **Rules.** Declarative rules already support `tag`, `cookie`,
   `cookieSecure`, `cookieHttpOnly`, `cookieSameSite`, `requestHost` — build the
   inventory check as rules, plus a detector-driven "unexpected tag" rule. The
   `tagpolicy` engine (delivered) already expresses tag/request/event
   expectations, cookie attribute assertions, data-layer value rules, and
   unexpected/duplicate hygiene from a single policy document; wire it into the
   run flow and treat `ScanConfig` inventory fields as a policy source.
5. **API.** `GET /api/v1/runs/{id}/tags`, `.../cookies`, `.../requests`
   (filters: host, purpose, third-party).

**Persistence.** Fill the `page_tags`/`page_cookies`/`page_requests` tables
during `finalize`.

**Tests.** Detector signatures against fixture HTML/JS; inventory pass/fail;
cookie classification; a browser-render fixture with a third-party pixel.

**Acceptance.** A scan of a fixture page reports detected tags, flags a missing
required tag, and flags an unexpected third-party host.

---

## Module 3 — Privacy Compliance 🚧

**Objective.** Verify consent behaviour: no non-essential tags or cookies may
fire before consent, and generate an auditable privacy report.

> **Delivered: compliance engine** (`internal/privacy`). A user-authored
> YAML/JSON `Config` selects the regulations to evaluate (`gdpr`/`ccpa`/`lgpd`/
> `eprivacy`), classifies cookies and third-party hosts, and declares CMP tags,
> subject rights, DPIA triggers, cross-border hosts and opt-out patterns.
> `privacy.Evaluate(ctx, cfg, rescanner)` re-scans the page under each consent
> state (`accept_all`/`reject_all`/`necessary_only`/`gpc`), evaluates per-
> regulation checks, scores each 0–100, and emits advisory flags. `Rescanner`
> is the browser seam (`FakeRescanner` in tests); `Report.Findings()` maps
> violations onto `domain.Finding`. Schemas: `schemas/privacy.schema.json`,
> `schemas/privacyreport.schema.json`; examples: `examples/privacy/`; CLI:
> `cmd/privacy`; docs: `docs/privacy.md`. Wiring the browser-based rescanner and
> reporter plugins (below) is still open.

**New work**

1. **Consent capture.** Extend `Observation` with a `ConsentState` snapshot:
   detected CMP (OneTrust, Cookiebot, TrustArc), banner presence, and the
   consent categories granted at observation time.
2. **Pre/post-consent comparison.** In `RenderBrowser`, capture requests before
   interacting with the banner and after accepting/declining; diff them.
   Reuse the browser worker with an interaction script (see Module 4's action
   schema).
3. **Reporter plugins.** Implement `pluginsdk.Reporter` for HTML and PDF export
   of cookie/tag inventories and consent violations.
4. **Jurisdiction profiles.** Scan-level config (`gdpr`, `ccpa`, custom) selects
   which rule packs apply.

**Rules.** `privacy.pre-consent-marketing-tag`,
`privacy.cmp-missing`, `privacy.consent-not-honored`,
`privacy.cookie-without-consent`.

**Tests.** Fixture page with a fake CMP that leaks a pixel pre-consent →
expects one finding; a compliant fixture → zero findings.

**Acceptance.** A one-click report lists every cookie/tag by purpose and cites
pre-consent violations with evidence.

---

## Module 4 — Journey Testing 🚧

**Objective.** Execute scripted multi-step user flows in a real browser and
validate behaviour (tags, requests, errors) at each step.

> **Delivered: journey engine** (`internal/journey`). A YAML/JSON `Journey` of
> `navigate`/`click`/`fill`/`wait` steps runs sequentially in one browser
> session; after each step the engine detects tags, checks the step's
> `expect`ations (url/title/status, tags, data-layer events, cookies) and records
> the per-step diff of new requests/console/cookies/tags/events, capturing a
> screenshot on failure. `journey.NewEngine(...).Run(...)` returns a
> step-by-step `JourneyReport`; `PlaywrightSessionFactory` drives the browser
> worker (extended with `open`/`step`/`screenshot`/`close`), and `FakeSession`
> keeps tests offline. Schemas: `schemas/journey.schema.json`,
> `schemas/journeyreport.schema.json`; examples: `examples/journey/`; CLI:
> `cmd/journey`; docs: `docs/journey.md`. Persisting/streaming runs (below) is
> still open.

**Schema.** `journeys` exists; add `journey_steps` and `journey_runs`
(migration `0002`).

**Step schema** (shared with the browser worker):

```jsonc
{ "action": "goto",   "url": "https://acme.test/login" }
{ "action": "fill",   "selector": "#email", "value": "a@b.c" }
{ "action": "click",  "selector": "button[type=submit]" }
{ "action": "wait",   "selector": ".dashboard", "timeoutMs": 10000 }
{ "action": "assert", "selector": "h1", "textContains": "Welcome" }
```

**New work**

1. Extend `tools/browser-renderer` to accept a `steps[]` script and emit a
   per-step observation stream.
2. `internal/journey` package: `Runner` that drives the worker, persists step
   results, and runs validators after each step.
3. API: `POST /api/v1/journeys`, `POST /api/v1/journeys/{id}/runs`, streamed via
   the same SSE/WS endpoints (runs are unified under `scan_runs`).

**Tests.** Worker-protocol contract test; a fixture app with a login flow;
assertion failure produces a finding; step timeout is isolated.

**Acceptance.** A journey run streams step-by-step results and fails loudly on a
broken flow.

---

## Module 5 — Alerting 🚧

**Objective.** Notify humans when a run crosses a threshold, and prove the
notification was delivered.

**Schema.** `alert_rules`, `alert_events` exist.

**Rule model**

```jsonc
{
  "name": "Critical findings on checkout",
  "match": { "scanId": "…", "urlContains": "/checkout" },
  "when":  { "severityAtLeast": "high", "countGte": 1 },
  "channels": [ { "type": "slack", "target": "#alerts" } ]
}
```

**New work**

1. **Evaluator.** After `finalize`, evaluate alert rules against the run's
   findings/severity counts; persist `alert_events`.
2. **Integration plugins.** Implement `pluginsdk.Integration`:
   `integration.webhook`, `integration.slack`, `integration.email`. Delivery is
   retried with backoff; every attempt is recorded.
3. **Scheduler integration.** Combine with the cross-cutting scheduler for
   "nightly audit + alert on regression".
4. **API.** CRUD `/api/v1/alert-rules`; `GET /api/v1/alert-events`.

**Tests.** Threshold matching table; a fake webhook receiver asserts payload and
retry/backoff; duplicate suppression within a window.

**Acceptance.** A run that introduces a high finding fires exactly one alert to
a test webhook.

---

## Suggested delivery order

1. Cross-cutting: ~~Postgres adapter~~ (✅ done) → shared event bus → auth.
2. Module 2 (tag/cookie) — extends Module 1 directly, highest immediate value.
3. Module 3 (privacy) — builds on Module 2's inventory.
4. Module 5 (alerting) — small, independent, unlocks scheduled audits.
5. Module 4 (journeys) — largest; needs the browser interaction schema.

Each module ships with: a design doc under `docs/modules/`,
the implementation, its plugin interface, and a test plan section — matching the
Module 1 template.
