# Pageguard

**Open-source web governance and martech auditing.** Pageguard crawls your
sites, executes their client-side tags in a real browser, and validates what it
finds against a plugin-based rule engine — so you can prove that every page
loads the tags it should, stores only the cookies it should, and respects the
privacy promises you make.
[text](demo.webm)
> Formerly scoped as "OpenObserve". Renamed to avoid collision with the
> existing OpenObserve observability project.

## What works today

This repository contains the **foundation** plus a complete, tested
**Module 1 (Page Scanning)** implementation.

| Area | Status |
| --- | --- |
| Go scan orchestrator (BFS crawler, worker pool, robots.txt) | ✅ implemented + tested |
| HTTP fetcher + headless-browser renderer abstraction (Playwright worker) | ✅ implemented |
| HTML/link/meta analysis, SPA detection, starter tag detector | ✅ implemented |
| Plugin host: declarative rule engine + out-of-process Node/TS validators | ✅ implemented + tested |
| TypeScript Plugin SDK | ✅ implemented |
| REST API + SSE + WebSocket progress streams | ✅ implemented + tested |
| In-memory store | ✅ implemented + tested |
| PostgreSQL store adapter (`internal/store/postgres`, pgx) + migrations | ✅ implemented + PG-gated tests |
| Tag validation engine (`internal/tagpolicy`, YAML/JSON policies → pass/fail report) | ✅ implemented + tested |
| Journey engine (`internal/journey`, scripted multi-step flows → step report) | ✅ implemented + tested |
| Privacy compliance engine (`internal/privacy`, consent-state simulation → per-regulation report) | ✅ implemented + tested |
| Next.js + Tailwind dashboard | ✅ scans, run controls, live progress, sitemap tree, findings filter/search, evidence drawer |
| Dockerfile, docker-compose, Helm chart | ✅ provided |
| Alerting (§5) | 🚧 design in `docs/roadmap.md` |

## Repository layout

```
cmd/pageguard/          Server entrypoint (API + orchestrator)
cmd/journey/            CLI runner for journey definitions
cmd/privacy/            CLI runner for privacy compliance checks
internal/
  domain/               Pure domain types + Store interface
  events/               In-process pub/sub hub + run emitter
  scanner/              HTTP fetcher, HTML/robots analysis, renderers, tag detector
  orchestrator/         Crawl manager (frontier, worker pool, scope rules)
  plugin/               Validator host, declarative rule engine, node + loader
  tagpolicy/            Tag/Cookie validation engine (YAML/JSON policies)
  journey/              Journey tester (multi-step browser flows + reports)
  privacy/              Privacy compliance engine (consent-state simulation)
  service/              Application layer (start/cancel runs)
  api/                  REST handlers, SSE + WebSocket streaming
  ws/                   Dependency-free RFC6455 WebSocket server
  store/memory/         In-memory Store implementation
pkg/pluginsdk/          Public Go plugin contract (imported by plugins)
plugins/                Example plugins (declarative + Node)
sdk-ts/                 TypeScript Plugin SDK (@pageguard/plugin-sdk)
tools/browser-renderer/ Playwright stdio worker (render + journey sessions)
web/                    Next.js dashboard
migrations/             PostgreSQL schema
deploy/                 docker-compose + Helm chart
docs/                   Architecture, module designs, plugin ABI, test plan
examples/               Example policies (tagpolicy) and journeys
schemas/                JSON Schemas (ScanResult, TagPolicy, Journey, …)
```

## Quick start

Requires **Go 1.25+**.

```bash
# Run the API + orchestrator (in-memory store, built-in rules).
go run ./cmd/pageguard --addr :8080 --plugins ./plugins

# In another shell, create a project, a scan, and start a run.
curl -s localhost:8080/api/v1/projects -H 'content-type: application/json' \
  -d '{"name":"Acme"}'
```

### PostgreSQL

Set `DATABASE_URL` to switch from the in-memory store to PostgreSQL; migrations
are applied on startup (`--migrate=false` to disable).

```bash
DATABASE_URL='postgres://pageguard:pageguard@localhost:5432/pageguard?sslmode=disable' \
  go run ./cmd/pageguard --plugins ./plugins
```

Then stream progress for a run (SSE):

```bash
curl -N localhost:8080/api/v1/runs/<runId>/events
# ...or WebSocket:
#   ws://localhost:8080/api/v1/runs/<runId>/stream
```

### Enable the headless browser

```bash
cd tools/browser-renderer && npm install && npx playwright install chromium
cd ../..
go run ./cmd/pageguard --browser-cmd "node tools/browser-renderer/index.mjs"
```

Then create scans with `"renderMode": "browser"` (or `"auto"` to upgrade only
SPA-like pages).

### Docker Compose

```bash
docker compose -f deploy/docker-compose.yml up --build
# API :8080 · dashboard :3000 · postgres :5432
```

### Dashboard

```bash
cd web && npm install && npm run dev   # http://localhost:3000
```

The dashboard creates scans, starts runs, shows live progress, a run history
per scan, and a sitemap tree of the crawled site, lets you cancel/resume runs,
filter and search findings, and open a per-page evidence drawer (tags,
third-party requests, cookies, console). Point it at a non-default API origin
with `NEXT_PUBLIC_API_BASE`. See `docs/ui.md`.

### Deploy on Netlify

Netlify can host the Next.js dashboard, but not the Go API (it needs a
long-running server, SSE/WebSocket streaming, Playwright, and PostgreSQL).

1. Host the backend container (`Dockerfile`) on Render, Fly.io, Railway, or a
   VPS with PostgreSQL, and expose it over HTTPS. `render.yaml` deploys the API
   and database on Render in one step; see `docs/render-deploy.md`.
2. In Netlify, import this repository. The root `netlify.toml` builds `web/`
   with the Next.js runtime.
3. Set `NEXT_PUBLIC_API_BASE=https://<api-host>` in Netlify before deploying;
   the value is baked in at build time.
4. Set `PAGEGUARD_CORS_ORIGINS=https://<your-site>.netlify.app` on the backend
   and restart it.

Full instructions, including Render and Fly.io examples, are in
`docs/netlify-deploy.md`.

## Development

```bash
go build ./...      # compile everything
go vet ./...        # static checks
go test ./...       # full test suite (no external services required)
```

Tests spin up in-process HTTP servers, so nothing needs Postgres, Docker, or a
browser to run. The PostgreSQL store test is skipped unless you opt in:

```bash
PAGEGUARD_TEST_DATABASE_URL='postgres://pageguard:pageguard@localhost:5432/pageguard?sslmode=disable' \
  go test ./internal/store/postgres
```

## Design principles

1. **Dependency-light core.** The Go core uses only the standard library, so it
   builds and tests offline. Third-party code is confined to isolated packages:
   `jackc/pgx` lives in `internal/store/postgres` (behind `domain.Store`) and
   `gopkg.in/yaml.v3` lives in `internal/tagpolicy` and `internal/journey`
   (behind their document loaders). Heavier integrations (browsers, stores) sit
   behind interfaces.
2. **Everything is an interface at the edges.** `Fetcher`, `Renderer`, `Store`
   and `pluginsdk.Validator` are the seams; production and test implementations
   swap freely.
3. **Plugins are a boundary, not a hook.** The TypeScript SDK talks to the host
   over a stable JSON protocol; plugin failures are isolated and never abort a
   scan.
4. **Evidence over assertions.** Findings carry the observation that produced
   them; raw cookie values are hashed at the edge and never persisted.

## Scan orchestration, checkpoints & geolocation

Module 2 makes scans resumable and cross-border aware. Each successfully scanned
page is checkpointed as a versioned `ScanResult` (JSON Schema:
`schemas/scanresult.schema.json`), so re-executing a run reuses checkpoints
instead of re-fetching and never duplicates pages or findings:

```bash
# Re-launch a finished/failed run from its checkpoints.
curl -X POST localhost:8080/api/v1/runs/<runId>/resume

# Inspect the per-page evidence.
curl localhost:8080/api/v1/runs/<runId>/results
curl "localhost:8080/api/v1/runs/<runId>/result?url=https://example.com/"
```

Observe a site from multiple regions through region-pinned egress proxies, and
optionally capture a HAR archive and client-side data layer:

```bash
go run ./cmd/pageguard \
  --proxies 'us=http://us.proxy:8080,de=http://de.proxy:8080' \
  --capture-har --capture-datalayer
```

Set `"geolocations": ["us", "de"]` on a scan to collect cross-border evidence;
each `ScanResult` records the `geolocation` and credential-free `proxyHost`.
See `docs/modules/02-scan-orchestration.md` for details.

## Tag validation policies

Given a scanned page (`domain.ScanResult`) and a user-authored **TagPolicy**
(YAML or JSON), the `internal/tagpolicy` engine validates that expected tags are
present, data-layer values match their schema, there are no duplicate or
orphaned tags, and the cookie inventory is complete and categorized — emitting a
**ValidationReport** with a pass/fail result per rule:

```go
policy, _ := tagpolicy.LoadFile("examples/tagpolicy/acme.yaml")
report := tagpolicy.Validate(scanResult, policy)
if !report.Passed {
    findings := report.Findings() // domain.Finding, pluginId "tagpolicy"
}
```

A policy supports `must_exist`, `must_not_exist`, `value_rules` and
`consent_requirements`; see `examples/tagpolicy/acme.yaml`, the JSON Schema in
`schemas/tagpolicy.schema.json`, and the reference in `docs/tag-policy.md`.

## Journey testing

A **Journey** is a scripted, multi-step browser flow (navigate / click / fill /
wait). The `internal/journey` engine runs every step in a single browser
session, validates the page after each one (URL, title, tags, data-layer events,
cookies), records the per-step diff of new requests/cookies/tags/events, and
captures a screenshot on failure — returning a step-by-step **JourneyReport**:

```go
j, _ := journey.LoadFile("examples/journey/checkout.yaml")
engine := journey.NewEngine(
    journey.NewPlaywrightSessionFactory("node", "tools/browser-renderer/index.mjs"),
)
report, _ := engine.Run(ctx, j, journey.RunOptions{ScreenshotDir: "shots"})
```

```bash
go run ./cmd/journey --journey examples/journey/checkout.yaml --format text
```

Journeys may be authored in YAML or JSON; see `examples/journey/checkout.yaml`,
the JSON Schema in `schemas/journey.schema.json`, and the reference in
`docs/journey.md`.

## Privacy compliance

A **privacy policy** declares the regulations to evaluate (GDPR, CCPA, LGPD,
ePrivacy), how to classify cookies and third-party hosts, and the CMP/rights/
DPIA/cross-border/opt-out inputs. The `internal/privacy` engine re-scans the page
under each consent state (`accept_all`, `reject_all`, `necessary_only`, `gpc`),
evaluates per-regulation checks, scores each 0–100, and returns a report with
advisory flags:

```go
cfg, _ := privacy.LoadFile("examples/privacy/acme.yaml")
report, _ := privacy.Evaluate(ctx, cfg, rescanner) // rescanner: privacy.Rescanner
findings := report.Findings()                       // → domain.Finding
```

```bash
go run ./cmd/privacy --policy examples/privacy/acme.yaml \
  --observations examples/privacy/observations.json --format text
```

Policies may be authored in YAML or JSON; see `examples/privacy/acme.yaml`, the
JSON Schema in `schemas/privacy.schema.json`, and the reference in
`docs/privacy.md`.

## License

MIT.
