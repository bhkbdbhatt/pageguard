"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import {
  createAudit,
  createProject,
  getHealth,
  listProjects,
  type Project,
} from "../lib/api";
import { Icon, type IconName } from "./icons";
import { Modal } from "./ui";

type AuditPrefill = { url?: string } | undefined;

interface AppContextValue {
  project: Project | null;
  projects: Project[];
  loadingProjects: boolean;
  setProject: (project: Project) => void;
  refreshProjects: () => Promise<Project[]>;
  openAudit: (prefill?: AuditPrefill) => void;
}

const AppContext = createContext<AppContextValue | null>(null);

export function useApp() {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useApp must be used inside AppShell");
  return ctx;
}

const NAV: { href: string; label: string; icon: IconName }[] = [
  { href: "/", label: "Overview Dashboard", icon: "grid" },
  { href: "/audits", label: "Audit Runs & Crawler", icon: "radar" },
  { href: "/inspector", label: "Beacon & DataLayer", icon: "code" },
  { href: "/rules", label: "Rules Configurator", icon: "shield" },
  { href: "/consent", label: "Consent & CMP Matrix", icon: "cookie" },
];

export default function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [projects, setProjects] = useState<Project[]>([]);
  const [project, setProjectState] = useState<Project | null>(null);
  const [loadingProjects, setLoadingProjects] = useState(true);
  const [inflight, setInflight] = useState(0);
  const [auditOpen, setAuditOpen] = useState(false);
  const [auditPrefill, setAuditPrefill] = useState<AuditPrefill>();
  const [projectOpen, setProjectOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshProjects = useCallback(async () => {
    const list = await listProjects();
    setProjects(list);
    setProjectState((current) => {
      if (current && list.some((p) => p.id === current.id)) return current;
      const stored = typeof window !== "undefined" ? window.localStorage.getItem("pageguard.project") : null;
      const preferred = list.find((p) => p.id === stored);
      const next = preferred ?? list[0] ?? null;
      if (next && typeof window !== "undefined") window.localStorage.setItem("pageguard.project", next.id);
      return next;
    });
    return list;
  }, []);

  useEffect(() => {
    refreshProjects()
      .catch((err) => setError((err as Error).message))
      .finally(() => setLoadingProjects(false));
  }, [refreshProjects]);

  useEffect(() => {
    let active = true;
    const tick = () =>
      getHealth()
        .then((h) => active && setInflight(h.inflight))
        .catch(() => undefined);
    tick();
    const id = window.setInterval(tick, 8000);
    return () => {
      active = false;
      window.clearInterval(id);
    };
  }, []);

  const setProject = useCallback((p: Project) => {
    setProjectState(p);
    if (typeof window !== "undefined") window.localStorage.setItem("pageguard.project", p.id);
  }, []);

  const openAudit = useCallback((prefill?: AuditPrefill) => {
    setAuditPrefill(prefill);
    setAuditOpen(true);
  }, []);

  const value = useMemo(
    () => ({ project, projects, loadingProjects, setProject, refreshProjects, openAudit }),
    [project, projects, loadingProjects, setProject, refreshProjects, openAudit],
  );

  return (
    <AppContext.Provider value={value}>
      <div className="min-h-screen lg:flex">
        <aside className="hidden w-64 shrink-0 flex-col border-r border-slate-800 bg-slate-950/80 lg:flex">
          <div className="flex h-16 items-center gap-2.5 border-b border-slate-800 px-5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600 text-white shadow-lg shadow-indigo-950/40">
              <Icon name="shield" className="h-4 w-4" />
            </div>
            <div>
              <div className="text-[11px] font-extrabold uppercase tracking-widest text-white">OpenTag</div>
              <div className="text-[10px] font-semibold uppercase tracking-[0.22em] text-indigo-400">Validator</div>
            </div>
          </div>
          <nav className="flex-1 space-y-1 px-3 py-4">
            <p className="px-2 pb-2 text-[10px] font-semibold uppercase tracking-widest text-slate-600">Governance</p>
            {NAV.map((item) => {
              const active = item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-xs font-medium transition ${
                    active
                      ? "border border-indigo-800/50 bg-indigo-950/40 text-indigo-200"
                      : "border border-transparent text-slate-400 hover:bg-slate-900 hover:text-slate-200"
                  }`}
                >
                  <Icon name={item.icon} className="h-4 w-4" />
                  {item.label}
                </Link>
              );
            })}
          </nav>
          <div className="border-t border-slate-800 p-3">
            <div className="rounded-xl border border-slate-800 bg-slate-900/70 p-3">
              <div className="flex items-center gap-2 text-[11px] font-semibold text-slate-300">
                <span className={`h-2 w-2 rounded-full ${inflight > 0 ? "animate-pulse bg-emerald-400" : "bg-slate-600"}`} />
                {inflight > 0 ? `${inflight} run${inflight === 1 ? "" : "s"} in flight` : "Scanner idle"}
              </div>
              <p className="mt-1 text-[10px] leading-relaxed text-slate-500">
                Evidence is captured before consent. Reports are derived from stored page checkpoints.
              </p>
            </div>
          </div>
        </aside>

        <div className="flex min-w-0 flex-1 flex-col">
          <header className="sticky top-0 z-30 border-b border-slate-800 bg-slate-950/85 backdrop-blur">
            <div className="flex h-16 items-center justify-between gap-3 px-4 sm:px-6">
              <div className="flex items-center gap-3">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600 text-white lg:hidden">
                  <Icon name="shield" className="h-4 w-4" />
                </div>
                <div>
                  <h1 className="text-sm font-bold text-white">Tag Governance Console</h1>
                  <p className="hidden text-[11px] text-slate-500 sm:block">
                    Beacon validation, data layer inspection and consent enforcement
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <div className="hidden items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/70 px-2.5 py-1.5 sm:flex">
                  <Icon name="activity" className="h-3.5 w-3.5 text-emerald-400" />
                  <span className="text-[10px] font-semibold uppercase tracking-wide text-slate-400">
                    {inflight} live
                  </span>
                </div>
                <div className="relative">
                  <select
                    aria-label="Project"
                    value={project?.id ?? ""}
                    onChange={(e) => {
                      const next = projects.find((p) => p.id === e.target.value);
                      if (next) setProject(next);
                    }}
                    className="h-9 max-w-[150px] rounded-lg border border-slate-700 bg-slate-900 px-2.5 text-xs text-slate-200 focus:border-indigo-500 focus:outline-none sm:max-w-[220px]"
                  >
                    {projects.length === 0 && <option value="">No project</option>}
                    {projects.map((p) => (
                      <option key={p.id} value={p.id}>{p.name}</option>
                    ))}
                  </select>
                </div>
                <button onClick={() => setProjectOpen(true)} className="btn hidden sm:inline-flex" title="Create project">
                  <Icon name="plus" className="h-3.5 w-3.5" />
                  Project
                </button>
                <button onClick={() => openAudit()} className="btn-primary h-9">
                  <Icon name="play" className="h-3.5 w-3.5" />
                  Run Audit
                </button>
              </div>
            </div>
            <nav className="flex gap-1 overflow-x-auto border-t border-slate-800/70 px-3 py-2 lg:hidden">
              {NAV.map((item) => {
                const active = item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`flex shrink-0 items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-[11px] font-medium ${
                      active ? "bg-indigo-950/50 text-indigo-200" : "text-slate-400"
                    }`}
                  >
                    <Icon name={item.icon} className="h-3.5 w-3.5" />
                    {item.label.split(" ")[0]}
                  </Link>
                );
              })}
            </nav>
          </header>

          <main className="min-w-0 flex-1 px-4 py-5 sm:px-6 sm:py-6">{children}</main>
        </div>
      </div>

      <RunAuditModal
        open={auditOpen}
        prefill={auditPrefill}
        projectId={project?.id}
        onClose={() => setAuditOpen(false)}
        onCreated={(runId) => {
          setAuditOpen(false);
          router.push(`/audits?run=${runId}`);
        }}
      />
      <NewProjectModal
        open={projectOpen}
        onClose={() => setProjectOpen(false)}
        onCreated={(p) => {
          setProject(p);
          setProjectOpen(false);
        }}
        onRefresh={refreshProjects}
      />
      {error && (
        <div className="fixed bottom-4 left-4 z-50 max-w-sm rounded-xl border border-rose-800/60 bg-rose-950/90 px-4 py-3 text-xs text-rose-200 shadow-xl">
          {error}
          <button onClick={() => setError(null)} className="ml-3 font-bold text-rose-300">Dismiss</button>
        </div>
      )}
    </AppContext.Provider>
  );
}

function RunAuditModal({
  open,
  prefill,
  projectId,
  onClose,
  onCreated,
}: {
  open: boolean;
  prefill: AuditPrefill;
  projectId?: string;
  onClose: () => void;
  onCreated: (runId: string) => void;
}) {
  const [url, setUrl] = useState("");
  const [maxPages, setMaxPages] = useState("50");
  const [maxDepth, setMaxDepth] = useState("3");
  const [consent, setConsent] = useState("pre_consent");
  const [geo, setGeo] = useState("");
  const [mode, setMode] = useState("auto");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setUrl(prefill?.url ?? "");
      setError(null);
    }
  }, [open, prefill]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const result = await createAudit({
        projectId,
        startUrl: url,
        maxPages: Number(maxPages) || 50,
        maxDepth: Number(maxDepth) || 3,
        consentSimulation: consent,
        geolocation: geo || undefined,
        renderMode: mode as "auto" | "http" | "browser",
      });
      onCreated(result.run.id);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Run Tag Governance Audit" subtitle="Crawl a site, capture beacons and evaluate configured rules.">
      <form onSubmit={submit} className="space-y-4">
        {!projectId && (
          <p className="rounded-lg border border-amber-800/60 bg-amber-950/40 px-3 py-2 text-xs text-amber-200">
            Create a project first using the Project button.
          </p>
        )}
        <div>
          <label className="label">Start URL</label>
          <input className="input" required placeholder="https://www.example.com/" value={url} onChange={(e) => setUrl(e.target.value)} />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">Max pages</label>
            <input className="input mono" type="number" min={1} max={5000} value={maxPages} onChange={(e) => setMaxPages(e.target.value)} />
          </div>
          <div>
            <label className="label">Max depth</label>
            <input className="input mono" type="number" min={1} max={10} value={maxDepth} onChange={(e) => setMaxDepth(e.target.value)} />
          </div>
        </div>
        <div>
          <label className="label">Consent simulation</label>
          <select className="input" value={consent} onChange={(e) => setConsent(e.target.value)}>
            <option value="pre_consent">Pre-Consent State Only (No interaction)</option>
            <option value="accept_all">Simulate Full Opt-In (Accept All)</option>
            <option value="reject_all">Simulate Opt-Out</option>
          </select>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">Simulated geo</label>
            <select className="input" value={geo} onChange={(e) => setGeo(e.target.value)}>
              <option value="">Default / no proxy</option>
              <option value="EU">EU</option>
              <option value="US">US</option>
              <option value="UK">UK</option>
            </select>
          </div>
          <div>
            <label className="label">Render mode</label>
            <select className="input" value={mode} onChange={(e) => setMode(e.target.value)}>
              <option value="auto">Auto (HTTP + browser fallback)</option>
              <option value="browser">Headless browser</option>
              <option value="http">HTTP only</option>
            </select>
          </div>
        </div>
        {error && <p className="rounded-lg border border-rose-800/60 bg-rose-950/40 px-3 py-2 text-xs text-rose-200">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" className="btn" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn-primary" disabled={busy || !projectId || !url}>
            <Icon name="play" className="h-3.5 w-3.5" />
            {busy ? "Starting…" : "Start audit"}
          </button>
        </div>
      </form>
    </Modal>
  );
}

function NewProjectModal({
  open,
  onClose,
  onCreated,
  onRefresh,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: (project: Project) => void;
  onRefresh: () => Promise<Project[]>;
}) {
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setName("");
      setError(null);
    }
  }, [open]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const project = await createProject(name);
      await onRefresh();
      onCreated(project);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Create project" subtitle="Projects scope scans, rules and governance reports." width="max-w-md">
      <form onSubmit={submit} className="space-y-4">
        <div>
          <label className="label">Project name</label>
          <input className="input" required placeholder="Acme Commerce" value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        {error && <p className="text-xs text-rose-300">{error}</p>}
        <div className="flex justify-end gap-2">
          <button type="button" className="btn" onClick={onClose}>Cancel</button>
          <button type="submit" className="btn-primary" disabled={busy || !name.trim()}>
            {busy ? "Creating…" : "Create project"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
