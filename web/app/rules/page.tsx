"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useApp } from "../../components/AppShell";
import { Icon } from "../../components/icons";
import {
  Card,
  CardHeader,
  EmptyState,
  ErrorBanner,
  Loading,
  Modal,
  SeverityBadge,
  StatusBadge,
} from "../../components/ui";
import {
  createRule,
  deleteRule,
  getValidation,
  installStarterRules,
  listAudits,
  listRules,
  updateRule,
  type AuditRule,
  type AuditRuleInput,
  type RuleStat,
} from "../../lib/api";
import { formatPercent, shortId, timeAgo } from "../../lib/format";

// Must match the engine's supported targets (internal/rules + governance
// validation): beacon, cookie, event, network, page, request, tag.
const TARGETS = ["beacon", "request", "cookie", "tag", "event", "network", "page"] as const;
const OPERATORS = ["exists", "not_exists", "equals", "not_equals", "matches", "not_matches"] as const;
const SEVERITIES = ["info", "low", "medium", "high", "critical"] as const;

const emptyRule: AuditRuleInput = {
  name: "",
  description: "",
  enabled: true,
  vendor: "",
  target: "request",
  operator: "exists",
  urlPattern: "",
  parameter: "",
  event: "",
  expected: "",
  severity: "high",
};

function RulesPage() {
  const { project, loadingProjects } = useApp();
  const [rules, setRules] = useState<AuditRule[]>([]);
  const [stats, setStats] = useState<RuleStat[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<AuditRule | null>(null);
  const [form, setForm] = useState<AuditRuleInput>(emptyRule);
  const [busy, setBusy] = useState(false);
  const [filter, setFilter] = useState<"all" | "enabled" | "disabled">("all");
  const [vendor, setVendor] = useState("all");

  const refresh = useCallback(async () => {
    if (!project) {
      setRules([]);
      setStats([]);
      setLoading(false);
      return;
    }
    try {
      const [r, audits] = await Promise.all([listRules(project.id), listAudits(project.id, 1)]);
      setRules(r);
      if (audits[0]) {
        try {
          const v = await getValidation(audits[0].runId);
          setStats(v.ruleStats ?? []);
        } catch {
          setStats([]);
        }
      } else {
        setStats([]);
      }
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
    rules.forEach((r) => r.vendor && set.add(r.vendor));
    return [...set].sort();
  }, [rules]);

  const filtered = useMemo(() => {
    return rules.filter((r) => {
      if (filter === "enabled" && !r.enabled) return false;
      if (filter === "disabled" && r.enabled) return false;
      if (vendor !== "all" && r.vendor !== vendor) return false;
      return true;
    });
  }, [rules, filter, vendor]);

  const statMap = useMemo(() => {
    const map = new Map<string, RuleStat>();
    stats.forEach((s) => map.set(s.rule.id, s));
    return map;
  }, [stats]);

  function openCreate() {
    setEditing(null);
    setForm(emptyRule);
    setModalOpen(true);
  }

function openEdit(rule: AuditRule) {
  setEditing(rule);
  setForm({
    name: rule.name,
    description: rule.description,
    enabled: rule.enabled,
    vendor: rule.vendor ?? "",
    target: rule.target,
    operator: rule.operator,
    urlPattern: rule.urlPattern ?? "",
    parameter: rule.parameter ?? "",
    event: rule.event ?? "",
    expected: rule.expected ?? "",
    severity: rule.severity,
  });
  setModalOpen(true);
}


  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!project) return;
    setBusy(true);
    setError(null);
    try {
      if (editing) {
        await updateRule(editing.id, form);
      } else {
        await createRule({ ...form, projectId: project.id });
      }
      setModalOpen(false);
      await refresh();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  async function toggle(rule: AuditRule) {
    if (!project) return;
    try {
      await updateRule(rule.id, { enabled: !rule.enabled });
      await refresh();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  async function remove(rule: AuditRule) {
    if (!project) return;
    if (!confirm("Delete this rule? Findings generated from it remain in history.")) return;
    try {
      await deleteRule(rule.id);
      await refresh();
    } catch (err) {
      setError((err as Error).message);
    }
  }

  async function loadStarter() {
    if (!project) return;
    setBusy(true);
    try {
      await installStarterRules(project.id);
      await refresh();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  if (loadingProjects) return <Loading label="Loading workspace" />;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-[0.25em] text-indigo-400">Policy engine</p>
          <h1 className="mt-1 text-xl font-bold text-white sm:text-2xl">Rules Configurator</h1>
          <p className="mt-1 text-xs text-slate-500">Define positive and negative tag governance rules. Rules persist in memory and PostgreSQL.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button className="btn h-9" onClick={loadStarter} disabled={!project || busy}>
            <Icon name="download" className="h-3.5 w-3.5" />
            Load starter set
          </button>
          <button className="btn-primary h-9" onClick={openCreate} disabled={!project}>
            <Icon name="plus" className="h-3.5 w-3.5" />
            New rule
          </button>
        </div>
      </div>

      <ErrorBanner message={error} />

      {!project ? (
        <Card><EmptyState icon="globe" title="No workspace yet" description="Create a project before configuring rules." /></Card>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
            <Kpi label="Total rules" value={rules.length} />
            <Kpi label="Enabled" value={rules.filter((r) => r.enabled).length} />
            <Kpi label="Vendors scoped" value={vendors.length} />
            <Kpi label="Failures (latest runs)" value={stats.reduce((sum, s) => sum + s.failed, 0)} />
          </div>

          <Card>
            <CardHeader
              title="Governance rules"
              subtitle={`${filtered.length} rule${filtered.length === 1 ? "" : "s"} shown`}
              right={
                <div className="flex flex-wrap items-center gap-2">
                  <select className="input h-8 w-28 py-1 text-xs" value={filter} onChange={(e) => setFilter(e.target.value as any)}>
                    <option value="all">All</option>
                    <option value="enabled">Enabled</option>
                    <option value="disabled">Disabled</option>
                  </select>
                  <select className="input h-8 w-32 py-1 text-xs" value={vendor} onChange={(e) => setVendor(e.target.value)}>
                    <option value="all">All vendors</option>
                    {vendors.map((v) => <option key={v} value={v}>{v}</option>)}
                  </select>
                </div>
              }
            />
            <div className="overflow-x-auto">
              <table className="w-full min-w-[920px] text-xs">
                <thead className="table-head">
                  <tr>
                    <th className="px-4 py-2.5">Rule</th>
                    <th className="px-4 py-2.5">Scope</th>
                    <th className="px-4 py-2.5">Match</th>
                    <th className="px-4 py-2.5">Severity</th>
                    <th className="px-4 py-2.5">Status</th>
                    <th className="px-4 py-2.5 text-right">Pass / Fail</th>
                    <th className="px-4 py-2.5 text-right">Pass rate</th>
                    <th className="px-4 py-2.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {loading && <tr><td colSpan={8} className="px-4 py-8 text-center text-slate-500">Loading rulesâ€¦</td></tr>}
                  {!loading && filtered.length === 0 && (
                    <tr>
                      <td colSpan={8} className="px-4 py-10 text-center">
                        <p className="text-slate-500">No rules match filters.</p>
                        <div className="mt-3 flex justify-center gap-2">
                          <button className="btn" onClick={loadStarter} disabled={busy}><Icon name="download" className="h-3.5 w-3.5" /> Load starters</button>
                          <button className="btn-primary" onClick={openCreate}><Icon name="plus" className="h-3.5 w-3.5" /> Create rule</button>
                        </div>
                      </td>
                    </tr>
                  )}
                  {!loading && filtered.map((rule) => {
                    const s = statMap.get(rule.id);
                    return (
                      <tr key={rule.id} className="table-row">
                        <td className="px-4 py-2.5">
                          <div className="font-semibold text-slate-200">{rule.name}</div>
                          <div className="mt-0.5 text-slate-500">{rule.description || "â€”"}</div>
                          <div className="mono mt-0.5 text-[10px] text-slate-600">{shortId(rule.id)}</div>
                        </td>
                        <td className="px-4 py-2.5">
                          <div className="flex flex-wrap gap-1.5">
                            <span className="pill">target: {rule.target}</span>
                            {rule.vendor && <span className="pill border-indigo-800/50 bg-indigo-950/40 text-indigo-300">{rule.vendor}</span>}
                          </div>
                        </td>
                        <td className="px-4 py-2.5">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <span className="pill">{rule.operator}</span>
                            {rule.urlPattern && <span className="mono text-[10px] text-slate-500">url:{rule.urlPattern}</span>}
                            {rule.parameter && <span className="mono text-[10px] text-slate-500">param:{rule.parameter}</span>}{rule.event && <span className="mono text-[10px] text-slate-500">event:{rule.event}</span>}
                          </div>
                          <p className="mt-1 text-[10px] text-slate-600">
                            Positive passes if any match satisfies. Negative fails if any forbidden match exists.
                          </p>
                        </td>
                        <td className="px-4 py-2.5"><SeverityBadge severity={rule.severity} /></td>
                        <td className="px-4 py-2.5"><StatusBadge status={rule.enabled ? "succeeded" : "canceled"} /></td>
                        <td className="px-4 py-2.5 text-right text-slate-400">
                          {s ? `${s.passed} / ${s.failed}` : "â€”"}
                        </td>
                        <td className="px-4 py-2.5 text-right font-semibold text-slate-200">
                          {s ? formatPercent(s.passRate) : "â€”"}
                        </td>
                        <td className="px-4 py-2.5 text-right">
                          <div className="flex justify-end gap-1.5">
                            <button className="btn h-7 px-2 py-0.5" onClick={() => toggle(rule)}>
                              {rule.enabled ? "Disable" : "Enable"}
                            </button>
                            <button className="btn h-7 px-2 py-0.5" onClick={() => openEdit(rule)}>
                              <Icon name="edit" className="h-3 w-3" /> Edit
                            </button>
                            <button className="btn h-7 px-2 py-0.5 text-rose-300" onClick={() => remove(rule)}>
                              <Icon name="trash" className="h-3 w-3" /> Delete
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </Card>

          {stats.length > 0 && (
            <Card>
              <CardHeader title="Rule performance" subtitle="Aggregated across the latest evaluated runs" />
              <div className="grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-4">
                {stats.slice(0, 8).map((s) => (
                  <div key={s.rule.id} className="rounded-xl border border-slate-800 bg-slate-900/60 p-3">
                    <div className="flex items-center justify-between gap-2">
                      <p className="truncate text-xs font-semibold text-slate-200">{s.rule.name}</p>
                      <SeverityBadge severity={s.rule.severity} />
                    </div>
                    <div className="mt-2 flex items-end justify-between">
                      <div>
                        <p className="text-[10px] text-slate-500">Pass rate</p>
                        <p className="mt-0.5 text-lg font-extrabold text-white">{formatPercent(s.passRate)}</p>
                      </div>
                      <div className="text-right text-[11px] text-slate-500">
                        <div>{s.passed} pass Â· {s.failed} fail</div>
                        <div>updated {timeAgo(new Date().toISOString())}</div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </>
      )}

      <Modal open={modalOpen} onClose={() => setModalOpen(false)} title={editing ? "Edit rule" : "Create rule"} subtitle="Define when to flag or enforce tag behavior." width="max-w-2xl">
        <form onSubmit={submit} className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <label className="label">Rule name</label>
              <input className="input" required value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
            </div>
            <div className="sm:col-span-2">
              <label className="label">Description</label>
              <input className="input" value={form.description} onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))} placeholder="Explain the governance requirement" />
            </div>
            <div>
              <label className="label">Vendor</label>
              <input className="input" value={form.vendor ?? ""} onChange={(e) => setForm((f) => ({ ...f, vendor: e.target.value }))} placeholder="Google, Meta, Adobeâ€¦" />
            </div>
            <div>
              <label className="label">Severity</label>
              <select className="input" value={form.severity} onChange={(e) => setForm((f) => ({ ...f, severity: e.target.value as any }))}>
                {SEVERITIES.map((s) => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div>
              <label className="label">Target</label>
              <select className="input" value={form.target} onChange={(e) => setForm((f) => ({ ...f, target: e.target.value as any }))}>
                {TARGETS.map((t) => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label className="label">Operator</label>
              <select className="input" value={form.operator} onChange={(e) => setForm((f) => ({ ...f, operator: e.target.value as any }))}>
                {OPERATORS.map((o) => <option key={o} value={o}>{o}</option>)}
              </select>
            </div>
            <div>
              <label className="label">URL pattern (optional)</label>
              <input className="input mono" value={form.urlPattern ?? ""} onChange={(e) => setForm((f) => ({ ...f, urlPattern: e.target.value }))} placeholder="tracking.example.com or /collect" />
            </div>
            <div>
              <label className="label">Parameter / event</label>
              <input className="input mono" value={form.parameter ?? ""} onChange={(e) => setForm((f) => ({ ...f, parameter: e.target.value }))} placeholder="query key, cookie name, tag id" />
            </div>
            <div>
              <label className="label">Expected value</label>
              <input className="input mono" value={form.expected ?? ""} onChange={(e) => setForm((f) => ({ ...f, expected: e.target.value }))} placeholder="Exact value or regex" />
            </div>
            <div className="sm:col-span-2 flex items-center gap-2">
              <input id="enabled" type="checkbox" checked={form.enabled} onChange={(e) => setForm((f) => ({ ...f, enabled: e.target.checked }))} />
              <label htmlFor="enabled" className="text-xs text-slate-300">Rule enabled</label>
            </div>
          </div>
          <div className="rounded-lg border border-slate-800 bg-slate-900/60 p-3 text-[11px] text-slate-500">
            <strong className="text-slate-300">Semantics:</strong> <span className="text-emerald-400">positive</span> (exists/equals/matches) pass if any matching evidence satisfies the condition.{" "}
            <span className="text-rose-400">negative</span> (not_exists/not_equals/not_matches) fail if any forbidden evidence matches.
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setModalOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={busy || !form.name}>
              {busy ? "Savingâ€¦" : editing ? "Save changes" : "Create rule"}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

function Kpi({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/70 p-4">
      <p className="text-[10px] uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-2xl font-extrabold text-white">{value}</p>
    </div>
  );
}

export default RulesPage;



