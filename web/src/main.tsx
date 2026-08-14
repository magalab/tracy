import { StrictMode, useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

type TraceSummary = { project_id: string; trace_id: string; start_time: string; end_time: string; span_count: number; status: string; input_tokens: number; output_tokens: number };
type Span = { project_id: string; trace_id: string; span_id: string; parent_span_id?: string; name: string; kind: string; start_time: string; duration: number; status: string; input?: string; output?: string; attributes?: Record<string, unknown> };
type Page = { items: TraceSummary[]; next_cursor?: string };

const tokenKey = "tracy.api_token";
async function request<T>(path: string, token: string): Promise<T> {
  const response = await fetch(path, { headers: { Authorization: `Bearer ${token}` } });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error?.message ?? `Request failed (${response.status})`);
  return data as T;
}
function formatDuration(nanoseconds: number) { const ms = nanoseconds / 1_000_000; return ms < 1 ? `${Math.round(nanoseconds / 1_000)}μs` : `${ms.toFixed(2)}ms`; }
function App() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenKey) ?? "");
  const [draftToken, setDraftToken] = useState(token);
  const [page, setPage] = useState<Page>({ items: [] });
  const [selected, setSelected] = useState<Span[]>([]);
  const [selectedID, setSelectedID] = useState("");
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [cursor, setCursor] = useState<string | undefined>();
  const visibleItems = useMemo(() => page.items.filter((item) => item.trace_id.includes(filter)), [page.items, filter]);
  async function loadPage(nextCursor?: string) { if (!token) return; setLoading(true); setError(""); try { const suffix = nextCursor ? `&cursor=${encodeURIComponent(nextCursor)}` : ""; const next = await request<Page>(`/api/v1/traces?limit=50${suffix}`, token); setPage(next); setCursor(next.next_cursor); } catch (err) { setError(err instanceof Error ? err.message : String(err)); } finally { setLoading(false); } }
  async function openTrace(id: string) { setSelectedID(id); setError(""); try { const result = await request<{ spans: Span[] }>(`/api/v1/traces/${encodeURIComponent(id)}`, token); setSelected(result.spans); } catch (err) { setError(err instanceof Error ? err.message : String(err)); } }
  function saveToken() { localStorage.setItem(tokenKey, draftToken); setToken(draftToken); setSelected([]); setSelectedID(""); }
  useEffect(() => { if (token) void loadPage(); }, [token]);
  return <div className="app-shell">
    <header className="topbar"><div><span className="eyebrow">SELF-HOSTED OBSERVABILITY</span><h1>Tracy <span>Trace Explorer</span></h1></div><div className="token-form"><input value={draftToken} onChange={(e) => setDraftToken(e.target.value)} type="password" placeholder="API key" onKeyDown={(e) => e.key === "Enter" && saveToken()} /><button onClick={saveToken}>Connect</button></div></header>
    <main className="workspace">
      <section className="list-panel"><div className="panel-heading"><div><span className="eyebrow">PROJECT / DEFAULT</span><h2>Recent traces</h2></div><button className="ghost" onClick={() => void loadPage()} disabled={loading}>↻ Refresh</button></div><div className="toolbar"><input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="Filter by trace ID…" /><span>{visibleItems.length} traces</span></div>{error && <div className="error">{error}</div>}{!token && <div className="empty"><strong>Connect a project</strong><p>Enter an API key above to inspect traces.</p></div>}{token && loading && <div className="empty">Loading traces…</div>}{token && !loading && visibleItems.length === 0 && <div className="empty"><strong>No traces yet</strong><p>Send a span to the ingest endpoint and refresh.</p></div>}<div className="trace-list">{visibleItems.map((trace) => <button className={`trace-row ${selectedID === trace.trace_id ? "active" : ""}`} key={trace.trace_id} onClick={() => void openTrace(trace.trace_id)}><span className={`status-dot ${trace.status}`} /><span className="trace-main"><strong>{trace.trace_id}</strong><small>{new Date(trace.start_time).toLocaleString()} · {trace.span_count} spans</small></span><span className="trace-meta"><b>{trace.status}</b><small>{new Date(trace.end_time).getTime() - new Date(trace.start_time).getTime()}ms</small></span></button>)}</div>{cursor && <button className="load-more" onClick={() => void loadPage(cursor)}>Load older traces</button>}</section>
      <section className="detail-panel">{selectedID ? <><div className="detail-heading"><div><span className="eyebrow">TRACE DETAIL</span><h2>{selectedID}</h2></div><span className="pill">{selected.length} spans</span></div><div className="span-tree">{selected.map((span) => <article className="span-card" key={span.span_id}><div className="span-line"><span className="tree-mark">{span.parent_span_id ? "└" : "●"}</span><span className={`status-dot ${span.status}`} /><strong>{span.name}</strong><span className="kind">{span.kind || "custom"}</span><span className="duration">{formatDuration(span.duration)}</span></div><div className="span-subline">{span.span_id} · {new Date(span.start_time).toLocaleTimeString()}</div>{(span.input || span.output) && <div className="io-grid">{span.input && <div><label>INPUT</label><pre>{span.input}</pre></div>}{span.output && <div><label>OUTPUT</label><pre>{span.output}</pre></div>}</div>}{span.attributes && Object.keys(span.attributes).length > 0 && <details><summary>Attributes ({Object.keys(span.attributes).length})</summary><pre>{JSON.stringify(span.attributes, null, 2)}</pre></details>}</article>)}</div></> : <div className="detail-empty"><div className="orbit">✦</div><h2>Select a trace</h2><p>Choose a trace from the list to inspect its span tree, timing, inputs and outputs.</p></div>}</section>
    </main>
    <footer><span>TRACY / LOCAL MODE</span><span>SQLite trace store · API v1</span></footer>
  </div>;
}
createRoot(document.getElementById("root")!).render(<StrictMode><App /></StrictMode>);
