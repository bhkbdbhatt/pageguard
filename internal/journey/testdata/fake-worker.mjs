// A deterministic stand-in for tools/browser-renderer used by the journey
// Playwright protocol tests. It implements the stateful session command set
// (open/step/screenshot/close) and returns fabricated page state.
import readline from "node:readline";

function write(obj) {
  process.stdout.write(JSON.stringify(obj) + "\n");
}

let stepCount = 0;

function stateFor(step) {
  stepCount++;
  const requested = step.value || "https://acme.test/";
  const state = {
    url: requested,
    finalUrl: requested,
    status: 200,
    title: `Fake ${stepCount}`,
    html:
      "<html><head>" +
      "<script src=\"https://connect.facebook.net/en_US/fbevents.js\"></script>" +
      "</head><body><div id=\"root\"></div></body></html>",
    requests: [
      {
        method: "GET",
        url: "https://connect.facebook.net/en_US/fbevents.js",
        host: "connect.facebook.net",
        status: 200,
        resourceType: "script",
      },
    ],
    cookies: [{ name: "sid", domain: "acme.test", firstParty: true }],
    console: [{ level: "log", text: "hello" }],
    dataLayer: [{ name: "dataLayer[0]", value: "{\"event\":\"page_view\"}" }],
    durationMs: 5,
  };
  if (step.action === "click") {
    state.finalUrl = "https://acme.test/product/1";
    state.url = state.finalUrl;
    state.dataLayer.push({
      name: "dataLayer[1]",
      value: "{\"event\":\"add_to_cart\"}",
    });
  }
  return state;
}

const rl = readline.createInterface({ input: process.stdin });
rl.on("line", (line) => {
  const trimmed = line.trim();
  if (!trimmed) return;
  let req;
  try {
    req = JSON.parse(trimmed);
  } catch (err) {
    write({ ok: false, error: err.message });
    return;
  }
  switch (req.cmd) {
    case "open":
      write({ ok: true });
      return;
    case "close":
      write({ ok: true });
      process.exit(0);
      return;
    case "screenshot":
      write({ ok: true, data: Buffer.from("PNG").toString("base64") });
      return;
    case "step":
      if ((req.step || {}).action === "explode") {
        write({ ok: false, error: "simulated step failure" });
        return;
      }
      write({ ok: true, state: stateFor(req.step || {}) });
      return;
    default:
      write({ ok: false, error: `unknown cmd: ${req.cmd}` });
  }
});
