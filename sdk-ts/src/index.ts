export const PROTOCOL_VERSION = 1;

export * from "./types.js";
export { serveValidator } from "./runtime.js";
export type { ValidateFn } from "./runtime.js";

import type { Finding, Observation } from "./types.js";

/**
 * Convenience helpers for authoring rules without repeating boilerplate.
 */
export const find = {
  /** Requests whose host is not the page host. */
  thirdPartyRequests(obs: Observation) {
    return obs.requests.filter((r) => r.thirdParty);
  },
  /** Requests whose URL is plaintext HTTP. */
  insecureRequests(obs: Observation) {
    return obs.requests.filter((r) => r.url.startsWith("http://"));
  },
  /** Cookies missing the Secure flag. */
  insecureCookies(obs: Observation) {
    return obs.cookies.filter((c) => !c.secure);
  },
  /** Cookies missing HttpOnly. */
  nonHttpOnlyCookies(obs: Observation) {
    return obs.cookies.filter((c) => !c.httpOnly);
  },
  /** Whether a tag with the given id or name is present. */
  hasTag(obs: Observation, idOrName: string) {
    return obs.tags.some((t) => t.id === idOrName || t.name === idOrName);
  },
};

/** Build a finding with sensible defaults. */
export function finding(
  ruleId: string,
  severity: Finding["severity"],
  title: string,
  extra: Partial<Finding> = {},
): Finding {
  return { ruleId, severity, title, ...extra };
}
