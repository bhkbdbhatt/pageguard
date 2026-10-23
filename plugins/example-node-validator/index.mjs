// Example out-of-process validator.
//
// Protocol: the host writes a single JSON request to stdin and reads a single
// JSON response from stdout:
//   {"type":"validate","observation":{...},"scope":{...},"rules":[...]}
//   {"ok":true,"findings":[{...}]}
//
// This file is dependency-free so it runs with a bare `node`.

import { readFileSync } from "node:fs";

function main() {
  let req;
  try {
    req = JSON.parse(readFileSync(0, "utf8"));
  } catch (err) {
    return respond({ ok: false, error: `invalid request: ${err.message}`, findings: [] });
  }
  if (req.type !== "validate") {
    return respond({ ok: false, error: `unsupported type: ${req.type}`, findings: [] });
  }

  const obs = req.observation ?? {};
  const findings = [];

  const insecure = (obs.requests ?? []).filter((r) => r.url?.startsWith("http://"));
  if (insecure.length > 0) {
    findings.push({
      ruleId: "example-node.insecure-request",
      severity: "medium",
      title: "Plaintext HTTP request observed",
      description: `${insecure.length} request(s) were served over http://.`,
      evidence: { sample: insecure.slice(0, 5).map((r) => r.url) },
    });
  }

  const thirdParty = (obs.requests ?? []).filter((r) => r.thirdParty);
  if (thirdParty.length > 0) {
    const hosts = [...new Set(thirdParty.map((r) => r.host))];
    findings.push({
      ruleId: "example-node.third-parties",
      severity: "info",
      title: "Third-party requests detected",
      evidence: { count: thirdParty.length, hosts },
    });
  }

  if ((obs.consoleErrors ?? []).length > 0) {
    findings.push({
      ruleId: "example-node.console-errors",
      severity: "low",
      title: "Browser console errors during load",
      evidence: { errors: obs.consoleErrors.slice(0, 10) },
    });
  }

  respond({ ok: true, findings });
}

function respond(obj) {
  process.stdout.write(JSON.stringify(obj));
}

main();
