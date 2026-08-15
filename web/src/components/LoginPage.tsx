import { useState } from "react";
import Button from "@douyinfe/semi-ui/lib/es/button";
import { Form } from "@douyinfe/semi-ui/lib/es/form";
import { usePreferences } from "../i18n";

type LoginPageProps = {
  onLogin: (email: string, password: string) => Promise<void>;
};

export function LoginPage({ onLogin }: LoginPageProps) {
  const { language, theme, t, toggleLanguage, toggleTheme } = usePreferences();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(values: { email: string; password: string }) {
    setLoading(true);
    setError("");
    try {
      await onLogin(values.email, values.password);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-card">
        <div className="auth-preferences" aria-label={t("preferences")}>
          <Button
            aria-label={t("language")}
            className="preference-button"
            theme="borderless"
            type="tertiary"
            onClick={toggleLanguage}
          >
            {language === "en" ? "中" : "EN"}
          </Button>
          <Button
            aria-label={theme === "dark" ? t("theme") : t("darkTheme")}
            className="preference-button"
            theme="borderless"
            type="tertiary"
            onClick={toggleTheme}
            title={theme === "dark" ? t("theme") : t("darkTheme")}
          >
            {theme === "dark" ? "☼" : "☾"}
          </Button>
        </div>
        <img className="auth-brand-mark" src="/tracy.svg" alt="Tracy" />
        <Form
          layout="vertical"
          onSubmit={(values) => void submit(values as { email: string; password: string })}
        >
          <Form.Input
            field="email"
            label={t("email")}
            placeholder={t("email")}
            rules={[{ required: true, message: t("emailRequired") }]}
            type="email"
            autoComplete="username"
          />
          <Form.Input
            field="password"
            label={t("password")}
            placeholder={t("password")}
            rules={[{ required: true, message: t("passwordRequired") }]}
            type="password"
            autoComplete="current-password"
          />
          {error && <div className="auth-error">{error}</div>}
          <Button htmlType="submit" loading={loading} theme="solid" type="primary" block>
            {t("signIn")}
          </Button>
        </Form>
      </section>
    </main>
  );
}
