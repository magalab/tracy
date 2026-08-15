import type {
  APIKey,
  CreatedAPIKey,
  DashboardMetrics,
  Page,
  Span,
  User,
  Workspace,
} from "../types";

export async function request<T>(
  path: string,
  token: string,
  init?: RequestInit,
  workspaceID?: string,
): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(workspaceID ? { "X-Tracy-Workspace-ID": workspaceID } : {}),
      ...init?.headers,
    },
  });
  const raw = await response.text();
  let data: { error?: { message?: string } } | undefined;
  if (raw) {
    try {
      data = JSON.parse(raw) as { error?: { message?: string } };
    } catch {
      data = undefined;
    }
  }
  if (!response.ok) throw new Error(data?.error?.message ?? `Request failed (${response.status})`);
  return data as T;
}

export function listTraces(
  token: string,
  workspaceID: string,
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
  return request<Page>(`/api/v1/traces?${params.toString()}`, token, undefined, workspaceID);
}

export function getDashboardMetrics(token: string, workspaceID: string) {
  return request<DashboardMetrics>("/api/v1/dashboard", token, undefined, workspaceID);
}

export async function getTrace(
  id: string,
  token: string,
  workspaceID: string,
  options: { cursor?: string; limit?: number } = {},
) {
  const params = new URLSearchParams();
  if (options.cursor) params.set("cursor", options.cursor);
  if (options.limit) params.set("limit", String(options.limit));
  const query = params.toString();
  return request<{ trace_id: string; spans: Span[]; next_cursor?: string }>(
    `/api/v1/traces/${encodeURIComponent(id)}${query ? `?${query}` : ""}`,
    token,
    undefined,
    workspaceID,
  );
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
  return request<{ user: User; workspace?: Workspace }>("/api/v1/auth/me", token);
}

export function logout(token: string) {
  return request<{ logged_out: boolean }>("/api/v1/auth/logout", token, { method: "POST" });
}

export function listWorkspaces(token: string) {
  return request<{ items: Workspace[] }>("/api/v1/workspaces", token);
}

export function createWorkspace(token: string, name: string) {
  return request<{ workspace: Workspace }>("/api/v1/workspaces", token, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
}

export function switchWorkspace(token: string, id: string) {
  return request<{ workspace: Workspace }>(
    `/api/v1/workspaces/${encodeURIComponent(id)}/switch`,
    token,
    { method: "POST" },
  );
}

export function listWorkspaceKeys(token: string, workspaceID: string) {
  return request<{ items: APIKey[] }>(
    `/api/v1/workspaces/${encodeURIComponent(workspaceID)}/keys`,
    token,
  );
}

export function createWorkspaceKey(
  token: string,
  workspaceID: string,
  name: string,
  expiresAt?: string,
) {
  return request<{ api_key: CreatedAPIKey }>(
    `/api/v1/workspaces/${encodeURIComponent(workspaceID)}/keys`,
    token,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, ...(expiresAt ? { expires_at: expiresAt } : {}) }),
    },
  );
}

export function revokeWorkspaceKey(token: string, workspaceID: string, keyID: string) {
  return request<{ revoked: boolean }>(
    `/api/v1/workspaces/${encodeURIComponent(workspaceID)}/keys/${encodeURIComponent(keyID)}/revoke`,
    token,
    { method: "POST" },
  );
}
