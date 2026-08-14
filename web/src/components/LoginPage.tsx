import { useState } from "react";
import { usePreferences } from "../i18n";

type LoginPageProps = {
  onLogin: (email: string, password: string) => Promise<void>;
};

export function LoginPage({ onLogin }: LoginPageProps) {
  const { t } = usePreferences();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit() {
    setLoading(true);
    setError("");
    try {
      await onLogin(email, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-card">
        <img className="auth-brand-mark" src="/tracy.svg" alt="Tracy" />
        <span className="eyebrow">{t("selfHosted")}</span>
        <h1>{t("welcomeBack")}</h1>
        <p>{t("signInHint")}</p>
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
        {error && <div className="auth-error">{error}</div>}
        <button onClick={() => void submit()} disabled={loading}>
          {loading ? t("signingIn") : t("signIn")}
        </button>
      </section>
    </main>
  );
}
