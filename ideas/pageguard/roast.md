# Roast: For mid-market ecommerce and martech teams with 500-5,000 pages and weekly releases, Pageguard is a self-hosted, real-browser audit that proves which tags, cookies, and beacons actually fired against policy, instead of an enterprise ObservePoint/Tag Inspector contract or a manual QA checklist.

## Kill Shots

- **There is no license, so "open source" is legally false.** `README.md:278` says "MIT." The marketing site repeats "MIT licensed" at least six times. There is no `LICENSE` file, and GitHub reports no license. Under default copyright, a company cannot legally copy, modify, or deploy this. The entire acquisition strategy dies in legal review before a buyer ever runs a scan.

- **There is no demand engine and no revenue path.** The repository was created 2026-10-23 and has 0 stars, 0 forks, and 0 releases. There is no signup, login, billing, usage metering, or self-serve cloud flow in `web/` or `internal/`. The only call to action is `git clone` on a Netlify page that still links to the old `bhkbdbhatt/openobserve` repository. Open source without a community is a hosting bill, not a business.

- **The detector catalogue is too small to be trusted.** Pageguard ships 15 built-in signatures (`internal/scanner/tags.go:102`) and 4 martech vendors (`internal/martech/martech.go:22`). Tag Inspector advertises 2,700+ categorized tags; OneTrust claims 45 million pre-categorized cookies. Pageguard will miss most real stacks. Every miss is a false negative, and one false negative in a compliance audit destroys the product's only asset: trust.

- **There is no retention loop.** A finding has severity and evidence but no owner, status, or verification (`internal/domain/observation.go:51`). Alerts are single-attempt, with no retry, backoff, or dedup. Email delivery is not implemented. Auth middleware is defined but never wired into the binary (`internal/api/api.go:40`; no `WithAuth(...)` call in `cmd/pageguard/main.go`). A customer scans once, fixes the issue, and has no scheduled reason to come back.

## What Survived

- **The problem is real and already has a price umbrella.** ObservePoint's published entry tier is roughly $599/month for 4,000 page scans. Tag Inspector is quote-only. OneTrust has a reported ~$10,000/year floor. There is room for a lower-cost, self-hosted or low-friction cloud alternative. The pain is not invented.

- **The technical foundation is real, not a mockup.** `go test ./...` passes across the API, orchestrator, scanner, plugin, privacy, journey, rules, stores, and WebSocket packages. The repository has resumable scans, browser rendering, PostgreSQL persistence, 90 API route patterns, 13 dashboard pages, and a working privacy rescanner wired to Playwright. This is more than a landing page with a GitHub link.

- **"Self-hosted, no per-page pricing" is a genuine wedge.** Every direct competitor is SaaS-only, quote-gated, or both. Self-hosting is attractive to privacy-sensitive buyers and agencies. The existing schedules, alerts, reports, policy engine, and plugin SDK are the right raw materials for a recurring product.

## Hand-Waved

- **The buyer.** "Martech, privacy, and engineering" is three buyers and no budget owner. The repository never names the economic buyer, the daily user, or the person who signs.
- **Willingness to pay.** No design partners, no letters of intent, no pre-sales, no pricing experiment, no waitlist. UNANSWERED.
- **Accuracy.** No benchmark corpus, no false-positive rate, no false-negative rate, no vendor coverage target. UNANSWERED.
- **Security and compliance readiness.** `SECURITY.md` is an unedited template. Auth is never enabled in the shipped binary. There is no CI, no pen test, no SOC 2, no data-processing agreement, and no legal review of the GDPR/CCPA/LGPD/ePrivacy claims. UNANSWERED.
- **Support model.** The plugin SDK asks customers to write validators. Who supports those plugins when they break at 3 AM? UNANSWERED.
- **Distribution.** No release, no hosted demo with signup, no content engine, no community, no agency channel. UNANSWERED.
- **Retention.** There are no customers, so there is no retention data. The repository has schedules and alerts, but no evidence anyone uses them repeatedly. UNANSWERED.
- **Licensing and business model.** MIT? Open-core? Hosted-only? Support contracts? The repo says MIT; the marketing site says free forever; the roadmap implies a commercial platform. UNANSWERED.

## Riskiest Assumption

- **Mid-market teams will pay for recurring audit evidence and a verified fix trail, not just a free one-off scan.**
- **Cheapest test:** Recruit 5 design partners from mid-market ecommerce or agencies. Charge $500 for a 30-day concierge pilot: baseline scan, policy setup, weekly scheduled scans, Slack alerts, and fix verification. Pass if 5/5 finish onboarding, at least 3 run three or more scheduled audits, at least 2 sign a $199-$499/month letter of intent, and at least 1 real violation is verified fixed. If fewer than 2 would pay, stop.

## Verdict

**RESHAPE**

The problem and the engineering are real, but the current product is an unlicensed, unlaunched, unretained tool with a catalogue too small to trust. Reshape it into a narrow, hosted, retention-first audit loop for one ICP, with self-hosting as the distribution channel rather than the business model.
