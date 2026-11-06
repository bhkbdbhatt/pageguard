# Idea: Pageguard

**Captured:** 2026-11-06
**Slug:** pageguard
**Source:** D:\openobserve (repository review)

## One-liner

For mid-market ecommerce and martech teams with 500-5,000 pages and weekly releases, Pageguard is a self-hosted, real-browser audit that proves which tags, cookies, and beacons actually fired against policy, instead of an enterprise ObservePoint/Tag Inspector contract or a manual QA checklist.

## Raw description (verbatim)

> look at the repo and make a file for roast me and come up with the next phase plan in another plan-roast.md which has actualy customer retention and build up

## What the repository actually contains

Pageguard (formerly scoped as "OpenObserve") is an open-source web governance and martech auditing platform. It crawls sites, executes client-side tags in a real browser, and validates what it finds against a plugin-based rule engine. The repository contains:

- A Go scan orchestrator with BFS crawling, worker pool, robots.txt handling, resumable page checkpoints, and multi-geolocation proxy support.
- HTTP and Playwright browser rendering, HAR capture, data-layer capture, tag detection, cookie detection, and request capture.
- A plugin host with declarative rules and out-of-process Node/TypeScript validators.
- Tag/cookie policy validation, multi-step journey testing, and privacy/consent simulation.
- REST API, SSE/WebSocket progress streams, PostgreSQL and in-memory stores, cron schedules, alerts, incidents, reports, and a Next.js dashboard.
- 123 Go files, roughly 22.4k lines of Go, 35 test files, and passing tests.

## Repo facts that frame the roast

| Fact | Evidence |
| --- | --- |
| Claims MIT license | `README.md:276-278`; `marketing/index.html` repeats "MIT licensed" |
| No LICENSE file and no GitHub license | Repository root; GitHub API license field is empty |
| No CI workflow | `.github/` contains only `FUNDING.yml` |
| Security policy is an unedited template | `SECURITY.md:5`, `SECURITY.md:17` |
| 15 built-in tag signatures | `internal/scanner/tags.go:102` |
| 4 martech vendors | `internal/martech/martech.go:22` |
| Findings have no owner/status/verification | `internal/domain/observation.go:51` |
| Auth middleware exists but is never enabled | `internal/api/api.go:40`; no `WithAuth(...)` call in `cmd/pageguard/main.go` |
| 0 GitHub stars, 0 forks, 0 releases | GitHub API, repository created 2026-10-23 |
| Repository was renamed, links were not | Remote is `bhkbdbhatt/openobserve`; GitHub repo is now `bhkbdbhatt/pageguard` |

## Assumed buyer

The repository does not name one. The roast assumes the narrowest plausible buyer: a mid-market ecommerce or lead-generation company with a real tag manager, multiple third-party pixels, a consent banner, 500-5,000 pages, and at least weekly releases. The person who feels the pain is the analytics lead or growth engineer; the person who signs is the head of digital, analytics, or engineering.
