export type TraceSummary = {
  project_id: string;
  trace_id: string;
  start_time: string;
  end_time: string;
  span_count: number;
  status: string;
  input_tokens: number;
  output_tokens: number;
};

export type Span = {
  project_id: string;
  trace_id: string;
  span_id: string;
  parent_span_id?: string;
  name: string;
  kind: string;
  start_time: string;
  duration: number;
  status: string;
  input?: string;
  output?: string;
  attributes?: Record<string, unknown>;
};

export type Annotation = {
  id: string;
  trace_id: string;
  span_id?: string;
  key: string;
  score?: number;
  label?: string;
  comment?: string;
  created_at: string;
};

export type Page = { items: TraceSummary[]; next_cursor?: string };

export type DashboardMetrics = {
  request_count: number;
  error_count: number;
  error_rate: number;
  input_tokens: number;
  output_tokens: number;
  p95_latency_ms: number;
  usage_breakdown?: { key: string; span_count: number }[];
};

export type Workspace = {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
};

export type User = { id: string; email: string; name: string };
