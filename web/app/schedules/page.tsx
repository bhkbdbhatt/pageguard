"use client";

import { useMemo, useState } from "react";
import { DemoBanner, DemoPageHeader } from "../../components/demo";
import { Icon } from "../../components/icons";
import {
  Card,
  CardHeader,
  EmptyState,
  KpiCard,
  Modal,
  Pill,
  StatusBadge,
  Toggle,
} from "../../components/ui";
import type { RunStatus } from "../../lib/api";
import { DEMO_SCHEDULES, type DemoSchedule } from "../../lib/demo";
import { cronLabel, formatNumber, formatPercent } from "../../lib/format";

const CADENCE_PRESETS = [
  { value: "0 6 * * *", label: "Daily · 06:00" },
  { value: "30 8 * * 1-5", label: "Weekdays · 08:30" },
  { value: "0 * * * *", label: "Hourly" },
  { value: "*/15 * * * *", label: "Every 15 minutes" },
  { value: "0 6 * * 1", label: "Weekly · Mon 06:00" },
  { value: "0 2 1 * *", label: "Monthly · 1st 02:00" },
];

const TIMEZONES = ["UTC", "Europe/London", "Europe/Berlin", "America/New_York", "Asia/Singapore"];

const CONSENT_OPTIONS = [
  { value: "pre_consent", label: "Pre-consent snapshot" },
  { value: "accept_all", label: "Accept all" },
  { value: "reject_all", label: "Reject all" },
  { value: "necessary_only", label: "Necessary only" },
];

export default function SchedulesPage() {
  const [schedules, setSchedules] = useState<DemoSchedule[]>(DEMO_SCHEDULES);
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [cadence, setCadence] = useState(CADENCE_PRESETS[0].value);
  const [timezone, setTimezone] = useState("UTC");
  const [pages, setPages] = useState("120");
  const [depth, setDepth] = useState("3");
  const [consent, setConsent] = useState("pre_consent");
  const [notify, setNotify] = useState(true);

  const active = schedules.filter((schedule) => schedule.enabled);
  const failed = schedules.filter((schedule) => schedule.lastStatus === "failed").length;
  const failureRate = schedules.length > 0 ? Math.round((failed / schedules.length) * 100) : 0;
  const nextRun = useMemo(() => active[0]?.nextRunLabel ?? "—", [active]);

  function toggleSchedule(id: string, enabled: boolean) {
    setSchedules((current) =>
      current.map((schedule) => (schedule.id === id ? { ...schedule, enabled } : schedule)),
    );
  }

  function runNow(id: string) {
    setSchedules((current) =>
      current.map((schedule) =>
        schedule.id === id
          ? { ...schedule, lastStatus: "running" as RunStatus, lastRunLabel: "Just now" }
          : schedule,
      ),
    );
  }

  function createSchedule(event: React.FormEvent) {
    event.preventDefault();
    const next: DemoSchedule = {
      id: `sch-${Date.now()}`,
      name: name.trim() || "Untitled schedule",
      url: url.trim() || "https://www.acme.com/",
      cadence: cronLabel(cadence),
      cron: cadence,
      timezone,
      nextRun: "",
      nextRunLabel: "Tomorrow · 06:00",
      lastStatus: "queued",
      lastRunLabel: "Never",
      owner: "You",
      enabled: true,
      pages: Number(pages) || 50,
      depth: Number(depth) || 3,
      consent,
    };
    setSchedules((current) => [next, ...current]);
    setCreateOpen(false);
    setName("");
    setUrl("");
    setCadence(CADENCE_PRESETS[0].value);
    setTimezone("UTC");
    setPages("120");
    setDepth("3");
    setConsent("pre_consent");
    setNotify(true);
  }

  return (
    <div className="space-y-5">
      <DemoPageHeader
        eyebrow="Automation"
        title="Scheduled Audits"
        subtitle="Keep governance evidence fresh with recurring crawls across every site, region and consent state."
        icon="calendar"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            <Icon name="plus" className="h-3.5 w-3.5" />
            New schedule
          </button>
        }
      />

      <DemoBanner />

      <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        <KpiCard label="Active schedules" value={`${active.length}/${schedules.length}`} hint="paused schedules excluded" tone="indigo" icon="calendar" />
        <KpiCard label="Next run" value={nextRun} hint="Production storefront" tone="sky" icon="clock" />
        <KpiCard label="Runs this week" value={formatNumber(42)} hint="38 succeeded · 4 failed" tone="emerald" icon="play" />
        <KpiCard label="Failure rate" value={formatPercent(failureRate, 0)} hint="last completed run per schedule" tone={failureRate > 15 ? "rose" : "amber"} icon="alert" />
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <Card className="xl:col-span-1">
          <CardHeader title="Upcoming runs" subtitle="Next scheduled executions" right={<Pill tone="indigo">{active.length} active</Pill>} />
          <ol className="space-y-1 p-4">
            {active.slice(0, 5).map((schedule, index) => (
              <li key={schedule.id} className="relative flex gap-3 pb-4 last:pb-0">
                {index < active.slice(0, 5).length - 1 && (
                  <span className="absolute left-[7px] top-4 h-full w-px bg-slate-800" aria-hidden="true" />
                )}
                <span
                  className={`relative mt-1.5 h-3.5 w-3.5 shrink-0 rounded-full border-2 ${
                    schedule.lastStatus === "running"
                      ? "animate-pulse border-sky-400 bg-sky-950"
                      : schedule.lastStatus === "failed"
                        ? "border-rose-500 bg-rose-950"
                        : "border-emerald-500 bg-emerald-950"
                  }`}
                />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate text-xs font-semibold text-slate-200">{schedule.name}</span>
                    <span className="shrink-0 font-mono text-[10px] text-indigo-300">{schedule.nextRunLabel}</span>
                  </div>
                  <div className="mt-1 flex items-center justify-between gap-2 text-[10px] text-slate-500">
                    <span className="truncate">{schedule.cadence}</span>
                    <span className="truncate">{schedule.owner}</span>
                  </div>
                </div>
              </li>
            ))}
            {active.length === 0 && (
              <li className="py-6 text-center text-xs text-slate-500">No active schedules. Resume one below.</li>
            )}
          </ol>
        </Card>

        <Card className="xl:col-span-2">
          <CardHeader
            title="Audit schedules"
            subtitle="Recurring crawl definitions for this project."
            right={<Pill tone="slate">{formatNumber(schedules.length)} total</Pill>}
          />
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
                    <th scope="col" className="px-4 py-2.5">Owner</th>
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
                          <span className="rounded border border-slate-700 bg-slate-800/60 px-1 py-0.5">{formatNumber(schedule.pages)} pages</span>
                          <span className="rounded border border-slate-700 bg-slate-800/60 px-1 py-0.5">depth {schedule.depth}</span>
                          <span className="rounded border border-slate-700 bg-slate-800/60 px-1 py-0.5">{schedule.consent.replace("_", " ")}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="text-slate-300">{schedule.cadence}</div>
                        <div className="mt-0.5 text-[10px] text-slate-500">{schedule.timezone}</div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="text-slate-300">{schedule.nextRunLabel}</div>
                        <div className="mono mt-0.5 text-[10px] text-slate-500">{schedule.cron}</div>
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={schedule.lastStatus} />
                        <div className="mt-1 text-[10px] text-slate-500">{schedule.lastRunLabel}</div>
                      </td>
                      <td className="px-4 py-3 text-slate-400">{schedule.owner}</td>
                      <td className="px-4 py-3">
                        <Toggle checked={schedule.enabled} onChange={(next) => toggleSchedule(schedule.id, next)} label={`Enable ${schedule.name}`} />
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex justify-end gap-1">
                          <button
                            className="rounded-lg p-1.5 text-slate-500 transition hover:bg-slate-800 hover:text-indigo-300"
                            onClick={() => runNow(schedule.id)}
                            aria-label={`Run ${schedule.name} now`}
                            title="Run now"
                          >
                            <Icon name="play" className="h-3.5 w-3.5" />
                          </button>
                          <button
                            className="rounded-lg p-1.5 text-slate-500 transition hover:bg-rose-950/40 hover:text-rose-300"
                            onClick={() => setSchedules((current) => current.filter((item) => item.id !== schedule.id))}
                            aria-label={`Delete ${schedule.name}`}
                            title="Delete schedule"
                          >
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

      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="New audit schedule"
        subtitle="Create a recurring crawl with its own limits, consent state and region."
      >
        <form onSubmit={createSchedule} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="schedule-name">Schedule name</label>
              <input id="schedule-name" className="input" placeholder="Production storefront" value={name} onChange={(event) => setName(event.target.value)} />
            </div>
            <div>
              <label className="label" htmlFor="schedule-url">Start URL</label>
              <input id="schedule-url" className="input" placeholder="https://www.acme.com/" value={url} onChange={(event) => setUrl(event.target.value)} />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="schedule-cadence">Cadence</label>
              <select id="schedule-cadence" className="input" value={cadence} onChange={(event) => setCadence(event.target.value)}>
                {CADENCE_PRESETS.map((item) => (
                  <option key={item.value} value={item.value}>{item.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="label" htmlFor="schedule-timezone">Timezone</label>
              <select id="schedule-timezone" className="input" value={timezone} onChange={(event) => setTimezone(event.target.value)}>
                {TIMEZONES.map((item) => (
                  <option key={item} value={item}>{item}</option>
                ))}
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
                {CONSENT_OPTIONS.map((item) => (
                  <option key={item.value} value={item.value}>{item.label}</option>
                ))}
              </select>
            </div>
          </div>
          <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-950/40 px-3 py-2.5">
            <div>
              <p className="text-xs font-semibold text-slate-200">Notify on completion</p>
              <p className="text-[10px] text-slate-500">Send a summary to the active alert channels.</p>
            </div>
            <Toggle checked={notify} onChange={setNotify} label="Notify on completion" />
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setCreateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={!name.trim() || !url.trim()}>
              <Icon name="calendar" className="h-3.5 w-3.5" />
              Create schedule
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
