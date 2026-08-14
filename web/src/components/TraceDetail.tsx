import type { CSSProperties } from "react";
import { AnnotationPanel, type AnnotationDraft } from "./AnnotationPanel";
import type { Annotation, Span } from "../types";

type TraceDetailProps = {
  selectedID: string;
  selected: Span[];
  annotations: Annotation[];
  draft: AnnotationDraft;
  onDraftChange: (draft: AnnotationDraft) => void;
  onAddAnnotation: () => void;
  onDeleteAnnotation: (id: string) => void;
};

function formatDuration(nanoseconds: number) {
  const ms = nanoseconds / 1_000_000;
  return ms < 1 ? `${Math.round(nanoseconds / 1_000)}μs` : `${ms.toFixed(2)}ms`;
}

function SpanTree({ spans }: { spans: Span[] }) {
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
                  <span className="field-label">INPUT</span>
                  <pre>{span.input}</pre>
                </div>
              )}
              {span.output && (
                <div>
                  <span className="field-label">OUTPUT</span>
                  <pre>{span.output}</pre>
                </div>
              )}
            </div>
          )}
          {span.attributes && Object.keys(span.attributes).length > 0 && (
            <details>
              <summary>Attributes ({Object.keys(span.attributes).length})</summary>
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
}: TraceDetailProps) {
  if (!selectedID) {
    return (
      <section className="detail-panel">
        <div className="detail-empty">
          <div className="orbit">✦</div>
          <h2>Select a trace</h2>
          <p>Choose a trace from the list to inspect its span tree, timing, inputs and outputs.</p>
        </div>
      </section>
    );
  }

  return (
    <section className="detail-panel">
      <div className="detail-heading">
        <div>
          <span className="eyebrow">TRACE DETAIL</span>
          <h2>{selectedID}</h2>
        </div>
        <span className="pill">{selected.length} spans</span>
      </div>
      <SpanTree spans={selected} />
      <AnnotationPanel
        annotations={annotations}
        draft={draft}
        onDraftChange={onDraftChange}
        onAdd={onAddAnnotation}
        onDelete={onDeleteAnnotation}
      />
    </section>
  );
}
