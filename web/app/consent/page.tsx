"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useApp } from "../../components/AppShell";
import { Icon } from "../../components/icons";
import { Card, CardHeader, EmptyState, ErrorBanner, Loading, StatusBadge } from "../../components/ui";
import { getConsent, listAudits, listPages, type AuditRow, type ConsentEntry, type ConsentMatrix, type Page } from "../../lib/api";
import { formatNumber, formatPercent, shortId } from "../../lib/format";

function ConsentInner() {
  const { project, loadingProjects } = useApp();
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryRun = searchParams.get("run");
  const queryUrl = searchParams.get("url");

  const [audits, setAudits] = useState<AuditRow[]>([]);
  const [runId, setRunId] = useState<string | null>(queryRun);
  const [pages, setPages] = useState<Page[]>([]);
  const [pageUrl, setPageUrl] = useState<string | null>(queryUrl);
  const [matrix, setMatrix] = useState<ConsentMatrix | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<"all" | "leaks" | "essential" | "nonessential">("all");
  const [query, setQuery] = useState("");

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
      return;
    }
    listPages(runId)
      .then((list) => {
        setPages(list);
        setPageUrl((current) => {
          if (current && list.some((p) => p.url === current)) return current;
          const scanned = list.find((p) => p.scannedAt) ?? list[0];
          return scanned?.url ?? null;
        });
      })
      .catch((err) => setError((err as Error).message));
  }, [runId]);

  useEffect(() => {
    if (!runId || !pageUrl) {
      setMatrix(null);
      return;
    }
    setLoadingDetail(true);
    setError(null);
    getConsent(runId, pageUrl)
      .then((m) => setMatrix(m))
      .catch((err) => setError((err as Error).message))
      .finally(() => setLoadingDetail(false));
  }, [runId, pageUrl]);

  const selectPage = useCallback(
    (url: string) => {
      setPageUrl(url);
      if (runId) router.replace(`/consent?run=${runId}&url=${encodeURIComponent(url)}`);
    },
    [router, runId],
  );

  const entries = useMemo(() => {
    const list = matrix?.entries ?? [];
    const q = query.trim().toLowerCase();
    return list.filter((e) => {
      if (filter === "leaks" && !e.leak) return false;
      if (filter === "essential" && !e.essential) return false;
      if (filter === "nonessential" && e.essential) return false;
      if (!q) return true;
      return [e.key, e.name, e.vendor, e.type, e.evidence].some((v) => String(v ?? "").toLowerCase().includes(q));
    });
  }, [matrix, filter, query]);

  const leakCount = matrix?.entries.filter((e) => e.leak).length ?? 0;

  if (loadingProjects) return <Loading label="Loading workspace" />;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-[0.25em] text-indigo-400">Pre-consent enforcement</p>
          <h1 className="mt-1 text-xl font-bold text-white sm:text-2xl">Consent & CMP Matrix</h1>
          <p className="mt-1 text-xs text-slate-500">
            Derived from captured pre-consent evidence. Interactions are not simulated in-browser for this matrix.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <select
            className="input h-9 max-w-[200px] py-1 text-xs"
            value={runId ?? ""}
            onChange={(e) => {
              const id = e.target.value || null;
              setRunId(id);
              setPageUrl(null);
              if (id) router.replace(`/consent?run=${id}`);
              else router.replace("/consent");
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
      <div className="rounded-lg border border-amber-800/50 bg-amber-950/30 px-4 py-2.5 text-xs text-amber-200">
        <strong>Limitation:</strong> The consent matrix is derived from captured pre-consent evidence, not browser consent interaction. Treat scores as coverage signals.
      </div>

      {!project ? (
        <Card><EmptyState icon="globe" title="No workspace yet" description="Create a project and run an audit to inspect consent coverage." /></Card>
      ) : !runId ? (
        <Card><EmptyState icon="radar" title="No audit selected" description="Select an audit run to view the CMP matrix." /></Card>
      ) : pages.length === 0 && !loadingDetail ? (
        <Card><EmptyState icon="search" title="No captured pages" description="This run has no resumable page evidence yet." /></Card>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-6">
            <Kpi label="Trackers" value={formatNumber(matrix?.trackers)} />
            <Kpi label="Beacons" value={formatNumber(matrix?.beacons)} />
            <Kpi label="Leaks" value={formatNumber(leakCount)} tone={leakCount > 0 ? "rose" : "slate"} />
            <Kpi label="Pre-consent score" value={formatPercent(matrix?.preConsentScore)} tone="amber" />
            <Kpi label="Post-consent score" value={formatPercent(matrix?.postConsentScore)} tone="emerald" />
            <Kpi label="CMP detected" value={matrix?.cmpDetected ? matrix.cmpVendor || "Yes" : "No"} />
          </div>

          <Card>
            <CardHeader
              title={`CMP matrix · ${pageUrl ? shortId(pageUrl) : "page"}`}
              subtitle={
                matrix
                  ? `Captured at ${matrix.generatedAt} · ${matrix.cmpDetected ? `CMP: ${matrix.cmpVendor}` : "No CMP identified"}`
                  : "Loading"
              }
              icon={<Icon name="cookie" className="h-4 w-4" />}
              right={
                <div className="flex flex-wrap items-center gap-2">
                  <select className="input h-8 w-32 py-1 text-xs" value={filter} onChange={(e) => setFilter(e.target.value as any)}>
                    <option value="all">All items</option>
                    <option value="leaks">Leaks only</option>
                    <option value="essential">Essential</option>
                    <option value="nonessential">Non-essential</option>
                  </select>
                  <div className="relative">
                    <Icon name="search" className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                    <input
                      className="input h-8 w-44 py-1 pl-8 text-xs sm:w-56"
                      placeholder="Search matrix…"
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                    />
                  </div>
                </div>
              }
            />
            <div className="overflow-x-auto">
              <table className="w-full min-w-[980px] text-xs">
                <thead className="table-head">
                  <tr>
                    <th className="px-4 py-2.5">Key</th>
                    <th className="px-4 py-2.5">Category</th>
                    <th className="px-4 py-2.5">Vendor / Name</th>
                    <th className="px-4 py-2.5">Type</th>
                    <th className="px-4 py-2.5">Essential</th>
                    <th className="px-4 py-2.5">Pre-consent</th>
                    <th className="px-4 py-2.5">Post-consent</th>
                    <th className="px-4 py-2.5">Leak</th>
                    <th className="px-4 py-2.5">Status</th>
                    <th className="px-4 py-2.5">Evidence</th>
                  </tr>
                </thead>
                <tbody>
                  {loadingDetail && <tr><td colSpan={10} className="px-4 py-8 text-center text-slate-500">Loading consent matrix…</td></tr>}
                  {!loadingDetail && entries.length === 0 && (
                    <tr>
                      <td colSpan={10} className="px-4 py-10 text-center">
                        <p className="text-slate-500">
                          {matrix?.entries.length === 0
                            ? "No trackers/cookies classified for this page under pre-consent capture."
                            : "No items match this filter."}
                        </p>
                      </td>
                    </tr>
                  )}
                  {!loadingDetail && entries.map((e, i) => (
                    <tr key={`${e.key}-${i}`} className="table-row">
                      <td className="mono px-4 py-2.5 text-slate-300">{e.key}</td>
                      <td className="px-4 py-2.5 text-slate-400">{e.category || "—"}</td>
                      <td className="px-4 py-2.5">
                        <div className="font-semibold text-slate-200">{e.name}</div>
                        <div className="text-slate-500">{e.vendor || "unknown"}</div>
                      </td>
                      <td className="px-4 py-2.5 text-slate-400">{e.type || "—"}</td>
                      <td className="px-4 py-2.5">
                        {e.essential ? <span className="pill border-sky-800/50 bg-sky-950/40 text-sky-300">essential</span> : <span className="pill">non-essential</span>}
                      </td>
                      <td className="px-4 py-2.5">{e.preConsent ? <span className="text-amber-300">present</span> : <span className="text-slate-600">—</span>}</td>
                      <td className="px-4 py-2.5">{e.postConsent ? <span className="text-emerald-300">present</span> : <span className="text-slate-600">—</span>}</td>
                      <td className="px-4 py-2.5">
                        {e.leak ? <span className="pill border-rose-800/60 bg-rose-950/50 text-rose-300">LEAK</span> : <span className="text-slate-600">—</span>}
                      </td>
                      <td className="px-4 py-2.5 text-slate-400">{e.status || "—"}</td>
                      <td className="max-w-[240px] px-4 py-2.5 text-slate-500">
                        <span className="mono block truncate" title={e.evidence}>{e.evidence || (e.cookies?.join(", ") || "—")}</span>
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

function Kpi({ label, value, tone = "slate" }: { label: string; value: string; tone?: "slate" | "rose" | "amber" | "emerald" }) {
  const colors = {
    slate: "text-white",
    rose: "text-rose-300",
    amber: "text-amber-300",
    emerald: "text-emerald-300",
  };
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/70 p-4">
      <p className="text-[10px] uppercase tracking-wide text-slate-500">{label}</p>
      <p className={`mt-1 truncate text-lg font-extrabold sm:text-xl ${colors[tone]}`}>{value}</p>
    </div>
  );
}

export default function ConsentPage() {
  return (
    <Suspense fallback={<Loading label="Loading consent matrix" />}>
      <ConsentInner />
    </Suspense>
  );
}
