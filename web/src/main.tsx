import { StrictMode, useState } from "react";
import { createRoot } from "react-dom/client";
import { AnnotationDraft } from "./components/AnnotationPanel";
import { MetricsOverview } from "./components/MetricsOverview";
import { TokenForm } from "./components/TokenForm";
import { TraceDetail } from "./components/TraceDetail";
import { TraceList } from "./components/TraceList";
import { useTraceExplorer } from "./hooks/useTraceExplorer";
import { PreferencesProvider, usePreferences } from "./i18n";
import "./styles.css";

const tokenKey = "tracy.api_token";

function App() {
  const { language, theme, t, toggleLanguage, toggleTheme } = usePreferences();
  const [token, setToken] = useState(() => localStorage.getItem(tokenKey) ?? "");
  const [draftToken, setDraftToken] = useState(token);
  const [filter, setFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [kindFilter, setKindFilter] = useState("");
  const [overviewOpen, setOverviewOpen] = useState(false);
  const [annotationDraft, setAnnotationDraft] = useState<AnnotationDraft>({
    key: "quality",
    score: "1",
    label: "",
    comment: "",
  });
  const explorer = useTraceExplorer(token, statusFilter, kindFilter);

  function saveToken() {
    localStorage.setItem(tokenKey, draftToken);
    setToken(draftToken);
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

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand-lockup">
          <img className="brand-mark" src="/tracy-mark.svg" alt="Tracy" />
          <div>
            <h1>
              Tracy <span>{t("traceExplorer")}</span>
            </h1>
          </div>
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
          <TokenForm
            value={draftToken}
            connected={Boolean(token)}
            onChange={setDraftToken}
            onSave={saveToken}
          />
        </div>
      </header>
      <main className="main-content">
        <section className={`overview-header ${overviewOpen ? "expanded" : "collapsed"}`}>
          <div>
            <span className="eyebrow">{t("operations")}</span>
            <h2>{t("traceHealth")}</h2>
            <p>{t("inspectHealth")}</p>
          </div>
          <div className="overview-actions">
            <div className="live-indicator">
              <span className="live-dot" /> {t("liveStore")}
            </div>
            <button
              className="overview-toggle"
              onClick={() => setOverviewOpen((open) => !open)}
              aria-label={overviewOpen ? t("hideOverview") : t("showOverview")}
              title={overviewOpen ? t("hideOverview") : t("showOverview")}
            >
              {overviewOpen ? "⌃" : "⌄"}
            </button>
          </div>
        </section>
        {overviewOpen && explorer.metrics && <MetricsOverview metrics={explorer.metrics} />}
        <section className="workspace">
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
          <TraceDetail
            selectedID={explorer.selectedID}
            selected={explorer.selected}
            annotations={explorer.annotations}
            draft={annotationDraft}
            onDraftChange={setAnnotationDraft}
            onAddAnnotation={addAnnotation}
            onDeleteAnnotation={(id) => void explorer.removeAnnotation(id)}
          />
        </section>
      </main>
      <footer>
        <span>{t("localMode")}</span>
        <span>{t("sqliteStore")}</span>
      </footer>
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
