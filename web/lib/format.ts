export function formatNumber(value?: number): string {
  return new Intl.NumberFormat("en-US").format(value ?? 0);
}

export function formatPercent(value?: number, digits = 1): string {
  return `${(value ?? 0).toFixed(digits)}%`;
}

export function shortId(id?: string): string {
  return id ? id.slice(0, 8) : "—";
}

export function formatDate(value?: string): string {
  if (!value) return "—";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function timeAgo(value?: string): string {
  if (!value) return "—";
  const then = new Date(value).getTime();
  if (Number.isNaN(then)) return "—";
  const seconds = Math.max(0, Math.floor((Date.now() - then) / 1000));
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export function duration(seconds?: number): string {
  if (!seconds) return "0s";
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m < 60) return `${m}m ${s}s`;
  return `${Math.floor(m / 60)}h ${m % 60}m`;
}

export function bytes(value?: number): string {
  if (value === undefined || value === null) return "—";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  if (value < 1024 * 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

export function hostOf(raw?: string): string {
  if (!raw) return "";
  try {
    return new URL(raw).hostname;
  } catch {
    return raw;
  }
}

export function pluralize(count: number, singular: string, plural = `${singular}s`): string {
  return `${formatNumber(count)} ${count === 1 ? singular : plural}`;
}

export function titleCase(value: string): string {
  return value
    .split(/[\s_-]+/)
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

export function cronLabel(cron: string): string {
  const presets: Record<string, string> = {
    "*/15 * * * *": "Every 15 minutes",
    "0 * * * *": "Hourly",
    "0 6 * * *": "Daily · 06:00",
    "0 6 * * 1": "Weekly · Mon 06:00",
    "0 2 1 * *": "Monthly · 1st 02:00",
    "30 8 * * 1-5": "Weekdays · 08:30",
  };
  return presets[cron] ?? cron;
}
