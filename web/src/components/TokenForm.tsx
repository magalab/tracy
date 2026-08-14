import { useState } from "react";
import { usePreferences } from "../i18n";

type TokenFormProps = {
  value: string;
  connected: boolean;
  onChange: (value: string) => void;
  onSave: () => void;
};

export function TokenForm({ value, connected, onChange, onSave }: TokenFormProps) {
  const { t } = usePreferences();
  const [open, setOpen] = useState(false);
  return (
    <div className={`connection-control ${connected ? "connected" : ""}`}>
      <button className="connection-summary" onClick={() => setOpen((current) => !current)}>
        <span className="token-state-dot" />
        <span>
          <b>{connected ? t("connected") : t("connectProject")}</b>
          <small>{connected ? t("apiKeyActive") : t("enterApiKey")}</small>
        </span>
        <span className="connection-chevron">⌄</span>
      </button>
      {open && (
        <div className="connection-popover">
          <div className="connection-popover-heading">
            <span>
              <b>{connected ? t("manageConnection") : t("connectProject")}</b>
              <small>{t("connectionHint")}</small>
            </span>
            <button className="popover-close" onClick={() => setOpen(false)} aria-label="Close">
              ×
            </button>
          </div>
          <input
            value={value}
            onChange={(event) => onChange(event.target.value)}
            type="password"
            aria-label={t("projectApiKey")}
            placeholder={t("pasteApiKey")}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                onSave();
                setOpen(false);
              }
            }}
          />
          <button
            className="connection-save"
            onClick={() => {
              onSave();
              setOpen(false);
            }}
          >
            {connected ? t("reconnect") : t("saveKey")}
          </button>
        </div>
      )}
    </div>
  );
}
