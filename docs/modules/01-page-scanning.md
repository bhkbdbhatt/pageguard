# Module 1 — Automated Page Scanning

## 1. Objective & scope

Produce a faithful, reproducible snapshot of what a site actually serves and
executes: crawl the URL graph, capture each page's HTTP exchange and (when
needed) its browser runtime, extract links, and emit an immutable
`Observation` for downstream validation.

**In scope:** crawling, scope control, `robots.txt`, HTTP + browser rendering,
link/meta extraction, SPA detection, starter tag detection, persistence,
progress events.

**Out of scope (later modules):** full tag registry reconciliation (M2),
cookie classification & consent mapping (M2/M3), scripted journeys (M4),
scheduling & alert delivery (M5).

## 2. Components

| Component | File | Responsibility |
| --- | --- | --- |
| HTTP fetcher | `internal/scanner/httpx.go` | GET with redirect chain, header/body capture, size cap |
| Renderer interface | `internal/scanner/renderer.go` | Browser abstraction + test double |
| Browser renderer | `internal/scanner/browser.go` | Playwright worker over stdio JSON |
| HTML analysis | `internal/scanner/htmlx.go` | Link/script/img/iframe extraction, title/meta, SPA heuristic |
| robots.txt | `internal/scanner/robots.go` | User-agent groups, Allow/Disallow, wildcards, `$`, crawl-delay |
| Tag detector | `internal/scanner/tags.go` | Signature-based tag recognition (starter set) |
| Scanner façade | `internal/scanner/scanner.go` | Mode selection, observation assembly, cookie hashing |
| Crawl manager | `internal/orchestrator/manager.go` | Frontier, worker pool, scope, persistence, events |
| robots cache | `internal/orchestrator/robots_cache.go` | Per-run robots fetch/cache |
| Scope globs | `internal/orchestrator/pattern.go` | Include/exclude glob → anchored regex |

## 3. Data flow

```
Execute(scan, run)
  └─ seed frontier from StartURLs (canonicalise + dedupe)
     └─ for each BFS depth:
          scanBatch(batch) ──► N workers ──► scanOne(url)
                                             ├─ Scanner.Scan
                                             │    ├─ Fetch (HTTP)  ──┐
                                             │    └─ Render (browser)┘ (mode-dependent)
                                             │    → Observation
                                             ├─ annotateRequests (host, third-party)
                                             ├─ persist Page · emit page.completed
                                             └─ Host.Validate(Observation) → Findings
          allowURL(link) = http(s) ∧ same-origin? ∧ include/exclude ∧ robots
          enqueue next-depth links (dedup)
     └─ finalize: terminal status + stats, emit run.completed
```

## 4. Configuration reference (`ScanConfig`)

| Field | Default | Meaning |
| --- | --- | --- |
| `maxDepth` | 3 | BFS depth limit (start URLs are depth 0) |
| `maxPages` | 500 | Hard cap on pages scanned per run |
| `concurrency` | 4 | Worker pool size per depth |
| `sameOriginOnly` | false | Restrict discovery to the start hosts |
| `respectRobotsTxt` | false | Honour robots.txt for discovered links |
| `includePatterns` | `[]` | Glob allowlist (matched against the full URL) |
| `excludePatterns` | `[]` | Glob denylist |
| `renderMode` | `http` | `http` \| `browser` \| `auto` |
| `renderWaitMs` | 0 | Extra wait after network-idle |
| `userAgent` | `PageguardBot/…` | UA for fetch, render, robots |
| `timeoutMs` | 20000 | Per-page timeout |
| `viewport` | 1366×768 | Browser viewport |
| `maxBodyBytes` | 4 MiB | HTTP body cap (truncation flagged) |

Start URLs always bypass scope filters and robots (the operator chose them);
filters apply to *discovered* links.

## 5. Storage schema (tables written)

- `scan_runs` — status, `stats` jsonb, timestamps.
- `run_pages` — one row per URL; unique `(run_id, url)`.
- `run_events` — the progress log (also streamed live).
- `page_requests`, `page_cookies`, `page_tags` — evidence (M2 fills these
  exhaustively; M1 populates the document request, cookies from `Set-Cookie`,
  and detected tags).
- `findings` — produced by the plugin host.

See `migrations/0001_init.sql`. The in-memory store mirrors these semantics.

## 6. Implementation notes

**URL canonicalisation.** Fragment removed; scheme and host lowercased. This is
the dedupe key for the visited set.

**Scope globs.** Patterns compile to an *anchored* case-insensitive regex
(`*`→`.*`, `?`→`.`). Anchoring means a pattern must describe the whole URL, so
`*/blog/*` matches `…/blog/post` but `/admin/*` does not match `https://x/admin`
unless written as `*/admin/*`.

**robots.txt.** Groups are parsed per user-agent; the most specific matching
group wins (longest agent token, `*` as fallback). Longest Allow/Disallow wins,
with Allow preferred on ties. Supports `*` wildcards and `$` anchors.

**HTML extraction.** A dependency-free, regex-based scanner resolves relative
references and filters non-HTTP schemes (`javascript:`, `mailto:`, `data:`, …).
It is intentionally a *function* so a full DOM parser can replace it behind the
same signature without touching callers. Known limitation: attribute-order and
exotic-quoting edge cases; covered by tests and adequate for crawling.

**SPA detection.** `LooksLikeSPA` checks for common mount points
(`#root`, `#__next`, `#app`, `data-reactroot`, `ng-version`) and small shell
documents with script bundles. Used only in `RenderAuto` to upgrade to a
browser; a browser/HTTP fetch failure always degrades gracefully.

**Cookie safety.** Only a truncated SHA-256 of the value leaves the scanner
(`valueHash`); raw values never reach storage or plugins.

**Bounded workers.** Each depth's batch is dispatched to `min(concurrency,
batchSize)` workers via an index channel; results land in a pre-sized slice at
the worker's index, avoiding result locks and preserving order.

## 7. Plugin interface

The scanner's output (`domain.Observation`) is converted to the public
`pluginsdk.Observation` by `internal/plugin/convert.go`. Validators receive it
and return `[]pluginsdk.Finding`. Module 1 ships built-in declarative rules
(`internal/plugin/builtins.go`): HTTP error status, missing title, missing meta
description, non-HTTPS, oversized HTML. See `docs/plugin-abi.md`.

## 8. Test plan & fixtures

All tests run with no external services. HTTP behaviour is exercised against
`httptest.Server` instances; browser behaviour against `scanner.FakeRenderer`.

| Test | File | Verifies |
| --- | --- | --- |
| Link extraction & filtering | `scanner/htmlx_test.go` | relative resolution, fragment/mailto/js filtering, title/meta |
| SPA heuristic | `scanner/htmlx_test.go` | mount-point detection |
| robots parsing | `scanner/robots_test.go` | groups, wildcards, `$`, crawl-delay, Allow override |
| Observation assembly | `scanner/scanner_test.go` | tags, cookie hashing/flags, error propagation |
| Render modes | `scanner/scanner_test.go` | `auto` upgrade, `browser` direct, HTTP fallback |
| Crawl + scope + robots | `orchestrator/manager_test.go` | 4-page crawl, `/private` excluded, external excluded, findings |
| `maxPages` cap | `orchestrator/manager_test.go` | never exceeds budget |
| include/exclude | `orchestrator/manager_test.go` | only allowlisted links followed |
| fatal config | `orchestrator/manager_test.go` | no valid start URLs → failed run |
| glob semantics | `orchestrator/manager_test.go` | anchored matching |

### Fixtures

- **In-process site** (`orchestrator/manager_test.go` `siteHandler`): a small
  graph (`/` → `/a`, `/b` (404), `/private`, external) with `robots.txt`
  disallowing `/private`. Asserts exact scanned set and the `core/http-error-status`
  finding.
- **SPA shell** (`scanner/scanner_test.go`): `<div id="root">` + bundle drives
  the `RenderAuto` upgrade; `FakeRenderer` supplies a rendered DOM and a
  synthetic API request.
- **Cookie fixture**: `Set-Cookie: sid=topsecret; Secure; HttpOnly; SameSite=Lax`
  asserts the raw value is absent and only a 16-hex-char hash is stored.

Run everything:

```bash
go test ./internal/scanner/... ./internal/orchestrator/...
```

## 9. Failure modes

| Failure | Behaviour |
| --- | --- |
| DNS/timeout on a page | page saved with `error`, run continues, `pagesFailed++` |
| Browser render error | falls back to HTTP observation (never fatal) |
| robots.txt missing | treated as allow-all |
| Body exceeds cap | truncated; `Observation.htmlTruncated = true` |
| No valid start URLs | run `failed` with an explanatory error |
| Client disconnect on stream | hub subscription cancelled; scan unaffected |

## 10. Future work

- Swap regex HTML parsing for a real DOM parser (behind `ExtractLinks`).
- Persist observed requests/cookies/tags to their dedicated tables (M2).
- Sitemap.xml seeding and `<link rel="canonical">` de-duplication.
- robots `crawl-delay` throttling per host.
- Shared event bus (Redis/NATS) for multi-replica streaming (the PostgreSQL
  store adapter is implemented).
