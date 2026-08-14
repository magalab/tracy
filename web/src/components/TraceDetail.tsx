import type { CSSProperties } from "react";
import { AnnotationPanel, type AnnotationDraft } from "./AnnotationPanel";
import type { Annotation, Span } from "../types";
import { usePreferences } from "../i18n";

type TraceDetailProps = {
  selectedID: string;
  selected: Span[];
  annotations: Annotation[];
  draft: AnnotationDraft;
  onDraftChange: (draft: AnnotationDraft) => void;
  onAddAnnotation: () => void;
  onDeleteAnnotation: (id: string) => void;
  onBack: () => void;
};

function formatDuration(nanoseconds: number) {
  const ms = nanoseconds / 1_000_000;
  return ms < 1 ? `${Math.round(nanoseconds / 1_000)}μs` : `${ms.toFixed(2)}ms`;
}

function SpanTree({ spans }: { spans: Span[] }) {
  const { t } = usePreferences();
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

  function renderSpan(span: Span, depth: number): React.ReactNode {
    return (
      <div className="span-tree-node" key={span.span_id}>
        <article
          className={`span-card ${depth > 0 ? "nested" : ""}`}
          style={{ "--tree-depth": depth } as CSSProperties}
        >
          <div className="span-line">
            <span className="tree-mark">{depth > 0 ? "└─" : "●"}</span>
            <span className={`status-dot ${span.status}`} />
            <strong>{span.name}</strong>
            <span className="kind">{span.kind || "custom"}</span>
            <span className="duration">{formatDuration(span.duration)}</span>
          </div>
          <div className="span-subline">
            {span.span_id} · {new Date(span.start_time).toLocaleTimeString()}
          </div>
          {(span.input || span.output) && (
            <div className="io-grid">
              {span.input && (
                <div>
                  <span className="field-label">{t("input")}</span>
                  <pre>{span.input}</pre>
                </div>
              )}
              {span.output && (
                <div>
                  <span className="field-label">{t("output")}</span>
                  <pre>{span.output}</pre>
                </div>
              )}
            </div>
          )}
          {span.attributes && Object.keys(span.attributes).length > 0 && (
            <details>
              <summary>
                {t("attributes")} ({Object.keys(span.attributes).length})
              </summary>
              <pre>{JSON.stringify(span.attributes, null, 2)}</pre>
            </details>
          )}
        </article>
        {children.get(span.span_id)?.map((child) => renderSpan(child, depth + 1))}
      </div>
    );
  }

  return <div className="span-tree">{roots.map((span) => renderSpan(span, 0))}</div>;
}

export function TraceDetail({
  selectedID,
  selected,
  annotations,
  draft,
  onDraftChange,
  onAddAnnotation,
  onDeleteAnnotation,
  onBack,
}: TraceDetailProps) {
  const { t } = usePreferences();
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

  return (
    <section className="detail-panel">
      <div className="trace-detail-toolbar">
        <button className="detail-back" onClick={onBack} aria-label={t("backToTraces")}>
          ×
        </button>
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
      <div className="detail-heading">
        <div>
          <h2>{t("traceDetail")}</h2>
        </div>
        <div className="detail-tabs">
          <button className="active">{t("run")}</button>
          <button>{t("metadata")}</button>
          <button>{t("feedback")}</button>
        </div>
      </div>
      <div className="detail-workbench">
        <div className="detail-tree-column">
          <SpanTree spans={selected} />
        </div>
        <div className="detail-content-column">
          <div className="detail-section-title">
            <h3>{t("input")}</h3>
            <span>{t("raw")}</span>
          </div>
          <div className="detail-json-placeholder">
            {selected[0]?.input || selected[0]?.output || "—"}
          </div>
          <div className="detail-section-title">
            <h3>{t("output")}</h3>
            <span>{t("raw")}</span>
          </div>
          <div className="detail-json-placeholder">{selected[0]?.output || "—"}</div>
          <AnnotationPanel
            annotations={annotations}
            draft={draft}
            onDraftChange={onDraftChange}
            onAdd={onAddAnnotation}
            onDelete={onDeleteAnnotation}
          />
        </div>
        <aside className="detail-facts-panel">
          <span>{t("status")}</span>
          <b>
            {selected.some((span) => span.status === "error") ? t("errors") : t("healthyStatus")}
          </b>
          <span>{t("spanID")}</span>
          <b>{selected[0]?.span_id || "—"}</b>
          <span>{t("duration")}</span>
          <b>{formatDuration(selected.reduce((total, span) => total + span.duration, 0))}</b>
          <span>{t("startTime")}</span>
          <b>{selected[0] ? new Date(selected[0].start_time).toLocaleString() : "—"}</b>
        </aside>
      </div>
    </section>
  );
}
