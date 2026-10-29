"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useApp } from "../../components/AppShell";
import { Icon } from "../../components/icons";
import {
  Card,
  CardHeader,
  EmptyState,
  ErrorBanner,
  KpiCard,
  Loading,
  Modal,
  PageHeader,
  Pill,
  StatusBadge,
  Toggle,
} from "../../components/ui";
import {
  createSchedule,
  deleteSchedule,
  listSchedules,
  runScheduleNow,
  updateSchedule,
  type Schedule,
} from "../../lib/api";
import { cronLabel, formatDate, formatNumber, timeAgo } from "../../lib/format";

const CADENCE_PRESETS = [
  { value: "0 6 * * *", label: "Daily - 06:00" },
  { value: "30 8 * * 1-5", label: "Weekdays - 08:30" },
  { value: "0 * * * *", label: "Hourly" },
  { value: "*/15 * * * *", label: "Every 15 minutes" },
  { value: "0 6 * * 1", label: "Weekly - Mon 06:00" },
  { value: "0 2 1 * *", label: "Monthly - 1st 02:00" },
];

const TIMEZONES = ["UTC", "Europe/London", "Europe/Berlin", "America/New_York", "Asia/Singapore"];

const CONSENT_OPTIONS = [
  { value: "pre_consent", label: "Pre-consent snapshot" },
  { value: "accept_all", label: "Accept all" },
  { value: "reject_all", label: "Reject all" },
  { value: "necessary_only", label: "Necessary only" },
];

export default function SchedulesPage() {
  const { project, loadingProjects } = useApp();
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [cadence, setCadence] = useState(CADENCE_PRESETS[0].value);
  const [timezone, setTimezone] = useState("UTC");
  const [pages, setPages] = useState("120");
  const [depth, setDepth] = useState("3");
  const [consent, setConsent] = useState("pre_consent");

  const load = useCallback(async () => {
    if (!project) {
      setSchedules([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      setSchedules(await listSchedules(project.id));
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [project]);

  useEffect(() => {
    void load();
  }, [load]);

  const active = useMemo(() => schedules.filter((schedule) => schedule.enabled), [schedules]);
  const failed = schedules.filter((schedule) => schedule.lastStatus === "failed").length;
  const next = useMemo(() => {
    const upcoming = active
      .map((schedule) => schedule.nextRunAt ? new Date(schedule.nextRunAt).getTime() : Number.MAX_SAFE_INTEGER)
      .sort((a, b) => a - b);
    return upcoming[0] && upcoming[0] !== Number.MAX_SAFE_INTEGER ? new Date(upcoming[0]).toISOString() : undefined;
  }, [active]);

  async function toggleSchedule(schedule: Schedule, enabled: boolean) {
    setBusy(schedule.id);
    try {
      const updated = await updateSchedule(schedule.id, { enabled });
      setSchedules((current) => current.map((item) => (item.id === schedule.id ? updated : item)));
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function runNow(schedule: Schedule) {
    setBusy(schedule.id);
    try {
      const run = await runScheduleNow(schedule.id);
      setNotice(`Started run ${run.id.slice(0, 8)} for ${schedule.name}.`);
      setError(null);
      window.setTimeout(() => void load(), 1500);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function removeSchedule(schedule: Schedule) {
    if (!window.confirm(`Delete schedule "${schedule.name}"?`)) return;
    setBusy(schedule.id);
    try {
      await deleteSchedule(schedule.id);
      setSchedules((current) => current.filter((item) => item.id !== schedule.id));
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function createNew(event: React.FormEvent) {
    event.preventDefault();
    if (!project) return;
    setBusy("create");
    try {
      const created = await createSchedule({
        projectId: project.id,
        name: name.trim(),
        url: url.trim(),
        cron: cadence,
        timezone,
        maxPages: Number(pages) || 50,
        maxDepth: Number(depth) || 3,
        consent,
        enabled: true,
      });
      setSchedules((current) => [created, ...current]);
      setCreateOpen(false);
      setName("");
      setUrl("");
      setCadence(CADENCE_PRESETS[0].value);
      setTimezone("UTC");
      setPages("120");
      setDepth("3");
      setConsent("pre_consent");
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  if (!project && !loadingProjects) {
    return (
      <EmptyState
        icon="calendar"
        title="No project selected"
        description="Create a project and run an audit before scheduling recurring crawls."
      />
    );
  }

  return (
    <div className="space-y-5">
      <PageHeader
        eyebrow="Automation"
        title="Scheduled Audits"
        subtitle="Keep governance evidence fresh with recurring crawls across every site, region and consent state."
        icon="calendar"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)} disabled={!project}>
            <Icon name="plus" className="h-3.5 w-3.5" />
            New schedule
          </button>
        }
      />

      <ErrorBanner message={error} />
      {notice && (
        <div className="flex items-center justify-between rounded-xl border border-emerald-900/60 bg-emerald-950/30 px-3 py-2 text-xs text-emerald-200">
          <span>{notice}</span>
          <button className="text-emerald-300" onClick={() => setNotice(null)} aria-label="Dismiss"><Icon name="x" className="h-3.5 w-3.5" /></button>
        </div>
      )}

      {loading ? (
        <Loading label="Loading schedules" />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
            <KpiCard label="Active schedules" value={`${active.length}/${schedules.length}`} hint="paused schedules excluded" tone="indigo" icon="calendar" />
            <KpiCard label="Next run" value={next ? timeAgo(next) : "None"} hint={next ? formatDate(next) : "no upcoming executions"} tone="sky" icon="clock" />
            <KpiCard label="Total schedules" value={formatNumber(schedules.length)} hint="recurring crawl definitions" tone="emerald" icon="play" />
            <KpiCard label="Failing schedules" value={formatNumber(failed)} hint="last completed run" tone={failed > 0 ? "rose" : "amber"} icon="alert" />
          </div>

          <div className="grid gap-4 xl:grid-cols-3">
            <Card className="xl:col-span-1">
              <CardHeader title="Upcoming runs" subtitle="Next scheduled executions" right={<Pill tone="indigo">{active.length} active</Pill>} />
              <ol className="space-y-1 p-4">
                {active.slice(0, 6).map((schedule) => (
                  <li key={schedule.id} className="flex gap-3 pb-3 last:pb-0">
                    <span className={`mt-1.5 h-3 w-3 shrink-0 rounded-full border-2 ${schedule.lastStatus === "failed" ? "border-rose-500 bg-rose-950" : "border-emerald-500 bg-emerald-950"}`} />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center justify-between gap-2">
                        <span className="truncate text-xs font-semibold text-slate-200">{schedule.name}</span>
                        <span className="shrink-0 font-mono text-[10px] text-indigo-300">{schedule.nextRunAt ? formatDate(schedule.nextRunAt) : "-"}</span>
                      </div>
                      <div className="mt-1 flex items-center justify-between gap-2 text-[10px] text-slate-500">
                        <span className="truncate">{cronLabel(schedule.cron)}</span>
                        <span className="truncate">{schedule.timezone}</span>
                      </div>
                    </div>
                  </li>
                ))}
                {active.length === 0 && <li className="py-6 text-center text-xs text-slate-500">No active schedules.</li>}
              </ol>
            </Card>

            <Card className="xl:col-span-2">
              <CardHeader title="Audit schedules" subtitle="Recurring crawl definitions for this project." right={<Pill tone="slate">{formatNumber(schedules.length)} total</Pill>} />
              {schedules.length === 0 ? (
                <EmptyState
                  icon="calendar"
                  title="No schedules yet"
                  description="Create a recurring audit to keep evidence and consent checks current."
                  action={<button className="btn-primary" onClick={() => setCreateOpen(true)}><Icon name="plus" className="h-3.5 w-3.5" />New schedule</button>}
                />
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[900px] text-xs" aria-label="Audit schedules">
                    <thead className="table-head">
                      <tr>
                        <th scope="col" className="px-4 py-2.5">Schedule</th>
                        <th scope="col" className="px-4 py-2.5">Cadence</th>
                        <th scope="col" className="px-4 py-2.5">Next run</th>
                        <th scope="col" className="px-4 py-2.5">Last run</th>
                        <th scope="col" className="px-4 py-2.5">Enabled</th>
                        <th scope="col" className="px-4 py-2.5"><span className="sr-only">Actions</span></th>
                      </tr>
                    </thead>
                    <tbody>
                      {schedules.map((schedule) => (
                        <tr key={schedule.id} className="table-row">
                          <td className="px-4 py-3">
                            <div className="font-semibold text-slate-200">{schedule.name}</div>
                            <div className="mt-0.5 max-w-[240px] truncate font-mono text-[10px] text-slate-500">{schedule.url}</div>
                            <div className="mt-1 flex flex-wrap gap-1 text-[9px] text-slate-500">
                              <span className="rounded border border-slate-700 bg-slate-800/60 px-1 py-0.5">{formatNumber(schedule.maxPages)} pages</span>
                              <span className="rounded border border-slate-700 bg-slate-800/60 px-1 py-0.5">depth {schedule.maxDepth}</span>
                              <span className="rounded border border-slate-700 bg-slate-800/60 px-1 py-0.5">{schedule.consent.replaceAll("_", " ")}</span>
                            </div>
                          </td>
                          <td className="px-4 py-3">
                            <div className="text-slate-300">{cronLabel(schedule.cron)}</div>
                            <div className="mt-0.5 text-[10px] text-slate-500">{schedule.timezone}</div>
                          </td>
                          <td className="px-4 py-3">
                            <div className="text-slate-300">{schedule.nextRunAt ? formatDate(schedule.nextRunAt) : "-"}</div>
                            <div className="mono mt-0.5 text-[10px] text-slate-500">{schedule.cron}</div>
                          </td>
                          <td className="px-4 py-3">
                            {schedule.lastStatus ? <StatusBadge status={schedule.lastStatus} /> : <span className="text-[10px] text-slate-500">Never</span>}
                            <div className="mt-1 text-[10px] text-slate-500">{schedule.lastRunAt ? timeAgo(schedule.lastRunAt) : "-"}</div>
                          </td>
                          <td className="px-4 py-3">
                            <Toggle checked={schedule.enabled} onChange={(value) => void toggleSchedule(schedule, value)} label={`Enable ${schedule.name}`} />
                          </td>
                          <td className="px-4 py-3">
                            <div className="flex justify-end gap-1">
                              <button className="rounded-lg p-1.5 text-slate-500 transition hover:bg-slate-800 hover:text-indigo-300" onClick={() => void runNow(schedule)} disabled={busy === schedule.id} aria-label={`Run ${schedule.name} now`} title="Run now">
                                <Icon name="play" className="h-3.5 w-3.5" />
                              </button>
                              <button className="rounded-lg p-1.5 text-slate-500 transition hover:bg-rose-950/40 hover:text-rose-300" onClick={() => void removeSchedule(schedule)} disabled={busy === schedule.id} aria-label={`Delete ${schedule.name}`} title="Delete schedule">
                                <Icon name="trash" className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </Card>
          </div>
        </>
      )}

      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title="New audit schedule" subtitle="Create a recurring crawl with its own limits, consent state and region.">
        <form onSubmit={createNew} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="schedule-name">Schedule name</label>
              <input id="schedule-name" className="input" placeholder="Production storefront" value={name} onChange={(event) => setName(event.target.value)} required />
            </div>
            <div>
              <label className="label" htmlFor="schedule-url">Start URL</label>
              <input id="schedule-url" className="input" placeholder="https://www.example.com/" value={url} onChange={(event) => setUrl(event.target.value)} required />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="schedule-cadence">Cadence</label>
              <select id="schedule-cadence" className="input" value={cadence} onChange={(event) => setCadence(event.target.value)}>
                {CADENCE_PRESETS.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
              </select>
            </div>
            <div>
              <label className="label" htmlFor="schedule-timezone">Timezone</label>
              <select id="schedule-timezone" className="input" value={timezone} onChange={(event) => setTimezone(event.target.value)}>
                {TIMEZONES.map((item) => <option key={item} value={item}>{item}</option>)}
              </select>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="label" htmlFor="schedule-pages">Max pages</label>
              <input id="schedule-pages" className="input mono" type="number" min={1} max={5000} value={pages} onChange={(event) => setPages(event.target.value)} />
            </div>
            <div>
              <label className="label" htmlFor="schedule-depth">Max depth</label>
              <input id="schedule-depth" className="input mono" type="number" min={1} max={10} value={depth} onChange={(event) => setDepth(event.target.value)} />
            </div>
            <div>
              <label className="label" htmlFor="schedule-consent">Consent state</label>
              <select id="schedule-consent" className="input" value={consent} onChange={(event) => setConsent(event.target.value)}>
                {CONSENT_OPTIONS.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
              </select>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setCreateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={!name.trim() || !url.trim() || busy === "create"}>
              <Icon name="calendar" className="h-3.5 w-3.5" />
              Create schedule
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
