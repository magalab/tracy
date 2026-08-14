import { usePreferences, type TranslationKey } from "../i18n";
import type { User, Workspace } from "../types";
import { UserMenu } from "./UserMenu";
import { WorkspaceMenu } from "./WorkspaceMenu";

type AppSidebarProps = {
  user: User;
  workspace: Workspace;
  workspaces: Workspace[];
  activeID: string;
  onSelectWorkspace: (id: string) => Promise<void>;
  onCreateWorkspace: (name: string) => Promise<void>;
  onLogout: () => void;
};

type NavItem = { icon: string; label: TranslationKey; active?: boolean };
type NavGroup = { label: TranslationKey; items: NavItem[] };

const groups: NavGroup[] = [
  {
    label: "observability",
    items: [{ icon: "⌘", label: "trace", active: true }],
  },
] as const;

export function AppSidebar({
  user,
  workspace,
  workspaces,
  activeID,
  onSelectWorkspace,
  onCreateWorkspace,
  onLogout,
}: AppSidebarProps) {
  const { t } = usePreferences();

  return (
    <aside className="app-sidebar">
      <div className="sidebar-brand">
        <img className="brand-mark" src="/tracy.svg" alt="Tracy" />
        <strong>Tracy</strong>
        <button className="sidebar-collapse" aria-label={t("collapseSidebar")}>
          ‹
        </button>
      </div>
      <WorkspaceMenu
        workspaces={workspaces}
        activeID={activeID}
        onSelect={onSelectWorkspace}
        onCreate={onCreateWorkspace}
      />
      <nav className="sidebar-nav" aria-label="Primary navigation">
        {groups.map((group) => (
          <div className="sidebar-nav-group" key={group.label}>
            <span className="sidebar-nav-label">{t(group.label)}</span>
            {group.items.map((item) => (
              <button
                className={`sidebar-nav-item ${item.active ? "active" : ""}`}
                key={item.label}
              >
                <span className="sidebar-nav-icon">{item.icon}</span>
                <span>{t(item.label)}</span>
              </button>
            ))}
          </div>
        ))}
      </nav>
      <div className="sidebar-user">
        <UserMenu user={user} workspace={workspace} onLogout={onLogout} />
      </div>
    </aside>
  );
}
