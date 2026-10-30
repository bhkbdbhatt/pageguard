"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useApp } from "../../components/AppShell";
import { Icon } from "../../components/icons";
import {
  Card,
  CardHeader,
  EmptyState,
  ErrorBanner,
  Loading,
  StatusBadge,
} from "../../components/ui";
import {
  getBeacons,
  getScanResult,
  listAudits,
  listPages,
  type AuditRow,
  type BeaconReport,
  type DataLayerEntry,
  type Page,
  type ScanResult,
} from "../../lib/api";
import { bytes, duration, formatNumber, shortId, timeAgo } from "../../lib/format";

type Tab = "beacons" | "datalayer" | "cookies" | "console" | "raw";
type DataLayerView = "pushes" | "attributes";

const TABS: { id: Tab; label: string; icon: "activity" | "code" | "cookie" | "terminal" | "file" }[] = [
  { id: "beacons", label: "Beacon stream", icon: "activity" },
  { id: "datalayer", label: "DataLayer", icon: "code" },
  { id: "cookies", label: "Cookies", icon: "cookie" },
  { id: "console", label: "Console", icon: "terminal" },
  { id: "raw", label: "Raw evidence", icon: "file" },
];

function pretty(value: unknown) {
  if (value === undefined) return "—";
  if (typeof value === "string") return value;
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

// The worker serialises entry values to a JSON string (the schema stores the
// raw JSON of the variable as a string), so strings that look like JSON are
// decoded for display.
function decodeEntryValue(value: unknown) {
  if (typeof value === "string" && (value.startsWith("{") || value.startsWith("["))) {
    try {
      return JSON.parse(value);
    } catch {
      /* keep raw text */
    }
  }
  return value;
}

// layerBase reduces "dataLayer[3]" / "adobeDataLayer.getState" to the variable
// the entry belongs to, so entries can be grouped and filtered per system.
function layerBase(name: string) {
  return name.split("[")[0].split(".")[0];
}

function layerLabel(base: string) {
  switch (base) {
    case "dataLayer":
      return "GTM / Adobe Launch";
    case "adobeDataLayer":
      return "Adobe Launch (ACDL)";
    case "digitalData":
      return "Adobe DTM";
    case "utag_data":
      return "Tealium";
    case "_paq":
      return "Matomo";
    case "_satellite":
      return "Adobe Launch";
    default:
      return base || "custom layer";
  }
}

type AttributeRow = { layer: string; label: string; path: string; value: unknown };

// layerState merges the successive pushes of one layer into the data layer's
// final variable state (later pushes win), matching how tag managers resolve
// data layer variables at call time. The Adobe Client Data Layer's
// adobeDataLayer.getState() snapshot is an entry of its layer and merges in.
function layerState(entries: DataLayerEntry[]): Record<string, unknown> | null {
  let state: Record<string, unknown> | null = null;
  for (const entry of entries) {
    const decoded = decodeEntryValue(entry.value);
    if (decoded && typeof decoded === "object" && !Array.isArray(decoded)) {
      state = { ...(state ?? {}), ...(decoded as Record<string, unknown>) };
    }
  }
  return state;
}

// flattenValue turns nested state into dotted-path attribute rows, matching the
// variable naming used by tag managers (e.g. page.pageInfo.pageName).
function flattenValue(path: string, value: unknown, out: { path: string; value: unknown }[]) {
  if (Array.isArray(value)) {
    if (value.length === 0) {
      out.push({ path, value });
      return;
    }
    value.forEach((v, i) => flattenValue(`${path}[${i}]`, v, out));
    return;
  }
  if (value && typeof value === "object") {
    const fields = Object.entries(value);
    if (fields.length === 0) {
      out.push({ path, value });
      return;
    }
    for (const [k, v] of fields) flattenValue(path ? `${path}.${k}` : k, v, out);
    return;
  }
  out.push({ path, value });
}

function InspectorInner() {
  const { project, loadingProjects } = useApp();
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryRun = searchParams.get("run");
  const queryUrl = searchParams.get("url");

  const [audits, setAudits] = useState<AuditRow[]>([]);
  const [runId, setRunId] = useState<string | null>(queryRun);
  const [pages, setPages] = useState<Page[]>([]);
  const [pageUrl, setPageUrl] = useState<string | null>(queryUrl);
  const [result, setResult] = useState<ScanResult | null>(null);
  const [beacons, setBeacons] = useState<BeaconReport | null>(null);
  const [tab, setTab] = useState<Tab>("beacons");
  const [query, setQuery] = useState("");
  const [vendor, setVendor] = useState("all");
  const [layer, setLayer] = useState("all");
  const [dlView, setDlView] = useState<DataLayerView>("pushes");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState<string | null>(null);

  useEffect(() => {
    if (!project) {
      setAudits([]);
      setLoading(false);
      return;
    }
    listAudits(project.id)
      .then((rows) => {
        setAudits(rows);
        setRunId((current) => current ?? rows[0]?.runId ?? null);
      })
      .catch((err) => setError((err as Error).message))
      .finally(() => setLoading(false));
  }, [project]);

  useEffect(() => {
    if (queryRun) setRunId(queryRun);
    if (queryUrl) setPageUrl(queryUrl);
  }, [queryRun, queryUrl]);

  useEffect(() => {
    if (!runId) {
      setPages([]);
      setPageUrl(null);
      setResult(null);
      setBeacons(null);
      return;
    }
    setLoadingDetail(true);
    listPages(runId)
      .then((list) => {
        setPages(list);
        setPageUrl((current) => {
          if (current && list.some((p) => p.url === current)) return current;
          const scanned = list.find((p) => p.scannedAt) ?? list[0];
          return scanned?.url ?? null;
        });
      })
      .catch((err) => setError((err as Error).message))
      .finally(() => setLoadingDetail(false));
  }, [runId]);

  useEffect(() => {
    if (!runId || !pageUrl) {
      setResult(null);
      setBeacons(null);
      return;
    }
    let active = true;
    setLoadingDetail(true);
    setError(null);
    Promise.all([getScanResult(runId, pageUrl), getBeacons(runId, pageUrl)])
      .then(([evidence, beaconReport]) => {
        if (!active) return;
        setResult(evidence);
        setBeacons(beaconReport);
      })
      .catch((err) => active && setError((err as Error).message))
      .finally(() => active && setLoadingDetail(false));
    return () => {
      active = false;
    };
  }, [runId, pageUrl]);

  const selectPage = useCallback(
    (url: string) => {
      setPageUrl(url);
      setExpanded(null);
      setLayer("all");
      if (runId) router.replace(`/inspector?run=${runId}&url=${encodeURIComponent(url)}`);
    },
    [router, runId],
  );

  const selectedPage = pages.find((p) => p.url === pageUrl);
  const vendors = useMemo(() => Object.keys(beacons?.vendors ?? {}).sort(), [beacons]);

  const filteredBeacons = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (beacons?.beacons ?? []).filter((b) => {
      if (vendor !== "all" && b.vendor !== vendor) return false;
      if (!q) return true;
      return [b.url, b.host, b.vendor, b.method, b.resourceType].some((v) => v.toLowerCase().includes(q));
    });
  }, [beacons, query, vendor]);

  const dataLayers = useMemo(
    () => Array.from(new Set((result?.dataLayer ?? []).map((entry) => layerBase(entry.name)))).sort(),
    [result],
  );

  const filteredDataLayer = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (result?.dataLayer ?? []).filter((entry) => {
      if (layer !== "all" && layerBase(entry.name) !== layer) return false;
      if (!q) return true;
      return entry.name.toLowerCase().includes(q) || pretty(decodeEntryValue(entry.value)).toLowerCase().includes(q);
    });
  }, [result, query, layer]);

  // Final-state view: each capture layer's pushes are merged and flattened into
  // `path: value` attribute rows, like a Tag Manager debugger would show.
  const dataLayerAttributes = useMemo(() => {
    const q = query.trim().toLowerCase();
    const byLayer = new Map<string, DataLayerEntry[]>();
    for (const entry of result?.dataLayer ?? []) {
      const base = layerBase(entry.name);
      if (layer !== "all" && base !== layer) continue;
      const present = byLayer.get(base) ?? [];
      present.push(entry);
      byLayer.set(base, present);
    }
    return Array.from(byLayer.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .flatMap(([base, entries]) => {
        const state = layerState(entries);
        if (!state) return [];
        const rows: Omit<AttributeRow, "layer" | "label">[] = [];
        flattenValue("", state, rows);
        return rows
          .filter((row) => !q || row.path.toLowerCase().includes(q) || pretty(row.value).toLowerCase().includes(q))
          .map((row) => ({ ...row, layer: base, label: layerLabel(base) }));
      });
  }, [result, query, layer]);

  const filteredCookies = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (result?.cookies ?? []).filter((c) => {
      if (!q) return true;
      return [c.name, c.domain, c.sameSite].some((v) => v.toLowerCase().includes(q));
    });
  }, [result, query]);

  const filteredConsole = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (result?.console ?? []).filter((m) => {
      if (!q) return true;
      return m.level.toLowerCase().includes(q) || m.text.toLowerCase().includes(q);
    });
  }, [result, query]);

  const minStarted = useMemo(
    () => Math.min(...(beacons?.beacons ?? []).map((b) => b.startedAtMs).filter((n) => Number.isFinite(n)), Infinity),
    [beacons],
  );
  const relative = (ms: number) =>
    Number.isFinite(ms) && Number.isFinite(minStarted) ? `+${Math.max(0, (ms - minStarted) / 1000).toFixed(2)}s` : "—";

  async function copy(label: string, text: string) {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(label);
      window.setTimeout(() => setCopied(null), 1600);
    } catch {
      setError("Clipboard is unavailable in this browser.");
    }
  }

  if (loadingProjects) return <Loading label="Loading workspace" />;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-[0.25em] text-indigo-400">Evidence explorer</p>
          <h1 className="mt-1 text-xl font-bold text-white sm:text-2xl">Beacon & DataLayer Inspector</h1>
          <p className="mt-1 text-xs text-slate-500">Inspect captured network requests, data layer pushes, cookies and console output per page.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <select
            aria-label="Audit run"
            className="input h-9 max-w-[200px] py-1 text-xs"
            value={runId ?? ""}
            onChange={(e) => {
              const id = e.target.value || null;
              setRunId(id);
              setPageUrl(null);
              if (id) router.replace(`/inspector?run=${id}`);
              else router.replace("/inspector");
            }}
          >
            {audits.length === 0 && <option value="">No audit runs</option>}
            {audits.map((a) => (
              <option key={a.runId} value={a.runId}>
                {shortId(a.runId)} · {a.scanName}
              </option>
            ))}
          </select>
          <select
            aria-label="Page"
            className="input h-9 max-w-[280px] py-1 text-xs"
            value={pageUrl ?? ""}
            onChange={(e) => selectPage(e.target.value)}
            disabled={pages.length === 0}
          >
            {pages.length === 0 && <option value="">No captured pages</option>}
            {pages.map((p) => (
              <option key={p.id} value={p.url}>
                {p.statusCode || "ERR"} · {p.url}
              </option>
            ))}
          </select>
        </div>
      </div>

      <ErrorBanner message={error} />

      {!project ? (
        <Card><EmptyState icon="globe" title="No workspace yet" description="Create a project and run an audit to inspect evidence." /></Card>
      ) : !runId ? (
        <Card><EmptyState icon="radar" title="No audit selected" description="Start an audit to capture beacon and data layer evidence." /></Card>
      ) : pages.length === 0 && !loadingDetail ? (
        <Card><EmptyState icon="search" title="No captured pages" description="This run has no resumable page evidence yet." /></Card>
      ) : (
        <>
          <Card>
            <div className="flex flex-wrap items-center justify-between gap-3 p-4">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="mono truncate text-xs font-semibold text-slate-200">{result?.finalUrl || pageUrl || "Select a page"}</span>
                  {selectedPage && <StatusBadge status={(selectedPage.status ?? selectedPage.statusCode ?? 200) >= 400 || selectedPage.error ? "failed" : "succeeded"} />}
                  {result?.rendered && <span className="pill border-indigo-800/50 bg-indigo-950/40 text-indigo-300">JS rendered</span>}
                </div>
                <div className="mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-slate-500">
                  <span>{result?.title || selectedPage?.title || "Untitled page"}</span>
                  {result && <span>HTTP {result.statusCode}</span>}
                  {result && <span>{duration(result.durationMs)}</span>}
                  {selectedPage?.depth !== undefined && <span>depth {selectedPage.depth}</span>}
                  {selectedPage?.scannedAt && <span>captured {timeAgo(selectedPage.scannedAt)}</span>}
                </div>
              </div>
              <div className="flex items-center gap-1.5 text-[11px]">
                <button className="btn h-8" onClick={() => copy("evidence", pretty(result))} disabled={!result}>
                  <Icon name="copy" className="h-3 w-3" />
                  {copied === "evidence" ? "Copied" : "Copy JSON"}
                </button>
                <a
                  className="btn h-8"
                  href={`data:application/json;charset=utf-8,${encodeURIComponent(pretty(result))}`}
                  download={`evidence-${result?.pageId || "page"}.json`}
                >
                  <Icon name="download" className="h-3 w-3" /> Download
                </a>
              </div>
            </div>
            <div className="grid grid-cols-2 divide-x divide-slate-800 border-t border-slate-800 sm:grid-cols-4">
              <Metric label="Beacons" value={formatNumber(beacons?.count)} />
              <Metric label="Network requests" value={formatNumber(result?.requests?.length)} />
              <Metric label="DataLayer entries" value={formatNumber(result?.dataLayer?.length)} />
              <Metric label="DOM payload" value={result ? bytes(result.domBytes) : "—"} />
            </div>
          </Card>

          <Card>
            <div className="flex flex-col gap-3 border-b border-slate-800 p-3 lg:flex-row lg:items-center lg:justify-between">
              <div className="flex gap-1 overflow-x-auto">
                {TABS.map((t) => (
                  <button
                    key={t.id}
                    onClick={() => { setTab(t.id); setQuery(""); }}
                    className={`flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-[11px] font-semibold transition ${
                      tab === t.id ? "bg-indigo-950/60 text-indigo-200" : "text-slate-500 hover:text-slate-200"
                    }`}
                  >
                    <Icon name={t.icon} className="h-3.5 w-3.5" />
                    {t.label}
                  </button>
                ))}
              </div>
              {tab !== "raw" && (
                <div className="flex flex-wrap items-center gap-2">
                  {tab === "beacons" && (
                    <select className="input h-8 w-32 py-1 text-xs" value={vendor} onChange={(e) => setVendor(e.target.value)}>
                      <option value="all">All vendors</option>
                      {vendors.map((v) => <option key={v} value={v}>{v}</option>)}
                    </select>
                  )}
                  {tab === "datalayer" && (
                    <>
                      <select
                        aria-label="Data layer"
                        className="input h-8 w-44 py-1 text-xs"
                        value={layer}
                        onChange={(e) => setLayer(e.target.value)}
                      >
                        <option value="all">All layers</option>
                        {dataLayers.map((l) => <option key={l} value={l}>{layerLabel(l)}</option>)}
                      </select>
                      <div className="flex rounded-lg border border-slate-800 bg-slate-950/40 p-0.5">
                        {(["pushes", "attributes"] as const).map((v) => (
                          <button
                            key={v}
                            onClick={() => setDlView(v)}
                            className={`rounded-md px-2.5 py-0.5 text-[11px] font-semibold capitalize transition ${
                              dlView === v ? "bg-indigo-950/60 text-indigo-200" : "text-slate-500 hover:text-slate-200"
                            }`}
                          >
                            {v}
                          </button>
                        ))}
                      </div>
                    </>
                  )}
                  <div className="relative">
                    <Icon name="search" className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                    <input
                      className="input h-8 w-44 py-1 pl-8 text-xs sm:w-60"
                      placeholder={`Search ${tab}…`}
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                    />
                  </div>
                </div>
              )}
            </div>

            {loadingDetail && <div className="p-5"><Loading label="Loading page evidence" /></div>}

            {!loadingDetail && tab === "beacons" && (
              <div className="overflow-x-auto">
                <table className="w-full min-w-[860px] text-xs">
                  <thead className="table-head">
                    <tr>
                      <th className="px-4 py-2.5">Time</th>
                      <th className="px-4 py-2.5">Vendor</th>
                      <th className="px-4 py-2.5">Method</th>
                      <th className="px-4 py-2.5">Endpoint</th>
                      <th className="px-4 py-2.5">Type</th>
                      <th className="px-4 py-2.5 text-right">Status</th>
                      <th className="px-4 py-2.5 text-right">Latency</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredBeacons.length === 0 && (
                      <tr><td colSpan={7} className="px-4 py-10 text-center text-slate-500">No tracker beacons match this filter.</td></tr>
                    )}
                    {filteredBeacons.map((b, i) => (
                      <BeaconRows
                        key={`${b.url}-${i}`}
                        beacon={b}
                        time={relative(b.startedAtMs)}
                        open={expanded === `${b.url}-${i}`}
                        onToggle={() => setExpanded(expanded === `${b.url}-${i}` ? null : `${b.url}-${i}`)}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {!loadingDetail && tab === "datalayer" && dlView === "attributes" && (
              <div className="overflow-x-auto">
                <table className="w-full min-w-[760px] text-xs">
                  <thead className="table-head">
                    <tr>
                      <th className="w-44 px-4 py-2.5">Layer</th>
                      <th className="w-72 px-4 py-2.5">Attribute</th>
                      <th className="px-4 py-2.5">Value</th>
                    </tr>
                  </thead>
                  <tbody>
                    {dataLayerAttributes.length === 0 && (
                      <tr><td colSpan={3} className="px-4 py-10 text-center text-slate-500">No data layer attributes captured.</td></tr>
                    )}
                    {dataLayerAttributes.map((row, i) => (
                      <tr key={`${row.layer}-${row.path}-${i}`} className="table-row align-top">
                        <td className="mono px-4 py-2 text-indigo-300">{row.label}</td>
                        <td className="mono px-4 py-2 break-all text-slate-200">{row.path}</td>
                        <td className="px-4 py-2">
                          <pre className="mono whitespace-pre-wrap break-all text-[11px] text-slate-400">{pretty(row.value)}</pre>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {!loadingDetail && tab === "datalayer" && dlView === "pushes" && (
              <div className="overflow-x-auto">
                <table className="w-full min-w-[700px] text-xs">
                  <thead className="table-head">
                    <tr>
                      <th className="w-72 px-4 py-2.5">Event / key</th>
                      <th className="px-4 py-2.5">Value</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredDataLayer.length === 0 && (
                      <tr>
                        <td colSpan={2} className="px-4 py-10 text-center text-slate-500">
                          No data layer entries captured.
                          <p className="mt-1 text-[11px] text-slate-600">
                            Capture covers window.dataLayer (GTM / Adobe Launch), adobeDataLayer (Adobe Client Data Layer),
                            digitalData (Adobe DTM), utag_data (Tealium), _paq (Matomo) and any custom *layer* variables.
                          </p>
                        </td>
                      </tr>
                    )}
                    {filteredDataLayer.map((entry, i) => (
                      <tr key={`${entry.name}-${i}`} className="table-row align-top">
                        <td className="mono px-4 py-2.5 text-indigo-300">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <span className="pill border-indigo-800/50 bg-indigo-950/40 text-indigo-300">{layerLabel(layerBase(entry.name))}</span>
                            <span className="break-all">{entry.name}</span>
                          </div>
                        </td>
                        <td className="px-4 py-2.5">
                          <pre className="mono max-h-56 overflow-auto whitespace-pre-wrap break-all text-[11px] text-slate-400">{pretty(decodeEntryValue(entry.value))}</pre>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {!loadingDetail && tab === "cookies" && (
              <div className="overflow-x-auto">
                <table className="w-full min-w-[820px] text-xs">
                  <thead className="table-head">
                    <tr>
                      <th className="px-4 py-2.5">Name</th>
                      <th className="px-4 py-2.5">Domain</th>
                      <th className="px-4 py-2.5">Path</th>
                      <th className="px-4 py-2.5">SameSite</th>
                      <th className="px-4 py-2.5">Flags</th>
                      <th className="px-4 py-2.5">Party</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredCookies.length === 0 && (
                      <tr><td colSpan={6} className="px-4 py-10 text-center text-slate-500">No cookies captured.</td></tr>
                    )}
                    {filteredCookies.map((c) => (
                      <tr key={`${c.name}-${c.domain}-${c.path}`} className="table-row">
                        <td className="mono px-4 py-2.5 text-slate-200">{c.name}</td>
                        <td className="mono px-4 py-2.5 text-slate-400">{c.domain}</td>
                        <td className="mono px-4 py-2.5 text-slate-500">{c.path}</td>
                        <td className="px-4 py-2.5 text-slate-400">{c.sameSite || "—"}</td>
                        <td className="px-4 py-2.5">
                          <div className="flex flex-wrap gap-1">
                            {c.secure && <span className="pill">secure</span>}
                            {c.httpOnly && <span className="pill">httpOnly</span>}
                            {c.session && <span className="pill">session</span>}
                            {!c.secure && !c.httpOnly && !c.session && <span className="text-slate-600">—</span>}
                          </div>
                        </td>
                        <td className="px-4 py-2.5">
                          <span className={`pill ${c.firstParty ? "border-sky-800/50 bg-sky-950/40 text-sky-300" : "border-amber-800/50 bg-amber-950/40 text-amber-300"}`}>
                            {c.firstParty ? "first-party" : "third-party"}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {!loadingDetail && tab === "console" && (
              <div className="max-h-[520px] overflow-auto bg-slate-950/70 p-4">
                {filteredConsole.length === 0 && <p className="py-8 text-center text-xs text-slate-500">No console messages captured.</p>}
                <div className="space-y-1.5">
                  {filteredConsole.map((m, i) => (
                    <div key={i} className="mono flex gap-3 text-[11px] leading-relaxed">
                      <span className={`w-16 shrink-0 uppercase ${
                        m.level === "error" ? "text-rose-400" : m.level === "warning" || m.level === "warn" ? "text-amber-400" : "text-slate-600"
                      }`}>{m.level}</span>
                      <span className="whitespace-pre-wrap break-all text-slate-400">{m.text}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {!loadingDetail && tab === "raw" && (
              <div className="relative">
                <button className="btn absolute right-3 top-3 z-10 h-7 px-2 py-0.5" onClick={() => copy("raw", pretty(result))}>
                  <Icon name="copy" className="h-3 w-3" />
                  {copied === "raw" ? "Copied" : "Copy"}
                </button>
                <pre className="mono max-h-[560px] overflow-auto whitespace-pre-wrap break-all bg-slate-950/70 p-4 text-[11px] leading-relaxed text-slate-400">
                  {result ? pretty(result) : "No evidence selected."}
                </pre>
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="px-4 py-3">
      <div className="text-[10px] uppercase tracking-wide text-slate-500">{label}</div>
      <div className="mono mt-0.5 text-sm font-semibold text-slate-200">{value}</div>
    </div>
  );
}

function BeaconRows({
  beacon,
  time,
  open,
  onToggle,
}: {
  beacon: BeaconReport["beacons"][number];
  time: string;
  open: boolean;
  onToggle: () => void;
}) {
  const query = beacon.query ?? {};
  const entries = Object.entries(query);
  return (
    <>
      <tr className="table-row cursor-pointer" onClick={onToggle}>
        <td className="mono px-4 py-2.5 text-slate-500">{time}</td>
        <td className="px-4 py-2.5">
          <div className="flex items-center gap-2">
            <span className="pill border-indigo-800/50 bg-indigo-950/40 text-indigo-300">{beacon.vendor}</span>
            <span className="mono text-[10px] text-slate-500">{beacon.host}</span>
          </div>
        </td>
        <td className="mono px-4 py-2.5 text-slate-400">{beacon.method}</td>
        <td className="max-w-[360px] px-4 py-2.5">
          <span className="mono block truncate text-slate-300" title={beacon.url}>{beacon.url}</span>
        </td>
        <td className="px-4 py-2.5 text-slate-500">{beacon.resourceType}</td>
        <td className="px-4 py-2.5 text-right">
          <span className={`mono ${beacon.status >= 400 ? "text-rose-400" : "text-emerald-400"}`}>{beacon.status || "—"}</span>
        </td>
        <td className="px-4 py-2.5 text-right text-slate-500">{duration(beacon.durationMs)}</td>
      </tr>
      {open && (
        <tr>
          <td colSpan={7} className="bg-slate-900/60 px-4 py-3">
            <div className="grid gap-4 lg:grid-cols-2">
              <div>
                <p className="mb-1 text-[10px] uppercase tracking-wide text-slate-500">Request URL</p>
                <p className="mono break-all text-[11px] text-slate-300">{beacon.url}</p>
                <div className="mt-3 flex flex-wrap gap-2 text-[10px] text-slate-500">
                  {beacon.thirdParty && <span className="pill">third-party</span>}
                  <span className="pill">{bytes(beacon.sizeBytes)}</span>
                  <span className="pill">{beacon.resourceType}</span>
                </div>
              </div>
              <div>
                <p className="mb-1 text-[10px] uppercase tracking-wide text-slate-500">Query parameters</p>
                {entries.length === 0 ? (
                  <p className="text-[11px] text-slate-600">No query parameters.</p>
                ) : (
                  <div className="grid gap-1">
                    {entries.map(([k, v]) => (
                      <div key={k} className="flex gap-2 text-[11px]">
                        <span className="mono shrink-0 text-indigo-300">{k}:</span>
                        <span className="mono min-w-0 break-all text-slate-400">{v}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

export default function InspectorPage() {
  return (
    <Suspense fallback={<Loading label="Loading inspector" />}>
      <InspectorInner />
    </Suspense>
  );
}
