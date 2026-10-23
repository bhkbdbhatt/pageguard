# Module 2 — Scan Orchestration, Checkpoints & Geolocation

This module turns the scanner into a resumable, idempotent, geo-aware crawl.

## 1. Goals

- Emit a structured, per-page evidence record (`domain.ScanResult`) that is the
  single source of truth for a scanned page.
- Make runs **resumable** and **idempotent**: re-executing a run never re-fetches
  a page that already has a checkpoint, and never duplicates pages/findings.
- Detect **cross-border data collection** by observing a page from multiple
  geolocations through region-pinned egress proxies.
- Capture optional browser evidence: a HAR network archive and the client-side
  data layer (`window.dataLayer`, `digitalData`, …).

## 2. The ScanResult record

`internal/domain/scanresult.go` defines `ScanResult`, versioned by
`ScanResultSchemaVersion` (`1.0.0`). The published JSON schema is
`schemas/scanresult.schema.json`; `internal/domain/scanresult_test.go` asserts
the Go struct and the schema stay in lockstep.

Highlights:

| Field | Meaning |
| --- | --- |
| `schemaVersion` | Contract version; a change is breaking. |
| `runId`/`scanId`/`pageId` | 32-hex identifiers. |
| `url`/`finalUrl`/`depth` | Crawl coordinates (keyed by `runId`+`url`). |
| `attempt`/`resumed` | Distinguishes a fresh scan from a reused checkpoint. |
| `renderMode`/`rendered` | `http` \| `browser` \| `auto`. |
| `geolocation`/`proxyHost` | Vantage point; proxy credentials stripped. |
| `dom`/`domTruncated`/`domBytes` | Serialized document (capped at 2 MiB). |
| `requests`/`cookies`/`console` | Network + cookie (hash only) + console errors. |
| `dataLayer`/`har` | Optional GTM/Adobe payloads and HAR 1.2 archive. |
| `tags`/`links`/`metaDescription` | Detection + crawl metadata. |

## 3. Checkpointing and resume

- `Manager.Execute` (`internal/orchestrator/manager.go`) walks the frontier
  breadth-first. Before scanning a URL it calls
  `store.GetScanResult(runID, url)`; a hit rebuilds an `Observation` and links
  from the checkpoint and is marked `resumed`, so no fetch/render occurs.
- Fresh pages are wrapped by `buildScanResult` and written with
  `store.SaveScanResult`. The store upserts on `(run_id, url)`, so a rerun is
  idempotent.
- `finalize` **reconciles page and finding totals from persistence** rather than
  trusting accumulated counters, so a resumed run reports the same
  `RunStats.PagesScanned` / `Findings` as the original.
- `service.ResumeRun` + `POST /api/v1/runs/{id}/resume` re-launch a finished or
  failed run from its checkpoints (returns `409` if the run is still in flight).

## 4. Geolocation & proxy rotation

- `Config.Geolocations` lists region labels. A `geoPicker` assigns them to pages
  in round-robin order (atomic counter, safe across the worker pool).
- `internal/proxy`:
  - `Region` catalog maps labels (`us`, `de`, …) to coordinates/locale/timezone.
  - `Rotator.Next(region)` selects a healthy egress proxy for that region;
    `Report` applies a failure threshold and cooldown so bad proxies are skipped.
  - `RoundTripper` wires rotation into `net/http` via a region context value.
  - `ParseProxies` accepts `region=scheme://host:port` specs with region
    inheritance from the previous entry.
- The browser renderer receives the region (coordinates/locale/timezone) and the
  region's proxy per render; both are recorded on the resulting `ScanResult`.

## 5. Capture options

`WithCapture(har, dataLayer)` on the manager (or `--capture-har` /
`--capture-datalayer` on the CLI) requests a HAR archive and/or data-layer
payloads from the browser worker (`tools/browser-renderer/index.mjs`). HTTP-mode
scans simply leave these empty.

## 6. API

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/v1/runs/{id}/resume` | Resume from checkpoints (409 if running). |
| `GET` | `/api/v1/runs/{id}/results` | List a run's `ScanResult`s. |
| `GET` | `/api/v1/runs/{id}/result?url=` | Fetch one page's `ScanResult`. |

## 7. Tests

- `internal/proxy` — catalog, `ParseProxies`, round-robin, cooldown/recovery,
  regionless fallback, dead-proxy error, direct passthrough.
- `internal/scanner` — `ScanPage` geo/proxy/capture propagation; `Scan` wrapper.
- `internal/orchestrator` — resume idempotency (no refetch, no duplicate
  pages/findings, stable stats, `page.resumed` events), geolocation + proxy
  assignment, geo picker.
- `internal/store/memory` & `internal/store/postgres` — `ScanResult` upsert /
  get / list (Postgres gated on `PAGEGUARD_TEST_DATABASE_URL`).
- `internal/domain` — JSON round-trip and Go-struct↔schema contract.
