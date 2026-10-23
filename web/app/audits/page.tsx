"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useApp } from "../../components/AppShell";
import { Icon } from "../../components/icons";
import {
  Card,
  CardHeader,
  EmptyState,
  ErrorBanner,
  KpiCard,
  Loading,
  StatusBadge,
} from "../../components/ui";
import {
  auditExportURL,
  createAudit,
  getRun,
  listAudits,
  reportURL,
  runEventStreamURL,
  type AuditRow,
  type Run,
  type RunStatus,
} from "../../lib/api";
import { duration, formatNumber, formatPercent, shortId, timeAgo } from "../../lib/format";

const TERMINAL = new Set<RunStatus>(["succeeded", "failed", "canceled"]);

function AuditsInner() {
  const { project, loadingProjects, openAudit } = useApp();
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryRun = searchParams.get("run");

  const [audits, setAudits] = useState<AuditRow[]>([]);
  const [activeId, setActiveId] = useState<string | null>(queryRun);
  const [run, setRun] = useState<Run | null>(null);
  const [log, setLog] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const [url, setUrl] = useState("");
  const [maxPages, setMaxPages] = useState("50");
  const [maxDepth, setMaxDepth] = useState("3");
  const [consent, setConsent] = useState("pre_consent");
  const [geo, setGeo] = useState("");

  const refreshAudits = useCallback(async () => {
    if (!project) {
      setAudits([]);
      setLoading(false);
      return;
    }
    const rows = await listAudits(project.id);
    setAudits(rows);
    setActiveId((current) => current ?? rows[0]?.runId ?? null);
    setLoading(false);
  }, [project]);

  useEffect(() => {
    setLoading(true);
    refreshAudits().catch((err) => {
      setError((err as Error).message);
      setLoading(false);
    });
    const id = window.setInterval(() => refreshAudits().catch(() => undefined), 12000);
    return () => window.clearInterval(id);
  }, [refreshAudits]);

  useEffect(() => {
    if (queryRun) setActiveId(queryRun);
  }, [queryRun]);

  useEffect(() => {
    if (!activeId) {
      setRun(null);
      return;
    }
    let active = true;
    const load = () =>
      getRun(activeId)
        .then((r) => active && setRun(r))
        .catch((err) => active && setError((err as Error).message));
    load();
    const id = window.setInterval(load, 5000);
    return () => {
      active = false;
      window.clearInterval(id);
    };
  }, [activeId]);

  useEffect(() => {
    if (!activeId) return;
    const es = new EventSource(runEventStreamURL(activeId));
    const types = [
      "run.snapshot", "run.queued", "run.started", "page.discovered", "page.started",
      "page.completed", "page.failed", "finding.created", "run.completed", "run.failed", "log",
    ];
    const handler = (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data);
        const message =
          data?.payload?.message ??
          (data?.payload?.url ? `${event.type.replaceAll(".", " ")} · ${data.payload.url}` : event.type);
        setLog((prev) => [...prev.slice(-160), message]);
      } catch {
        /* ignore malformed telemetry */
      }
    };
    for (const type of types) es.addEventListener(type, handler as EventListener);
    return () => es.close();
  }, [activeId]);

  async function startCrawl(e: React.FormEvent) {
    e.preventDefault();
    if (!project) return;
    setBusy(true);
    setError(null);
    try {
      const result = await createAudit({
        projectId: project.id,
        startUrl: url,
        maxPages: Number(maxPages) || 50,
        maxDepth: Number(maxDepth) || 3,
        consentSimulation: consent,
        geolocation: geo || undefined,
        renderMode: "auto",
      });
      setActiveId(result.run.id);
      setLog([`run.queued · ${result.run.id}`]);
      router.replace(`/audits?run=${result.run.id}`);
      await refreshAudits();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  const discovered = run?.stats.pagesDiscovered ?? 0;
  const done = (run?.stats.pagesScanned ?? 0) + (run?.stats.pagesFailed ?? 0);
  const terminal = run ? TERMINAL.has(run.status) : false;
  const percent = terminal ? 100 : Math.min(100, Math.round((done / Math.max(discovered || done, 1)) * 100));
  const selectedRow = useMemo(() => audits.find((a) => a.runId === activeId), [audits, activeId]);

  if (loadingProjects) return <Loading label="Loading workspace" />;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-[0.25em] text-indigo-400">Crawler operations</p>
          <h1 className="mt-1 text-xl font-bold text-white sm:text-2xl">Audit Runs & Crawler</h1>
          <p className="mt-1 text-xs text-slate-500">Launch crawls, follow live crawl telemetry and open persisted audit reports.</p>
        </div>
        <div className="flex gap-2">
          <a className="btn h-9" href={auditExportURL(project?.id)}>
            <Icon name="download" className="h-3.5 w-3.5" />
            Export CSV
          </a>
          <button className="btn-primary h-9" onClick={() => openAudit()} disabled={!project}>
            <Icon name="play" className="h-3.5 w-3.5" />
            New audit
          </button>
        </div>
      </div>

      <ErrorBanner message={error} />

      {!project ? (
        <Card><EmptyState icon="globe" title="No workspace yet" description="Create a project from the header before launching crawls." /></Card>
      ) : (
        <>
          <Card>
            <CardHeader
              title="Crawl configuration"
              subtitle="The crawler uses stored checkpoints, so resumes never re-fetch completed pages."
              icon={<Icon name="radar" className="h-4 w-4" />}
            />
            <form onSubmit={startCrawl} className="grid gap-3 p-5 md:grid-cols-2 xl:grid-cols-6">
              <div className="md:col-span-2 xl:col-span-2">
                <label className="label">Start URL</label>
                <input className="input" required placeholder="https://www.example.com/" value={url} onChange={(e) => setUrl(e.target.value)} />
              </div>
              <div>
                <label className="label">Max pages</label>
                <input className="input mono" type="number" min={1} value={maxPages} onChange={(e) => setMaxPages(e.target.value)} />
              </div>
              <div>
                <label className="label">Max depth</label>
                <input className="input mono" type="number" min={1} max={10} value={maxDepth} onChange={(e) => setMaxDepth(e.target.value)} />
              </div>
              <div>
                <label className="label">Consent</label>
                <select className="input" value={consent} onChange={(e) => setConsent(e.target.value)}>
                  <option value="pre_consent">Pre-consent</option>
                  <option value="accept_all">Accept all</option>
                  <option value="reject_all">Reject all</option>
                </select>
              </div>
              <div>
                <label className="label">Geo</label>
                <select className="input" value={geo} onChange={(e) => setGeo(e.target.value)}>
                  <option value="">Default</option>
                  <option value="EU">EU</option>
                  <option value="US">US</option>
                  <option value="UK">UK</option>
                </select>
              </div>
              <div className="flex items-end md:col-span-2 xl:col-span-6">
                <button type="submit" className="btn-primary h-9" disabled={busy || !url}>
                  <Icon name="play" className="h-3.5 w-3.5" />
                  {busy ? "Starting crawler…" : "Start crawl"}
                </button>
              </div>
            </form>
          </Card>

          {activeId && (
            <div className="grid gap-4 xl:grid-cols-3">
              <div className="space-y-4 xl:col-span-2">
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <KpiCard label="Pages discovered" value={formatNumber(discovered)} hint={`${done} completed`} tone="sky" icon="globe" />
                  <KpiCard label="Captured beacons" value={formatNumber(run?.stats.beacons)} hint="tracker requests" tone="indigo" icon="activity" />
                  <KpiCard label="Violations" value={formatNumber(run?.stats.findings)} hint={`${run?.stats.criticalFindings ?? 0} critical`} tone="rose" icon="alert" />
                  <KpiCard label="Health" value={formatPercent(run?.stats.healthScore)} hint={`CMP ${formatPercent(run?.stats.consentScore)}`} tone="emerald" icon="shield" />
                </div>

                <Card>
                  <CardHeader
                    title={`Live crawler · ${shortId(activeId)}`}
                    subtitle={selectedRow?.scanName || "Audit run progress"}
                    right={<div className="flex items-center gap-2">{run && <StatusBadge status={run.status} />}</div>}
                  />
                  <div className="space-y-4 p-5">
                    <div>
                      <div className="mb-2 flex items-center justify-between text-xs text-slate-400">
                        <span>{done} / {discovered || "?"} pages</span>
                        <span className="mono text-indigo-300">{percent}%</span>
                      </div>
                      <div className="h-2 overflow-hidden rounded-full bg-slate-800">
                        <div
                          className={`h-full rounded-full transition-all ${terminal ? "bg-emerald-500" : "bg-indigo-500"}`}
                          style={{ width: `${percent}%` }}
                        />
                      </div>
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-slate-500">
                        <span>{run?.stats.pagesScanned ?? 0} scanned</span>
                        <span>{run?.stats.pagesFailed ?? 0} failed</span>
                        <span>{run?.stats.findings ?? 0} findings</span>
                        {run?.startedAt && <span>started {timeAgo(run.startedAt)}</span>}
                        {run?.finishedAt && <span>finished {timeAgo(run.finishedAt)}</span>}
                      </div>
                    </div>
                    <div className="rounded-xl border border-slate-800 bg-slate-950/90 p-3">
                      <div className="mb-2 flex items-center gap-2 text-[10px] uppercase tracking-wider text-slate-600">
                        <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                        event stream
                      </div>
                      <pre className="mono max-h-44 min-h-[96px] overflow-auto whitespace-pre-wrap text-[11px] leading-relaxed text-slate-400">
                        {log.length ? log.join("\n") : "Waiting for crawl telemetry…"}
                      </pre>
                    </div>
                  </div>
                </Card>
              </div>

              <Card>
                <CardHeader title="Run controls" subtitle="Persisted checkpoints and reports" icon={<Icon name="file" className="h-4 w-4" />} />
                <div className="space-y-3 p-5 text-xs">
                  <Row label="Run ID" value={<span className="mono">{shortId(activeId)}</span>} />
                  <Row label="Scan" value={selectedRow?.scanName || "—"} />
                  <Row label="Start URL" value={<span className="mono break-all text-[11px]">{selectedRow?.startUrl || "—"}</span>} />
                  <Row label="Duration" value={duration(selectedRow?.durationSec)} />
                  <Row label="Pages scanned" value={formatNumber(run?.stats.pagesScanned)} />
                  <Row label="Pages failed" value={formatNumber(run?.stats.pagesFailed)} />
                  <div className="grid gap-2 pt-2">
                    <a className="btn justify-center" href={reportURL(activeId, "html")} target="_blank" rel="noreferrer">
                      <Icon name="file" className="h-3.5 w-3.5" /> Open governance report
                    </a>
                    <Link className="btn justify-center" href={`/inspector?run=${activeId}`}>
                      <Icon name="code" className="h-3.5 w-3.5" /> Inspect beacons
                    </Link>
                    <Link className="btn justify-center" href={`/consent?run=${activeId}`}>
                      <Icon name="cookie" className="h-3.5 w-3.5" /> Consent matrix
                    </Link>
                    <Link className="btn justify-center" href={`/runs/${activeId}`}>
                      <Icon name="external" className="h-3.5 w-3.5" /> Full evidence & sitemap
                    </Link>
                  </div>
                  {run?.error && <p className="rounded-lg border border-rose-800/60 bg-rose-950/40 px-3 py-2 text-rose-200">{run.error}</p>}
                </div>
              </Card>
            </div>
          )}

          <Card>
            <CardHeader
              title="Audit history"
              subtitle={`${audits.length} persisted run${audits.length === 1 ? "" : "s"}`}
              right={<span className="text-[10px] text-slate-500">click a row to inspect</span>}
            />
            <div className="overflow-x-auto">
              <table className="w-full min-w-[900px] text-xs">
                <thead className="table-head">
                  <tr>
                    <th className="px-4 py-2.5">Run</th>
                    <th className="px-4 py-2.5">Scan</th>
                    <th className="px-4 py-2.5">Status</th>
                    <th className="px-4 py-2.5 text-right">Pages</th>
                    <th className="px-4 py-2.5 text-right">Beacons</th>
                    <th className="px-4 py-2.5 text-right">Findings</th>
                    <th className="px-4 py-2.5 text-right">Health</th>
                    <th className="px-4 py-2.5 text-right">Started</th>
                    <th className="px-4 py-2.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {loading && (
                    <tr><td colSpan={9} className="px-4 py-8 text-center text-slate-500">Loading audit history…</td></tr>
                  )}
                  {!loading && audits.length === 0 && (
                    <tr><td colSpan={9} className="px-4 py-10 text-center text-slate-500">No audits yet. Start a crawl above.</td></tr>
                  )}
                  {!loading && audits.map((a) => (
                    <tr
                      key={a.runId}
                      onClick={() => setActiveId(a.runId)}
                      className={`table-row cursor-pointer ${a.runId === activeId ? "bg-indigo-950/20" : ""}`}
                    >
                      <td className="mono px-4 py-2.5 text-slate-300">{shortId(a.runId)}</td>
                      <td className="max-w-[220px] truncate px-4 py-2.5 text-slate-300" title={a.startUrl}>{a.scanName}</td>
                      <td className="px-4 py-2.5"><StatusBadge status={a.status} /></td>
                      <td className="px-4 py-2.5 text-right text-slate-400">{formatNumber(a.stats.pagesScanned)}</td>
                      <td className="px-4 py-2.5 text-right text-slate-400">{formatNumber(a.stats.beacons)}</td>
                      <td className="px-4 py-2.5 text-right text-slate-400">{formatNumber(a.stats.findings)}</td>
                      <td className="px-4 py-2.5 text-right font-semibold text-slate-200">{formatPercent(a.stats.healthScore)}</td>
                      <td className="px-4 py-2.5 text-right text-slate-500">{a.startedAt ? timeAgo(a.startedAt) : "—"}</td>
                      <td className="px-4 py-2.5 text-right">
                        <div className="flex justify-end gap-1.5">
                          <a
                            className="btn h-7 px-2 py-0.5"
                            href={reportURL(a.runId, "html")}
                            target="_blank"
                            rel="noreferrer"
                            onClick={(e) => e.stopPropagation()}
                          >
                            Report
                          </a>
                          <Link
                            className="btn h-7 px-2 py-0.5"
                            href={`/inspector?run=${a.runId}`}
                            onClick={(e) => e.stopPropagation()}
                          >
                            Inspect
                          </Link>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>
        </>
      )}
    </div>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3 border-b border-slate-800/70 pb-2">
      <span className="shrink-0 text-slate-500">{label}</span>
      <span className="min-w-0 text-right text-slate-300">{value}</span>
    </div>
  );
}

export default function AuditsPage() {
  return (
    <Suspense fallback={<Loading label="Loading audits" />}>
      <AuditsInner />
    </Suspense>
  );
}
