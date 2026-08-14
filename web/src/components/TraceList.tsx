import type { TraceSummary } from "../types";
import { usePreferences } from "../i18n";

type TraceListProps = {
  traces: TraceSummary[];
  selectedID: string;
  filter: string;
  statusFilter: string;
  kindFilter: string;
  startDate: string;
  endDate: string;
  loading: boolean;
  token: string;
  error: string;
  cursor?: string;
  onFilterChange: (value: string) => void;
  onStatusChange: (value: string) => void;
  onKindChange: (value: string) => void;
  onStartDateChange: (value: string) => void;
  onEndDateChange: (value: string) => void;
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
  startDate,
  endDate,
  loading,
  token,
  error,
  cursor,
  onFilterChange,
  onStatusChange,
  onKindChange,
  onStartDateChange,
  onEndDateChange,
  onOpen,
  onRefresh,
  onLoadMore,
  onClearFilters,
}: TraceListProps) {
  const { t } = usePreferences();
  const visibleItems = traces.filter((item) =>
    item.trace_id.toLowerCase().includes(filter.toLowerCase()),
  );

  return (
    <section className="list-panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">{t("projectDefault")}</span>
          <h2>{t("recentTraces")}</h2>
        </div>
        <button className="ghost" onClick={onRefresh} disabled={loading}>
          {t("refresh")}
        </button>
      </div>
      <div className="toolbar">
        <div className="search-field">
          <span>⌕</span>
          <input
            aria-label={t("filterByTraceID")}
            value={filter}
            onChange={(event) => onFilterChange(event.target.value)}
            placeholder={t("searchTraceID")}
          />
        </div>
        <select
          aria-label="Filter by status"
          value={statusFilter}
          onChange={(event) => onStatusChange(event.target.value)}
        >
          <option value="">{t("allStatus")}</option>
          <option value="ok">{t("healthyStatus")}</option>
          <option value="error">{t("errors")}</option>
        </select>
        <label className="date-filter">
          <span>{t("startDate")}</span>
          <input
            aria-label={t("startDate")}
            type="date"
            value={startDate}
            onClick={(event) => event.currentTarget.showPicker?.()}
            onChange={(event) => onStartDateChange(event.target.value)}
          />
        </label>
        <label className="date-filter">
          <span>{t("endDate")}</span>
          <input
            aria-label={t("endDate")}
            type="date"
            value={endDate}
            onClick={(event) => event.currentTarget.showPicker?.()}
            onChange={(event) => onEndDateChange(event.target.value)}
          />
        </label>
        <select
          aria-label="Filter by kind"
          value={kindFilter}
          onChange={(event) => onKindChange(event.target.value)}
        >
          <option value="">{t("allKinds")}</option>
          <option value="llm">LLM</option>
          <option value="tool">Tool</option>
          <option value="agent">Agent</option>
        </select>
      </div>
      <div className="list-caption">
        <span>
          {visibleItems.length} {t("tracesInView")}
        </span>
        <span>{statusFilter || kindFilter ? t("filteredResults") : t("newestFirst")}</span>
      </div>
      {error && <div className="error">{error}</div>}
      {!token && (
        <div className="empty">
          <strong>{t("connectProject")}</strong>
          <p>{t("connectHint")}</p>
        </div>
      )}
      {token && loading && <div className="empty">{t("loadingTraces")}</div>}
      {token && !loading && visibleItems.length === 0 && (
        <div className="empty">
          <strong>{t("noTracesMatch")}</strong>
          <p>{t("clearFiltersHint")}</p>
          <button className="ghost" onClick={onClearFilters}>
            {t("clearFilters")}
          </button>
        </div>
      )}
      <div className="trace-list">
        {visibleItems.map((trace) => (
          <button
            className={`trace-row ${selectedID === trace.trace_id ? "active" : ""}`}
            key={trace.trace_id}
            aria-label={`${t("openTrace")} ${trace.trace_id}`}
            onClick={() => onOpen(trace.trace_id)}
          >
            <span className="trace-main">
              <div className="trace-title">
                <strong>{trace.trace_id}</strong>
                <span className={`status-badge ${trace.status}`}>
                  {trace.status === "error" ? t("errors") : t("healthyStatus")}
                </span>
              </div>
              <small>{new Date(trace.start_time).toLocaleString()}</small>
              <div className="trace-facts">
                <span>
                  {trace.span_count} {t("spans")}
                </span>
                <span>
                  {(trace.input_tokens + trace.output_tokens).toLocaleString()} {t("tokens")}
                </span>
              </div>
            </span>
            <span className="trace-meta">
              <b>
                {(
                  new Date(trace.end_time).getTime() - new Date(trace.start_time).getTime()
                ).toFixed(0)}
                ms
              </b>
            </span>
          </button>
        ))}
      </div>
      {cursor && (
        <button className="load-more" onClick={() => onLoadMore(cursor)}>
          {t("loadOlder")}
        </button>
      )}
    </section>
  );
}
