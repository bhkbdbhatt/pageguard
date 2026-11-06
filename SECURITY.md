# Security Policy

## Supported Versions

Pageguard is currently pre-1.0. Security fixes are applied to the latest commit
on `main`.

| Version | Supported |
| --- | --- |
| `main` | Yes |
| Older commits/tags | No |

## Reporting a Vulnerability

Please do not open a public issue for a suspected vulnerability.

Open a private security advisory at
`https://github.com/bhkbdbhatt/pageguard/security/advisories/new` with:

- a description of the issue and its impact;
- steps to reproduce or a proof of concept;
- affected version or commit;
- any suggested mitigation.

You can expect an acknowledgement within 3 business days. We will validate the
report, agree on a disclosure timeline, and credit reporters who want to be
named. Please give us a reasonable window to ship a fix before publishing
details.

## Scope

In scope:

- the Go API, scanner, orchestrator, stores, and authentication middleware;
- the Next.js dashboard;
- the browser-renderer worker and plugin host;
- deployment manifests and container images in this repository.

Out of scope:

- vulnerabilities in third-party services or dependencies that are already
  fixed upstream;
- reports requiring access to a target site without authorization;
- social engineering, spam, or denial-of-service tests against production
  infrastructure.

## Security Model

- API keys are bearer tokens (`pg_` + 64 hex characters). Only their SHA-256
  digest is stored.
- Authentication is optional for local evaluation and should be enabled for any
  shared or internet-facing deployment with `PAGEGUARD_AUTH=true`.
- Bootstrap an initial admin key with `PAGEGUARD_ADMIN_KEY`; rotate it after
  creating scoped keys.
- Raw cookie values are hashed at the edge and are not persisted.
