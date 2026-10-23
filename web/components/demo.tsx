"use client";

import { useState, type ReactNode } from "react";
import { Icon, type IconName } from "./icons";

export function DemoBadge({ className = "" }: { className?: string }) {
  return (
    <span
      className={`inline-flex items-center rounded-md border border-amber-700/60 bg-amber-950/40 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider text-amber-300 ${className}`}
    >
      Demo
    </span>
  );
}

export function DemoBanner({ children }: { children?: ReactNode }) {
  const [open, setOpen] = useState(true);
  if (!open) return null;
  return (
    <div className="flex items-start gap-2 rounded-xl border border-amber-800/50 bg-amber-950/25 px-4 py-2.5 text-[11px] leading-relaxed text-amber-200/90">
      <Icon name="info" className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span className="flex-1">
        {children ?? "Sample data for design review — not a live audit. Every action on this page is local to the browser."}
      </span>
      <button
        onClick={() => setOpen(false)}
        className="rounded p-0.5 text-amber-300/70 hover:bg-amber-900/40 hover:text-amber-200"
        aria-label="Dismiss demo notice"
      >
        <Icon name="x" className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

export function DemoPageHeader({
  eyebrow,
  title,
  subtitle,
  icon,
  actions,
}: {
  eyebrow: string;
  title: string;
  subtitle: string;
  icon?: IconName;
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
          <DemoBadge />
        </div>
        <p className="mt-1 max-w-3xl text-xs leading-relaxed text-slate-500">{subtitle}</p>
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  );
}
