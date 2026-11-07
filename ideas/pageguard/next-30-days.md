# Pageguard: compressed 30-day execution cycle

**Date:** 2026-11-07
**Source:** `ideas/pageguard/plan-roast.md`
**Goal:** Convert the post-launch repo into a usable paid pilot loop in 30 days.
**Constraint:** Founder-led. No hosted multi-tenant SaaS, SSO, or billing before a design partner pays for the loop.

---

## 1. Where the repo stands

### Shipped and verified

- MIT `LICENSE`, real `SECURITY.md`, `CONTRIBUTING.md`.
- CI for Go format/build/vet/test and the Next.js production build.
- API-key authentication with a bootstrap admin key, scoped reader/writer keys, project isolation, and dashboard key entry.
- Finding triage: status, owner, note, timestamps, filters, PATCH endpoint, run-detail UI.
- Tag catalogue: 65 signatures across analytics, ads, CDPs, consent, session replay, and tag managers.
- Martech vendor map: 25 vendors for host-based detection and reporting.
- Schedules, alert rules, incidents, delivery history, webhook/Slack/PagerDuty/email-backed channels.
- Reports in JSON, CSV, and HTML; journey runner; privacy checks.
- `go build`, `go vet`, `go test ./...`, and `next build` pass locally.

### Not shipped (do not claim)

- Hosted signup, login, organizations, team invites, billing, plan limits.
- White-label agency workspaces.
- PDF report export.
- Alert retry/backoff and same-fingerprint dedup.
- Run-to-run diff with new/resolved/regressed and verified-fix state.
- Policy-as-code CLI and GitHub Action.
- SSO, SOC 2, audit-log export, enterprise support.

### Positioning

> The open-source ObservePoint alternative for mid-market ecommerce. Weekly tag and consent drift alerts, with an evidence trail and a verified fix trail, without a $25k enterprise contract.

Sell the loop, not the dashboard: **audit -> alert -> assign -> fix -> verify -> report**.

---

## 2. Days 1-14: make the first value undeniable

**Outcome gate:** 10 external installs from the README; 5 design partners accept a paid 30-day pilot; a new user reaches a real finding with evidence in under 10 minutes.

### Days 1-2: release credibility

- [ ] Bump version to `v0.1.0`; tag and publish release notes.
- [ ] Publish a multi-arch Docker image and checksums.
- [ ] Add a one-command compose file that starts API + Postgres + web.
- [ ] Replace remaining old branding in Helm, package metadata, screenshots, and social previews.
- [ ] Add issue templates and a short code-of-conduct policy.
- [ ] Add a `SECURITY.md` link to the GitHub private advisory form in the README.

**Done when:** a stranger can `docker compose up`, run a scan, and see a finding without reading source.

### Days 3-5: activation surface

- [ ] Add a no-signup hosted demo with 3 sample sites and preloaded evidence.
- [ ] Add product analytics events: `install`, `project_created`, `scan_started`, `scan_completed`, `finding_opened`, `finding_triaged`, `alert_channel_added`, `report_exported`, `api_key_set`.
- [ ] Add first-run empty states that point to one URL field and one policy pack.
- [ ] Add policy packs for GA4, GTM, Meta Pixel, Google Ads, and OneTrust/Cookiebot.
- [ ] Add an API-key screen that creates, lists, rotates, and revokes keys without curl.
- [ ] Add a "copy evidence" action on every finding.

**Done when:** median time from install to first evidence-bearing finding is under 10 minutes.

### Days 6-9: the habit loop

- [ ] Add schedule presets: daily top pages, weekly full site, after-deploy webhook.
- [ ] Add alert retry with exponential backoff and a delivery test button.
- [ ] Deduplicate incidents by rule + page + fingerprint within 24 hours.
- [ ] Add per-channel severity thresholds and quiet hours.
- [ ] Send a five-line digest: new, resolved, regressed, consent changes, evidence link.
- [ ] Add a `POST /api/v1/hooks/deploy` endpoint that starts a scan for a project token.

**Done when:** a partner can connect Slack and receive a real digest without founder help.

### Days 10-14: the fix loop

- [ ] Add run-to-run diff: new, resolved, regressed, unchanged.
- [ ] Add a "verified fixed" state that only flips after a later scan passes.
- [ ] Add an optional external issue URL field to findings.
- [ ] Add a weekly digest email template and unsubscribe handling.
- [ ] Add an activation checklist inside the dashboard: scan, policy, alert channel, triage.
- [ ] Run 5 pilot kickoff calls and baseline scans.

**Done when:** 3 partners have run 3 scans each and at least 40% of their findings have an owner or status.

### Days 1-14 go-to-market

- [ ] Send the Product Hunt follow-up sequence in `acquisition-kit.md`.
- [ ] Publish one teardown: "pre-consent Meta pixel on a real DTC checkout".
- [ ] Offer 5 paid pilots at `$500 / 30 days`, credited toward the first annual plan.
- [ ] Book 10 discovery calls from the existing waitlist and outbound list.
- [ ] Ask every pilot for one false positive and one missing vendor signature.

**Cut if behind:** API-key management UI can stay curl-only; demo can be a recorded walkthrough; email can be webhook-backed.

---

## 3. Days 15-30: turn the loop into a paid pilot

**Outcome gate:** 3 paying accounts; week-4 retention at 60% for pilot cohort; false-positive rate under 5%; 2 partners say they would pay full price.

### Days 15-18: evidence and reports

- [ ] Add report scheduling: weekly digest and monthly evidence report.
- [ ] Add policy version, scan coverage, findings fixed, regressions caught, and consent violations to the report.
- [ ] Add print-friendly HTML for PDF export from the browser.
- [ ] Add an audit trail of status changes and report exports.
- [ ] Add report download links to alert digests.

### Days 19-22: developer lock-in

- [ ] Ship `pageguard check` CLI.
- [ ] Ship a GitHub Action that fails a pull request on policy violations.
- [ ] Add policy-as-code YAML with `pageguard policy init`.
- [ ] Add a documented API quickstart with curl and a Postman collection.
- [ ] Add a "run after deploy" sample for GitHub Actions and Vercel.

### Days 23-26: commercial plumbing

- [ ] Add Stripe Checkout and a 14-day trial for Cloud Starter and Cloud Team.
- [ ] Enforce scan quotas, site limits, user limits, and history retention.
- [ ] Add usage meters and overage warnings.
- [ ] Add annual pricing and a design-partner conversion offer.
- [ ] Add a self-serve cancellation and data-export path.

### Days 27-30: prove the loop

- [ ] Interview every pilot; record the trigger, the owner, and the fix time.
- [ ] Write 2 case studies with permission: findings caught, hours saved, evidence produced.
- [ ] Publish a comparison page: Pageguard vs ObservePoint vs Tag Inspector.
- [ ] Run a 50-account outbound week and 10 discovery calls.
- [ ] Review pricing against the observed willingness to pay.
- [ ] Decide: continue SaaS, stay open-core, or park the hosted motion.

**Cut if behind:** Stripe can be invoices plus manual plan flags for the first 3 customers. Do not cut the report or the fix verification; those are the retention loop.

---

## 4. Daily operating cadence

| Time | Work |
| --- | --- |
| First 90 minutes | Ship one retention-loop improvement or one bug that blocks activation |
| Midday | 5 outbound emails, 2 follow-ups, 1 teardown research block |
| Afternoon | 2 customer calls or 1 pilot check-in |
| Last 30 minutes | Review activation funnel, false positives, and failed scans; write tomorrow's single priority |

**Weekly review questions**

1. How many new installs reached a first finding with evidence?
2. How many activated accounts added an alert channel?
3. How many findings were triaged, and how many were marked invalid?
4. How many partners would be upset if Pageguard disappeared?
5. What is the one loop with the worst drop-off?

---

## 5. Instrument from day one

Log these as events or CRM fields; do not rely on memory.

| Metric | Definition | 30-day target |
| --- | --- | --- |
| Activation | First scan + first policy + first alert channel within 24h | 60% |
| Time to first value | Signup/install to first evidence-bearing finding | p50 < 10 min |
| Weekly active sites | Sites with at least one completed scan in 7 days | 60% of pilots |
| Findings triaged | Findings with owner or status within 7 days | 40% |
| Verified fixes | Findings resolved then re-checked by a later scan | 4 per account |
| Alert-to-return | Alert recipient opens dashboard/report within 48h | 40% |
| False-positive rate | Findings marked invalid / total findings | < 5% |
| Pilot conversion | Paid pilots that convert to a plan | 60% |
| MRR | Recurring revenue | 3 x $149-$499 |

---

## 6. Non-goals for this cycle

- Multi-tenant hosted login and organizations.
- White-label agency workspace.
- 250+ vendor signatures.
- SSO, SOC 2, enterprise security questionnaires.
- A broad "web governance platform" rebrand.

If a request does not improve the `audit -> alert -> assign -> fix -> verify -> report` loop, it waits until after day 30.

---

## 7. Risk register

| Risk | Signal | Response |
| --- | --- | --- |
| Install friction | Users drop before first scan | Ship compose, demo, and a 60-second quickstart |
| No alert habit | Scans happen but no Slack/email channel | Make the channel part of onboarding, not settings |
| False positives | Partners distrust findings | Add invalid-finding feedback and review the top 5 misses weekly |
| No owner | Findings stay open | Add owner required on acknowledge; digest names the owner |
| Hosted scope creep | Time goes to login/billing | Invoice first 3 pilots; keep self-host as the primary path |
| Weak willingness to pay | Pilots use it but will not pay | Narrow ICP to ecommerce teams with weekly releases and revenue exposure |

---

## 8. Definition of done for day 30

- A new user installs, scans, configures an alert, triages a finding, and exports evidence without founder help.
- Three customers pay.
- A finding can move `open -> acknowledged -> in_progress -> resolved` and become `verified fixed` after a later scan.
- A weekly digest and a monthly evidence report reach the buyer.
- The dashboard shows the activation checklist and the health of each account.
- The README, pricing page, and outbound copy only claim shipped behavior.

See `ideas/pageguard/acquisition-kit.md` for the messages, sequences, demo script, and offer.
