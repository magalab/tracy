import type { Annotation } from "../types";
import { usePreferences } from "../i18n";

export type AnnotationDraft = {
  key: string;
  score: string;
  label: string;
  comment: string;
};

type AnnotationPanelProps = {
  annotations: Annotation[];
  draft: AnnotationDraft;
  onDraftChange: (draft: AnnotationDraft) => void;
  onAdd: () => void;
  onDelete: (id: string) => void;
};

export function AnnotationPanel({
  annotations,
  draft,
  onDraftChange,
  onAdd,
  onDelete,
}: AnnotationPanelProps) {
  const { t } = usePreferences();
  return (
    <section className="annotation-panel">
      <div className="detail-heading">
        <div>
          <span className="eyebrow">{t("annotations")}</span>
          <h2>{t("annotations")}</h2>
        </div>
        <span className="pill">{annotations.length}</span>
      </div>
      <div className="annotation-form">
        <input
          value={draft.key}
          onChange={(event) => onDraftChange({ ...draft, key: event.target.value })}
          placeholder={t("annotationKey")}
        />
        <input
          value={draft.score}
          onChange={(event) => onDraftChange({ ...draft, score: event.target.value })}
          type="number"
          min="0"
          max="1"
          step="0.1"
          placeholder={t("score")}
        />
        <input
          value={draft.label}
          onChange={(event) => onDraftChange({ ...draft, label: event.target.value })}
          placeholder={t("label")}
        />
        <input
          value={draft.comment}
          onChange={(event) => onDraftChange({ ...draft, comment: event.target.value })}
          placeholder={t("comment")}
        />
        <button onClick={onAdd}>{t("add")}</button>
      </div>
      {annotations.map((item) => (
        <div className="annotation-row" key={item.id}>
          <span className="kind">{item.key}</span>
          <strong>{item.label || "—"}</strong>
          <span>{item.score ?? "—"}</span>
          <small>{item.comment}</small>
          <button className="delete-button" onClick={() => onDelete(item.id)}>
            ×
          </button>
        </div>
      ))}
    </section>
  );
}
