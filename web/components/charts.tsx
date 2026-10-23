"use client";

import type { TrendPoint, VendorSlice } from "../lib/api";
import { Icon } from "./icons";
import { ProgressRing, type Tone } from "./ui";

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

export function HorizontalBars({
  data,
}: {
  data: { label: string; value: number; hint?: string; color?: string }[];
}) {
  const max = Math.max(1, ...data.map((item) => item.value));
  return (
    <ul className="space-y-3">
      {data.map((item, index) => (
        <li key={item.label}>
          <div className="flex items-center justify-between gap-3 text-xs">
            <span className="truncate text-slate-300">{item.label}</span>
            <span className="shrink-0 font-mono text-[11px] text-slate-500">{item.hint ?? item.value}</span>
          </div>
          <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-slate-800">
            <div
              className="h-full rounded-full transition-all"
              style={{
                width: `${(item.value / max) * 100}%`,
                background: item.color ?? VENDOR_COLORS[index % VENDOR_COLORS.length],
              }}
            />
          </div>
        </li>
      ))}
    </ul>
  );
}

export function BarChart({
  data,
  height = 190,
  color = "#818cf8",
  suffix = "",
}: {
  data: { label: string; value: number }[];
  height?: number;
  color?: string;
  suffix?: string;
}) {
  const width = Math.max(320, data.length * 58);
  const pad = { l: 32, r: 12, t: 14, b: 30 };
  const innerW = width - pad.l - pad.r;
  const innerH = height - pad.t - pad.b;
  const max = Math.max(1, ...data.map((item) => item.value));
  const slot = innerW / Math.max(1, data.length);
  const barWidth = Math.min(28, slot * 0.55);

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="h-auto w-full" role="img" aria-label="Bar chart">
      {[0, 0.5, 1].map((fraction) => {
        const value = max * fraction;
        const y = pad.t + innerH - fraction * innerH;
        return (
          <g key={fraction}>
            <line x1={pad.l} x2={width - pad.r} y1={y} y2={y} stroke="#1e293b" strokeDasharray="3 5" />
            <text x={pad.l - 8} y={y + 3} textAnchor="end" fontSize="9" fill="#64748b">
              {Math.round(value)}
            </text>
          </g>
        );
      })}
      {data.map((item, index) => {
        const barHeight = (item.value / max) * innerH;
        const x = pad.l + index * slot + (slot - barWidth) / 2;
        const y = pad.t + innerH - barHeight;
        return (
          <g key={`${item.label}-${index}`}>
            <rect x={x} y={y} width={barWidth} height={barHeight} rx={4} fill={color} fillOpacity="0.85" />
            <text x={x + barWidth / 2} y={y - 5} textAnchor="middle" fontSize="9" fill="#94a3b8">
              {item.value}{suffix}
            </text>
            <text x={x + barWidth / 2} y={height - 9} textAnchor="middle" fontSize="9" fill="#64748b">
              {item.label}
            </text>
          </g>
        );
      })}
    </svg>
  );
}

export function Sparkline({
  points,
  color = "#34d399",
  height = 36,
}: {
  points: number[];
  color?: string;
  height?: number;
}) {
  if (points.length === 0) return null;
  const width = 120;
  const max = Math.max(...points);
  const min = Math.min(...points);
  const range = max - min || 1;
  const coords = points
    .map((point, index) => {
      const x = points.length === 1 ? width / 2 : (index / (points.length - 1)) * width;
      const y = height - ((point - min) / range) * (height - 4) - 2;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="h-9 w-full" preserveAspectRatio="none" aria-hidden="true">
      <polyline points={coords} fill="none" stroke={color} strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  );
}

export function StackedBar({
  segments,
  legend = true,
}: {
  segments: { label: string; value: number; color: string }[];
  legend?: boolean;
}) {
  const total = Math.max(1, segments.reduce((sum, segment) => sum + segment.value, 0));
  return (
    <div>
      <div className="flex h-2.5 overflow-hidden rounded-full bg-slate-800">
        {segments.map((segment) => (
          <div
            key={segment.label}
            title={`${segment.label}: ${segment.value}`}
            style={{ width: `${(segment.value / total) * 100}%`, background: segment.color }}
          />
        ))}
      </div>
      {legend && (
        <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1.5">
          {segments.map((segment) => (
            <span key={segment.label} className="flex items-center gap-1.5 text-[11px] text-slate-400">
              <span className="h-2 w-2 rounded-full" style={{ background: segment.color }} />
              {segment.label}
              <span className="font-mono text-slate-500">{segment.value}</span>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

export function Heatmap({
  rows,
  columns,
  cells,
}: {
  rows: string[];
  columns: string[];
  cells: ("pass" | "fail" | "na")[][];
}) {
  const cellStyles = {
    pass: "border-emerald-800/60 bg-emerald-950/40 text-emerald-300",
    fail: "border-rose-800/60 bg-rose-950/40 text-rose-300",
    na: "border-slate-800 bg-slate-900/60 text-slate-600",
  };
  const cellLabels = { pass: "Pass", fail: "Fail", na: "Not applicable" };
  const cellIcons = { pass: "check-circle", fail: "x-circle", na: "x" } as const;

  return (
    <div className="overflow-x-auto">
      <div
        className="grid min-w-[560px] gap-1.5"
        style={{ gridTemplateColumns: `minmax(190px,1fr) repeat(${columns.length}, minmax(74px,1fr))` }}
      >
        <div />
        {columns.map((column) => (
          <div key={column} className="px-2 py-1 text-center text-[10px] font-semibold uppercase tracking-wider text-slate-500">
            {column}
          </div>
        ))}
        {rows.map((row, rowIndex) => [
          <div key={`${row}-label`} className="flex items-center px-2 py-2 text-xs text-slate-300">{row}</div>,
          ...columns.map((column, columnIndex) => {
            const status = cells[rowIndex]?.[columnIndex] ?? "na";
            return (
              <div
                key={`${row}-${column}`}
                title={cellLabels[status]}
                className={`flex items-center justify-center rounded-lg border py-2 ${cellStyles[status]}`}
              >
                <span className="sr-only">{cellLabels[status]}</span>
                <Icon name={cellIcons[status]} className="h-3.5 w-3.5" />
              </div>
            );
          }),
        ])}
      </div>
    </div>
  );
}

export function RadialGauge({
  value,
  label,
  tone = "indigo",
  size = 92,
}: {
  value: number;
  label?: string;
  tone?: Tone;
  size?: number;
}) {
  return (
    <div className="flex flex-col items-center gap-1.5">
      <ProgressRing value={value} size={size} tone={tone} />
      {label && <span className="text-[10px] font-semibold uppercase tracking-wide text-slate-500">{label}</span>}
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
