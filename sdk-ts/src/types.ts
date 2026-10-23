/**
 * Public wire types shared with the Go host. These mirror
 * pkg/pluginsdk/plugin.go and are the stable contract versioned by
 * PROTOCOL_VERSION.
 */

export const PROTOCOL_VERSION = 1;

export type Kind = "validator" | "reporter" | "integration" | "detector";

export interface Manifest {
  id: string;
  name: string;
  version: string;
  kind: Kind;
  description?: string;
  author?: string;
  homepage?: string;
  license?: string;
  runtime: "builtin" | "node";
  entry?: string;
  rules?: RuleSpec[];
  permissions?: string[];
  configSchema?: Record<string, unknown>;
}

export interface NetworkRequest {
  method: string;
  url: string;
  host: string;
  status: number;
  resourceType: string;
  initiator?: string;
  requestHeaders?: Record<string, string>;
  responseHeaders?: Record<string, string>;
  durationMs?: number;
  sizeBytes?: number;
  thirdParty?: boolean;
}

export interface Cookie {
  name: string;
  domain: string;
  path: string;
  /** Raw values are never transmitted; only a short hash. */
  valueHash: string;
  secure: boolean;
  httpOnly: boolean;
  sameSite: string;
  expiresUnix?: number;
  session: boolean;
  firstParty: boolean;
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

export interface DataLayerEntry {
  name: string;
  value?: string;
}

export interface Observation {
  url: string;
  finalUrl: string;
  statusCode: number;
  contentType?: string;
  title: string;
  headers: Record<string, string>;
  html: string;
  htmlTruncated?: boolean;
  requests: NetworkRequest[];
  cookies: Cookie[];
  tags: Tag[];
  consoleErrors?: string[];
  /** Logical region the page was observed from. */
  geolocation?: string;
  /** Captured client-side data layer payloads. */
  dataLayer?: DataLayerEntry[];
  rendered: boolean;
  depth: number;
  pageId: string;
}

export interface Finding {
  ruleId: string;
  severity: "info" | "low" | "medium" | "high" | "critical";
  title: string;
  description?: string;
  evidence?: Record<string, unknown>;
}

export interface Scope {
  runId: string;
  projectId: string;
  scanId: string;
  config?: Record<string, unknown>;
}

export interface Predicate {
  field: string;
  name?: string;
  op: string;
  value?: unknown;
}

export interface Condition {
  all?: Predicate[];
  any?: Predicate[];
  none?: Predicate[];
}

export interface RuleSpec {
  id: string;
  title: string;
  description?: string;
  severity: string;
  category?: string;
  remediation?: string;
  condition: Condition;
}

export interface ValidateRequest {
  type: "validate";
  protocolVersion: number;
  observation: Observation;
  scope: Scope;
  rules?: RuleSpec[];
}

export interface ValidateResponse {
  ok: boolean;
  error?: string;
  findings: Finding[];
}
