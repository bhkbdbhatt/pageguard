"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import {
  cancelRun,
  getRun,
  getScanResult,
  listFindings,
  listPages,
  listScanResults,
  resumeRun,
  runEventStreamURL,
  type Finding,
  type Page,
  type Run,
  type RunStatus,
  type ScanResult,
  type Severity,
} from "../../../lib/api";
import Sitemap from "../../../components/Sitemap";

const SEVERITY_ORDER = ["critical", "high", "medium", "low", "info"] as const;
const SEVERITY_CLASS: Record<string, string> = {
  critical: "bg-red-100 text-red-800",
  high: "bg-orange-100 text-orange-800",
  medium: "bg-amber-100 text-amber-800",
  low: "bg-sky-100 text-sky-800",
  info: "bg-slate-100 text-slate-700",
};
const STATUS_CLASS: Record<RunStatus, string> = {
  queued: "bg-slate-100 text-slate-700",
  running: "bg-sky-100 text-sky-800",
  succeeded: "bg-emerald-100 text-emerald-800",
  failed: "bg-red-100 text-red-800",
  canceled: "bg-amber-100 text-amber-800",
};

const TERMINAL = new Set<RunStatus>(["succeeded", "failed", "canceled"]);

function StatusBadge({ status }: { status: RunStatus }) {
  return (
    <span className={`rounded px-1.5 py-0.5 text-xs font-medium ${STATUS_CLASS[status] ?? STATUS_CLASS.queued}`}>
      {status}
    </span>
  );
}

function Meta({ label, value }: { label: string; value?: React.ReactNode }) {
  if (value === undefined || value === null || value === "") return null;
  return (
    <div>
      <dt className="text-xs uppercase tracking-wide text-slate-400">{label}</dt>
      <dd className="truncate text-slate-700" title={typeof value === "string" ? value : undefined}>
        {value}
      </dd>
    </div>
  );
}

function Section({ title, count, children }: { title: string; count?: number; children: React.ReactNode }) {
  return (
    <section>
      <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-500">
        {title}
        {count !== undefined && <span className="ml-1 text-slate-400">({count})</span>}
      </h3>
      {children}
    </section>
  );
}

function EvidencePanel({
  runId,
  pageKey,
  finding,
  onClose,
}: {
  runId: string;
  pageKey: string;
  finding?: Finding | null;
  onClose: () => void;
}) {
  const [result, setResult] = useState<ScanResult | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setErr(null);
    setResult(null);
    getScanResult(runId, pageKey)
      .then((r) => {
        if (!cancelled) setResult(r);
      })
      .catch((e) => {
        if (!cancelled) setErr((e as Error).message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [runId, pageKey]);

  const requests = result?.requests ?? [];
  const thirdParty = requests.filter((r) => r.thirdParty);
  const tags = result?.tags ?? [];
  const cookies = result?.cookies ?? [];
  const consoleMsgs = result?.console ?? [];

  return (
    <div className="fixed inset-0 z-40 flex justify-end">
      <div className="absolute inset-0 bg-slate-900/30" onClick={onClose} />
      <aside className="relative z-50 flex h-full w-full max-w-xl flex-col overflow-hidden border-l border-slate-200 bg-white shadow-xl">
        <header className="flex items-start justify-between gap-3 border-b border-slate-100 px-4 py-3">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold">Page evidence</h2>
            <p className="truncate text-xs text-slate-500" title={result?.finalUrl || pageKey}>
              {result?.finalUrl || pageKey}
            </p>
          </div>
          <button onClick={onClose} className="rounded border border-slate-300 px-2 py-1 text-xs text-slate-600 hover:bg-slate-50">
            Close
          </button>
        </header>

        <div className="flex-1 space-y-5 overflow-auto px-4 py-3 text-sm">
          {loading && <p className="text-slate-500">Loading evidence…</p>}
          {err && <p className="rounded bg-red-50 px-3 py-2 text-red-700">{err}</p>}
          {result && (
            <>
              {finding && (
                <Section title="Finding">
                  <div className="rounded border border-slate-200 p-3">
                    <div className="flex items-center gap-2">
                      <span className={`rounded px-1.5 py-0.5 text-xs font-medium ${SEVERITY_CLASS[finding.severity]}`}>
                        {finding.severity}
                      </span>
                      <span className="font-medium">{finding.title}</span>
                    </div>
                    <p className="mt-1 text-xs text-slate-500">
                      <code>{finding.ruleId}</code> · {finding.pluginId}
                    </p>
                    {finding.description && <p className="mt-2 text-slate-600">{finding.description}</p>}
                    {finding.evidence && Object.keys(finding.evidence).length > 0 && (
                      <pre className="mt-2 overflow-auto rounded bg-slate-50 p-2 text-xs text-slate-600">
                        {JSON.stringify(finding.evidence, null, 2)}
                      </pre>
                    )}
                  </div>
                </Section>
              )}

              {result.error ? (
                <p className="rounded bg-red-50 px-3 py-2 text-red-700">{result.error}</p>
              ) : (
                <Section title="Page">
                  <dl className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                    <Meta label="Status" value={result.statusCode} />
                    <Meta label="Rendered" value={result.rendered ? "yes" : "no"} />
                    <Meta label="Depth" value={result.depth} />
                    <Meta label="Duration" value={`${result.durationMs} ms`} />
                    <Meta label="Geolocation" value={result.geolocation} />
                    <Meta label="Proxy" value={result.proxyHost} />
                    <Meta label="Title" value={result.title} />
                    <Meta label="Content type" value={result.contentType} />
                    <Meta label="DOM" value={result.domTruncated ? `${result.domBytes} B (truncated)` : `${result.domBytes} B`} />
                  </dl>
                </Section>
              )}

              <Section title="Tags" count={tags.length}>
                {tags.length === 0 ? (
                  <p className="text-slate-400">None detected.</p>
                ) : (
                  <ul className="divide-y divide-slate-100 rounded border border-slate-200">
                    {tags.map((t) => (
                      <li key={t.id} className="flex items-center justify-between gap-2 px-2 py-1.5">
                        <span className="min-w-0 truncate">
                          <span className="font-medium">{t.name}</span>
                          <span className="text-slate-400"> · {t.vendor}</span>
                        </span>
                        <span className="shrink-0 text-xs text-slate-500">
                          {t.category}
                          {t.version ? ` v${t.version}` : ""}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </Section>

              <Section title="Network requests" count={requests.length}>
                <p className="mb-1 text-xs text-slate-500">
                  {thirdParty.length} third-party of {requests.length}
                </p>
                {requests.length === 0 ? (
                  <p className="text-slate-400">None recorded.</p>
                ) : (
                  <div className="overflow-hidden rounded border border-slate-200">
                    <table className="w-full text-xs">
                      <thead className="bg-slate-50 text-left text-slate-500">
                        <tr>
                          <th className="px-2 py-1">Host</th>
                          <th className="px-2 py-1">Type</th>
                          <th className="px-2 py-1 text-right">Status</th>
                          <th className="px-2 py-1 text-right">ms</th>
                        </tr>
                      </thead>
                      <tbody>
                        {requests.slice(0, 100).map((r, i) => (
                          <tr key={`${r.url}-${i}`} className="border-t border-slate-100">
                            <td className="max-w-[14rem] truncate px-2 py-1" title={r.url}>
                              {r.thirdParty && <span className="mr-1 text-amber-600" title="third-party">●</span>}
                              {r.host}
                            </td>
                            <td className="px-2 py-1 text-slate-500">{r.resourceType}</td>
                            <td className={`px-2 py-1 text-right ${r.status >= 400 ? "text-red-600" : "text-slate-500"}`}>
                              {r.status || "—"}
                            </td>
                            <td className="px-2 py-1 text-right text-slate-500">{r.durationMs}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {requests.length > 100 && (
                      <p className="px-2 py-1 text-xs text-slate-400">Showing first 100 of {requests.length}.</p>
                    )}
                  </div>
                )}
              </Section>

              <Section title="Cookies" count={cookies.length}>
                {cookies.length === 0 ? (
                  <p className="text-slate-400">None observed.</p>
                ) : (
                  <ul className="divide-y divide-slate-100 rounded border border-slate-200">
                    {cookies.map((c) => (
                      <li key={`${c.domain}-${c.name}`} className="flex items-center justify-between gap-2 px-2 py-1.5">
                        <span className="min-w-0 truncate font-medium" title={c.name}>
                          {c.name}
                        </span>
                        <span className="shrink-0 text-xs text-slate-500">
                          {c.firstParty ? "1st" : "3rd"}
                          {c.secure ? " · secure" : ""}
                          {c.httpOnly ? " · httpOnly" : ""}
                          {c.sameSite ? ` · ${c.sameSite}` : ""}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </Section>

              <Section title="Console" count={consoleMsgs.length}>
                {consoleMsgs.length === 0 ? (
                  <p className="text-slate-400">No messages.</p>
                ) : (
                  <ul className="space-y-1">
                    {consoleMsgs.map((m, i) => (
                      <li
                        key={i}
                        className={`rounded px-2 py-1 text-xs ${
                          m.level === "error" ? "bg-red-50 text-red-700" : m.level === "warning" ? "bg-amber-50 text-amber-700" : "bg-slate-50 text-slate-600"
                        }`}
                      >
                        <span className="mr-1 uppercase">{m.level}</span>
                        {m.text}
                      </li>
                    ))}
                  </ul>
                )}
              </Section>
            </>
          )}
        </div>
      </aside>
    </div>
  );
}

export default function RunPage() {
  const params = useParams<{ id: string }>();
  const runId = params.id;
  const [run, setRun] = useState<Run | null>(null);
  const [pages, setPages] = useState<Page[]>([]);
  const [findings, setFindings] = useState<Finding[]>([]);
  const [results, setResults] = useState<ScanResult[]>([]);
  const [log, setLog] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [actionBusy, setActionBusy] = useState(false);
  const [severityFilter, setSeverityFilter] = useState<Severity | "all">("all");
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<{ key: string; finding: Finding | null } | null>(null);

  const refresh = useCallback(async () => {
    try {
      const [r, p, f, res] = await Promise.all([
        getRun(runId),
        listPages(runId),
        listFindings(runId),
        listScanResults(runId),
      ]);
      setRun(r);
      setPages(p);
      setFindings(f);
      setResults(res);
      return r;
    } catch (err) {
      setError((err as Error).message);
      return null;
    }
  }, [runId]);

  useEffect(() => {
    let closed = false;
    refresh();

    const es = new EventSource(runEventStreamURL(runId));
    const types = [
      "run.snapshot",
      "run.queued",
      "run.started",
      "page.discovered",
      "page.started",
      "page.completed",
      "page.failed",
      "finding.created",
      "run.completed",
      "run.failed",
      "log",
    ];
    const handler = (event: MessageEvent) => {
      if (closed) return;
      try {
        const data = JSON.parse(event.data);
        const msg =
          data?.payload?.message ??
          (data?.payload?.url ? `${event.type}: ${data.payload.url}` : event.type);
        setLog((prev) => [...prev.slice(-199), msg]);
      } catch {
        /* ignore */
      }
      refresh();
    };
    for (const type of types) es.addEventListener(type, handler as EventListener);
    es.onerror = () => {
      /* EventSource auto-reconnects */
    };

    return () => {
      closed = true;
      es.close();
    };
  }, [runId, refresh]);

  async function onCancel() {
    setActionBusy(true);
    setError(null);
    try {
      await cancelRun(runId);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionBusy(false);
      refresh();
    }
  }

  async function onResume() {
    setActionBusy(true);
    setError(null);
    try {
      await resumeRun(runId);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionBusy(false);
      refresh();
    }
  }

  // Map a page's final URL back to its checkpoint key (the original URL) so
  // findings (which record the final URL) resolve to the right ScanResult.
  const urlKey = useMemo(() => {
    const m = new Map<string, string>();
    for (const p of pages) {
      m.set(p.url, p.url);
      if (p.finalUrl) m.set(p.finalUrl, p.url);
    }
    return m;
  }, [pages]);

  const severityCounts = SEVERITY_ORDER.map((sev) => ({
    sev,
    count: findings.filter((f) => f.severity === sev).length,
  }));

  const filteredFindings = useMemo(() => {
    const q = query.trim().toLowerCase();
    return findings.filter((f) => {
      if (severityFilter !== "all" && f.severity !== severityFilter) return false;
      if (!q) return true;
      return [f.title, f.ruleId, f.pluginId, f.pageUrl, f.description]
        .filter(Boolean)
        .some((v) => String(v).toLowerCase().includes(q));
    });
  }, [findings, severityFilter, query]);

  const discovered = run?.stats.pagesDiscovered ?? 0;
  const done = run ? run.stats.pagesScanned + run.stats.pagesFailed : 0;
  const terminal = run ? TERMINAL.has(run.status) : false;
  const denom = Math.max(discovered, done, 1);
  const percent = terminal ? 100 : Math.min(100, Math.round((done / denom) * 100));

  return (
    <div className="space-y-6">
      <section className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">Run {runId.slice(0, 8)}</h1>
          <p className="mt-1 flex items-center gap-2 text-sm text-slate-500">
            Status: {run ? <StatusBadge status={run.status} /> : "…"}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {run && !terminal && (
            <button
              onClick={onCancel}
              disabled={actionBusy}
              className="rounded border border-red-300 px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-50 disabled:opacity-50"
            >
              Cancel
            </button>
          )}
          {run && terminal && (
            <button
              onClick={onResume}
              disabled={actionBusy}
              className="rounded bg-brand-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
            >
              Resume
            </button>
          )}
        </div>
      </section>

      {run && (
        <section className="rounded-lg border border-slate-200 bg-white p-4">
          <div className="mb-1 flex items-center justify-between text-sm text-slate-600">
            <span>
              {done} / {discovered || done} pages
            </span>
            <span>{percent}%</span>
          </div>
          <div className="h-2 w-full overflow-hidden rounded-full bg-slate-100">
            <div
              className={`h-full rounded-full transition-all ${terminal ? "bg-emerald-500" : "bg-brand-600"}`}
              style={{ width: `${percent}%` }}
            />
          </div>
          <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
            <span>{run.stats.pagesScanned} scanned</span>
            <span>{run.stats.pagesFailed} failed</span>
            <span>{run.stats.findings} findings</span>
            {run.startedAt && <span>started {new Date(run.startedAt).toLocaleTimeString()}</span>}
            {run.finishedAt && <span>finished {new Date(run.finishedAt).toLocaleTimeString()}</span>}
          </div>
        </section>
      )}

      {error && <p className="rounded bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}

      <section className="grid grid-cols-2 gap-3 sm:grid-cols-6">
        <button
          onClick={() => setSeverityFilter("all")}
          className={`rounded-lg px-3 py-2 text-center ring-2 ${
            severityFilter === "all" ? "ring-slate-800" : "ring-transparent"
          } bg-slate-100 text-slate-700`}
        >
          <div className="text-lg font-semibold">{findings.length}</div>
          <div className="text-xs uppercase tracking-wide">all</div>
        </button>
        {severityCounts.map(({ sev, count }) => (
          <button
            key={sev}
            onClick={() => setSeverityFilter(severityFilter === sev ? "all" : sev)}
            className={`rounded-lg px-3 py-2 text-center ring-2 ${
              severityFilter === sev ? "ring-slate-800" : "ring-transparent"
            } ${SEVERITY_CLASS[sev]}`}
          >
            <div className="text-lg font-semibold">{count}</div>
            <div className="text-xs uppercase tracking-wide">{sev}</div>
          </button>
        ))}
      </section>

      <div className="grid gap-6 lg:grid-cols-2">
        <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
          <div className="flex items-center justify-between gap-2 border-b border-slate-100 px-4 py-2">
            <h2 className="text-sm font-semibold">Findings</h2>
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search findings…"
              className="w-40 rounded border border-slate-300 px-2 py-1 text-xs sm:w-52"
            />
          </div>
          <ul className="max-h-96 divide-y divide-slate-100 overflow-auto text-sm">
            {filteredFindings.length === 0 && (
              <li className="px-4 py-3 text-slate-500">
                {findings.length === 0 ? "No findings yet." : "No findings match the filter."}
              </li>
            )}
            {filteredFindings.map((f) => (
              <li key={f.id}>
                <button
                  onClick={() => setSelected({ key: urlKey.get(f.pageUrl) ?? f.pageUrl, finding: f })}
                  className="w-full px-4 py-2 text-left hover:bg-slate-50"
                >
                  <div className="flex items-center gap-2">
                    <span className={`rounded px-1.5 py-0.5 text-xs ${SEVERITY_CLASS[f.severity]}`}>{f.severity}</span>
                    <span className="font-medium">{f.title}</span>
                  </div>
                  <div className="mt-0.5 text-xs text-slate-500">
                    <code>{f.ruleId}</code> · {f.pageUrl}
                  </div>
                </button>
              </li>
            ))}
          </ul>
        </section>

        <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
          <h2 className="border-b border-slate-100 px-4 py-2 text-sm font-semibold">Pages</h2>
          <ul className="max-h-96 divide-y divide-slate-100 overflow-auto text-sm">
            {pages.length === 0 && <li className="px-4 py-3 text-slate-500">No pages yet.</li>}
            {pages.map((p) => (
              <li key={p.id}>
                <button
                  onClick={() => setSelected({ key: p.url, finding: null })}
                  className="flex w-full items-center justify-between gap-2 px-4 py-2 text-left hover:bg-slate-50"
                >
                  <span className="truncate">{p.url}</span>
                  <span className={`shrink-0 text-xs ${p.error || (p.status ?? 0) >= 400 ? "text-red-600" : "text-slate-400"}`}>
                    {p.error ? "error" : p.status ?? "—"}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      </div>

      <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
        <div className="flex items-center justify-between border-b border-slate-100 px-4 py-2">
          <h2 className="text-sm font-semibold">Sitemap</h2>
          <span className="text-xs text-slate-500">crawled link structure</span>
        </div>
        <div className="max-h-96 overflow-auto px-4 py-3">
          <Sitemap results={results} onSelect={(url) => setSelected({ key: url, finding: null })} />
        </div>
      </section>

      <section className="rounded-lg border border-slate-200 bg-white">
        <h2 className="border-b border-slate-100 px-4 py-2 text-sm font-semibold">Live log</h2>
        <pre className="max-h-48 overflow-auto px-4 py-2 text-xs text-slate-600">{log.join("\n")}</pre>
      </section>

      {run && TERMINAL.has(run.status) && (run.status === "failed" || run.status === "canceled") && (
        <p className="text-sm text-red-700">Run ended: {run.error || run.status}</p>
      )}

      {selected && (
        <EvidencePanel runId={runId} pageKey={selected.key} finding={selected.finding} onClose={() => setSelected(null)} />
      )}
    </div>
  );
}
