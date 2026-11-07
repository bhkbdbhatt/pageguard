# Pageguard client acquisition kit

**Date:** 2026-11-07
**Use with:** `ideas/pageguard/next-30-days.md`
**Rule:** Never claim a feature that is not shipped. The current pitch is tag/consent drift evidence plus finding triage. Hosted signup, billing, white-label, PDF export, and verified-fix automation are roadmap items until they ship.

---

## 0. Fill these before sending anything

| Placeholder | Value |
| --- | --- |
| `{{PRODUCT_HUNT_URL}}` | Product Hunt listing URL |
| `{{SIGNUP_URL}}` | Demo/waitlist/install URL |
| `{{DEMO_URL}}` | Hosted demo URL |
| `{{CALENDAR_URL}}` | 15-minute call link |
| `{{FOUNDER_NAME}}` | Your name |
| `{{REPO_URL}}` | `https://github.com/bhkbdbhatt/pageguard` |
| `{{PRICE_STARTER}}` | `$149/month` |
| `{{PRICE_TEAM}}` | `$499/month` |
| `{{PILOT_PRICE}}` | `$500 for 30 days` |

---

## 1. Who to contact

### Primary: mid-market ecommerce / DTC

**Fit:** 500-5,000 pages, weekly or faster releases, GTM + GA4 + Meta/Google Ads, 1-3 people own analytics, roughly $20M-$200M revenue.

**Trigger events**

- A purchase or add-to-cart event broke after a release.
- A new pixel or consent banner was deployed.
- Attribution numbers dropped and engineering cannot reproduce it.
- Legal or privacy asked for pre-consent tag evidence.
- An agency handoff left no tag documentation.

**Titles:** Head of Analytics, Director of Ecommerce, Marketing Operations, Growth Engineering, Privacy Lead, CTO at smaller DTC brands.

### Secondary: agencies

**Fit:** 10-50 client sites, analytics/CRO practice, recurring reporting, no dedicated tag-governance tool.

**Titles:** Founder, Analytics Director, CRO Lead, Media Director.

### Anti-ICP

- Under 100 pages, no tag manager, no release cadence.
- Enterprise that requires SOC 2 or SSO before a pilot.
- Teams looking for a free scanner with no owner for fixes.

---

## 2. Offer

**Design-partner pilot**

> 30 days, one site, baseline scan, weekly drift digest, finding triage for your team, and a monthly evidence report. `{{PILOT_PRICE}}`, credited toward the first annual plan. I run the first scan with you and review the findings live.

**Why paid:** it filters for a real problem and gives both sides a reason to finish the loop.

**What the pilot must prove**

- A real finding with request URL, page URL, cookie/tag name, and timestamp.
- A named owner and a status change.
- A later scan that verifies the fix.
- A report the buyer can show to legal, finance, or leadership.

---

## 3. Message architecture

**Category:** tag and consent drift monitoring, not "web governance platform".

**One-liner**

> Pageguard catches broken tags, duplicate pixels, and pre-consent leaks before they cost revenue, then proves the fix with a re-scan.

**Problem paragraph**

> Your checkout changes every week. GA4, Meta, and consent rules change with it. Broken purchase events and pre-consent tags show up as attribution gaps and privacy risk, but nobody can reproduce them from a dashboard. Pageguard scans the live site, captures the evidence, and turns drift into an assigned, verifiable fix.

**Proof paragraph (honest)**

> Open source, MIT licensed. 65 built-in tag signatures, 25 vendor mappings, Playwright rendering, run history, alert channels, finding triage, and HTML/CSV/JSON evidence reports. Self-host in minutes; paid pilots get a hosted demo and founder-led onboarding.

**Do not say**

- "ObservePoint replacement" as a factual claim; say "an open-source alternative for mid-market teams".
- "SOC 2", "SSO", "white-label", "PDF report", "billing", or "multi-tenant" until shipped.
- "AI-powered" or "2,700 integrations".

---

## 4. Website copy blocks

### Hero

**Headline:** Catch tag drift before it costs you revenue.

**Subhead:** Pageguard scans your live site, captures broken beacons and pre-consent leaks, and gives your team an assigned, verifiable fix trail. Open source. Self-host in minutes.

**Primary CTA:** Run a sample audit
**Secondary CTA:** Start a 30-day pilot

### Three proof blocks

1. **Evidence, not a score.** Request URL, page URL, cookie name, timestamp, and screenshot for every finding.
2. **The fix loop.** Assign an owner, set a status, add a note, and re-scan to verify.
3. **Open source.** MIT licensed, plugin-based, and deployable with Docker or Postgres.

### Pricing page

| Plan | Price | Best for | Included |
| --- | --- | --- | --- |
| Self-host | Free | Developers evaluating the loop | 1 site, 500 pages per scan, manual scans, MIT license |
| Cloud Starter | `{{PRICE_STARTER}}` | First paid team | 2 sites, daily schedules, Slack/email alerts, 30-day history |
| Cloud Team | `{{PRICE_TEAM}}` | Core retention | 10 sites, longer history, more users, reports, integrations |
| Design partner | `{{PILOT_PRICE}}` | 5 early teams | 30 days, one site, founder onboarding, credited to annual |

Add this line below the table:

> Hosted signup and billing are rolling out. During the pilot, onboarding and invoicing are handled directly.

---

## 5. Product Hunt follow-up sequence

Send to everyone who engaged with `{{PRODUCT_HUNT_URL}}`. Keep it short. Stop after one reply or opt-out.

### Email 1: same day

**Subject:** Thanks for the Pageguard feedback

> Hi {{first_name}},
>
> Thanks for checking out Pageguard on Product Hunt. The most common question was "what does it actually catch?"
>
> Here is a 90-second sample audit of a DTC checkout: `{{DEMO_URL}}`
>
> If you have a site with weekly releases and GTM/GA4/Meta, I'm running 5 paid 30-day pilots. I do the first scan with you and review the findings live. Reply "pilot" and I'll send the details.
>
> {{FOUNDER_NAME}}

### Email 2: day 3

**Subject:** The one finding most teams miss

> Hi {{first_name}},
>
> The first scan usually surfaces one of three things: a purchase event firing before consent, a duplicate Meta pixel, or a tag that disappeared after a release.
>
> If you send me one URL, I'll tell you whether it is worth a deeper audit. No signup needed.
>
> {{FOUNDER_NAME}}

### Email 3: day 7

**Subject:** A 15-minute tag drift review?

> Hi {{first_name}},
>
> I'm offering 5 teams a 30-day design-partner pilot: baseline scan, weekly drift digest, finding triage, and an evidence report you can show to legal or leadership.
>
> `{{PILOT_PRICE}}`, credited toward the first annual plan. Want to see the sample report first? `{{CALENDAR_URL}}`
>
> {{FOUNDER_NAME}}

### Email 4: day 14

**Subject:** Close the loop on tag drift

> Hi {{first_name}},
>
> Last note from me. Pageguard is open source and self-hostable, so you can try the full loop without a call: `{{REPO_URL}}`
>
> If a broken purchase event or pre-consent tag is on your list this quarter, reply and I'll run the baseline.
>
> {{FOUNDER_NAME}}

---

## 6. Outbound: ecommerce analytics leads

### Email 1

**Subject:** Does your Meta pixel fire before consent?

> Hi {{first_name}},
>
> Quick question: after a release, how do you verify that GA4 purchase events, Meta pixels, and consent rules still fire in the right order?
>
> Most teams I talk to find out from an attribution drop or a privacy review. I built Pageguard to scan the live site, capture the evidence, and assign the fix.
>
> Worth 15 minutes to see if this matches your process? `{{CALENDAR_URL}}`
>
> {{FOUNDER_NAME}}

### Email 2: day 4

**Subject:** A real finding from a DTC checkout

> Hi {{first_name}},
>
> A sample scan found a Meta pixel firing before the consent banner resolved. The evidence included the request URL, page URL, cookie name, and timestamp, so the fix owner could reproduce it in minutes.
>
> That is the loop: audit, alert, assign, fix, re-scan, report. If you want the same baseline for {{company}}, reply with the storefront URL.
>
> {{FOUNDER_NAME}}

### Email 3: day 10

**Subject:** 30 days, one site, founder-led

> Hi {{first_name}},
>
> I'm taking 5 design partners for a 30-day pilot: baseline scan, weekly drift digest, triage for your team, and a monthly evidence report. `{{PILOT_PRICE}}`, credited to annual.
>
> If tag drift is not a priority this quarter, tell me and I'll close the loop. If it is, pick a time: `{{CALENDAR_URL}}`
>
> {{FOUNDER_NAME}}

---

## 7. Outbound: agencies

### Email 1

**Subject:** Recurring tag audits for your client sites

> Hi {{first_name}},
>
> Do your clients ask for tag or consent evidence between quarterly reviews? I built Pageguard so an agency can scan a client site, assign findings, and hand over an evidence report without a $25k enterprise contract.
>
> Open source, self-hostable, and I'm looking for 3 agencies to shape the multi-client workflow. Worth a look? `{{DEMO_URL}}`
>
> {{FOUNDER_NAME}}

### Email 2: day 5

**Subject:** White-label is roadmap, the audit loop is live

> Hi {{first_name}},
>
> To be direct: white-label reports and client workspaces are on the roadmap, not shipped. What is live is the audit-to-report loop, finding triage, and scheduled scans.
>
> I'm offering 3 agencies a 30-day pilot to pressure-test the workflow. If it saves your team reporting time, we can talk about a recurring revenue share.
>
> {{FOUNDER_NAME}}

### Email 3: day 12

**Subject:** One client site, one baseline

> Hi {{first_name}},
>
> Send me one client URL and I'll run the baseline before we talk. You will see the evidence format and decide whether it fits your reporting process.
>
> `{{CALENDAR_URL}}`
>
> {{FOUNDER_NAME}}

---

## 8. LinkedIn / X DM

> Saw {{company}} ships storefront changes weekly. Curious how you verify GA4/Meta events and consent order after a release. I built an open-source scanner that captures the evidence and assigns the fix. Happy to run one baseline, no pitch.

---

## 9. Discovery call script (15 minutes)

1. **Trigger (3 min):** "What changed recently that made tag or consent quality a priority?"
2. **Current process (3 min):** "How do you find out a purchase event broke? Who owns the fix? How do you prove it is fixed?"
3. **Cost (2 min):** "What happens when attribution is wrong for a week? Who notices first?"
4. **Fit (2 min):** "How many pages, how often do you release, and which tag manager and consent platform?"
5. **Demo (3 min):** Show one real finding, the triage state, and the re-scan evidence.
6. **Close (2 min):** "The pilot is 30 days, one site, `{{PILOT_PRICE}}`, credited to annual. I run the baseline with you. Does that fit this quarter?"

**Qualify out if:** no release cadence, no tag manager, no owner for fixes, or SSO/SOC 2 required before a pilot.

---

## 10. Demo script (10 minutes)

1. Start from a real site, not a fake dashboard.
2. Run a scan and show the live progress.
3. Open one finding: request URL, page URL, timestamp, cookie/tag name, screenshot.
4. Change the status to `acknowledged`, set an owner, add a note.
5. Show the status filter and the finding list.
6. Show the scheduled scan or alert channel.
7. Export the HTML/CSV evidence report.
8. Close with the next step: baseline for their site and a 30-day pilot.

**Do not tour every page.** The demo is the loop, not the navigation.

---

## 11. Objection handling

| Objection | Response |
| --- | --- |
| "We already have ObservePoint." | "Good. Most teams keep it for enterprise governance. Pageguard is for the weekly drift your team fixes between quarterly reviews. Run one baseline and compare the evidence." |
| "We can build this with a script." | "You can. The hard part is the evidence trail, owner, status, re-scan, and report that survives a budget review. That is what the pilot tests." |
| "Is it enterprise-ready?" | "No. It is open source and self-hostable, with API-key auth and project scoping. SSO and SOC 2 are customer-funded roadmap items." |
| "We need white-label." | "Not shipped. The audit-to-report loop is live. If the workflow fits, we can discuss agency packaging after the pilot." |
| "Too expensive." | "The pilot is `{{PILOT_PRICE}}` credited to annual. One broken purchase event or one privacy review usually costs more than that." |
| "We do not have time." | "I run the baseline and the first review. Your team only needs to name an owner for the findings." |
| "Send me information." | "I'll send the sample report and the repo. If it looks relevant, pick a 15-minute slot and I'll run your baseline." |

---

## 12. Content plan (2 posts per week)

| Week | Post | CTA |
| --- | --- | --- |
| 1 | Teardown: pre-consent Meta pixel on a DTC checkout | Run a sample audit |
| 1 | "Why tag audits fail after the report" | Start a pilot |
| 2 | GA4 purchase event drift after a release | Send one URL |
| 2 | Consent banner order: what evidence legal needs | Download sample report |
| 3 | Duplicate pixel detection without a tag manager | Try the demo |
| 3 | Open-source vs enterprise tag governance | Compare page |
| 4 | Case study: findings caught and verified fixed | Book a call |
| 4 | Agency workflow: one baseline per client | Agency pilot |

Every post must include one real screenshot or evidence artifact. No generic stock images.

---

## 13. Weekly scorecard

Track these every Friday. Use a spreadsheet until CRM exists.

| Metric | Target |
| --- | --- |
| New conversations started | 20/week |
| Replies | 5/week |
| Discovery calls booked | 5/week |
| Demos completed | 3/week |
| Pilots offered | 2/week |
| Pilots accepted | 1/week |
| Activation (first finding) | 60% |
| Findings triaged in pilot | 40% |
| False positives reported | < 5% |
| Paid conversions | 3 by day 30 |

**CRM fields:** company, segment, pages, release cadence, tag manager, consent platform, trigger event, owner, next step, pilot start, baseline URL, findings caught, verified fixes, objection, outcome.

---

## 14. Claims guardrails

Before publishing anything, verify the claim against the repo.

- Write "HTML/CSV/JSON report", not "PDF report".
- Write "webhook/Slack/PagerDuty and email-backed alerts", not "email service".
- Write "API-key auth and project scoping", not "SSO".
- Write "65 built-in signatures", not "thousands of integrations".
- Write "open-source alternative", not "ObservePoint replacement".
- Do not publish customer names, findings, or numbers without written permission.
- Do not promise verified-fix automation until run-to-run diff ships.

---

## 15. First 10 prospects worksheet

| # | Company | Segment | Contact | Trigger | First touch | Reply | Call | Pilot | Outcome |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | | | | | | | | | |
| 2 | | | | | | | | | |
| 3 | | | | | | | | | |
| 4 | | | | | | | | | |
| 5 | | | | | | | | | |
| 6 | | | | | | | | | |
| 7 | | | | | | | | | |
| 8 | | | | | | | | | |
| 9 | | | | | | | | | |
| 10 | | | | | | | | | |

Fill this before the next outbound block. No list, no outreach.
