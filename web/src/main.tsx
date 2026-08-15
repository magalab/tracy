import "@douyinfe/semi-ui/react19-adapter";
import { lazy, StrictMode, Suspense, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import Button from "@douyinfe/semi-ui/lib/es/button";
import "@douyinfe/semi-ui/lib/es/_base/base.css";
import { AppSidebar } from "./components/AppSidebar";
import { LoginPage } from "./components/LoginPage";
const MetricsOverview = lazy(() =>
  import("./components/MetricsOverview").then((module) => ({ default: module.MetricsOverview })),
);
const TraceDetail = lazy(() =>
  import("./components/TraceDetail").then((module) => ({ default: module.TraceDetail })),
);
import { TraceList } from "./components/TraceList";
import { WorkspacePicker } from "./components/WorkspacePicker";
import { APIKeysPage } from "./components/APIKeysPage";
import {
  getCurrentUser,
  logout as logoutRequest,
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
  const [workspaceError, setWorkspaceError] = useState("");
  const [filter, setFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [kindFilter, setKindFilter] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [activePage, setActivePage] = useState<"overview" | "traces" | "keys">("traces");
  const explorer = useTraceExplorer(
    token,
    activeWorkspaceID,
    statusFilter,
    kindFilter,
    startDate ? new Date(startDate.replace(" ", "T")).toISOString() : "",
    endDate ? new Date(endDate.replace(" ", "T")).toISOString() : "",
  );

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
    setWorkspaceError("");
    void listWorkspaces(token)
      .then((data) => {
        setWorkspaces(data.items);
        setActiveWorkspaceID((current) =>
          data.items.some((workspace) => workspace.id === current)
            ? current
            : (data.items[0]?.id ?? ""),
        );
      })
      .catch((err) => {
        setWorkspaceError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => setWorkspacesReady(true));
  }, [token, user]);

  async function login(email: string, password: string) {
    const data = await loginRequest(email, password);
    localStorage.setItem(sessionKey, data.access_token);
    localStorage.setItem(userKey, JSON.stringify(data.user));
    setToken(data.access_token);
    setUser(data.user);
    setActiveWorkspaceID(data.workspace.id);
    setAuthReady(true);
  }

  function logout() {
    if (token) void logoutRequest(token).catch(() => undefined);
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
    setActiveWorkspaceID(data.workspace.id);
  }

  async function selectWorkspace(id: string) {
    await switchWorkspace(token, id);
    explorer.clearSelection();
    setActiveWorkspaceID(id);
  }

  function clearFilters() {
    setFilter("");
    setStatusFilter("");
    setKindFilter("");
    setStartDate("");
    setEndDate("");
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
        error={workspaceError}
        onSelect={selectWorkspace}
        onCreate={createUserWorkspace}
      />
    );
  }

  return (
    <div className={`app-shell ${sidebarCollapsed ? "sidebar-is-collapsed" : ""}`}>
      <AppSidebar
        user={user}
        workspaces={workspaces}
        activeID={activeWorkspaceID}
        onSelectWorkspace={selectWorkspace}
        onCreateWorkspace={createUserWorkspace}
        onLogout={logout}
        collapsed={sidebarCollapsed}
        onToggle={() => setSidebarCollapsed((collapsed) => !collapsed)}
        activePage={activePage}
        onPageChange={(page) => {
          setActivePage(page);
          if (page !== "traces") explorer.clearSelection();
        }}
      />
      <div className="app-main">
        <header className="topbar">
          <div className="trace-context">
            <span className="eyebrow">
              {t(
                activePage === "overview"
                  ? "overview"
                  : activePage === "keys"
                    ? "apiKeys"
                    : "trace",
              )}
            </span>
            <strong className="topbar-workspace">{activeWorkspace.name}</strong>
          </div>
          <div className="topbar-actions">
            <div className="preference-actions">
              <Button
                className="preference-button"
                theme="borderless"
                type="tertiary"
                onClick={toggleLanguage}
              >
                {language === "en" ? "中" : "EN"}
              </Button>
              <Button
                className="preference-button"
                theme="borderless"
                type="tertiary"
                onClick={toggleTheme}
                aria-label={theme === "dark" ? t("theme") : t("darkTheme")}
                title={theme === "dark" ? t("theme") : t("darkTheme")}
              >
                {theme === "dark" ? "☼" : "☾"}
              </Button>
            </div>
          </div>
        </header>
        <main className="main-content">
          <Suspense fallback={<div className="page-loading">{t("loading")}</div>}>
            {activePage === "keys" ? (
              <APIKeysPage token={token} workspaceID={activeWorkspaceID} />
            ) : activePage === "overview" ? (
              <section className="overview-page">
                <div className="overview-page-heading">
                  <span className="eyebrow">{t("operations")}</span>
                  <h2>{t("traceHealth")}</h2>
                  <p>{t("inspectHealth")}</p>
                </div>
                {explorer.metrics && <MetricsOverview metrics={explorer.metrics} />}
              </section>
            ) : (
              <div className="trace-explorer-shell">
                <TraceList
                  traces={explorer.page.items}
                  selectedID={explorer.selectedID}
                  filter={filter}
                  statusFilter={statusFilter}
                  kindFilter={kindFilter}
                  startDate={startDate}
                  endDate={endDate}
                  loading={explorer.loading}
                  token={token}
                  error={explorer.error}
                  cursor={explorer.cursor}
                  onFilterChange={setFilter}
                  onStatusChange={setStatusFilter}
                  onKindChange={setKindFilter}
                  onStartDateChange={setStartDate}
                  onEndDateChange={setEndDate}
                  onOpen={(id) => void explorer.openTrace(id)}
                  onRefresh={() => {
                    void explorer.loadPage();
                    void explorer.loadMetrics();
                  }}
                  onLoadMore={(nextCursor) => void explorer.loadPage(nextCursor)}
                  onClearFilters={clearFilters}
                />
                {explorer.selectedID && (
                  <>
                    <button
                      aria-label={t("backToTraces")}
                      className="trace-detail-backdrop"
                      onClick={explorer.clearSelection}
                      type="button"
                    />
                    <div className="trace-detail-drawer">
                      <TraceDetail
                        selectedID={explorer.selectedID}
                        selected={explorer.selected}
                        hasMore={Boolean(explorer.selectedNextCursor)}
                        loadingMore={explorer.traceLoading}
                        onLoadMore={() => void explorer.loadMoreTrace()}
                        onBack={explorer.clearSelection}
                      />
                    </div>
                  </>
                )}
              </div>
            )}
          </Suspense>
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
