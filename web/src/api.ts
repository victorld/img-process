import type {
  ApiEnvelope,
  BackupDiffResult,
  Job,
  ScanActionCounts,
  ScanActionGroupedCounts,
  ScanActionItem,
  ScanEvent,
  ScanJobLog,
  Schedule,
  DirectoryListing,
  FileAnalysisResult,
  SystemStatus,
} from "./types";

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  const text = await response.text();
  const payload = text ? (JSON.parse(text) as ApiEnvelope<T>) : undefined;
  if (!response.ok) {
    const detail =
      payload?.data &&
      typeof payload.data === "object" &&
      "error" in payload.data &&
      typeof (payload.data as { error?: unknown }).error === "string"
        ? (payload.data as { error: string }).error
        : "";
    throw new Error(detail || payload?.msg || "请求失败");
  }
  return payload!.data;
}

export const api = {
  login: (username: string, password: string) =>
    request<{ authenticated: boolean; username: string }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
  logout: () =>
    request<{ authenticated: boolean }>("/api/auth/logout", { method: "POST" }),
  me: () =>
    request<{ authenticated: boolean; username: string }>("/api/auth/me"),
  getJobs: (params: URLSearchParams) =>
    request<{ list: Job[]; total: number }>(`/api/jobs?${params.toString()}`),
  createJob: (scanArgs: Record<string, unknown>) =>
    request<{ job: Job }>("/api/jobs", {
      method: "POST",
      body: JSON.stringify({ source: "manual", scanArgs }),
    }),
  getJob: (id: string | number) => request<{ job: Job }>(`/api/jobs/${id}`),
  deleteJob: (id: string | number) =>
    request<{ id: number; actionItems: number; events: number; logs: number; schedules: number }>(
      `/api/jobs/${id}`,
      { method: "DELETE" },
    ),
  getJobEvents: (id: string | number, params: URLSearchParams) =>
    request<{ list: ScanEvent[]; total: number }>(
      `/api/jobs/${id}/events?${params.toString()}`,
    ),
  getJobLogs: (id: string | number, params: URLSearchParams) =>
    request<{ list: ScanJobLog[]; total: number }>(
      `/api/jobs/${id}/logs?${params.toString()}`,
    ),
  getJobActionItems: (id: string | number, params: URLSearchParams) =>
    request<{
      list: ScanActionItem[];
      total: number;
      counts: ScanActionCounts;
      groupedCounts: ScanActionGroupedCounts;
    }>(`/api/jobs/${id}/action-items?${params.toString()}`),
  getJobBackupDiff: (id: string | number) =>
    request<BackupDiffResult>(`/api/jobs/${id}/backup-diff`),
  getJobActionPreviewUrl: (
    id: string | number,
    itemId: number,
    slot = "source",
    quality?: "thumb" | "full",
  ) => {
    const params = new URLSearchParams({
      itemId: String(itemId),
      slot,
    });
    if (quality) {
      params.set("quality", quality);
    }
    return `/api/jobs/${id}/action-preview?${params.toString()}`;
  },
  executeDeleteActionItem: (id: string | number, itemId: number) =>
    request<{ jobId: number; itemId: number }>(
      `/api/jobs/${id}/action-items/${itemId}/delete`,
      { method: "POST" },
    ),
  executeModifyShootTimeActionItem: (id: string | number, itemId: number) =>
    request<{ jobId: number; itemId: number }>(
      `/api/jobs/${id}/action-items/${itemId}/modify-shoot-time`,
      { method: "POST" },
    ),
  executeMoveActionItem: (id: string | number, itemId: number) =>
    request<{ jobId: number; itemId: number }>(
      `/api/jobs/${id}/action-items/${itemId}/move`,
      { method: "POST" },
    ),
  executeRenameActionItem: (id: string | number, itemId: number) =>
    request<{ jobId: number; itemId: number }>(
      `/api/jobs/${id}/action-items/${itemId}/rename`,
      { method: "POST" },
    ),
  executeDuplicateDeleteActionItem: (
    id: string | number,
    itemId: number,
    side: "A" | "B" | "PATH",
    path?: string,
  ) =>
    request<{ jobId: number; itemId: number; side: "A" | "B" | "PATH"; path?: string }>(
      `/api/jobs/${id}/action-items/${itemId}/delete-duplicate`,
      {
        method: "POST",
        body: JSON.stringify({ side, path }),
      },
    ),
  executeDeleteAllActionItems: (id: string | number) =>
    request<{ jobId: number; count: number }>(
      `/api/jobs/${id}/actions/delete-all`,
      { method: "POST" },
    ),
  executeDuplicateDelete: (id: string | number) =>
    request<{ jobId: number }>(`/api/jobs/${id}/actions/delete-duplicates`, {
      method: "POST",
    }),
  executePathDuplicateDelete: (id: string | number) =>
    request<{ jobId: number; count: number }>(
      `/api/jobs/${id}/actions/delete-path-duplicates`,
      { method: "POST" },
    ),
  getSchedules: (params: URLSearchParams) =>
    request<{ list: Schedule[]; total: number }>(
      `/api/schedules?${params.toString()}`,
    ),
  createSchedule: (payload: Record<string, unknown>) =>
    request<{ schedule: Schedule }>("/api/schedules", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  updateSchedule: (id: number, payload: Record<string, unknown>) =>
    request<{ schedule: Schedule }>(`/api/schedules/${id}`, {
      method: "PUT",
      body: JSON.stringify(payload),
    }),
  deleteSchedule: (id: number) =>
    request<{ id: number }>(`/api/schedules/${id}`, { method: "DELETE" }),
  enableSchedule: (id: number) =>
    request<{ schedule: Schedule }>(`/api/schedules/${id}/enable`, {
      method: "POST",
    }),
  disableSchedule: (id: number) =>
    request<{ schedule: Schedule }>(`/api/schedules/${id}/disable`, {
      method: "POST",
    }),
  runSchedule: (id: number) =>
    request<{ job: Job }>(`/api/schedules/${id}/run`, { method: "POST" }),
  getFileAnalysis: (params: URLSearchParams) =>
    request<FileAnalysisResult>(`/api/files/analysis?${params.toString()}`),
  getSystemStatus: () => request<SystemStatus>("/api/system/status"),
  listSystemDirectories: (path?: string) => {
    const params = new URLSearchParams();
    if (path) {
      params.set("path", path);
    }
    const query = params.toString();
    return request<DirectoryListing>(`/api/system/directories${query ? `?${query}` : ""}`);
  },
  selectSystemDirectory: (path: string | undefined, title: string) =>
    request<{ path: string }>("/api/system/select-directory", {
      method: "POST",
      body: JSON.stringify({ path, title }),
    }),
  updateSystemSettings: (config: Record<string, Record<string, unknown>>) =>
    request<Pick<SystemStatus, "config" | "configFile" | "configSource" | "readonlySections">>(
      "/api/system/settings",
      {
        method: "PUT",
        body: JSON.stringify({ config }),
      },
    ),
};
