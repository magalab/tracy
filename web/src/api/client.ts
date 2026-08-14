import type { Annotation, DashboardMetrics, Page, Span } from "../types";

export async function request<T>(path: string, token: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      ...init?.headers,
    },
  });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error?.message ?? `Request failed (${response.status})`);
  return data as T;
}

export function listTraces(
  token: string,
  options: { cursor?: string; status?: string; kind?: string },
) {
  const params = new URLSearchParams({ limit: "50" });
  if (options.cursor) params.set("cursor", options.cursor);
  if (options.status) params.set("status", options.status);
  if (options.kind) params.set("kind", options.kind);
  return request<Page>(`/api/v1/traces?${params.toString()}`, token);
}

export function getDashboardMetrics(token: string) {
  return request<DashboardMetrics>("/api/v1/dashboard", token);
}

export async function getTrace(id: string, token: string) {
  const [trace, annotations] = await Promise.all([
    request<{ spans: Span[] }>(`/api/v1/traces/${encodeURIComponent(id)}`, token),
    request<{ items: Annotation[] }>(`/api/v1/traces/${encodeURIComponent(id)}/annotations`, token),
  ]);
  return { spans: trace.spans, annotations: annotations.items };
}

export function createAnnotation(
  traceID: string,
  token: string,
  input: { key: string; score: number; label: string; comment: string },
) {
  return request<Annotation>(`/api/v1/traces/${encodeURIComponent(traceID)}/annotations`, token, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}

export function deleteAnnotation(id: string, token: string) {
  return request<void>(`/api/v1/annotations/${encodeURIComponent(id)}`, token, {
    method: "DELETE",
  });
}
