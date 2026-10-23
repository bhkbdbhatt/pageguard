"use client";

import { useMemo, useState } from "react";
import { DemoBanner, DemoPageHeader } from "../../components/demo";
import { Icon, type IconName } from "../../components/icons";
import {
  Card,
  KpiCard,
  Modal,
  Pill,
  SeverityBadge,
  Tabs,
  Toggle,
} from "../../components/ui";
import type { Severity } from "../../lib/api";
import {
  DEMO_ALERT_RULES,
  DEMO_CHANNELS,
  DEMO_INCIDENTS,
  type DemoAlertRule,
  type DemoIncident,
  type IncidentStatus,
} from "../../lib/demo";
import { formatNumber } from "../../lib/format";

type Tab = "rules" | "feed" | "channels";

const SEVERITIES: Severity[] = ["info", "low", "medium", "high", "critical"];
const CHANNELS = [
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

function channelIcon(channel: string): IconName {
  const icons: Record<string, IconName> = {
    slack: "message",
    email: "mail",
    webhook: "webhook",
    pagerduty: "bell",
  };
  return icons[channel] ?? "bell";
}

export default function AlertsPage() {
  const [tab, setTab] = useState<Tab>("rules");
  const [rules, setRules] = useState<DemoAlertRule[]>(DEMO_ALERT_RULES);
  const [incidents, setIncidents] = useState<DemoIncident[]>(DEMO_INCIDENTS);
  const [createOpen, setCreateOpen] = useState(false);
  const [tested, setTested] = useState<string[]>([]);

  const [name, setName] = useState("");
  const [severity, setSeverity] = useState<Severity>("high");
  const [condition, setCondition] = useState("");
  const [selectedChannels, setSelectedChannels] = useState<string[]>(["slack"]);

  const openIncidents = incidents.filter((incident) => incident.status === "open").length;
  const alertsToday = incidents.filter((incident) => incident.atLabel.startsWith("Today")).length;
  const connectedChannels = DEMO_CHANNELS.filter((channel) => channel.connected).length;
  const activeRules = rules.filter((rule) => rule.enabled).length;

  const sortedIncidents = useMemo(
    () => [...incidents].sort((a, b) => (a.at < b.at ? 1 : -1)),
    [incidents],
  );

  function toggleRule(id: string) {
    setRules((current) =>
      current.map((rule) => (rule.id === id ? { ...rule, enabled: !rule.enabled } : rule)),
    );
  }

  function setIncidentStatus(id: string, status: IncidentStatus) {
    setIncidents((current) =>
      current.map((incident) => (incident.id === id ? { ...incident, status } : incident)),
    );
  }

  function createRule(event: React.FormEvent) {
    event.preventDefault();
    const next: DemoAlertRule = {
      id: `rule-${Date.now()}`,
      name: name.trim() || "Untitled rule",
      severity,
      condition: condition.trim() || "Custom condition",
      channels: selectedChannels,
      enabled: true,
      lastTriggered: "Never",
      triggerCount: 0,
    };
    setRules((current) => [next, ...current]);
    setCreateOpen(false);
    setName("");
    setCondition("");
    setSeverity("high");
    setSelectedChannels(["slack"]);
  }

  return (
    <div className="space-y-5">
      <DemoPageHeader
        eyebrow="Monitoring"
        title="Alerts & Notifications"
        subtitle="Route governance findings to the right people with severity-aware rules and delivery channels."
        icon="bell"
        actions={
          <>
            <button className="btn" title="Demo only">
              <Icon name="settings" className="h-3.5 w-3.5" />
              Notification settings
            </button>
            <button className="btn-primary" onClick={() => setCreateOpen(true)}>
              <Icon name="plus" className="h-3.5 w-3.5" />
              Create rule
            </button>
          </>
        }
      />

      <DemoBanner />

      <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        <KpiCard label="Open incidents" value={formatNumber(openIncidents)} hint={`${incidents.length} total events`} tone="rose" icon="alert" />
        <KpiCard label="Alerts today" value={formatNumber(alertsToday)} hint={`${activeRules} active rules`} tone="amber" icon="bell" />
        <KpiCard label="Mean time to ack" value="18m" hint="last 7 days" tone="sky" icon="clock" />
        <KpiCard label="Connected channels" value={`${connectedChannels}/${DEMO_CHANNELS.length}`} hint="Slack, email, webhook" tone="emerald" icon="webhook" />
      </div>

      <Card>
        <div className="border-b border-slate-800 px-4 py-3">
          <Tabs
            value={tab}
            onChange={(id) => setTab(id as Tab)}
            tabs={[
              { id: "rules", label: "Alert rules", count: rules.length },
              { id: "feed", label: "Incident feed", count: openIncidents },
              { id: "channels", label: "Channels", count: DEMO_CHANNELS.length },
            ]}
          />
        </div>

        {tab === "rules" && (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[860px] text-xs" aria-label="Alert rules">
              <thead className="table-head">
                <tr>
                  <th scope="col" className="px-4 py-2.5">Rule</th>
                  <th scope="col" className="px-4 py-2.5">Condition</th>
                  <th scope="col" className="px-4 py-2.5">Severity</th>
                  <th scope="col" className="px-4 py-2.5">Channels</th>
                  <th scope="col" className="px-4 py-2.5">Last triggered</th>
                  <th scope="col" className="px-4 py-2.5">Enabled</th>
                  <th scope="col" className="px-4 py-2.5"><span className="sr-only">Actions</span></th>
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr key={rule.id} className="table-row">
                    <td className="px-4 py-3">
                      <div className="font-semibold text-slate-200">{rule.name}</div>
                      <div className="mono mt-0.5 text-[10px] text-slate-500">{rule.id}</div>
                    </td>
                    <td className="max-w-[280px] px-4 py-3 text-slate-400">{rule.condition}</td>
                    <td className="px-4 py-3"><SeverityBadge severity={rule.severity} /></td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {rule.channels.map((channel) => (
                          <span
                            key={channel}
                            className="inline-flex items-center gap-1 rounded border border-slate-700 bg-slate-800/60 px-1.5 py-0.5 text-[10px] text-slate-400"
                          >
                            <Icon name={channelIcon(channel)} className="h-3 w-3" />
                            {channel}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="text-slate-300">{rule.lastTriggered}</div>
                      <div className="text-[10px] text-slate-500">{formatNumber(rule.triggerCount)} triggers</div>
                    </td>
                    <td className="px-4 py-3">
                      <Toggle checked={rule.enabled} onChange={() => toggleRule(rule.id)} label={`Enable ${rule.name}`} />
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        className="rounded-lg p-1.5 text-slate-500 transition hover:bg-rose-950/40 hover:text-rose-300"
                        onClick={() => setRules((current) => current.filter((item) => item.id !== rule.id))}
                        aria-label={`Delete ${rule.name}`}
                        title="Delete rule"
                      >
                        <Icon name="trash" className="h-3.5 w-3.5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {tab === "feed" && (
          <ul className="divide-y divide-slate-800">
            {sortedIncidents.map((incident) => (
              <li key={incident.id} className="flex flex-col gap-3 px-4 py-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="flex min-w-0 items-start gap-3">
                  <span
                    className={`mt-1 h-2 w-2 shrink-0 rounded-full ${
                      incident.status === "open"
                        ? "bg-rose-400"
                        : incident.status === "acknowledged"
                          ? "bg-amber-400"
                          : "bg-emerald-400"
                    }`}
                  />
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <SeverityBadge severity={incident.severity} />
                      <span className="text-sm font-semibold text-slate-100">{incident.title}</span>
                      <Pill tone={STATUS_TONES[incident.status]}>{incident.status}</Pill>
                    </div>
                    <p className="mt-1 max-w-2xl text-xs leading-relaxed text-slate-400">{incident.message}</p>
                    <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px] text-slate-500">
                      <span className="inline-flex items-center gap-1"><Icon name="bell" className="h-3 w-3" />{incident.ruleName}</span>
                      <span className="inline-flex items-center gap-1"><Icon name={channelIcon(incident.channel.split(" ")[0].toLowerCase())} className="h-3 w-3" />{incident.channel}</span>
                      <span className="inline-flex items-center gap-1"><Icon name="clock" className="h-3 w-3" />{incident.atLabel}</span>
                      <span className="mono inline-flex items-center gap-1"><Icon name="radar" className="h-3 w-3" />{incident.runId.slice(0, 12)}</span>
                    </div>
                  </div>
                </div>
                <div className="flex shrink-0 gap-2 pl-5 lg:pl-0">
                  {incident.status === "open" && (
                    <button className="btn" onClick={() => setIncidentStatus(incident.id, "acknowledged")}>
                      <Icon name="check" className="h-3.5 w-3.5" />
                      Acknowledge
                    </button>
                  )}
                  {incident.status !== "resolved" && (
                    <button className="btn-primary" onClick={() => setIncidentStatus(incident.id, "resolved")}>
                      <Icon name="check-circle" className="h-3.5 w-3.5" />
                      Resolve
                    </button>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}

        {tab === "channels" && (
          <div className="grid gap-3 p-4 sm:grid-cols-2">
            {DEMO_CHANNELS.map((channel) => (
              <div key={channel.id} className="rounded-xl border border-slate-800 bg-slate-950/40 p-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-2.5">
                    <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-slate-800 bg-slate-900 text-indigo-300">
                      <Icon name={channelIcon(channel.kind)} className="h-4 w-4" />
                    </span>
                    <div>
                      <p className="text-sm font-semibold text-slate-100">{channel.name}</p>
                      <p className="text-[11px] text-slate-500">{channel.target}</p>
                    </div>
                  </div>
                  <Pill tone={channel.connected ? "emerald" : "slate"}>{channel.connected ? "connected" : "offline"}</Pill>
                </div>
                <div className="mt-3 flex items-center justify-between gap-3 border-t border-slate-800 pt-3">
                  <span className="text-[10px] text-slate-500">Last delivery · {channel.lastDelivery}</span>
                  <button
                    className="btn"
                    disabled={!channel.connected}
                    onClick={() => setTested((current) => (current.includes(channel.id) ? current : [...current, channel.id]))}
                  >
                    {tested.includes(channel.id) ? (
                      <>
                        <Icon name="check" className="h-3 w-3" />
                        Sent
                      </>
                    ) : (
                      <>
                        <Icon name="play" className="h-3 w-3" />
                        Send test
                      </>
                    )}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Create alert rule"
        subtitle="Trigger a notification when findings match this condition."
      >
        <form onSubmit={createRule} className="space-y-4">
          <div>
            <label className="label" htmlFor="alert-name">Rule name</label>
            <input
              id="alert-name"
              className="input"
              placeholder="Meta pixel before consent"
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="alert-severity">Minimum severity</label>
              <select
                id="alert-severity"
                className="input"
                value={severity}
                onChange={(event) => setSeverity(event.target.value as Severity)}
              >
                {SEVERITIES.map((item) => (
                  <option key={item} value={item}>{item}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="label" htmlFor="alert-condition">Condition</label>
              <input
                id="alert-condition"
                className="input"
                placeholder="rule = consent.pre-consent-marketing-tag"
                value={condition}
                onChange={(event) => setCondition(event.target.value)}
              />
            </div>
          </div>
          <div>
            <p className="label">Delivery channels</p>
            <div className="grid grid-cols-2 gap-2">
              {CHANNELS.map((channel) => {
                const active = selectedChannels.includes(channel.id);
                return (
                  <button
                    key={channel.id}
                    type="button"
                    aria-pressed={active}
                    onClick={() =>
                      setSelectedChannels((current) =>
                        current.includes(channel.id)
                          ? current.filter((item) => item !== channel.id)
                          : [...current, channel.id],
                      )
                    }
                    className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-xs font-medium transition ${
                      active
                        ? "border-indigo-700/60 bg-indigo-950/40 text-indigo-200"
                        : "border-slate-700 bg-slate-800/40 text-slate-400 hover:border-slate-600"
                    }`}
                  >
                    <Icon name={channelIcon(channel.id)} className="h-3.5 w-3.5" />
                    {channel.label}
                    {active && <Icon name="check" className="ml-auto h-3.5 w-3.5" />}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="flex justify-end gap-2 pt-1">
            <button type="button" className="btn" onClick={() => setCreateOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={!name.trim()}>
              <Icon name="plus" className="h-3.5 w-3.5" />
              Create rule
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
