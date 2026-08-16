import { useEffect, useState, type CSSProperties } from "react";
import type { Span } from "../types";
import { usePreferences } from "../i18n";
import { copyToClipboard } from "../utils/clipboard";
import Button from "@douyinfe/semi-ui/lib/es/button";
import TabPane from "@douyinfe/semi-ui/lib/es/tabs/TabPane";
import Tabs from "@douyinfe/semi-ui/lib/es/tabs";
import Toast from "@douyinfe/semi-ui/lib/es/toast";
import Tooltip from "@douyinfe/semi-ui/lib/es/tooltip";
import IconClose from "@douyinfe/semi-icons/lib/es/icons/IconClose";
import IconCopy from "@douyinfe/semi-icons/lib/es/icons/IconCopy";
import IconExpand from "@douyinfe/semi-icons/lib/es/icons/IconExpand";
import IconTick from "@douyinfe/semi-icons/lib/es/icons/IconTick";
import "../styles/trace-detail-core.css";
import "../styles/trace-detail-waterfall.css";

type TraceDetailProps = {
  selectedID: string;
  selected: Span[];
  details: {
    start_time: string;
    end_time: string;
    span_count: number;
  } | null;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => void;
  onBack: () => void;
};

function formatDuration(nanoseconds: number) {
  const ms = nanoseconds / 1_000_000;
  if (ms < 1) return `${Math.round(nanoseconds / 1_000)}μs`;
  if (ms < 1_000) return `${ms.toFixed(2)}ms`;
  if (ms < 60_000) return `${(ms / 1_000).toFixed(2)}s`;
  return `${Math.floor(ms / 60_000)}m ${((ms % 60_000) / 1_000).toFixed(1)}s`;
}

function formatOffset(milliseconds: number) {
  if (milliseconds < 1) return `${Math.round(milliseconds * 1000)}μs`;
  if (milliseconds < 1_000) return `${Math.round(milliseconds)}ms`;
  return `${(milliseconds / 1_000).toFixed(2)}s`;
}

function Waterfall({
  spans,
  details,
  selectedSpanID,
  onSelect,
}: {
  spans: Span[];
  details: TraceDetailProps["details"];
  selectedSpanID: string;
  onSelect: (id: string) => void;
}) {
  const starts = spans.map((span) => new Date(span.start_time).getTime());
  const ends = spans.map((span) => new Date(span.start_time).getTime() + span.duration / 1_000_000);
  const traceStart = details
    ? new Date(details.start_time).getTime()
    : starts.length
      ? Math.min(...starts)
      : 0;
  const traceEnd = details
    ? new Date(details.end_time).getTime()
    : ends.length
      ? Math.max(...ends)
      : 1;
  const traceDuration = Math.max(traceEnd - traceStart, 1);
  const spanByID = new Map(spans.map((span) => [span.span_id, span]));
  const depthCache = new Map<string, number>();

  function depth(span: Span, trail = new Set<string>()): number {
    if (!span.parent_span_id || !spanByID.has(span.parent_span_id) || trail.has(span.span_id))
      return 0;
    const cached = depthCache.get(span.span_id);
    if (cached !== undefined) return cached;
    const parent = spanByID.get(span.parent_span_id);
    if (!parent) return 0;
    const value = depth(parent, new Set(trail).add(span.span_id)) + 1;
    depthCache.set(span.span_id, value);
    return value;
  }

  const ticks = Array.from({ length: 5 }, (_, index) => {
    const ratio = index / 4;
    return { ratio, label: formatOffset(traceDuration * ratio) };
  });

  return (
    <div className="waterfall-scroll">
      <div className="waterfall">
        <div className="waterfall-header">
          <span className="waterfall-label">{spans.length} spans</span>
          <div className="waterfall-axis" aria-hidden="true">
            {ticks.map((tick) => (
              <span key={tick.ratio} style={{ left: `${tick.ratio * 100}%` }}>
                {tick.label}
              </span>
            ))}
          </div>
        </div>
        {spans.map((span) => {
          const start = new Date(span.start_time).getTime();
          const left = Math.max(0, ((start - traceStart) / traceDuration) * 100);
          const width = Math.max(0.7, (span.duration / 1_000_000 / traceDuration) * 100);
          const kind = span.kind.toLowerCase() || "custom";
          return (
            <div className="waterfall-row" key={span.span_id}>
              <div className="waterfall-label" style={{ paddingLeft: `${depth(span) * 16 + 8}px` }}>
                <span className={`waterfall-kind ${kind}`} aria-hidden="true" />
                <span className="waterfall-name" title={span.name}>
                  {span.name}
                </span>
                <span className="waterfall-kind-name">{span.kind || "custom"}</span>
              </div>
              <div className="waterfall-track">
                <button
                  aria-label={`${span.name}, ${formatDuration(span.duration)}, ${span.status || "ok"}`}
                  className={`waterfall-bar ${kind} ${span.status === "error" ? "error" : ""} ${selectedSpanID === span.span_id ? "active" : ""}`}
                  style={{ left: `${left}%`, width: `${width}%` }}
                  title={`${span.name} · ${formatDuration(span.duration)}`}
                  onClick={() => onSelect(span.span_id)}
                  type="button"
                />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function kindGlyph(kind: string) {
  switch (kind.toLowerCase()) {
    case "llm":
      return "◉";
    case "tool":
      return "↗";
    case "agent":
      return "{}";
    case "retriever":
      return "⌕";
    default:
      return "•";
  }
}

function parseJSON(value: unknown) {
  if (typeof value !== "string") return value;
  try {
    return JSON.parse(value) as unknown;
  } catch {
    return value;
  }
}

function formatJSON(value: unknown) {
  return JSON.stringify(parseJSON(value ?? {}), null, 2);
}

function formatPayload(value?: string) {
  if (!value) return "—";
  const parsed = parseJSON(value);
  return typeof parsed === "string" ? parsed : JSON.stringify(parsed, null, 2);
}

function stripSystemRuntime(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(stripSystemRuntime);
  if (!value || typeof value !== "object") return value;
  const result = Object.fromEntries(
    Object.entries(value)
      .filter(([key]) => key !== "system.runtime")
      .map(([key, item]) => [key, stripSystemRuntime(item)]),
  ) as Record<string, unknown>;
  if (result.system && typeof result.system === "object" && !Array.isArray(result.system)) {
    const { runtime: _runtime, ...system } = result.system as Record<string, unknown>;
    result.system = system;
  }
  return result;
}

function findSystemRuntime(value: unknown): unknown {
  if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
  const record = value as Record<string, unknown>;
  if ("system.runtime" in record) return parseJSON(record["system.runtime"]);
  if (record.system && typeof record.system === "object" && !Array.isArray(record.system)) {
    const system = record.system as Record<string, unknown>;
    if ("runtime" in system) return system.runtime;
  }
  if ("runtime" in record) return record.runtime;
  return undefined;
}

function SpanTree({
  spans,
  selectedSpanID,
  onSelect,
}: {
  spans: Span[];
  selectedSpanID: string;
  onSelect: (id: string) => void;
}) {
  const spanIDs = new Set(spans.map((span) => span.span_id));
  const children = new Map<string, Span[]>();
  const roots: Span[] = [];

  for (const span of spans) {
    if (!span.parent_span_id || !spanIDs.has(span.parent_span_id)) {
      roots.push(span);
      continue;
    }
    const siblings = children.get(span.parent_span_id) ?? [];
    siblings.push(span);
    children.set(span.parent_span_id, siblings);
  }

  function renderSpan(span: Span, depth: number, isLast: boolean): React.ReactNode {
    const childSpans = children.get(span.span_id) ?? [];
    const kind = span.kind || "custom";
    return (
      <div
        className={`span-tree-node ${depth > 0 ? "nested-node" : "root-node"} ${childSpans.length ? "has-children" : ""} ${isLast ? "last-node" : ""}`}
        key={span.span_id}
        style={{ "--tree-depth": depth } as CSSProperties}
      >
        <button
          aria-label={span.name}
          className={`span-card ${selectedSpanID === span.span_id ? "active" : ""}`}
          onClick={() => onSelect(span.span_id)}
          title={`${span.span_id} · ${new Date(span.start_time).toLocaleTimeString()}`}
          type="button"
          tabIndex={0}
        >
          <div className="span-line">
            <span className={`span-kind-icon ${kind.toLowerCase()}`} aria-hidden="true">
              {kindGlyph(kind)}
            </span>
            <strong>{span.name}</strong>
            <span className="kind">{kind}</span>
            <span className="duration">{formatDuration(span.duration)}</span>
          </div>
        </button>
        {childSpans.map((child, index) =>
          renderSpan(child, depth + 1, index === childSpans.length - 1),
        )}
      </div>
    );
  }

  return (
    <div className="span-tree">
      {roots.map((span, index) => renderSpan(span, 0, index === roots.length - 1))}
    </div>
  );
}

export function TraceDetail({
  selectedID,
  selected,
  details,
  hasMore,
  loadingMore,
  onLoadMore,
  onBack,
}: TraceDetailProps) {
  const { t } = usePreferences();
  const [selectedSpanID, setSelectedSpanID] = useState(selected[0]?.span_id ?? "");
  const [view, setView] = useState<"tree" | "waterfall">("tree");
  const [inspectorOpen, setInspectorOpen] = useState(false);
  const [expandedPayload, setExpandedPayload] = useState<"input" | "output" | null>(null);
  const [jsonMode, setJsonMode] = useState(true);
  const [copiedTarget, setCopiedTarget] = useState<string | null>(null);

  useEffect(() => {
    setInspectorOpen(false);
  }, [selectedID]);

  useEffect(() => {
    if (!selectedSpanID && selected[0]) setSelectedSpanID(selected[0].span_id);
  }, [selected, selectedSpanID]);

  if (!selectedID) {
    return (
      <section className="detail-panel">
        <div className="detail-empty">
          <div className="orbit">✦</div>
          <h2>{t("selectTrace")}</h2>
          <p>{t("selectTraceHint")}</p>
        </div>
      </section>
    );
  }

  const activeSpan = selected.find((span) => span.span_id === selectedSpanID) ?? selected[0];
  const traceDuration = details
    ? new Date(details.end_time).getTime() - new Date(details.start_time).getTime()
    : selected.reduce((max, span) => Math.max(max, span.duration / 1_000_000), 0);
  const attributes = activeSpan?.attributes ?? {};
  const { runtime: rootRuntime, metadata: rawMetadata, ...attributeMetadata } = attributes;
  const parsedMetadata = parseJSON(rawMetadata ?? attributeMetadata);
  const nestedMetadata: Record<string, unknown> =
    parsedMetadata && typeof parsedMetadata === "object" && !Array.isArray(parsedMetadata)
      ? (parsedMetadata as Record<string, unknown>)
      : {};
  const { runtime: nestedRuntime, ...metadataWithoutRuntime } = nestedMetadata;
  const metadataText = formatJSON(
    stripSystemRuntime(rawMetadata === undefined ? attributeMetadata : metadataWithoutRuntime),
  );
  const runtimeText = formatJSON(rootRuntime ?? nestedRuntime ?? findSystemRuntime(parsedMetadata));
  const inputText = jsonMode ? formatPayload(activeSpan?.input) : activeSpan?.input || "—";
  const outputText = jsonMode ? formatPayload(activeSpan?.output) : activeSpan?.output || "—";
  const expandedInputText = formatPayload(activeSpan?.input);
  const expandedOutputText = formatPayload(activeSpan?.output);

  function selectSpan(id: string) {
    if (selectedSpanID === id && inspectorOpen) {
      setInspectorOpen((open) => !open);
      return;
    }
    setSelectedSpanID(id);
    setInspectorOpen(true);
  }

  async function copyValue(target: string, value: string) {
    try {
      if (!(await copyToClipboard(value))) {
        Toast.error({ content: t("copyFailed"), showClose: false });
        return;
      }
      setCopiedTarget(target);
      Toast.success({ content: t("copied"), showClose: false });
      window.setTimeout(() => setCopiedTarget(null), 1600);
    } catch {
      Toast.error({ content: t("copyFailed"), showClose: false });
    }
  }

  function renderFactsPanel() {
    return (
      <div className="detail-facts-panel">
        <h3>{t("facts")}</h3>
        {renderFactBadges()}
        <div className="detail-fact-row">
          <span>{t("spanID")}</span>
          <b title={activeSpan?.span_id}>{activeSpan?.span_id || "—"}</b>
        </div>
        <div className="detail-fact-row">
          <span>{t("startTime")}</span>
          <b>{activeSpan ? new Date(activeSpan.start_time).toLocaleString() : "—"}</b>
        </div>
      </div>
    );
  }

  function renderFactBadges() {
    const status = activeSpan?.status?.toLowerCase();
    const statusClass = status === "error" ? "error" : status === "ok" ? "ok" : "unset";
    return (
      <div className="detail-facts-badges">
        <span className={`detail-fact-badge status-${statusClass}`}>
          {statusClass === "error"
            ? t("errors")
            : statusClass === "ok"
              ? t("healthyStatus")
              : t("unsetStatus")}
        </span>
        <span className="detail-fact-badge">{activeSpan?.kind || "custom"}</span>
        <span className="detail-fact-duration">{formatDuration(activeSpan?.duration ?? 0)}</span>
      </div>
    );
  }

  return (
    <section className="detail-panel">
      <div className="trace-detail-toolbar">
        <Button
          aria-label={t("backToTraces")}
          className="detail-back"
          icon={<IconClose />}
          onClick={onBack}
          theme="borderless"
          type="tertiary"
        />
        <strong>{selectedID}</strong>
        <span
          className={`status-badge ${selected.some((span) => span.status === "error") ? "error" : "ok"}`}
        >
          {selected.some((span) => span.status === "error") ? t("errors") : t("healthyStatus")}
        </span>
        <span className="detail-toolbar-fact">{formatDuration(traceDuration * 1_000_000)}</span>
        <span className="detail-toolbar-fact">
          {details?.span_count ?? selected.length} {t("spans")}
        </span>
      </div>
      <div className={`detail-workbench ${view}-mode ${inspectorOpen ? "inspector-open" : ""}`}>
        <div className="detail-tree-column">
          <div className="detail-view-switch">
            <button
              className={view === "tree" ? "active" : ""}
              onClick={() => setView("tree")}
              type="button"
            >
              {t("treeView")}
            </button>
            <button
              className={view === "waterfall" ? "active" : ""}
              onClick={() => setView("waterfall")}
              type="button"
            >
              {t("waterfallView")}
            </button>
          </div>
          {view === "tree" ? (
            <SpanTree
              spans={selected}
              selectedSpanID={activeSpan?.span_id ?? ""}
              onSelect={selectSpan}
            />
          ) : (
            <Waterfall
              spans={selected}
              details={details}
              selectedSpanID={activeSpan?.span_id ?? ""}
              onSelect={selectSpan}
            />
          )}
        </div>
        <aside className="detail-inspector" aria-label={t("spanDetails")}>
          <Button
            aria-label={t("close")}
            className="detail-inspector-close"
            icon={<IconClose />}
            onClick={() => setInspectorOpen(false)}
            theme="borderless"
            type="tertiary"
          />
          <div className="detail-content-column">
            {view === "waterfall" && (
              <div className="detail-facts-summary">
                {renderFactBadges()}
                <span className="detail-fact-summary-id" title={activeSpan?.span_id}>
                  {activeSpan?.span_id || "—"}
                </span>
                <span className="detail-fact-summary-time">
                  {t("startTime")}:{" "}
                  {activeSpan ? new Date(activeSpan.start_time).toLocaleString() : "—"}
                </span>
              </div>
            )}
            <Tabs className="detail-tabs" defaultActiveKey="run" type="line">
              <TabPane itemKey="run" tab={t("run")}>
                <div className="detail-tab-content">
                  <div className="detail-section-title">
                    <h3>{t("input")}</h3>
                    <div className="payload-actions">
                      <Tooltip content={copiedTarget === "input" ? t("copied") : t("copy")}>
                        <Button
                          aria-label={t("copy")}
                          className={
                            copiedTarget === "input" ? "copy-button copied" : "copy-button"
                          }
                          icon={copiedTarget === "input" ? <IconTick /> : <IconCopy />}
                          onClick={() => void copyValue("input", inputText)}
                          theme="borderless"
                          type="tertiary"
                        />
                      </Tooltip>
                      <Button
                        className="payload-json-toggle"
                        onClick={() => setJsonMode((mode) => !mode)}
                        theme="borderless"
                        type="tertiary"
                      >
                        {jsonMode ? "JSON" : "TEXT"}
                      </Button>
                      <Tooltip content={t("expand")}>
                        <Button
                          aria-label={t("expand")}
                          icon={<IconExpand />}
                          onClick={() => setExpandedPayload("input")}
                          theme="borderless"
                          type="tertiary"
                        />
                      </Tooltip>
                    </div>
                  </div>
                  <pre className="detail-json-placeholder">{inputText}</pre>
                  <div className="detail-section-title">
                    <h3>{t("output")}</h3>
                    <div className="payload-actions">
                      <Tooltip content={copiedTarget === "output" ? t("copied") : t("copy")}>
                        <Button
                          aria-label={t("copy")}
                          className={
                            copiedTarget === "output" ? "copy-button copied" : "copy-button"
                          }
                          icon={copiedTarget === "output" ? <IconTick /> : <IconCopy />}
                          onClick={() => void copyValue("output", outputText)}
                          theme="borderless"
                          type="tertiary"
                        />
                      </Tooltip>
                      <Button
                        className="payload-json-toggle"
                        onClick={() => setJsonMode((mode) => !mode)}
                        theme="borderless"
                        type="tertiary"
                      >
                        {jsonMode ? "JSON" : "TEXT"}
                      </Button>
                      <Tooltip content={t("expand")}>
                        <Button
                          aria-label={t("expand")}
                          icon={<IconExpand />}
                          onClick={() => setExpandedPayload("output")}
                          theme="borderless"
                          type="tertiary"
                        />
                      </Tooltip>
                    </div>
                  </div>
                  <pre className="detail-json-placeholder">{outputText}</pre>
                </div>
              </TabPane>
              <TabPane itemKey="metadata" tab={t("metadata")}>
                <div className="detail-tab-content">
                  <div className="detail-json-card">
                    <div className="detail-section-title">
                      <h3>{t("metadata")}</h3>
                    </div>
                    <pre className="detail-json-placeholder">{metadataText}</pre>
                  </div>
                  <div className="detail-json-card">
                    <div className="detail-section-title">
                      <h3>{t("runtime")}</h3>
                    </div>
                    <pre className="detail-json-placeholder">{runtimeText}</pre>
                  </div>
                </div>
              </TabPane>
            </Tabs>
          </div>
          {view === "tree" && renderFactsPanel()}
        </aside>
      </div>
      {(hasMore || loadingMore) && (
        <div className="trace-detail-pagination">
          <span>{t("tracePageHint")}</span>
          <Button
            className="load-more"
            loading={loadingMore}
            onClick={onLoadMore}
            disabled={loadingMore}
          >
            {t("loadMoreSpans")}
          </Button>
        </div>
      )}
      {expandedPayload && (
        <div className="payload-viewer-backdrop">
          <dialog className="payload-viewer" aria-label={t(expandedPayload)} aria-modal="true" open>
            <header>
              <h2>{t(expandedPayload)}</h2>
              <div className="payload-actions">
                <Tooltip
                  content={copiedTarget === `expanded-${expandedPayload}` ? t("copied") : t("copy")}
                >
                  <Button
                    aria-label={t("copy")}
                    className={
                      copiedTarget === `expanded-${expandedPayload}`
                        ? "copy-button copied"
                        : "copy-button"
                    }
                    icon={
                      copiedTarget === `expanded-${expandedPayload}` ? <IconTick /> : <IconCopy />
                    }
                    onClick={() =>
                      void copyValue(
                        `expanded-${expandedPayload}`,
                        expandedPayload === "input" ? expandedInputText : expandedOutputText,
                      )
                    }
                    theme="borderless"
                    type="tertiary"
                  />
                </Tooltip>
                <Button
                  aria-label={t("close")}
                  icon={<IconClose />}
                  onClick={() => setExpandedPayload(null)}
                  theme="borderless"
                  type="tertiary"
                />
              </div>
            </header>
            <pre>{expandedPayload === "input" ? expandedInputText : expandedOutputText}</pre>
          </dialog>
        </div>
      )}
    </section>
  );
}
