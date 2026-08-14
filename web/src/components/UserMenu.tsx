import { usePreferences } from "../i18n";
import type { User, Workspace } from "../types";

type UserMenuProps = {
  user: User;
  workspace: Workspace;
  onLogout: () => void;
};

export function UserMenu({ user, workspace, onLogout }: UserMenuProps) {
  const { t } = usePreferences();
  return (
    <button className="user-summary" onClick={onLogout} title={t("signOut")}>
      <span className="user-avatar">{user.name.slice(0, 1).toUpperCase()}</span>
      <span>
        <b>{workspace.name}</b>
        <small>
          {user.name} · {t("signOut")}
        </small>
      </span>
    </button>
  );
}
