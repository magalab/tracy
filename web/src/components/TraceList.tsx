import type { TraceSummary } from "../types";
import { usePreferences } from "../i18n";
import Button from "@douyinfe/semi-ui/lib/es/button";
import DatePicker from "@douyinfe/semi-ui/lib/es/datePicker";
import Empty from "@douyinfe/semi-ui/lib/es/empty";
import Input from "@douyinfe/semi-ui/lib/es/input";
import Select from "@douyinfe/semi-ui/lib/es/select";
import Spin from "@douyinfe/semi-ui/lib/es/spin";
import Table from "@douyinfe/semi-ui/lib/es/table";
import IconRefresh from "@douyinfe/semi-icons/lib/es/icons/IconRefresh";
import IconSearch from "@douyinfe/semi-icons/lib/es/icons/IconSearch";

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

function toDateTimeString(value: unknown) {
  if (value instanceof Date) {
    const year = value.getFullYear();
    const month = String(value.getMonth() + 1).padStart(2, "0");
    const day = String(value.getDate()).padStart(2, "0");
    const hours = String(value.getHours()).padStart(2, "0");
    const minutes = String(value.getMinutes()).padStart(2, "0");
    const seconds = String(value.getSeconds()).padStart(2, "0");
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
  }
  return typeof value === "string" ? value : "";
}

function formatTraceDuration(milliseconds: number) {
  if (milliseconds < 1) return "<1ms";
  if (milliseconds < 1_000) return `${Math.round(milliseconds)}ms`;
  if (milliseconds < 60_000) return `${(milliseconds / 1_000).toFixed(1)}s`;
  const minutes = Math.floor(milliseconds / 60_000);
  const seconds = Math.floor((milliseconds % 60_000) / 1_000);
  return `${minutes}m ${String(seconds).padStart(2, "0")}s`;
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
        formatTraceDuration(
          new Date(trace.end_time).getTime() - new Date(trace.start_time).getTime(),
        ),
    },
  ];
  return (
    <section className="list-panel">
      <div className="panel-heading">
        <div>
          <h2>{t("recentTraces")}</h2>
        </div>
        <Button
          aria-label={t("refresh")}
          className="ghost"
          icon={<IconRefresh />}
          theme="borderless"
          title={t("refresh")}
          type="tertiary"
          onClick={onRefresh}
          disabled={loading}
        />
      </div>
      <div className="toolbar">
        <Input
          aria-label={t("filterByTraceID")}
          className="trace-filter-search"
          prefix={<IconSearch />}
          value={filter}
          onChange={onFilterChange}
          placeholder={t("searchTraceID")}
        />
        <DatePicker
          aria-label={t("startDate")}
          className="trace-filter-date"
          format="yyyy-MM-dd HH:mm:ss"
          type="dateTimeRange"
          timePickerOpts={{ format: "HH:mm:ss" }}
          placeholder={[t("startDate"), t("endDate")]}
          rangeSeparator="→"
          value={startDate || endDate ? [startDate, endDate] : undefined}
          onChange={(value) => {
            const values = Array.isArray(value) ? value : [];
            onStartDateChange(toDateTimeString(values[0]));
            onEndDateChange(toDateTimeString(values[1]));
          }}
        />
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
      {error && (
        <div className="error" role="alert">
          <span className="error-icon">!</span>
          <span>{error === "Failed to fetch" ? t("serviceUnavailable") : error}</span>
        </div>
      )}
      {!token && (
        <div className="empty">
          <strong>{t("connectWorkspace")}</strong>
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
          size="small"
          onRow={(trace) =>
            trace
              ? {
                  className: selectedID === trace.trace_id ? "active" : "",
                  tabIndex: 0,
                  onClick: () => onOpen(trace.trace_id),
                  onKeyDown: (event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      onOpen(trace.trace_id);
                    }
                  },
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
