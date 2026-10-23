"use client";

import type { TrendPoint, VendorSlice } from "../lib/api";

const VENDOR_COLORS = ["#818cf8", "#34d399", "#fbbf24", "#fb7185", "#38bdf8", "#a78bfa"];

export function TrendChart({ points }: { points: TrendPoint[] }) {
  const width = 640;
  const height = 220;
  const pad = { l: 38, r: 12, t: 16, b: 26 };
  const innerW = width - pad.l - pad.r;
  const innerH = height - pad.t - pad.b;
  const safe = points.length > 0 ? points : [{ date: "", health: 0, consent: 0, runs: 0 }];
  const x = (i: number) => pad.l + (safe.length <= 1 ? innerW / 2 : (i / (safe.length - 1)) * innerW);
  const y = (v: number) => pad.t + innerH - (Math.max(0, Math.min(100, v)) / 100) * innerH;
  const line = (key: "health" | "consent") =>
    safe.map((p, i) => `${i === 0 ? "M" : "L"} ${x(i).toFixed(1)} ${y(p[key]).toFixed(1)}`).join(" ");
  const area = `${line("health")} L ${x(safe.length - 1).toFixed(1)} ${pad.t + innerH} L ${x(0).toFixed(1)} ${pad.t + innerH} Z`;
  const label = (date: string) => {
    if (!date) return "";
    const d = new Date(`${date}T00:00:00Z`);
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric", timeZone: "UTC" });
  };

  return (
    <div className="w-full">
      <div className="mb-2 flex items-center gap-4 text-[11px] text-slate-400">
        <span className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-indigo-400" /> Health</span>
        <span className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-emerald-400" /> Consent</span>
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} className="h-auto w-full" role="img" aria-label="Seven day health trend">
        <defs>
          <linearGradient id="healthFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#818cf8" stopOpacity="0.26" />
            <stop offset="100%" stopColor="#818cf8" stopOpacity="0" />
          </linearGradient>
        </defs>
        {[0, 25, 50, 75, 100].map((v) => (
          <g key={v}>
            <line x1={pad.l} x2={width - pad.r} y1={y(v)} y2={y(v)} stroke="#1e293b" strokeDasharray="3 5" />
            <text x={pad.l - 8} y={y(v) + 3} textAnchor="end" fontSize="9" fill="#64748b">{v}</text>
          </g>
        ))}
        <path d={area} fill="url(#healthFill)" />
        <path d={line("health")} fill="none" stroke="#818cf8" strokeWidth="2.2" strokeLinejoin="round" strokeLinecap="round" />
        <path d={line("consent")} fill="none" stroke="#34d399" strokeWidth="1.8" strokeDasharray="4 4" strokeLinejoin="round" strokeLinecap="round" />
        {safe.map((p, i) => (
          <g key={`${p.date}-${i}`}>
            <circle cx={x(i)} cy={y(p.health)} r="2.6" fill="#0f172a" stroke="#818cf8" strokeWidth="2" />
            <circle cx={x(i)} cy={y(p.consent)} r="2.2" fill="#0f172a" stroke="#34d399" strokeWidth="1.6" />
            <text x={x(i)} y={height - 8} textAnchor="middle" fontSize="9" fill="#64748b">{label(p.date)}</text>
          </g>
        ))}
      </svg>
    </div>
  );
}

export function VendorDonut({ vendors }: { vendors: VendorSlice[] }) {
  const total = vendors.reduce((sum, v) => sum + v.count, 0);
  const radius = 52;
  const circumference = 2 * Math.PI * radius;
  let offset = 0;
  const safe = vendors.length > 0 ? vendors : [{ vendor: "No data", count: 1, percent: 100 }];

  return (
    <div className="flex flex-col items-center gap-4 sm:flex-row">
      <div className="relative h-40 w-40 shrink-0">
        <svg viewBox="0 0 140 140" className="h-full w-full -rotate-90">
          <circle cx="70" cy="70" r={radius} fill="none" stroke="#1e293b" strokeWidth="16" />
          {safe.map((v, i) => {
            const fraction = total > 0 ? v.count / total : 1 / safe.length;
            const dash = fraction * circumference;
            const segment = (
              <circle
                key={v.vendor}
                cx="70"
                cy="70"
                r={radius}
                fill="none"
                stroke={VENDOR_COLORS[i % VENDOR_COLORS.length]}
                strokeWidth="16"
                strokeDasharray={`${dash} ${circumference - dash}`}
                strokeDashoffset={-offset}
                strokeLinecap="butt"
              />
            );
            offset += dash;
            return segment;
          })}
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-2xl font-extrabold text-white">{total}</span>
          <span className="text-[10px] uppercase tracking-wide text-slate-500">signals</span>
        </div>
      </div>
      <ul className="w-full space-y-2">
        {safe.map((v, i) => (
          <li key={v.vendor} className="flex items-center justify-between gap-3 text-xs">
            <span className="flex min-w-0 items-center gap-2 text-slate-300">
              <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: VENDOR_COLORS[i % VENDOR_COLORS.length] }} />
              <span className="truncate">{v.vendor}</span>
            </span>
            <span className="shrink-0 font-mono text-slate-500">
              {total > 0 ? `${v.percent.toFixed(1)}%` : "—"}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
