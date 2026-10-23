# Journey Testing

The journey engine (`internal/journey`) executes a scripted, multi-step user
flow in a real headless browser and validates the page after each step. A
**Journey** is an ordered list of **steps** — navigate, click, fill or wait — and
each step may declare expectations about the resulting page (URL, title, tags,
data-layer events, cookies).

Unlike a scan, which renders many independent URLs, a journey runs every step in
a **single browser session**, so cookies, storage and the DOM persist from one
step to the next. After each step the engine captures the full page state,
detects tags, checks the step's expectations, and records what changed since the
previous step. The output is a step-by-step **JourneyReport**, with a screenshot
captured on any failure.

The package is library-first: it depends only on the standard library plus the
existing `internal/scanner` tag detector and `internal/domain`. The real browser
driver talks to the same Playwright worker as the scanner
(`tools/browser-renderer`); a `FakeSession` keeps tests offline.

## Quick start

```go
j, _ := journey.LoadFile("examples/journey/checkout.yaml")

engine := journey.NewEngine(
    journey.NewPlaywrightSessionFactory("node", "tools/browser-renderer/index.mjs"),
)
report, err := engine.Run(ctx, j, journey.RunOptions{ScreenshotDir: "shots"})
if err != nil {
    log.Fatal(err)
}
if !report.Passed {
    _ = (journey.TextFormatter{}).Format(os.Stdout, report)
}
```

Or from the command line:

```bash
go run ./cmd/journey \
  --journey examples/journey/checkout.yaml \
  --browser-cmd "node tools/browser-renderer/index.mjs" \
  --format text --screenshots ./shots
# exits 0 when every step passes, 1 otherwise
```

Journeys may be authored in **YAML** (recommended) or **JSON**. Both encodings
are validated against `schemas/journey.schema.json`; the report is described by
`schemas/journeyreport.schema.json`.

## The journey document

```yaml
version: 1
name: "Checkout flow"
description: "Add to cart, checkout and purchase for acme.com"
base_url: "https://acme.com"     # resolves relative navigate targets

defaults:
  timeout_ms: 20000              # inherited by steps
  wait_ms: 300

session:
  viewport: { width: 1366, height: 768 }
  locale: en-US
  timezone: America/New_York
  user_agent: "…"
  ignore_https_errors: false
  capture_data_layer: true       # default true

steps: [ ... ]
```

A complete, runnable example is `examples/journey/checkout.yaml` (and the
identical `checkout.json`).

### `steps`

Each step performs one action and optionally asserts on the page state produced
by it.

```yaml
- action: navigate               # navigate | click | fill | wait
  description: Open the home page
  value: /                       # URL for navigate, or input value for fill
  timeout_ms: 20000              # overrides defaults.timeout_ms
  wait_ms: 300                   # extra settle delay before capture
  expect: { ... }
```

| Action | Required | Behaviour |
| --- | --- | --- |
| `navigate` | `value` | Navigates to `value` (relative to `base_url` when set). |
| `click` | `selector` | Clicks the element. |
| `fill` (alias `type`) | `selector`, `value` | Sets the input's value. |
| `wait` | `selector` *or* `timeout_ms` | Waits for the selector, or a fixed delay. |

### `expect`

All provided fields must hold; an empty `expect` means "the action must not
error". Each field becomes a named check in the report.

| Field | Passes when |
| --- | --- |
| `url_contains` | the final URL contains the substring. |
| `title_contains` | the page title contains the substring. |
| `status` | the last navigation status equals the value. |
| `tags` | every listed tag id is detected on the page. |
| `absent_tags` | none of the listed tag ids is detected. |
| `events` | every data-layer `event` has fired by this step. |
| `cookies` | every named cookie is present. |
| `absent_cookies` | none of the named cookies are present. |

Tag ids are those produced by the scanner detector
(`internal/scanner/tags.go`), e.g. `google-tag-manager`, `meta-pixel`,
`google-analytics-4`. Data-layer events are the top-level `event` field of
captured `window.dataLayer` / `digitalData` payloads.

## The report

```jsonc
{
  "schemaVersion": "1.0.0",
  "journey": "Checkout flow",
  "startedAt": "2026-01-01T00:00:00Z",
  "finishedAt": "2026-01-01T00:00:12Z",
  "durationMs": 12000,
  "passed": false,
  "summary": { "steps": 7, "passed": 6, "failed": 1 },
  "steps": [
    {
      "index": 5,
      "action": "click",
      "selector": "button#place-order",
      "status": "fail",                 // pass | fail | error
      "message": "expected data-layer event \"purchase\" did not fire",
      "durationMs": 412,
      "checks": [
        { "name": "event:purchase", "status": "fail",
          "message": "expected data-layer event \"purchase\" did not fire" },
        { "name": "url_contains", "status": "pass",
          "expected": "/order-confirmation", "actual": "https://acme.com/cart" }
      ],
      "diff": {
        "newRequests": [ ... ],         // since the previous step
        "newConsole":  [ ... ],
        "newCookies":  [ ... ],
        "newTags":     [ ... ],
        "newEvents":   [ "add_to_cart" ]
      },
      "screenshot": "shots/step-06-click.png"
    }
  ]
}
```

- `status` is `pass`, `fail` (a check failed) or `error` (the action itself
  threw, e.g. a timeout). An `error` stops the run — the session is no longer in
  a known-good state.
- `diff` reports only what is **new** relative to the previous step, so a
  reviewer can see the effect of each action: which requests, cookies, tags and
  data-layer events the click or navigation introduced.
- `screenshot` is written only on failure, to `RunOptions.ScreenshotDir` (or a
  `pageguard-journey` directory under the OS temp dir).

### Formatters

```go
_ = (journey.JSONFormatter{}).Format(w, report) // indented JSON
_ = (journey.TextFormatter{}).Format(w, report) // human-readable summary
```

## Architecture

```
Journey (YAML/JSON)
      │
      ▼
  Engine.Run ──► SessionFactory.Open ──> Session (one browser context)
     │                                        │
     │  for each step:                        │
     │    Session.Perform(step) ──────────────┘
     │      └─ PageState{URL, status, HTML, requests, cookies, console, dataLayer}
     │           ├─ scanner.Detector().Detect → tags
     │           ├─ dataLayerEvents           → events
     │           ├─ checkExpectations         → []CheckResult
     │           └─ diffState(prev, cur)      → Diff
     └─ Report (summary + steps) ── Formatter
```

- `Session` / `SessionFactory` (`session.go`) are the seam between the engine and
  a browser; `PlaywrightSessionFactory` (`playwright.go`) drives the Node worker
  over newline-delimited JSON, and `FakeSession` backs the tests.
- `Engine.Run` (`engine.go`) applies defaults, opens one session, runs the steps
  in order, and stops early on an action error.
- `Report` and the formatters live in `report.go`.

### Worker protocol

`tools/browser-renderer` gains a stateful session protocol alongside its
existing one-shot render. A request without a `cmd` field is a legacy render;
with one it is a session command:

```
{"cmd":"open","opts":{...}}   -> {"ok":true}
{"cmd":"step","step":{...}}   -> {"ok":true,"state":{...}}
{"cmd":"screenshot"}          -> {"ok":true,"data":"<base64 png>"}
{"cmd":"close"}               -> {"ok":true}
```

The worker keeps requests, console messages and the browser context alive across
steps. The Go side slices the cumulative request/console lists to compute each
step's diff.

## Wiring

Like `tagpolicy`, the journey engine is intentionally decoupled from run
execution. Remaining Module 4 work to make journeys first-class:

1. Migration `0002`: add `journey_steps` and `journey_runs` (the `journeys`
   table already stores a journey's `steps` as `jsonb`).
2. Persist step results and stream them over the existing SSE/WS endpoints.
3. API: `POST /api/v1/journeys`, `POST /api/v1/journeys/{id}/runs`.

## Testing

`go test ./internal/journey` covers YAML/JSON parsing and equivalence, strict
decoding, defaults and validation, relative-URL resolution, pass/fail/error
execution, per-step diffs, data-layer event extraction (including the worker's
string-encoded payloads), screenshots on failure, the formatters, the worker
protocol against `testdata/fake-worker.mjs` (skipped when `node` is absent), and
the struct ↔ schema lockstep. See `docs/test-plan.md`.
