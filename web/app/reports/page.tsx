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
  Tabs,
  Toggle,
} from "../../components/ui";
import {
  createReport,
  createReportSchedule,
  deleteReport,
  deleteReportSchedule,
  listReportSchedules,
  listReportTemplates,
  listReports,
  reportDownloadURL,
  runReportScheduleNow,
  updateReportSchedule,
  type Report,
  type ReportFormat,
  type ReportSchedule,
  type ReportTemplate,
} from "../../lib/api";
import { bytes, cronLabel, formatDate, formatNumber, timeAgo } from "../../lib/format";

type Tab = "library" | "schedules" | "templates";

const FORMATS: ReportFormat[] = ["json", "csv", "html"];
const CADENCE_PRESETS = [
  { value: "0 6 * * *", label: "Daily - 06:00" },
  { value: "0 6 * * 1", label: "Weekly - Mon 06:00" },
  { value: "0 2 1 * *", label: "Monthly - 1st 02:00" },
  { value: "0 * * * *", label: "Hourly" },
];
const TIMEZONES = ["UTC", "Europe/London", "Europe/Berlin", "America/New_York", "Asia/Singapore"];

export default function ReportsPage() {
  const { project, loadingProjects } = useApp();
  const [tab, setTab] = useState<Tab>("library");
  const [reports, setReports] = useState<Report[]>([]);
  const [schedules, setSchedules] = useState<ReportSchedule[]>([]);
  const [templates, setTemplates] = useState<ReportTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const [generateOpen, setGenerateOpen] = useState(false);
  const [templateId, setTemplateId] = useState("tpl-governance");
  const [format, setFormat] = useState<ReportFormat>("json");
  const [reportName, setReportName] = useState("");
  const [owner, setOwner] = useState("");

  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [scheduleName, setScheduleName] = useState("");
  const [scheduleTemplate, setScheduleTemplate] = useState("tpl-governance");
  const [scheduleFormat, setScheduleFormat] = useState<ReportFormat>("json");
  const [scheduleCron, setScheduleCron] = useState(CADENCE_PRESETS[0].value);
  const [scheduleTimezone, setScheduleTimezone] = useState("UTC");
  const [recipients, setRecipients] = useState("");

  const load = useCallback(async () => {
    if (!project) {
      setReports([]);
      setSchedules([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const [nextReports, nextSchedules, nextTemplates] = await Promise.all([
        listReports(project.id),
        listReportSchedules(project.id),
        listReportTemplates(),
      ]);
      setReports(nextReports);
      setSchedules(nextSchedules);
      setTemplates(nextTemplates);
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

  useEffect(() => {
    if (!project) return;
    const id = window.setInterval(() => {
      void Promise.all([listReports(project.id), listReportSchedules(project.id)])
        .then(([nextReports, nextSchedules]) => {
          setReports(nextReports);
          setSchedules(nextSchedules);
        })
        .catch(() => undefined);
    }, 10000);
    return () => window.clearInterval(id);
  }, [project]);

  const ready = reports.filter((report) => report.status === "ready");
  const totalBytes = ready.reduce((sum, report) => sum + report.sizeBytes, 0);
  const activeSchedules = schedules.filter((schedule) => schedule.enabled).length;

  const templateName = useMemo(() => {
    const map: Record<string, string> = {};
    templates.forEach((template) => { map[template.id] = template.name; });
    return map;
  }, [templates]);

  async function generate(event: React.FormEvent) {
    event.preventDefault();
    if (!project) return;
    setBusy("generate");
    try {
      const created = await createReport({
        projectId: project.id,
        templateId,
        format,
        name: reportName.trim() || undefined,
        owner: owner.trim() || undefined,
        scope: { type: "project", id: project.id, label: project.name },
      });
      setReports((current) => [created, ...current]);
      setGenerateOpen(false);
      setReportName("");
      setOwner("");
      setNotice(`Generated "${created.name}".`);
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function removeReport(report: Report) {
    if (!window.confirm(`Delete report "${report.name}"?`)) return;
    setBusy(report.id);
    try {
      await deleteReport(report.id);
      setReports((current) => current.filter((item) => item.id !== report.id));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function toggleSchedule(schedule: ReportSchedule) {
    setBusy(schedule.id);
    try {
      const updated = await updateReportSchedule(schedule.id, { enabled: !schedule.enabled });
      setSchedules((current) => current.map((item) => (item.id === schedule.id ? updated : item)));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function removeSchedule(schedule: ReportSchedule) {
    if (!window.confirm(`Delete report schedule "${schedule.name}"?`)) return;
    setBusy(schedule.id);
    try {
      await deleteReportSchedule(schedule.id);
      setSchedules((current) => current.filter((item) => item.id !== schedule.id));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function runSchedule(schedule: ReportSchedule) {
    setBusy(schedule.id);
    try {
      const report = await runReportScheduleNow(schedule.id);
      setReports((current) => [report, ...current]);
      setNotice(`Generated "${report.name}" from schedule ${schedule.name}.`);
      setError(null);
      void load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function submitSchedule(event: React.FormEvent) {
    event.preventDefault();
    if (!project) return;
    setBusy("schedule");
    try {
      const created = await createReportSchedule({
        projectId: project.id,
        templateId: scheduleTemplate,
        name: scheduleName.trim() || undefined,
        cron: scheduleCron,
        timezone: scheduleTimezone,
        format: scheduleFormat,
        recipients: recipients.split(",").map((item) => item.trim()).filter(Boolean),
        scope: { type: "project", id: project.id, label: project.name },
        enabled: true,
      });
      setSchedules((current) => [created, ...current]);
      setScheduleOpen(false);
      setScheduleName("");
      setScheduleTemplate("tpl-governance");
      setScheduleFormat("json");
      setScheduleCron(CADENCE_PRESETS[0].value);
      setScheduleTimezone("UTC");
      setRecipients("");
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
        icon="folder"
        title="No project selected"
        description="Create a project and run an audit to generate compliance reports."
      />
    );
  }

  return (
    <div className="space-y-5">
      <PageHeader
        eyebrow="Evidence"
        title="Reports & Exports"
        subtitle="Generate governance, consent, inventory, journey and privacy artefacts, then download or schedule them."
        icon="folder"
        actions={
          <>
            <button className="btn" onClick={() => setScheduleOpen(true)} disabled={!project}>
              <Icon name="calendar" className="h-3.5 w-3.5" />
              Schedule
            </button>
            <button className="btn-primary" onClick={() => setGenerateOpen(true)} disabled={!project}>
              <Icon name="file" className="h-3.5 w-3.5" />
              Generate report
            </button>
          </>
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
        <Loading label="Loading reports" />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
            <KpiCard label="Reports" value={formatNumber(reports.length)} hint={`${formatNumber(ready.length)} ready`} tone="indigo" icon="folder" />
            <KpiCard label="Scheduled jobs" value={`${activeSchedules}/${schedules.length}`} hint="recurring report generation" tone="sky" icon="calendar" />
            <KpiCard label="Stored size" value={bytes(totalBytes)} hint="generated artefacts" tone="emerald" icon="database" />
            <KpiCard label="Templates" value={formatNumber(templates.length)} hint="JSON, CSV or HTML" tone="amber" icon="file" />
          </div>

          <Card>
            <div className="border-b border-slate-800 px-4 py-3">
              <Tabs
                value={tab}
                onChange={(id) => setTab(id as Tab)}
                tabs={[
                  { id: "library", label: "Report library", count: reports.length },
                  { id: "schedules", label: "Schedules", count: schedules.length },
                  { id: "templates", label: "Templates", count: templates.length },
                ]}
              />
            </div>

            {tab === "library" && (
              reports.length === 0 ? (
                <EmptyState icon="folder" title="No reports yet" description="Generate your first report from the latest audit data." action={<button className="btn-primary" onClick={() => setGenerateOpen(true)}><Icon name="file" className="h-3.5 w-3.5" />Generate report</button>} />
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[820px] text-xs" aria-label="Generated reports">
                    <thead className="table-head">
                      <tr>
                        <th scope="col" className="px-4 py-2.5">Report</th>
                        <th scope="col" className="px-4 py-2.5">Template</th>
                        <th scope="col" className="px-4 py-2.5">Format</th>
                        <th scope="col" className="px-4 py-2.5">Size</th>
                        <th scope="col" className="px-4 py-2.5">Created</th>
                        <th scope="col" className="px-4 py-2.5"><span className="sr-only">Actions</span></th>
                      </tr>
                    </thead>
                    <tbody>
                      {reports.map((report) => (
                        <tr key={report.id} className="table-row">
                          <td className="px-4 py-3">
                            <div className="font-semibold text-slate-200">{report.name}</div>
                            <div className="mt-0.5 text-[10px] text-slate-500">{report.scope.label || report.scope.type}{report.owner ? ` / ${report.owner}` : ""}</div>
                          </td>
                          <td className="px-4 py-3 text-slate-400">{templateName[report.templateId] ?? report.templateId}</td>
                          <td className="px-4 py-3"><Pill tone="slate">{report.format}</Pill></td>
                          <td className="px-4 py-3 text-slate-400">{bytes(report.sizeBytes)}</td>
                          <td className="px-4 py-3 text-[11px] text-slate-500">{formatDate(report.createdAt)}</td>
                          <td className="px-4 py-3">
                            <div className="flex justify-end gap-1">
                              <a className="rounded-lg p-1.5 text-slate-500 transition hover:bg-slate-800 hover:text-indigo-300" href={reportDownloadURL(report.id)} aria-label={`Download ${report.name}`} title="Download">
                                <Icon name="download" className="h-3.5 w-3.5" />
                              </a>
                              <button className="rounded-lg p-1.5 text-slate-500 transition hover:bg-rose-950/40 hover:text-rose-300" onClick={() => void removeReport(report)} disabled={busy === report.id} aria-label={`Delete ${report.name}`} title="Delete">
                                <Icon name="trash" className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )
            )}

            {tab === "schedules" && (
              schedules.length === 0 ? (
                <EmptyState icon="calendar" title="No scheduled reports" description="Schedule recurring report generation so evidence is always current." action={<button className="btn-primary" onClick={() => setScheduleOpen(true)}><Icon name="calendar" className="h-3.5 w-3.5" />New schedule</button>} />
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[860px] text-xs" aria-label="Report schedules">
                    <thead className="table-head">
                      <tr>
                        <th scope="col" className="px-4 py-2.5">Schedule</th>
                        <th scope="col" className="px-4 py-2.5">Cadence</th>
                        <th scope="col" className="px-4 py-2.5">Format</th>
                        <th scope="col" className="px-4 py-2.5">Next run</th>
                        <th scope="col" className="px-4 py-2.5">Enabled</th>
                        <th scope="col" className="px-4 py-2.5"><span className="sr-only">Actions</span></th>
                      </tr>
                    </thead>
                    <tbody>
                      {schedules.map((schedule) => (
                        <tr key={schedule.id} className="table-row">
                          <td className="px-4 py-3">
                            <div className="font-semibold text-slate-200">{schedule.name}</div>
                            <div className="mt-0.5 text-[10px] text-slate-500">{templateName[schedule.templateId] ?? schedule.templateId}{schedule.recipients.length ? ` / ${schedule.recipients.length} recipients` : ""}</div>
                          </td>
                          <td className="px-4 py-3">
                            <div className="text-slate-300">{cronLabel(schedule.cron)}</div>
                            <div className="mt-0.5 text-[10px] text-slate-500">{schedule.timezone}</div>
                          </td>
                          <td className="px-4 py-3"><Pill tone="slate">{schedule.format}</Pill></td>
                          <td className="px-4 py-3">
                            <div className="text-slate-300">{schedule.nextRunAt ? formatDate(schedule.nextRunAt) : "-"}</div>
                            <div className="mt-0.5 text-[10px] text-slate-500">{schedule.lastRunAt ? `last ${timeAgo(schedule.lastRunAt)}` : "never run"}</div>
                          </td>
                          <td className="px-4 py-3"><Toggle checked={schedule.enabled} onChange={() => void toggleSchedule(schedule)} label={`Enable ${schedule.name}`} /></td>
                          <td className="px-4 py-3">
                            <div className="flex justify-end gap-1">
                              <button className="rounded-lg p-1.5 text-slate-500 transition hover:bg-slate-800 hover:text-indigo-300" onClick={() => void runSchedule(schedule)} disabled={busy === schedule.id} aria-label={`Run ${schedule.name} now`} title="Run now">
                                <Icon name="play" className="h-3.5 w-3.5" />
                              </button>
                              <button className="rounded-lg p-1.5 text-slate-500 transition hover:bg-rose-950/40 hover:text-rose-300" onClick={() => void removeSchedule(schedule)} disabled={busy === schedule.id} aria-label={`Delete ${schedule.name}`} title="Delete">
                                <Icon name="trash" className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )
            )}

            {tab === "templates" && (
              <div className="grid gap-3 p-4 md:grid-cols-2 xl:grid-cols-3">
                {templates.map((template) => (
                  <button
                    key={template.id}
                    className="rounded-xl border border-slate-800 bg-slate-950/40 p-4 text-left transition hover:border-slate-700"
                    onClick={() => {
                      setTemplateId(template.id);
                      setGenerateOpen(true);
                    }}
                  >
                    <div className="flex items-center justify-between">
                      <p className="text-xs font-semibold text-slate-200">{template.name}</p>
                      <Icon name="file" className="h-4 w-4 text-indigo-300" />
                    </div>
                    <p className="mt-2 text-[11px] leading-relaxed text-slate-500">{template.description}</p>
                    <div className="mt-3 flex flex-wrap gap-1">
                      {template.formats.map((item) => <span key={item} className="rounded border border-slate-700 bg-slate-800/60 px-1.5 py-0.5 text-[9px] uppercase tracking-wide text-slate-400">{item}</span>)}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </Card>
        </>
      )}

      <Modal open={generateOpen} onClose={() => setGenerateOpen(false)} title="Generate report" subtitle="Reports are generated synchronously from the latest project data.">
        <form onSubmit={generate} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="report-template">Template</label>
              <select id="report-template" className="input" value={templateId} onChange={(event) => setTemplateId(event.target.value)}>
                {templates.map((template) => <option key={template.id} value={template.id}>{template.name}</option>)}
              </select>
            </div>
            <div>
              <label className="label" htmlFor="report-format">Format</label>
              <select id="report-format" className="input" value={format} onChange={(event) => setFormat(event.target.value as ReportFormat)}>
                {FORMATS.map((item) => <option key={item} value={item}>{item.toUpperCase()}</option>)}
              </select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="report-name">Name (optional)</label>
              <input id="report-name" className="input" value={reportName} onChange={(event) => setReportName(event.target.value)} placeholder="Auto-generated from template and date" />
            </div>
            <div>
              <label className="label" htmlFor="report-owner">Owner</label>
              <input id="report-owner" className="input" value={owner} onChange={(event) => setOwner(event.target.value)} placeholder="Privacy team" />
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setGenerateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={busy === "generate"}>Generate</button>
          </div>
        </form>
      </Modal>

      <Modal open={scheduleOpen} onClose={() => setScheduleOpen(false)} title="Schedule report" subtitle="Generate a report automatically on a cron cadence.">
        <form onSubmit={submitSchedule} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="schedule-report-name">Name</label>
              <input id="schedule-report-name" className="input" value={scheduleName} onChange={(event) => setScheduleName(event.target.value)} placeholder="Weekly governance" />
            </div>
            <div>
              <label className="label" htmlFor="schedule-report-template">Template</label>
              <select id="schedule-report-template" className="input" value={scheduleTemplate} onChange={(event) => setScheduleTemplate(event.target.value)}>
                {templates.map((template) => <option key={template.id} value={template.id}>{template.name}</option>)}
              </select>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="label" htmlFor="schedule-report-cron">Cadence</label>
              <select id="schedule-report-cron" className="input" value={scheduleCron} onChange={(event) => setScheduleCron(event.target.value)}>
                {CADENCE_PRESETS.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
              </select>
            </div>
            <div>
              <label className="label" htmlFor="schedule-report-timezone">Timezone</label>
              <select id="schedule-report-timezone" className="input" value={scheduleTimezone} onChange={(event) => setScheduleTimezone(event.target.value)}>
                {TIMEZONES.map((item) => <option key={item} value={item}>{item}</option>)}
              </select>
            </div>
            <div>
              <label className="label" htmlFor="schedule-report-format">Format</label>
              <select id="schedule-report-format" className="input" value={scheduleFormat} onChange={(event) => setScheduleFormat(event.target.value as ReportFormat)}>
                {FORMATS.map((item) => <option key={item} value={item}>{item.toUpperCase()}</option>)}
              </select>
            </div>
          </div>
          <div>
            <label className="label" htmlFor="schedule-report-recipients">Recipients (comma separated)</label>
            <input id="schedule-report-recipients" className="input" value={recipients} onChange={(event) => setRecipients(event.target.value)} placeholder="privacy@example.com, security@example.com" />
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setScheduleOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={busy === "schedule"}>Create schedule</button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
