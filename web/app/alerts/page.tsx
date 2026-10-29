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
  SeverityBadge,
  Tabs,
  Toggle,
} from "../../components/ui";
import {
  createAlertChannel,
  createAlertRule,
  deleteAlertChannel,
  deleteAlertRule,
  listAlertChannels,
  listAlertDeliveries,
  listAlertIncidents,
  listAlertRules,
  testAlertChannel,
  updateAlertIncident,
  updateAlertRule,
  type AlertChannel,
  type AlertChannelKind,
  type AlertDelivery,
  type AlertIncident,
  type AlertRule,
  type IncidentStatus,
  type Severity,
} from "../../lib/api";
import { formatDate, formatNumber, timeAgo } from "../../lib/format";

type Tab = "rules" | "feed" | "channels";

const SEVERITIES: Severity[] = ["info", "low", "medium", "high", "critical"];
const CHANNEL_KINDS: Array<{ id: AlertChannelKind; label: string }> = [
  { id: "slack", label: "Slack" },
  { id: "email", label: "Email" },
  { id: "webhook", label: "Webhook" },
  { id: "pagerduty", label: "PagerDuty" },
];

const STATUS_TONES: Record<IncidentStatus, "rose" | "amber" | "emerald"> = {
  open: "rose",
  acknowledged: "amber",
  resolved: "emerald",
};

function channelIcon(kind: string): IconName {
  const icons: Record<string, IconName> = { slack: "message", email: "mail", webhook: "webhook", pagerduty: "bell" };
  return icons[kind] ?? "bell";
}

export default function AlertsPage() {
  const { project, loadingProjects } = useApp();
  const [tab, setTab] = useState<Tab>("rules");
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [channels, setChannels] = useState<AlertChannel[]>([]);
  const [incidents, setIncidents] = useState<AlertIncident[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const [ruleOpen, setRuleOpen] = useState(false);
  const [ruleName, setRuleName] = useState("");
  const [ruleSeverity, setRuleSeverity] = useState<Severity>("high");
  const [conditionKind, setConditionKind] = useState<"severity" | "failure" | "tags">("severity");
  const [conditionSeverity, setConditionSeverity] = useState<Severity>("high");
  const [tagList, setTagList] = useState("");
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);

  const [channelOpen, setChannelOpen] = useState(false);
  const [channelName, setChannelName] = useState("");
  const [channelKind, setChannelKind] = useState<AlertChannelKind>("slack");
  const [channelTarget, setChannelTarget] = useState("");
  const [channelSecret, setChannelSecret] = useState("");

  const [selectedIncident, setSelectedIncident] = useState<AlertIncident | null>(null);
  const [deliveries, setDeliveries] = useState<AlertDelivery[]>([]);

  const load = useCallback(async () => {
    if (!project) {
      setRules([]);
      setChannels([]);
      setIncidents([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const [nextRules, nextChannels, nextIncidents] = await Promise.all([
        listAlertRules(project.id),
        listAlertChannels(project.id),
        listAlertIncidents(project.id),
      ]);
      setRules(nextRules);
      setChannels(nextChannels);
      setIncidents(nextIncidents);
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
    if (!selectedIncident) {
      setDeliveries([]);
      return;
    }
    listAlertDeliveries(selectedIncident.id)
      .then(setDeliveries)
      .catch((err) => setError((err as Error).message));
  }, [selectedIncident]);

  const openIncidents = incidents.filter((incident) => incident.status === "open").length;
  const activeRules = rules.filter((rule) => rule.enabled).length;
  const connectedChannels = channels.filter((channel) => channel.connected).length;

  const sortedIncidents = useMemo(
    () => [...incidents].sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1)),
    [incidents],
  );

  async function toggleRule(rule: AlertRule) {
    setBusy(rule.id);
    try {
      const updated = await updateAlertRule(rule.id, { enabled: !rule.enabled });
      setRules((current) => current.map((item) => (item.id === rule.id ? updated : item)));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function removeRule(rule: AlertRule) {
    if (!window.confirm(`Delete alert rule "${rule.name}"?`)) return;
    setBusy(rule.id);
    try {
      await deleteAlertRule(rule.id);
      setRules((current) => current.filter((item) => item.id !== rule.id));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function submitRule(event: React.FormEvent) {
    event.preventDefault();
    if (!project) return;
    setBusy("rule");
    try {
      const condition =
        conditionKind === "failure"
          ? { onRunFailure: true }
          : conditionKind === "tags"
            ? { tagMissing: tagList.split(",").map((tag) => tag.trim()).filter(Boolean) }
            : { minSeverity: conditionSeverity };
      const created = await createAlertRule({
        projectId: project.id,
        name: ruleName.trim(),
        severity: ruleSeverity,
        condition,
        channelIds: selectedChannels,
      });
      setRules((current) => [created, ...current]);
      setRuleOpen(false);
      setRuleName("");
      setConditionKind("severity");
      setConditionSeverity("high");
      setTagList("");
      setSelectedChannels([]);
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function submitChannel(event: React.FormEvent) {
    event.preventDefault();
    if (!project) return;
    setBusy("channel");
    try {
      const created = await createAlertChannel({
        projectId: project.id,
        name: channelName.trim(),
        kind: channelKind,
        target: channelTarget.trim(),
        secretRef: channelSecret.trim() || undefined,
      });
      setChannels((current) => [...current, created]);
      setChannelOpen(false);
      setChannelName("");
      setChannelKind("slack");
      setChannelTarget("");
      setChannelSecret("");
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function testChannel(channel: AlertChannel) {
    setBusy(channel.id);
    try {
      const delivery = await testAlertChannel(channel.id);
      setNotice(`Test notification to ${channel.name}: ${delivery.status}.`);
      setError(null);
      void load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function removeChannel(channel: AlertChannel) {
    if (!window.confirm(`Delete channel "${channel.name}"?`)) return;
    setBusy(channel.id);
    try {
      await deleteAlertChannel(channel.id);
      setChannels((current) => current.filter((item) => item.id !== channel.id));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function setIncidentStatus(incident: AlertIncident, status: IncidentStatus) {
    setBusy(incident.id);
    try {
      const updated = await updateAlertIncident(incident.id, { status, by: "console" });
      setIncidents((current) => current.map((item) => (item.id === incident.id ? updated : item)));
      setSelectedIncident((current) => (current?.id === updated.id ? updated : current));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  if (!project && !loadingProjects) {
    return (
      <EmptyState
        icon="bell"
        title="No project selected"
        description="Create a project and run an audit to start evaluating alert rules."
      />
    );
  }

  return (
    <div className="space-y-5">
      <PageHeader
        eyebrow="Notifications"
        title="Alerts & Notifications"
        subtitle="Trigger incidents on failed runs, severe findings or missing tags, then route them to Slack, webhooks, email or PagerDuty."
        icon="bell"
        actions={
          <button className="btn-primary" onClick={() => (tab === "channels" ? setChannelOpen(true) : setRuleOpen(true))} disabled={!project}>
            <Icon name="plus" className="h-3.5 w-3.5" />
            {tab === "channels" ? "New channel" : "New rule"}
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
        <Loading label="Loading alerts" />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
            <KpiCard label="Open incidents" value={formatNumber(openIncidents)} hint="awaiting acknowledgement" tone={openIncidents > 0 ? "rose" : "emerald"} icon="alert" />
            <KpiCard label="Active rules" value={`${activeRules}/${rules.length}`} hint="evaluated after every run" tone="indigo" icon="shield" />
            <KpiCard label="Channels" value={formatNumber(channels.length)} hint={`${connectedChannels} connected`} tone="sky" icon="bell" />
            <KpiCard label="Total incidents" value={formatNumber(incidents.length)} hint="recent alert history" tone="amber" icon="activity" />
          </div>

          <Card>
            <div className="border-b border-slate-800 px-4 py-3">
              <Tabs
                value={tab}
                onChange={(id) => setTab(id as Tab)}
                tabs={[
                  { id: "rules", label: "Rules", count: rules.length },
                  { id: "feed", label: "Incident feed", count: incidents.length },
                  { id: "channels", label: "Channels", count: channels.length },
                ]}
              />
            </div>

            {tab === "rules" && (
              rules.length === 0 ? (
                <EmptyState icon="shield" title="No alert rules" description="Create a rule to be notified when a run fails or findings cross your severity threshold." action={<button className="btn-primary" onClick={() => setRuleOpen(true)}><Icon name="plus" className="h-3.5 w-3.5" />New rule</button>} />
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[820px] text-xs" aria-label="Alert rules">
                    <thead className="table-head">
                      <tr>
                        <th scope="col" className="px-4 py-2.5">Rule</th>
                        <th scope="col" className="px-4 py-2.5">Severity</th>
                        <th scope="col" className="px-4 py-2.5">Condition</th>
                        <th scope="col" className="px-4 py-2.5">Channels</th>
                        <th scope="col" className="px-4 py-2.5">Triggered</th>
                        <th scope="col" className="px-4 py-2.5">Enabled</th>
                        <th scope="col" className="px-4 py-2.5"><span className="sr-only">Actions</span></th>
                      </tr>
                    </thead>
                    <tbody>
                      {rules.map((rule) => (
                        <tr key={rule.id} className="table-row">
                          <td className="px-4 py-3 font-semibold text-slate-200">{rule.name}</td>
                          <td className="px-4 py-3"><SeverityBadge severity={rule.severity} /></td>
                          <td className="px-4 py-3 text-slate-400">{rule.conditionText || "Custom condition"}</td>
                          <td className="px-4 py-3">
                            <div className="flex flex-wrap gap-1">
                              {rule.channelIds.map((id) => {
                                const channel = channels.find((item) => item.id === id);
                                return <Pill key={id} tone="sky">{channel?.name ?? id.slice(0, 6)}</Pill>;
                              })}
                              {rule.channelIds.length === 0 && <span className="text-[10px] text-slate-500">none</span>}
                            </div>
                          </td>
                          <td className="px-4 py-3 text-slate-400">
                            <div>{formatNumber(rule.triggerCount)}x</div>
                            <div className="text-[10px] text-slate-500">{rule.lastTriggeredAt ? timeAgo(rule.lastTriggeredAt) : "never"}</div>
                          </td>
                          <td className="px-4 py-3"><Toggle checked={rule.enabled} onChange={() => void toggleRule(rule)} label={`Enable ${rule.name}`} /></td>
                          <td className="px-4 py-3">
                            <button className="ml-auto flex rounded-lg p-1.5 text-slate-500 transition hover:bg-rose-950/40 hover:text-rose-300" onClick={() => void removeRule(rule)} disabled={busy === rule.id} aria-label={`Delete ${rule.name}`}>
                              <Icon name="trash" className="h-3.5 w-3.5" />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )
            )}

            {tab === "feed" && (
              sortedIncidents.length === 0 ? (
                <EmptyState icon="bell" title="No incidents yet" description="Incidents appear here when a rule matches a completed run." />
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[860px] text-xs" aria-label="Alert incidents">
                    <thead className="table-head">
                      <tr>
                        <th scope="col" className="px-4 py-2.5">Incident</th>
                        <th scope="col" className="px-4 py-2.5">Severity</th>
                        <th scope="col" className="px-4 py-2.5">Rule</th>
                        <th scope="col" className="px-4 py-2.5">Status</th>
                        <th scope="col" className="px-4 py-2.5">When</th>
                        <th scope="col" className="px-4 py-2.5"><span className="sr-only">Actions</span></th>
                      </tr>
                    </thead>
                    <tbody>
                      {sortedIncidents.map((incident) => (
                        <tr key={incident.id} className="table-row cursor-pointer" onClick={() => setSelectedIncident(incident)}>
                          <td className="px-4 py-3">
                            <div className="font-semibold text-slate-200">{incident.title}</div>
                            <div className="mt-0.5 max-w-[360px] truncate text-[11px] text-slate-500">{incident.message}</div>
                          </td>
                          <td className="px-4 py-3"><SeverityBadge severity={incident.severity} /></td>
                          <td className="px-4 py-3 text-slate-400">{incident.ruleName || "-"}</td>
                          <td className="px-4 py-3"><Pill tone={STATUS_TONES[incident.status]}>{incident.status}</Pill></td>
                          <td className="px-4 py-3 text-[11px] text-slate-500">{formatDate(incident.createdAt)}</td>
                          <td className="px-4 py-3">
                            <div className="flex justify-end gap-1" onClick={(event) => event.stopPropagation()}>
                              {incident.status === "open" && (
                                <button className="btn" onClick={() => void setIncidentStatus(incident, "acknowledged")} disabled={busy === incident.id}>Ack</button>
                              )}
                              {incident.status !== "resolved" && (
                                <button className="btn" onClick={() => void setIncidentStatus(incident, "resolved")} disabled={busy === incident.id}>Resolve</button>
                              )}
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )
            )}

            {tab === "channels" && (
              channels.length === 0 ? (
                <EmptyState icon="bell" title="No channels configured" description="Add a Slack webhook, generic webhook, PagerDuty routing key or email target." action={<button className="btn-primary" onClick={() => setChannelOpen(true)}><Icon name="plus" className="h-3.5 w-3.5" />New channel</button>} />
              ) : (
                <div className="grid gap-3 p-4 md:grid-cols-2 xl:grid-cols-3">
                  {channels.map((channel) => (
                    <div key={channel.id} className="rounded-xl border border-slate-800 bg-slate-950/40 p-4">
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-800 bg-slate-900 text-indigo-300">
                            <Icon name={channelIcon(channel.kind)} className="h-4 w-4" />
                          </span>
                          <div>
                            <p className="text-xs font-semibold text-slate-200">{channel.name}</p>
                            <p className="text-[10px] uppercase tracking-wide text-slate-500">{channel.kind}</p>
                          </div>
                        </div>
                        <Pill tone={channel.connected ? "emerald" : "slate"}>{channel.connected ? "connected" : "untested"}</Pill>
                      </div>
                      <p className="mono mt-3 truncate text-[10px] text-slate-500">{channel.target || channel.secretRef || "-"}</p>
                      <p className="mt-1 text-[10px] text-slate-500">{channel.lastDeliveryAt ? `Last delivery ${timeAgo(channel.lastDeliveryAt)}` : "No deliveries yet"}</p>
                      <div className="mt-3 flex justify-end gap-2">
                        <button className="btn" onClick={() => void testChannel(channel)} disabled={busy === channel.id}>Test</button>
                        <button className="rounded-lg p-1.5 text-slate-500 transition hover:bg-rose-950/40 hover:text-rose-300" onClick={() => void removeChannel(channel)} disabled={busy === channel.id} aria-label={`Delete ${channel.name}`}>
                          <Icon name="trash" className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )
            )}
          </Card>
        </>
      )}

      <Modal open={ruleOpen} onClose={() => setRuleOpen(false)} title="New alert rule" subtitle="Evaluate findings and run state after every audit.">
        <form onSubmit={submitRule} className="space-y-4">
          <div>
            <label className="label" htmlFor="rule-name">Rule name</label>
            <input id="rule-name" className="input" value={ruleName} onChange={(event) => setRuleName(event.target.value)} placeholder="Critical consent leaks" required />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="rule-severity">Incident severity</label>
              <select id="rule-severity" className="input" value={ruleSeverity} onChange={(event) => setRuleSeverity(event.target.value as Severity)}>
                {SEVERITIES.map((severity) => <option key={severity} value={severity}>{severity}</option>)}
              </select>
            </div>
            <div>
              <label className="label" htmlFor="rule-condition">Condition</label>
              <select id="rule-condition" className="input" value={conditionKind} onChange={(event) => setConditionKind(event.target.value as typeof conditionKind)}>
                <option value="severity">Minimum finding severity</option>
                <option value="failure">Any run failure</option>
                <option value="tags">Missing tags</option>
              </select>
            </div>
          </div>
          {conditionKind === "severity" && (
            <div>
              <label className="label" htmlFor="condition-severity">Minimum severity</label>
              <select id="condition-severity" className="input" value={conditionSeverity} onChange={(event) => setConditionSeverity(event.target.value as Severity)}>
                {SEVERITIES.map((severity) => <option key={severity} value={severity}>{severity}</option>)}
              </select>
            </div>
          )}
          {conditionKind === "tags" && (
            <div>
              <label className="label" htmlFor="condition-tags">Required tags (comma separated)</label>
              <input id="condition-tags" className="input" value={tagList} onChange={(event) => setTagList(event.target.value)} placeholder="Google Analytics, Meta Pixel" />
            </div>
          )}
          <div>
            <p className="label">Channels</p>
            <div className="flex flex-wrap gap-2">
              {channels.map((channel) => {
                const checked = selectedChannels.includes(channel.id);
                return (
                  <button
                    key={channel.id}
                    type="button"
                    className={`rounded-lg border px-2.5 py-1.5 text-[11px] transition ${checked ? "border-indigo-700 bg-indigo-950/50 text-indigo-200" : "border-slate-800 bg-slate-900 text-slate-400"}`}
                    onClick={() => setSelectedChannels((current) => checked ? current.filter((id) => id !== channel.id) : [...current, channel.id])}
                  >
                    {channel.name}
                  </button>
                );
              })}
              {channels.length === 0 && <span className="text-[11px] text-slate-500">No channels configured yet. Add one on the Channels tab.</span>}
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setRuleOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={!ruleName.trim() || busy === "rule"}>Create rule</button>
          </div>
        </form>
      </Modal>

      <Modal open={channelOpen} onClose={() => setChannelOpen(false)} title="New notification channel" subtitle="Secrets are referenced by environment variable name and never returned by the API.">
        <form onSubmit={submitChannel} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="channel-name">Name</label>
              <input id="channel-name" className="input" value={channelName} onChange={(event) => setChannelName(event.target.value)} placeholder="Privacy on-call" required />
            </div>
            <div>
              <label className="label" htmlFor="channel-kind">Kind</label>
              <select id="channel-kind" className="input" value={channelKind} onChange={(event) => setChannelKind(event.target.value as AlertChannelKind)}>
                {CHANNEL_KINDS.map((kind) => <option key={kind.id} value={kind.id}>{kind.label}</option>)}
              </select>
            </div>
          </div>
          <div>
            <label className="label" htmlFor="channel-target">Target URL or address</label>
            <input id="channel-target" className="input" value={channelTarget} onChange={(event) => setChannelTarget(event.target.value)} placeholder={channelKind === "pagerduty" ? "PagerDuty routing key" : "https://hooks.slack.com/services/..."} />
          </div>
          <div>
            <label className="label" htmlFor="channel-secret">Secret env var (optional)</label>
            <input id="channel-secret" className="input mono" value={channelSecret} onChange={(event) => setChannelSecret(event.target.value)} placeholder="PAGEGUARD_SLACK_WEBHOOK" />
            <p className="mt-1 text-[10px] text-slate-500">Use when the destination secret should not be stored in the database.</p>
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setChannelOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={!channelName.trim() || busy === "channel"}>Create channel</button>
          </div>
        </form>
      </Modal>

      <Modal open={!!selectedIncident} onClose={() => setSelectedIncident(null)} title={selectedIncident?.title ?? "Incident"} subtitle={selectedIncident ? `${selectedIncident.ruleName || "manual"} / ${formatDate(selectedIncident.createdAt)}` : undefined}>
        {selectedIncident && (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center gap-2">
              <SeverityBadge severity={selectedIncident.severity} />
              <Pill tone={STATUS_TONES[selectedIncident.status]}>{selectedIncident.status}</Pill>
              {selectedIncident.runId && <span className="mono text-[10px] text-slate-500">run {selectedIncident.runId.slice(0, 8)}</span>}
            </div>
            <p className="rounded-xl border border-slate-800 bg-slate-950/40 p-3 text-xs text-slate-300">{selectedIncident.message}</p>
            <div>
              <h3 className="text-xs font-semibold text-white">Delivery attempts</h3>
              {deliveries.length === 0 ? (
                <p className="mt-2 text-[11px] text-slate-500">No delivery records for this incident.</p>
              ) : (
                <ul className="mt-2 space-y-1.5">
                  {deliveries.map((delivery) => (
                    <li key={delivery.id} className="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2 text-[11px]">
                      <span className="text-slate-300">{delivery.status} / attempt {delivery.attempt}</span>
                      <span className="text-slate-500">{formatDate(delivery.attemptedAt)}</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
            <div className="flex justify-end gap-2">
              {selectedIncident.status === "open" && <button className="btn" onClick={() => void setIncidentStatus(selectedIncident, "acknowledged")} disabled={busy === selectedIncident.id}>Acknowledge</button>}
              {selectedIncident.status !== "resolved" && <button className="btn-primary" onClick={() => void setIncidentStatus(selectedIncident, "resolved")} disabled={busy === selectedIncident.id}>Resolve</button>}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
