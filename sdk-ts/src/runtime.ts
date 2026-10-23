import { readFileSync } from "node:fs";
import type { Finding, Observation, RuleSpec, Scope, ValidateRequest, ValidateResponse } from "./types.js";

/**
 * A validator function receives the immutable observation, the run scope, and
 * any rule specs declared in the plugin manifest (useful for configurable
 * plugins). It returns zero or more findings.
 */
export type ValidateFn = (
  observation: Observation,
  scope: Scope,
  rules: RuleSpec[],
) => Finding[] | Promise<Finding[]>;

function write(resp: ValidateResponse): void {
  process.stdout.write(JSON.stringify(resp));
}

/**
 * serveValidator runs a validator plugin to completion.
 *
 * The Go host invokes the process once per observation: it writes a single JSON
 * {@link ValidateRequest} to stdin (closing it) and reads a single JSON
 * {@link ValidateResponse} from stdout. Call this from your plugin entrypoint.
 *
 * @example
 * ```ts
 * import { serveValidator } from "@pageguard/plugin-sdk";
 * serveValidator((obs) => {
 *   if (obs.statusCode >= 400) {
 *     return [{ ruleId: "no-http-errors", severity: "high", title: "HTTP error" }];
 *   }
 *   return [];
 * });
 * ```
 */
export function serveValidator(validate: ValidateFn): void {
  const raw = readFileSync(0, "utf8");
  let req: ValidateRequest;
  try {
    req = JSON.parse(raw) as ValidateRequest;
  } catch (err) {
    write({ ok: false, error: `invalid request JSON: ${(err as Error).message}`, findings: [] });
    return;
  }

  if (req.type !== "validate") {
    write({ ok: false, error: `unsupported message type: ${String(req.type)}`, findings: [] });
    return;
  }

  Promise.resolve()
    .then(() => validate(req.observation, req.scope, req.rules ?? []))
    .then((findings) => write({ ok: true, findings: findings ?? [] }))
    .catch((err: unknown) => {
      const message = err instanceof Error ? err.message : String(err);
      write({ ok: false, error: message, findings: [] });
    });
}
