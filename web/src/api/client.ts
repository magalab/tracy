import type { Annotation, DashboardMetrics, Page, Span, User, Workspace } from "../types";

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
  options: {
    cursor?: string;
    status?: string;
    kind?: string;
    startTime?: string;
    endTime?: string;
  },
) {
  const params = new URLSearchParams({ limit: "50" });
  if (options.cursor) params.set("cursor", options.cursor);
  if (options.status) params.set("status", options.status);
  if (options.kind) params.set("kind", options.kind);
  if (options.startTime) params.set("start_time", options.startTime);
  if (options.endTime) params.set("end_time", options.endTime);
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

export async function login(email: string, password: string) {
  return request<{ access_token: string; user: User; workspace: Workspace }>(
    "/api/v1/auth/login",
    "",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    },
  );
}

export function getCurrentUser(token: string) {
  return request<{ user: User; workspace: Workspace }>("/api/v1/auth/me", token);
}

export function listWorkspaces(token: string) {
  return request<{ items: Workspace[]; active_id: string }>("/api/v1/workspaces", token);
}

export function createWorkspace(token: string, name: string) {
  return request<{ workspace: Workspace; active_id: string }>("/api/v1/workspaces", token, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
}

export function switchWorkspace(token: string, id: string) {
  return request<{ workspace: Workspace; active_id: string }>(
    `/api/v1/workspaces/${encodeURIComponent(id)}/switch`,
    token,
    { method: "POST" },
  );
}
