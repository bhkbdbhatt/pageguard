# Pageguard Dashboard (Web UI)

The dashboard is a **Next.js (App Router) + React + Tailwind CSS** single-page
app under `web/`. It is a thin client over the REST + SSE API: it holds no
server state and talks to the API through `web/lib/api.ts`.

```
web/
  app/
    layout.tsx          App shell (header + main container)
    page.tsx            Dashboard: scans, create form, run history
    globals.css         Tailwind entry + base styles
    runs/[id]/page.tsx  Run detail: progress, controls, findings, pages, log
  components/
    Sitemap.tsx         Crawled-site tree built from ScanResult links
  lib/api.ts            Typed API client (fetch wrappers + domain types)
tailwind.config.ts      Theme (brand palette)
```

## Dashboard — `/`

- **Create scan** — name, one-or-more start URLs. Posts to
  `POST /api/v1/scans` with default crawl config
  (`maxDepth: 3`, `maxPages: 200`, `concurrency: 4`, same-origin, respect
  `robots.txt`).
- **Scans table** — every scan for the current project.
- **Run** — starts a run (`POST /api/v1/scans/{id}/runs`) and navigates to the
  run detail page.
- **History** — expands an inline panel showing that scan's previous runs
  (`GET /api/v1/scans/{id}/runs`), newest first. Each row shows status, scanned
  / failed / findings totals, start time, and an **Open** link to the run.

## Run detail — `/runs/[id]`

- **Status & progress** — a progress bar derived from run stats
  (`pagesScanned + pagesFailed` over `pagesDiscovered`); terminal runs show
  100%. Live counts for scanned, failed, findings, and start/finish times.
- **Run controls**
  - **Cancel** while the run is `queued`/`running` →
    `POST /api/v1/runs/{id}/cancel`.
  - **Resume** for a terminal run → `POST /api/v1/runs/{id}/resume`
    (idempotent; already-scanned pages are loaded from checkpoints).
- **Severity summary** — the five severity cards are also filters. Click a card
  to show only that severity; click again (or click **all**) to clear.
- **Findings** — filterable list. The severity filter and the free-text search
  box match on title, rule ID, plugin ID, page URL, and description. Click a
  finding to open the evidence drawer.
- **Pages** — every visited URL with its status (errors/4xx–5xx in red). Click a
  page to open the evidence drawer for that URL.
- **Sitemap** — a tree view of the crawled site, built from the per-page
  checkpoints (`GET /api/v1/runs/{id}/results`). Each node is a page; nesting
  follows the discovered document links, so roots are the start URLs and
  children are the links found on their parents. Nodes are colour-coded by
  outcome (ok / redirect / 4xx–5xx / error), cycles are marked, and clicking a
  node opens its evidence drawer. The header shows the page count and max crawl
  depth.
- **Live log** — the last 200 progress events, streamed over
  **Server-Sent Events** (`GET /api/v1/runs/{id}/events`, via `EventSource`).
  Every event also triggers a refresh of the run, pages, and findings.
- **Evidence drawer** — a slide-over that fetches the page's
  `ScanResult` (`GET /api/v1/runs/{id}/result?url=`) and shows:
  - the finding itself (severity, rule, plugin, description, raw evidence JSON);
  - page metadata (status, rendered, depth, duration, geolocation, proxy, DOM
    size);
  - detected **tags**, **network requests** (third-party marked, first 100
    shown), **cookies** (first/third party, secure / HttpOnly / SameSite), and
    **console** messages.

  Findings record the *final* URL (after redirects) while a `ScanResult` is
  keyed by the *original* URL, so the drawer resolves the finding's URL back to
  the checkpoint key using the pages list.

## API client

`web/lib/api.ts` exports typed wrappers and the response shapes used by the UI.
Configure the backend with `NEXT_PUBLIC_API_BASE` (default `http://localhost:8080`).

| Function | Endpoint |
| --- | --- |
| `listProjects` / `createProject` | `GET` / `POST /api/v1/projects` |
| `listScans` / `createScan` | `GET` / `POST /api/v1/scans` |
| `startRun` | `POST /api/v1/scans/{id}/runs` |
| `listScanRuns` | `GET /api/v1/scans/{id}/runs` |
| `getRun` | `GET /api/v1/runs/{id}` |
| `cancelRun` | `POST /api/v1/runs/{id}/cancel` |
| `resumeRun` | `POST /api/v1/runs/{id}/resume` |
| `listPages` | `GET /api/v1/runs/{id}/pages` |
| `listFindings` | `GET /api/v1/runs/{id}/findings` |
| `listScanResults` | `GET /api/v1/runs/{id}/results` |
| `getScanResult` | `GET /api/v1/runs/{id}/result?url=` |
| `runEventStreamURL` | `GET /api/v1/runs/{id}/events` (SSE) |
| `runWebSocketURL` | `GET /api/v1/runs/{id}/stream` (WebSocket) |

## Local development

```bash
# Terminal 1 — API (in-memory store, built-in rules)
go run ./cmd/pageguard --addr :8080 --plugins ./plugins

# Terminal 2 — dashboard
cd web && npm install && npm run dev   # http://localhost:3000
```

Set `NEXT_PUBLIC_API_BASE` to point at a non-default API origin. Under Docker
Compose the dashboard is published on `:3000`.
