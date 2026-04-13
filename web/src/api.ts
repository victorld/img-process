import type { ApiEnvelope, Job, ScanActionCounts, ScanActionItem, ScanEvent, ScanJobLog, Schedule, SystemStatus } from './types'

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  const text = await response.text()
  const payload = text ? (JSON.parse(text) as ApiEnvelope<T>) : undefined
  if (!response.ok) {
    throw new Error(payload?.msg || '请求失败')
  }
  return payload!.data
}

export const api = {
  login: (username: string, password: string) =>
    request<{ authenticated: boolean; username: string }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  logout: () => request<{ authenticated: boolean }>('/api/auth/logout', { method: 'POST' }),
  me: () => request<{ authenticated: boolean; username: string }>('/api/auth/me'),
  getJobs: (params: URLSearchParams) =>
    request<{ list: Job[]; total: number }>(`/api/jobs?${params.toString()}`),
  createJob: (scanArgs: Record<string, unknown>) =>
    request<{ job: Job }>('/api/jobs', {
      method: 'POST',
      body: JSON.stringify({ source: 'manual', scanArgs }),
    }),
  getJob: (id: string | number) => request<{ job: Job }>(`/api/jobs/${id}`),
  getJobEvents: (id: string | number, params: URLSearchParams) =>
    request<{ list: ScanEvent[]; total: number }>(`/api/jobs/${id}/events?${params.toString()}`),
  getJobLogs: (id: string | number, params: URLSearchParams) =>
    request<{ list: ScanJobLog[]; total: number }>(`/api/jobs/${id}/logs?${params.toString()}`),
  getJobActionItems: (id: string | number, params: URLSearchParams) =>
    request<{ list: ScanActionItem[]; total: number; counts: ScanActionCounts }>(`/api/jobs/${id}/action-items?${params.toString()}`),
  getJobActionPreviewUrl: (id: string | number, itemId: number, slot = 'source') =>
    `/api/jobs/${id}/action-preview?itemId=${itemId}&slot=${slot}`,
  executeDuplicateDelete: (id: string | number) =>
    request<{ jobId: number }>(`/api/jobs/${id}/actions/delete-duplicates`, { method: 'POST' }),
  getSchedules: (params: URLSearchParams) =>
    request<{ list: Schedule[]; total: number }>(`/api/schedules?${params.toString()}`),
  createSchedule: (payload: Record<string, unknown>) =>
    request<{ schedule: Schedule }>('/api/schedules', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  updateSchedule: (id: number, payload: Record<string, unknown>) =>
    request<{ schedule: Schedule }>(`/api/schedules/${id}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    }),
  deleteSchedule: (id: number) => request<{ id: number }>(`/api/schedules/${id}`, { method: 'DELETE' }),
  enableSchedule: (id: number) => request<{ schedule: Schedule }>(`/api/schedules/${id}/enable`, { method: 'POST' }),
  disableSchedule: (id: number) => request<{ schedule: Schedule }>(`/api/schedules/${id}/disable`, { method: 'POST' }),
  runSchedule: (id: number) => request<{ job: Job }>(`/api/schedules/${id}/run`, { method: 'POST' }),
  getSystemStatus: () => request<SystemStatus>('/api/system/status'),
}
