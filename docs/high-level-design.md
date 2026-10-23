# Pageguard — High-Level Design (HLD)

## 1. Purpose

Pageguard is a self-hostable web-governance and martech-auditing platform. It
crawls a site, renders pages in a real browser, detects tags/cookies/third-party
requests, and validates what it finds against a plugin-based rule engine. Output
is a stream of findings plus a per-page evidence record (`ScanResult`).

This document describes the system **as built** at the component level. See
`docs/low-level-design.md` for types, algorithms and persistence detail.

## 2. Goals & constraints

| Goal | Notes |
| --- | --- |
| Crawl + render | HTTP fetch and headless-browser render (`http`/`browser`/`auto`). |
| Evidence | Tags, cookies (hash only), requests, DOM, HAR, data layer. |
| Cross-border | Observe a page from multiple geolocations via region proxies. |
| Resumable | Per-page checkpoints; re-running is idempotent. |
| Extensible | TypeScript plugin SDK; declarative + out-of-process Node rules. |
| Streamable | Live progress over SSE + WebSocket. |
| Deployable | Single Go binary; Docker Compose + Helm; PostgreSQL storage. |

Hard constraints: Go backend, PostgreSQL, Next.js + Tailwind dashboard,
REST + WebSocket API, Docker Compose + Helm. The Go core depends only on the
standard library except `jackc/pgx` (isolated in `internal/store/postgres`).

## 3. System context

```mermaid
flowchart TB
  subgraph Actors
    OP["Operator (browser)"]
    CI["CI / scheduler"]
  end

  subgraph Platform["Pageguard platform"]
    DASH["Next.js dashboard"]
    API["REST + SSE/WS API"]
    SVC["Service layer"]
    ORCH["Orchestrator"]
    SCANNER["Scanner"]
    HOST["Plugin host"]
    HUB["Event hub"]
    STORE["domain.Store"]
  end

  PGDB[("PostgreSQL")]
  WORKER["Playwright worker (Node)"]
  WEB["Target websites"]
  PROXY["Regional egress proxies"]

  OP --> DASH
  DASH -->|"REST + SSE/WS"| API
  CI -->|"REST"| API
  API --> SVC
  SVC --> ORCH
  API --> HUB
  SVC --> STORE
  ORCH --> SCANNER
  ORCH --> HOST
  ORCH --> HUB
  ORCH --> STORE
  SCANNER --> WEB
  SCANNER --> PROXY
  SCANNER -->|"stdio JSON"| WORKER
  WORKER --> WEB
  WORKER --> PROXY
  STORE --> PGDB
```

## 4. Logical layers

```mermaid
flowchart TB
  subgraph Edge["Edge / presentation"]
    HTTP["api: REST handlers, SSE, WS relay"]
    WSC["ws: RFC6455 server"]
    UI["web: Next.js + Tailwind"]
  end

  subgraph App["Application"]
    SERVICE["service: run lifecycle, CRUD facade"]
    HUB["events: hub + run emitter"]
  end

  subgraph Core["Domain core"]
    ORCH["orchestrator: BFS, checkpoints, geo"]
    SCANNER["scanner: fetch/render/analyse"]
    PLUGIN["plugin: rules + node host"]
    PROXY["proxy: rotation + regions"]
    DOMAIN["domain: types + Store interface"]
  end

  subgraph Infra["Infrastructure"]
    MEM["store/memory"]
    PGS["store/postgres (pgx)"]
  end

  UI -->|"REST/stream"| HTTP
  HTTP --> WSC
  HTTP --> SERVICE
  HTTP --> HUB
  SERVICE --> ORCH
  SERVICE --> HUB
  ORCH --> SCANNER
  ORCH --> PLUGIN
  ORCH --> PROXY
  ORCH --> HUB
  SERVICE --> DOMAIN
  ORCH --> DOMAIN
  SCANNER --> DOMAIN
  PLUGIN --> DOMAIN
  MEM -.implements.-> DOMAIN
  PGS -.implements.-> DOMAIN
```

Dependency direction is strict: `domain` imports nothing from the project;
`scanner`, `plugin`, `events`, `proxy` depend on `domain`; `orchestrator`
depends on all of them; `service` on `orchestrator`; `api` on `service` +
`plugin`; `cmd` wires everything.

## 5. End-to-end scan flow

```mermaid
sequenceDiagram
  autonumber
  actor User
  participant API as API (REST)
  participant SVC as Service
  participant ORCH as Orchestrator
  participant SCAN as Scanner
  participant WK as Browser worker
  participant HOST as Plugin host
  participant HUB as Event hub
  participant DB as Store

  User->>API: POST /scans/{id}/runs
  API->>SVC: StartRun(scanID)
  SVC->>DB: CreateRun(queued)
  SVC->>HUB: publish run.queued
  SVC-->>API: run (202)
  SVC->>ORCH: Execute(ctx, scan, run) async
  ORCH->>HUB: run.started

  loop BFS by depth
    loop each URL in batch
      ORCH->>DB: GetScanResult(runID, url)
      alt checkpoint exists
        DB-->>ORCH: ScanResult
        ORCH->>HUB: page.resumed
      else fresh scan
        ORCH->>SCAN: ScanPage(url, opts)
        alt browser mode
          SCAN->>WK: render (JSON over stdio)
          WK-->>SCAN: html, requests, cookies, har, dataLayer
        else http mode
          SCAN->>SCAN: HTTP fetch + analyse
        end
        SCAN-->>ORCH: Observation + links
        ORCH->>DB: SavePage + SaveScanResult
        ORCH->>HOST: Validate(observation)
        HOST-->>ORCH: findings
        ORCH->>DB: SaveFinding
        ORCH->>HUB: page.completed + finding.created
      end
    end
  end

  ORCH->>DB: UpdateRun(status, stats)
  ORCH->>HUB: run.completed / run.failed
  HUB-->>User: SSE / WebSocket stream
```

## 6. Cross-border collection (multi-geolocation)

```mermaid
flowchart LR
  subgraph Config
    GEO["ScanConfig.geolocations = [us, de]"]
  end
  PICK["geoPicker round-robin"]
  ROT["proxy.Rotator"]
  CTX["browser context: coords + locale + timezone"]
  SR["ScanResult.geolocation + proxyHost"]

  GEO --> PICK --> ROT --> CTX
  PICK --> SR
  ROT --> SR
```

Each page is pinned to a region; the region selects an egress proxy and drives
the browser context, so the resulting evidence records the vantage point.

## 7. Deployment topology

```mermaid
flowchart TB
  subgraph Host["Docker host / Kubernetes cluster"]
    subgraph Compose["docker-compose"]
      API["pageguard API + orchestrator (:8080)"]
      DASH["dashboard (:3000)"]
      DB[("postgres (:5432)")]
    end
    VOL[("migrations volume")]
  end
  BROWSER["Playwright Chromium sidecar"]
  INTERNET["Internet"]

  DASH -->|"NEXT_PUBLIC_API_BASE"| API
  API --> DB
  API -.optional browser-cmd.-> BROWSER
  API --> INTERNET
  VOL --> API
```

The API runs as a non-root container and applies migrations on startup. It is
stateless apart from PostgreSQL, but live streaming uses an in-memory hub, so a
stream must reach the replica that owns the run (single replica for live
streams until a shared event bus is added).

## 8. Module roadmap

```mermaid
flowchart LR
  M1["Module 1: Page scanning"] --> M2["Module 2: Orchestration, checkpoints, geolocation"]
  M2 --> M3["Module 3: Tag & cookie validation"]
  M3 --> M4["Module 4: Privacy compliance"]
  M4 --> M5["Module 5: Journey testing & alerting"]
```

Modules 1–2 are implemented. Within Modules 3–5 the policy-based tag-validation
engine (`internal/tagpolicy`), the journey tester (`internal/journey`) and the
privacy compliance engine (`internal/privacy`, consent-state simulation) are
delivered; the remaining work (browser-based rescanner, alerting, wiring) is
designed in `docs/roadmap.md`.

## 9. Quality attributes

- **Reliability:** per-page checkpoints + idempotent resume; a page failure never
  aborts the run; plugin errors are isolated per validator.
- **Security:** raw cookie values hashed before storage; out-of-process plugins
  run with no ambient credentials; request bodies capped and unknown JSON fields
  rejected.
- **Performance:** bounded per-depth worker pool; level-synchronized BFS;
  non-blocking event fan-out so a slow client cannot stall a scan.
- **Extensibility:** `Fetcher`, `Renderer`, `Store` and `pluginsdk.Validator`
  are the seams; production and test implementations swap freely.
