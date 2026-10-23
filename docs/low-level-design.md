# Pageguard — Low-Level Design (LLD)

Companion to `docs/high-level-design.md`. This document details packages, types,
algorithms, persistence and concurrency. All diagram labels reflect the code in
`D:\openobserve`.

## 1. Package dependency graph

```mermaid
flowchart LR
  CMD["cmd/pageguard"]
  API["internal/api"]
  WS["internal/ws"]
  SVC["internal/service"]
  ORCH["internal/orchestrator"]
  SCAN["internal/scanner"]
  PLUG["internal/plugin"]
  PROXY["internal/proxy"]
  EVT["internal/events"]
  DOM["internal/domain"]
  MEM["internal/store/memory"]
  PGS["internal/store/postgres"]
  SDK["pkg/pluginsdk"]
  TAGP["internal/tagpolicy"]
  JRNY["internal/journey"]
  PRIV["internal/privacy"]

  CMD --> API
  CMD --> SVC
  CMD --> ORCH
  CMD --> SCAN
  CMD --> PLUG
  CMD --> PROXY
  CMD --> MEM
  CMD --> PGS
  API --> SVC
  API --> PLUG
  API --> WS
  SVC --> ORCH
  SVC --> EVT
  SVC --> DOM
  ORCH --> SCAN
  ORCH --> PLUG
  ORCH --> PROXY
  ORCH --> EVT
  ORCH --> DOM
  SCAN --> DOM
  SCAN --> PROXY
  PLUG --> DOM
  PLUG --> SDK
  PROXY --> DOM
  EVT --> DOM
  MEM --> DOM
  PGS --> DOM
  TAGP --> DOM
  JRNY --> DOM
  JRNY --> SCAN
  PRIV --> DOM
```

## 2. Domain model

```mermaid
classDiagram
  class Project {
    +ID id
    +string name
    +string slug
    +time createdAt
    +time updatedAt
  }

  class Scan {
    +ID id
    +ID projectId
    +string name
    +string[] startUrls
    +ScanConfig config
    +string[] ruleSetIds
    +bool enabled
  }

  class ScanConfig {
    +int maxDepth
    +int maxPages
    +int concurrency
    +bool sameOriginOnly
    +bool respectRobotsTxt
    +string[] includePatterns
    +string[] excludePatterns
    +RenderMode renderMode
    +int renderWaitMs
    +string userAgent
    +int timeoutMs
    +Viewport viewport
    +int64 maxBodyBytes
    +string[] geolocations
    +bool captureHAR
    +bool captureDataLayer
    +Normalize()
  }

  class Run {
    +ID id
    +ID scanId
    +ID projectId
    +RunStatus status
    +RunStats stats
    +string error
    +time createdAt
    +time startedAt
    +time finishedAt
  }

  class RunStats {
    +int pagesDiscovered
    +int pagesScanned
    +int pagesFailed
    +int findings
  }

  class Page {
    +ID id
    +ID runId
    +string url
    +string finalUrl
    +int status
    +int depth
    +bool rendered
    +string title
    +string contentType
    +int64 durationMs
    +string error
  }

  class Observation {
    +string url
    +string finalUrl
    +int statusCode
    +string contentType
    +string title
    +string[] headers
    +string html
    +bool htmlTruncated
    +NetworkRequest[] requests
    +Cookie[] cookies
    +Tag[] tags
    +ConsoleMessage[] console
    +DataLayerEntry[] dataLayer
    +json har
    +string geolocation
    +bool rendered
    +int depth
    +ID pageId
  }

  class ScanResult {
    +string schemaVersion
    +ID runId
    +ID scanId
    +ID pageId
    +string url
    +string finalUrl
    +int depth
    +int attempt
    +bool resumed
    +int statusCode
    +string renderMode
    +string geolocation
    +string proxyHost
    +string dom
    +NetworkRequest[] requests
    +Cookie[] cookies
    +DataLayerEntry[] dataLayer
    +json har
    +Tag[] tags
    +string[] links
    +string error
  }

  class Cookie {
    +string name
    +string domain
    +string path
    +string valueHash
    +bool secure
    +bool httpOnly
    +string sameSite
    +bool session
    +bool firstParty
  }

  class NetworkRequest {
    +string method
    +string url
    +string host
    +int status
    +string resourceType
    +bool thirdParty
  }

  class Tag {
    +string id
    +string vendor
    +string name
    +TagCategory category
    +string evidence
    +float confidence
  }

  class Finding {
    +ID id
    +ID runId
    +ID pageId
    +string pageUrl
    +string pluginId
    +string ruleId
    +Severity severity
    +string title
    +string description
    +object evidence
  }

  Project "1" --> "*" Scan : owns
  Scan "1" --> "*" Run : executes
  Run "1" --> "*" Page : produces
  Run "1" --> "*" ScanResult : checkpoints
  Run "1" --> "*" Finding : yields
  Observation "1" --> "*" NetworkRequest
  Observation "1" --> "*" Cookie
  Observation "1" --> "*" Tag
  ScanResult ..> Observation : serialises
```

## 3. Identifiers and enums

- `domain.ID` is a 32-character lowercase hex string (`NewID()`); PostgreSQL
  stores it as `uuid` via `toUUID` / `toID` (dashless accepted, normalized).
- `RenderMode`: `http` | `browser` | `auto`.
- `RunStatus`: `queued` | `running` | `succeeded` | `failed` | `canceled`.
- `Severity`: `info` | `low` | `medium` | `high` | `critical`.
- `TagCategory`: `analytics` | `advertising` | `tag_manager` | `cdp` |
  `consent` | `monitoring` | `unknown`.

## 4. Run state machine

```mermaid
stateDiagram-v2
  [*] --> queued : StartRun
  queued --> running : Execute begins
  running --> succeeded : finalize(no error)
  running --> failed : fatal error / no start URLs
  running --> canceled : ctx cancelled
  succeeded --> queued : ResumeRun
  failed --> queued : ResumeRun
  canceled --> queued : ResumeRun
  succeeded --> [*]
  failed --> [*]
  canceled --> [*]
```

## 5. Orchestrator (`internal/orchestrator`)

`Manager` fields: `store domain.Store`, `hub *events.Hub`,
`scanner *scanner.Scanner`, `plugins *plugin.Host`, `rotator *proxy.Rotator`,
`captureHAR`, `captureDataLayer`. Options: `WithProxyRotator`, `WithCapture`.

### 5.1 Execute algorithm

```mermaid
flowchart TD
  START["Execute(ctx, scan, run)"] --> NORM["cfg.Normalize()"]
  NORM --> RESET["reset run.Stats + Error"]
  RESET --> EMIT["emitter = events.NewEmitter"]
  EMIT --> SEED["seed frontier from StartURLs"]
  SEED --> EMPTY{"any start URL?"}
  EMPTY -->|no| FAIL["finalize(reason)"]
  EMPTY -->|yes| LOOP{"depth loop:<br/>current non-empty and scanned below MaxPages"}
  LOOP -->|no| FIN["finalize()"]
  LOOP -->|yes| BATCH["batch = next slice of current"]
  BATCH --> WORK["scanBatch: bounded worker pool"]
  WORK --> COLLECT["collect pageResult per index"]
  COLLECT --> NEXT["build next frontier from links"]
  NEXT --> LOOP
```

### 5.2 scanOne: checkpoint or fresh

```mermaid
flowchart TD
  S["scanOne(item)"] --> STARTED["emit page.started"]
  STARTED --> GET["store.GetScanResult(runID, url)"]
  GET --> HIT{"checkpoint found?"}
  HIT -->|yes| REBUILD["observationFromScanResult + linksFromScanResult"]
  REBUILD --> RESUMED["return resumed=true"]
  HIT -->|no| GEO["geolocation = geoPicker.next()"]
  GEO --> OPTS["ScanOptions{Depth, Geolocation, Capture*, Proxy*}"]
  OPTS --> ROT{"rotator set?"}
  ROT -->|yes| PICK["rotator.Next(region) -> proxy"]
  ROT -->|no| CALL
  PICK --> CALL["scanner.ScanPage(...)"]
  CALL --> ERR{"error?"}
  ERR -->|yes| REPORT["rotator.Report(err); emit page.failed"]
  ERR -->|no| OK["rotator.Report(nil)"]
  OK --> ANNOT["annotateRequests (host + thirdParty)"]
  ANNOT --> BUILD["buildScanResult (DOM capped 2 MiB)"]
  BUILD --> SAVE["store.SaveScanResult"]
  SAVE --> DONE["return pageResult"]
```

### 5.3 finalize: reconciled statistics

`finalize` sets the terminal status, then **reconciles** `PagesScanned`,
`PagesFailed` and `Findings` from the store (`ListPages` + `ListFindings`) so a
resumed run reports the same totals as a fresh one. `PagesDiscovered` comes from
the emitter (rebuilt during traversal).

### 5.4 Scope decisions

`allowURL` applies, in order: scheme (`http`/`https`), same-origin policy,
include globs, exclude globs, then `robots.txt` (user-agent aware, longest
match, wildcards, `$`). The `geoPicker` uses an atomic counter for lock-free
round-robin across workers.

## 6. Scanner (`internal/scanner`)

```mermaid
flowchart TD
  SP["ScanPage(url, cfg, opts)"] --> NORM["cfg.Normalize()"]
  NORM --> BROWSER{"mode==browser AND renderer?"}
  BROWSER -->|yes| RENDER["render() -> finalize()"]
  BROWSER -->|no| FETCH["Fetcher.Fetch (HTTP)"]
  FETCH --> OBS["build Observation from FetchResult"]
  OBS --> AUTO{"mode==auto AND looksLikeSPA?"}
  AUTO -->|yes| RENDER
  AUTO -->|no| FIN["finalize(obs)"]
  RENDER --> FIN2["finalize(rendered obs)"]
  FIN --> OUT["PageResult{Observation, Links, MetaDescription, HAR, DataLayer}"]
  FIN2 --> OUT
```

Key helpers:

- `cookiesFromHeaders` / worker cookie mapping → `Cookie` with `hashValue`
  (SHA-256 truncated to 8 bytes → 16 hex chars). Raw values never leave the edge.
- `ExtractLinks(base *url.URL, html)` returns `Links{Documents,...}` used for the
  frontier; `ExtractMetaDescription`, `ExtractTitle`.
- `LooksLikeSPA` drives `auto` upgrade; browser failure leaves the HTTP result.
- Renderers implement `Renderer` (`Render`, `Close`): `BrowserRenderer` speaks
  newline-delimited JSON to `tools/browser-renderer/index.mjs`; `FakeRenderer`
  serves tests.

### 5.4 Render worker protocol

```mermaid
sequenceDiagram
  participant SC as BrowserRenderer (Go)
  participant WK as index.mjs (Node)
  SC->>WK: {"url":..., "opts":{userAgent,timeoutMs,waitMs,geolocation,geoLatitude,geoLongitude,locale,timezone,proxyServer,proxyUser,proxyPass,captureHar,captureDataLayer}}
  WK->>WK: launch chromium, newContext(geo/proxy/locale/tz/HAR)
  WK->>WK: page.goto, wait, evaluate dataLayer
  WK-->>SC: {"ok":true,"result":{finalUrl,status,contentType,html,title,requests,cookies,console,dataLayer,har,durationMs}}
```

## 6. Proxy & geolocation (`internal/proxy`)

```mermaid
stateDiagram-v2
  [*] --> healthy
  healthy --> cooling : Report(err), failures >= threshold
  cooling --> healthy : cooldown elapsed
  note right of healthy
    Next(region) picks round-robin;
    falls back to regionless proxies.
  end note
```

- `Region` catalog (`us`, `ca`, `uk`, `ie`, `de`, `fr`, `nl`, `au`, `jp`, `sg`,
  `in`, `br`) maps labels to coordinates, locale and timezone.
- `Rotator.Next(region)` selects a healthy proxy for the region; `Report(err)`
  increments failures and applies a cooldown threshold.
- `ParseProxies` parses `region=scheme://host:port` specs with region inheritance
  from the previous entry.
- `RoundTripper` reads the region from context (`WithRegion`) and retries up to
  `MaxAttempts`.

## 7. Plugin host (`internal/plugin`)

```mermaid
sequenceDiagram
  autonumber
  participant ORCH as Orchestrator
  participant HOST as plugin.Host
  participant REG as Registry
  participant V as Validator (builtin | node)
  participant CONV as convert

  ORCH->>HOST: Validate(ValidateInput)
  HOST->>CONV: ToSDK(Observation) + Scope
  loop each validator allowed by ruleSetIds
    HOST->>V: Validate(ctx, sdkObs, scope)
    V-->>HOST: []Finding (SDK)
    Note over HOST: plugin error is isolated (continue)
    HOST->>CONV: map SDK findings -> domain.Finding
  end
  HOST-->>ORCH: []*domain.Finding
```

- `runtime: builtin` — `ruleengine` evaluates declarative `RuleSpec` predicates
  against the SDK observation.
- `runtime: node` — `node.go` spawns a child process and speaks one-shot JSON
  over stdio (`pluginsdk.ProtocolVersion = 1`).
- `loader` reads each plugin directory's `manifest.json`.

## 8. Events & streaming (`internal/events`, `internal/api`, `internal/ws`)

```mermaid
sequenceDiagram
  autonumber
  participant ORCH as Orchestrator
  participant EM as Emitter
  participant HUB as Hub
  participant DB as Store (run_events)
  participant CL as SSE / WS client

  ORCH->>EM: IncScanned / PageCompleted / Finding
  EM->>HUB: Publish(event)
  EM->>DB: AppendEvent (best effort)
  HUB-->>CL: buffered channel (drop oldest if slow)
  CL-->>CL: render JSON frame
```

- `Hub` fan-out is non-blocking with a 256-event per-subscriber buffer;
  `dropped` counts overflow. Each event carries a monotonic global ID.
- `api` exposes both `GET /runs/{id}/events` (SSE) and
  `GET /runs/{id}/stream` (WebSocket) over the same payloads.
- Event types include `run.queued`, `run.started`, `page.discovered`,
  `page.started`, `page.completed`, `page.resumed`, `page.failed`,
  `finding.created`, `run.completed`, `run.failed`, `log`.

## 9. Persistence

`domain.Store` is the contract; `internal/store/memory` (maps + mutexes) and
`internal/store/postgres` (pgx) implement it.

### 9.1 Store interface

```mermaid
classDiagram
  class Store {
    <<interface>>
    +CreateProject()
    +GetProject()
    +ListProjects()
    +CreateScan()
    +UpdateScan()
    +GetScan()
    +ListScans()
    +CreateRun()
    +UpdateRun()
    +GetRun()
    +ListRuns()
    +SavePage()
    +GetPage()
    +ListPages()
    +SaveScanResult()
    +GetScanResult()
    +ListScanResults()
    +SaveFinding()
    +ListFindings()
    +AppendEvent()
    +ListEvents()
  }
  class Memory
  class Postgres
  Store <|.. Memory
  Store <|.. Postgres
```

### 9.2 Relational schema (ER)

```mermaid
erDiagram
  projects ||--o{ scans : owns
  projects ||--o{ scan_runs : owns
  projects ||--o{ journeys : owns
  projects ||--o{ alert_rules : owns
  scans ||--o{ scan_runs : executes
  scan_runs ||--o{ run_pages : "produces pages"
  scan_runs ||--o{ scan_results : "checkpoints by run and url"
  scan_runs ||--o{ findings : yields
  scan_runs ||--o{ run_events : emits
  run_pages ||--o{ page_requests : observed
  run_pages ||--o{ page_cookies : observed
  run_pages ||--o{ page_tags : detected
  run_pages ||--o{ findings : "linked page"
  alert_rules ||--o{ alert_events : triggers

  projects {
    uuid id PK
    text name
    text slug UK
  }
  scans {
    uuid id PK
    uuid project_id FK
    text name
    text start_urls
    jsonb config
    text rule_set_ids
    bool enabled
  }
  scan_runs {
    uuid id PK
    uuid scan_id FK
    uuid project_id FK
    text status
    jsonb stats
    text error
  }
  run_pages {
    uuid id PK
    uuid run_id FK
    text url
    int status
    int depth
    bool rendered
  }
  scan_results {
    uuid run_id PK
    text url PK
    jsonb result
  }
  findings {
    uuid id PK
    uuid run_id FK
    uuid page_id FK
    text plugin_id
    text rule_id
    text severity
    jsonb evidence
  }
  page_cookies {
    bigint id PK
    uuid page_id FK
    text name
    text value_hash
    bool first_party
  }
  page_requests {
    bigint id PK
    uuid page_id FK
    text method
    text url
    text host
    bool third_party
  }
  page_tags {
    bigint id PK
    uuid page_id FK
    text tag_id
    text vendor
    text category
  }
  run_events {
    bigint id PK
    uuid run_id FK
    text type
    jsonb payload
  }
```

### 9.3 ScanResult checkpoint semantics

- Keyed by `(run_id, url)` with an upsert, so re-execution is idempotent.
- Stored as a single `jsonb` column matching `schemas/scanresult.schema.json`
  (version `ScanResultSchemaVersion = "1.0.0"`).
- DOM is capped at `maxStoredDOM = 2 MiB`; the raw HTML length is kept in
  `domBytes` and truncation is flagged by `domTruncated`.
- Cookie values are stored only as `valueHash` (SHA-256 truncated to 8 bytes →
  16 hex chars).

## 10. HTTP API

| Method | Path | Handler |
| --- | --- | --- |
| GET | `/healthz` | `handleHealth` |
| GET/POST | `/api/v1/projects` | `handleListProjects` / `handleCreateProject` |
| GET/POST | `/api/v1/scans` | `handleListScans` / `handleCreateScan` |
| GET | `/api/v1/scans/{id}` | `handleGetScan` |
| POST | `/api/v1/scans/{id}/runs` | `handleStartRun` |
| GET | `/api/v1/runs/{id}` | `handleGetRun` |
| POST | `/api/v1/runs/{id}/cancel` | `handleCancelRun` |
| POST | `/api/v1/runs/{id}/resume` | `handleResumeRun` (409 if in flight) |
| GET | `/api/v1/runs/{id}/pages` | `handleListPages` |
| GET | `/api/v1/runs/{id}/results` | `handleListResults` |
| GET | `/api/v1/runs/{id}/result?url=` | `handleGetResult` |
| GET | `/api/v1/runs/{id}/findings` | `handleListFindings` |
| GET | `/api/v1/runs/{id}/events` | `handleSSE` |
| GET | `/api/v1/runs/{id}/stream` | `handleWS` |
| GET | `/api/v1/plugins` | `handleListPlugins` |

## 11. Concurrency & error handling

```mermaid
flowchart LR
  subgraph Run["One goroutine per run"]
    BFS["level-synchronized BFS"]
    POOL["bounded worker pool (cfg.concurrency)"]
    IDX["results written by index (no lock)"]
  end
  HUB2["Hub fan-out: short lock, non-blocking send"]
  CXL["context cancellation -> canceled status"]

  BATCH["depth batch"] --> POOL --> IDX
  ORCHCTX["run ctx"] -.cancels.-> BATCH
```

- **One goroutine per run**; runs are independent.
- **Bounded worker pool per depth**; results written to a pre-sized slice by
  index (no result locking). Level-synchronized BFS gives deterministic
  accounting and deadlock-free scheduling.
- **Event fan-out** never blocks the producer.
- **Error handling:** page errors are recorded (`PagesFailed`) and do not abort
  the run; only bad configuration/cancellation is fatal; plugin errors are
  isolated per validator.

## 11. Extension points

| Seam | Interface | Production | Test |
| --- | --- | --- | --- |
| Fetch | `scanner.Fetcher` | `HTTPFetcher` | `fakeFetcher` |
| Render | `scanner.Renderer` | `BrowserRenderer` | `FakeRenderer` / recording |
| Persist | `domain.Store` | `store/postgres` | `store/memory` |
| Rules | `pluginsdk.Validator` | builtin + Node plugins | ruleengine tests |
| Proxy | `proxy.Rotator` | `proxy.Rotator` | rotator tests |

## 11.1 Configuration defaults (`ScanConfig.Normalize`)

| Field | Default |
| --- | --- |
| `MaxDepth` | 3 |
| `MaxPages` | 500 |
| `Concurrency` | 4 |
| `RenderMode` | `http` |
| `TimeoutMs` | 20000 |
| `MaxBodyBytes` | 4 MiB |
| `Viewport` | 1366 × 768 |
| `UserAgent` | `PageguardBot/0.1 (+https://github.com/pageguard/pageguard)` |

## 11. Testing map

| Package | Coverage |
| --- | --- |
| `internal/proxy` | catalog, `ParseProxies`, round-robin, cooldown, dead proxy |
| `internal/scanner` | HTTP extraction, tags/cookies, geo/proxy/capture propagation |
| `internal/orchestrator` | BFS, scope/robots, resume idempotency, geo+proxy |
| `internal/plugin` | rule engine, node host, loader |
| `internal/store/memory` | CRUD + `ScanResult` upsert |
| `internal/store/postgres` | full round-trip (gated on `PAGEGUARD_TEST_DATABASE_URL`) |
| `internal/domain` | `ScanResult` JSON round-trip + schema contract |
| `internal/tagpolicy` | policy load (YAML/JSON), value rules, duplicates/orphans, consent, report + schema lockstep |
| `internal/journey` | journey load (YAML/JSON), step execution pass/fail/error, per-step diffs, data-layer events, worker protocol + schema lockstep |
| `internal/privacy` | policy load (YAML/JSON), consent-state simulation, per-regulation checks, scoring, flags, findings + schema lockstep |
| `internal/api`, `internal/events`, `internal/ws` | handlers, hub fan-out, RFC6455 |
