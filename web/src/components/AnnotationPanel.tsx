import { useState } from "react";
import type { Annotation } from "../types";
import { usePreferences } from "../i18n";
import { Button, Form, Tooltip } from "@douyinfe/semi-ui";

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
  const [formVersion, setFormVersion] = useState(0);

  function submit() {
    onAdd();
    setFormVersion((version) => version + 1);
  }

  return (
    <section className="annotation-panel">
      <div className="detail-heading">
        <div>
          <span className="eyebrow">{t("annotations")}</span>
          <h2>{t("annotations")}</h2>
        </div>
        <span className="pill">{annotations.length}</span>
      </div>
      <Form
        className="annotation-form"
        key={formVersion}
        layout="horizontal"
        initValues={draft}
        onSubmit={submit}
        onValueChange={(values) => onDraftChange(values as AnnotationDraft)}
      >
        <Form.Input
          field="key"
          placeholder={t("annotationKey")}
          rules={[{ required: true, message: t("annotationKeyRequired") }]}
        />
        <Form.Input field="score" max={1} min={0} placeholder={t("score")} type="number" />
        <Form.Input field="label" placeholder={t("label")} />
        <Form.Input field="comment" placeholder={t("comment")} />
        <Button htmlType="submit" theme="solid" type="primary">
          {t("add")}
        </Button>
      </Form>
      {annotations.map((item) => (
        <div className="annotation-row" key={item.id}>
          <span className="kind">{item.key}</span>
          <strong>{item.label || "—"}</strong>
          <span>{item.score ?? "—"}</span>
          <small>{item.comment}</small>
          <Tooltip content={t("delete")}>
            <Button
              aria-label={t("delete")}
              className="delete-button"
              onClick={() => onDelete(item.id)}
              type="tertiary"
              theme="borderless"
            >
              ×
            </Button>
          </Tooltip>
        </div>
      ))}
    </section>
  );
}
