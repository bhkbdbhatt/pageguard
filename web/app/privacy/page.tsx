"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { Heatmap, RadialGauge } from "../../components/charts";
import { DemoBanner, DemoPageHeader } from "../../components/demo";
import { Icon } from "../../components/icons";
import {
  Card,
  CardHeader,
  EmptyState,
  KpiCard,
  Pill,
  SeverityBadge,
  Tabs,
} from "../../components/ui";
import {
  DEMO_CONSENT_STATES,
  DEMO_PRIVACY_CHECKS,
  DEMO_PRIVACY_GAPS,
  DEMO_PRIVACY_PROFILES,
  DEMO_REGULATIONS,
  type RegulationId,
} from "../../lib/demo";
import { formatNumber, formatPercent } from "../../lib/format";

type Tab = "matrix" | "gaps";
type ComplianceStatus = "pass" | "warn" | "fail";

function statusTone(status: ComplianceStatus) {
  if (status === "pass") return "emerald" as const;
  if (status === "warn") return "amber" as const;
  return "rose" as const;
}

function regulationName(id: RegulationId) {
  return DEMO_REGULATIONS.find((regulation) => regulation.id === id)?.name ?? id.toUpperCase();
}

export default function PrivacyPage() {
  const [profileId, setProfileId] = useState(DEMO_PRIVACY_PROFILES[0].id);
  const [tab, setTab] = useState<Tab>("matrix");

  const profile = DEMO_PRIVACY_PROFILES.find((item) => item.id === profileId) ?? DEMO_PRIVACY_PROFILES[0];

  const regulations = useMemo(
    () => DEMO_REGULATIONS.filter((regulation) => profile.regulationIds.includes(regulation.id)),
    [profile],
  );

  const gaps = useMemo(
    () => DEMO_PRIVACY_GAPS.filter((gap) => profile.regulationIds.includes(gap.regulation)),
    [profile],
  );

  const complianceScore =
    regulations.length > 0
      ? regulations.reduce((sum, regulation) => sum + regulation.score, 0) / regulations.length
      : 0;

  const matrixRows = DEMO_PRIVACY_CHECKS.map((check) => check.label);
  const matrixColumns = regulations.map((regulation) => regulation.name);
  const matrixCells = DEMO_PRIVACY_CHECKS.map((check) =>
    regulations.map((regulation) => check.results[regulation.id]),
  );

  const activeProfiles = DEMO_PRIVACY_PROFILES.filter((item) => item.active).length;

  return (
    <div className="space-y-5">
      <DemoPageHeader
        eyebrow="Jurisdiction compliance"
        title="Privacy Jurisdiction Profiles"
        subtitle="Evaluate evidence against GDPR, CCPA/CPRA, LGPD and ePrivacy profiles. Complements the per-run CMP matrix."
        icon="scale"
        actions={
          <>
            <Link href="/consent" className="btn">
              <Icon name="cookie" className="h-3.5 w-3.5" />
              Open CMP matrix
            </Link>
            <button className="btn-primary" title="Demo only">
              <Icon name="download" className="h-3.5 w-3.5" />
              Export evidence
            </button>
          </>
        }
      />

      <DemoBanner />

      <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        <KpiCard label="Active profiles" value={`${activeProfiles}/${DEMO_PRIVACY_PROFILES.length}`} hint="scoped to operating regions" tone="indigo" icon="scale" />
        <KpiCard label="Applicable regulations" value={formatNumber(profile.regulationIds.length)} hint={profile.region} tone="sky" icon="globe" />
        <KpiCard label="Compliance score" value={formatPercent(complianceScore, 0)} hint="average across active regulations" tone={complianceScore >= 85 ? "emerald" : complianceScore >= 70 ? "amber" : "rose"} icon="shield" />
        <KpiCard label="Open gaps" value={formatNumber(gaps.length)} hint="remediation required" tone={gaps.length > 0 ? "rose" : "emerald"} icon="alert" />
      </div>

      <Card>
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 px-4 py-3">
          <div>
            <h2 className="text-sm font-semibold text-white">Jurisdiction profile</h2>
            <p className="mt-0.5 text-xs text-slate-500">Select a profile to scope scores, checks and remediation gaps.</p>
          </div>
          <Tabs
            value={profileId}
            onChange={setProfileId}
            tabs={DEMO_PRIVACY_PROFILES.map((item) => ({ id: item.id, label: item.name }))}
          />
        </div>
        <div className="grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-4">
          {DEMO_REGULATIONS.map((regulation) => {
            const included = profile.regulationIds.includes(regulation.id);
            return (
              <div
                key={regulation.id}
                className={`rounded-xl border p-4 transition ${
                  included
                    ? "border-indigo-800/50 bg-indigo-950/20"
                    : "border-slate-800 bg-slate-950/40 opacity-55"
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="text-sm font-bold text-white">{regulation.name}</p>
                    <p className="mt-0.5 text-[10px] leading-relaxed text-slate-500">{regulation.fullName}</p>
                  </div>
                  <Pill tone={statusTone(regulation.status)}>{regulation.status}</Pill>
                </div>
                <div className="mt-4 flex items-center gap-4">
                  <RadialGauge value={regulation.score} tone={statusTone(regulation.status)} size={74} />
                  <div className="space-y-1.5 text-[11px] text-slate-500">
                    <div className="flex items-center justify-between gap-3">
                      <span>Checks</span>
                      <span className="font-mono text-slate-300">{regulation.checks}</span>
                    </div>
                    <div className="flex items-center justify-between gap-3">
                      <span>Failures</span>
                      <span className="font-mono text-rose-300">{regulation.failures}</span>
                    </div>
                    <div className="flex items-center justify-between gap-3">
                      <span>Evaluated</span>
                      <span className="text-slate-400">{regulation.lastEvaluated}</span>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </Card>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {DEMO_CONSENT_STATES.map((state) => (
          <div key={state.id} className="card-flat p-4">
            <div className="flex items-center justify-between gap-2">
              <p className="text-xs font-semibold text-slate-200">{state.label}</p>
              <Pill tone={state.status === "pass" ? "emerald" : "amber"}>{state.status}</Pill>
            </div>
            <div className="mt-3 grid grid-cols-3 gap-2 text-center">
              <div>
                <div className="text-lg font-extrabold text-white">{state.trackers}</div>
                <div className="text-[9px] uppercase tracking-wide text-slate-500">Trackers</div>
              </div>
              <div>
                <div className="text-lg font-extrabold text-white">{state.blocked}</div>
                <div className="text-[9px] uppercase tracking-wide text-slate-500">Blocked</div>
              </div>
              <div>
                <div className={`text-lg font-extrabold ${state.leaks > 0 ? "text-rose-300" : "text-emerald-300"}`}>{state.leaks}</div>
                <div className="text-[9px] uppercase tracking-wide text-slate-500">Leaks</div>
              </div>
            </div>
          </div>
        ))}
      </div>

      <Card>
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 px-4 py-3">
          <Tabs
            value={tab}
            onChange={(id) => setTab(id as Tab)}
            tabs={[
              { id: "matrix", label: "Compliance matrix", count: DEMO_PRIVACY_CHECKS.length },
              { id: "gaps", label: "Remediation gaps", count: gaps.length },
            ]}
          />
          <span className="text-[10px] text-slate-500">
            Scope · <span className="text-slate-300">{profile.name}</span> ({profile.region})
          </span>
        </div>

        {tab === "matrix" && (
          <div className="p-4">
            <Heatmap rows={matrixRows} columns={matrixColumns} cells={matrixCells} />
            <div className="mt-4 flex flex-wrap items-center gap-4 text-[10px] text-slate-500">
              <span className="inline-flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-emerald-400" /> Pass</span>
              <span className="inline-flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-rose-400" /> Fail</span>
              <span className="inline-flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-slate-600" /> Not applicable</span>
            </div>
          </div>
        )}

        {tab === "gaps" && (
          gaps.length === 0 ? (
            <EmptyState
              icon="check-circle"
              title="No open gaps"
              description="This profile has no failing checks in the latest evidence set."
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[860px] text-xs" aria-label="Remediation gaps">
                <thead className="table-head">
                  <tr>
                    <th scope="col" className="px-4 py-2.5">Regulation</th>
                    <th scope="col" className="px-4 py-2.5">Check</th>
                    <th scope="col" className="px-4 py-2.5">Severity</th>
                    <th scope="col" className="px-4 py-2.5">Evidence</th>
                    <th scope="col" className="px-4 py-2.5">Remediation</th>
                  </tr>
                </thead>
                <tbody>
                  {gaps.map((gap) => (
                    <tr key={gap.id} className="table-row">
                      <td className="px-4 py-3">
                        <Pill tone="indigo">{regulationName(gap.regulation)}</Pill>
                      </td>
                      <td className="px-4 py-3 font-semibold text-slate-200">{gap.check}</td>
                      <td className="px-4 py-3"><SeverityBadge severity={gap.severity} /></td>
                      <td className="max-w-[280px] px-4 py-3 text-slate-400">{gap.detail}</td>
                      <td className="max-w-[280px] px-4 py-3 text-slate-400">{gap.remediation}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        )}
      </Card>

      <Card>
        <CardHeader
          title="How profiles scope compliance"
          subtitle="Profiles select which regulation checks apply to a region; evidence is derived from captured run checkpoints."
          right={<Pill tone="slate">Design preview</Pill>}
        />
        <div className="grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-4">
          {DEMO_PRIVACY_PROFILES.map((item) => (
            <div key={item.id} className="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs font-semibold text-slate-200">{item.name}</span>
                <Pill tone={item.active ? "emerald" : "slate"}>{item.active ? "active" : "paused"}</Pill>
              </div>
              <p className="mt-1 text-[10px] text-slate-500">{item.region}</p>
              <div className="mt-2 flex flex-wrap gap-1">
                {item.regulationIds.map((id) => (
                  <span key={id} className="rounded border border-slate-700 bg-slate-800/60 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-slate-400">
                    {regulationName(id)}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
}
