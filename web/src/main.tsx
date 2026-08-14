import { StrictMode, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { AnnotationDraft } from "./components/AnnotationPanel";
import { AppSidebar } from "./components/AppSidebar";
import { LoginPage } from "./components/LoginPage";
import { TraceDetail } from "./components/TraceDetail";
import { TraceList } from "./components/TraceList";
import { WorkspacePicker } from "./components/WorkspacePicker";
import {
  getCurrentUser,
  listWorkspaces,
  login as loginRequest,
  createWorkspace,
  switchWorkspace,
} from "./api/client";
import { useTraceExplorer } from "./hooks/useTraceExplorer";
import { PreferencesProvider, usePreferences } from "./i18n";
import type { User, Workspace } from "./types";
import "./styles.css";

const sessionKey = "tracy.session_token";
const userKey = "tracy.user";

function App() {
  const { language, theme, t, toggleLanguage, toggleTheme } = usePreferences();
  const [token, setToken] = useState(() => localStorage.getItem(sessionKey) ?? "");
  const [user, setUser] = useState<User | null>(() => {
    const raw = localStorage.getItem(userKey);
    if (!raw) return null;
    try {
      return JSON.parse(raw) as User;
    } catch {
      return null;
    }
  });
  const [authReady, setAuthReady] = useState(false);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [activeWorkspaceID, setActiveWorkspaceID] = useState("");
  const [workspacesReady, setWorkspacesReady] = useState(false);
  const [filter, setFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [kindFilter, setKindFilter] = useState("");
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [annotationDraft, setAnnotationDraft] = useState<AnnotationDraft>({
    key: "quality",
    score: "1",
    label: "",
    comment: "",
  });
  const explorer = useTraceExplorer(token, statusFilter, kindFilter);

  useEffect(() => {
    if (!token) {
      setUser(null);
      setAuthReady(true);
      setWorkspacesReady(true);
      return;
    }
    void getCurrentUser(token)
      .then((data) => {
        setUser(data.user);
        setAuthReady(true);
      })
      .catch(() => {
        localStorage.removeItem(sessionKey);
        localStorage.removeItem(userKey);
        setToken("");
        setAuthReady(true);
      });
  }, [token]);

  useEffect(() => {
    if (!user || !token) return;
    setWorkspacesReady(false);
    void listWorkspaces(token)
      .then((data) => {
        setWorkspaces(data.items);
        setActiveWorkspaceID(data.active_id);
      })
      .finally(() => setWorkspacesReady(true));
  }, [token, user]);

  async function login(email: string, password: string) {
    const data = await loginRequest(email, password);
    localStorage.setItem(sessionKey, data.access_token);
    localStorage.setItem(userKey, JSON.stringify(data.user));
    setToken(data.access_token);
    setUser(data.user);
    setAuthReady(true);
  }

  function logout() {
    localStorage.removeItem(sessionKey);
    localStorage.removeItem(userKey);
    setToken("");
    setUser(null);
    setWorkspaces([]);
    setActiveWorkspaceID("");
  }

  async function createUserWorkspace(name: string) {
    const data = await createWorkspace(token, name);
    setWorkspaces((items) => [...items, data.workspace]);
    setActiveWorkspaceID(data.active_id);
  }

  async function selectWorkspace(id: string) {
    await switchWorkspace(token, id);
    setActiveWorkspaceID(id);
    void explorer.loadPage();
    void explorer.loadMetrics();
  }

  function clearFilters() {
    setFilter("");
    setStatusFilter("");
    setKindFilter("");
  }

  function addAnnotation() {
    void explorer.addAnnotation({
      key: annotationDraft.key,
      score: Number(annotationDraft.score),
      label: annotationDraft.label,
      comment: annotationDraft.comment,
    });
    setAnnotationDraft((draft) => ({ ...draft, label: "", comment: "" }));
  }

  const activeWorkspace = workspaces.find((workspace) => workspace.id === activeWorkspaceID);

  if (!authReady) return <div className="page-loading">{t("loading")}</div>;
  if (!user) return <LoginPage onLogin={login} />;
  if (!workspacesReady) return <div className="page-loading">{t("loading")}</div>;
  if (!activeWorkspace) {
    return (
      <WorkspacePicker
        workspaces={workspaces}
        activeID={activeWorkspaceID}
        onSelect={selectWorkspace}
        onCreate={createUserWorkspace}
      />
    );
  }

  return (
    <div className="app-shell">
      <AppSidebar
        user={user}
        workspaces={workspaces}
        activeID={activeWorkspaceID}
        onSelectWorkspace={selectWorkspace}
        onCreateWorkspace={createUserWorkspace}
        onLogout={logout}
        collapsed={sidebarCollapsed}
        onToggle={() => setSidebarCollapsed((collapsed) => !collapsed)}
        metrics={explorer.metrics}
      />
      <div className="app-main">
        <header className="topbar">
          <div className="trace-context">
            <span className="eyebrow">{t("trace")}</span>
            <h1>{t("traceExplorer")}</h1>
          </div>
          <div className="topbar-actions">
            <div className="preference-actions">
              <button className="preference-button" onClick={toggleLanguage}>
                {language === "en" ? "中" : "EN"}
              </button>
              <button
                className="preference-button"
                onClick={toggleTheme}
                aria-label={theme === "dark" ? t("theme") : t("darkTheme")}
                title={theme === "dark" ? t("theme") : t("darkTheme")}
              >
                {theme === "dark" ? "☼" : "☾"}
              </button>
            </div>
          </div>
        </header>
        <main className="main-content">
          {explorer.selectedID ? (
            <TraceDetail
              selectedID={explorer.selectedID}
              selected={explorer.selected}
              annotations={explorer.annotations}
              draft={annotationDraft}
              onDraftChange={setAnnotationDraft}
              onAddAnnotation={addAnnotation}
              onDeleteAnnotation={(id) => void explorer.removeAnnotation(id)}
              onBack={explorer.clearSelection}
            />
          ) : (
            <TraceList
              traces={explorer.page.items}
              selectedID={explorer.selectedID}
              filter={filter}
              statusFilter={statusFilter}
              kindFilter={kindFilter}
              loading={explorer.loading}
              token={token}
              error={explorer.error}
              cursor={explorer.cursor}
              onFilterChange={setFilter}
              onStatusChange={setStatusFilter}
              onKindChange={setKindFilter}
              onOpen={(id) => void explorer.openTrace(id)}
              onRefresh={() => {
                void explorer.loadPage();
                void explorer.loadMetrics();
              }}
              onLoadMore={(nextCursor) => void explorer.loadPage(nextCursor)}
              onClearFilters={clearFilters}
            />
          )}
        </main>
        <footer>
          <span>{t("localMode")}</span>
          <span>{t("sqliteStore")}</span>
        </footer>
      </div>
    </div>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <PreferencesProvider>
      <App />
    </PreferencesProvider>
  </StrictMode>,
);
