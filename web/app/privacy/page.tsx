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
  ProgressBar,
  Select,
  StatusBadge,
  Toggle,
} from "../../components/ui";
import {
  deletePrivacyProfile,
  listPrivacyBatches,
  listPrivacyEvaluations,
  listPrivacyProfiles,
  startPrivacyBatch,
  updatePrivacyProfile,
  createPrivacyProfile,
  type PrivacyBatch,
  type PrivacyEvaluation,
  type PrivacyProfile,
} from "../../lib/api";
import { formatDate, formatNumber, formatPercent, timeAgo } from "../../lib/format";

const REGULATIONS = [
  { id: "gdpr", label: "GDPR" },
  { id: "ccpa", label: "CCPA" },
  { id: "lgpd", label: "LGPD" },
  { id: "eprivacy", label: "ePrivacy" },
];

const REGION_OPTIONS = ["EU", "US", "BR", "UK", "APAC"];

export default function PrivacyPage() {
  const { project, loadingProjects } = useApp();
  const [profiles, setProfiles] = useState<PrivacyProfile[]>([]);
  const [batches, setBatches] = useState<PrivacyBatch[]>([]);
  const [evaluations, setEvaluations] = useState<PrivacyEvaluation[]>([]);
  const [selectedBatchId, setSelectedBatchId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const [profileOpen, setProfileOpen] = useState(false);
  const [profileName, setProfileName] = useState("");
  const [profileRegion, setProfileRegion] = useState("EU");
  const [profileRegs, setProfileRegs] = useState<string[]>(["gdpr"]);

  const [batchOpen, setBatchOpen] = useState(false);
  const [batchProfileId, setBatchProfileId] = useState("");
  const [batchUrl, setBatchUrl] = useState("");

  const load = useCallback(async () => {
    if (!project) {
      setProfiles([]);
      setBatches([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const [nextProfiles, nextBatches] = await Promise.all([
        listPrivacyProfiles(project.id),
        listPrivacyBatches(project.id),
      ]);
      setProfiles(nextProfiles);
      setBatches(nextBatches);
      setSelectedBatchId((current) => (current && nextBatches.some((batch) => batch.id === current) ? current : nextBatches[0]?.id ?? null));
      if (!batchProfileId) setBatchProfileId(nextProfiles[0]?.id ?? "");
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [batchProfileId, project]);

  useEffect(() => {
    void load();
  }, [load]);

  const selectedBatch = batches.find((batch) => batch.id === selectedBatchId) ?? null;

  useEffect(() => {
    if (!selectedBatchId) {
      setEvaluations([]);
      return;
    }
    let active = true;
    const loadEvaluations = () =>
      listPrivacyEvaluations(selectedBatchId)
        .then((rows) => active && setEvaluations(rows))
        .catch((err) => active && setError((err as Error).message));
    void loadEvaluations();
    const id = window.setInterval(loadEvaluations, 4000);
    return () => {
      active = false;
      window.clearInterval(id);
    };
  }, [selectedBatchId]);

  useEffect(() => {
    if (!project) return;
    const id = window.setInterval(() => {
      void listPrivacyBatches(project.id).then(setBatches).catch(() => undefined);
    }, 8000);
    return () => window.clearInterval(id);
  }, [project]);

  const reports = useMemo(() => evaluations.map((evaluation) => evaluation.report).filter(Boolean), [evaluations]);
  const averageScore = reports.length > 0
    ? reports.reduce((sum, report) => sum + (report?.score ?? 0), 0) / reports.length
    : 0;
  const passedCount = reports.filter((report) => report?.passed).length;
  const violations = reports.reduce((sum, report) => sum + (report?.summary.violations ?? 0), 0);
  const running = batches.filter((batch) => batch.status === "running" || batch.status === "queued").length;

  async function toggleProfile(profile: PrivacyProfile) {
    setBusy(profile.id);
    try {
      const updated = await updatePrivacyProfile(profile.id, { active: !profile.active });
      setProfiles((current) => current.map((item) => (item.id === profile.id ? updated : item)));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function removeProfile(profile: PrivacyProfile) {
    if (!window.confirm(`Delete profile "${profile.name}"?`)) return;
    setBusy(profile.id);
    try {
      await deletePrivacyProfile(profile.id);
      setProfiles((current) => current.filter((item) => item.id !== profile.id));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function submitProfile(event: React.FormEvent) {
    event.preventDefault();
    if (!project) return;
    setBusy("profile");
    try {
      const created = await createPrivacyProfile({
        projectId: project.id,
        name: profileName.trim(),
        region: profileRegion,
        regulationIds: profileRegs,
      });
      setProfiles((current) => [...current, created]);
      setProfileOpen(false);
      setProfileName("");
      setProfileRegion("EU");
      setProfileRegs(["gdpr"]);
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(null);
    }
  }

  async function submitBatch(event: React.FormEvent) {
    event.preventDefault();
    if (!project) return;
    setBusy("batch");
    try {
      const batch = await startPrivacyBatch({
        projectId: project.id,
        profileId: batchProfileId,
        startUrl: batchUrl.trim(),
      });
      setBatches((current) => [batch, ...current]);
      setSelectedBatchId(batch.id);
      setBatchOpen(false);
      setBatchUrl("");
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
        icon="scale"
        title="No project selected"
        description="Create a project to evaluate pages against privacy regulations."
      />
    );
  }

  return (
    <div className="space-y-5">
      <PageHeader
        eyebrow="Compliance"
        title="Privacy Jurisdictions"
        subtitle="Re-scan pages under consent states and score them against GDPR, CCPA, LGPD and ePrivacy rules."
        icon="scale"
        actions={
          <>
            <button className="btn" onClick={() => setProfileOpen(true)} disabled={!project}>
              <Icon name="plus" className="h-3.5 w-3.5" />
              New profile
            </button>
            <button className="btn-primary" onClick={() => setBatchOpen(true)} disabled={!project || profiles.length === 0}>
              <Icon name="play" className="h-3.5 w-3.5" />
              New evaluation
            </button>
          </>
        }
      />

      <ErrorBanner message={error} />

      {loading ? (
        <Loading label="Loading privacy profiles" />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
            <KpiCard label="Profiles" value={formatNumber(profiles.length)} hint={`${profiles.filter((profile) => profile.active).length} active`} tone="indigo" icon="shield" />
            <KpiCard label="Average score" value={formatPercent(averageScore, 0)} hint={`${formatNumber(reports.length)} evaluations`} tone={averageScore >= 80 ? "emerald" : "amber"} icon="activity" />
            <KpiCard label="Regulations passed" value={`${passedCount}/${reports.length}`} hint="latest batch results" tone="emerald" icon="check" />
            <KpiCard label="Violations" value={formatNumber(violations)} hint={`${running} batch(es) running`} tone={violations > 0 ? "rose" : "slate"} icon="alert" />
          </div>

          <div className="grid gap-4 xl:grid-cols-5">
            <div className="space-y-4 xl:col-span-2">
              <Card>
                <CardHeader title="Jurisdiction profiles" subtitle="Policy sets applied to evaluations" right={<Pill tone="slate">{formatNumber(profiles.length)}</Pill>} />
                {profiles.length === 0 ? (
                  <EmptyState icon="shield" title="No profiles" description="Create a profile to select which regulations apply." />
                ) : (
                  <ul className="divide-y divide-slate-800">
                    {profiles.map((profile) => (
                      <li key={profile.id} className="flex items-start justify-between gap-3 px-4 py-3">
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="truncate text-xs font-semibold text-slate-200">{profile.name}</span>
                            <Pill tone="sky">{profile.region || "global"}</Pill>
                          </div>
                          <div className="mt-1 flex flex-wrap gap-1">
                            {profile.regulationIds.map((id) => <span key={id} className="rounded border border-slate-700 bg-slate-800/60 px-1 py-0.5 text-[9px] uppercase tracking-wide text-slate-400">{id}</span>)}
                          </div>
                        </div>
                        <div className="flex shrink-0 items-center gap-2">
                          <Toggle checked={profile.active} onChange={() => void toggleProfile(profile)} label={`Enable ${profile.name}`} />
                          <button className="text-slate-500 transition hover:text-rose-300" onClick={() => void removeProfile(profile)} disabled={busy === profile.id} aria-label={`Delete ${profile.name}`}>
                            <Icon name="trash" className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </li>
                    ))}
                  </ul>
                )}
              </Card>

              <Card>
                <CardHeader title="Evaluation batches" subtitle="Queued and completed page scans" right={<Pill tone="slate">{formatNumber(batches.length)}</Pill>} />
                {batches.length === 0 ? (
                  <EmptyState icon="scale" title="No evaluations" description="Start an evaluation to score a page under your profile." />
                ) : (
                  <ul className="divide-y divide-slate-800">
                    {batches.map((batch) => (
                      <li key={batch.id}>
                        <button className={`w-full px-4 py-3 text-left transition ${batch.id === selectedBatchId ? "bg-indigo-950/30" : "hover:bg-slate-900/60"}`} onClick={() => setSelectedBatchId(batch.id)}>
                          <div className="flex items-center justify-between gap-2">
                            <span className="truncate text-xs font-semibold text-slate-200">{profiles.find((profile) => profile.id === batch.profileId)?.name ?? batch.startUrl}</span>
                            <StatusBadge status={batch.status} />
                          </div>
                          <div className="mt-1 truncate font-mono text-[10px] text-slate-500">{batch.startUrl}</div>
                          <div className="mt-1 flex items-center justify-between text-[10px] text-slate-500">
                            <span>{batch.completedPages}/{batch.totalPages} pages</span>
                            <span>{timeAgo(batch.createdAt)}</span>
                          </div>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </Card>
            </div>

            <Card className="xl:col-span-3">
              <CardHeader
                title={selectedBatch ? "Evaluation results" : "Results"}
                subtitle={selectedBatch ? `${selectedBatch.startUrl} / ${formatDate(selectedBatch.createdAt)}` : "Select a batch to inspect per-regulation results."}
                right={selectedBatch ? <StatusBadge status={selectedBatch.status} /> : undefined}
              />
              {!selectedBatch ? (
                <EmptyState icon="scale" title="Nothing selected" description="Select an evaluation batch to view regulation scores." />
              ) : evaluations.length === 0 ? (
                <EmptyState icon="clock" title="No evaluations yet" description="The worker has not produced results for this batch." />
              ) : (
                <div className="space-y-3 p-4">
                  {evaluations.map((evaluation) => (
                    <div key={evaluation.id} className="rounded-xl border border-slate-800 bg-slate-950/40 p-4">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="min-w-0">
                          <p className="truncate text-xs font-semibold text-slate-200">{evaluation.url}</p>
                          <p className="mt-0.5 text-[10px] text-slate-500">{evaluation.finishedAt ? timeAgo(evaluation.finishedAt) : timeAgo(evaluation.createdAt)}</p>
                        </div>
                        <div className="flex items-center gap-2">
                          {evaluation.report && <Pill tone={evaluation.report.passed ? "emerald" : "rose"}>{evaluation.report.score}/100</Pill>}
                          <StatusBadge status={evaluation.status} />
                        </div>
                      </div>
                      {evaluation.error && <p className="mt-2 text-[11px] text-rose-300">{evaluation.error}</p>}
                      {evaluation.report && (
                        <div className="mt-3 space-y-2">
                          {evaluation.report.regulations.map((regulation) => (
                            <div key={regulation.regulation} className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
                              <div className="flex items-center justify-between gap-2 text-[11px]">
                                <span className="font-semibold uppercase tracking-wide text-slate-300">{regulation.regulation}</span>
                                <span className={regulation.passed ? "text-emerald-300" : "text-rose-300"}>{regulation.score}/100</span>
                              </div>
                              <div className="mt-1.5"><ProgressBar value={regulation.score} tone={regulation.passed ? "emerald" : "rose"} /></div>
                              {regulation.checks.filter((check) => check.status === "fail").slice(0, 4).map((check) => (
                                <p key={check.id} className="mt-1.5 text-[10px] text-rose-300">x {check.id}{check.message ? `: ${check.message}` : ""}</p>
                              ))}
                            </div>
                          ))}
                          {(evaluation.report.flags ?? []).length > 0 && (
                            <div className="rounded-lg border border-amber-900/50 bg-amber-950/20 px-3 py-2">
                              {(evaluation.report.flags ?? []).slice(0, 4).map((flag) => (
                                <p key={`${flag.id}-${flag.title}`} className="text-[10px] text-amber-200">{flag.id}: {flag.title}</p>
                              ))}
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </Card>
          </div>
        </>
      )}

      <Modal open={profileOpen} onClose={() => setProfileOpen(false)} title="New privacy profile" subtitle="Choose the jurisdiction and regulations to evaluate.">
        <form onSubmit={submitProfile} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="profile-name">Name</label>
              <input id="profile-name" className="input" value={profileName} onChange={(event) => setProfileName(event.target.value)} placeholder="EU storefront" required />
            </div>
            <div>
              <label className="label" htmlFor="profile-region">Region</label>
              <select id="profile-region" className="input" value={profileRegion} onChange={(event) => setProfileRegion(event.target.value)}>
                {REGION_OPTIONS.map((region) => <option key={region} value={region}>{region}</option>)}
              </select>
            </div>
          </div>
          <div>
            <p className="label">Regulations</p>
            <div className="flex flex-wrap gap-2">
              {REGULATIONS.map((regulation) => {
                const checked = profileRegs.includes(regulation.id);
                return (
                  <button
                    key={regulation.id}
                    type="button"
                    className={`rounded-lg border px-2.5 py-1.5 text-[11px] transition ${checked ? "border-indigo-700 bg-indigo-950/50 text-indigo-200" : "border-slate-800 bg-slate-900 text-slate-400"}`}
                    onClick={() => setProfileRegs((current) => checked ? current.filter((id) => id !== regulation.id) : [...current, regulation.id])}
                  >
                    {regulation.label}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setProfileOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={!profileName.trim() || profileRegs.length === 0 || busy === "profile"}>Create profile</button>
          </div>
        </form>
      </Modal>

      <Modal open={batchOpen} onClose={() => setBatchOpen(false)} title="New privacy evaluation" subtitle="The page is re-scanned under each consent state and scored per regulation.">
        <form onSubmit={submitBatch} className="space-y-4">
          <div>
            <p className="label">Profile</p>
            <Select value={batchProfileId} onChange={setBatchProfileId} options={profiles.map((profile) => ({ value: profile.id, label: `${profile.name} (${profile.region})` }))} label="Profile" />
          </div>
          <div>
            <label className="label" htmlFor="batch-url">Start URL</label>
            <input id="batch-url" className="input" value={batchUrl} onChange={(event) => setBatchUrl(event.target.value)} placeholder="https://www.example.com/" required />
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" className="btn" onClick={() => setBatchOpen(false)}>Cancel</button>
            <button type="submit" className="btn-primary" disabled={!batchProfileId || !batchUrl.trim() || busy === "batch"}>Start evaluation</button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
