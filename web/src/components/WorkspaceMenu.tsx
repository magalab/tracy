import { useState } from "react";
import { usePreferences } from "../i18n";
import type { Workspace } from "../types";
import Button from "@douyinfe/semi-ui/lib/es/button";
import Input from "@douyinfe/semi-ui/lib/es/input";
import Popover from "@douyinfe/semi-ui/lib/es/popover";

type WorkspaceMenuProps = {
  workspaces: Workspace[];
  activeID: string;
  onSelect: (id: string) => Promise<void>;
  onCreate: (name: string) => Promise<void>;
};

export function WorkspaceMenu({ workspaces, activeID, onSelect, onCreate }: WorkspaceMenuProps) {
  const { t } = usePreferences();
  const [open, setOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const active = workspaces.find((workspace) => workspace.id === activeID);

  async function create() {
    if (!name.trim()) return;
    setCreating(true);
    try {
      await onCreate(name.trim());
      setName("");
      setOpen(false);
    } finally {
      setCreating(false);
    }
  }

  const content = (
    <div className="workspace-menu-popover">
      <span className="workspace-menu-title">{t("switchWorkspace")}</span>
      <div className="workspace-menu-list">
        {workspaces.map((workspace) => (
          <button
            className={`workspace-menu-option ${workspace.id === activeID ? "active" : ""}`}
            key={workspace.id}
            type="button"
            onClick={() => {
              void onSelect(workspace.id);
              setOpen(false);
            }}
          >
            <span>{workspace.name}</span>
            {workspace.id === activeID && <span>✓</span>}
          </button>
        ))}
      </div>
      <div className="workspace-menu-create">
        <span>{t("newWorkspace")}</span>
        <div>
          <Input
            value={name}
            onChange={setName}
            placeholder={t("workspaceName")}
            onKeyDown={(event) => event.key === "Enter" && void create()}
          />
          <Button
            onClick={() => void create()}
            disabled={creating || !name.trim()}
            loading={creating}
          >
            +
          </Button>
        </div>
      </div>
    </div>
  );

  return (
    <div className="workspace-menu">
      <Popover
        content={content}
        position="bottomLeft"
        trigger="click"
        visible={open}
        onVisibleChange={setOpen}
      >
        <button className="workspace-menu-trigger" type="button" aria-expanded={open}>
          <span className="workspace-menu-icon">
            {active?.name.slice(0, 1).toUpperCase() ?? "W"}
          </span>
          <span>
            <small>{t("workspaceLabel")}</small>
            <b>{active?.name ?? t("chooseWorkspace")}</b>
          </span>
          <span className="workspace-menu-chevron">⌄</span>
        </button>
      </Popover>
    </div>
  );
}
