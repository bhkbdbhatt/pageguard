"use client";

import type { ReactNode } from "react";
import type { RunStatus, Severity } from "../lib/api";
import { Icon } from "./icons";

export function Card({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return <section className={`card ${className}`}>{children}</section>;
}

export function CardHeader({
  title,
  subtitle,
  right,
  icon,
}: {
  title: string;
  subtitle?: string;
  right?: ReactNode;
  icon?: ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-3 border-b border-slate-800 px-5 py-4">
      <div className="flex min-w-0 items-start gap-3">
        {icon && <div className="mt-0.5 text-slate-400">{icon}</div>}
        <div className="min-w-0">
          <h2 className="truncate text-sm font-semibold text-white">{title}</h2>
          {subtitle && <p className="mt-0.5 text-xs text-slate-500">{subtitle}</p>}
        </div>
      </div>
      {right}
    </div>
  );
}

const SEVERITY_STYLES: Record<Severity, string> = {
  critical: "border-rose-800/60 bg-rose-950/50 text-rose-300",
  high: "border-orange-800/60 bg-orange-950/40 text-orange-300",
  medium: "border-amber-800/60 bg-amber-950/40 text-amber-300",
  low: "border-sky-800/60 bg-sky-950/40 text-sky-300",
  info: "border-slate-700 bg-slate-800/60 text-slate-300",
};

export function SeverityBadge({ severity }: { severity: Severity }) {
  return (
    <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide ${SEVERITY_STYLES[severity] ?? SEVERITY_STYLES.info}`}>
      {severity}
    </span>
  );
}

const STATUS_STYLES: Record<RunStatus, string> = {
  queued: "border-slate-700 bg-slate-800/60 text-slate-300",
  running: "border-sky-800/60 bg-sky-950/50 text-sky-300",
  succeeded: "border-emerald-800/60 bg-emerald-950/50 text-emerald-300",
  failed: "border-rose-800/60 bg-rose-950/50 text-rose-300",
  canceled: "border-amber-800/60 bg-amber-950/40 text-amber-300",
};

export function StatusBadge({ status }: { status: RunStatus }) {
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wide ${STATUS_STYLES[status] ?? STATUS_STYLES.queued}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${status === "running" ? "animate-pulse bg-sky-400" : status === "succeeded" ? "bg-emerald-400" : status === "failed" ? "bg-rose-400" : status === "canceled" ? "bg-amber-400" : "bg-slate-400"}`} />
      {status}
    </span>
  );
}

export function Pill({
  children,
  tone = "slate",
}: {
  children: ReactNode;
  tone?: "slate" | "indigo" | "emerald" | "rose" | "amber" | "sky";
}) {
  const tones = {
    slate: "border-slate-700 bg-slate-800/60 text-slate-300",
    indigo: "border-indigo-800/60 bg-indigo-950/50 text-indigo-300",
    emerald: "border-emerald-800/60 bg-emerald-950/50 text-emerald-300",
    rose: "border-rose-800/60 bg-rose-950/50 text-rose-300",
    amber: "border-amber-800/60 bg-amber-950/40 text-amber-300",
    sky: "border-sky-800/60 bg-sky-950/40 text-sky-300",
  };
  return (
    <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${tones[tone]}`}>
      {children}
    </span>
  );
}

export function EmptyState({
  icon = "radar",
  title,
  description,
  action,
}: {
  icon?: Parameters<typeof Icon>[0]["name"];
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center px-6 py-14 text-center">
      <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-2xl border border-slate-800 bg-slate-900 text-slate-500">
        <Icon name={icon} className="h-6 w-6" />
      </div>
      <h3 className="text-sm font-semibold text-white">{title}</h3>
      <p className="mt-1 max-w-md text-xs leading-relaxed text-slate-500">{description}</p>
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

export function Loading({ label = "Loading" }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 px-6 py-14 text-xs text-slate-500">
      <span className="h-3 w-3 animate-spin rounded-full border-2 border-slate-700 border-t-indigo-400" />
      {label}…
    </div>
  );
}

export function ErrorBanner({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <div className="flex items-start gap-2 rounded-xl border border-rose-800/60 bg-rose-950/40 px-4 py-3 text-xs text-rose-200">
      <Icon name="alert" className="mt-0.5 h-4 w-4 shrink-0" />
      <span className="break-words">{message}</span>
    </div>
  );
}

export function Modal({
  open,
  onClose,
  title,
  subtitle,
  children,
  width = "max-w-xl",
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle?: string;
  children: ReactNode;
  width?: string;
}) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-slate-950/80 backdrop-blur-sm" onClick={onClose} />
      <div className={`relative w-full ${width} overflow-hidden rounded-2xl border border-slate-800 bg-slate-900 shadow-2xl`}>
        <div className="flex items-start justify-between gap-4 border-b border-slate-800 px-5 py-4">
          <div>
            <h2 className="text-sm font-semibold text-white">{title}</h2>
            {subtitle && <p className="mt-1 text-xs text-slate-500">{subtitle}</p>}
          </div>
          <button onClick={onClose} className="rounded-lg p-1 text-slate-500 hover:bg-slate-800 hover:text-white" aria-label="Close">
            <Icon name="x" className="h-4 w-4" />
          </button>
        </div>
        <div className="max-h-[80vh] overflow-y-auto p-5">{children}</div>
      </div>
    </div>
  );
}

export function KpiCard({
  label,
  value,
  hint,
  tone = "slate",
  icon,
  onClick,
}: {
  label: string;
  value: string;
  hint?: string;
  tone?: "slate" | "indigo" | "emerald" | "rose" | "amber" | "sky";
  icon: Parameters<typeof Icon>[0]["name"];
  onClick?: () => void;
}) {
  const tones = {
    slate: "from-slate-800/70 to-slate-900/30 text-slate-300",
    indigo: "from-indigo-950/70 to-slate-900/30 text-indigo-300",
    emerald: "from-emerald-950/70 to-slate-900/30 text-emerald-300",
    rose: "from-rose-950/70 to-slate-900/30 text-rose-300",
    amber: "from-amber-950/60 to-slate-900/30 text-amber-300",
    sky: "from-sky-950/70 to-slate-900/30 text-sky-300",
  };
  const Tag = onClick ? "button" : "div";
  return (
    <Tag
      onClick={onClick}
      className={`card group relative overflow-hidden bg-gradient-to-br p-4 text-left ${tones[tone]} ${onClick ? "hover:border-slate-700" : ""}`}
    >
      <div className="flex items-center justify-between">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-slate-500">{label}</span>
        <Icon name={icon} className="h-4 w-4 opacity-70" />
      </div>
      <div className="mt-3 text-2xl font-extrabold text-white">{value}</div>
      {hint && <div className="mt-1 text-[11px] text-slate-500">{hint}</div>}
    </Tag>
  );
}
