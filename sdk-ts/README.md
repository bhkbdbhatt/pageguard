# @pageguard/plugin-sdk

Write Pageguard validators in TypeScript.

## Quick start

```bash
mkdir my-plugin && cd my-plugin
npm init -y
npm i @pageguard/plugin-sdk
```

`manifest.json`:

```json
{
  "id": "my-plugin",
  "name": "My Plugin",
  "version": "0.1.0",
  "kind": "validator",
  "runtime": "node",
  "entry": "index.mjs"
}
```

`index.mjs`:

```js
import { serveValidator, finding } from "@pageguard/plugin-sdk";

serveValidator((obs) => {
  const out = [];
  if (obs.statusCode >= 400) {
    out.push(finding("my-plugin/status", "high", `HTTP ${obs.statusCode}`));
  }
  for (const r of obs.requests.filter((r) => r.thirdParty)) {
    out.push(finding("my-plugin/third-party", "info", `Third party: ${r.host}`));
  }
  return out;
});
```

Point the host at the directory:

```bash
pageguard --plugins ./plugins
```

## Protocol

The host invokes your process **once per page**. It writes a single JSON
`ValidateRequest` to stdin (then closes it) and reads a single JSON
`ValidateResponse` from stdout. `serveValidator` handles this for you.

## Helpers

`find.thirdPartyRequests`, `find.insecureRequests`, `find.insecureCookies`,
`find.hasTag`, and `finding(...)` are convenience utilities.
