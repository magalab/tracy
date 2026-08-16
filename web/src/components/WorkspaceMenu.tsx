import { useState } from "react";
import IconCopy from "@douyinfe/semi-icons/lib/es/icons/IconCopy";
import IconTick from "@douyinfe/semi-icons/lib/es/icons/IconTick";
import Toast from "@douyinfe/semi-ui/lib/es/toast";
import { usePreferences } from "../i18n";
import type { Workspace } from "../types";
import Button from "@douyinfe/semi-ui/lib/es/button";
import Input from "@douyinfe/semi-ui/lib/es/input";
import Popover from "@douyinfe/semi-ui/lib/es/popover";
import { APIError } from "../api/client";
import { copyToClipboard } from "../utils/clipboard";

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
  const [copiedID, setCopiedID] = useState(false);
  const active = workspaces.find((workspace) => workspace.id === activeID);

  async function create() {
    if (!name.trim()) return;
    setCreating(true);
    try {
      await onCreate(name.trim());
      setName("");
      setOpen(false);
    } catch (err) {
      let message = t("workspaceCreateFailed");
      if (err instanceof APIError && err.status === 409) {
        message = t("workspaceNameExists");
      } else if (err instanceof Error) {
        message = err.message;
      }
      Toast.error({
        content: message,
        showClose: false,
      });
    } finally {
      setCreating(false);
    }
  }

  async function copyWorkspaceID() {
    if (!active?.id) return;
    try {
      if (!(await copyToClipboard(active.id))) {
        Toast.error({ content: t("copyFailed"), showClose: false });
        return;
      }
      setCopiedID(true);
      Toast.success({ content: t("workspaceIDCopied"), showClose: false });
      window.setTimeout(() => setCopiedID(false), 1800);
    } catch {
      Toast.error({
        content: t("copyFailed"),
        showClose: false,
      });
    }
  }

  const content = (
    <div className="workspace-menu-popover">
      <span className="workspace-menu-title">{t("switchWorkspace")}</span>
      <div className="workspace-menu-current">
        <span>
          <small>{t("workspaceID")}</small>
          <b>{active?.id ?? "—"}</b>
        </span>
        <Button
          className="workspace-copy-button"
          aria-label={copiedID ? t("workspaceIDCopied") : t("copyWorkspaceID")}
          title={copiedID ? t("workspaceIDCopied") : t("copyWorkspaceID")}
          icon={copiedID ? <IconTick /> : <IconCopy />}
          onClick={() => void copyWorkspaceID()}
          disabled={!active?.id}
        />
      </div>
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
