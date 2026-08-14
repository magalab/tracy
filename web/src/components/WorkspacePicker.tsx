import { useState } from "react";
import { usePreferences } from "../i18n";
import type { Workspace } from "../types";

type WorkspacePickerProps = {
  workspaces: Workspace[];
  activeID: string;
  onSelect: (id: string) => Promise<void>;
  onCreate: (name: string) => Promise<void>;
};

export function WorkspacePicker({
  workspaces,
  activeID,
  onSelect,
  onCreate,
}: WorkspacePickerProps) {
  const { t } = usePreferences();
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);

  async function create() {
    if (!name.trim()) return;
    setCreating(true);
    try {
      await onCreate(name.trim());
      setName("");
    } finally {
      setCreating(false);
    }
  }

  return (
    <main className="workspace-page">
      <section className="workspace-card">
        <img className="auth-brand-mark" src="/tracy.svg" alt="Tracy" />
        <span className="eyebrow">{t("workspaceLabel")}</span>
        <h1>{t("chooseWorkspace")}</h1>
        <div className="workspace-list">
          {workspaces.map((workspace) => (
            <button
              className={`workspace-option ${activeID === workspace.id ? "active" : ""}`}
              key={workspace.id}
              onClick={() => void onSelect(workspace.id)}
            >
              <span className="workspace-icon">{workspace.name.slice(0, 1).toUpperCase()}</span>
              <span>
                <b>{workspace.name}</b>
                <small>{workspace.id}</small>
              </span>
              <span>→</span>
            </button>
          ))}
        </div>
        <div className="workspace-create">
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder={t("workspaceName")}
            onKeyDown={(event) => event.key === "Enter" && void create()}
          />
          <button onClick={() => void create()} disabled={creating || !name.trim()}>
            {t("createWorkspace")}
          </button>
        </div>
      </section>
    </main>
  );
}
