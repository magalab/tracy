import { usePreferences, type TranslationKey } from "../i18n";
import type { User, Workspace } from "../types";
import { UserMenu } from "./UserMenu";
import { WorkspaceMenu } from "./WorkspaceMenu";

type AppSidebarProps = {
  user: User;
  workspaces: Workspace[];
  activeID: string;
  onSelectWorkspace: (id: string) => Promise<void>;
  onCreateWorkspace: (name: string) => Promise<void>;
  onLogout: () => void;
  collapsed: boolean;
  onToggle: () => void;
  activePage: "overview" | "traces";
  onPageChange: (page: "overview" | "traces") => void;
};

type NavItem = { icon: string; label: TranslationKey; page: "overview" | "traces" };
type NavGroup = { label: TranslationKey; items: NavItem[] };

const groups: NavGroup[] = [
  {
    label: "observability",
    items: [
      { icon: "◒", label: "overview", page: "overview" },
      { icon: "⌘", label: "trace", page: "traces" },
    ],
  },
] as const;

export function AppSidebar({
  user,
  workspaces,
  activeID,
  onSelectWorkspace,
  onCreateWorkspace,
  onLogout,
  collapsed,
  onToggle,
  activePage,
  onPageChange,
}: AppSidebarProps) {
  const { t } = usePreferences();

  return (
    <aside className={`app-sidebar ${collapsed ? "collapsed" : ""}`}>
      <div className="sidebar-brand">
        <img className="brand-mark" src="/tracy.svg" alt="Tracy" />
        <strong>Tracy</strong>
        <button className="sidebar-collapse" aria-label={t("collapseSidebar")} onClick={onToggle}>
          {collapsed ? "›" : "‹"}
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
                className={`sidebar-nav-item ${activePage === item.page ? "active" : ""}`}
                key={item.label}
                onClick={() => onPageChange(item.page)}
              >
                <span className="sidebar-nav-icon">{item.icon}</span>
                <span>{t(item.label)}</span>
              </button>
            ))}
          </div>
        ))}
      </nav>
      <div className="sidebar-user">
        <UserMenu user={user} onLogout={onLogout} />
      </div>
    </aside>
  );
}
