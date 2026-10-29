export const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";

export interface Project {
  id: string;
  name: string;
  slug: string;
}

export interface ScanConfig {
  maxDepth?: number;
  maxPages?: number;
  concurrency?: number;
  sameOriginOnly?: boolean;
  respectRobotsTxt?: boolean;
  renderMode?: "http" | "browser" | "auto";
  userAgent?: string;
  captureDataLayer?: boolean;
  captureHar?: boolean;
  consentSimulation?: string;
  geolocations?: string[];
}

export interface Scan {
  id: string;
  projectId: string;
  name: string;
  startUrls: string[];
  config: ScanConfig;
  ruleSetIds?: string[];
  enabled: boolean;
}

export interface RunStats {
  pagesDiscovered: number;
  pagesScanned: number;
  pagesFailed: number;
  findings: number;
  beacons?: number;
  criticalFindings?: number;
  healthScore?: number;
  consentScore?: number;
  vendorScores?: Record<string, number>;
}

export type RunStatus = "queued" | "running" | "succeeded" | "failed" | "canceled";

export interface Run {
  id: string;
  scanId: string;
  projectId: string;
  status: RunStatus;
  startedAt?: string;
  finishedAt?: string;
  stats: RunStats;
  error?: string;
  createdAt?: string;
}

export interface Page {
  id: string;
  url: string;
  finalUrl?: string;
  status: number;
  statusCode?: number;
  depth?: number;
  rendered: boolean;
  title?: string;
  error?: string;
  scannedAt?: string;
}


export type Severity = "info" | "low" | "medium" | "high" | "critical";

export interface Finding {
  id: string;
  ruleId: string;
  pluginId: string;
  severity: Severity;
  title: string;
  description: string;
  pageUrl: string;
  evidence?: Record<string, unknown>;
}

export interface NetworkRequest {
  method: string;
  url: string;
  host: string;
  status: number;
  resourceType: string;
  initiator?: string;
  durationMs: number;
  sizeBytes: number;
  thirdParty: boolean;
}

export interface Tag {
  id: string;
  vendor: string;
  name: string;
  category: string;
  version?: string;
  evidence: string;
  confidence: number;
}

export interface Cookie {
  name: string;
  domain: string;
  path: string;
  secure: boolean;
  httpOnly: boolean;
  sameSite: string;
  session: boolean;
  firstParty: boolean;
}

export interface ConsoleMessage {
  level: string;
  text: string;
}

export interface DataLayerEntry {
  name: string;
  value?: unknown;
}

export interface ScanResult {
  schemaVersion: string;
  runId: string;
  scanId: string;
  pageId: string;
  url: string;
  finalUrl: string;
  depth: number;
  attempt: number;
  resumed: boolean;
  statusCode: number;
  contentType?: string;
  title?: string;
  rendered: boolean;
  renderMode: string;
  geolocation?: string;
  proxyHost?: string;
  startedAt: string;
  finishedAt: string;
  durationMs: number;
  domTruncated: boolean;
  domBytes: number;
  requests: NetworkRequest[] | null;
  cookies: Cookie[] | null;
  console: ConsoleMessage[] | null;
  dataLayer: DataLayerEntry[] | null;
  tags: Tag[] | null;
  links: string[] | null;
  metaDescription?: string;
  error?: string;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { ...((init?.headers as Record<string, string> | undefined) ?? {}) };
  if (init?.body !== undefined && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
    cache: "no-store",
  });
  if (!res.ok) {
    const text = await res.text();
    let message = text;
    try {
      const parsed = JSON.parse(text) as { error?: string };
      message = parsed.error ?? text;
    } catch {
      /* plain text error */
    }
    throw new Error(message || `${res.status} ${res.statusText}`);
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export async function listProjects(): Promise<Project[]> {
  return (await request<{ projects: Project[] }>("/api/v1/projects")).projects ?? [];
}

export async function createProject(name: string): Promise<Project> {
  return request<Project>("/api/v1/projects", { method: "POST", body: JSON.stringify({ name }) });
}

export async function listScans(projectId?: string): Promise<Scan[]> {
  const qs = projectId ? `?projectId=${encodeURIComponent(projectId)}` : "";
  return (await request<{ scans: Scan[] }>(`/api/v1/scans${qs}`)).scans ?? [];
}

export async function createScan(input: {
  projectId: string;
  name: string;
  startUrls: string[];
  config?: ScanConfig;
}): Promise<Scan> {
  return request<Scan>("/api/v1/scans", { method: "POST", body: JSON.stringify(input) });
}

export async function startRun(scanId: string): Promise<Run> {
  return request<Run>(`/api/v1/scans/${scanId}/runs`, { method: "POST" });
}

export async function listScanRuns(scanId: string, limit = 25): Promise<Run[]> {
  const qs = limit ? `?limit=${limit}` : "";
  return (await request<{ runs: Run[] }>(`/api/v1/scans/${scanId}/runs${qs}`)).runs ?? [];
}

export async function getRun(id: string): Promise<Run> {
  return request<Run>(`/api/v1/runs/${id}`);
}

export async function cancelRun(id: string): Promise<void> {
  await request<{ status: string }>(`/api/v1/runs/${id}/cancel`, { method: "POST" });
}

export async function resumeRun(id: string): Promise<Run> {
  return request<Run>(`/api/v1/runs/${id}/resume`, { method: "POST" });
}

export async function listScanResults(runId: string, limit = 500): Promise<ScanResult[]> {
  const qs = limit ? `?limit=${limit}` : "";
  return (await request<{ results: ScanResult[] }>(`/api/v1/runs/${runId}/results${qs}`)).results ?? [];
}

export async function getScanResult(runId: string, url: string): Promise<ScanResult> {
  return request<ScanResult>(`/api/v1/runs/${runId}/result?url=${encodeURIComponent(url)}`);
}

export async function listPages(runId: string): Promise<Page[]> {
  return (await request<{ pages: Page[] }>(`/api/v1/runs/${runId}/pages`)).pages ?? [];
}

export async function listFindings(runId: string, minSeverity?: string): Promise<Finding[]> {
  const qs = minSeverity ? `?minSeverity=${minSeverity}` : "";
  return (await request<{ findings: Finding[] }>(`/api/v1/runs/${runId}/findings${qs}`)).findings ?? [];
}

export function runEventStreamURL(runId: string): string {
  return `${API_BASE}/api/v1/runs/${runId}/events`;
}

export function runWebSocketURL(runId: string): string {
  return API_BASE.replace(/^http/, "ws") + `/api/v1/runs/${runId}/stream`;
}

export type RuleTarget = "request" | "event" | "tag" | "cookie" | "beacon" | "datalayer" | "consent" | "page" | "script" | "network" | "any";
export type RuleOperator = "exists" | "not_exists" | "equals" | "not_equals" | "matches" | "not_matches";

export interface AuditRule {
  id: string;
  projectId: string;
  name: string;
  vendor: string;
  enabled: boolean;
  urlPattern: string;
  target: RuleTarget;
  parameter: string;
  event?: string;
  operator: RuleOperator;
  expected: string;
  severity: Severity;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface AuditRuleInput {
  projectId?: string;
  name: string;
  vendor?: string;
  enabled?: boolean;
  urlPattern?: string;
  target: RuleTarget;
  parameter?: string;
  event?: string;
  operator: RuleOperator;
  expected?: string;
  severity: Severity;
  description?: string;
}

export interface KPI {
  pagesAudited: number;
  pagesTarget: number;
  healthScore: number;
  beacons: number;
  criticalFindings: number;
  complianceScore: number;
  findings: number;
  consentLeaks: number;
  pagesValidated: number;
}

export interface TrendPoint {
  date: string;
  health: number;
  consent: number;
  runs: number;
}

export interface VendorSlice {
  vendor: string;
  count: number;
  percent: number;
}

export interface OverviewViolation {
  id: string;
  runId: string;
  pageUrl: string;
  ruleId: string;
  title: string;
  description: string;
  vendor: string;
  severity: Severity;
  at: string;
}

export interface Overview {
  project?: Project;
  generatedAt: string;
  kpi: KPI;
  trend: TrendPoint[];
  vendors: VendorSlice[];
  violations: OverviewViolation[];
  latestRun?: Run;
  inFlight: number;
}

export interface AuditRow {
  runId: string;
  scanId: string;
  projectId: string;
  scanName: string;
  startUrl: string;
  status: RunStatus;
  startedAt?: string;
  finishedAt?: string;
  durationSec: number;
  stats: RunStats;
  error?: string;
  createdAt: string;
}

export interface RuleEvaluation {
  ruleId: string;
  name: string;
  vendor: string;
  target: string;
  severity: Severity;
  status: "pass" | "fail" | "skipped";
  reason?: string;
  evidence?: Record<string, unknown>;
}

export interface PageValidation {
  url: string;
  finalUrl?: string;
  pageId: string;
  statusCode: number;
  durationMs: number;
  error?: string;
  passed: boolean;
  evaluated: number;
  results: RuleEvaluation[];
}

export interface RuleStat {
  rule: AuditRule;
  passed: number;
  failed: number;
  skipped: number;
  passRate: number;
}

export interface ValidationReport {
  runId: string;
  scanId: string;
  generatedAt: string;
  consentSimulation?: string;
  healthScore: number;
  consentScore: number;
  beacons: number;
  pages: PageValidation[];
  ruleStats: RuleStat[];
  vendorScores: Record<string, number>;
  findings: Finding[];
}

export interface Beacon {
  url: string;
  host: string;
  vendor: string;
  method: string;
  resourceType: string;
  status: number;
  thirdParty: boolean;
  durationMs: number;
  sizeBytes: number;
  startedAtMs: number;
  query?: Record<string, string>;
}

export interface BeaconReport {
  runId: string;
  url: string;
  pageUrl: string;
  pageId: string;
  statusCode: number;
  durationMs: number;
  count: number;
  vendors: Record<string, number>;
  beacons: Beacon[];
}

export interface ConsentEntry {
  key: string;
  category: string;
  name: string;
  vendor: string;
  type: string;
  essential: boolean;
  preConsent: boolean;
  postConsent: boolean;
  leak: boolean;
  status: string;
  evidence?: string;
  cookies?: string[];
}

export interface ConsentMatrix {
  runId: string;
  url: string;
  pageUrl: string;
  pageId: string;
  generatedAt: string;
  consentSimulation?: string;
  cmpDetected: boolean;
  cmpVendor?: string;
  trackers: number;
  beacons: number;
  leaks: number;
  preConsentScore: number;
  postConsentScore: number;
  entries: ConsentEntry[];
}

export async function getOverview(projectId?: string): Promise<Overview> {
  const qs = projectId ? `?projectId=${encodeURIComponent(projectId)}` : "";
  return request<Overview>(`/api/v1/overview${qs}`);
}

export async function listAudits(projectId?: string, limit = 100): Promise<AuditRow[]> {
  const params = new URLSearchParams();
  if (projectId) params.set("projectId", projectId);
  if (limit) params.set("limit", String(limit));
  const qs = params.toString() ? `?${params.toString()}` : "";
  return (await request<{ audits: AuditRow[] }>(`/api/v1/audits${qs}`)).audits ?? [];
}

export async function createAudit(input: {
  projectId?: string;
  name?: string;
  startUrl: string;
  maxPages?: number;
  maxDepth?: number;
  renderMode?: "http" | "browser" | "auto";
  consentSimulation?: string;
  geolocation?: string;
}): Promise<{ scan: Scan; run: Run }> {
  return request<{ scan: Scan; run: Run }>("/api/v1/audits", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function auditExportURL(projectId?: string): string {
  const qs = projectId ? `?projectId=${encodeURIComponent(projectId)}` : "";
  return `${API_BASE}/api/v1/audits/export${qs}`;
}

export async function getValidation(runId: string): Promise<ValidationReport> {
  return request<ValidationReport>(`/api/v1/runs/${runId}/validation`);
}

export async function getBeacons(runId: string, url?: string): Promise<BeaconReport> {
  const qs = url ? `?url=${encodeURIComponent(url)}` : "";
  return request<BeaconReport>(`/api/v1/runs/${runId}/beacons${qs}`);
}

export async function getConsent(runId: string, url?: string): Promise<ConsentMatrix> {
  const qs = url ? `?url=${encodeURIComponent(url)}` : "";
  return request<ConsentMatrix>(`/api/v1/runs/${runId}/consent${qs}`);
}

export function reportURL(runId: string, format: "json" | "csv" | "html"): string {
  return `${API_BASE}/api/v1/runs/${runId}/report?format=${format}`;
}

export async function listRules(projectId?: string): Promise<AuditRule[]> {
  const qs = projectId ? `?projectId=${encodeURIComponent(projectId)}` : "";
  return (await request<{ rules: AuditRule[] }>(`/api/v1/rules${qs}`)).rules ?? [];
}

export async function createRule(input: AuditRuleInput): Promise<AuditRule> {
  return request<AuditRule>("/api/v1/rules", { method: "POST", body: JSON.stringify(input) });
}

export async function updateRule(id: string, patch: Partial<AuditRuleInput>): Promise<AuditRule> {
  return request<AuditRule>(`/api/v1/rules/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
}

export async function deleteRule(id: string): Promise<void> {
  await request<void>(`/api/v1/rules/${id}`, { method: "DELETE" });
}

export async function installStarterRules(projectId?: string): Promise<AuditRule[]> {
  return (
    await request<{ rules: AuditRule[] }>("/api/v1/rules/starter", {
      method: "POST",
      body: JSON.stringify({ projectId }),
    })
  ).rules ?? [];
}

export async function getHealth(): Promise<{ status: string; inflight: number }> {
  return request<{ status: string; inflight: number }>("/healthz");
}

// ---- feature clients ----

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

export interface InventorySnapshot {
  projectId: string;
  runId?: string;
  scanId?: string;
  pagesScanned: number;
  tags: InventoryTag[];
  cookies: InventoryCookie[];
  requests: InventoryRequest[];
  generatedAt: string;
}

export interface Schedule {
  id: string;
  projectId: string;
  scanId?: string;
  name: string;
  url: string;
  cron: string;
  timezone: string;
  enabled: boolean;
  maxPages: number;
  maxDepth: number;
  consent: string;
  owner: string;
  lastRunId?: string;
  lastStatus?: RunStatus;
  lastRunAt?: string;
  nextRunAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ScheduleInput {
  projectId?: string;
  scanId?: string;
  name: string;
  url: string;
  cron: string;
  timezone?: string;
  enabled?: boolean;
  maxPages?: number;
  maxDepth?: number;
  consent?: string;
  owner?: string;
}

export interface AlertCondition {
  minSeverity?: Severity;
  ruleIds?: string[];
  onRunFailure?: boolean;
  tagMissing?: string[];
}

export type IncidentStatus = "open" | "acknowledged" | "resolved";

export interface AlertRule {
  id: string;
  projectId: string;
  name: string;
  enabled: boolean;
  severity: Severity;
  condition: AlertCondition;
  conditionText?: string;
  channels?: string[];
  channelIds: string[];
  lastTriggeredAt?: string;
  triggerCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface AlertRuleInput {
  projectId?: string;
  name: string;
  enabled?: boolean;
  severity: Severity;
  condition: AlertCondition;
  conditionText?: string;
  channelIds?: string[];
}

export type AlertChannelKind = "slack" | "webhook" | "email" | "pagerduty";

export interface AlertChannel {
  id: string;
  projectId: string;
  name: string;
  kind: AlertChannelKind;
  target: string;
  secretRef?: string;
  connected: boolean;
  lastDeliveryAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface AlertChannelInput {
  projectId?: string;
  name: string;
  kind: AlertChannelKind;
  target: string;
  secretRef?: string;
}

export interface AlertIncident {
  id: string;
  projectId: string;
  ruleId?: string;
  runId?: string;
  severity: Severity;
  title: string;
  message: string;
  ruleName: string;
  channel: string;
  status: IncidentStatus;
  acknowledgedAt?: string;
  acknowledgedBy?: string;
  resolvedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface AlertDelivery {
  id: string;
  incidentId: string;
  channelId: string;
  status: string;
  attempt: number;
  error?: string;
  attemptedAt: string;
  deliveredAt?: string;
}

export interface JourneyStep {
  order: number;
  action: "navigate" | "click" | "fill" | "type" | "wait";
  selector?: string;
  value?: string;
  timeoutMs?: number;
  expect?: Record<string, unknown>;
}

export interface Journey {
  id: string;
  projectId: string;
  name: string;
  description?: string;
  owner?: string;
  baseUrl?: string;
  enabled: boolean;
  definition?: unknown;
  steps?: JourneyStep[];
  createdAt: string;
  updatedAt: string;
}

export interface JourneyInput {
  projectId?: string;
  name: string;
  description?: string;
  owner?: string;
  baseUrl?: string;
  enabled?: boolean;
  steps?: JourneyStep[];
}

export interface JourneyRun {
  id: string;
  journeyId: string;
  projectId: string;
  status: "running" | "passed" | "failed" | "canceled";
  stepsPassed: number;
  stepsTotal: number;
  durationMs: number;
  error?: string;
  report?: unknown;
  startedAt: string;
  finishedAt?: string;
  createdAt: string;
}

export interface PrivacyProfile {
  id: string;
  projectId: string;
  name: string;
  region: string;
  regulationIds: string[];
  config?: unknown;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface PrivacyProfileInput {
  projectId?: string;
  name: string;
  region?: string;
  regulationIds?: string[];
  config?: unknown;
  active?: boolean;
}

export interface PrivacyBatch {
  id: string;
  projectId: string;
  profileId?: string;
  startUrl: string;
  status: RunStatus;
  totalPages: number;
  completedPages: number;
  failedPages: number;
  concurrency: number;
  error?: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}

export type PrivacyCheckStatus = "pass" | "fail" | "skipped";

export interface PrivacyCheck {
  id: string;
  status: PrivacyCheckStatus;
  severity: Severity;
  message?: string;
  evidence?: Record<string, unknown>;
}

export interface PrivacyRegulationReport {
  regulation: string;
  score: number;
  passed: boolean;
  checks: PrivacyCheck[];
}

export interface PrivacyReport {
  schemaVersion: string;
  policyName: string;
  url: string;
  generatedAt: string;
  score: number;
  passed: boolean;
  summary: { regulations: number; passed: number; failed: number; violations: number; flags: number };
  states?: Array<{ state: string; blocked: boolean; cookiesSet?: string[]; thirdPartyHosts?: string[] }>;
  regulations: PrivacyRegulationReport[];
  flags?: Array<{ id: string; regulation?: string; severity: Severity; title: string; description?: string }>;
}

export interface PrivacyEvaluation {
  id: string;
  batchId: string;
  projectId: string;
  profileId?: string;
  url: string;
  status: RunStatus;
  report?: PrivacyReport;
  error?: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface ReportScope {
  type: "project" | "run";
  id?: string;
  label?: string;
}

export interface ReportTemplate {
  id: string;
  name: string;
  description: string;
  formats: string[];
}

export type ReportFormat = "json" | "csv" | "html";

export interface Report {
  id: string;
  projectId: string;
  templateId: string;
  name: string;
  scope: ReportScope;
  format: ReportFormat;
  status: string;
  owner: string;
  contentType?: string;
  sizeBytes: number;
  error?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ReportInput {
  projectId?: string;
  templateId: string;
  name?: string;
  format?: ReportFormat;
  scope?: ReportScope;
  owner?: string;
}

export interface ReportSchedule {
  id: string;
  projectId: string;
  templateId: string;
  name: string;
  cadence: string;
  cron: string;
  timezone: string;
  recipients: string[];
  scope: ReportScope;
  format: ReportFormat;
  enabled: boolean;
  nextRunAt?: string;
  lastRunAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ReportScheduleInput {
  projectId?: string;
  templateId: string;
  name?: string;
  cadence?: string;
  cron: string;
  timezone?: string;
  recipients?: string[];
  scope?: ReportScope;
  format?: ReportFormat;
  enabled?: boolean;
}

function projectQuery(projectId?: string, extra?: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  if (projectId) params.set("projectId", projectId);
  Object.entries(extra ?? {}).forEach(([key, value]) => {
    if (value !== undefined && value !== "") params.set(key, String(value));
  });
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

export async function getInventory(projectId?: string, runId?: string): Promise<InventorySnapshot> {
  return request<InventorySnapshot>(`/api/v1/inventory${projectQuery(projectId, { runId })}`);
}

export async function listSchedules(projectId?: string): Promise<Schedule[]> {
  return (await request<{ schedules: Schedule[] }>(`/api/v1/schedules${projectQuery(projectId)}`)).schedules ?? [];
}

export async function createSchedule(input: ScheduleInput): Promise<Schedule> {
  return request<Schedule>("/api/v1/schedules", { method: "POST", body: JSON.stringify(input) });
}

export async function updateSchedule(id: string, patch: Partial<ScheduleInput>): Promise<Schedule> {
  return request<Schedule>(`/api/v1/schedules/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
}

export async function deleteSchedule(id: string): Promise<void> {
  await request<void>(`/api/v1/schedules/${id}`, { method: "DELETE" });
}

export async function runScheduleNow(id: string): Promise<Run> {
  return (await request<{ run: Run }>(`/api/v1/schedules/${id}/run`, { method: "POST" })).run;
}

export async function listAlertRules(projectId?: string): Promise<AlertRule[]> {
  return (await request<{ rules: AlertRule[] }>(`/api/v1/alert-rules${projectQuery(projectId)}`)).rules ?? [];
}

export async function createAlertRule(input: AlertRuleInput): Promise<AlertRule> {
  return request<AlertRule>("/api/v1/alert-rules", { method: "POST", body: JSON.stringify(input) });
}

export async function updateAlertRule(id: string, patch: Partial<AlertRuleInput>): Promise<AlertRule> {
  return request<AlertRule>(`/api/v1/alert-rules/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
}

export async function deleteAlertRule(id: string): Promise<void> {
  await request<void>(`/api/v1/alert-rules/${id}`, { method: "DELETE" });
}

export async function listAlertChannels(projectId?: string): Promise<AlertChannel[]> {
  return (await request<{ channels: AlertChannel[] }>(`/api/v1/alert-channels${projectQuery(projectId)}`)).channels ?? [];
}

export async function createAlertChannel(input: AlertChannelInput): Promise<AlertChannel> {
  return request<AlertChannel>("/api/v1/alert-channels", { method: "POST", body: JSON.stringify(input) });
}

export async function updateAlertChannel(id: string, patch: Partial<AlertChannelInput>): Promise<AlertChannel> {
  return request<AlertChannel>(`/api/v1/alert-channels/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
}

export async function deleteAlertChannel(id: string): Promise<void> {
  await request<void>(`/api/v1/alert-channels/${id}`, { method: "DELETE" });
}

export async function testAlertChannel(id: string): Promise<AlertDelivery> {
  const res = await request<{ delivery: AlertDelivery }>(`/api/v1/alert-channels/${id}/test`, { method: "POST" });
  return res.delivery;
}

export async function listAlertIncidents(projectId?: string, limit = 100): Promise<AlertIncident[]> {
  return (await request<{ incidents: AlertIncident[] }>(`/api/v1/alert-incidents${projectQuery(projectId, { limit })}`)).incidents ?? [];
}

export async function updateAlertIncident(id: string, patch: { status: IncidentStatus; by?: string }): Promise<AlertIncident> {
  return request<AlertIncident>(`/api/v1/alert-incidents/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
}

export async function listAlertDeliveries(incidentId: string): Promise<AlertDelivery[]> {
  return (await request<{ deliveries: AlertDelivery[] }>(`/api/v1/alert-incidents/${incidentId}/deliveries`)).deliveries ?? [];
}

export async function listJourneys(projectId?: string): Promise<Journey[]> {
  return (await request<{ journeys: Journey[] }>(`/api/v1/journeys${projectQuery(projectId)}`)).journeys ?? [];
}

export async function getJourney(id: string): Promise<Journey> {
  return request<Journey>(`/api/v1/journeys/${id}`);
}

export async function createJourney(input: JourneyInput): Promise<Journey> {
  return request<Journey>("/api/v1/journeys", { method: "POST", body: JSON.stringify(input) });
}

export async function updateJourney(id: string, patch: Partial<JourneyInput>): Promise<Journey> {
  return request<Journey>(`/api/v1/journeys/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
}

export async function deleteJourney(id: string): Promise<void> {
  await request<void>(`/api/v1/journeys/${id}`, { method: "DELETE" });
}

export async function listJourneyRuns(journeyId: string, limit = 25): Promise<JourneyRun[]> {
  return (await request<{ runs: JourneyRun[] }>(`/api/v1/journeys/${journeyId}/runs${projectQuery(undefined, { limit })}`)).runs ?? [];
}

export async function runJourney(id: string): Promise<JourneyRun> {
  return (await request<{ run: JourneyRun }>(`/api/v1/journeys/${id}/runs`, { method: "POST" })).run;
}

export async function getJourneyRun(id: string): Promise<JourneyRun> {
  return request<JourneyRun>(`/api/v1/journey-runs/${id}`);
}

export async function listPrivacyProfiles(projectId?: string): Promise<PrivacyProfile[]> {
  return (await request<{ profiles: PrivacyProfile[] }>(`/api/v1/privacy/profiles${projectQuery(projectId)}`)).profiles ?? [];
}

export async function createPrivacyProfile(input: PrivacyProfileInput): Promise<PrivacyProfile> {
  return request<PrivacyProfile>("/api/v1/privacy/profiles", { method: "POST", body: JSON.stringify(input) });
}

export async function updatePrivacyProfile(id: string, patch: Partial<PrivacyProfileInput>): Promise<PrivacyProfile> {
  return request<PrivacyProfile>(`/api/v1/privacy/profiles/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
}

export async function deletePrivacyProfile(id: string): Promise<void> {
  await request<void>(`/api/v1/privacy/profiles/${id}`, { method: "DELETE" });
}

export async function listPrivacyBatches(projectId?: string, limit = 50): Promise<PrivacyBatch[]> {
  return (await request<{ batches: PrivacyBatch[] }>(`/api/v1/privacy/batches${projectQuery(projectId, { limit })}`)).batches ?? [];
}

export async function getPrivacyBatch(id: string): Promise<PrivacyBatch> {
  return request<PrivacyBatch>(`/api/v1/privacy/batches/${id}`);
}

export async function startPrivacyBatch(input: { projectId?: string; profileId: string; startUrl: string }): Promise<PrivacyBatch> {
  return request<PrivacyBatch>("/api/v1/privacy/batches", { method: "POST", body: JSON.stringify(input) });
}

export async function listPrivacyEvaluations(batchId: string, limit = 100): Promise<PrivacyEvaluation[]> {
  return (await request<{ evaluations: PrivacyEvaluation[] }>(`/api/v1/privacy/batches/${batchId}/evaluations${projectQuery(undefined, { limit })}`)).evaluations ?? [];
}

export async function listReportTemplates(): Promise<ReportTemplate[]> {
  return (await request<{ templates: ReportTemplate[] }>("/api/v1/report-templates")).templates ?? [];
}

export async function listReports(projectId?: string, limit = 50): Promise<Report[]> {
  return (await request<{ reports: Report[] }>(`/api/v1/reports${projectQuery(projectId, { limit })}`)).reports ?? [];
}

export async function createReport(input: ReportInput): Promise<Report> {
  return request<Report>("/api/v1/reports", { method: "POST", body: JSON.stringify(input) });
}

export async function getReport(id: string): Promise<Report> {
  return request<Report>(`/api/v1/reports/${id}`);
}

export async function deleteReport(id: string): Promise<void> {
  await request<void>(`/api/v1/reports/${id}`, { method: "DELETE" });
}

export function reportDownloadURL(id: string): string {
  return `${API_BASE}/api/v1/reports/${id}/download`;
}

export async function listReportSchedules(projectId?: string): Promise<ReportSchedule[]> {
  return (await request<{ schedules: ReportSchedule[] }>(`/api/v1/report-schedules${projectQuery(projectId)}`)).schedules ?? [];
}

export async function createReportSchedule(input: ReportScheduleInput): Promise<ReportSchedule> {
  return request<ReportSchedule>("/api/v1/report-schedules", { method: "POST", body: JSON.stringify(input) });
}

export async function updateReportSchedule(id: string, patch: Partial<ReportScheduleInput>): Promise<ReportSchedule> {
  return request<ReportSchedule>(`/api/v1/report-schedules/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
}

export async function deleteReportSchedule(id: string): Promise<void> {
  await request<void>(`/api/v1/report-schedules/${id}`, { method: "DELETE" });
}

export async function runReportScheduleNow(id: string): Promise<Report> {
  return request<Report>(`/api/v1/report-schedules/${id}/run`, { method: "POST" });
}
