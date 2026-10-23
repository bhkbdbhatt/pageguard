"use client";

import { useMemo } from "react";
import type { ScanResult } from "../lib/api";

interface SitemapNode {
  url: string;
  title?: string;
  status: number;
  error?: string;
  children: SitemapNode[];
}

function normalize(url: string): string {
  try {
    const u = new URL(url);
    u.hash = "";
    return u.toString();
  } catch {
    return url;
  }
}

function dotClass(n: SitemapNode): string {
  if (n.error) return "bg-red-500";
  if (n.status >= 400) return "bg-orange-500";
  if (n.status >= 300) return "bg-amber-500";
  return "bg-emerald-500";
}

function NodeList({
  nodes,
  path,
  onSelect,
}: {
  nodes: SitemapNode[];
  path: Set<string>;
  onSelect: (url: string) => void;
}) {
  return (
    <ul className="space-y-0.5">
      {nodes.map((n) => {
        const cycle = path.has(n.url);
        const nextPath = cycle ? path : new Set(path).add(n.url);
        return (
          <li key={n.url}>
            <button
              onClick={() => onSelect(n.url)}
              className="flex w-full items-center gap-2 rounded px-1 py-0.5 text-left hover:bg-slate-50"
              title={n.url}
            >
              <span className={`h-2 w-2 shrink-0 rounded-full ${dotClass(n)}`} />
              <span className="truncate text-slate-700">{n.title || n.url}</span>
              {n.title && <span className="truncate text-xs text-slate-400">{n.url}</span>}
              {cycle && <span className="shrink-0 text-xs text-slate-400">(cycle)</span>}
            </button>
            {!cycle && n.children.length > 0 && (
              <div className="ml-3 border-l border-slate-200 pl-3">
                <NodeList nodes={n.children} path={nextPath} onSelect={onSelect} />
              </div>
            )}
          </li>
        );
      })}
    </ul>
  );
}

export default function Sitemap({
  results,
  onSelect,
}: {
  results: ScanResult[];
  onSelect: (url: string) => void;
}) {
  const { roots, count, maxDepth } = useMemo(() => {
    const byURL = new Map<string, SitemapNode>();
    const byNorm = new Map<string, string>();
    for (const r of results) {
      byURL.set(r.url, {
        url: r.url,
        title: r.title,
        status: r.statusCode,
        error: r.error,
        children: [],
      });
      byNorm.set(normalize(r.url), r.url);
    }

    const hasParent = new Set<string>();
    for (const r of results) {
      const node = byURL.get(r.url);
      if (!node) continue;
      const seen = new Set<string>();
      for (const raw of r.links ?? []) {
        const key = byURL.has(raw) ? raw : byNorm.get(normalize(raw));
        if (!key || key === r.url || seen.has(key)) continue;
        seen.add(key);
        const child = byURL.get(key);
        if (!child) continue;
        node.children.push(child);
        hasParent.add(key);
      }
    }

    const all = [...byURL.values()];
    for (const n of all) n.children.sort((a, b) => a.url.localeCompare(b.url));

    let roots = all.filter((n) => !hasParent.has(n.url));
    if (roots.length === 0) roots = all.filter((n) => n.children.length === 0);
    if (roots.length === 0) roots = all;

    const depthOf = new Map(results.map((r) => [r.url, r.depth]));
    roots.sort((a, b) => (depthOf.get(a.url) ?? 0) - (depthOf.get(b.url) ?? 0) || a.url.localeCompare(b.url));

    const maxDepth = results.reduce((m, r) => Math.max(m, r.depth), 0);
    return { roots, count: all.length, maxDepth };
  }, [results]);

  if (results.length === 0) {
    return (
      <p className="text-sm text-slate-500">
        The sitemap is built from per-page checkpoints and appears as pages are scanned.
      </p>
    );
  }

  return (
    <div className="space-y-2 text-sm">
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
        <span>{count} pages</span>
        <span>max depth {maxDepth}</span>
        <span className="flex items-center gap-1">
          <span className="h-2 w-2 rounded-full bg-emerald-500" /> ok
        </span>
        <span className="flex items-center gap-1">
          <span className="h-2 w-2 rounded-full bg-amber-500" /> redirect
        </span>
        <span className="flex items-center gap-1">
          <span className="h-2 w-2 rounded-full bg-orange-500" /> 4xx/5xx
        </span>
        <span className="flex items-center gap-1">
          <span className="h-2 w-2 rounded-full bg-red-500" /> error
        </span>
      </div>
      <NodeList nodes={roots} path={new Set()} onSelect={onSelect} />
    </div>
  );
}
