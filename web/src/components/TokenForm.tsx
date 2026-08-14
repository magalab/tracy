import { usePreferences } from "../i18n";

type TokenFormProps = {
  value: string;
  connected: boolean;
  onChange: (value: string) => void;
  onSave: () => void;
};

export function TokenForm({ value, connected, onChange, onSave }: TokenFormProps) {
  const { t } = usePreferences();
  return (
    <div className={`token-form ${connected ? "connected" : ""}`}>
      <div className="token-state">
        <span className="token-state-dot" />
        <span>
          <b>{connected ? t("connected") : t("connectProject")}</b>
        </span>
      </div>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        type="password"
        aria-label={t("projectApiKey")}
        placeholder={t("pasteApiKey")}
        onKeyDown={(event) => event.key === "Enter" && onSave()}
      />
      <button onClick={onSave}>{connected ? t("reconnect") : t("connect")}</button>
    </div>
  );
}
