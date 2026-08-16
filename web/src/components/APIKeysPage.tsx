import IconClose from "@douyinfe/semi-icons/lib/es/icons/IconClose";
import IconCopy from "@douyinfe/semi-icons/lib/es/icons/IconCopy";
import IconDeleteStroked from "@douyinfe/semi-icons/lib/es/icons/IconDeleteStroked";
import IconPlus from "@douyinfe/semi-icons/lib/es/icons/IconPlus";
import IconTick from "@douyinfe/semi-icons/lib/es/icons/IconTick";
import Modal from "@douyinfe/semi-ui/lib/es/modal";
import Toast from "@douyinfe/semi-ui/lib/es/toast";
import { useCallback, useEffect, useState } from "react";
import { usePreferences } from "../i18n";
import { createWorkspaceKey, listWorkspaceKeys, revokeWorkspaceKey } from "../api/client";
import type { APIKey } from "../types";

type APIKeysPageProps = { token: string; workspaceID: string };

export function APIKeysPage({ token, workspaceID }: APIKeysPageProps) {
  const { t } = usePreferences();
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [name, setName] = useState("");
  const [expiration, setExpiration] = useState("never");
  const [newToken, setNewToken] = useState("");
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setKeys((await listWorkspaceKeys(token, workspaceID)).items);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [token, workspaceID]);

  useEffect(() => {
    void load();
  }, [load]);

  async function create() {
    if (!name.trim()) return;
    const expirationDays = expiration === "never" ? 0 : Number(expiration);
    const expirationDate = expirationDays
      ? new Date(Date.now() + expirationDays * 24 * 60 * 60 * 1000).toISOString()
      : undefined;
    setSaving(true);
    try {
      const data = await createWorkspaceKey(token, workspaceID, name.trim(), expirationDate);
      setKeys((items) => [...items, data.api_key]);
      setNewToken(data.api_key.token);
      setCopied(false);
      setName("");
      setExpiration("never");
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function revoke(key: APIKey) {
    Modal.confirm({
      title: t("revokeKey"),
      content: t("revokeKeyConfirm"),
      centered: true,
      width: 480,
      okText: t("revoke"),
      cancelText: t("cancel"),
      okType: "danger",
      onOk: async () => {
        try {
          await revokeWorkspaceKey(token, workspaceID, key.id);
          setKeys((items) =>
            items.map((item) => (item.id === key.id ? { ...item, revoked: true } : item)),
          );
        } catch (err) {
          setError(err instanceof Error ? err.message : String(err));
        }
      },
    });
  }

  async function copyToken() {
    try {
      await navigator.clipboard.writeText(newToken);
      setCopied(true);
      Toast.success({ content: t("copied"), showClose: false });
      window.setTimeout(() => setCopied(false), 1800);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      Toast.error({ content: t("copyFailed"), showClose: false });
    }
  }

  return (
    <section className="keys-page">
      <div className="key-create-card">
        <div>
          <strong>{t("createAPIKey")}</strong>
          <p>{t("keyPlaintextHint")}</p>
        </div>
        <div className="key-create-form">
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder={t("keyName")}
          />
          <select
            value={expiration}
            onChange={(event) => setExpiration(event.target.value)}
            aria-label={t("expiresAt")}
          >
            <option value="never">{t("neverExpires")}</option>
            <option value="30">{t("expires30Days")}</option>
            <option value="90">{t("expires90Days")}</option>
            <option value="365">{t("expires1Year")}</option>
          </select>
          <button
            className="icon-button"
            type="button"
            onClick={() => void create()}
            disabled={saving || !name.trim()}
            aria-label={saving ? t("creating") : t("create")}
            title={saving ? t("creating") : t("create")}
          >
            <IconPlus spin={saving} />
          </button>
        </div>
      </div>
      {newToken && (
        <div className="new-token-card">
          <strong>{t("keyCreated")}</strong>
          <p>{t("keyCreatedHint")}</p>
          <div className="new-token-row">
            <code>{newToken}</code>
            <button
              type="button"
              className={`icon-button ${copied ? "success" : ""}`}
              onClick={() => void copyToken()}
              aria-label={copied ? t("copied") : t("copy")}
              title={copied ? t("copied") : t("copy")}
            >
              {copied ? <IconTick /> : <IconCopy />}
            </button>
            <button
              className="icon-button subtle"
              type="button"
              onClick={() => {
                setNewToken("");
                setCopied(false);
              }}
              aria-label={t("close")}
              title={t("close")}
            >
              <IconClose />
            </button>
          </div>
        </div>
      )}
      {error && <p className="keys-error">{error}</p>}
      <div className="keys-table-wrap">
        {loading ? (
          <p className="table-state">{t("loading")}</p>
        ) : keys.length === 0 ? (
          <p className="table-state">{t("noKeys")}</p>
        ) : (
          <table className="keys-table">
            <thead>
              <tr>
                <th>{t("keyName")}</th>
                <th>{t("status")}</th>
                <th>{t("expiresAt")}</th>
                <th>{t("lastUsed")}</th>
                <th aria-label={t("revoke")} />
              </tr>
            </thead>
            <tbody>
              {keys.map((key) => (
                <tr key={key.id}>
                  <td>
                    <strong>{key.name}</strong>
                    <small>{key.id}</small>
                  </td>
                  <td>
                    <span className={`key-status ${key.revoked ? "revoked" : "active"}`}>
                      {key.revoked ? t("revoked") : t("active")}
                    </span>
                  </td>
                  <td>{key.expires_at ? new Date(key.expires_at).toLocaleDateString() : "—"}</td>
                  <td>{key.last_used_at ? new Date(key.last_used_at).toLocaleString() : "—"}</td>
                  <td>
                    {!key.revoked && (
                      <button
                        className="icon-button compact danger"
                        type="button"
                        aria-label={`${t("revoke")} ${key.name}`}
                        title={`${t("revoke")} ${key.name}`}
                        onClick={() => void revoke(key)}
                      >
                        <IconDeleteStroked />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </section>
  );
}
