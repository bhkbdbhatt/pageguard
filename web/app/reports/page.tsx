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
  SearchInput,
  Select,
  Toggle,
  type Tone,
} from "../../components/ui";
import {
  DEMO_REPORTS,
  DEMO_REPORT_SCHEDULES,
  DEMO_REPORT_TEMPLATES,
  type DemoReportRow,
  type DemoReportSchedule,
  type ReportFormat,
  type ReportStatus,
} from "../../lib/demo";
import { formatNumber } from "../../lib/format";

const TONE_STYLES: Record<Tone, string> = {
  slate: "border-slate-700 bg-slate-800/60 text-slate-300",
  indigo: "border-indigo-800/60 bg-indigo-950/50 text-indigo-300",
  emerald: "border-emerald-800/60 bg-emerald-950/50 text-emerald-300",
  rose: "border-rose-800/60 bg-rose-950/50 text-rose-300",
  amber: "border-amber-800/60 bg-amber-950/40 text-amber-300",
  sky: "border-sky-800/60 bg-sky-950/40 text-sky-300",
};

const STATUS_TONES: Record<ReportStatus, "emerald" | "sky" | "rose"> = {
  ready: "emerald",
  generating: "sky",
  failed: "rose",
};

const SCOPES = [
  { value: "Production storefront", label: "Production storefront" },
  { value: "EU operations", label: "EU operations" },
  { value: "Checkout flow", label: "Checkout flow" },
  { value: "All sites", label: "All sites" },
];

export default function ReportsPage() {
  const [reports, setReports] = useState<DemoReportRow[]>(DEMO_REPORTS);
  const [schedules, setSchedules] = useState<DemoReportSchedule[]>(DEMO_REPORT_SCHEDULES);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");
  const [generateOpen, setGenerateOpen] = useState(false);
  const [templateId, setTemplateId] = useState(DEMO_REPORT_TEMPLATES[0].id);
  const [scope, setScope] = useState(SCOPES[0].value);
  const [format, setFormat] = useState<ReportFormat>("html");
  const [delivery, setDelivery] = useState("download");

  const template = DEMO_REPORT_TEMPLATES.find((item) => item.id === templateId) ?? DEMO_REPORT_TEMPLATES[0];

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return reports.filter((report) => {
      if (status !== "all" && report.status !== status) return false;
      if (!q) return true;
      return [report.name, report.template, report.scope, report.owner].some((value) =>
        value.toLowerCase().includes(q),
      );
    });
  }, [query, reports, status]);

  const enabledSchedules = schedules.filter((schedule) => schedule.enabled).length;
  const readyReports = reports.filter((report) => report.status === "ready").length;

  function openGenerate(id: string) {
    const next = DEMO_REPORT_TEMPLATES.find((item) => item.id === id) ?? DEMO_REPORT_TEMPLATES[0];
    setTemplateId(next.id);
    setFormat(next.formats[0]);
    setGenerateOpen(true);
  }

  function changeTemplate(id: string) {
    const next = DEMO_REPORT_TEMPLATES.find((item) => item.id === id) ?? DEMO_REPORT_TEMPLATES[0];
    setTemplateId(next.id);
    setFormat(next.formats[0]);
  }

  function generateReport(event: React.FormEvent) {
    event.preventDefault();
    const id = `rep-${Date.now()}`;
    const next: DemoReportRow = {
      id,
      name: `${scope} · ${template.name.toLowerCase()}`,
      template: template.name,
      scope,
      format,
      createdAt: "now",
      createdLabel: "Just now",
      size: "—",
      status: "generating",
      owner: "You",
    };
    setReports((current) => [next, ...current]);
    setGenerateOpen(false);
    window.setTimeout(() => {
      setReports((current) =>
        current.map((report) =>
          report.id === id ? { ...report, status: "ready", size: "642 KB" } : report,
        ),
      );
    }, 1600);
  }

  return (
    <div className="space-y-5">
      <DemoPageHeader
        eyebrow="Evidence & exports"
        title="Reports & Exports Hub"
        subtitle="Generate governance, privacy and journey evidence in HTML, CSV or JSON, and schedule recurring deliveries."
        icon="folder"
        actions={
          <>
            <button className="btn" title="Demo only">
              <Icon name="calendar" className="h-3.5 w-3.5" />
              Schedule export
            </button>
            <button className="btn-primary" onClick={() => openGenerate(templateId)}>
              <Icon name="plus" className="h-3.5 w-3.5" />
              Generate report
            </button>
          </>
        }
      />

      <DemoBanner />

      <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        <KpiCard label="Reports generated" value={formatNumber(reports.length)} hint={`${readyReports} ready to download`} tone="indigo" icon="folder" />
        <KpiCard label="Scheduled exports" value={formatNumber(enabledSchedules)} hint={`${schedules.length} schedules configured`} tone="sky" icon="calendar" />
        <KpiCard label="Last export" value="09:52" hint="Governance summary · HTML" tone="emerald" icon="download" />
        <KpiCard label="Evidence stored" value="3.7 MB" hint="across 6 reports" tone="amber" icon="database" />
      </div>

      <Card>
        <CardHeader
          title="Report templates"
          subtitle="Start from a governed template and scope it to a site, run or date range."
          right={<Pill tone="slate">{formatNumber(DEMO_REPORT_TEMPLATES.length)} templates</Pill>}
        />
        <div className="grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3">
          {DEMO_REPORT_TEMPLATES.map((item) => (
            <div key={item.id} className="flex flex-col rounded-xl border border-slate-800 bg-slate-950/40 p-4">
              <div className="flex items-start justify-between gap-3">
                <span className={`flex h-9 w-9 items-center justify-center rounded-lg border ${TONE_STYLES[item.tone]}`}>
                  <Icon name={item.icon} className="h-4 w-4" />
                </span>
                <div className="flex flex-wrap justify-end gap-1">
                  {item.formats.map((itemFormat) => (
                    <span key={itemFormat} className="rounded border border-slate-700 bg-slate-800/60 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wide text-slate-400">
                      {itemFormat}
                    </span>
                  ))}
                </div>
              </div>
              <h3 className="mt-3 text-sm font-semibold text-slate-100">{item.name}</h3>
              <p className="mt-1 flex-1 text-[11px] leading-relaxed text-slate-500">{item.description}</p>
              <button className="btn mt-3 w-full" onClick={() => openGenerate(item.id)}>
                <Icon name="download" className="h-3.5 w-3.5" />
                Generate
              </button>
            </div>
          ))}
          <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-slate-800 bg-slate-950/20 p-4 text-center">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-slate-800 bg-slate-900 text-slate-500">
              <Icon name="plus" className="h-4 w-4" />
            </span>
            <h3 className="mt-3 text-sm font-semibold text-slate-300">Custom template</h3>
            <p className="mt-1 text-[11px] leading-relaxed text-slate-500">Compose sections and branding for client deliverables.</p>
            <button className="btn mt-3 w-full" disabled title="Demo only">Coming soon</button>
          </div>
        </div>
      </Card>

      <div className="grid gap-4 xl:grid-cols-3">
        <Card className="xl:col-span-2">
          <CardHeader
            title="Report library"
            subtitle="Generated evidence for this project."
            right={
              <div className="flex flex-wrap items-center gap-2">
                <SearchInput value={query} onChange={setQuery} placeholder="Search reports" className="w-44" />
                <Select
                  value={status}
                  onChange={setStatus}
                  options={[
                    { value: "all", label: "All statuses" },
                    { value: "ready", label: "Ready" },
                    { value: "generating", label: "Generating" },
                    { value: "failed", label: "Failed" },
                  ]}
                />
              </div>
            }
          />
          {filtered.length === 0 ? (
            <EmptyState
              icon="folder"
              title="No matching reports"
              description="Generate a report or widen the search and status filters."
              action={<button className="btn-primary" onClick={() => openGenerate(templateId)}><Icon name="plus" className="h-3.5 w-3.5" />Generate report</button>}
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[760px] text-xs" aria-label="Generated reports">
                <thead className="table-head">
                  <tr>
                    <th scope="col" className="px-4 py-2.5">Report</th>
                    <th scope="col" className="px-4 py-2.5">Scope</th>
                    <th scope="col" className="px-4 py-2.5">Format</th>
                    <th scope="col" className="px-4 py-2.5">Created</th>
                    <th scope="col" className="px-4 py-2.5">Size</th>
                    <th scope="col" className="px-4 py-2.5">Status</th>
                    <th scope="col" className="px-4 py-2.5"><span className="sr-only">Download</span></th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((report) => (
                    <tr key={report.id} className="table-row">
                      <td className="px-4 py-3">
                        <div className="font-semibold text-slate-200">{report.name}</div>
                        <div className="mt-0.5 text-[10px] text-slate-500">{report.template} · {report.owner}</div>
                      </td>
                      <td className="px-4 py-3 text-slate-400">{report.scope}</td>
                      <td className="px-4 py-3"><span className="rounded border border-slate-700 bg-slate-800/60 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wide text-slate-400">{report.format}</span></td>
                      <td className="px-4 py-3 text-slate-400">{report.createdLabel}</td>
                      <td className="px-4 py-3 text-slate-400">{report.size}</td>
                      <td className="px-4 py-3"><Pill tone={STATUS_TONES[report.status]}>{report.status}</Pill></td>
                      <td className="px-4 py-3 text-right">
                        <button
                          className="btn"
                          disabled={report.status !== "ready"}
                          title={report.status === "ready" ? "Demo download" : "Not ready"}
                        >
                          <Icon name="download" className="h-3.5 w-3.5" />
                          Download
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>

        <Card>
          <CardHeader title="Print preview" subtitle="Governance summary · HTML" right={<Pill tone="indigo">A4</Pill>} />
          <div className="p-4">
            <div className="mx-auto aspect-[1/1.35] w-full max-w-[280px] overflow-hidden rounded-lg border border-slate-700 bg-white p-4 shadow-2xl shadow-slate-950/50">
              <div className="flex items-center gap-1.5">
                <span className="h-4 w-4 rounded bg-indigo-600" />
                <span className="text-[7px] font-extrabold uppercase tracking-widest text-slate-800">OpenTagValidator</span>
              </div>
              <div className="mt-3 text-[11px] font-bold leading-tight text-slate-900">Governance Summary</div>
              <div className="text-[6px] text-slate-500">Acme Commerce · Production storefront · 23 Oct 2026</div>
              <div className="mt-3 grid grid-cols-3 gap-1">
                {[
                  { label: "Health", value: "86%" },
                  { label: "Beacons", value: "312" },
                  { label: "Leaks", value: "3" },
                ].map((item) => (
                  <div key={item.label} className="rounded border border-slate-200 bg-slate-50 p-1">
                    <div className="text-[5px] uppercase tracking-wide text-slate-500">{item.label}</div>
                    <div className="text-[9px] font-bold text-slate-900">{item.value}</div>
                  </div>
                ))}
              </div>
              <div className="mt-3 space-y-1">
                {[82, 64, 94, 48, 72].map((width, index) => (
                  <div key={index} className="flex items-center gap-1">
                    <span className="w-10 shrink-0 text-[5px] text-slate-500">{["Vendor", "Rules", "Consent", "Journey", "Privacy"][index]}</span>
                    <span className="h-1 flex-1 rounded-full bg-slate-200">
                      <span className="block h-1 rounded-full bg-indigo-500" style={{ width: `${width}%` }} />
                    </span>
                  </div>
                ))}
              </div>
              <div className="mt-3 space-y-1">
                {[1, 2, 3, 4, 5, 6].map((row) => (
                  <div key={row} className="h-1 rounded bg-slate-200" style={{ width: `${100 - row * 7}%` }} />
                ))}
              </div>
              <div className="mt-3 border-t border-slate-200 pt-1 text-[5px] text-slate-400">
                Page 1 of 4 · Evidence derived from stored run checkpoints
              </div>
            </div>
            <p className="mt-3 text-center text-[10px] text-slate-500">Print-ready HTML · sample preview</p>
          </div>
        </Card>
      </div>

      <Card>
        <CardHeader
          title="Scheduled exports"
          subtitle="Deliver evidence packs on a recurring cadence."
          right={<Pill tone="slate">{enabledSchedules} active</Pill>}
        />
        <div className="overflow-x-auto">
          <table className="w-full min-w-[720px] text-xs" aria-label="Scheduled exports">
            <thead className="table-head">
              <tr>
                <th scope="col" className="px-4 py-2.5">Schedule</th>
                <th scope="col" className="px-4 py-2.5">Template</th>
                <th scope="col" className="px-4 py-2.5">Cadence</th>
                <th scope="col" className="px-4 py-2.5">Recipients</th>
                <th scope="col" className="px-4 py-2.5">Next run</th>
                <th scope="col" className="px-4 py-2.5">Enabled</th>
              </tr>
            </thead>
            <tbody>
              {schedules.map((schedule) => (
                <tr key={schedule.id} className="table-row">
                  <td className="px-4 py-3 font-semibold text-slate-200">{schedule.name}</td>
                  <td className="px-4 py-3 text-slate-400">{schedule.template}</td>
                  <td className="px-4 py-3 text-slate-400">{schedule.cadence}</td>
                  <td className="max-w-[220px] truncate px-4 py-3 text-slate-400">{schedule.recipients}</td>
                  <td className="px-4 py-3 text-slate-400">{schedule.nextRunLabel}</td>
                  <td className="px-4 py-3">
                    <Toggle
                      checked={schedule.enabled}
                      onChange={(next) =>
                        setSchedules((current) =>
                          current.map((item) => (item.id === schedule.id ? { ...item, enabled: next } : item)),
                        )
                      }
                      label={`Enable ${schedule.name}`}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <Modal
        open={generateOpen}
        onClose={() => setGenerateOpen(false)}
        title="Generate report"
        subtitle="Choose a template, scope and delivery format."
      >
        <form onSubmit={generateReport} className="space-y-4">
          <div>
            <label className="label" htmlFor="report-template">Template</label>
            <select
              id="report-template"
              className="input"
              value={templateId}
              onChange={(event) => changeTemplate(event.target.value)}
            >
              {DEMO_REPORT_TEMPLATES.map((item) => (
                <option key={item.id} value={item.id}>{item.name}</option>
              ))}
            </select>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="report-scope">Scope</label>
              <select id="report-scope" className="input" value={scope} onChange={(event) => setScope(event.target.value)}>
                {SCOPES.map((item) => (
                  <option key={item.value} value={item.value}>{item.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="label" htmlFor="report-format">Format</label>
              <select
                id="report-format"
                className="input"
                value={format}
                onChange={(event) => setFormat(event.target.value as ReportFormat)}
              >
                {template.formats.map((item) => (
                  <option key={item} value={item}>{item.toUpperCase()}</option>
                ))}
              </select>
            </div>
          </div>
          <div>
            <label className="label" htmlFor="report-delivery">Delivery</label>
            <select id="report-delivery" className="input" value={delivery} onChange={(event) => setDelivery(event.target.value)}>
              <option value="download">Download only</option>
              <option value="email">Email recipients</option>
              <option value="slack">Post to Slack</option>
            </select>
          </div>
          <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
            <p className="text-[10px] font-semibold uppercase tracking-wider text-slate-500">Preview</p>
            <p className="mt-1 text-xs text-slate-300">{scope} · {template.name}</p>
            <p className="mt-0.5 text-[11px] text-slate-500">{template.description}</p>
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setGenerateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary">
              <Icon name="download" className="h-3.5 w-3.5" />
              Generate {format.toUpperCase()}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
