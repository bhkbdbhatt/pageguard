# Pageguard — Feature Plan v2

Updated 2026-10-30 · status: proposed (implementation not started)

> Supersedes the *OpenTagValidator Dashboard Integration Plan* (original v1).
> v1 is **shipped**: dark dashboard, overview/audits/inspector/rules/consent,
> audit rules with persistence, validation/beacons/consent/report APIs, CSV
> export, inventory, cron schedules, alerting (channels/incidents/deliveries),
> journey runner, privacy jurisdiction evaluator, reports + report schedules —
> 60+ REST routes, both stores complete, zero TODOs in code.

## Goal

Harden the platform for sharing and unattended operation, close the quality
gaps in alerting/reporting, and surface the evidence and reports the engines
already produce but the dashboard doesn't show. Everything stays
zero-config-compatible: an empty configuration behaves exactly like today.

## Baseline gaps (why these features)

| # | Gap | Evidence |
| --- | --- | --- |
| G1 | No authentication/tenancy; any caller can hit every route | no auth middleware in `internal/api`; empty `projectId` falls back to first project (`internal/service/features_shared.go:13`) |
| G2 | Alert delivery: single attempt, no retry/backoff/dedup window | `internal/service/features_alerts.go:358-372` |
| G3 | Email channels + report recipients stored but never sent | `features_alerts.go:393`, `features_reports.go` (`recipients` unused) |
| G4 | Multi-replica: 30 s ticker would double-fire schedules | `service.Tick` (`features_shared.go:78-91`), no advisory locks |
| G5 | Rule targets: UI offers 10, backend validates 4 → 400s | `web/app/rules/page.tsx:30` vs `internal/service/governance_service.go:249-253` |
| G6 | Run-scoped tag/cookie/request endpoints missing (roadmap Module 2 §5) | data only reachable via `/runs/{id}/results` |
| G7 | Journey runs poll; steps/screenshots/expect diffs never visible | `features_journeys.go:123-193`; `web/app/journeys/page.tsx:340-366` |
| G8 | Privacy batches evaluate one URL only | `features_privacy.go:189-207` (`TotalPages: 1`) |
| G9 | HTML reports are JSON pretty-printed in `<pre>` | `features_reports.go:504-519` |
| G10 | Run detail page light-themed; evidence panel has no DataLayer | `web/app/runs/[id]/page.tsx:451,476` |
| G11 | Dead scaffolding: `alert_events` table + `domain.AlertEvent` unused | `migrations/0001_init.sql`, `internal/domain/domain.go:204` |
| G12 | `docs/roadmap.md` statuses stale (scheduling marked ⬜) | vs `internal/cron` + `StartBackground`, both wired |

## Workstreams

Backend B1–B10, frontend F1–F9. P0 = must, P1 = should, P2 = nice.

| ID | Feature | Priority |
| --- | --- | --- |
| B1 | Authentication & tenancy (API keys) | P0 |
| B2 | Alert delivery retry/backoff + dedup | P1 |
| B3 | Email delivery (SMTP) | P1 |
| B4 | Postgres advisory locks for schedulers | P1 |
| B5 | Rule-target parity | P0 |
| B6 | Run inventory endpoints (tags/cookies/requests) | P1 |
| B7 | Journey run streaming + step reports | P1 |
| B8 | Multi-page privacy batches | P1 |
| B9 | First-class HTML reports | P2 |
| B10 | Scaffolding & docs cleanup | P2 |
| F1 | Run detail: dark restyle + DataLayer evidence | P0 |
| F2 | Rules page target parity | P0 |
| F3 | Alerts page: conditions + delivery history | P1 |
| F4 | Journeys: step report viewer + definition editor | P1 |
| F5 | Privacy: multi-page batches + score gauges | P1 |
| F6 | Inventory: run selector | P1 |
| F7 | Settings page (API keys) | P1 |
| F8 | Chart components used across pages | P2 |
| F9 | Demo-mode toggle (`NEXT_PUBLIC_DEMO=1`) | P2 |

Out of scope: PDF export (print CSS in B9 instead), Redis/NATS event bus,
RBAC beyond `admin|reader|writer`, journey assertion authoring beyond the
existing `expect` engine.

---

## Backend changes

### B1. Authentication & tenancy (API keys)

- New `domain.APIKey`: `ID, ProjectID, Name, Prefix, KeyHash (sha256 of
  `pg_…` secret, plaintext never stored), Roles []string {admin,reader,writer},
  Enabled, ExpiresAt, LastUsedAt, CreatedAt`.
- Store methods `CreateAPIKey/GetAPIKey/UpdateAPIKey/DeleteAPIKey/ListAPIKeys`
  in both stores + migration `0005_api_keys.sql` (`pageguard_api_keys`).
- `internal/api/auth.go` middleware, applied to `/api/v1/*` (skip `/healthz`):
  parse `Authorization: Bearer pg_*`, hash-compare, set project scope in
  request context, bump `LastUsedAt` (batched). Writers/required roles:
  mutating verbs demand `writer`/`admin`; readers are read-only.
- `PAGEGUARD_AUTH=keys` enables enforcement; default `none` keeps the API
  open (backward compatible). Keys are created via CLI:
  `pageguard --issue-key --project <id> --name <name> --roles admin` prints
  the secret once.
- `features_shared.resolveProject` becomes an error when a scoped non-admin
  key omits `projectId` (no silent first-project fallback).
- Limits: max 32 keys/project; key disabled → 403.

### B2. Alert delivery retries & dedup

- Wrap delivery in `internal/service/features_alerts.go`: up to 3 attempts,
  backoff 5 s → 30 s → 2 min; every attempt persists an `alert_deliveries`
  row (status `pending|retrying|delivered|failed`, `attempt`, error text).
- Delivery moves out of the run-finalize goroutine: incidents start
  `pending`, `next_attempt_at` scheduled; `service.Tick` processes due
  deliveries (also closes the retry loop on API restarts).
- Dedup window: `alert_rules.min_interval_minutes` (default 30). Same
  (ruleId, url, severity-ish fingerprint) inside the window updates the
  existing incident (`trigger_count++`) instead of creating a new one.
- `GET /api/v1/alert-incidents/?status=&limit=` gains filters and paging.

### B3. Email delivery (SMTP)

- New `internal/mailer` package behind a `Mailer` interface: SMTP
  implementation (no external deps) configured by
  `PAGEGUARD_SMTP_HOST/PORT/USER/PASS/FROM/STARTTLS`; absent → feature
  reports the existing "not configured" error verbatim.
- Alert channels `kind=email` deliver through the mailer (B2 pipeline).
- Report schedules email their artifact link/content to `recipients` on
  generation; failure recorded in the report row (`email_status`).

### B4. Scheduler advisory locks (multi-replica safe)

- Add `TryAdvisoryLock(ctx, key string) (acquired bool, unlock func(),
  err error)` to `domain.Store`; Postgres: `pg_try_advisory_lock`; memory:
  always true.
- `service.Tick` acquires `pageguard:tick` before processing schedules,
  report schedules and privacy batches; skips the tick when not acquired.
  Helm chart may then move above 1 replica.

### B5. Rule-target parity

- `internal/rules` adds matchers: `beacon` (network request filtered through
  `martech.IsTrackerRequest`, vendor-aware), `network` (any request, incl.
  document), `page` (URL/pattern assertions only). `tag` matches script
  URL/global context, `event` keeps dotted datapath semantics.
- `governance_service.validateRule` target set becomes
  `request|event|tag|cookie|beacon|network|page` + aliases
  `datalayer→event`, `script→tag`; `consent/any` are rejected with a
  message listing valid targets.
- Starter rules gain one network example (e.g. no Adobe toolbar beacon on
  non-cart pages).

### B6. Run inventory endpoints

- `GET /api/v1/runs/{id}/tags` (`vendor`, `purpose`, `thirdParty`),
  `…/cookies` (`party`, `purpose`), `…/requests` (`host`, `method`,
  `resourceType`, `thirdParty`) with paging.
- Implementation: bucket the run's `ScanResult`s (same source as
  `features_inventory.go:14-162`), no new tables; `/api/v1/inventory?runId=`
  reuses it for the latest-run case.

### B7. Journey run streaming & step artifacts

- Journey engine emits onto the events bus: `journey.step.started`,
  `journey.step.finished` (with step result + expect diffs),
  `journey.finished` → persistent SSE `GET /api/v1/journey-runs/{id}/events`
  (reuse the `run.snapshot`/heartbeat handling in `internal/api/stream.go`).
- Screenshots: `journey_runs` gains `screenshot_uri`; engine uploads the
  failure PNG (data URI for memory store, file/URI path for Postgres);
  `GET /api/v1/journey-runs/{id}/screenshot?step=` returns image bytes.

### B8. Multi-page privacy batches

- `PrivacyBatch` input accepts `urls[]` or `scanId` (derive page set from the
  run's successful pages, cap 50); `TotalPages/CompletedPages` become real;
  evaluations run sequentially with per-row resume on restart.
- Existing single-URL API shape stays valid (`urls` defaults to `[startUrl]`).

### B9. First-class HTML reports

- `internal/reports/render.go`: one styled, dependency-free HTML template per
  report type (governance/consent/inventory/journey/privacy) — cards, tables,
  print CSS (`@media print`), no JS. JSON/CSV formats unchanged; artifact
  content-type correct; `?download` stream disposition unchanged.

### B10. Cleanup

- Drop unused `alert_events`-related scaffolding: remove `domain.AlertEvent`
  and note the table as deprecated (no destructive migration).
- Remove previously deferred `Integration` plugin stub commentary; keep the
  validator-only plugin registry (B3 handles email in-service).
- Update `docs/roadmap.md` statuses (scheduling ✅, alerting 🚧→B2/B3,
  auth ⬜→B1) and `docs/ui.md` for the new pages.

---

## Frontend changes

### F1. Run detail: dark restyle + evidence DataLayer

- Restyle `web/app/runs/[id]/page.tsx` to the dashboard dark tokens (today
  light backgrounds at `:451,476`).
- EvidencePanel gains a **DataLayer** section — lift the Inspector's
  decoded/magic (`layerBase`, `pretty(decodeEntryValue)`) rendering from
  `web/app/inspector/page.tsx` into `web/components/DataLayerList.tsx` and
  use it in both places, plus an "Open in Inspector" deep link.

### F2. Rules target parity

- `web/app/rules/page.tsx:30` target list trimmed/aligned to B5 exactly;
  unsupported selections can no longer produce 400s.

### F3. Alerts UI depth

- Rule builder adds the `ruleIds` match condition (backend-supported,
  `features_alerts.go:486-504`) and min-interval setting (B2).
- Email option stays but is marked "requires SMTP" until configured.
- Incident drawer lists delivery attempts with per-attempt status/timestamp
  (B2 rows), plus retry-now button.

### F4. Journeys step reports

- Run drawer shows the step timeline streaming live via the new SSE feed
  (step status, expect diffs, duration), historical runs render the stored
  step report; failure screenshots lazy-load per step (B7 endpoint).
- Create form gains a full-definition JSON editor tab (engine supports a
  richer schema than the 4 legacy actions the UI builds today) and the
  `type` step action.

### F5. Privacy batches & gauges

- New evaluation modal supports pasting a URL list or picking a prior run's
  pages (B8) with progress (`completed/total`) while the batch runs.
- Regulation scores use the unused `RadialGauge` chart; consent-coverage
  heatmap (`Heatmap` component) on the matrix.

### F6. Inventory run selector

- Run dropdown in `web/app/inventory/page.tsx` (API + client already take
  `runId`, `web/lib/api.ts:904`); defaults to latest run.

### F7. Settings page (API keys)

- New `web/app/settings/page.tsx`: create keys (name, project, roles,
  expiry), show-copy-once secret, enable/disable/delete, last-used column.
  Requires `Authorization` plumbing in `web/lib/api.ts` (401 → key prompt).

### F8. Charts inoculation

- Use the idle chart components: per-vendor `HorizontalBars` on Inventory,
  `StackedBar` for request third/first-party mix on Overview.

### F9. Demo-mode toggle

- Wire the dormant `web/lib/demo.ts` via `NEXT_PUBLIC_DEMO=1`: `request()`
  in `web/lib/api.ts` short-circuits to demo fixtures keyed by path (+
  `DemoBadge` in AppShell). For demos/design review only; default off.

---

## New/changed API surface

| Route | Purpose | Workstream |
| --- | --- | --- |
| `GET/POST /api/v1/api-keys`, `PATCH/DELETE /api/v1/api-keys/{id}` | Key management for `admin` keys | B1 |
| `GET /api/v1/runs/{id}/tags\|cookies\|requests` | Run-scoped inventory | B6 |
| `GET /api/v1/journey-runs/{id}/events` | Journey SSE | B7 |
| `GET /api/v1/journey-runs/{id}/screenshot?step=` | Step screenshots | B7 |
| `PATCH /api/v1/alert-incidents/{id}/retry` | Requeue delivery | B2 |
| Filters: `GET /api/v1/alert-incidents/?status=&limit=` | Triage queue paging | B2 |
| `AlertRule.min_interval_minutes`, `PrivacyBatch.urls[]\|scanId` | Model fields | B2/B8 |
| Auth is applied to **all** existing routes | tenancy | B1 |

Environment additions: `PAGEGUARD_AUTH`, `PAGEGUARD_SMTP_*`,
`NEXT_PUBLIC_DEMO`. Migration: `0005_api_keys.sql` (+ `min_interval_minutes`
column, `journey_runs.screenshot_uri`).

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` (auth middleware, retry
  backoff with injected clock, advisory-lock acquire/skip, target parity,
  inventory endpoints, HTML render golden files).
- `npx tsc --noEmit`; `npm run build` in `web/`.
- API smoke: run an audit with `PAGEGUARD_AUTH=keys` (401 unauth, 200 with
  key, scoped project isolation), force an alert (3 delivery attempts in
  table), generate an HTML report and open in browser.

## Execution order

1. B5 + F2 (parity quick-win), B1 auth backend, F7 settings page.
2. B2 + B3 + B4 (delivery pipeline), F3.
3. B6 endpoints, F6.
4. B7 + F4 (journey streaming/report viewer).
5. B8 + F5 (privacy batches).
6. B9 reports; B10 cleanup + docs refresh.
7. F1 (run detail polish), F8, F9 last.
