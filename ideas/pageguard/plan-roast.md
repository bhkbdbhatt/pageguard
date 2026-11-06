# Plan: Pageguard retention-first build-up

**Date:** 2026-11-06
**Carries forward:** RESHAPE verdict from `ideas/pageguard/roast.md`
**Core thesis:** Open source is the distribution channel, not the business. The product is a recurring proof loop: audit -> alert -> assign -> fix -> verify -> report. One ICP, one loop, one paid plan before adding breadth.

---

## 1. Narrow the ICP

Do not sell to "martech, privacy, and engineering." Pick the buyer who feels the pain weekly and can approve a $149-$499/month card.

| Segment | Definition | Why they buy | Why they churn |
| --- | --- | --- | --- |
| **Primary: mid-market ecommerce / DTC** | 500-5,000 pages, weekly or faster releases, GTM + GA4 + Meta/Google Ads, 1-3 people own analytics, $20M-$200M revenue | Broken purchase events, duplicate pixels, pre-consent tags, and lost attribution cost real revenue | No one owns the fix; audit becomes another dashboard nobody opens |
| **Secondary: agencies** | 10-50 client sites, analytics or CRO practice | Recurring audit deliverable, white-label reports, multi-client workspace | Lower ACV, higher churn, demands white-label early |
| **Anti-ICP** | Under 100 pages, no tag manager, no release cadence, enterprise requiring SOC 2 on day one | Too little drift to justify a subscription | High support, low willingness to pay |

**Positioning:** "The open-source ObservePoint alternative for mid-market ecommerce. Weekly tag and consent drift alerts, with a verified fix trail, without a $25k enterprise contract."

**Do not say:** "web governance platform." That is a category, not a purchase. Say "tag drift alerts" and "consent leak evidence."

---

## 2. Define retention before building retention

A signup is not a customer. A scan is not retention.

**North star metric:** verified audits per active site per week.

**Active account:** at least one completed scan in the last 14 days **and** one alert, digest, or report consumed in the last 30 days.

| Metric | Definition | Day-30 target | Day-90 target |
| --- | --- | --- | --- |
| Activation | First scan + first policy + first alert channel within 24 hours | 60% | 65% |
| Time to first value | Signup to first finding with evidence | p50 under 10 minutes | p50 under 5 minutes |
| Week-4 retention | Active in week 4 after activation | 60% | 65% |
| Week-12 retention | Active in week 12 after activation | 50% | 55% |
| 90-day logo retention | Paying logos still paying at day 90 | 70% (design partners) | 80% |
| Alert-to-return | Alert recipient opens the dashboard or report within 48 hours | 40% | 50% |
| Findings triaged | Findings with owner or status set within 7 days | 40% | 50% |
| Verified fixes | Findings moved to fixed and re-checked by a later scan | 4 per account per month | 6 per account per month |
| Scans per active site | Completed scans per active site per week | 2.5 | 3.0 |
| False-positive rate | Findings a customer marks invalid / total findings | under 5% | under 2% |
| Gross revenue retention | Revenue retained excluding expansion | 85% | 85% |
| Net revenue retention | Revenue including site/scan expansion | 100% | 110% |
| Gross margin | Revenue minus infrastructure and support | 70% | 75% |
| CAC payback | Months to recover acquisition cost | under 6 months | under 4 months |

If week-4 retention is under 40% after three cohorts, the problem is not the funnel. The product is not returning value. Change the ICP or the loop.

---

## 3. The five retention loops

| Loop | Trigger | Product surface | Retention metric |
| --- | --- | --- | --- |
| **1. First value** | New signup or self-host install | URL -> baseline scan -> policy pack -> first finding with evidence -> first report | Activation, time to first value |
| **2. Weekly drift** | Cron schedule | Scan -> diff vs baseline -> new / resolved / regressed -> Slack or email digest | Scans per active site, alert-to-return |
| **3. Fix verification** | Finding assigned | Owner, status, issue link, notes -> next scan verifies -> "verified fixed" badge and audit log | Findings triaged, verified fixes |
| **4. Monthly evidence** | Calendar or report schedule | Governance report with trend, policy version, consent evidence, and audit trail | Report exports, renewal justification |
| **5. Expansion** | New site, user, or client | Add site, add user, add policy pack, agency client workspace | NRR, sites per account |

### Loop details

**First value (day 0)**
- One URL field. No project setup wizard before the first scan.
- Prebuilt policy packs: GA4, GTM, Meta Pixel, Google Ads, Adobe Analytics, OneTrust/Cookiebot consent.
- The "aha" is a real finding with a request URL, page URL, cookie name, and timestamp. Not a health score.
- Deliver a first report even if there are zero critical findings; zero findings is still evidence.

**Weekly drift (the habit)**
- Default schedule: daily for the top 100 pages, weekly for the rest. Do not make the user design a cron expression.
- Digest format: 5 lines, not a dashboard. New issues, resolved issues, regressions, consent changes, link to evidence.
- Alert dedup: same rule + page + fingerprint inside 24 hours updates the existing incident. No alert storms.
- Quiet hours and per-channel severity thresholds.

**Fix verification (the lock-in)**
- Every finding gets: owner, status (`open`, `acknowledged`, `fixed`, `regressed`, `ignored`), note, and optional Jira/Linear/GitHub issue link.
- The next scheduled scan re-checks the policy. A fixed finding becomes "verified fixed" only after the scan passes.
- Keep the historical trail. This is the artifact the buyer shows to legal, finance, or leadership.

**Monthly evidence (the budget justification)**
- One-page HTML/PDF: audits run, pages covered, findings fixed, regressions caught, consent violations, policy versions, and a signature/timestamp.
- Send to the buyer and the legal/privacy owner. This is what survives budget reviews.

**Expansion (the revenue)**
- Add a second site after the first site is healthy.
- Add seats for analytics, legal, and engineering.
- Agency workspaces: one dashboard per client, white-label report, consolidated billing.

---

## 4. What to build, in order

### Phase 0 - Legitimacy (days 0-14)

The repository cannot be evaluated until these are fixed.

- Add a real `LICENSE` file. MIT matches the marketing claim. Do it before accepting any contribution.
- Fix repository links: `bhkbdbhatt/openobserve` -> `bhkbdbhatt/pageguard` in README, marketing site, Helm chart, and package metadata.
- Replace the unedited `SECURITY.md` template with a real reporting process and supported versions.
- Add `CONTRIBUTING.md`, issue templates, and a code of conduct link that is accurate.
- Add CI: `go build ./...`, `go vet ./...`, `go test -race ./...`, web build, Docker build.
- Publish `v0.1.0` with a Docker image and checksums. No release means no adoption.
- Deploy a hosted demo with a one-click sample scan. No signup required for the demo.
- Add PostHog or equivalent analytics for the activation funnel.
- Publish an honest pricing page: free self-host, cloud waitlist, and a comparison against ObservePoint and Tag Inspector.

**Gate:** 10 external users self-host from the README without asking for help; 5 design partners agree to a paid pilot; legal review passes.

### Phase 1 - Activation (days 15-42)

- Hosted cloud signup with GitHub OAuth or magic link.
- Organization, project, and site model. API keys already exist; expose them in the UI.
- Onboarding: URL -> scan -> policy pack -> alert channel -> first report. No empty dashboard.
- Top 50 vendor signatures for the ICP, not 2,700. Add an "unknown tag" review queue so misses become catalogue work, not silent failure.
- Slack and email alert delivery with retry, backoff, and dedup.
- Schedule UX with presets: daily, weekly, after deploy (webhook/CLI).
- Usage metering and scan quotas from day one, even if billing comes later.

**Gate:** 60% activation; time to first value under 10 minutes; 5 design partners each run three scans in 14 days.

### Phase 2 - Retention (days 43-84)

- Finding workflow: owner, status, note, issue link, verified-fix state.
- Run-to-run diff: new, resolved, regressed; baseline snapshots; drift timeline.
- Weekly digest and monthly evidence report.
- Policy-as-code: YAML policy in the customer's repo, `pageguard check` CLI, GitHub Action that fails a pull request on a violation.
- Jira, Linear, and GitHub Issues integrations.
- Multi-site dashboard and agency client workspaces.
- Health score and automated nudges for inactive accounts.

**Gate:** week-4 retention at 60%; 3 paying customers at $199+/month; 90-day logo retention at 70%.

### Phase 3 - Monetization (days 85-180)

- Stripe billing, 14-day trial, plan limits, and overage handling.
- Design-partner conversion to annual plans.
- Two case studies with numbers: findings caught, hours saved, revenue protected, audit evidence produced.
- Outbound motion: 50 qualified accounts per week, 10 discovery calls, 5 demos.
- Content and SEO: comparison pages, tag-drift teardowns, consent-leak writeups.
- Agency partner program with 20% recurring revenue share.

**Gate:** 10 paying accounts, $2,000 MRR, gross margin above 70%, CAC payback under 6 months.

### Phase 4 - Moat (months 7-12)

- Grow the vendor catalogue to 250+ signatures for the ICP, with confidence scoring and a false-positive feedback loop.
- Community plugin catalogue: signatures, policy packs, and integrations contributed by agencies.
- GTM/GA4/Adobe/consent-platform integrations that read configuration and compare it to production.
- Benchmark reports from anonymized scan data: "tag health by industry."
- Enterprise readiness only when a customer pays for it: SSO, audit log export, self-host support, SOC 2 roadmap.

**Gate:** $10,000 MRR, 30 paying accounts, 80% gross retention, 110% net revenue retention, 5 active agency partners.

---

## 5. Pricing

Anchor against ObservePoint's published entry tier (~$599/month for 4,000 page scans) and Tag Inspector's quote-only enterprise pricing. Do not anchor against free tools; free tools do not produce evidence or verification.

| Plan | Price | Included | Purpose |
| --- | --- | --- | --- |
| **Self-host** | Free | 1 site, 500 pages per scan, manual scans, community support, MIT license | Distribution and developer trust |
| **Cloud Starter** | $149/month | 2 sites, 5,000 page scans/month, daily schedules, Slack + email alerts, 30-day history, 2 users | First paid conversion |
| **Cloud Team** | $499/month | 10 sites, 30,000 page scans/month, 12-month history, 10 users, Jira/CI integration, monthly evidence report | Core retention plan |
| **Agency** | $1,499/month | 50 sites, client workspaces, white-label reports, 20 users, consolidated billing | Multi-client channel |
| **Enterprise** | Custom | SSO, audit log export, self-host support, custom scan volume, security review | Only when demanded |

**Unit economics check:** 5,000 scans at roughly 4 seconds of browser compute is about 5.6 CPU-hours. At $0.06/vCPU-hour that is under $1 of raw compute; budget $20-$40 per account for rendering overhead, storage, and email. At $149, gross margin is roughly 73%-87% before payment fees. At $499, above 90%. Keep support under 30 minutes per account per month or the Starter plan is not viable.

**Do not offer unlimited scans.** It attracts the wrong customer, creates abuse, and makes costs unforecastable.

---

## 6. Ninety-day week-by-week

| Week | Product | Go-to-market | Gate / metric |
| --- | --- | --- | --- |
| 1 | LICENSE, repo links, SECURITY, CI, release workflow | 20 customer interviews; 6 pilot candidates | Legal review unblocked |
| 2 | Hosted demo, analytics, pricing page, waitlist | 5 paid pilot offers ($500/30 days) | 5 pilots accepted |
| 3 | Onboarding wizard, 5 policy packs | Pilot kickoff calls; baseline scans | 5 baselines complete |
| 4 | Slack/email alerts with retry and dedup; schedule presets | First weekly digest sent manually if needed | 3 partners run 3 scans |
| 5 | Finding status, owner, issue link | Pilot check-ins; collect false positives | 40% of findings triaged |
| 6 | Run-to-run diff, regressions, verified-fix state | Pilot review; first testimonial draft | 2 partners say they would pay |
| 7 | Weekly digest automation, monthly evidence report | Publish comparison page and demo video | 50% alert-to-return |
| 8 | Policy-as-code CLI and GitHub Action | First paid conversion; outbound list of 100 | 3 paying accounts |
| 9 | Stripe, trial, plan limits, agency workspace | Launch on Show HN, Reddit, LinkedIn | 10 trial signups |
| 10 | Top 50 vendors + unknown-tag triage | 2 teardown posts; agency outreach | False-positive rate under 5% |
| 11 | Health score, nudges, churn instrumentation | 50 outbound/week; case study interviews | Week-4 retention at 60% |
| 12 | Go/no-go review; fix the weakest loop | Pricing review; annual plan offer | 10 paying or PARK SaaS |

---

## 7. Retention operations (the unglamorous part)

### Health score

```
health =
  30 * (completed scans in last 14 days >= 2)
+ 20 * (active alert channel in last 30 days)
+ 20 * (active schedule exists)
+ 15 * (findings triaged in last 30 days)
+ 15 * (report exported in last 30 days)
```

- **Green (70-100):** leave alone; ask for a case study or referral.
- **Yellow (40-69):** automated nudge plus a 15-minute check-in.
- **Red (0-39):** founder-led call within 48 hours. Find the missing loop: no schedule, no alert channel, no owner, or too many false positives.

### Lifecycle cadence

| Time | Action |
| --- | --- |
| Day 0 | Welcome, one URL, baseline scan, policy pack, alert channel |
| Day 1 | "Here is the first finding" email with evidence link |
| Day 3 | Check policy coverage and false positives |
| Day 7 | First weekly digest; offer a 15-minute review |
| Day 14 | Health check; if no schedule, create one with the customer |
| Day 30 | Quarterly business review: findings caught, fixes verified, coverage |
| Day 60 | Renewal value report; propose annual plan or site expansion |
| Day 90 | Renewal; churn interview if not renewing |

### Churn taxonomy

Every cancellation gets one reason: no time, no budget, too many false positives, missing integration, too complex, no recurring need, or competitor. Do not accept "not using it" as a reason. Ask what they did instead.

### Win-back

At 30, 60, and 90 days after churn, send a one-page diff: what changed on their site, what Pageguard would have caught, and a reactivation offer. This is cheap and produces real product insight.

### Support

- One business-day response for all plans.
- Four-hour response for critical alert delivery failures.
- Public status page.
- A searchable docs site with policy-pack recipes.
- Every support ticket tagged with a product gap. Weekly triage. No ticket disappears into a inbox.

---

## 8. Distribution build-up

### Open source

- Ship a real release and a Docker image. Add GitHub topics: `web-governance`, `tag-management`, `martech`, `privacy`, `playwright`, `observability`.
- README first screen: demo GIF, one-command quick start, hosted demo link, comparison table.
- `good first issue` labels for vendor signatures and policy packs.
- Monthly office hours and a Discord/Slack community.
- Plugin and policy-pack contest with a small prize; publish the winners.

### Content and SEO

Build pages for buyer-intent queries, not thought-leadership:

- "ObservePoint alternative"
- "Tag Inspector alternative"
- "tag audit tool"
- "GTM tag monitoring"
- "cookie consent audit"
- "pre-consent tag checker"
- "tag drift detection"
- "web governance open source"

Publish two pieces per week for 12 weeks. Each piece ends with a free one-URL scan. Teardown posts of real (consenting) sites outperform generic explainers.

### Agency channel

- Recruit 10 agencies that already run analytics or CRO for ecommerce clients.
- Offer 20% recurring revenue share, a client workspace, and white-label reports.
- Give them a free internal account and a co-branded audit template.
- Measure: activated client sites per agency, not signups per agency.

### Outbound

- 50 accounts per week, 10 discovery calls, 5 demos.
- Target titles: head of analytics, director of digital, growth engineering lead, ecommerce operations.
- Opening line: "We found three tags firing before consent on your checkout. Want the evidence?"
- Do not lead with "open source." Lead with the finding.

---

## 9. Experiments and kill criteria

| Experiment | Hypothesis | Pass | Fail | Cost |
| --- | --- | --- | --- | --- |
| 5 paid pilots | Mid-market will pay for recurring evidence | 2 of 5 sign a $199-$499/month LOI | Fewer than 2 | 2 weeks founder time |
| 200-page benchmark | Accuracy is good enough to sell | False-positive rate under 5% | Over 10% | 3 days |
| Comparison page | Buyers search for alternatives | 5% visitor-to-demo conversion | Under 1% | 1 day |
| 50 outbound accounts | The finding-led pitch works | 5 demos from 50 | 0-1 demos | 1 week |
| Agency pilot | Agencies can resell | 3 agencies activate 3 client sites each | 0-1 agencies | 2 weeks |
| Pricing test | $149 vs $299 vs $499 | At least 3 choose $299+ | All choose $149 or churn | 2 weeks |

**Kill or park criteria**

- License, security policy, CI, or release not fixed by day 14: stop. Do not market an unlicensed product.
- Fewer than 2 of 5 pilots would pay after 30 days: park the SaaS. Keep the open-source tool as a portfolio project.
- False-positive rate over 10% on the benchmark: do not sell. Fix catalogue and triage first.
- Week-4 retention under 40% after three cohorts: change ICP or change the loop. Do not add features.
- CAC over $1,000 with ACV under $2,000: abandon direct sales. Switch to agency channel or paid self-host support only.
- Support over 2 hours per customer per week: the scope is too broad. Cut non-ICP features.

---

## 10. What not to build yet

Do not touch these until the retention loop is proven:

- Journey testing beyond the existing engine.
- Multi-page privacy batches.
- PDF export and pixel-perfect reports.
- Mobile app SDK scanning.
- SSO, SCIM, and advanced RBAC.
- Plugin marketplace and paid plugins.
- A 2,700-tag catalogue built upfront. Build the top 50 for the ICP and an unknown-tag triage queue.
- AI summaries, chat, or "insights." The buyer wants evidence, not another layer of interpretation.
- Custom features for non-ICP customers.

---

## 11. Instrumentation

Track these events from day one:

```
signup_started
signup_completed
project_created
site_added
scan_started
scan_completed
scan_failed
first_finding_viewed
policy_pack_applied
alert_channel_added
schedule_created
finding_status_changed
finding_verified_fixed
report_exported
invite_sent
subscription_started
subscription_canceled
```

Weekly dashboard: activation rate, time to first value, week-4 retention by cohort, scans per active site, alert-to-return, findings triaged, verified fixes, false-positive rate, MRR, churn reasons.

---

## 12. Immediate 14-day checklist

- [ ] Add `LICENSE` (MIT) and fix every `openobserve` link.
- [ ] Replace `SECURITY.md`; add `CONTRIBUTING.md` and issue templates.
- [ ] Add CI: build, vet, race tests, web build, Docker build.
- [ ] Publish `v0.1.0` with a Docker image and checksums.
- [ ] Deploy a hosted demo with a sample scan.
- [ ] Add PostHog (or equivalent) and the event list above.
- [ ] Write 5 policy packs: GA4, GTM, Meta Pixel, Google Ads, OneTrust/Cookiebot.
- [ ] Add the top 50 vendor signatures for the ICP.
- [ ] Recruit 6 design partners and offer a $500/30-day pilot.
- [ ] Build finding status/owner and verified-fix state.
- [ ] Ship Slack + email alert delivery with retry and dedup.
- [ ] Publish an honest ObservePoint/Tag Inspector comparison page.
- [ ] Run 10 customer interviews; ask for the last tag incident, current process, budget, and what they would pay.
- [ ] Day-14 gate review: if the license, demo, and 5 pilots are not real, stop and fix them before writing more code.
