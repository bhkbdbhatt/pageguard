"use client";

import { useEffect, useMemo, useState } from "react";
import { DemoBanner, DemoPageHeader } from "../../components/demo";
import { Icon, type IconName } from "../../components/icons";
import {
  Card,
  EmptyState,
  KpiCard,
  Modal,
  Pill,
  SearchInput,
} from "../../components/ui";
import {
  DEMO_JOURNEYS,
  type DemoJourney,
  type JourneyAction,
  type JourneyStepStatus,
} from "../../lib/demo";
import { formatNumber, formatPercent } from "../../lib/format";

const ACTION_ICONS: Record<JourneyAction, IconName> = {
  navigate: "globe",
  click: "arrow",
  type: "edit",
  wait: "clock",
  assert: "check-circle",
  extract: "code",
};

const STATUS_TONES: Record<JourneyStepStatus, "emerald" | "rose" | "sky" | "slate"> = {
  passed: "emerald",
  failed: "rose",
  running: "sky",
  pending: "slate",
};

export default function JourneysPage() {
  const [journeys, setJourneys] = useState<DemoJourney[]>(DEMO_JOURNEYS);
  const [selectedId, setSelectedId] = useState(DEMO_JOURNEYS[0].id);
  const [query, setQuery] = useState("");
  const [run, setRun] = useState<{ journeyId: string; step: number; done: boolean } | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [description, setDescription] = useState("");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return journeys;
    return journeys.filter((journey) =>
      [journey.name, journey.url, journey.description, journey.owner].some((value) =>
        value.toLowerCase().includes(q),
      ),
    );
  }, [journeys, query]);

  const selected = journeys.find((journey) => journey.id === selectedId) ?? journeys[0];
  const activeRun = run && selected && run.journeyId === selected.id ? run : null;

  useEffect(() => {
    if (!run || run.done) return;
    const journey = journeys.find((item) => item.id === run.journeyId);
    if (!journey) return;
    const timer = window.setTimeout(() => {
      setRun((current) => {
        if (!current) return current;
        const total = journey.steps.length;
        if (current.step + 1 >= total) return { ...current, step: total, done: true };
        return { ...current, step: current.step + 1 };
      });
    }, 650);
    return () => window.clearTimeout(timer);
  }, [journeys, run]);

  const averagePassRate =
    journeys.length > 0
      ? Math.round(journeys.reduce((sum, journey) => sum + journey.passRate, 0) / journeys.length)
      : 0;
  const averageSteps =
    journeys.length > 0
      ? Math.round(journeys.reduce((sum, journey) => sum + journey.steps.length, 0) / journeys.length)
      : 0;

  function statusFor(index: number): JourneyStepStatus {
    if (!selected) return "pending";
    if (!activeRun) return selected.steps[index]?.status ?? "pending";
    const failIndex = selected.steps.findIndex((step) => step.status === "failed");
    if (activeRun.done) {
      if (failIndex >= 0 && index >= failIndex) return index === failIndex ? "failed" : "pending";
      return "passed";
    }
    if (index < activeRun.step) return "passed";
    if (index === activeRun.step) return "running";
    return "pending";
  }

  function startRun() {
    if (!selected) return;
    setRun({ journeyId: selected.id, step: 0, done: false });
  }

  function createJourney(event: React.FormEvent) {
    event.preventDefault();
    const next: DemoJourney = {
      id: `journey-${Date.now()}`,
      name: name.trim() || "Untitled journey",
      url: url.trim() || "https://www.acme.com/",
      description: description.trim() || "New demo journey with starter steps.",
      owner: "You",
      passRate: 0,
      lastRun: "Never",
      avgDuration: "—",
      steps: [
        { order: 1, action: "navigate", target: "Start page", selector: "/", status: "pending" },
        { order: 2, action: "click", target: "Primary CTA", selector: "[data-testid=cta]", status: "pending" },
        { order: 3, action: "assert", target: "dataLayer event", selector: "dataLayer", expected: "event = cta_click", status: "pending" },
      ],
      runs: [],
    };
    setJourneys((current) => [next, ...current]);
    setSelectedId(next.id);
    setCreateOpen(false);
    setName("");
    setUrl("");
    setDescription("");
  }

  const displayStep = selected
    ? selected.steps[Math.min(activeRun?.step ?? 0, Math.max(0, selected.steps.length - 1))]
    : undefined;

  return (
    <div className="space-y-5">
      <DemoPageHeader
        eyebrow="Journey testing"
        title="Journey Test Runner"
        subtitle="Script multi-step user flows, replay them in a browser and assert the data layer at every step."
        icon="route"
        actions={
          <button className="btn-primary" onClick={() => setCreateOpen(true)}>
            <Icon name="plus" className="h-3.5 w-3.5" />
            New journey
          </button>
        }
      />

      <DemoBanner />

      <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        <KpiCard label="Journeys" value={formatNumber(journeys.length)} hint={`${formatNumber(averageSteps)} steps on average`} tone="indigo" icon="route" />
        <KpiCard label="Average pass rate" value={formatPercent(averagePassRate, 0)} hint="across all journeys" tone="emerald" icon="check-circle" />
        <KpiCard label="Runs this week" value={formatNumber(19)} hint="2 failed · 17 passed" tone="sky" icon="play" />
        <KpiCard label="Average duration" value="1m 05s" hint="browser render + assertions" tone="amber" icon="clock" />
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <Card className="xl:col-span-1">
          <div className="border-b border-slate-800 px-4 py-3">
            <SearchInput value={query} onChange={setQuery} placeholder="Search journeys" />
          </div>
          <ul className="max-h-[560px] divide-y divide-slate-800 overflow-y-auto">
            {filtered.map((journey) => {
              const active = journey.id === selected?.id;
              return (
                <li key={journey.id}>
                  <button
                    onClick={() => {
                      setSelectedId(journey.id);
                      setRun(null);
                    }}
                    className={`flex w-full flex-col gap-2 px-4 py-3 text-left transition ${
                      active ? "bg-indigo-950/30" : "hover:bg-slate-800/30"
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className={`text-xs font-semibold ${active ? "text-indigo-200" : "text-slate-200"}`}>{journey.name}</span>
                      <Pill tone={journey.passRate >= 90 ? "emerald" : journey.passRate >= 70 ? "amber" : "rose"}>
                        {journey.passRate}%
                      </Pill>
                    </div>
                    <span className="truncate font-mono text-[10px] text-slate-500">{journey.url}</span>
                    <span className="flex items-center gap-2 text-[10px] text-slate-500">
                      <Icon name="layers" className="h-3 w-3" />
                      {journey.steps.length} steps
                      <span className="text-slate-700">·</span>
                      <Icon name="clock" className="h-3 w-3" />
                      {journey.lastRun}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        </Card>

        <Card className="xl:col-span-2">
          {!selected ? (
            <EmptyState icon="route" title="No journey selected" description="Create a journey to start asserting data layer events." />
          ) : (
            <>
              <div className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-800 px-5 py-4">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="text-sm font-semibold text-white">{selected.name}</h2>
                    <Pill tone={selected.passRate >= 90 ? "emerald" : selected.passRate >= 70 ? "amber" : "rose"}>
                      {selected.passRate}% pass
                    </Pill>
                  </div>
                  <p className="mt-1 max-w-xl text-xs leading-relaxed text-slate-500">{selected.description}</p>
                  <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px] text-slate-500">
                    <span className="inline-flex items-center gap-1"><Icon name="globe" className="h-3 w-3" />{selected.url}</span>
                    <span className="inline-flex items-center gap-1"><Icon name="tag" className="h-3 w-3" />{selected.owner}</span>
                    <span className="inline-flex items-center gap-1"><Icon name="clock" className="h-3 w-3" />avg {selected.avgDuration}</span>
                  </div>
                </div>
                <button
                  className="btn-primary"
                  onClick={startRun}
                  disabled={!!activeRun && !activeRun.done}
                >
                  <Icon name={activeRun && !activeRun.done ? "pause" : "play"} className="h-3.5 w-3.5" />
                  {activeRun && !activeRun.done ? "Running…" : "Run journey"}
                </button>
              </div>

              <div className="space-y-4 p-4">
                <div className="overflow-hidden rounded-xl border border-slate-800">
                  <div className="flex items-center gap-1.5 border-b border-slate-800 bg-slate-900 px-3 py-2">
                    <span className="h-2 w-2 rounded-full bg-rose-400/70" />
                    <span className="h-2 w-2 rounded-full bg-amber-400/70" />
                    <span className="h-2 w-2 rounded-full bg-emerald-400/70" />
                    <span className="ml-2 flex-1 truncate rounded bg-slate-950 px-2 py-0.5 font-mono text-[10px] text-slate-500">
                      {selected.url}
                    </span>
                  </div>
                  <div className="relative h-48 bg-gradient-to-br from-slate-950 to-slate-900 p-4">
                    <div className="grid grid-cols-3 gap-2 opacity-60">
                      <div className="h-16 rounded-lg border border-slate-800 bg-slate-900/80" />
                      <div className="h-16 rounded-lg border border-slate-800 bg-slate-900/80" />
                      <div className="h-16 rounded-lg border border-slate-800 bg-slate-900/80" />
                    </div>
                    <div className="mt-3 h-2 w-2/3 rounded bg-slate-800" />
                    <div className="mt-2 h-2 w-1/2 rounded bg-slate-800" />
                    <div className="absolute inset-x-3 bottom-3 flex items-center gap-2 rounded-lg border border-indigo-800/50 bg-indigo-950/80 px-3 py-2 text-[10px] text-indigo-200">
                      <Icon name={displayStep ? ACTION_ICONS[displayStep.action] : "route"} className="h-3.5 w-3.5" />
                      <span className="truncate">
                        {displayStep
                          ? `Step ${displayStep.order} · ${displayStep.action} · ${displayStep.target}`
                          : "Ready to run"}
                      </span>
                      {activeRun && (
                        <span className="ml-auto shrink-0 font-mono text-indigo-300">
                          {Math.min(activeRun.step + 1, selected.steps.length)}/{selected.steps.length}
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                <ol className="space-y-2">
                  {selected.steps.map((step, index) => {
                    const status = statusFor(index);
                    return (
                      <li
                        key={`${selected.id}-${step.order}`}
                        className={`flex items-start gap-3 rounded-xl border px-3 py-2.5 transition ${
                          status === "running"
                            ? "border-sky-800/60 bg-sky-950/25"
                            : status === "failed"
                              ? "border-rose-800/50 bg-rose-950/20"
                              : "border-slate-800 bg-slate-950/30"
                        }`}
                      >
                        <span
                          className={`mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border text-[10px] font-bold ${
                            status === "passed"
                              ? "border-emerald-800/60 bg-emerald-950/50 text-emerald-300"
                              : status === "failed"
                                ? "border-rose-800/60 bg-rose-950/50 text-rose-300"
                                : status === "running"
                                  ? "animate-pulse border-sky-800/60 bg-sky-950/50 text-sky-300"
                                  : "border-slate-700 bg-slate-800/60 text-slate-500"
                          }`}
                        >
                          {step.order}
                        </span>
                        <Icon name={ACTION_ICONS[step.action]} className="mt-1 h-3.5 w-3.5 shrink-0 text-slate-500" />
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="text-xs font-semibold text-slate-200">{step.target}</span>
                            <span className="rounded bg-slate-800/80 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-slate-400">{step.action}</span>
                          </div>
                          <div className="mt-1 truncate font-mono text-[10px] text-slate-500">{step.selector}</div>
                          {(step.value || step.expected) && (
                            <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-slate-500">
                              {step.value && <span>value: <span className="font-mono text-slate-400">{step.value}</span></span>}
                              {step.expected && <span>expect: <span className="font-mono text-slate-400">{step.expected}</span></span>}
                            </div>
                          )}
                        </div>
                        <Pill tone={STATUS_TONES[status]}>{status}</Pill>
                      </li>
                    );
                  })}
                </ol>
              </div>
            </>
          )}
        </Card>
      </div>

      {selected && (
        <Card>
          <div className="flex items-center justify-between border-b border-slate-800 px-5 py-4">
            <div>
              <h2 className="text-sm font-semibold text-white">Recent runs</h2>
              <p className="mt-0.5 text-xs text-slate-500">Step-level outcomes from the latest executions.</p>
            </div>
            <Pill tone="slate">{formatNumber(selected.runs.length)} runs</Pill>
          </div>
          {selected.runs.length === 0 ? (
            <EmptyState icon="play" title="No runs yet" description="Run this journey to capture step evidence and timing." />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[640px] text-xs" aria-label="Journey run history">
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
                  {selected.runs.map((item) => (
                    <tr key={item.id} className="table-row">
                      <td className="mono px-4 py-3 text-[11px] text-slate-300">{item.id}</td>
                      <td className="px-4 py-3">
                        <Pill tone={item.status === "passed" ? "emerald" : item.status === "failed" ? "rose" : "sky"}>{item.status}</Pill>
                      </td>
                      <td className="px-4 py-3 text-slate-400">{item.stepsPassed}/{item.stepsTotal}</td>
                      <td className="px-4 py-3 text-slate-400">{item.durationSec}s</td>
                      <td className="px-4 py-3 text-slate-400">{item.atLabel}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      )}

      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="New journey"
        subtitle="Create a demo journey with starter navigate, click and assert steps."
      >
        <form onSubmit={createJourney} className="space-y-4">
          <div>
            <label className="label" htmlFor="journey-name">Journey name</label>
            <input id="journey-name" className="input" placeholder="Guest checkout" value={name} onChange={(event) => setName(event.target.value)} />
          </div>
          <div>
            <label className="label" htmlFor="journey-url">Start URL</label>
            <input id="journey-url" className="input" placeholder="https://www.acme.com/checkout" value={url} onChange={(event) => setUrl(event.target.value)} />
          </div>
          <div>
            <label className="label" htmlFor="journey-description">Description</label>
            <textarea
              id="journey-description"
              className="input min-h-[76px] resize-y"
              placeholder="What should this journey prove?"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
            />
          </div>
          <p className="rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2 text-[11px] text-slate-500">
            Starter steps are generated automatically. Step editing arrives with the production API.
          </p>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setCreateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={!name.trim()}>
              <Icon name="plus" className="h-3.5 w-3.5" />
              Create journey
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
