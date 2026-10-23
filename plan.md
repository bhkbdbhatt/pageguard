# OpenTagValidator Dashboard Integration Plan

## Goal

Replace the current light Pageguard dashboard with the dark **OpenTagValidator**
UI from `opentagvalidator_dashboard.html`, backed by real Pageguard audit data.
Keep the existing scan/run engine, SSE streaming, sitemap, and evidence
functionality, and extend the API with rules, validation, consent, overview, and
report features.

Confirmed decisions:

- Replace the current Pageguard UI.
- Real data only; empty states when no audits exist.
- Rules persist in both memory and PostgreSQL.
- Reports export as JSON, CSV, and print-ready HTML (no binary PDF library).

## Feature mapping

| Mockup section | Implementation |
| --- | --- |
| Overview | KPI cards, 7-day health trend, vendor distribution, live findings feed, re-scan, governance export |
| Audit Runs & Crawler | Real audit form, live run metrics/terminal via SSE, audit history, CSV export |
| Beacon & DataLayer Inspector | Run/page selector, beacon list with vendor filters, decoded query params, DataLayer JSON, raw URL, validation result |
| Rules Configurator | Rule CRUD, enable/disable, search, create/edit modal, pass rates from latest run |
| Consent & CMP Matrix | Derived pre-consent snapshot vs post-consent expectation, leak detection, compliance score |
| Header/sidebar | Project selector, live in-flight count, Run Audit modal, active navigation |

## Assumptions and limitations

- The existing scanner captures pre-consent observations. Consent matrix is
  **derived from captured evidence** (tags, cookies, third-party requests), not
  from a second browser interaction. Labels state this clearly.
- "Download PDF" uses a print-optimized HTML report and the browser print dialog.
- Authentication/tenancy remains out of scope; `projectId` scopes all new data.
- No demo/seed metrics are shown; the UI provides empty states and an optional
  "Load starter rules" action.

## Backend changes

### 1. Domain model

Add to `internal/domain`:

- `AuditRule` with `ID`, `ProjectID`, `Name`, `Vendor`, `Enabled`,
  `URLPattern`, `Target` (`request|event|tag|cookie`), `Parameter`, `Event`,
  `Operator` (`exists|not_exists|equals|not_equals|matches|not_matches`),
  `Expected`, `Severity`, `Description`, timestamps.
- Extend `RunStats` with `Beacons`, `CriticalFindings`, `HealthScore`,
  `ConsentScore`, `VendorScores map[string]float64`.
- Extend `ScanConfig` with `ConsentSimulation` metadata.

### 2. Store contract

Add rule persistence methods to `domain.Store`:

- `CreateRule`, `GetRule`, `UpdateRule`, `DeleteRule`, `ListRules`.

Implement in `internal/store/memory` and `internal/store/postgres`, plus
`migrations/0003_audit_rules.sql` with an `audit_rules` table.

### 3. Rule engine (`internal/rules`)

- `request`: query-parameter matching on URL-scoped network requests.
- `event`: flatten `dataLayer` JSON and check dotted paths/event names.
- `tag`: match tag ID/name, optionally vendor-scoped.
- `cookie`: match cookie name.
- Vendor host patterns for Adobe, GA4/GTM, Meta, OneTrust when no regex is set.
- Produce per-rule pass/fail/skipped results with evidence.
- Convert failures into `domain.Finding` with `PluginID: "rules"`.

### 4. Run integration

- Load enabled rules at run start.
- Evaluate each fresh page, persist findings, emit `finding.created`.
- Recompute run stats in `finalize`: health, beacons, critical findings,
  consent score, per-vendor pass rates.

### 5. Governance services (`internal/governance`)

- Overview aggregation, 7-day trend, vendor distribution, violation feed.
- Audit history across project scans and CSV export.
- Validation aggregate per run/rule.
- Beacon extraction and consent matrix.
- Report builder for JSON, CSV, and print-ready HTML.

### 6. API additions

CORS middleware with `PAGEGUARD_CORS_ORIGINS` (default localhost:3000/3001).

New endpoints:

- `GET /api/v1/overview?projectId=`
- `GET /api/v1/audits?projectId=&limit=`
- `POST /api/v1/audits`
- `GET /api/v1/audits/export?projectId=`
- `GET /api/v1/runs/{id}/validation`
- `GET /api/v1/runs/{id}/beacons?url=`
- `GET /api/v1/runs/{id}/consent?url=`
- `GET /api/v1/runs/{id}/report?format=json|csv|html`
- `GET|POST /api/v1/rules`
- `PATCH|DELETE /api/v1/rules/{id}`

## Frontend changes

- Dark shell with sticky header, sidebar, project selector, in-flight count,
  Run Audit modal, inline SVG icons and charts.
- Extend `web/lib/api.ts` with all new endpoint wrappers and types.
- `/` Overview: KPIs, trend chart, vendor donut, filterable violation feed.
- `/audits`: crawl form, live SSE progress/terminal, audit history, exports.
- `/inspector`: run/page selection, beacon list, payload/DataLayer/raw tabs.
- `/rules`: CRUD table, toggles, search, create/edit modal, pass rates.
- `/consent`: pre/post consent matrix and leak summary.
- `/runs/[id]`: keep existing evidence/sitemap/controls, restyled dark.

## Metric definitions

- **Health score:** passed validated pages / validated pages; fallback severity
  penalty when no rules exist.
- **Captured beacons:** tracker-like network requests.
- **Critical anomalies:** critical findings in the latest project run.
- **CMP compliance:** `100 × (1 − pre-consent leaks / non-essential trackers)`.
- **Trend:** daily average health over seven days and vendor pass rates.

## Verification

- `gofmt -w` on changed Go files.
- `go build ./...`, `go test ./...`, `go vet ./...`.
- `cd web && npm run build` and `npx tsc --noEmit`.
- API smoke test: project → audit → overview/validation/beacons/consent/rules/report.

## Execution order

1. Create `plan.md`.
2. Domain types, store methods, memory/Postgres implementations, migration.
3. Rule engine and tests.
4. Orchestrator integration.
5. Governance services and tests.
6. API routes, CORS, tests.
7. Frontend shell, theme, API client, components.
8. Five dashboard sections and run-detail restyle.
9. Docs and full verification.
