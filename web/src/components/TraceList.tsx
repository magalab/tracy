import type { TraceSummary } from "../types";

type TraceListProps = {
  traces: TraceSummary[];
  selectedID: string;
  filter: string;
  statusFilter: string;
  kindFilter: string;
  loading: boolean;
  token: string;
  error: string;
  cursor?: string;
  onFilterChange: (value: string) => void;
  onStatusChange: (value: string) => void;
  onKindChange: (value: string) => void;
  onOpen: (id: string) => void;
  onRefresh: () => void;
  onLoadMore: (cursor: string) => void;
  onClearFilters: () => void;
};

export function TraceList({
  traces,
  selectedID,
  filter,
  statusFilter,
  kindFilter,
  loading,
  token,
  error,
  cursor,
  onFilterChange,
  onStatusChange,
  onKindChange,
  onOpen,
  onRefresh,
  onLoadMore,
  onClearFilters,
}: TraceListProps) {
  const visibleItems = traces.filter((item) =>
    item.trace_id.toLowerCase().includes(filter.toLowerCase()),
  );

  return (
    <section className="list-panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">PROJECT / DEFAULT</span>
          <h2>Recent traces</h2>
        </div>
        <button className="ghost" onClick={onRefresh} disabled={loading}>
          ↻ Refresh
        </button>
      </div>
      <div className="toolbar">
        <div className="search-field">
          <span>⌕</span>
          <input
            aria-label="Filter by trace ID"
            value={filter}
            onChange={(event) => onFilterChange(event.target.value)}
            placeholder="Search trace ID…"
          />
        </div>
        <select
          aria-label="Filter by status"
          value={statusFilter}
          onChange={(event) => onStatusChange(event.target.value)}
        >
          <option value="">All status</option>
          <option value="ok">Healthy</option>
          <option value="error">Errors</option>
        </select>
        <select
          aria-label="Filter by kind"
          value={kindFilter}
          onChange={(event) => onKindChange(event.target.value)}
        >
          <option value="">All kinds</option>
          <option value="llm">LLM</option>
          <option value="tool">Tool</option>
          <option value="agent">Agent</option>
        </select>
      </div>
      <div className="list-caption">
        <span>{visibleItems.length} traces in view</span>
        <span>{statusFilter || kindFilter ? "Filtered results" : "Newest first"}</span>
      </div>
      {error && <div className="error">{error}</div>}
      {!token && (
        <div className="empty">
          <strong>Connect a project</strong>
          <p>Enter an API key above to inspect traces.</p>
        </div>
      )}
      {token && loading && <div className="empty">Loading traces…</div>}
      {token && !loading && visibleItems.length === 0 && (
        <div className="empty">
          <strong>No traces match</strong>
          <p>Try clearing the filters or send a span to the ingest endpoint.</p>
          <button className="ghost" onClick={onClearFilters}>
            Clear filters
          </button>
        </div>
      )}
      <div className="trace-list">
        {visibleItems.map((trace) => (
          <button
            className={`trace-row ${selectedID === trace.trace_id ? "active" : ""}`}
            key={trace.trace_id}
            aria-label={`Open trace ${trace.trace_id}`}
            onClick={() => onOpen(trace.trace_id)}
          >
            <span className={`status-dot ${trace.status}`} />
            <span className="trace-main">
              <div className="trace-title">
                <strong>{trace.trace_id}</strong>
                <span className={`status-badge ${trace.status}`}>
                  {trace.status === "error" ? "Error" : "Healthy"}
                </span>
              </div>
              <small>{new Date(trace.start_time).toLocaleString()}</small>
              <div className="trace-facts">
                <span>{trace.span_count} spans</span>
                <span>{(trace.input_tokens + trace.output_tokens).toLocaleString()} tokens</span>
              </div>
            </span>
            <span className="trace-meta">
              <b>
                {(
                  new Date(trace.end_time).getTime() - new Date(trace.start_time).getTime()
                ).toFixed(0)}
                ms
              </b>
              <small>open trace&nbsp; →</small>
            </span>
          </button>
        ))}
      </div>
      {cursor && (
        <button className="load-more" onClick={() => onLoadMore(cursor)}>
          Load older traces
        </button>
      )}
    </section>
  );
}
