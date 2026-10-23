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
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
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
