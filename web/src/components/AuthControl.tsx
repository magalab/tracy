import { useState } from "react";
import { usePreferences } from "../i18n";

type AuthUser = { name: string; email: string };

type AuthControlProps = {
  user: AuthUser | null;
  onLogin: (email: string, password: string) => Promise<void>;
  onLogout: () => void;
};

export function AuthControl({ user, onLogin, onLogout }: AuthControlProps) {
  const { t } = usePreferences();
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit() {
    setLoading(true);
    setError("");
    try {
      await onLogin(email, password);
      setOpen(false);
      setPassword("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  if (user) {
    return (
      <button className="user-summary" onClick={onLogout} title={t("signOut")}>
        <span className="user-avatar">{user.name.slice(0, 1).toUpperCase()}</span>
        <span>
          <b>{user.name}</b>
          <small>{t("signedIn")}</small>
        </span>
      </button>
    );
  }

  return (
    <div className="auth-control">
      <button className="auth-trigger" onClick={() => setOpen((current) => !current)}>
        {t("signIn")}
      </button>
      {open && (
        <div className="auth-popover">
          <div className="connection-popover-heading">
            <span>
              <b>{t("signIn")}</b>
              <small>{t("signInHint")}</small>
            </span>
            <button
              className="popover-close"
              onClick={() => setOpen(false)}
              aria-label={t("close")}
            >
              ×
            </button>
          </div>
          <input
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            type="email"
            placeholder={t("email")}
            autoComplete="username"
          />
          <input
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            type="password"
            placeholder={t("password")}
            autoComplete="current-password"
            onKeyDown={(event) => event.key === "Enter" && void submit()}
          />
          {error && <small className="auth-error">{error}</small>}
          <button className="connection-save" onClick={() => void submit()} disabled={loading}>
            {loading ? t("signingIn") : t("signIn")}
          </button>
        </div>
      )}
    </div>
  );
}
