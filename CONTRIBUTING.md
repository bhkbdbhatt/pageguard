# Contributing to Pageguard

Thanks for helping improve Pageguard. This project is pre-1.0, so small,
well-tested changes are easier to review than large rewrites.

## Development Setup

Requirements:

- Go 1.25+
- Node.js 20+ (dashboard, plugin SDK, and browser renderer)
- Docker (optional, for PostgreSQL and container testing)

```bash
go build ./...
go test ./...
cd web && npm install && npm run build
```

The default test suite uses in-memory stores and in-process HTTP servers. No
database or browser is required. PostgreSQL tests are opt-in:

```bash
PAGEGUARD_TEST_DATABASE_URL='postgres://pageguard:pageguard@localhost:5432/pageguard?sslmode=disable' \
  go test ./internal/store/postgres
```

## Before Opening a Pull Request

1. Keep the change focused on one problem.
2. Add or update tests for behavior changes.
3. Run `gofmt`, `go vet ./...`, and `go test ./...`.
4. Run `npm run build` in `web/` if dashboard code changed.
5. Update docs when behavior, configuration, or API contracts change.
6. Do not commit secrets, customer data, or generated build output.

## Commit Style

Use short imperative subjects, for example:

```text
add finding triage status
fix postgres finding scan
wire api key auth on startup
```

## Reporting Bugs

Include the Pageguard version or commit, deployment mode (memory or
PostgreSQL), browser renderer settings, reproduction steps, and relevant logs.
Redact API keys, customer URLs, and cookie values.

## Security

Do not open a public issue for vulnerabilities. Follow `SECURITY.md`.
