// Headless-browser worker for the Pageguard scanner.
//
// The Go scanner (internal/scanner/browser.go) spawns this process once and
// communicates over newline-delimited JSON:
//
//   request : {"url":"https://...","opts":{
//              "userAgent":"...","timeoutMs":20000,"waitMs":500,
//              "width":1366,"height":768,
//              "geolocation":"us","geoLatitude":..,"geoLongitude":..,"geoAccuracy":..,
//              "locale":"en-US","timezone":"America/New_York",
//              "proxyServer":"http://host:port","proxyUser":"..","proxyPass":"..",
//              "captureHar":true,"captureDataLayer":true}}
//   response: {"ok":true,"result":{...}}  |  {"ok":false,"error":"..."}
//
// The worker also implements a stateful "session" protocol used by the journey
// tester (internal/journey). A session keeps one browser context alive across
// steps so cookies, storage and the DOM persist:
//
//   {"cmd":"open","opts":{...}}   -> {"ok":true,"engine":"playwright",
//                                    "browser":"chromium","version":"..."}
//   {"cmd":"step","step":{...}}   -> {"ok":true,"state":{...}}
//   {"cmd":"screenshot"}          -> {"ok":true,"data":"<base64 png>"}
//   {"cmd":"close"}               -> {"ok":true}
//
// A request without a "cmd" field is treated as a legacy one-shot render.
//
// Enable it with:  pageguard --browser-cmd "node tools/browser-renderer/index.mjs"
//
// Install the browser once with:  npx playwright install chromium

import readline from "node:readline";
import { createHash } from "node:crypto";
import { promises as fs } from "node:fs";
import os from "node:os";
import path from "node:path";
import { chromium } from "playwright";

let browser;
// A persistent, stateful browser session used by the journey tester. Unlike a
// one-shot render it keeps cookies, storage and the DOM alive across steps.
let session = null;

async function getBrowser() {
  if (!browser) {
    browser = await chromium.launch({ headless: true, args: ["--no-sandbox"] });
  }
  return browser;
}

function hashValue(value) {
  return createHash("sha256").update(value ?? "").digest("hex").slice(0, 16);
}

function proxyConfig(opts) {
  if (!opts.proxyServer) return undefined;
  const proxy = { server: opts.proxyServer };
  if (opts.proxyUser) proxy.username = opts.proxyUser;
  if (opts.proxyPass) proxy.password = opts.proxyPass;
  return proxy;
}

function hasGeo(opts) {
  return typeof opts.geoLatitude === "number" && typeof opts.geoLongitude === "number";
}

function pageEvaluate(page, fn, arg) {
  return page
    .evaluate(fn, arg)
    .catch(() => null);
}

// --- client-side data layer capture ----------------------------------------
//
// Covers the data-layer conventions of the major tag management systems so the
// Inspector can surface them:
//
//   dataLayer       — Google Tag Manager (and Adobe Launch's default layer)
//   adobeDataLayer  — Adobe Client Data Layer (ACDL), callable object with
//                     getState(); this is what Adobe Launch properties expose
//   digitalData     — legacy Adobe DTM / IBM customer experience layer
//   utag_data       — Tealium iQ (utag.data mirrors the profile after load)
//   _paq            — Matomo
//   _mtm            — Matomo Tag Manager
//
// Adobe Launch (and CED setups generally) also allow custom data-layer names,
// so a generic window scan picks up array/object properties whose name
// mentions a "layer". Unknown names are only kept when the value carries
// layer content (a non-empty array of pushes, or an object with own keys);
// globals holding DOM nodes, functions, or UI-widget namespaces are skipped.
// Callable layers exposing a getState() function are captured through their
// merged state, matching the ACDL convention.
//
// Adobe runtime evidence is captured alongside the layers:
//   adobeDataLayer.getState() — merged ACDL state
//   _satellite.buildInfo      — Launch property build info
//   s (AppMeasurement)        — account/version/tracking server of the classic
//                               Adobe Analytics library, when it quacks like it
//   utag.data                 — Tealium iQ profile data
//
// Values are serialised to a JSON string (the ScanResult schema contract:
// value holds raw JSON of the variable as a string). Serialisation is cycle-
// safe via the ancestor chain: only objects on the current path are marked
// "[Circular]", while models that merely share a sub-object emit it once per
// reference. Depth, array width and entry count are capped so pathological
// content cannot stall the page or the stdio protocol.

// Runs inside the page context; arg = { names, max }.
function dataLayerEvaluate({ names, max }) {
  const MAX_DEPTH = 32;
  const MAX_ARRAY_ENTRIES = 512;
  const MAX_ENTRIES = 1024;

  const out = [];
  const seen = new Set();

  // Serialise one value to a bounded JSON string. ancestors is the set of
  // objects on the current path (added before descending, removed on the way
  // out), so only genuine cycles become "[Circular]". Returns null when the
  // value carries no JSON at all (functions, throwing getters, ...).
  const serialise = (v) => {
    const walk = (x, depth, ancestors) => {
      const t = typeof x;
      if (x === null) return "null";
      if (t === "string" || t === "number" || t === "boolean")
        return JSON.stringify(x); // NaN/Infinity -> null, like JSON.stringify
      if (t === "bigint") return JSON.stringify(x.toString());
      if (t !== "object") return undefined; // function, symbol, undefined
      if (x instanceof Date) return JSON.stringify(x);
      if (typeof Node !== "undefined" && x instanceof Node) return '"[Node]"';
      if (ancestors.has(x)) return '"[Circular]"';
      if (depth >= MAX_DEPTH) return '"[Depth]"';
      ancestors.add(x);
      if (Array.isArray(x)) {
        const n = Math.min(x.length, MAX_ARRAY_ENTRIES);
        const parts = [];
        for (let i = 0; i < n; i++) {
          const s = walk(x[i], depth + 1, ancestors);
          parts.push(s === undefined ? "null" : s);
        }
        if (n < x.length) parts.push('"[Truncated]"');
        ancestors.delete(x);
        return `[${parts.join(",")}]`;
      }
      const parts = [];
      for (const k of Object.keys(x)) {
        const s = walk(x[k], depth + 1, ancestors);
        if (s !== undefined) parts.push(`${JSON.stringify(k)}:${s}`);
      }
      ancestors.delete(x);
      return `{${parts.join(",")}}`;
    };
    let text;
    try {
      text = walk(v, 0, new Set());
    } catch {
      /* exotic getters can throw mid-walk */
    }
    if (typeof text !== "string") return null;
    return text.length > max ? text.slice(0, max) : text;
  };

  // hasContent gates the generic window scan: an empty array or object holds
  // no layer evidence, so unknown names are skipped until they hold something.
  // A genuinely unreadable value falls through to the serialiser, which is
  // guarded on its own.
  const hasContent = (v) => {
    try {
      return Array.isArray(v) ? v.length > 0 : Object.keys(v).length > 0;
    } catch {
      return true;
    }
  };

  const push = (name, value, unknown = false) => {
    if (value === null || value === undefined || out.length >= MAX_ENTRIES)
      return;
    if (unknown && !hasContent(value)) return;
    if (typeof value === "object" && value.nodeType) return; // DOM node
    if (Array.isArray(value)) {
      if (seen.has(name)) return;
      seen.add(name);
      // One entry per push, so the Inspector can replay layer events in order.
      const n = Math.min(value.length, MAX_ARRAY_ENTRIES);
      for (let i = 0; i < n; i++) {
        if (out.length >= MAX_ENTRIES) return;
        const s = serialise(value[i]);
        if (s !== null) out.push({ name: `${name}[${i}]`, value: s });
      }
      return;
    }
    if (typeof value !== "object") return;
    if (seen.has(name)) return;
    seen.add(name);
    const s = serialise(value);
    if (s !== null) out.push({ name, value: s });
  };

  for (const name of names) {
    let v;
    try {
      v = window[name];
    } catch {
      /* cross-origin / hardened globals may throw */
    }
    // Callable layers (Adobe ACDL) expose their merged state via getState();
    // the underlying event queue lives inside the callable object.
    let callable = false;
    try {
      callable = !!v && typeof v.getState === "function";
    } catch {
      /* opaque proxy */
    }
    if (callable) {
      try {
        const s = serialise(v.getState());
        if (s) out.push({ name: `${name}.getState`, value: s });
      } catch {
        /* getState may throw on hardened pages */
      }
      continue;
    }
    push(name, v);
  }

  // Tealium iQ: utag.data mirrors the loaded profile's data object.
  try {
    const utag = window.utag;
    const data = utag && utag.data;
    if (
      data &&
      typeof data === "object" &&
      !Array.isArray(data) &&
      Object.keys(data).length > 0
    ) {
      const s = serialise(data);
      if (s) out.push({ name: "utag.data", value: s });
    }
  } catch {
    /* ignore */
  }

  // Adobe Launch runtime evidence (build/env of the property).
  try {
    const buildInfo = window._satellite && window._satellite.buildInfo;
    if (buildInfo) {
      const s = serialise({ buildInfo });
      if (s) out.push({ name: "_satellite", value: s });
    }
  } catch {
    /* ignore */
  }

  // Adobe AppMeasurement (classic Analytics): "s" is such a common window
  // global that only treat it as AppMeasurement when it has both the track
  // function and an account string.
  try {
    const s = window.s;
    if (
      s &&
      typeof s === "object" &&
      typeof s.t === "function" &&
      typeof s.account === "string" &&
      s.account
    ) {
      const snap = serialise({
        account: s.account,
        version: s.version,
        trackingServer: s.trackingServer,
        trackingServerSecure: s.trackingServerSecure,
        currencyCode: s.currencyCode,
      });
      if (snap) out.push({ name: "s.tracking", value: snap });
    }
  } catch {
    /* ignore */
  }

  // Generic discovery of custom-named layers (covers Adobe Launch custom data
  // layer names, e.g. window.acdlDataLayer, window.analyticsLayer ...).
  let keys = [];
  try {
    keys = Object.keys(window);
  } catch {
    /* ignore */
  }
  for (const k of keys) {
    if (seen.has(k) || !/layer/i.test(k) || out.length >= MAX_ENTRIES) continue;
    let v;
    try {
      v = window[k];
    } catch {
      continue;
    }
    if (v === null || v === undefined) continue;
    if (typeof Node !== "undefined" && v instanceof Node) continue;
    let callable = false;
    try {
      callable = typeof v.getState === "function";
    } catch {
      /* opaque proxy */
    }
    if (callable) {
      try {
        const s = serialise(v.getState());
        if (s) {
          seen.add(k);
          out.push({ name: `${k}.getState`, value: s });
        }
      } catch {
        /* ignore */
      }
      continue;
    }
    if (typeof v !== "object" || v.nodeType) continue;
    push(k, v, true);
  }

  return out;
}

const DATA_LAYER_NAMES = ["dataLayer", "adobeDataLayer", "digitalData", "utag_data", "_paq", "_mtm"];
// Per-entry JSON cap; guards the stdio protocol against pathological content.
const DATA_LAYER_MAX_JSON = 1024 * 1024;

async function captureDataLayer(page) {
  if (!page) return [];
  const out = await pageEvaluate(page, dataLayerEvaluate, {
    names: DATA_LAYER_NAMES,
    max: DATA_LAYER_MAX_JSON,
  });
  return Array.isArray(out) ? out : [];
}

async function render(url, opts) {
  const b = await getBrowser();
  const harPath = opts.captureHar
    ? path.join(os.tmpdir(), `pageguard-${Date.now()}-${Math.random().toString(16).slice(2)}.har`)
    : null;

  const contextOptions = {
    userAgent: opts.userAgent || undefined,
    viewport: { width: opts.width || 1366, height: opts.height || 768 },
    ignoreHTTPSErrors: true,
  };
  const proxy = proxyConfig(opts);
  if (proxy) contextOptions.proxy = proxy;
  if (opts.locale) contextOptions.locale = opts.locale;
  if (opts.timezone) contextOptions.timezoneId = opts.timezone;
  if (hasGeo(opts)) {
    contextOptions.geolocation = {
      latitude: opts.geoLatitude,
      longitude: opts.geoLongitude,
      accuracy: opts.geoAccuracy || 50,
    };
    contextOptions.permissions = ["geolocation"];
    contextOptions.locale = contextOptions.locale || "en-US";
  }
  if (opts.captureHar) {
    contextOptions.recordHar = { path: harPath, content: "embed", mode: "full" };
  }

  const context = await b.newContext(contextOptions);

  const requests = [];
  const consoleMessages = [];

  context.on("request", (req) => {
    let host = "";
    try {
      host = new URL(req.url()).host;
    } catch {
      /* ignore */
    }
    requests.push({
      method: req.method(),
      url: req.url(),
      host,
      status: 0,
      resourceType: req.resourceType(),
      requestHeaders: safeHeaders(req.headers()),
    });
  });

  const page = await context.newPage();
  page.on("console", (msg) => consoleMessages.push({ level: msg.type(), text: msg.text() }));
  page.on("pageerror", (err) => consoleMessages.push({ level: "error", text: String(err) }));

  const started = Date.now();
  let response;
  try {
    response = await page.goto(url, {
      waitUntil: "networkidle",
      timeout: opts.timeoutMs || 20000,
    });
  } catch (err) {
    await context.close().catch(() => {});
    await cleanupHar(harPath);
    throw err;
  }
  if (opts.waitMs) await page.waitForTimeout(opts.waitMs);

  const html = await page.content();
  const title = await page.title();
  const finalUrl = page.url();

  const cookies = (await context.cookies()).map((c) => ({
    name: c.name,
    domain: c.domain,
    path: c.path,
    valueHash: hashValue(c.value),
    secure: c.secure,
    httpOnly: c.httpOnly,
    sameSite: c.sameSite || "unspecified",
    expiresUnix: c.expires ? Math.floor(c.expires) : 0,
    session: !c.expires,
    firstParty: true,
  }));

  let dataLayer = [];
  if (opts.captureDataLayer) {
    dataLayer = await captureDataLayer(page);
  }

  await context.close();

  const har = opts.captureHar ? await readHar(harPath) : null;

  return {
    finalUrl,
    status: response ? response.status() : 0,
    contentType: response ? (response.headers()["content-type"] || "") : "",
    html,
    title,
    requests,
    cookies,
    console: consoleMessages,
    dataLayer,
    har: har || undefined,
    durationMs: Date.now() - started,
  };
}

async function readHar(harPath) {
  if (!harPath) return null;
  try {
    const raw = await fs.readFile(harPath, "utf8");
    return JSON.parse(raw);
  } catch {
    return null;
  } finally {
    await cleanupHar(harPath);
  }
}

async function cleanupHar(harPath) {
  if (!harPath) return;
  await fs.unlink(harPath).catch(() => {});
}

function safeHeaders(headers) {
  const out = {};
  for (const [k, v] of Object.entries(headers || {})) {
    out[k] = String(v);
  }
  return out;
}

function write(obj) {
  process.stdout.write(JSON.stringify(obj) + "\n");
}

// --- stateful session (journey tester) -----------------------------------

async function openSession(opts) {
  const b = await getBrowser();
  const contextOptions = {
    userAgent: opts.userAgent || undefined,
    viewport: { width: opts.width || 1366, height: opts.height || 768 },
    ignoreHTTPSErrors: opts.ignoreHTTPSErrors !== false,
  };
  if (opts.locale) contextOptions.locale = opts.locale;
  if (opts.timezone) contextOptions.timezoneId = opts.timezone;
  if (opts.gpc) {
    contextOptions.extraHTTPHeaders = {
      ...(contextOptions.extraHTTPHeaders || {}),
      "Sec-GPC": "1",
      DNT: "1",
    };
  }

  const context = await b.newContext(contextOptions);
  if (opts.gpc) {
    await context.addInitScript(() => {
      try {
        Object.defineProperty(navigator, "globalPrivacyControl", { get: () => true });
        Object.defineProperty(navigator, "doNotTrack", { get: () => "1" });
      } catch {
        /* best effort */
      }
    });
  }
  const requests = [];
  const consoleMessages = [];

  context.on("request", (req) => {
    let host = "";
    try {
      host = new URL(req.url()).host;
    } catch {
      /* ignore */
    }
    requests.push({
      method: req.method(),
      url: req.url(),
      host,
      status: 0,
      resourceType: req.resourceType(),
      requestHeaders: safeHeaders(req.headers()),
    });
  });

  const page = await context.newPage();
  page.on("console", (msg) => consoleMessages.push({ level: msg.type(), text: msg.text() }));
  page.on("pageerror", (err) => consoleMessages.push({ level: "error", text: String(err) }));

  session = {
    browser: b,
    context,
    page,
    requests,
    consoleMessages,
    captureDataLayer: opts.captureDataLayer !== false,
    lastStatus: 0,
  };
  return session;
}

async function performStep(step) {
  if (!session) throw new Error("no open session");
  const { page } = session;
  const timeout = step.timeoutMs || 20000;
  const requested = step.value || page.url();

  switch (step.action) {
    case "navigate": {
      const resp = await page.goto(step.value, { waitUntil: "networkidle", timeout });
      session.lastStatus = resp ? resp.status() : 0;
      break;
    }
    case "click":
      await page.click(step.selector, { timeout });
      break;
    case "fill":
    case "type":
      await page.fill(step.selector, step.value || "", { timeout });
      break;
    case "wait":
      if (step.selector) await page.waitForSelector(step.selector, { timeout });
      else await page.waitForTimeout(step.timeoutMs || 1000);
      break;
    default:
      throw new Error(`unknown action: ${step.action}`);
  }
  if (step.waitMs) await page.waitForTimeout(step.waitMs);

  return collectState(requested);
}

async function collectState(requested) {
  const { page, context, requests, consoleMessages, captureDataLayer, lastStatus } = session;
  const started = Date.now();
  const html = await page.content();
  const title = await page.title();
  const finalUrl = page.url();

  const cookies = (await context.cookies()).map((c) => ({
    name: c.name,
    domain: c.domain,
    path: c.path,
    valueHash: hashValue(c.value),
    secure: c.secure,
    httpOnly: c.httpOnly,
    sameSite: c.sameSite || "unspecified",
    expiresUnix: c.expires ? Math.floor(c.expires) : 0,
    session: !c.expires,
    firstParty: true,
  }));

  let dataLayer = [];
  if (captureDataLayer) {
    dataLayer = await captureDataLayer(page);
  }

  return {
    url: requested,
    finalUrl,
    status: lastStatus,
    title,
    html,
    requests,
    cookies,
    console: consoleMessages,
    dataLayer,
    durationMs: Date.now() - started,
  };
}

// consent clicks the first matching consent control, best-effort. It never
// fails when no selector matches so a privacy evaluation can still observe the
// page as rendered.
async function consentSession(req) {
  if (!session) throw new Error("no open session");
  const { page } = session;
  const timeout = req.timeoutMs || 1500;
  let clicked = "";
  for (const selector of req.selectors || []) {
    if (!selector) continue;
    try {
      const el = await page.waitForSelector(selector, { timeout, state: "visible" });
      if (!el) continue;
      await el.click({ timeout });
      clicked = selector;
      break;
    } catch {
      /* try the next selector */
    }
  }
  if (req.waitMs) await page.waitForTimeout(req.waitMs);
  const state = await collectState(page.url());
  return { clicked, state };
}

async function screenshotSession() {
  if (!session) throw new Error("no open session");
  const buf = await session.page.screenshot({ type: "png" });
  return buf.toString("base64");
}

async function closeSession() {
  if (!session) return;
  const { context } = session;
  session = null;
  await context.close().catch(() => {});
}

async function handleCommand(req) {
  switch (req.cmd) {
    case "open": {
      const opened = await openSession(req.opts || {});
      write({
        ok: true,
        engine: "playwright",
        browser: "chromium",
        version: opened.browser.version(),
      });
      return true;
    }
    case "step": {
      const state = await performStep(req.step || {});
      write({ ok: true, state });
      return true;
    }
    case "consent": {
      const result = await consentSession(req);
      write({ ok: true, state: result.state, clicked: result.clicked });
      return true;
    }
    case "screenshot": {
      const data = await screenshotSession();
      write({ ok: true, data });
      return true;
    }
    case "close":
      await closeSession();
      write({ ok: true });
      return true;
    default:
      write({ ok: false, error: `unknown cmd: ${req.cmd}` });
      return true;
  }
}

const rl = readline.createInterface({ input: process.stdin });
rl.on("line", async (line) => {
  const trimmed = line.trim();
  if (!trimmed) return;
  let req;
  try {
    req = JSON.parse(trimmed);
  } catch (err) {
    write({ ok: false, error: `invalid request JSON: ${err.message}` });
    return;
  }
  try {
    if (req.cmd) {
      await handleCommand(req);
      return;
    }
    const result = await render(req.url, req.opts || {});
    write({ ok: true, result });
  } catch (err) {
    write({ ok: false, error: String(err && err.message ? err.message : err) });
  }
});

process.on("SIGTERM", shutdown);
process.on("SIGINT", shutdown);
async function shutdown() {
  if (session) await session.context.close().catch(() => {});
  if (browser) await browser.close().catch(() => {});
  process.exit(0);
}
