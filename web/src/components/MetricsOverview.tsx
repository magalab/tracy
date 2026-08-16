import type { DashboardMetrics } from "../types";
import { usePreferences } from "../i18n";

export function MetricsOverview({
  metrics,
  compact = false,
}: {
  metrics: DashboardMetrics;
  compact?: boolean;
}) {
  const { t } = usePreferences();
  return (
    <section className={`overview-grid ${compact ? "compact" : ""}`}>
      <article className="metric-card metric-card-primary">
        <span className="metric-label">{t("requests")}</span>
        <strong>{metrics.request_count.toLocaleString()}</strong>
        <small>{t("tracesObserved")}</small>
      </article>
      <article className="metric-card">
        <span className="metric-label">{t("errorRate")}</span>
        <strong>
          {(metrics.error_rate * 100).toFixed(1)}
          <em>%</em>
        </strong>
        <small>
          {metrics.error_count.toLocaleString()} {t("failedTraces")}
        </small>
        <span className={`metric-status ${metrics.error_count ? "warning" : "good"}`}>
          {metrics.error_count ? t("needsAttention") : t("healthy")}
        </span>
      </article>
      <article className="metric-card">
        <span className="metric-label">{t("p95Latency")}</span>
        <strong>
          {metrics.p95_latency_ms.toFixed(1)}
          <em>ms</em>
        </strong>
        <small>{t("endToEnd")}</small>
        <span className="metric-status neutral">{t("tailPerformance")}</span>
      </article>
      <article className="metric-card">
        <span className="metric-label">{t("tokenVolume")}</span>
        <strong>{(metrics.input_tokens + metrics.output_tokens).toLocaleString()}</strong>
        <small>
          {metrics.input_tokens.toLocaleString()} in · {metrics.output_tokens.toLocaleString()} out
        </small>
        <span className="metric-status neutral">{t("acrossTraces")}</span>
      </article>
    </section>
  );
}
