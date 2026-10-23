"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useApp } from "../components/AppShell";
import { TrendChart, VendorDonut } from "../components/charts";
import { Icon } from "../components/icons";
import {
  Card,
  CardHeader,
  EmptyState,
  ErrorBanner,
  KpiCard,
  Loading,
  SeverityBadge,
} from "../components/ui";
import {
  getOverview,
  listScans,
  reportURL,
  type Overview,
  type Scan,
} from "../lib/api";
import { formatNumber, formatPercent, timeAgo } from "../lib/format";

export default function OverviewPage() {
  const { project, loadingProjects, openAudit } = useApp();
  const [overview, setOverview] = useState<Overview | null>(null);
  const [scans, setScans] = useState<Scan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [vendor, setVendor] = useState("all");

  const refresh = useCallback(async () => {
    if (!project) {
      setOverview(null);
      setLoading(false);
      return;
    }
    try {
      const [data, scanList] = await Promise.all([getOverview(project.id), listScans(project.id)]);
      setOverview(data);
      setScans(scanList);
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [project]);

  useEffect(() => {
    setLoading(true);
    refresh();
    const id = window.setInterval(refresh, 15000);
    return () => window.clearInterval(id);
  }, [refresh]);

  const vendors = useMemo(() => {
    const set = new Set<string>();
    overview?.vendors.forEach((v) => set.add(v.vendor));
    overview?.violations.forEach((v) => v.vendor && set.add(v.vendor));
    return [...set].sort();
  }, [overview]);

  const violations = useMemo(() => {
    const list = overview?.violations ?? [];
    const q = query.trim().toLowerCase();
    return list.filter((v) => {
      if (vendor !== "all" && v.vendor !== vendor) return false;
      if (!q) return true;
      return [v.title, v.description, v.vendor, v.pageUrl, v.ruleId]
        .some((v) => String(v ?? "").toLowerCase().includes(q));
    });
  }, [overview, query, vendor]);

  const latestURL = scans[0]?.startUrls?.[0];
  const kpi = overview?.kpi;

  if (loadingProjects) return <Loading label="Loading workspace" />;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-[0.25em] text-indigo-400">Command center</p>
          <h1 className="mt-1 text-xl font-bold text-white sm:text-2xl">Overview Dashboard</h1>
          <p className="mt-1 text-xs text-slate-500">
            {project ? `${project.name} · evidence from the latest completed audit` : "Create a project to start auditing"}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {overview?.latestRun && (
            <a className="btn h-9" href={reportURL(overview.latestRun.id, "html")} target="_blank" rel="noreferrer">
              <Icon name="download" className="h-3.5 w-3.5" />
              Export Governance Report
            </a>
          )}
          <button className="btn-primary h-9" onClick={() => openAudit({ url: latestURL })} disabled={!project}>
            <Icon name="refresh" className="h-3.5 w-3.5" />
            Re-scan site
          </button>
        </div>
      </div>

      <ErrorBanner message={error} />

      {!project ? (
        <Card>
          <EmptyState
            icon="globe"
            title="No workspace yet"
            description="Create a project to scope scans, governance rules and audit reports."
          />
        </Card>
      ) : !overview?.latestRun && !loading ? (
        <Card>
          <EmptyState
            icon="radar"
            title="No audits captured yet"
            description="Run your first crawl to populate beacon counts, validation scores and the consent matrix."
            action={
              <button className="btn-primary" onClick={() => openAudit({ url: latestURL })}>
                <Icon name="play" className="h-3.5 w-3.5" />
                Run first audit
              </button>
            }
          />
        </Card>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-5">
            <KpiCard
              label="Health score"
              value={formatPercent(kpi?.healthScore)}
              hint={`${formatNumber(kpi?.pagesValidated)} validated pages`}
              tone="indigo"
              icon="shield"
            />
            <KpiCard
              label="Captured beacons"
              value={formatNumber(kpi?.beacons)}
              hint="Tracker-like network requests"
              tone="sky"
              icon="activity"
            />
            <KpiCard
              label="Critical anomalies"
              value={formatNumber(kpi?.criticalFindings)}
              hint={`${formatNumber(kpi?.findings)} total findings`}
              tone="rose"
              icon="alert"
            />
            <KpiCard
              label="CMP compliance"
              value={formatPercent(kpi?.complianceScore)}
              hint={`${formatNumber(kpi?.consentLeaks)} pre-consent leak${kpi?.consentLeaks === 1 ? "" : "s"}`}
              tone="emerald"
              icon="cookie"
            />
            <KpiCard
              label="Pages audited"
              value={formatNumber(kpi?.pagesAudited)}
              hint={kpi?.pagesTarget ? `of ${formatNumber(kpi.pagesTarget)} crawl target` : "latest audit totals"}
              tone="amber"
              icon="radar"
            />
          </div>

          <div className="grid gap-4 xl:grid-cols-3">
            <Card className="xl:col-span-2">
              <CardHeader
                title="Governance health trend"
                subtitle="Daily average across completed audits · last 7 days"
                right={
                  overview?.latestRun && (
                    <Link href={`/runs/${overview.latestRun.id}`} className="btn">
                      Open run <Icon name="external" className="h-3 w-3" />
                    </Link>
                  )
                }
              />
              <div className="p-4">
                <TrendChart points={overview?.trend ?? []} />
              </div>
            </Card>
            <Card>
              <CardHeader title="Vendor signal distribution" subtitle="Tags and known tracker hosts" />
              <div className="p-5">
                <VendorDonut vendors={overview?.vendors ?? []} />
              </div>
            </Card>
          </div>

          <Card>
            <CardHeader
              title="Live violation feed"
              subtitle={`${overview?.violations.length ?? 0} findings in the latest run`}
              right={
                <div className="flex flex-wrap items-center gap-2">
                  <select className="input h-8 w-32 py-1 text-xs" value={vendor} onChange={(e) => setVendor(e.target.value)}>
                    <option value="all">All vendors</option>
                    {vendors.map((v) => <option key={v} value={v}>{v}</option>)}
                  </select>
                  <div className="relative">
                    <Icon name="search" className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                    <input
                      className="input h-8 w-40 py-1 pl-8 text-xs sm:w-56"
                      placeholder="Search violations…"
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                    />
                  </div>
                </div>
              }
            />
            <div className="overflow-x-auto">
              <table className="w-full min-w-[760px] text-xs">
                <thead className="table-head">
                  <tr>
                    <th className="px-4 py-2.5">Severity</th>
                    <th className="px-4 py-2.5">Rule</th>
                    <th className="px-4 py-2.5">Vendor</th>
                    <th className="px-4 py-2.5">Page</th>
                    <th className="px-4 py-2.5">Observed</th>
                    <th className="px-4 py-2.5 text-right">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {violations.length === 0 && (
                    <tr>
                      <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                        {overview?.violations.length === 0
                          ? "No violations in the latest run."
                          : "No violations match this filter."}
                      </td>
                    </tr>
                  )}
                  {violations.slice(0, 60).map((v) => (
                    <tr key={v.id} className="table-row">
                      <td className="px-4 py-2.5"><SeverityBadge severity={v.severity} /></td>
                      <td className="px-4 py-2.5">
                        <div className="font-semibold text-slate-200">{v.title}</div>
                        <div className="mono mt-0.5 text-[10px] text-slate-500">{v.ruleId}</div>
                      </td>
                      <td className="px-4 py-2.5 text-slate-400">{v.vendor || "—"}</td>
                      <td className="max-w-[260px] truncate px-4 py-2.5 text-slate-400" title={v.pageUrl}>
                        {v.pageUrl}
                      </td>
                      <td className="px-4 py-2.5 text-slate-500">{timeAgo(v.at)}</td>
                      <td className="px-4 py-2.5 text-right">
                        <Link
                          className="btn h-7 px-2 py-0.5"
                          href={`/inspector?run=${v.runId}&url=${encodeURIComponent(v.pageUrl)}`}
                        >
                          Inspect <Icon name="arrow" className="h-3 w-3" />
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>
        </>
      )}

      {loading && project && !overview && <Loading label="Loading governance metrics" />}
    </div>
  );
}
