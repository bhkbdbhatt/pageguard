"use client";

import { useEffect, useId, type ReactNode } from "react";
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
  const titleId = useId();
  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-slate-950/80 backdrop-blur-sm" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className={`relative w-full ${width} overflow-hidden rounded-2xl border border-slate-800 bg-slate-900 shadow-2xl`}
      >
        <div className="flex items-start justify-between gap-4 border-b border-slate-800 px-5 py-4">
          <div>
            <h2 id={titleId} className="text-sm font-semibold text-white">{title}</h2>
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

export type Tone = "slate" | "indigo" | "emerald" | "rose" | "amber" | "sky";

const TONE_HEX: Record<Tone, string> = {
  slate: "#94a3b8",
  indigo: "#818cf8",
  emerald: "#34d399",
  rose: "#fb7185",
  amber: "#fbbf24",
  sky: "#38bdf8",
};

export function Tabs({
  tabs,
  value,
  onChange,
  className = "",
}: {
  tabs: { id: string; label: string; count?: number }[];
  value: string;
  onChange: (id: string) => void;
  className?: string;
}) {
  return (
    <div
      role="tablist"
      className={`flex flex-wrap gap-1 rounded-xl border border-slate-800 bg-slate-900/60 p-1 ${className}`}
    >
      {tabs.map((tab, index) => {
        const active = tab.id === value;
        return (
          <button
            key={tab.id}
            role="tab"
            aria-selected={active}
            tabIndex={active ? 0 : -1}
            onClick={() => onChange(tab.id)}
            onKeyDown={(event) => {
              if (event.key !== "ArrowRight" && event.key !== "ArrowLeft") return;
              event.preventDefault();
              const next = event.key === "ArrowRight" ? (index + 1) % tabs.length : (index - 1 + tabs.length) % tabs.length;
              const buttons = event.currentTarget.parentElement?.querySelectorAll<HTMLButtonElement>('[role="tab"]');
              buttons?.[next]?.focus();
              onChange(tabs[next].id);
            }}
            className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition ${
              active ? "bg-indigo-600 text-white shadow-lg shadow-indigo-950/30" : "text-slate-400 hover:bg-slate-800 hover:text-slate-200"
            }`}
          >
            {tab.label}
            {tab.count !== undefined && (
              <span className={`rounded px-1 text-[10px] ${active ? "bg-indigo-500/60 text-white" : "bg-slate-800 text-slate-400"}`}>
                {tab.count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}

export function Toggle({
  checked,
  onChange,
  label,
  disabled,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      title={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition ${
        checked ? "border-indigo-500 bg-indigo-600" : "border-slate-700 bg-slate-800"
      } ${disabled ? "cursor-not-allowed opacity-50" : "hover:border-slate-500"}`}
    >
      <span className={`h-3.5 w-3.5 rounded-full bg-white transition ${checked ? "translate-x-[18px]" : "translate-x-0.5"}`} />
    </button>
  );
}

export function Drawer({
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
  const titleId = useId();
  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50">
      <div className="absolute inset-0 bg-slate-950/70 backdrop-blur-sm" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className={`absolute right-0 top-0 flex h-full w-full ${width} flex-col border-l border-slate-800 bg-slate-900 shadow-2xl`}
      >
        <div className="flex items-start justify-between gap-4 border-b border-slate-800 px-5 py-4">
          <div className="min-w-0">
            <h2 id={titleId} className="truncate text-sm font-semibold text-white">{title}</h2>
            {subtitle && <p className="mt-1 truncate text-xs text-slate-500">{subtitle}</p>}
          </div>
          <button onClick={onClose} className="rounded-lg p-1 text-slate-500 hover:bg-slate-800 hover:text-white" aria-label="Close panel">
            <Icon name="x" className="h-4 w-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-5">{children}</div>
      </div>
    </div>
  );
}

export function ProgressBar({ value, tone = "indigo" }: { value: number; tone?: Tone }) {
  const pct = Math.max(0, Math.min(100, value));
  return (
    <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-800">
      <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, background: TONE_HEX[tone] }} />
    </div>
  );
}

export function ProgressRing({
  value,
  size = 64,
  tone = "indigo",
  label,
}: {
  value: number;
  size?: number;
  tone?: Tone;
  label?: string;
}) {
  const stroke = Math.max(4, Math.round(size / 10));
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const pct = Math.max(0, Math.min(100, value));
  const dash = (pct / 100) * circumference;
  return (
    <div className="relative inline-flex shrink-0 items-center justify-center" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90" aria-hidden="true">
        <circle cx={size / 2} cy={size / 2} r={radius} fill="none" stroke="#1e293b" strokeWidth={stroke} />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={TONE_HEX[tone]}
          strokeWidth={stroke}
          strokeLinecap="round"
          strokeDasharray={`${dash} ${circumference - dash}`}
        />
      </svg>
      <span className="absolute inset-0 flex flex-col items-center justify-center leading-none">
        <span className="font-bold text-white" style={{ fontSize: Math.max(10, size / 4.6) }}>{Math.round(pct)}%</span>
        {label && <span className="mt-0.5 text-[9px] uppercase tracking-wide text-slate-500">{label}</span>}
      </span>
    </div>
  );
}

export function StatDelta({
  value,
  direction = "up",
  suffix = "",
}: {
  value: string;
  direction?: "up" | "down" | "flat";
  suffix?: string;
}) {
  const tone = direction === "down" ? "text-rose-300" : direction === "flat" ? "text-slate-400" : "text-emerald-300";
  const icon = direction === "down" ? "arrow-down" : direction === "flat" ? "arrow" : "arrow-up";
  return (
    <span className={`inline-flex items-center gap-1 text-[11px] font-semibold ${tone}`}>
      <Icon name={icon} className="h-3 w-3" />
      {value}
      {suffix}
    </span>
  );
}

export function Toolbar({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`flex flex-wrap items-center gap-2 ${className}`}>{children}</div>;
}

export function SearchInput({
  value,
  onChange,
  placeholder = "Search",
  className = "",
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}) {
  return (
    <div className={`relative ${className}`}>
      <Icon name="search" className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
      <input
        className="input h-9 py-1.5 pl-8 text-xs"
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
        aria-label={placeholder}
      />
    </div>
  );
}

export function Select({
  value,
  onChange,
  options,
  label,
  className = "",
}: {
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
  label?: string;
  className?: string;
}) {
  return (
    <label className={`flex items-center gap-2 text-[11px] text-slate-400 ${className}`}>
      {label && <span className="whitespace-nowrap">{label}</span>}
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        aria-label={label ?? "Filter"}
        className="h-9 rounded-lg border border-slate-700 bg-slate-800/80 px-2 text-xs text-slate-200 focus:border-indigo-500 focus:outline-none"
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </select>
    </label>
  );
}

export function PageHeader({
  eyebrow,
  title,
  subtitle,
  icon,
  actions,
}: {
  eyebrow: string;
  title: string;
  subtitle: string;
  icon?: Parameters<typeof Icon>[0]["name"];
  actions?: ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-3">
      <div className="min-w-0">
        <p className="text-[10px] font-semibold uppercase tracking-[0.25em] text-indigo-400">{eyebrow}</p>
        <div className="mt-1 flex flex-wrap items-center gap-2">
          {icon && (
            <span className="flex h-7 w-7 items-center justify-center rounded-lg border border-slate-800 bg-slate-900 text-indigo-300">
              <Icon name={icon} className="h-4 w-4" />
            </span>
          )}
          <h1 className="text-xl font-bold text-white sm:text-2xl">{title}</h1>
        </div>
        <p className="mt-1 max-w-3xl text-xs leading-relaxed text-slate-500">{subtitle}</p>
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  );
}
