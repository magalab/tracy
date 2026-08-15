export type TraceSummary = {
  workspace_id: string;
  trace_id: string;
  start_time: string;
  end_time: string;
  span_count: number;
  status: string;
  input_tokens: number;
  output_tokens: number;
};

export type Span = {
  workspace_id: string;
  trace_id: string;
  span_id: string;
  parent_span_id?: string;
  name: string;
  kind: string;
  start_time: string;
  duration: number;
  status: string;
  status_message?: string;
  input?: string;
  output?: string;
  input_tokens?: number;
  output_tokens?: number;
  attributes?: Record<string, unknown>;
};

export type Page = { items: TraceSummary[]; next_cursor?: string };

export type DashboardMetrics = {
  request_count: number;
  error_count: number;
  error_rate: number;
  input_tokens: number;
  output_tokens: number;
  p95_latency_ms: number;
  latency_sampled?: boolean;
  usage_breakdown?: { key: string; span_count: number }[];
};

export type Workspace = {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
};

export type User = { id: string; email: string; name: string };

export type APIKey = {
  id: string;
  workspace_id: string;
  name: string;
  role: string;
  expires_at?: string;
  revoked: boolean;
  last_used_at?: string;
};

export type CreatedAPIKey = APIKey & { token: string };
