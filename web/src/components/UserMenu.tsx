import { useEffect, useRef, useState } from "react";
import { usePreferences } from "../i18n";
import type { User } from "../types";

type UserMenuProps = {
  user: User;
  onLogout: () => void;
};

export function UserMenu({ user, onLogout }: UserMenuProps) {
  const { t } = usePreferences();
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;

    function handlePointerDown(event: PointerEvent) {
      if (!menuRef.current?.contains(event.target as Node)) setOpen(false);
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") setOpen(false);
    }

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  return (
    <div className="user-menu" ref={menuRef}>
      <button
        className="user-summary"
        onClick={() => setOpen((current) => !current)}
        aria-expanded={open}
        aria-haspopup="menu"
        title={t("signedIn")}
        type="button"
      >
        <span className="user-avatar">{user.name.slice(0, 1).toUpperCase()}</span>
        <span>
          <b>{user.name}</b>
          <small>{user.email}</small>
        </span>
        <span className="user-menu-chevron" aria-hidden="true">
          {open ? "⌃" : "⌄"}
        </span>
      </button>
      {open && (
        <div className="user-popover" role="menu">
          <div className="user-popover-account">
            <strong>{user.name}</strong>
            <small>{user.email}</small>
          </div>
          <button
            className="user-popover-action"
            onClick={() => {
              setOpen(false);
              onLogout();
            }}
            role="menuitem"
            type="button"
          >
            {t("signOut")}
          </button>
        </div>
      )}
    </div>
  );
}
