import type { TraceSummary } from "../types";
import { usePreferences } from "../i18n";
import { Button, DatePicker, Empty, Select, Spin, Table } from "@douyinfe/semi-ui";

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

function toDateString(value: unknown) {
  if (value instanceof Date) {
    const year = value.getFullYear();
    const month = String(value.getMonth() + 1).padStart(2, "0");
    const day = String(value.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
  }
  return typeof value === "string" ? value.slice(0, 10) : "";
}

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
  const columns = [
    {
      title: "TraceID",
      dataIndex: "trace_id",
      key: "trace_id",
      width: "34%",
      render: (_: string, trace: TraceSummary) => (
        <div className="trace-table-id">
          <strong>{trace.trace_id}</strong>
          <span className={`status-badge ${trace.status}`}>
            {trace.status === "error" ? t("errors") : t("healthyStatus")}
          </span>
        </div>
      ),
    },
    {
      title: t("startTime"),
      dataIndex: "start_time",
      key: "start_time",
      width: "22%",
      render: (value: string) => new Date(value).toLocaleString(),
    },
    {
      title: t("spans"),
      dataIndex: "span_count",
      key: "span_count",
      align: "right" as const,
    },
    {
      title: t("tokens"),
      key: "tokens",
      align: "right" as const,
      render: (_: unknown, trace: TraceSummary) =>
        (trace.input_tokens + trace.output_tokens).toLocaleString(),
    },
    {
      title: t("duration"),
      key: "duration",
      align: "right" as const,
      render: (_: unknown, trace: TraceSummary) =>
        `${(new Date(trace.end_time).getTime() - new Date(trace.start_time).getTime()).toFixed(0)}ms`,
    },
  ];
  return (
    <section className="list-panel">
      <div className="panel-heading">
        <div>
          <span className="eyebrow">{t("projectDefault")}</span>
          <h2>{t("recentTraces")}</h2>
        </div>
        <Button
          className="ghost"
          theme="borderless"
          type="tertiary"
          onClick={onRefresh}
          disabled={loading}
        >
          {t("refresh")}
        </Button>
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
        <Select
          aria-label={t("allStatus")}
          className="trace-filter-select"
          placeholder={t("allStatus")}
          showClear
          value={statusFilter || undefined}
          onChange={(value) => onStatusChange(String(value ?? ""))}
        >
          <Select.Option value="ok">{t("healthyStatus")}</Select.Option>
          <Select.Option value="error">{t("errors")}</Select.Option>
        </Select>
        <DatePicker
          aria-label={t("startDate")}
          className="trace-filter-date"
          format="yyyy-MM-dd"
          placeholder={t("startDate")}
          value={startDate || undefined}
          onChange={(value) => onStartDateChange(toDateString(value))}
        />
        <DatePicker
          aria-label={t("endDate")}
          className="trace-filter-date"
          format="yyyy-MM-dd"
          placeholder={t("endDate")}
          value={endDate || undefined}
          onChange={(value) => onEndDateChange(toDateString(value))}
        />
        <Select
          aria-label={t("allKinds")}
          className="trace-filter-select"
          placeholder={t("allKinds")}
          showClear
          value={kindFilter || undefined}
          onChange={(value) => onKindChange(String(value ?? ""))}
        >
          <Select.Option value="llm">LLM</Select.Option>
          <Select.Option value="tool">Tool</Select.Option>
          <Select.Option value="agent">Agent</Select.Option>
        </Select>
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
      {token && loading && (
        <div className="table-state">
          <Spin tip={t("loadingTraces")} />
        </div>
      )}
      {token && !loading && visibleItems.length === 0 && (
        <div className="table-state">
          <Empty description={t("noTracesMatch")} />
          <p>{t("clearFiltersHint")}</p>
          <Button className="ghost" theme="borderless" type="tertiary" onClick={onClearFilters}>
            {t("clearFilters")}
          </Button>
        </div>
      )}
      {!loading && visibleItems.length > 0 && (
        <Table<TraceSummary>
          className="trace-table"
          columns={columns}
          dataSource={visibleItems}
          empty={<Empty description={t("noTracesMatch")} />}
          pagination={false}
          rowKey="trace_id"
          onRow={(trace) =>
            trace
              ? {
                  className: selectedID === trace.trace_id ? "active" : "",
                  onClick: () => onOpen(trace.trace_id),
                }
              : {}
          }
        />
      )}
      {cursor && (
        <Button
          className="load-more"
          theme="borderless"
          type="tertiary"
          onClick={() => onLoadMore(cursor)}
        >
          {t("loadOlder")}
        </Button>
      )}
    </section>
  );
}
