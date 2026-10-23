"use client";

import { useMemo, useState, type ReactNode } from "react";
import { DemoBanner, DemoPageHeader } from "../../components/demo";
import { Icon } from "../../components/icons";
import {
  Card,
  Drawer,
  EmptyState,
  KpiCard,
  Pill,
  ProgressBar,
  SearchInput,
  Select,
  Tabs,
} from "../../components/ui";
import {
  DEMO_COOKIES,
  DEMO_PROJECT,
  DEMO_REQUESTS,
  DEMO_RUN,
  DEMO_TAGS,
  type InventoryCookie,
  type InventoryRequest,
  type InventoryTag,
} from "../../lib/demo";
import { bytes, formatNumber, formatPercent } from "../../lib/format";

type Tab = "tags" | "cookies" | "requests";
type Party = "all" | "first" | "third";

type Selection =
  | { kind: "tag"; item: InventoryTag }
  | { kind: "cookie"; item: InventoryCookie }
  | { kind: "request"; item: InventoryRequest };

const CATEGORY_TONES: Record<string, "indigo" | "emerald" | "amber" | "rose" | "sky" | "slate"> = {
  Analytics: "emerald",
  Advertising: "rose",
  Consent: "indigo",
  Behavior: "amber",
  "Tag manager": "sky",
};

function categoryTone(category: string) {
  return CATEGORY_TONES[category] ?? "slate";
}

function Attribute({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex items-center rounded border border-slate-700 bg-slate-800/60 px-1.5 py-0.5 text-[10px] text-slate-400">
      {children}
    </span>
  );
}

function Detail({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
      <p className="text-[10px] font-semibold uppercase tracking-wider text-slate-500">{label}</p>
      <div className="mt-1 break-words text-xs text-slate-200">{children}</div>
    </div>
  );
}

export default function InventoryPage() {
  const [tab, setTab] = useState<Tab>("tags");
  const [query, setQuery] = useState("");
  const [vendor, setVendor] = useState("all");
  const [category, setCategory] = useState("all");
  const [party, setParty] = useState<Party>("all");
  const [selected, setSelected] = useState<Selection | null>(null);

  const vendors = useMemo(() => {
    const set = new Set<string>();
    DEMO_TAGS.forEach((tag) => set.add(tag.vendor));
    DEMO_COOKIES.forEach((cookie) => set.add(cookie.vendor));
    DEMO_REQUESTS.forEach((request) => set.add(request.vendor));
    return [...set].sort();
  }, []);

  const categories = useMemo(() => [...new Set(DEMO_TAGS.map((tag) => tag.category))].sort(), []);

  const tags = useMemo(() => {
    const q = query.trim().toLowerCase();
    return DEMO_TAGS.filter((tag) => {
      if (vendor !== "all" && tag.vendor !== vendor) return false;
      if (category !== "all" && tag.category !== category) return false;
      if (!q) return true;
      return [tag.name, tag.vendor, tag.category, tag.evidence].some((value) =>
        String(value ?? "").toLowerCase().includes(q),
      );
    });
  }, [category, query, vendor]);

  const cookies = useMemo(() => {
    const q = query.trim().toLowerCase();
    return DEMO_COOKIES.filter((cookie) => {
      if (vendor !== "all" && cookie.vendor !== vendor) return false;
      if (party === "first" && !cookie.firstParty) return false;
      if (party === "third" && cookie.firstParty) return false;
      if (!q) return true;
      return [cookie.name, cookie.domain, cookie.vendor, cookie.purpose].some((value) =>
        String(value ?? "").toLowerCase().includes(q),
      );
    });
  }, [party, query, vendor]);

  const requests = useMemo(() => {
    const q = query.trim().toLowerCase();
    return DEMO_REQUESTS.filter((request) => {
      if (vendor !== "all" && request.vendor !== vendor) return false;
      if (party === "first" && request.thirdParty) return false;
      if (party === "third" && !request.thirdParty) return false;
      if (!q) return true;
      return [request.url, request.host, request.vendor, request.resourceType].some((value) =>
        String(value ?? "").toLowerCase().includes(q),
      );
    });
  }, [party, query, vendor]);

  const thirdPartyCount = DEMO_REQUESTS.filter((request) => request.thirdParty).length;
  const lowConfidence = DEMO_TAGS.filter((tag) => tag.confidence < 0.7).length;

  const vendorOptions = [
    { value: "all", label: "All vendors" },
    ...vendors.map((name) => ({ value: name, label: name })),
  ];
  const categoryOptions = [
    { value: "all", label: "All categories" },
    ...categories.map((name) => ({ value: name, label: name })),
  ];
  const partyOptions = [
    { value: "all", label: "All parties" },
    { value: "first", label: "First party" },
    { value: "third", label: "Third party" },
  ];

  const selectedTitle = selected
    ? selected.kind === "tag"
      ? selected.item.name
      : selected.kind === "cookie"
        ? selected.item.name
        : selected.item.host
    : "";
  const selectedSubtitle = selected
    ? selected.kind === "tag"
      ? `${selected.item.vendor} · ${selected.item.category}`
      : selected.kind === "cookie"
        ? `${selected.item.vendor} · ${selected.item.domain}`
        : selected.item.url
    : undefined;

  const rowCount = tab === "tags" ? tags.length : tab === "cookies" ? cookies.length : requests.length;

  return (
    <div className="space-y-5">
      <DemoPageHeader
        eyebrow="Data inventory"
        title="Tag & Cookie Inventory"
        subtitle={`Every tag, cookie and third-party request captured from ${DEMO_PROJECT.name} in the latest audit run (${DEMO_RUN.id}).`}
        icon="database"
        actions={
          <>
            <button className="btn" title="Demo only">
              <Icon name="download" className="h-3.5 w-3.5" />
              Export CSV
            </button>
            <button className="btn-primary" title="Demo only">
              <Icon name="refresh" className="h-3.5 w-3.5" />
              Compare runs
            </button>
          </>
        }
      />

      <DemoBanner />

      <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        <KpiCard label="Tags detected" value={formatNumber(DEMO_TAGS.length)} hint={`across ${DEMO_RUN.pages} audited pages`} tone="indigo" icon="tag" />
        <KpiCard label="Cookies observed" value={formatNumber(DEMO_COOKIES.length)} hint="first and third party" tone="emerald" icon="cookie" />
        <KpiCard label="Third-party requests" value={formatNumber(thirdPartyCount)} hint={`${DEMO_REQUESTS.length} hosts in total`} tone="rose" icon="activity" />
        <KpiCard label="Low-confidence tags" value={formatNumber(lowConfidence)} hint="needs analyst review" tone="amber" icon="alert" />
      </div>

      <Card>
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 px-4 py-3">
          <Tabs
            value={tab}
            onChange={(id) => setTab(id as Tab)}
            tabs={[
              { id: "tags", label: "Tags", count: DEMO_TAGS.length },
              { id: "cookies", label: "Cookies", count: DEMO_COOKIES.length },
              { id: "requests", label: "Network requests", count: DEMO_REQUESTS.length },
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
          <EmptyState
            icon="search"
            title="No matching inventory"
            description="Adjust the search or filters to widen the result set."
            action={
              <button
                className="btn"
                onClick={() => {
                  setQuery("");
                  setVendor("all");
                  setCategory("all");
                  setParty("all");
                }}
              >
                Reset filters
              </button>
            }
          />
        ) : (
          <div className="overflow-x-auto">
            {tab === "tags" && (
              <table className="w-full min-w-[760px] text-xs" aria-label="Detected tags">
                <thead className="table-head">
                  <tr>
                    <th scope="col" className="px-4 py-2.5">Vendor / tag</th>
                    <th scope="col" className="px-4 py-2.5">Category</th>
                    <th scope="col" className="w-40 px-4 py-2.5">Confidence</th>
                    <th scope="col" className="px-4 py-2.5">Pages</th>
                    <th scope="col" className="px-4 py-2.5">First seen</th>
                    <th scope="col" className="px-4 py-2.5"><span className="sr-only">Open</span></th>
                  </tr>
                </thead>
                <tbody>
                  {tags.map((tag) => (
                    <tr
                      key={tag.id}
                      className="table-row cursor-pointer"
                      onClick={() => setSelected({ kind: "tag", item: tag })}
                    >
                      <td className="px-4 py-3">
                        <div className="font-semibold text-slate-200">{tag.vendor}</div>
                        <div className="mt-0.5 text-[11px] text-slate-500">{tag.name}</div>
                      </td>
                      <td className="px-4 py-3"><Pill tone={categoryTone(tag.category)}>{tag.category}</Pill></td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <ProgressBar value={tag.confidence * 100} tone={tag.confidence < 0.7 ? "amber" : "emerald"} />
                          <span className="mono w-9 shrink-0 text-right text-[11px] text-slate-400">
                            {formatPercent(tag.confidence * 100, 0)}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-slate-400">{formatNumber(tag.pages)}</td>
                      <td className="mono px-4 py-3 text-[11px] text-slate-500">{tag.firstSeen}</td>
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
                    <th scope="col" className="px-4 py-2.5"><span className="sr-only">Open</span></th>
                  </tr>
                </thead>
                <tbody>
                  {cookies.map((cookie) => (
                    <tr
                      key={cookie.name}
                      className="table-row cursor-pointer"
                      onClick={() => setSelected({ kind: "cookie", item: cookie })}
                    >
                      <td className="px-4 py-3">
                        <div className="mono font-semibold text-slate-200">{cookie.name}</div>
                        <div className="mt-0.5 font-mono text-[11px] text-slate-500">{cookie.domain}</div>
                      </td>
                      <td className="px-4 py-3 text-slate-300">{cookie.vendor}</td>
                      <td className="px-4 py-3"><Pill tone={categoryTone(cookie.purpose)}>{cookie.purpose}</Pill></td>
                      <td className="px-4 py-3">
                        <Pill tone={cookie.firstParty ? "sky" : "rose"}>{cookie.firstParty ? "First" : "Third"}</Pill>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-1">
                          {cookie.secure && <Attribute>Secure</Attribute>}
                          {cookie.httpOnly && <Attribute>HttpOnly</Attribute>}
                          <Attribute>{cookie.sameSite}</Attribute>
                          <Attribute>{cookie.session ? "Session" : "Persistent"}</Attribute>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-slate-400">{formatNumber(cookie.pages)}</td>
                      <td className="px-4 py-3 text-right text-slate-600"><Icon name="chevron-right" className="ml-auto h-4 w-4" /></td>
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
                    <th scope="col" className="px-4 py-2.5">Avg ms</th>
                    <th scope="col" className="px-4 py-2.5">Count</th>
                    <th scope="col" className="px-4 py-2.5"><span className="sr-only">Open</span></th>
                  </tr>
                </thead>
                <tbody>
                  {requests.map((request) => (
                    <tr
                      key={request.url}
                      className="table-row cursor-pointer"
                      onClick={() => setSelected({ kind: "request", item: request })}
                    >
                      <td className="px-4 py-3">
                        <div className="font-semibold text-slate-200">{request.host}</div>
                        <div className="mt-0.5 max-w-[280px] truncate font-mono text-[11px] text-slate-500">{request.url}</div>
                      </td>
                      <td className="px-4 py-3 text-slate-300">{request.vendor}</td>
                      <td className="px-4 py-3"><span className="mono text-[11px] text-slate-400">{request.resourceType}</span></td>
                      <td className="px-4 py-3">
                        <Pill tone={request.thirdParty ? "rose" : "sky"}>{request.thirdParty ? "Third" : "First"}</Pill>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`mono text-[11px] ${request.status >= 400 ? "text-rose-300" : "text-emerald-300"}`}>{request.status}</span>
                      </td>
                      <td className="px-4 py-3 text-slate-400">{bytes(request.sizeBytes)}</td>
                      <td className="px-4 py-3 text-slate-400">{request.durationMs}</td>
                      <td className="px-4 py-3 text-slate-400">{formatNumber(request.count)}</td>
                      <td className="px-4 py-3 text-right text-slate-600"><Icon name="chevron-right" className="ml-auto h-4 w-4" /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}
      </Card>

      <Drawer open={!!selected} onClose={() => setSelected(null)} title={selectedTitle} subtitle={selectedSubtitle}>
        {selected && (
          <div className="space-y-5">
            {selected.kind === "tag" && (
              <div className="grid grid-cols-2 gap-3">
                <Detail label="Vendor">{selected.item.vendor}</Detail>
                <Detail label="Category"><Pill tone={categoryTone(selected.item.category)}>{selected.item.category}</Pill></Detail>
                <Detail label="Purpose">{selected.item.purpose}</Detail>
                <Detail label="Version">{selected.item.version ?? "—"}</Detail>
                <Detail label="Confidence">{formatPercent(selected.item.confidence * 100, 0)}</Detail>
                <Detail label="Pages">{formatNumber(selected.item.pages)}</Detail>
                <Detail label="First seen">{selected.item.firstSeen}</Detail>
                <Detail label="Last seen">{selected.item.lastSeen}</Detail>
              </div>
            )}

            {selected.kind === "cookie" && (
              <div className="grid grid-cols-2 gap-3">
                <Detail label="Vendor">{selected.item.vendor}</Detail>
                <Detail label="Purpose"><Pill tone={categoryTone(selected.item.purpose)}>{selected.item.purpose}</Pill></Detail>
                <Detail label="Domain">{selected.item.domain}</Detail>
                <Detail label="Path">{selected.item.path}</Detail>
                <Detail label="Party">{selected.item.firstParty ? "First party" : "Third party"}</Detail>
                <Detail label="Lifetime">{selected.item.session ? "Session" : "Persistent"}</Detail>
                <Detail label="Secure / HttpOnly">{`${selected.item.secure ? "Secure" : "Not secure"} · ${selected.item.httpOnly ? "HttpOnly" : "Script readable"}`}</Detail>
                <Detail label="SameSite">{selected.item.sameSite}</Detail>
                <Detail label="Pages">{formatNumber(selected.item.pages)}</Detail>
              </div>
            )}

            {selected.kind === "request" && (
              <div className="grid grid-cols-2 gap-3">
                <Detail label="Vendor">{selected.item.vendor}</Detail>
                <Detail label="Host">{selected.item.host}</Detail>
                <Detail label="Method / type">{`${selected.item.method} · ${selected.item.resourceType}`}</Detail>
                <Detail label="Status">{selected.item.status}</Detail>
                <Detail label="Party">{selected.item.thirdParty ? "Third party" : "First party"}</Detail>
                <Detail label="Initiator">{selected.item.initiator ?? "—"}</Detail>
                <Detail label="Avg duration">{selected.item.durationMs} ms</Detail>
                <Detail label="Avg size">{bytes(selected.item.sizeBytes)}</Detail>
                <Detail label="Occurrences">{formatNumber(selected.item.count)}</Detail>
                <Detail label="Pages">{formatNumber(selected.item.pages)}</Detail>
              </div>
            )}

            <div>
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-white">Raw evidence</h3>
                <button
                  className="btn"
                  onClick={() => {
                    void navigator.clipboard?.writeText(JSON.stringify(selected.item, null, 2));
                  }}
                >
                  <Icon name="copy" className="h-3.5 w-3.5" />
                  Copy JSON
                </button>
              </div>
              <pre className="mono mt-2 max-h-80 overflow-auto rounded-xl border border-slate-800 bg-slate-950/70 p-3 text-[11px] leading-relaxed text-slate-300">
                {JSON.stringify(selected.item, null, 2)}
              </pre>
            </div>

            <div>
              <h3 className="text-xs font-semibold text-white">Related pages</h3>
              <ul className="mt-2 space-y-1.5 text-[11px] text-slate-400">
                <li className="rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2 font-mono">/checkout</li>
                <li className="rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2 font-mono">/products/aurora-jacket</li>
                <li className="rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2 font-mono">/campaigns/autumn-sale</li>
                <li className="rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2 font-mono">+ {formatNumber((selected.kind === "tag" ? selected.item.pages : selected.item.pages) - 3)} more</li>
              </ul>
            </div>

            <p className="rounded-xl border border-slate-800 bg-slate-950/40 px-3 py-2 text-[11px] text-slate-500">
              Demo evidence only. In production this panel is populated from the run checkpoint APIs.
            </p>
          </div>
        )}
      </Drawer>
    </div>
  );
}
