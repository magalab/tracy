import { usePreferences } from "../i18n";
import type { User } from "../types";

type UserMenuProps = {
  user: User;
  onLogout: () => void;
};

export function UserMenu({ user, onLogout }: UserMenuProps) {
  const { t } = usePreferences();
  return (
    <button className="user-summary" onClick={onLogout} title={t("signOut")}>
      <span className="user-avatar">{user.name.slice(0, 1).toUpperCase()}</span>
      <span>
        <b>{user.name}</b>
        <small>{user.email}</small>
      </span>
    </button>
  );
}
