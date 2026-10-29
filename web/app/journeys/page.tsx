"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useApp } from "../../components/AppShell";
import { Icon, type IconName } from "../../components/icons";
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
  SearchInput,
  Toggle,
} from "../../components/ui";
import {
  createJourney,
  deleteJourney,
  getJourneyRun,
  listJourneyRuns,
  listJourneys,
  runJourney,
  updateJourney,
  type Journey,
  type JourneyRun,
  type JourneyStep,
} from "../../lib/api";
import { duration, formatDate, formatNumber, formatPercent, timeAgo } from "../../lib/format";

const ACTION_ICONS: Record<string, IconName> = {
  navigate: "globe",
  click: "arrow",
  fill: "edit",
  type: "edit",
  wait: "clock",
};

const RUN_TONES: Record<string, "emerald" | "rose" | "sky" | "slate"> = {
  passed: "emerald",
  failed: "rose",
  running: "sky",
  canceled: "slate",
};

type StepDraft = {
  action: JourneyStep["action"];
  selector: string;
  value: string;
  timeoutMs: string;
};

const EMPTY_STEP: StepDraft = { action: "navigate", selector: "", value: "", timeoutMs: "15000" };

export default function JourneysPage() {
  const { project, loadingProjects } = useApp();
  const [journeys, setJourneys] = useState<Journey[]>([]);
  const [runs, setRuns] = useState<JourneyRun[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [description, setDescription] = useState("");
  const [owner, setOwner] = useState("");
  const [steps, setSteps] = useState<StepDraft[]>([{ ...EMPTY_STEP }]);

  const load = useCallback(async () => {
    if (!project) {
      setJourneys([]);
      setRuns([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const list = await listJourneys(project.id);
      setJourneys(list);
      setSelectedId((current) => (current && list.some((item) => item.id === current) ? current : list[0]?.id ?? null));
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

  const selected = journeys.find((journey) => journey.id === selectedId) ?? null;

  useEffect(() => {
    if (!selected) {
      setRuns([]);
      return;
    }
    let active = true;
    const loadRuns = () =>
      listJourneyRuns(selected.id)
        .then((rows) => active && setRuns(rows))
        .catch((err) => active && setError((err as Error).message));
    void loadRuns();
    const id = window.setInterval(loadRuns, 5000);
    return () => {
      active = false;
      window.clearInterval(id);
    };
  }, [selected]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return journeys;
    return journeys.filter((journey) =>
      [journey.name, journey.baseUrl, journey.description, journey.owner].some((value) =>
        String(value ?? "").toLowerCase().includes(q),
      ),
    );
  }, [journeys, query]);

  const latest = runs[0];
  const passRate = runs.length > 0 ? (runs.filter((run) => run.status === "passed").length / runs.length) * 100 : 0;
  const averageDuration = runs.length > 0 ? runs.reduce((sum, run) => sum + run.durationMs, 0) / runs.length : 0;

  async function toggleJourney(journey: Journey) {
    setBusy(journey.id);
    try {
      const updated = await updateJourney(journey.id, { enabled: !journey.enabled });
      setJourneys((current) => current.map((item) => (item.id === journey.id ? updated : item)));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function removeJourney(journey: Journey) {
    if (!window.confirm(`Delete journey "${journey.name}"?`)) return;
    setBusy(journey.id);
    try {
      await deleteJourney(journey.id);
      setJourneys((current) => current.filter((item) => item.id !== journey.id));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function startRun(journey: Journey) {
    setBusy(journey.id);
    try {
      const run = await runJourney(journey.id);
      setRuns((current) => [run, ...current]);
      setError(null);
      window.setTimeout(() => {
        void getJourneyRun(run.id).then((fresh) => setRuns((current) => current.map((item) => (item.id === fresh.id ? fresh : item)))).catch(() => undefined);
        void listJourneyRuns(journey.id).then(setRuns).catch(() => undefined);
      }, 2500);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  function addStep() {
    setSteps((current) => [...current, { ...EMPTY_STEP, action: current.length === 0 ? "navigate" : "click" }]);
  }

  function removeStep(index: number) {
    setSteps((current) => current.filter((_, i) => i !== index));
  }

  function updateStep(index: number, patch: Partial<StepDraft>) {
    setSteps((current) => current.map((step, i) => (i === index ? { ...step, ...patch } : step)));
  }

  async function submitJourney(event: React.FormEvent) {
    event.preventDefault();
    if (!project) return;
    setBusy("create");
    try {
      const payloadSteps: JourneyStep[] = steps.map((step, index) => ({
        order: index + 1,
        action: step.action,
        selector: step.selector.trim() || undefined,
        value: step.value.trim() || undefined,
        timeoutMs: Number(step.timeoutMs) || undefined,
      }));
      const created = await createJourney({
        projectId: project.id,
        name: name.trim(),
        baseUrl: baseUrl.trim(),
        description: description.trim(),
        owner: owner.trim(),
        enabled: true,
        steps: payloadSteps,
      });
      setJourneys((current) => [created, ...current]);
      setSelectedId(created.id);
      setCreateOpen(false);
      setName("");
      setBaseUrl("");
      setDescription("");
      setOwner("");
      setSteps([{ ...EMPTY_STEP }]);
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
        icon="route"
        title="No project selected"
        description="Create a project to define and run journey tests."
      />
    );
  }

  return (
    <div className="space-y-5">
      <PageHeader
        eyebrow="Synthetic monitoring"
        title="Journey Test Runner"
        subtitle="Replay critical user journeys in a real browser and catch broken flows, missing tags or consent regressions."
        icon="route"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)} disabled={!project}>
            <Icon name="plus" className="h-3.5 w-3.5" />
            New journey
          </button>
        }
      />

      <ErrorBanner message={error} />

      {loading ? (
        <Loading label="Loading journeys" />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
            <KpiCard label="Journeys" value={formatNumber(journeys.length)} hint={`${journeys.filter((journey) => journey.enabled).length} enabled`} tone="indigo" icon="route" />
            <KpiCard label="Pass rate" value={formatPercent(passRate, 0)} hint={`${formatNumber(runs.length)} recent runs`} tone={passRate < 80 ? "rose" : "emerald"} icon="check" />
            <KpiCard label="Latest run" value={latest ? latest.status : "None"} hint={latest ? timeAgo(latest.startedAt) : "run a journey to begin"} tone="sky" icon="play" />
            <KpiCard label="Avg duration" value={duration(Math.round(averageDuration / 1000))} hint="recent executions" tone="amber" icon="clock" />
          </div>

          <div className="grid gap-4 xl:grid-cols-5">
            <Card className="xl:col-span-2">
              <CardHeader title="Definitions" subtitle="Saved journey documents" right={<Pill tone="slate">{formatNumber(journeys.length)}</Pill>} />
              <div className="border-b border-slate-800 px-4 py-2">
                <SearchInput value={query} onChange={setQuery} placeholder="Search journeys" />
              </div>
              {filtered.length === 0 ? (
                <EmptyState icon="route" title="No journeys" description="Create a journey to start monitoring critical flows." />
              ) : (
                <ul className="divide-y divide-slate-800">
                  {filtered.map((journey) => {
                    const active = journey.id === selectedId;
                    return (
                      <li key={journey.id}>
                        <button
                          className={`w-full px-4 py-3 text-left transition ${active ? "bg-indigo-950/30" : "hover:bg-slate-900/60"}`}
                          onClick={() => setSelectedId(journey.id)}
                        >
                          <div className="flex items-center justify-between gap-2">
                            <span className="truncate text-xs font-semibold text-slate-200">{journey.name}</span>
                            <Pill tone={journey.enabled ? "emerald" : "slate"}>{journey.enabled ? "enabled" : "paused"}</Pill>
                          </div>
                          <div className="mt-1 truncate font-mono text-[10px] text-slate-500">{journey.baseUrl || "no base URL"}</div>
                          <div className="mt-1 text-[10px] text-slate-500">{(journey.steps?.length ?? 0)} steps {journey.owner ? ` / ${journey.owner}` : ""}</div>
                        </button>
                      </li>
                    );
                  })}
                </ul>
              )}
            </Card>

            <div className="space-y-4 xl:col-span-3">
              {!selected ? (
                <Card><EmptyState icon="route" title="Select a journey" description="Choose a journey to inspect its steps and recent runs." /></Card>
              ) : (
                <>
                  <Card>
                    <CardHeader
                      title={selected.name}
                      subtitle={selected.description || selected.baseUrl || "No description"}
                      right={
                        <div className="flex items-center gap-2">
                          <Toggle checked={selected.enabled} onChange={() => void toggleJourney(selected)} label={`Enable ${selected.name}`} />
                          <button className="rounded-lg p-1.5 text-slate-500 transition hover:bg-rose-950/40 hover:text-rose-300" onClick={() => void removeJourney(selected)} aria-label={`Delete ${selected.name}`}>
                            <Icon name="trash" className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      }
                    />
                    <div className="space-y-3 p-4">
                      <ol className="space-y-2">
                        {(selected.steps ?? []).map((step, index) => (
                          <li key={`${step.order}-${index}`} className="flex items-start gap-2 rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2">
                            <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded border border-slate-700 bg-slate-900 text-[10px] text-slate-400">{index + 1}</span>
                            <span className="mt-0.5 text-indigo-300"><Icon name={ACTION_ICONS[step.action] ?? "arrow"} className="h-4 w-4" /></span>
                            <div className="min-w-0 flex-1">
                              <p className="text-xs font-semibold text-slate-200">{step.action}</p>
                              <p className="mono mt-0.5 truncate text-[10px] text-slate-500">{step.selector || step.value || "-"}</p>
                            </div>
                            {step.timeoutMs ? <span className="text-[10px] text-slate-500">{step.timeoutMs} ms</span> : null}
                          </li>
                        ))}
                        {(selected.steps ?? []).length === 0 && <li className="rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-4 text-center text-[11px] text-slate-500">Journey steps are stored in the raw definition only.</li>}
                      </ol>
                      <div className="flex justify-end">
                        <button className="btn-primary" onClick={() => void startRun(selected)} disabled={busy === selected.id || !selected.enabled}>
                          <Icon name="play" className="h-3.5 w-3.5" />
                          Run now
                        </button>
                      </div>
                    </div>
                  </Card>

                  <Card>
                    <CardHeader title="Recent runs" subtitle="Step outcomes and failures" right={<Pill tone="slate">{formatNumber(runs.length)}</Pill>} />
                    {runs.length === 0 ? (
                      <EmptyState icon="play" title="No runs yet" description="Run this journey to capture step-level results." />
                    ) : (
                      <div className="overflow-x-auto">
                        <table className="w-full min-w-[640px] text-xs" aria-label="Journey runs">
                          <thead className="table-head">
                            <tr>
                              <th scope="col" className="px-4 py-2.5">Run</th>
                              <th scope="col" className="px-4 py-2.5">Status</th>
                              <th scope="col" className="px-4 py-2.5">Steps</th>
                              <th scope="col" className="px-4 py-2.5">Duration</th>
                              <th scope="col" className="px-4 py-2.5">Started</th>
                            </tr>
                          </thead>
                          <tbody>
                            {runs.map((run) => (
                              <tr key={run.id} className="table-row">
                                <td className="px-4 py-3">
                                  <div className="mono text-[11px] text-slate-300">{run.id.slice(0, 10)}</div>
                                  {run.error && <div className="mt-0.5 max-w-[260px] truncate text-[10px] text-rose-300">{run.error}</div>}
                                </td>
                                <td className="px-4 py-3"><Pill tone={RUN_TONES[run.status] ?? "slate"}>{run.status}</Pill></td>
                                <td className="px-4 py-3 text-slate-400">{run.stepsPassed}/{run.stepsTotal}</td>
                                <td className="px-4 py-3 text-slate-400">{duration(Math.round(run.durationMs / 1000))}</td>
                                <td className="px-4 py-3 text-[11px] text-slate-500">{formatDate(run.startedAt)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </Card>
                </>
              )}
            </div>
          </div>
        </>
      )}

      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title="New journey" subtitle="Define a browser flow with navigate, click, fill and wait steps. A navigate step should point at the start URL.">
        <form onSubmit={submitJourney} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="journey-name">Name</label>
              <input id="journey-name" className="input" value={name} onChange={(event) => setName(event.target.value)} placeholder="Checkout flow" required />
            </div>
            <div>
              <label className="label" htmlFor="journey-owner">Owner</label>
              <input id="journey-owner" className="input" value={owner} onChange={(event) => setOwner(event.target.value)} placeholder="Privacy team" />
            </div>
          </div>
          <div>
            <label className="label" htmlFor="journey-base">Base URL</label>
            <input id="journey-base" className="input" value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} placeholder="https://www.example.com/" />
          </div>
          <div>
            <label className="label" htmlFor="journey-description">Description</label>
            <input id="journey-description" className="input" value={description} onChange={(event) => setDescription(event.target.value)} placeholder="Add to cart and start checkout" />
          </div>
          <div>
            <div className="flex items-center justify-between">
              <p className="label mb-0">Steps</p>
              <button type="button" className="btn" onClick={addStep}><Icon name="plus" className="h-3.5 w-3.5" />Add step</button>
            </div>
            <div className="mt-2 space-y-2">
              {steps.map((step, index) => (
                <div key={index} className="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-[10px] font-semibold uppercase tracking-wide text-slate-500">Step {index + 1}</span>
                    <button type="button" className="text-slate-500 hover:text-rose-300" onClick={() => removeStep(index)} aria-label={`Remove step ${index + 1}`}>
                      <Icon name="x" className="h-3.5 w-3.5" />
                    </button>
                  </div>
                  <div className="mt-2 grid grid-cols-4 gap-2">
                    <select className="input" value={step.action} onChange={(event) => updateStep(index, { action: event.target.value as JourneyStep["action"] })} aria-label={`Step ${index + 1} action`}>
                      <option value="navigate">Navigate</option>
                      <option value="click">Click</option>
                      <option value="fill">Fill</option>
                      <option value="wait">Wait</option>
                    </select>
                    <input className="input" value={step.selector} onChange={(event) => updateStep(index, { selector: event.target.value })} placeholder="Selector" aria-label={`Step ${index + 1} selector`} />
                    <input className="input" value={step.value} onChange={(event) => updateStep(index, { value: event.target.value })} placeholder={step.action === "navigate" ? "https://..." : "Value"} aria-label={`Step ${index + 1} value`} />
                    <input className="input mono" type="number" min={1000} value={step.timeoutMs} onChange={(event) => updateStep(index, { timeoutMs: event.target.value })} placeholder="Timeout ms" aria-label={`Step ${index + 1} timeout`} />
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setCreateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={!name.trim() || steps.length === 0 || busy === "create"}>Create journey</button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
