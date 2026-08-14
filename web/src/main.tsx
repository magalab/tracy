import { StrictMode, useState } from "react";
import { createRoot } from "react-dom/client";
import { AnnotationDraft } from "./components/AnnotationPanel";
import { MetricsOverview } from "./components/MetricsOverview";
import { TokenForm } from "./components/TokenForm";
import { TraceDetail } from "./components/TraceDetail";
import { TraceList } from "./components/TraceList";
import { useTraceExplorer } from "./hooks/useTraceExplorer";
import "./styles.css";

const tokenKey = "tracy.api_token";

function App() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenKey) ?? "");
  const [draftToken, setDraftToken] = useState(token);
  const [filter, setFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [kindFilter, setKindFilter] = useState("");
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
        <div>
          <span className="eyebrow">SELF-HOSTED OBSERVABILITY</span>
          <h1>
            Tracy <span>Trace Explorer</span>
          </h1>
        </div>
        <TokenForm value={draftToken} onChange={setDraftToken} onSave={saveToken} />
      </header>
      <main className="main-content">
        <section className="overview-header">
          <div>
            <span className="eyebrow">OPERATIONS / LAST 24 HOURS</span>
            <h2>Trace health at a glance</h2>
            <p>Inspect throughput, failures and latency before drilling into a run.</p>
          </div>
          <div className="live-indicator">
            <span className="live-dot" /> LIVE LOCAL STORE
          </div>
        </section>
        {explorer.metrics && <MetricsOverview metrics={explorer.metrics} />}
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
        <span>TRACY / LOCAL MODE</span>
        <span>SQLite trace store · API v1</span>
      </footer>
    </div>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
