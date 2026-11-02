"use client";

import { useCallback, useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { Icon, type IconName } from "./icons";

const TOUR_STORAGE_KEY = "pageguard.tour.seen";
const HIGHLIGHT_PAD = 7;
const TOOLTIP_GAP = 12;
const VIEW_MARGIN = 16;

export type TourStep = {
  id: string;
  title: string;
  body: string;
  icon: IconName;
  selectors?: string[];
  cta?: string;
};

export const TOUR_STEPS: TourStep[] = [
  {
    id: "welcome",
    icon: "spark",
    title: "Welcome to OpenTag Validator",
    body: "This console puts your tags, beacons and consent flows under governance — crawl any site, capture pre-consent evidence and validate it against your rules. Take a quick tour to find your way around.",
  },
  {
    id: "navigation",
    icon: "list",
    selectors: ['[data-tour="sidebar"]', '[data-tour="mobile-nav"]'],
    title: "Everything has a home",
    body: "The sidebar groups the console into Governance tools you use daily — Audits, Inspector, Rules and Consent — plus Preview areas such as Inventory, Schedules and Alerts.",
  },
  {
    id: "project-select",
    icon: "database",
    selectors: ['[data-tour="project-select"]'],
    title: "Work is scoped per project",
    body: "Scans, governance rules and reports are organized per project. Switch between them here — the active project is remembered across visits.",
  },
  {
    id: "project-create",
    icon: "plus",
    selectors: ['[data-tour="project-btn"]'],
    title: "Create more workspaces",
    body: "Audit another site, brand or client? Add a named project here so its scans and rules stay separate.",
  },
  {
    id: "run-audit",
    icon: "play",
    selectors: ['[data-tour="run-audit"]'],
    title: "Kick off a crawl",
    body: "Run an audit to crawl your site, capture beacons before consent and evaluate every configured rule. Findings and scores land on the Overview dashboard.",
  },
  {
    id: "inflight",
    icon: "activity",
    selectors: ['[data-tour="inflight-chip"]'],
    title: "Watch analysis live",
    body: "This badge counts audits running right now and pulses while evidence is being captured.",
  },
  {
    id: "finish",
    icon: "spark",
    title: "You're all set",
    body: "Pick or create a project, then run your first audit. The Overview dashboard fills up with health scores, consent results and rule findings. Replay this tour anytime from the help button.",
    cta: "Run your first audit",
  },
];

type Box = { top: number; left: number; width: number; height: number };
type Rect = Box & { right: number; bottom: number };
type Anchor = { rect: Rect };

export function hasSeenTour(): boolean {
  try {
    return window.localStorage.getItem(TOUR_STORAGE_KEY) === "1";
  } catch {
    return true;
  }
}

export function rememberTour(): void {
  try {
    window.localStorage.setItem(TOUR_STORAGE_KEY, "1");
  } catch {
    return;
  }
}

function isVisible(el: HTMLElement | null): el is HTMLElement {
  if (!el) return false;
  const rect = el.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0) return false;
  const style = window.getComputedStyle(el);
  return style.visibility !== "hidden" && style.display !== "none";
}

function resolveTarget(step: TourStep): HTMLElement | null {
  if (!step.selectors?.length) return null;
  for (const selector of step.selectors) {
    let el: HTMLElement | null = null;
    try {
      el = document.querySelector<HTMLElement>(selector);
    } catch {
      el = null;
    }
    if (isVisible(el)) return el;
  }
  return null;
}

function clamp(value: number, min: number, max: number): number {
  if (max <= min) return min;
  return Math.min(Math.max(value, min), max);
}

function centerLeft(rect: Rect, width: number): number {
  return rect.left + rect.width / 2 - width / 2;
}

export default function SpotlightTour({
  open,
  steps,
  onClose,
  onCta,
}: {
  open: boolean;
  steps: TourStep[];
  onClose: () => void;
  onCta?: () => void;
}) {
  const [visibleSteps, setVisibleSteps] = useState<TourStep[]>([]);
  const [index, setIndex] = useState(0);
  const [anchor, setAnchor] = useState<Anchor | null>(null);
  const [tip, setTip] = useState<{ top: number; left: number } | null>(null);
  const tipRef = useRef<HTMLDivElement | null>(null);
  const primaryRef = useRef<HTMLButtonElement | null>(null);
  const titleId = useId();

  const collectVisible = useCallback(() => {
    const list = steps.filter((step) => !step.selectors?.length || resolveTarget(step) !== null);
    setVisibleSteps(list);
  }, [steps]);

  const step = visibleSteps[Math.min(index, Math.max(0, visibleSteps.length - 1))];
  const last = visibleSteps.length - 1;
  const centered = !step?.selectors?.length;
  const positioned = centered || tip !== null;

  useEffect(() => {
    if (!open) return;
    setIndex(0);
    collectVisible();
  }, [open, collectVisible]);

  useEffect(() => {
    if (!open) return;
    let frame = 0;
    const handler = () => {
      if (frame) return;
      frame = window.requestAnimationFrame(() => {
        frame = 0;
        collectVisible();
      });
    };
    window.addEventListener("resize", handler);
    return () => {
      window.removeEventListener("resize", handler);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, [open, collectVisible]);

  useEffect(() => {
    if (visibleSteps.length && index > last) setIndex(last);
  }, [index, last, visibleSteps.length]);

  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prev;
    };
  }, [open]);

  useEffect(() => {
    if (!open || !step) return;
    let frame = 0;
    const measure = () => {
      if (centered) {
        setAnchor(null);
        setTip(null);
        return;
      }
      const el = resolveTarget(step);
      if (!el) {
        setAnchor(null);
        setTip(null);
        return;
      }
      const rect = el.getBoundingClientRect();
      setAnchor({ rect });
      setTip(null);
    };
    measure();
    const onMove = () => {
      if (frame) return;
      frame = window.requestAnimationFrame(() => {
        frame = 0;
        measure();
      });
    };
    window.addEventListener("scroll", onMove, true);
    window.addEventListener("resize", onMove);
    return () => {
      window.removeEventListener("scroll", onMove, true);
      window.removeEventListener("resize", onMove);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, [open, step, centered]);

  useLayoutEffect(() => {
    if (!open || !anchor) {
      if (!anchor) setTip(null);
      return;
    }
    const tipEl = tipRef.current;
    if (!tipEl) return;
    const size = tipEl.getBoundingClientRect();
    const view = { width: window.innerWidth, height: window.innerHeight };
    const rect = anchor.rect;
    const fitsBottom = rect.bottom + TOOLTIP_GAP + size.height + VIEW_MARGIN <= view.height;
    const fitsTop = rect.top - TOOLTIP_GAP - size.height - VIEW_MARGIN >= 0;
    const fitsRight = rect.right + TOOLTIP_GAP + size.width + VIEW_MARGIN <= view.width;
    const fitsLeft = rect.left - TOOLTIP_GAP - size.width - VIEW_MARGIN >= 0;
    let top: number;
    let left: number;
    if (fitsBottom) {
      top = rect.bottom + TOOLTIP_GAP;
      left = clamp(centerLeft(rect, size.width), VIEW_MARGIN, view.width - size.width - VIEW_MARGIN);
    } else if (fitsTop) {
      top = rect.top - TOOLTIP_GAP - size.height;
      left = clamp(centerLeft(rect, size.width), VIEW_MARGIN, view.width - size.width - VIEW_MARGIN);
    } else if (fitsRight) {
      left = rect.right + TOOLTIP_GAP;
      top = clamp(rect.top + rect.height / 2 - size.height / 2, VIEW_MARGIN, view.height - size.height - VIEW_MARGIN);
    } else if (fitsLeft) {
      left = rect.left - TOOLTIP_GAP - size.width;
      top = clamp(rect.top + rect.height / 2 - size.height / 2, VIEW_MARGIN, view.height - size.height - VIEW_MARGIN);
    } else {
      top = clamp(rect.bottom, VIEW_MARGIN, view.height - size.height - VIEW_MARGIN);
      left = clamp(centerLeft({ ...rect, left: 0, width: view.width }, size.width), VIEW_MARGIN, view.width - size.width - VIEW_MARGIN);
    }
    if (!tip || Math.abs(tip.top - top) > 0.5 || Math.abs(tip.left - left) > 0.5) {
      setTip({ top, left });
    }
  }, [open, anchor, tip]);

  const go = useCallback(
    (next: number) => {
      const bounded = Math.max(0, Math.min(next, last));
      if (bounded === index) return;
      setIndex(bounded);
    },
    [index, last],
  );

  useEffect(() => {
    if (!open || !step) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
      } else if (event.key === "ArrowRight") {
        event.preventDefault();
        if (index >= last) {
          if (step.cta && onCta) onCta();
          else onClose();
        } else {
          setIndex(index + 1);
        }
      } else if (event.key === "ArrowLeft") {
        event.preventDefault();
        go(index - 1);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, step, index, last, go, onClose, onCta]);

  useEffect(() => {
    if (!open) return;
    const id = window.setTimeout(() => primaryRef.current?.focus(), 20);
    return () => window.clearTimeout(id);
  }, [open, index, step?.id]);

  if (!open || !step) return null;

  const highlight: Box | null = anchor
    ? {
        top: anchor.rect.top - HIGHLIGHT_PAD,
        left: anchor.rect.left - HIGHLIGHT_PAD,
        width: anchor.rect.width + HIGHLIGHT_PAD * 2,
        height: anchor.rect.height + HIGHLIGHT_PAD * 2,
      }
    : null;

  const isLast = index >= last;
  const primaryLabel = isLast ? (step.cta ?? "Finish tour") : "Next";

  function primaryAction() {
    if (isLast) {
      if (step.cta && onCta) onCta();
      else onClose();
      return;
    }
    setIndex(index + 1);
  }

  return (
    <div className="fixed inset-0 z-[60]">
      <div className="fixed inset-0" onClick={onClose} aria-hidden="true" />
      {!highlight && <div className="fixed inset-0 bg-slate-950/80 backdrop-blur-sm" />}
      {highlight && (
        <>
          <div
            className="pointer-events-none fixed"
            style={{ ...highlight, boxShadow: "0 0 0 9999px rgba(2, 6, 23, 0.82)" }}
          />
          <div
            className="pointer-events-none fixed rounded-xl border-2 border-indigo-400/70 motion-safe:animate-pulse"
            style={{ ...highlight }}
          />
        </>
      )}
      <div
        ref={tipRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className={`fixed w-[340px] max-w-[calc(100vw-2rem)] rounded-2xl border border-slate-800 bg-slate-900 p-4 shadow-2xl shadow-indigo-950/40 ${
          positioned ? "opacity-100" : "opacity-0"
        } ${tip ? "transition-[top,left] duration-150" : ""} ${centered ? "left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2" : ""}`}
        style={anchor ? { top: tip?.top ?? 0, left: tip?.left ?? 0, visibility: tip ? "visible" : "hidden" } : undefined}
      >
        <div className="flex items-start gap-3">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-indigo-800/50 bg-indigo-950/50 text-indigo-300">
            <Icon name={step.icon} className="h-4 w-4" />
          </div>
          <div className="min-w-0 flex-1">
            <h2 id={titleId} className="text-sm font-semibold text-white">
              {step.title}
            </h2>
            <p className="mt-1 text-xs leading-relaxed text-slate-400">{step.body}</p>
          </div>
          <button
            onClick={onClose}
            aria-label="Skip tour"
            className="-mr-1 -mt-1 rounded-lg p-1 text-slate-500 transition hover:bg-slate-800 hover:text-white"
          >
            <Icon name="x" className="h-4 w-4" />
          </button>
        </div>
        <div className="mt-4 flex items-center justify-between gap-2 border-t border-slate-800 pt-3">
          <button
            onClick={onClose}
            className="text-[11px] font-semibold text-slate-500 transition hover:text-slate-300"
          >
            Skip tour
          </button>
          <div className="flex items-center gap-1.5" role="group" aria-label="Tour progress">
            {visibleSteps.map((item, i) => (
              <button
                key={item.id}
                aria-label={`Go to step ${i + 1}: ${item.title}`}
                onClick={() => go(i)}
                className={`h-1.5 w-1.5 rounded-full transition ${
                  i === index ? "bg-indigo-400" : i < index ? "bg-indigo-800/80" : "bg-slate-700"
                }`}
              />
            ))}
            <span className="ml-1 text-[10px] font-semibold uppercase tracking-widest text-slate-500">
              {index + 1} of {visibleSteps.length}
            </span>
          </div>
          <div className="flex items-center gap-1.5">
            {index > 0 && (
              <button onClick={() => go(index - 1)} className="btn h-8 px-2.5" aria-label="Previous step">
                <Icon name="chevron" className="h-3.5 w-3.5" />
                Back
              </button>
            )}
            <button ref={primaryRef} onClick={primaryAction} className="btn-primary h-8">
              {primaryLabel}
              {!isLast && <Icon name="chevron-right" className="h-3.5 w-3.5" />}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
