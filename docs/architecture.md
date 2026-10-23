# Pageguard — System Architecture

## 1. Goals & constraints

Pageguard is a self-hostable, MIT-licensed web governance and martech auditing
platform. It must:

- Crawl sites (HTTP) and execute client-side tags (headless browser).
- Detect tags, pixels, cookies and third-party requests.
- Validate behaviour against pluggable rules and surface findings.
- Stream live progress to clients.
- Extend via a TypeScript plugin SDK without forking the core.

Hard constraints: Go (or Rust) orchestrator, PostgreSQL storage, Next.js +
Tailwind frontend, REST + WebSocket API, Docker Compose and Helm deployment.

**Decision:** Go for the backend. Rationale: ergonomic concurrency for fan-out
crawling, a mature `net/http` server, single-binary deployment, and a large
Kubernetes ecosystem. The core deliberately uses **only the standard library**
so it compiles and tests without network access; external capability lives
behind interfaces. The single exception is the PostgreSQL store, which uses
`jackc/pgx` and is confined to `internal/store/postgres`.

## 2. Component overview

```
                ┌──────────────────────────────────────────────┐
   REST/WS      │                 API layer                     │
 ┌────────────► │  api.Server  ·  SSE  ·  ws.Conn               │
 │              └───────────────┬──────────────────────────────┘
 │                              │
 │                     ┌────────▼────────┐
 │                     │    Service       │  start/cancel runs, CRUD
 │                     └────────┬────────┘
 │                              │
 │                    ┌─────────▼──────────┐
 │   progress events  │     Orchestrator    │  Manager.Execute
 │  ◄───── Hub ───────┤  frontier · workers │
 │                    └───┬────────────┬────┘
 │                        │            │
 │              ┌─────────▼───┐   ┌────▼────────┐
 │              │  Scanner     │   │  Plugin Host │
 │              │ fetch/render │   │  validators  │
 │              └──────┬───────┘   └────┬────────┘
 │                     │                │
 │          ┌──────────▼─────┐   ┌──────▼─────────┐
 │          │ HTTP / Browser │   │ rule engine +  │
 │          │ (Playwright)   │   │ Node plugins   │
 │          └────────────────┘   └────────────────┘
 │
 │              ┌──────────────────────────────┐
 └──────────────┤  domain.Store (memory | PG)   │
                └──────────────────────────────┘
```

### Layers and dependency direction

`domain` has no imports from the project. `scanner`, `plugin`, `events` depend
on `domain`. `orchestrator` depends on all three. `service` depends on
`orchestrator`. `api` depends on `service` + `plugin`. `cmd` wires everything.

This strict direction keeps the domain pure and makes each layer unit-testable.

## 3. Data flow — a scan run

1. **Client** `POST /api/v1/scans/{id}/runs`. `service.StartRun` creates a
   `queued` run, publishes `run.queued`, and launches `Manager.Execute` in a
   goroutine with a cancellable context.
2. **Manager** seeds the frontier from `scan.StartURLs`, deduplicating on a
   canonical URL (fragment stripped, host lowercased).
3. For each BFS depth, the manager fans the URL batch out to a bounded worker
   pool (`Config.Concurrency` workers).
4. Each worker calls `Scanner.ScanPage`, which:
   - fetches via `Fetcher` (HTTP), and
   - optionally renders via `Renderer` (browser) for `browser`/`auto` modes.
   It returns an immutable `domain.Observation` plus extracted links.
5. The manager persists the `Page`, runs each applicable `Validator`, persists
   `Finding`s, and enqueues in-scope discovered links for the next depth. It
   also writes a per-page `domain.ScanResult` checkpoint keyed by `(run, url)`.
6. Every step emits a typed `domain.Event` on the `events.Hub`; the API relays
   them to subscribed SSE/WebSocket clients.
7. `finalize` reconciles page/finding totals from storage, sets terminal status,
   and emits `run.completed` / `run.failed`.

Scope control at step 5 uses: same-origin policy, include/exclude globs, and
`robots.txt` (user-agent aware, longest-match Allow/Disallow, wildcards, `$`).

### Per-page checkpoints, resume and geolocation

Every scanned page is checkpointed as a `domain.ScanResult` keyed by
`(runId, url)` (see `docs/modules/02-scan-orchestration.md` and
`schemas/scanresult.schema.json`). `Manager.Execute` is therefore **idempotent
and resumable**: a URL already present in the store is loaded from its
checkpoint (links and all) instead of being re-fetched, and `finalize`
reconciles page/finding totals from persistence so a resumed run reports the
same figures as a fresh one. `POST /api/v1/runs/{id}/resume` re-launches a
finished or failed run this way.

When `Config.Geolocations` is set, a round-robin picker assigns a region to
each page; the region drives the browser context (coordinates, locale, timezone)
and selects a matching egress proxy from the `proxy.Rotator` when one is
configured. Each `ScanResult` records the `geolocation` and (credential-free)
`proxyHost`, giving cross-border data-collection an auditable vantage point.

Scope control at step 5 uses: same-origin policy, include/exclude globs, and
`robots.txt` (user-agent aware, longest-match Allow/Disallow, wildcards, `$`).

## 4. Concurrency model

- **One goroutine per run** (manager). Runs are independent.
- **Bounded worker pool per depth** for page scans. Workers consume an index
  channel; results are written to a pre-sized slice by index, so no result
  locking is needed.
- **Level-synchronized BFS**: a depth fully completes before the next begins.
  This trades a little latency for deterministic accounting and deadlock-free
  scheduling (no dynamic channel-close races). Per-level concurrency is
  preserved.
- **Event hub**: fan-out under a short lock, delivery non-blocking with a
  per-subscriber buffer (dropped events are counted). A slow client can never
  stall a scan.

## 5. Storage

The `domain.Store` interface is the persistence contract. Two implementations:

- `internal/store/memory` — maps + mutexes; used by tests and local runs.
- `internal/store/postgres` — the production backend, implemented on
  `jackc/pgx` (the project's only third-party dependency), against the schema
  in `migrations/0001_init.sql`. It is selected when `DATABASE_URL` is set and
  applies migrations on startup.

Domain IDs are 32-char hex strings; PostgreSQL stores them as `uuid` (the
dashless form is accepted on write and normalized back on read). Open-ended
structures (scan config, run stats, evidence, event payloads) round-trip as
`jsonb`.

### Schema summary

| Table | Purpose |
| --- | --- |
| `projects` | Tenant boundary |
| `scans` | Reusable audit definitions (config as `jsonb`) |
| `scan_runs` | Executions with status + stats |
| `run_pages` | Per-URL results (unique per run+url) |
| `scan_results` | Per-page checkpoints (evidence `jsonb`, unique per run+url) |
| `page_requests` | Observed network exchanges (tag/cookie evidence) |
| `page_cookies` | Observed cookies (value **hash** only) |
| `page_tags` | Detected martech tags |
| `findings` | Rule violations linked to run + page |
| `run_events` | Persisted progress log (replay/resume) |
| `journeys` | Module 4 scripted flows |
| `alert_rules`, `alert_events` | Module 5 alerting |
| `plugin_registrations` | Installed plugin manifests |

Indexes cover the hot paths: findings by run+severity, requests by run+host,
pages by run, and active runs.

## 6. API surface

| Method | Path | Description |
| --- | --- | --- |
| GET | `/healthz` | Liveness/readiness |
| GET/POST | `/api/v1/projects` | List / create projects |
| GET/POST | `/api/v1/scans` | List / create scans |
| GET | `/api/v1/scans/{id}` | Fetch a scan |
| GET | `/api/v1/scans/{id}/runs` | Run history for a scan (newest first) |
| POST | `/api/v1/scans/{id}/runs` | Start a run (202 Accepted) |
| GET | `/api/v1/runs/{id}` | Run status + stats |
| POST | `/api/v1/runs/{id}/cancel` | Cancel an in-flight run |
| GET | `/api/v1/runs/{id}/pages` | Page results |
| GET | `/api/v1/runs/{id}/findings` | Findings (filter by severity/rule/plugin) |
| POST | `/api/v1/runs/{id}/resume` | Re-launch a finished/failed run from checkpoints |
| GET | `/api/v1/runs/{id}/results` | Per-page `ScanResult` checkpoints |
| GET | `/api/v1/runs/{id}/result?url=` | One page's `ScanResult` |
| GET | `/api/v1/runs/{id}/events` | **SSE** progress stream |
| GET | `/api/v1/runs/{id}/stream` | **WebSocket** progress stream |
| GET | `/api/v1/plugins` | Registered validators |

SSE is the default transport (no handshake, proxy-friendly, auto-reconnect);
WebSocket is offered for lower-latency clients. Both share the same event hub
and payloads.

## 7. Plugin architecture

Plugins are either **declarative** (a manifest of rules evaluated by the
built-in engine) or **executable** (a Node/TypeScript process speaking a
one-shot JSON protocol over stdio). The public contract lives in
`pkg/pluginsdk` (Go) and `sdk-ts` (TypeScript). See `docs/plugin-abi.md`.

The host registers validators in a `Registry`; `Host.Validate` converts the
internal `domain.Observation` into the public SDK shape, runs every applicable
validator, and converts results back into `domain.Finding`. A validator error
is isolated and does not affect other validators or the run.

## 8. Deployment

- **Single node:** `deploy/docker-compose.yml` (API, dashboard, Postgres; the
  DB auto-applies `migrations/`).
- **Kubernetes:** `deploy/helm/pageguard` (Deployment, Service, ConfigMap,
  Secret, optional Ingress/SA/HPA). The API is stateless; scale horizontally.
  Note: persistence can be shared via PostgreSQL, but live run streaming uses
  an in-memory event hub, so a stream must reach the replica that owns the run.
  A shared event bus (Redis/NATS) is the remaining prerequisite for
  multi-replica streaming — captured in the roadmap; until then run a single
  replica when relying on live streams.

## 9. Security posture

- CSP: none needed for the API; the dashboard is a separate origin.
- Raw cookie values are hashed (SHA-256, truncated) before storage/transport.
- Out-of-process plugins run as a child process with no ambient credentials;
  the manifest `permissions` list is an advisory allowlist for future sandboxing.
- Request bodies are capped (1 MiB) and JSON is parsed with unknown-field
  rejection.
- The API container runs as a non-root user.

## 10. Roadmap

See `docs/roadmap.md` for Modules 2–5 (tag/cookie validation, privacy
compliance, journey testing, alerting) and the Postgres adapter plan.
