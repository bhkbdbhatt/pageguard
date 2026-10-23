# Pageguard Test Plan

## 1. Philosophy

The Go core is **stdlib-first**: the only third-party dependencies are
`jackc/pgx` (isolated in `internal/store/postgres`) and `gopkg.in/yaml.v3`
(isolated in `internal/tagpolicy`), so the entire suite runs offline with
`go test ./...` — no Postgres, Docker, or browser required. Network behaviour is
exercised against `net/http/httptest` servers; browser behaviour against
`scanner.FakeRenderer`; plugin behaviour against both the real Node interpreter
(skipped when absent) and in-process fakes.

## 2. Running the suite

```bash
go test ./...                         # everything
go test ./internal/scanner/...        # one package
go test -race ./...                   # race detector (recommended in CI)
go test -cover ./...                  # coverage summary
go test -run TestManager ./internal/orchestrator -v
```

CI should run `go build ./...`, `go vet ./...`, `go test -race ./...`.

## 3. Test layers

| Layer | Scope | Example |
| --- | --- | --- |
| Unit | One function/type | `robots.ParseRobots`, `evalPredicate` |
| Component | One package via its public API | `Scanner.Scan`, `Manager.Execute` |
| Integration | Multiple packages wired together | `plugin.LoadDir` → `NodeValidator` |
| End-to-end (in-process) | Full HTTP surface | `api` test: REST + SSE + WS |

## 4. Coverage matrix

| Package | Test file | Focus |
| --- | --- | --- |
| `internal/scanner` | `htmlx_test.go` | link extraction, relative URLs, scheme filtering, title/meta, SPA heuristic |
| `internal/scanner` | `robots_test.go` | groups, UA selection, wildcards, `$` anchor, Allow override, crawl-delay |
| `internal/scanner` | `scanner_test.go` | observation assembly, tag detection, cookie hashing/flags, `http`/`browser`/`auto` modes, error propagation |
| `internal/orchestrator` | `manager_test.go` | BFS crawl, scope (same-origin/robots/globs), `maxPages`, fatal config, findings emission |
| `internal/plugin` | `ruleengine_test.go` | every field/operator, `all`/`any`/`none` semantics, coercion |
| `internal/plugin` | `host_test.go` | registry, findings conversion, plugin isolation, rule-set filtering |
| `internal/plugin` | `loader_test.go` | manifest loading, declarative + Node registration, **Node e2e** (skips if `node` absent) |
| `internal/tagpolicy` | `load_test.go` | YAML/JSON equivalence, strict decoding, defaults, invalid policies, size cap |
| `internal/tagpolicy` | `datalayer_test.go` | path tokenising/resolution, event extraction |
| `internal/tagpolicy` | `engine_test.go` | `must_exist`/`must_not_exist`, cookie attributes, value rules + events, duplicates, orphans, unexpected cookies, consent, summary + findings |
| `internal/tagpolicy` | `schema_test.go` | policy + report struct ↔ JSON-schema lockstep |
| `internal/journey` | `load_test.go` | YAML/JSON equivalence, strict decoding, size cap, example fixtures |
| `internal/journey` | `journey_test.go` | defaults, validation, relative-URL resolution, capture-data-layer override |
| `internal/journey` | `engine_test.go` | pass/fail/error execution, per-step diffs, data-layer events, screenshots, expectation checks |
| `internal/journey` | `report_test.go` | summary finalization, JSON + text formatters |
| `internal/journey` | `playwright_test.go` | worker protocol against `testdata/fake-worker.mjs` (skips if `node` absent) |
| `internal/journey` | `schema_test.go` | journey + report struct ↔ JSON-schema lockstep |
| `internal/privacy` | `load_test.go` | YAML/JSON equivalence, strict decoding, defaults, invalid regexes, size cap, example fixtures |
| `internal/privacy` | `simulate_test.go` | `FakeRescanner`/`RescanFunc`, `ScanResult` adapter |
| `internal/privacy` | `engine_test.go` | consent-state simulation, per-regulation checks (GDPR/CCPA/ePrivacy), GPC, DPIA + cross-border flags, scoring, regulation selection, request/essential-host classification |
| `internal/privacy` | `report_test.go` | violations → findings, finalization/scoring, JSON + text formatters |
| `internal/privacy` | `schema_test.go` | policy + report struct ↔ JSON-schema lockstep |
| `internal/events` | `hub_test.go` | fan-out, unsubscribe, dropped-count |
| `internal/store/memory` | `memory_test.go` | CRUD, filters, sorting, `ErrNotFound` |
| `internal/ws` | `ws_test.go` | handshake, accept hash, text frames, ping/pong, close |
| `internal/api` | `api_test.go` | REST lifecycle, validation errors, SSE events, WS stream |

## 5. Fixtures

Fixtures are constructed in-process; there are **no golden files to drift**.

| Fixture | Location | Purpose |
| --- | --- | --- |
| `siteHandler` | `orchestrator/manager_test.go` | Graph `/` → `/a`, `/b` (404), `/private`, external; serves `robots.txt` disallowing `/private` |
| SPA shell | `scanner/scanner_test.go` | `<div id="root">` + bundle to exercise `RenderAuto` upgrade |
| `FakeRenderer` | `scanner/renderer.go` | Deterministic rendered DOM, requests and console output |
| Cookie fixture | `scanner/scanner_test.go` | `sid=topsecret; Secure; HttpOnly; SameSite=Lax` → asserts hash-only storage |
| Node plugin | `plugins/example-node-validator` | Real stdio round-trip through the loader/host |
| Declarative plugin | `plugins/example-tag-rules` | Rule manifest parsing + evaluation |
| Tag policies | `examples/tagpolicy/acme.yaml` + `acme.json` | YAML/JSON policies must parse to the same document (equivalence test) |
| Journeys | `examples/journey/checkout.yaml` + `checkout.json` | YAML/JSON journeys must parse to the same document (equivalence test) |
| Journey worker | `internal/journey/testdata/fake-worker.mjs` | Deterministic session worker for the journey protocol test |
| Privacy policies | `examples/privacy/acme.yaml` + `acme.json` | YAML/JSON policies must parse to the same document (equivalence test) |
| Privacy observations | `examples/privacy/observations.json` | Per-consent-state page fixtures consumed by `cmd/privacy` |

### Assertion conventions

- Exact-set assertions for crawl coverage (no hidden extra pages).
- The cookie test asserts the raw value `"topsecret"` is **absent** everywhere
  and that `valueHash` is a 16-char hex string.
- The Node e2e test calls `exec.LookPath("node")` and `t.Skip`s when missing, so
  Windows-only or Node-less environments stay green.

## 6. Adding tests

1. Put tests next to the code (`xxx_test.go`) in the same package for internals,
   or `_test` package when asserting only the public surface.
2. Prefer `httptest.Server` over mocks for anything HTTP.
3. New rule fields/operators in the engine **must** add a table row in
   `ruleengine_test.go`. New policy fields in `internal/tagpolicy` **must** add a
   case in `engine_test.go` and (for new keys) a property in
   `schemas/tagpolicy.schema.json` so the lockstep test keeps passing.
4. New crawl behaviours must extend `manager_test.go`'s fixture graph.
5. Keep tests hermetic: no network, no sleeps longer than a few ms, no shared
   global state.

## 7. Not yet covered (roadmap)

- Browser renderer protocol contract test (`tools/browser-renderer`).
- Dashboard component tests (React Testing Library) once pages stabilize.
- TypeScript SDK unit tests (`node --test`) for `serveValidator`.
- Load/soak tests for the orchestrator under `-race` at high concurrency.

The PostgreSQL store has a full round-trip integration test in
`internal/store/postgres/postgres_test.go`, skipped by default and enabled with
`PAGEGUARD_TEST_DATABASE_URL` (see the README).
