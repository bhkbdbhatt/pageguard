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
//   {"cmd":"open","opts":{...}}   -> {"ok":true}
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
    dataLayer = await page
      .evaluate(() => {
        const out = [];
        const safe = (v) => {
          try {
            return JSON.stringify(v);
          } catch {
            return null;
          }
        };
        if (Array.isArray(window.dataLayer)) {
          window.dataLayer.forEach((v, i) => {
            const s = safe(v);
            if (s !== null) out.push({ name: `dataLayer[${i}]`, value: s });
          });
        }
        if (window.digitalData && typeof window.digitalData === "object") {
          const s = safe(window.digitalData);
          if (s !== null) out.push({ name: "digitalData", value: s });
        }
        return out;
      })
      .catch(() => []);
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

  session = {
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
    dataLayer = await page
      .evaluate(() => {
        const out = [];
        const safe = (v) => {
          try {
            return JSON.stringify(v);
          } catch {
            return null;
          }
        };
        if (Array.isArray(window.dataLayer)) {
          window.dataLayer.forEach((v, i) => {
            const s = safe(v);
            if (s !== null) out.push({ name: `dataLayer[${i}]`, value: s });
          });
        }
        if (window.digitalData && typeof window.digitalData === "object") {
          const s = safe(window.digitalData);
          if (s !== null) out.push({ name: "digitalData", value: s });
        }
        return out;
      })
      .catch(() => []);
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
    case "open":
      await openSession(req.opts || {});
      write({ ok: true });
      return true;
    case "step": {
      const state = await performStep(req.step || {});
      write({ ok: true, state });
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
