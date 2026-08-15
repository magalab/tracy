import { useEffect, useState, type CSSProperties } from "react";
import type { Span } from "../types";
import { usePreferences } from "../i18n";
import Button from "@douyinfe/semi-ui/lib/es/button";
import TabPane from "@douyinfe/semi-ui/lib/es/tabs/TabPane";
import Tabs from "@douyinfe/semi-ui/lib/es/tabs";
import Tooltip from "@douyinfe/semi-ui/lib/es/tooltip";
import IconClose from "@douyinfe/semi-icons/lib/es/icons/IconClose";
import IconCopy from "@douyinfe/semi-icons/lib/es/icons/IconCopy";
import IconExpand from "@douyinfe/semi-icons/lib/es/icons/IconExpand";
import IconTick from "@douyinfe/semi-icons/lib/es/icons/IconTick";

type TraceDetailProps = {
  selectedID: string;
  selected: Span[];
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
  hasMore,
  loadingMore,
  onLoadMore,
  onBack,
}: TraceDetailProps) {
  const { t } = usePreferences();
  const [selectedSpanID, setSelectedSpanID] = useState(selected[0]?.span_id ?? "");
  const [expandedPayload, setExpandedPayload] = useState<"input" | "output" | null>(null);
  const [jsonMode, setJsonMode] = useState(true);
  const [copiedTarget, setCopiedTarget] = useState<string | null>(null);

  useEffect(() => {
    setSelectedSpanID(selected[0]?.span_id ?? "");
  }, [selectedID, selected]);

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

  async function copyValue(target: string, value: string) {
    if (!navigator.clipboard) return;
    await navigator.clipboard.writeText(value);
    setCopiedTarget(target);
    window.setTimeout(() => setCopiedTarget(null), 1600);
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
        <span className="detail-toolbar-fact">
          {formatDuration(selected.reduce((total, span) => total + span.duration, 0))}
        </span>
        <span className="detail-toolbar-fact">
          {selected.length} {t("spans")}
        </span>
      </div>
      <div className="detail-workbench">
        <div className="detail-tree-column">
          <SpanTree
            spans={selected}
            selectedSpanID={activeSpan?.span_id ?? ""}
            onSelect={setSelectedSpanID}
          />
        </div>
        <div className="detail-content-column">
          <Tabs className="detail-tabs" defaultActiveKey="run" type="line">
            <TabPane itemKey="run" tab={t("run")}>
              <div className="detail-tab-content">
                <div className="detail-section-title">
                  <h3>{t("input")}</h3>
                  <div className="payload-actions">
                    <Tooltip content={copiedTarget === "input" ? t("copied") : t("copy")}>
                      <Button
                        aria-label={t("copy")}
                        className={copiedTarget === "input" ? "copy-button copied" : "copy-button"}
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
                        className={copiedTarget === "output" ? "copy-button copied" : "copy-button"}
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
        <aside className="detail-facts-panel">
          <span>{t("status")}</span>
          <b>{activeSpan?.status || "—"}</b>
          <span>{t("spanID")}</span>
          <b>{activeSpan?.span_id || "—"}</b>
          <span>{t("type")}</span>
          <b>{activeSpan?.kind || "custom"}</b>
          <span>{t("duration")}</span>
          <b>{formatDuration(activeSpan?.duration ?? 0)}</b>
          <span>{t("startTime")}</span>
          <b>{activeSpan ? new Date(activeSpan.start_time).toLocaleString() : "—"}</b>
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
