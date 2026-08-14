import type { DashboardMetrics } from "../types";

export function MetricsOverview({ metrics }: { metrics: DashboardMetrics }) {
  return (
    <section className="overview-grid">
      <article className="metric-card metric-card-primary">
        <span className="metric-label">Requests</span>
        <strong>{metrics.request_count.toLocaleString()}</strong>
        <small>traces observed</small>
        <span className="metric-mark">↗</span>
      </article>
      <article className="metric-card">
        <span className="metric-label">Error rate</span>
        <strong>
          {(metrics.error_rate * 100).toFixed(1)}
          <em>%</em>
        </strong>
        <small>{metrics.error_count.toLocaleString()} failed traces</small>
        <span className={`metric-status ${metrics.error_count ? "warning" : "good"}`}>
          {metrics.error_count ? "needs attention" : "healthy"}
        </span>
      </article>
      <article className="metric-card">
        <span className="metric-label">P95 latency</span>
        <strong>
          {metrics.p95_latency_ms.toFixed(1)}
          <em>ms</em>
        </strong>
        <small>end-to-end trace duration</small>
        <span className="metric-status neutral">tail performance</span>
      </article>
      <article className="metric-card">
        <span className="metric-label">Token volume</span>
        <strong>{(metrics.input_tokens + metrics.output_tokens).toLocaleString()}</strong>
        <small>
          {metrics.input_tokens.toLocaleString()} in · {metrics.output_tokens.toLocaleString()} out
        </small>
        <span className="metric-status neutral">across all traces</span>
      </article>
    </section>
  );
}
