import { useState } from "react";
import Button from "@douyinfe/semi-ui/lib/es/button";
import { Form } from "@douyinfe/semi-ui/lib/es/form";
import { usePreferences } from "../i18n";

type LoginPageProps = {
  onLogin: (email: string, password: string) => Promise<void>;
};

export function LoginPage({ onLogin }: LoginPageProps) {
  const { t } = usePreferences();
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
        <img className="auth-brand-mark" src="/tracy.svg" alt="Tracy" />
        <span className="eyebrow">{t("selfHosted")}</span>
        <h1>{t("welcomeBack")}</h1>
        <p>{t("signInHint")}</p>
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
