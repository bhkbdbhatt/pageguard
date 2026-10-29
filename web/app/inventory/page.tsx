"use client";

import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useApp } from "../../components/AppShell";
import { Icon } from "../../components/icons";
import {
  Card,
  Drawer,
  EmptyState,
  ErrorBanner,
  KpiCard,
  Loading,
  PageHeader,
  Pill,
  ProgressBar,
  SearchInput,
  Select,
  Tabs,
} from "../../components/ui";
import {
  getInventory,
  type InventoryCookie,
  type InventoryRequest,
  type InventorySnapshot,
  type InventoryTag,
} from "../../lib/api";
import { bytes, formatDate, formatNumber, formatPercent } from "../../lib/format";

type Tab = "tags" | "cookies" | "requests";
type Party = "all" | "first" | "third";

type Selection =
  | { kind: "tag"; item: InventoryTag }
  | { kind: "cookie"; item: InventoryCookie }
  | { kind: "request"; item: InventoryRequest };

const CATEGORY_TONES: Record<string, "indigo" | "emerald" | "amber" | "rose" | "sky" | "slate"> = {
  analytics: "emerald",
  advertising: "rose",
  consent: "indigo",
  behavior: "amber",
  "tag_manager": "sky",
};

function categoryTone(category: string) {
  return CATEGORY_TONES[String(category).toLowerCase()] ?? "slate";
}

function Detail({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
      <p className="text-[10px] font-semibold uppercase tracking-wider text-slate-500">{label}</p>
      <div className="mt-1 break-words text-xs text-slate-200">{children}</div>
    </div>
  );
}

function csvEscape(value: unknown): string {
  const text = String(value ?? "");
  return /[",\n]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text;
}

function downloadCsv(filename: string, header: string[], rows: unknown[][]) {
  const body = [header, ...rows].map((row) => row.map(csvEscape).join(",")).join("\n");
  const blob = new Blob([body], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export default function InventoryPage() {
  const { project, loadingProjects } = useApp();
  const [snapshot, setSnapshot] = useState<InventorySnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>("tags");
  const [query, setQuery] = useState("");
  const [vendor, setVendor] = useState("all");
  const [category, setCategory] = useState("all");
  const [party, setParty] = useState<Party>("all");
  const [selected, setSelected] = useState<Selection | null>(null);

  const load = useCallback(async () => {
    if (!project) {
      setSnapshot(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      setSnapshot(await getInventory(project.id));
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [project]);

  useEffect(() => {
    void load();
  }, [load]);

  const tags = snapshot?.tags ?? [];
  const cookies = snapshot?.cookies ?? [];
  const requests = snapshot?.requests ?? [];

  const vendors = useMemo(() => {
    const set = new Set<string>();
    tags.forEach((tag) => set.add(tag.vendor));
    cookies.forEach((cookie) => set.add(cookie.vendor));
    requests.forEach((request) => set.add(request.vendor));
    set.delete("");
    return [...set].sort();
  }, [tags, cookies, requests]);

  const categories = useMemo(() => [...new Set(tags.map((tag) => String(tag.category)))].sort(), [tags]);

  const filteredTags = useMemo(() => {
    const q = query.trim().toLowerCase();
    return tags.filter((tag) => {
      if (vendor !== "all" && tag.vendor !== vendor) return false;
      if (category !== "all" && String(tag.category) !== category) return false;
      if (!q) return true;
      return [tag.name, tag.vendor, tag.category, tag.evidence].some((value) =>
        String(value ?? "").toLowerCase().includes(q),
      );
    });
  }, [category, query, tags, vendor]);

  const filteredCookies = useMemo(() => {
    const q = query.trim().toLowerCase();
    return cookies.filter((cookie) => {
      if (vendor !== "all" && cookie.vendor !== vendor) return false;
      if (party === "first" && !cookie.firstParty) return false;
      if (party === "third" && cookie.firstParty) return false;
      if (!q) return true;
      return [cookie.name, cookie.domain, cookie.vendor, cookie.purpose].some((value) =>
        String(value ?? "").toLowerCase().includes(q),
      );
    });
  }, [cookies, party, query, vendor]);

  const filteredRequests = useMemo(() => {
    const q = query.trim().toLowerCase();
    return requests.filter((request) => {
      if (vendor !== "all" && request.vendor !== vendor) return false;
      if (party === "first" && request.thirdParty) return false;
      if (party === "third" && !request.thirdParty) return false;
      if (!q) return true;
      return [request.url, request.host, request.vendor, request.resourceType].some((value) =>
        String(value ?? "").toLowerCase().includes(q),
      );
    });
  }, [party, query, requests, vendor]);

  const thirdPartyCount = requests.filter((request) => request.thirdParty).length;
  const lowConfidence = tags.filter((tag) => tag.confidence < 0.7).length;
  const rowCount = tab === "tags" ? filteredTags.length : tab === "cookies" ? filteredCookies.length : filteredRequests.length;

  function exportCsv() {
    if (tab === "tags") {
      downloadCsv(
        "pageguard-tags.csv",
        ["vendor", "name", "category", "confidence", "pages", "first_seen", "last_seen"],
        filteredTags.map((tag) => [tag.vendor, tag.name, tag.category, tag.confidence, tag.pages, tag.firstSeen, tag.lastSeen]),
      );
    } else if (tab === "cookies") {
      downloadCsv(
        "pageguard-cookies.csv",
        ["name", "domain", "vendor", "purpose", "first_party", "secure", "http_only", "samesite", "pages"],
        filteredCookies.map((cookie) => [
          cookie.name, cookie.domain, cookie.vendor, cookie.purpose, cookie.firstParty, cookie.secure, cookie.httpOnly, cookie.sameSite, cookie.pages,
        ]),
      );
    } else {
      downloadCsv(
        "pageguard-requests.csv",
        ["method", "url", "host", "vendor", "type", "third_party", "status", "count", "pages"],
        filteredRequests.map((request) => [
          request.method, request.url, request.host, request.vendor, request.resourceType, request.thirdParty, request.status, request.count, request.pages,
        ]),
      );
    }
  }

  const selectedTitle = selected
    ? selected.kind === "tag"
      ? selected.item.name
      : selected.kind === "cookie"
        ? selected.item.name
        : selected.item.host
    : "";
  const selectedSubtitle = selected
    ? selected.kind === "tag"
      ? `${selected.item.vendor} / ${selected.item.category}`
      : selected.kind === "cookie"
        ? `${selected.item.vendor} / ${selected.item.domain}`
        : selected.item.url
    : undefined;

  const vendorOptions = [{ value: "all", label: "All vendors" }, ...vendors.map((name) => ({ value: name, label: name }))];
  const categoryOptions = [{ value: "all", label: "All categories" }, ...categories.map((name) => ({ value: name, label: name }))];
  const partyOptions = [
    { value: "all", label: "All parties" },
    { value: "first", label: "First party" },
    { value: "third", label: "Third party" },
  ];

  if (!project && !loadingProjects) {
    return (
      <EmptyState
        icon="database"
        title="No project selected"
        description="Create a project and run an audit to populate the tag and cookie inventory."
      />
    );
  }

  return (
    <div className="space-y-5">
      <PageHeader
        eyebrow="Data inventory"
        title="Tag & Cookie Inventory"
        subtitle={`Tags, cookies and third-party requests aggregated from the latest audit run${snapshot?.pagesScanned ? ` across ${formatNumber(snapshot.pagesScanned)} pages` : ""}.`}
        icon="database"
        actions={
          <>
            <button className="btn" onClick={() => void load()} disabled={loading}>
              <Icon name="refresh" className="h-3.5 w-3.5" />
              Refresh
            </button>
            <button className="btn-primary" onClick={exportCsv} disabled={rowCount === 0}>
              <Icon name="download" className="h-3.5 w-3.5" />
              Export CSV
            </button>
          </>
        }
      />

      <ErrorBanner message={error} />

      {loading ? (
        <Loading label="Loading inventory" />
      ) : !snapshot ? (
        <EmptyState
          icon="database"
          title="No inventory yet"
          description="Run an audit for this project to aggregate tags, cookies and network requests."
        />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
            <KpiCard label="Tags detected" value={formatNumber(tags.length)} hint={`across ${formatNumber(snapshot.pagesScanned)} pages`} tone="indigo" icon="tag" />
            <KpiCard label="Cookies observed" value={formatNumber(cookies.length)} hint="first and third party" tone="emerald" icon="cookie" />
            <KpiCard label="Third-party requests" value={formatNumber(thirdPartyCount)} hint={`${formatNumber(requests.length)} aggregated requests`} tone="rose" icon="activity" />
            <KpiCard label="Low-confidence tags" value={formatNumber(lowConfidence)} hint="needs analyst review" tone="amber" icon="alert" />
          </div>

          <Card>
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 px-4 py-3">
              <Tabs
                value={tab}
                onChange={(id) => setTab(id as Tab)}
                tabs={[
                  { id: "tags", label: "Tags", count: tags.length },
                  { id: "cookies", label: "Cookies", count: cookies.length },
                  { id: "requests", label: "Network requests", count: requests.length },
                ]}
              />
              <div className="flex flex-wrap items-center gap-2">
                <SearchInput value={query} onChange={setQuery} placeholder="Search inventory" className="w-full sm:w-56" />
                <Select value={vendor} onChange={setVendor} options={vendorOptions} label="Vendor" />
                {tab === "tags" ? (
                  <Select value={category} onChange={setCategory} options={categoryOptions} label="Category" />
                ) : (
                  <Select value={party} onChange={(value) => setParty(value as Party)} options={partyOptions} label="Party" />
                )}
              </div>
            </div>

            {rowCount === 0 ? (
              <EmptyState icon="search" title="No matching inventory" description="Adjust the search or filters to widen the result set." />
            ) : (
              <div className="overflow-x-auto">
                {tab === "tags" && (
                  <table className="w-full min-w-[760px] text-xs" aria-label="Detected tags">
                    <thead className="table-head">
                      <tr>
                        <th scope="col" className="px-4 py-2.5">Vendor / tag</th>
                        <th scope="col" className="px-4 py-2.5">Category</th>
                        <th scope="col" className="w-44 px-4 py-2.5">Confidence</th>
                        <th scope="col" className="px-4 py-2.5">Pages</th>
                        <th scope="col" className="px-4 py-2.5">Last seen</th>
                        <th scope="col" className="px-4 py-2.5"><span className="sr-only">Open</span></th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredTags.map((tag) => (
                        <tr key={tag.id} className="table-row cursor-pointer" onClick={() => setSelected({ kind: "tag", item: tag })}>
                          <td className="px-4 py-3">
                            <div className="font-semibold text-slate-200">{tag.vendor}</div>
                            <div className="mt-0.5 text-[11px] text-slate-500">{tag.name}</div>
                          </td>
                          <td className="px-4 py-3"><Pill tone={categoryTone(tag.category)}>{String(tag.category).replaceAll("_", " ")}</Pill></td>
                          <td className="px-4 py-3">
                            <div className="flex items-center gap-2">
                              <ProgressBar value={tag.confidence * 100} tone={tag.confidence < 0.7 ? "amber" : "emerald"} />
                              <span className="mono w-9 shrink-0 text-right text-[11px] text-slate-400">{formatPercent(tag.confidence * 100, 0)}</span>
                            </div>
                          </td>
                          <td className="px-4 py-3 text-slate-400">{formatNumber(tag.pages)}</td>
                          <td className="mono px-4 py-3 text-[11px] text-slate-500">{formatDate(tag.lastSeen)}</td>
                          <td className="px-4 py-3 text-right text-slate-600"><Icon name="chevron-right" className="ml-auto h-4 w-4" /></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}

                {tab === "cookies" && (
                  <table className="w-full min-w-[820px] text-xs" aria-label="Observed cookies">
                    <thead className="table-head">
                      <tr>
                        <th scope="col" className="px-4 py-2.5">Cookie</th>
                        <th scope="col" className="px-4 py-2.5">Vendor</th>
                        <th scope="col" className="px-4 py-2.5">Purpose</th>
                        <th scope="col" className="px-4 py-2.5">Party</th>
                        <th scope="col" className="px-4 py-2.5">Attributes</th>
                        <th scope="col" className="px-4 py-2.5">Pages</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredCookies.map((cookie) => (
                        <tr key={`${cookie.name}-${cookie.domain}`} className="table-row cursor-pointer" onClick={() => setSelected({ kind: "cookie", item: cookie })}>
                          <td className="px-4 py-3">
                            <div className="mono font-semibold text-slate-200">{cookie.name}</div>
                            <div className="mt-0.5 font-mono text-[11px] text-slate-500">{cookie.domain}</div>
                          </td>
                          <td className="px-4 py-3 text-slate-300">{cookie.vendor || "-"}</td>
                          <td className="px-4 py-3"><Pill tone={categoryTone(cookie.purpose)}>{cookie.purpose || "unknown"}</Pill></td>
                          <td className="px-4 py-3"><Pill tone={cookie.firstParty ? "sky" : "rose"}>{cookie.firstParty ? "First" : "Third"}</Pill></td>
                          <td className="px-4 py-3 text-[11px] text-slate-500">
                            {[cookie.secure ? "Secure" : null, cookie.httpOnly ? "HttpOnly" : null, cookie.sameSite, cookie.session ? "Session" : "Persistent"].filter(Boolean).join(" / ")}
                          </td>
                          <td className="px-4 py-3 text-slate-400">{formatNumber(cookie.pages)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}

                {tab === "requests" && (
                  <table className="w-full min-w-[900px] text-xs" aria-label="Network requests">
                    <thead className="table-head">
                      <tr>
                        <th scope="col" className="px-4 py-2.5">Host</th>
                        <th scope="col" className="px-4 py-2.5">Vendor</th>
                        <th scope="col" className="px-4 py-2.5">Type</th>
                        <th scope="col" className="px-4 py-2.5">Party</th>
                        <th scope="col" className="px-4 py-2.5">Status</th>
                        <th scope="col" className="px-4 py-2.5">Size</th>
                        <th scope="col" className="px-4 py-2.5">Count</th>
                        <th scope="col" className="px-4 py-2.5">Pages</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredRequests.map((request) => (
                        <tr key={`${request.method}-${request.url}`} className="table-row cursor-pointer" onClick={() => setSelected({ kind: "request", item: request })}>
                          <td className="px-4 py-3">
                            <div className="font-semibold text-slate-200">{request.host}</div>
                            <div className="mt-0.5 max-w-[280px] truncate font-mono text-[11px] text-slate-500">{request.url}</div>
                          </td>
                          <td className="px-4 py-3 text-slate-300">{request.vendor || "-"}</td>
                          <td className="px-4 py-3"><span className="mono text-[11px] text-slate-400">{request.resourceType}</span></td>
                          <td className="px-4 py-3"><Pill tone={request.thirdParty ? "rose" : "sky"}>{request.thirdParty ? "Third" : "First"}</Pill></td>
                          <td className="px-4 py-3"><span className={`mono text-[11px] ${request.status >= 400 ? "text-rose-300" : "text-emerald-300"}`}>{request.status}</span></td>
                          <td className="px-4 py-3 text-slate-400">{bytes(request.sizeBytes)}</td>
                          <td className="px-4 py-3 text-slate-400">{formatNumber(request.count)}</td>
                          <td className="px-4 py-3 text-slate-400">{formatNumber(request.pages)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            )}
          </Card>
        </>
      )}

      <Drawer open={!!selected} onClose={() => setSelected(null)} title={selectedTitle} subtitle={selectedSubtitle}>
        {selected && (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-3">
              {selected.kind === "tag" && (
                <>
                  <Detail label="Vendor">{selected.item.vendor}</Detail>
                  <Detail label="Category">{String(selected.item.category).replaceAll("_", " ")}</Detail>
                  <Detail label="Purpose">{selected.item.purpose || "-"}</Detail>
                  <Detail label="Version">{selected.item.version || "-"}</Detail>
                  <Detail label="Confidence">{formatPercent(selected.item.confidence * 100, 0)}</Detail>
                  <Detail label="Pages">{formatNumber(selected.item.pages)}</Detail>
                  <Detail label="First seen">{formatDate(selected.item.firstSeen)}</Detail>
                  <Detail label="Last seen">{formatDate(selected.item.lastSeen)}</Detail>
                </>
              )}
              {selected.kind === "cookie" && (
                <>
                  <Detail label="Vendor">{selected.item.vendor || "-"}</Detail>
                  <Detail label="Purpose">{selected.item.purpose || "-"}</Detail>
                  <Detail label="Domain">{selected.item.domain}</Detail>
                  <Detail label="Path">{selected.item.path}</Detail>
                  <Detail label="Party">{selected.item.firstParty ? "First party" : "Third party"}</Detail>
                  <Detail label="Lifetime">{selected.item.session ? "Session" : "Persistent"}</Detail>
                  <Detail label="SameSite">{selected.item.sameSite}</Detail>
                  <Detail label="Pages">{formatNumber(selected.item.pages)}</Detail>
                </>
              )}
              {selected.kind === "request" && (
                <>
                  <Detail label="Vendor">{selected.item.vendor || "-"}</Detail>
                  <Detail label="Host">{selected.item.host}</Detail>
                  <Detail label="Method / type">{`${selected.item.method} / ${selected.item.resourceType}`}</Detail>
                  <Detail label="Status">{selected.item.status}</Detail>
                  <Detail label="Party">{selected.item.thirdParty ? "Third party" : "First party"}</Detail>
                  <Detail label="Initiator">{selected.item.initiator || "-"}</Detail>
                  <Detail label="Avg duration">{selected.item.durationMs} ms</Detail>
                  <Detail label="Avg size">{bytes(selected.item.sizeBytes)}</Detail>
                  <Detail label="Occurrences">{formatNumber(selected.item.count)}</Detail>
                  <Detail label="Pages">{formatNumber(selected.item.pages)}</Detail>
                </>
              )}
            </div>

            <div>
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-white">Raw evidence</h3>
                <button className="btn" onClick={() => void navigator.clipboard?.writeText(JSON.stringify(selected.item, null, 2))}>
                  <Icon name="copy" className="h-3.5 w-3.5" />
                  Copy JSON
                </button>
              </div>
              <pre className="mono mt-2 max-h-80 overflow-auto rounded-xl border border-slate-800 bg-slate-950/70 p-3 text-[11px] leading-relaxed text-slate-300">
                {JSON.stringify(selected.item, null, 2)}
              </pre>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
}
