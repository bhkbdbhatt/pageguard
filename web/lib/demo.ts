import type { IconName } from "../components/icons";
import type { Cookie, NetworkRequest, RunStatus, Severity, Tag } from "./api";

/** Fixed clock so demo labels never cause server/client hydration mismatches. */
export const DEMO_NOW = "2026-10-23T14:30:00Z";

export const DEMO_PROJECT = {
  id: "demo-acme",
  name: "Acme Commerce",
  slug: "acme-commerce",
};

export const DEMO_RUN = {
  id: "run_7f3a91c2",
  label: "Today · 09:42",
  pages: 48,
  duration: "6m 12s",
};

/* ------------------------------------------------------------------ inventory */

export interface InventoryTag extends Tag {
  purpose: string;
  pages: number;
  firstSeen: string;
  lastSeen: string;
}

export interface InventoryCookie extends Cookie {
  vendor: string;
  purpose: string;
  pages: number;
}

export interface InventoryRequest extends NetworkRequest {
  vendor: string;
  count: number;
  pages: number;
}

export const DEMO_TAGS: InventoryTag[] = [
  {
    id: "tag-gtm",
    vendor: "Google Tag Manager",
    name: "GTM container · GTM-7KQ2X",
    category: "Tag manager",
    version: "v2",
    confidence: 0.99,
    evidence: "https://www.googletagmanager.com/gtm.js?id=GTM-7KQ2X",
    purpose: "Loads analytics and marketing tags",
    pages: 48,
    firstSeen: "10:42:03",
    lastSeen: "10:48:15",
  },
  {
    id: "tag-ga4",
    vendor: "Google Analytics",
    name: "GA4 · G-4XZ8Q1LMN2",
    category: "Analytics",
    version: "gtag.js",
    confidence: 0.98,
    evidence: "window.dataLayer.push({ event: 'page_view' })",
    purpose: "Traffic and conversion analytics",
    pages: 48,
    firstSeen: "10:42:04",
    lastSeen: "10:48:15",
  },
  {
    id: "tag-meta",
    vendor: "Meta",
    name: "Meta Pixel · 612904551",
    category: "Advertising",
    version: "fbevents 2.9",
    confidence: 0.96,
    evidence: "https://connect.facebook.net/en_US/fbevents.js",
    purpose: "Paid social attribution",
    pages: 42,
    firstSeen: "10:42:06",
    lastSeen: "10:48:12",
  },
  {
    id: "tag-adobe",
    vendor: "Adobe",
    name: "Adobe Analytics · acme-prod",
    category: "Analytics",
    version: "AppMeasurement 2.24",
    confidence: 0.94,
    evidence: "https://assets.adobedtm.com/launch-EN.../launch.min.js",
    purpose: "Product and journey analytics",
    pages: 31,
    firstSeen: "10:42:08",
    lastSeen: "10:47:58",
  },
  {
    id: "tag-onetrust",
    vendor: "OneTrust",
    name: "OneTrust CMP · b8f1c2",
    category: "Consent",
    version: "2026.1",
    confidence: 0.99,
    evidence: "https://cdn.cookielaw.org/scripttemplates/otSDKStub.js",
    purpose: "Consent capture and enforcement",
    pages: 48,
    firstSeen: "10:42:02",
    lastSeen: "10:48:15",
  },
  {
    id: "tag-hotjar",
    vendor: "Hotjar",
    name: "Hotjar · 3412907",
    category: "Behavior",
    version: "hjid",
    confidence: 0.91,
    evidence: "https://static.hotjar.com/c/hotjar-3412907.js",
    purpose: "Session replay and heatmaps",
    pages: 18,
    firstSeen: "10:43:11",
    lastSeen: "10:47:41",
  },
  {
    id: "tag-tiktok",
    vendor: "TikTok",
    name: "TikTok Pixel · CQ7A1B",
    category: "Advertising",
    version: "pixel 1.4",
    confidence: 0.88,
    evidence: "https://analytics.tiktok.com/i18n/pixel/events.js",
    purpose: "Paid social attribution",
    pages: 12,
    firstSeen: "10:44:02",
    lastSeen: "10:47:33",
  },
  {
    id: "tag-linkedin",
    vendor: "LinkedIn",
    name: "Insight Tag · 904512",
    category: "Advertising",
    version: "insight 2.0",
    confidence: 0.62,
    evidence: "https://snap.licdn.com/li.lms-analytics/insight.min.js",
    purpose: "B2B campaign attribution",
    pages: 6,
    firstSeen: "10:45:26",
    lastSeen: "10:46:54",
  },
];

export const DEMO_COOKIES: InventoryCookie[] = [
  {
    name: "_ga",
    domain: ".acme.com",
    path: "/",
    secure: true,
    httpOnly: false,
    sameSite: "Lax",
    session: false,
    firstParty: true,
    vendor: "Google Analytics",
    purpose: "Analytics",
    pages: 48,
  },
  {
    name: "_ga_7KQ2X",
    domain: ".acme.com",
    path: "/",
    secure: true,
    httpOnly: false,
    sameSite: "Lax",
    session: false,
    firstParty: true,
    vendor: "Google Analytics",
    purpose: "Analytics",
    pages: 48,
  },
  {
    name: "_gid",
    domain: ".acme.com",
    path: "/",
    secure: true,
    httpOnly: false,
    sameSite: "Lax",
    session: false,
    firstParty: true,
    vendor: "Google Analytics",
    purpose: "Analytics",
    pages: 48,
  },
  {
    name: "_fbp",
    domain: ".acme.com",
    path: "/",
    secure: true,
    httpOnly: false,
    sameSite: "Lax",
    session: false,
    firstParty: true,
    vendor: "Meta",
    purpose: "Advertising",
    pages: 42,
  },
  {
    name: "OptanonConsent",
    domain: ".acme.com",
    path: "/",
    secure: true,
    httpOnly: false,
    sameSite: "Lax",
    session: false,
    firstParty: true,
    vendor: "OneTrust",
    purpose: "Consent",
    pages: 48,
  },
  {
    name: "OptanonAlertBoxClosed",
    domain: ".acme.com",
    path: "/",
    secure: true,
    httpOnly: false,
    sameSite: "Lax",
    session: false,
    firstParty: true,
    vendor: "OneTrust",
    purpose: "Consent",
    pages: 48,
  },
  {
    name: "s_cc",
    domain: ".acme.com",
    path: "/",
    secure: true,
    httpOnly: false,
    sameSite: "Lax",
    session: true,
    firstParty: true,
    vendor: "Adobe",
    purpose: "Analytics",
    pages: 31,
  },
  {
    name: "_hjSessionUser_3412907",
    domain: ".acme.com",
    path: "/",
    secure: true,
    httpOnly: true,
    sameSite: "Lax",
    session: false,
    firstParty: true,
    vendor: "Hotjar",
    purpose: "Behavior",
    pages: 18,
  },
  {
    name: "ttclid",
    domain: ".acme.com",
    path: "/",
    secure: true,
    httpOnly: false,
    sameSite: "Lax",
    session: true,
    firstParty: true,
    vendor: "TikTok",
    purpose: "Advertising",
    pages: 12,
  },
];

export const DEMO_REQUESTS: InventoryRequest[] = [
  {
    method: "GET",
    url: "https://www.googletagmanager.com/gtm.js?id=GTM-7KQ2X",
    host: "www.googletagmanager.com",
    status: 200,
    resourceType: "script",
    initiator: "document",
    durationMs: 128,
    sizeBytes: 184320,
    thirdParty: true,
    vendor: "Google Tag Manager",
    count: 48,
    pages: 48,
  },
  {
    method: "POST",
    url: "https://www.google-analytics.com/g/collect?v=2&tid=G-4XZ8Q1LMN2",
    host: "www.google-analytics.com",
    status: 204,
    resourceType: "fetch",
    initiator: "gtag.js",
    durationMs: 74,
    sizeBytes: 1024,
    thirdParty: true,
    vendor: "Google Analytics",
    count: 312,
    pages: 48,
  },
  {
    method: "GET",
    url: "https://connect.facebook.net/en_US/fbevents.js",
    host: "connect.facebook.net",
    status: 200,
    resourceType: "script",
    initiator: "GTM",
    durationMs: 96,
    sizeBytes: 94208,
    thirdParty: true,
    vendor: "Meta",
    count: 42,
    pages: 42,
  },
  {
    method: "GET",
    url: "https://static.hotjar.com/c/hotjar-3412907.js",
    host: "static.hotjar.com",
    status: 200,
    resourceType: "script",
    initiator: "GTM",
    durationMs: 112,
    sizeBytes: 61440,
    thirdParty: true,
    vendor: "Hotjar",
    count: 18,
    pages: 18,
  },
  {
    method: "GET",
    url: "https://analytics.tiktok.com/i18n/pixel/events.js",
    host: "analytics.tiktok.com",
    status: 200,
    resourceType: "script",
    initiator: "GTM",
    durationMs: 138,
    sizeBytes: 73728,
    thirdParty: true,
    vendor: "TikTok",
    count: 12,
    pages: 12,
  },
  {
    method: "GET",
    url: "https://cdn.cookielaw.org/scripttemplates/otSDKStub.js",
    host: "cdn.cookielaw.org",
    status: 200,
    resourceType: "script",
    initiator: "document",
    durationMs: 58,
    sizeBytes: 43008,
    thirdParty: true,
    vendor: "OneTrust",
    count: 48,
    pages: 48,
  },
  {
    method: "POST",
    url: "https://acme.com/api/cart/items",
    host: "acme.com",
    status: 200,
    resourceType: "fetch",
    initiator: "app.js",
    durationMs: 44,
    sizeBytes: 512,
    thirdParty: false,
    vendor: "Acme Commerce",
    count: 96,
    pages: 24,
  },
  {
    method: "GET",
    url: "https://sc.omtrdc.net/b/ss/acme-prod/1/JS-2.24.0/s412345",
    host: "sc.omtrdc.net",
    status: 200,
    resourceType: "image",
    initiator: "AppMeasurement",
    durationMs: 64,
    sizeBytes: 2048,
    thirdParty: true,
    vendor: "Adobe",
    count: 31,
    pages: 31,
  },
];

/* ------------------------------------------------------------------ schedules */

export interface DemoSchedule {
  id: string;
  name: string;
  url: string;
  cadence: string;
  cron: string;
  timezone: string;
  nextRun: string;
  nextRunLabel: string;
  lastStatus: RunStatus;
  lastRunLabel: string;
  owner: string;
  enabled: boolean;
  pages: number;
  depth: number;
  consent: string;
}

export const DEMO_SCHEDULES: DemoSchedule[] = [
  {
    id: "sch-001",
    name: "Production storefront",
    url: "https://www.acme.com/",
    cadence: "Daily · 06:00",
    cron: "0 6 * * *",
    timezone: "Europe/London",
    nextRun: "2026-10-24T06:00:00Z",
    nextRunLabel: "Tomorrow · 06:00",
    lastStatus: "succeeded",
    lastRunLabel: "Today · 06:00",
    owner: "Dana Whitfield",
    enabled: true,
    pages: 250,
    depth: 4,
    consent: "pre_consent",
  },
  {
    id: "sch-002",
    name: "EU checkout flow",
    url: "https://eu.acme.com/checkout",
    cadence: "Weekdays · 08:30",
    cron: "30 8 * * 1-5",
    timezone: "Europe/Berlin",
    nextRun: "2026-10-26T08:30:00Z",
    nextRunLabel: "Mon · 08:30",
    lastStatus: "failed",
    lastRunLabel: "Today · 08:30",
    owner: "Lena Fischer",
    enabled: true,
    pages: 60,
    depth: 3,
    consent: "accept_all",
  },
  {
    id: "sch-003",
    name: "Campaign landing pages",
    url: "https://www.acme.com/campaigns/",
    cadence: "Hourly",
    cron: "0 * * * *",
    timezone: "UTC",
    nextRun: "2026-10-23T15:00:00Z",
    nextRunLabel: "in 30 min",
    lastStatus: "running",
    lastRunLabel: "Today · 14:00",
    owner: "Marco Ruiz",
    enabled: true,
    pages: 120,
    depth: 2,
    consent: "reject_all",
  },
  {
    id: "sch-004",
    name: "Support portal",
    url: "https://help.acme.com/",
    cadence: "Weekly · Mon 06:00",
    cron: "0 6 * * 1",
    timezone: "UTC",
    nextRun: "2026-10-26T06:00:00Z",
    nextRunLabel: "Mon · 06:00",
    lastStatus: "succeeded",
    lastRunLabel: "Mon · 06:00",
    owner: "Dana Whitfield",
    enabled: false,
    pages: 80,
    depth: 3,
    consent: "pre_consent",
  },
  {
    id: "sch-005",
    name: "Partner microsites",
    url: "https://partners.acme.com/",
    cadence: "Monthly · 1st 02:00",
    cron: "0 2 1 * *",
    timezone: "UTC",
    nextRun: "2026-11-01T02:00:00Z",
    nextRunLabel: "Nov 1 · 02:00",
    lastStatus: "canceled",
    lastRunLabel: "Oct 1 · 02:00",
    owner: "Priya Nair",
    enabled: true,
    pages: 40,
    depth: 2,
    consent: "pre_consent",
  },
  {
    id: "sch-006",
    name: "Marketing blog",
    url: "https://blog.acme.com/",
    cadence: "Daily · 06:00",
    cron: "0 6 * * *",
    timezone: "America/New_York",
    nextRun: "2026-10-24T10:00:00Z",
    nextRunLabel: "Tomorrow · 06:00 ET",
    lastStatus: "succeeded",
    lastRunLabel: "Today · 06:00 ET",
    owner: "Marco Ruiz",
    enabled: true,
    pages: 180,
    depth: 4,
    consent: "pre_consent",
  },
];

/* --------------------------------------------------------------------- alerts */

export interface DemoAlertRule {
  id: string;
  name: string;
  severity: Severity;
  condition: string;
  channels: string[];
  enabled: boolean;
  lastTriggered: string;
  triggerCount: number;
}

export type IncidentStatus = "open" | "acknowledged" | "resolved";

export interface DemoIncident {
  id: string;
  severity: Severity;
  title: string;
  message: string;
  ruleName: string;
  channel: string;
  at: string;
  atLabel: string;
  status: IncidentStatus;
  runId: string;
}

export interface DemoChannel {
  id: string;
  name: string;
  kind: string;
  target: string;
  connected: boolean;
  lastDelivery: string;
}

export const DEMO_ALERT_RULES: DemoAlertRule[] = [
  {
    id: "rule-001",
    name: "Critical beacon leak",
    severity: "critical",
    condition: "Any critical finding in the latest run",
    channels: ["slack", "pagerduty"],
    enabled: true,
    lastTriggered: "Today · 09:48",
    triggerCount: 3,
  },
  {
    id: "rule-002",
    name: "Meta pixel before consent",
    severity: "high",
    condition: "consent.pre-consent-marketing-tag fails",
    channels: ["slack", "email"],
    enabled: true,
    lastTriggered: "Today · 09:47",
    triggerCount: 8,
  },
  {
    id: "rule-003",
    name: "Missing GTM container",
    severity: "high",
    condition: "Tag GTM-7KQ2X not present on checkout pages",
    channels: ["email"],
    enabled: true,
    lastTriggered: "Yesterday · 18:12",
    triggerCount: 2,
  },
  {
    id: "rule-004",
    name: "Run failed",
    severity: "medium",
    condition: "onRunFailure is true",
    channels: ["slack"],
    enabled: true,
    lastTriggered: "Today · 08:34",
    triggerCount: 5,
  },
  {
    id: "rule-005",
    name: "GDPR consent drift",
    severity: "medium",
    condition: "privacy.consent-not-honored fails in EU profile",
    channels: ["slack", "webhook"],
    enabled: false,
    lastTriggered: "Oct 18 · 11:20",
    triggerCount: 1,
  },
];

export const DEMO_INCIDENTS: DemoIncident[] = [
  {
    id: "inc-001",
    severity: "critical",
    title: "Meta pixel fired before consent",
    message: "connect.facebook.net/en_US/fbevents.js loaded on /checkout before OneTrust consent.",
    ruleName: "Critical beacon leak",
    channel: "PagerDuty",
    at: "2026-10-23T09:48:00Z",
    atLabel: "Today · 09:48",
    status: "open",
    runId: DEMO_RUN.id,
  },
  {
    id: "inc-002",
    severity: "high",
    title: "TikTok pixel missing consent gate",
    message: "12 campaign pages loaded analytics.tiktok.com without an explicit opt-in.",
    ruleName: "Meta pixel before consent",
    channel: "Slack · #privacy-alerts",
    at: "2026-10-23T09:47:00Z",
    atLabel: "Today · 09:47",
    status: "open",
    runId: DEMO_RUN.id,
  },
  {
    id: "inc-003",
    severity: "high",
    title: "GTM container absent on checkout",
    message: "GTM-7KQ2X was not detected on three /checkout variants after the 08:30 run.",
    ruleName: "Missing GTM container",
    channel: "Email · data-platform@acme.com",
    at: "2026-10-22T18:12:00Z",
    atLabel: "Yesterday · 18:12",
    status: "acknowledged",
    runId: "run_5c1e84ab",
  },
  {
    id: "inc-004",
    severity: "medium",
    title: "EU checkout crawl failed",
    message: "Renderer timed out after 90s on https://eu.acme.com/checkout/payment.",
    ruleName: "Run failed",
    channel: "Slack · #gov-ops",
    at: "2026-10-23T08:34:00Z",
    atLabel: "Today · 08:34",
    status: "resolved",
    runId: "run_1a9b73de",
  },
  {
    id: "inc-005",
    severity: "low",
    title: "Hotjar session cookie drift",
    message: "_hjSessionUser_3412907 persisted for 400 days; policy expects 365.",
    ruleName: "GDPR consent drift",
    channel: "Webhook · compliance-sink",
    at: "2026-10-18T11:20:00Z",
    atLabel: "Oct 18 · 11:20",
    status: "resolved",
    runId: "run_9e2d40bc",
  },
  {
    id: "inc-006",
    severity: "info",
    title: "New vendor detected",
    message: "LinkedIn Insight Tag appeared on six pages for the first time.",
    ruleName: "Critical beacon leak",
    channel: "Slack · #gov-ops",
    at: "2026-10-17T16:05:00Z",
    atLabel: "Oct 17 · 16:05",
    status: "resolved",
    runId: "run_3f7c28aa",
  },
];

export const DEMO_CHANNELS: DemoChannel[] = [
  {
    id: "ch-slack",
    name: "Slack",
    kind: "slack",
    target: "#privacy-alerts · #gov-ops",
    connected: true,
    lastDelivery: "Today · 09:48",
  },
  {
    id: "ch-email",
    name: "Email",
    kind: "email",
    target: "data-platform@acme.com",
    connected: true,
    lastDelivery: "Today · 09:47",
  },
  {
    id: "ch-webhook",
    name: "Webhook",
    kind: "webhook",
    target: "https://hooks.acme.com/compliance",
    connected: true,
    lastDelivery: "Oct 18 · 11:20",
  },
  {
    id: "ch-pagerduty",
    name: "PagerDuty",
    kind: "message",
    target: "privacy-oncall",
    connected: false,
    lastDelivery: "Never",
  },
];

/* ------------------------------------------------------------------- journeys */

export type JourneyAction = "navigate" | "click" | "type" | "wait" | "assert" | "extract";
export type JourneyStepStatus = "pending" | "running" | "passed" | "failed";

export interface DemoJourneyStep {
  order: number;
  action: JourneyAction;
  target: string;
  selector: string;
  value?: string;
  expected?: string;
  status: JourneyStepStatus;
}

export interface DemoJourneyRun {
  id: string;
  status: "passed" | "failed" | "running";
  at: string;
  atLabel: string;
  stepsPassed: number;
  stepsTotal: number;
  durationSec: number;
}

export interface DemoJourney {
  id: string;
  name: string;
  url: string;
  description: string;
  owner: string;
  passRate: number;
  lastRun: string;
  avgDuration: string;
  steps: DemoJourneyStep[];
  runs: DemoJourneyRun[];
}

export const DEMO_JOURNEYS: DemoJourney[] = [
  {
    id: "journey-checkout",
    name: "Guest checkout",
    url: "https://www.acme.com/checkout",
    description: "Add a product, pay as a guest and confirm purchase events fire once.",
    owner: "Dana Whitfield",
    passRate: 92,
    lastRun: "Today · 09:51",
    avgDuration: "1m 48s",
    steps: [
      { order: 1, action: "navigate", target: "Product detail page", selector: "/products/aurora-jacket", status: "passed" },
      { order: 2, action: "click", target: "Add to cart", selector: "[data-testid=add-to-cart]", expected: "Cart badge = 1", status: "passed" },
      { order: 3, action: "click", target: "Checkout", selector: "a[href='/checkout']", status: "passed" },
      { order: 4, action: "type", target: "Email field", selector: "#email", value: "qa+checkout@acme.com", status: "passed" },
      { order: 5, action: "click", target: "Continue as guest", selector: "[data-testid=guest-checkout]", status: "passed" },
      { order: 6, action: "assert", target: "purchase dataLayer event", selector: "dataLayer", expected: "event = purchase", status: "failed" },
      { order: 7, action: "extract", target: "Order confirmation ID", selector: "[data-testid=order-id]", status: "pending" },
    ],
    runs: [
      { id: "jr-1041", status: "failed", at: "2026-10-23T09:51:00Z", atLabel: "Today · 09:51", stepsPassed: 5, stepsTotal: 7, durationSec: 112 },
      { id: "jr-1038", status: "passed", at: "2026-10-22T09:50:00Z", atLabel: "Yesterday · 09:50", stepsPassed: 7, stepsTotal: 7, durationSec: 104 },
      { id: "jr-1034", status: "passed", at: "2026-10-21T09:52:00Z", atLabel: "Oct 21 · 09:52", stepsPassed: 7, stepsTotal: 7, durationSec: 109 },
    ],
  },
  {
    id: "journey-consent",
    name: "Consent banner flow",
    url: "https://www.acme.com/",
    description: "Reject optional cookies and verify no marketing beacons fire.",
    owner: "Lena Fischer",
    passRate: 78,
    lastRun: "Today · 08:15",
    avgDuration: "58s",
    steps: [
      { order: 1, action: "navigate", target: "Homepage", selector: "/", status: "passed" },
      { order: 2, action: "wait", target: "OneTrust banner", selector: "#onetrust-banner-sdk", expected: "visible", status: "passed" },
      { order: 3, action: "click", target: "Reject all", selector: "#onetrust-reject-all-handler", status: "passed" },
      { order: 4, action: "assert", target: "No Meta beacon", selector: "network", expected: "facebook.net absent", status: "failed" },
      { order: 5, action: "extract", target: "Consent cookie", selector: "OptanonConsent", status: "pending" },
    ],
    runs: [
      { id: "jr-1040", status: "failed", at: "2026-10-23T08:15:00Z", atLabel: "Today · 08:15", stepsPassed: 3, stepsTotal: 5, durationSec: 61 },
      { id: "jr-1037", status: "failed", at: "2026-10-22T08:14:00Z", atLabel: "Yesterday · 08:14", stepsPassed: 3, stepsTotal: 5, durationSec: 57 },
      { id: "jr-1032", status: "passed", at: "2026-10-21T08:16:00Z", atLabel: "Oct 21 · 08:16", stepsPassed: 5, stepsTotal: 5, durationSec: 55 },
    ],
  },
  {
    id: "journey-search",
    name: "Search and filter",
    url: "https://www.acme.com/search",
    description: "Search, apply a filter and confirm GA4 view_item_list fires.",
    owner: "Marco Ruiz",
    passRate: 100,
    lastRun: "Today · 07:40",
    avgDuration: "42s",
    steps: [
      { order: 1, action: "navigate", target: "Search page", selector: "/search?q=jacket", status: "passed" },
      { order: 2, action: "type", target: "Search input", selector: "#search-input", value: "rain jacket", status: "passed" },
      { order: 3, action: "click", target: "Category filter", selector: "[data-filter=outerwear]", status: "passed" },
      { order: 4, action: "assert", target: "GA4 view_item_list", selector: "dataLayer", expected: "event = view_item_list", status: "passed" },
    ],
    runs: [
      { id: "jr-1039", status: "passed", at: "2026-10-23T07:40:00Z", atLabel: "Today · 07:40", stepsPassed: 4, stepsTotal: 4, durationSec: 41 },
      { id: "jr-1036", status: "passed", at: "2026-10-22T07:41:00Z", atLabel: "Yesterday · 07:41", stepsPassed: 4, stepsTotal: 4, durationSec: 44 },
    ],
  },
  {
    id: "journey-account",
    name: "Account sign-up",
    url: "https://www.acme.com/account/register",
    description: "Create an account and confirm the Adobe registration event.",
    owner: "Priya Nair",
    passRate: 64,
    lastRun: "Yesterday · 16:22",
    avgDuration: "1m 12s",
    steps: [
      { order: 1, action: "navigate", target: "Register page", selector: "/account/register", status: "passed" },
      { order: 2, action: "type", target: "Email", selector: "#register-email", value: "qa+account@acme.com", status: "passed" },
      { order: 3, action: "type", target: "Password", selector: "#register-password", value: "••••••••", status: "passed" },
      { order: 4, action: "click", target: "Create account", selector: "[data-testid=register-submit]", status: "passed" },
      { order: 5, action: "assert", target: "Adobe registration event", selector: "digitalData", expected: "event = registration", status: "failed" },
    ],
    runs: [
      { id: "jr-1035", status: "failed", at: "2026-10-22T16:22:00Z", atLabel: "Yesterday · 16:22", stepsPassed: 4, stepsTotal: 5, durationSec: 73 },
      { id: "jr-1031", status: "passed", at: "2026-10-21T16:20:00Z", atLabel: "Oct 21 · 16:20", stepsPassed: 5, stepsTotal: 5, durationSec: 69 },
    ],
  },
];

/* -------------------------------------------------------------------- privacy */

export type RegulationId = "gdpr" | "ccpa" | "lgpd" | "eprivacy";
export type PrivacyResult = "pass" | "fail" | "na";

export interface DemoRegulation {
  id: RegulationId;
  name: string;
  fullName: string;
  score: number;
  status: "pass" | "warn" | "fail";
  checks: number;
  failures: number;
  lastEvaluated: string;
}

export interface DemoPrivacyProfile {
  id: string;
  name: string;
  region: string;
  regulationIds: RegulationId[];
  active: boolean;
}

export interface DemoPrivacyCheck {
  id: string;
  label: string;
  category: string;
  results: Record<RegulationId, PrivacyResult>;
}

export interface DemoPrivacyGap {
  id: string;
  regulation: RegulationId;
  check: string;
  severity: Severity;
  detail: string;
  remediation: string;
}

export const DEMO_REGULATIONS: DemoRegulation[] = [
  { id: "gdpr", name: "GDPR", fullName: "EU General Data Protection Regulation", score: 86, status: "warn", checks: 24, failures: 3, lastEvaluated: "Today · 09:48" },
  { id: "ccpa", name: "CCPA", fullName: "California Consumer Privacy Act / CPRA", score: 94, status: "pass", checks: 18, failures: 1, lastEvaluated: "Today · 09:48" },
  { id: "lgpd", name: "LGPD", fullName: "Brazil Lei Geral de Proteção de Dados", score: 71, status: "fail", checks: 16, failures: 5, lastEvaluated: "Yesterday · 18:12" },
  { id: "eprivacy", name: "ePrivacy", fullName: "EU ePrivacy Directive", score: 81, status: "warn", checks: 14, failures: 2, lastEvaluated: "Today · 09:48" },
];

export const DEMO_PRIVACY_PROFILES: DemoPrivacyProfile[] = [
  { id: "global", name: "Global baseline", region: "All regions", regulationIds: ["gdpr", "ccpa", "lgpd", "eprivacy"], active: true },
  { id: "eu", name: "EU operations", region: "European Union", regulationIds: ["gdpr", "eprivacy"], active: true },
  { id: "us", name: "US operations", region: "United States", regulationIds: ["ccpa"], active: true },
  { id: "br", name: "Brazil operations", region: "Brazil", regulationIds: ["lgpd"], active: true },
];

export const DEMO_PRIVACY_CHECKS: DemoPrivacyCheck[] = [
  {
    id: "pc-01",
    label: "Consent before non-essential beacons",
    category: "Consent",
    results: { gdpr: "fail", ccpa: "pass", lgpd: "fail", eprivacy: "fail" },
  },
  {
    id: "pc-02",
    label: "Reject-all honours tracker blocking",
    category: "Consent",
    results: { gdpr: "fail", ccpa: "pass", lgpd: "fail", eprivacy: "fail" },
  },
  {
    id: "pc-03",
    label: "Global Privacy Control respected",
    category: "Signals",
    results: { gdpr: "pass", ccpa: "pass", lgpd: "na", eprivacy: "pass" },
  },
  {
    id: "pc-04",
    label: "Cookie retention within policy",
    category: "Cookies",
    results: { gdpr: "pass", ccpa: "pass", lgpd: "fail", eprivacy: "pass" },
  },
  {
    id: "pc-05",
    label: "Cross-border transfer disclosure",
    category: "Governance",
    results: { gdpr: "pass", ccpa: "na", lgpd: "fail", eprivacy: "na" },
  },
  {
    id: "pc-06",
    label: "Opt-out link for sale/share",
    category: "Rights",
    results: { gdpr: "na", ccpa: "fail", lgpd: "na", eprivacy: "na" },
  },
  {
    id: "pc-07",
    label: "Data subject request channel",
    category: "Rights",
    results: { gdpr: "pass", ccpa: "pass", lgpd: "fail", eprivacy: "pass" },
  },
];

export const DEMO_PRIVACY_GAPS: DemoPrivacyGap[] = [
  {
    id: "gap-01",
    regulation: "gdpr",
    check: "Consent before non-essential beacons",
    severity: "critical",
    detail: "Meta and TikTok pixels fire on /checkout before OneTrust consent is granted.",
    remediation: "Gate fbevents.js and TikTok events behind the OneTrust marketing category.",
  },
  {
    id: "gap-02",
    regulation: "eprivacy",
    check: "Reject-all honours tracker blocking",
    severity: "high",
    detail: "Rejecting all cookies still leaves the Hotjar session replay script active.",
    remediation: "Bind the Hotjar snippet to the OneTrust performance category callback.",
  },
  {
    id: "gap-03",
    regulation: "lgpd",
    check: "Cross-border transfer disclosure",
    severity: "high",
    detail: "Adobe Analytics traffic to sc.omtrdc.net is not listed in the privacy notice.",
    remediation: "Update the notice and DPIA with the Adobe data-processing addendum.",
  },
  {
    id: "gap-04",
    regulation: "lgpd",
    check: "Data subject request channel",
    severity: "medium",
    detail: "No LGPD-specific request form is linked from the privacy footer.",
    remediation: "Publish an LGPD request form and route it to the DPO queue.",
  },
  {
    id: "gap-05",
    regulation: "ccpa",
    check: "Opt-out link for sale/share",
    severity: "medium",
    detail: "Do Not Sell or Share link is not present in the site footer.",
    remediation: "Add the CPRA opt-out link and honour GPC on first load.",
  },
  {
    id: "gap-06",
    regulation: "gdpr",
    check: "Cookie retention within policy",
    severity: "low",
    detail: "LinkedIn Insight cookie lifetime exceeds the 365-day retention policy.",
    remediation: "Cap the li_fat_id cookie lifetime in the tag configuration.",
  },
];

export const DEMO_CONSENT_STATES = [
  { id: "accept_all", label: "Accept all", trackers: 18, blocked: 0, leaks: 2, status: "warn" as const },
  { id: "reject_all", label: "Reject all", trackers: 4, blocked: 14, leaks: 1, status: "pass" as const },
  { id: "necessary_only", label: "Necessary only", trackers: 5, blocked: 13, leaks: 1, status: "pass" as const },
  { id: "gpc", label: "Global Privacy Control", trackers: 3, blocked: 15, leaks: 0, status: "pass" as const },
];

/* -------------------------------------------------------------------- reports */

export type ReportFormat = "json" | "csv" | "html";

export interface DemoReportTemplate {
  id: string;
  name: string;
  description: string;
  icon: IconName;
  formats: ReportFormat[];
  tone: Tone;
}

export type ReportStatus = "ready" | "generating" | "failed";

export interface DemoReportRow {
  id: string;
  name: string;
  template: string;
  scope: string;
  format: ReportFormat;
  createdAt: string;
  createdLabel: string;
  size: string;
  status: ReportStatus;
  owner: string;
}

export interface DemoReportSchedule {
  id: string;
  name: string;
  template: string;
  cadence: string;
  recipients: string;
  nextRunLabel: string;
  enabled: boolean;
}

export const DEMO_REPORT_TEMPLATES: DemoReportTemplate[] = [
  {
    id: "tpl-governance",
    name: "Governance summary",
    description: "Health score, rule pass rates, beacons and top findings.",
    icon: "shield",
    formats: ["html", "json"],
    tone: "indigo",
  },
  {
    id: "tpl-privacy",
    name: "Privacy compliance",
    description: "Per-jurisdiction scores, consent states and remediation gaps.",
    icon: "scale",
    formats: ["html", "csv"],
    tone: "emerald",
  },
  {
    id: "tpl-consent",
    name: "Consent matrix",
    description: "Pre/post consent snapshots with every tracker and leak.",
    icon: "cookie",
    formats: ["csv", "json"],
    tone: "rose",
  },
  {
    id: "tpl-journey",
    name: "Journey results",
    description: "Step-level pass/fail evidence for scripted user flows.",
    icon: "route",
    formats: ["html", "json"],
    tone: "amber",
  },
  {
    id: "tpl-inventory",
    name: "Tag & cookie inventory",
    description: "Detected tags, cookies and third-party request inventory.",
    icon: "database",
    formats: ["csv", "json"],
    tone: "sky",
  },
];

export const DEMO_REPORTS: DemoReportRow[] = [
  {
    id: "rep-1041",
    name: "Acme Commerce · governance",
    template: "Governance summary",
    scope: "Production storefront",
    format: "html",
    createdAt: "2026-10-23T09:52:00Z",
    createdLabel: "Today · 09:52",
    size: "1.8 MB",
    status: "ready",
    owner: "Dana Whitfield",
  },
  {
    id: "rep-1040",
    name: "EU privacy · Q4 evidence",
    template: "Privacy compliance",
    scope: "EU operations",
    format: "html",
    createdAt: "2026-10-23T09:50:00Z",
    createdLabel: "Today · 09:50",
    size: "924 KB",
    status: "ready",
    owner: "Lena Fischer",
  },
  {
    id: "rep-1039",
    name: "Consent matrix · full crawl",
    template: "Consent matrix",
    scope: "Production storefront",
    format: "csv",
    createdAt: "2026-10-23T08:05:00Z",
    createdLabel: "Today · 08:05",
    size: "312 KB",
    status: "ready",
    owner: "Marco Ruiz",
  },
  {
    id: "rep-1038",
    name: "Guest checkout journey",
    template: "Journey results",
    scope: "Checkout flow",
    format: "json",
    createdAt: "2026-10-22T18:14:00Z",
    createdLabel: "Yesterday · 18:14",
    size: "148 KB",
    status: "ready",
    owner: "Dana Whitfield",
  },
  {
    id: "rep-1037",
    name: "Vendor inventory · October",
    template: "Tag & cookie inventory",
    scope: "All sites",
    format: "csv",
    createdAt: "2026-10-22T17:40:00Z",
    createdLabel: "Yesterday · 17:40",
    size: "486 KB",
    status: "generating",
    owner: "Priya Nair",
  },
  {
    id: "rep-1036",
    name: "Partner microsites · audit",
    template: "Governance summary",
    scope: "Partner microsites",
    format: "json",
    createdAt: "2026-10-21T11:08:00Z",
    createdLabel: "Oct 21 · 11:08",
    size: "—",
    status: "failed",
    owner: "Marco Ruiz",
  },
];

export const DEMO_REPORT_SCHEDULES: DemoReportSchedule[] = [
  {
    id: "rs-01",
    name: "Weekly board governance pack",
    template: "Governance summary",
    cadence: "Weekly · Mon 07:00",
    recipients: "exec@acme.com, dpo@acme.com",
    nextRunLabel: "Mon · 07:00",
    enabled: true,
  },
  {
    id: "rs-02",
    name: "Monthly privacy evidence",
    template: "Privacy compliance",
    cadence: "Monthly · 1st 08:00",
    recipients: "dpo@acme.com, legal@acme.com",
    nextRunLabel: "Nov 1 · 08:00",
    enabled: true,
  },
  {
    id: "rs-03",
    name: "Daily consent drift digest",
    template: "Consent matrix",
    cadence: "Daily · 07:30",
    recipients: "data-platform@acme.com",
    nextRunLabel: "Tomorrow · 07:30",
    enabled: false,
  },
];

export type Tone = "slate" | "indigo" | "emerald" | "rose" | "amber" | "sky";
