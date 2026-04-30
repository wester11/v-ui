import { useAuth } from '../store/auth'
import type {
  AuditEntry,
  Config,
  CreateConfigRequest,
  CreatePeerResponse,
  CreateServerResponse,
  FleetHealthResult,
  FleetRedeployResult,
  NodeCheckResult,
  Peer,
  Server,
  Stats,
  TokenResponse,
  User,
  SystemVersionInfo,
  SystemUpdateResult,
} from '../types'

export class ApiError extends Error {
  constructor(public status: number, public code?: string) {
    super(code || `HTTP ${status}`)
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const { accessToken, refreshToken, setTokens, logout } = useAuth.getState()
  const headers = new Headers(init.headers)
  if (init.body) headers.set('Content-Type', 'application/json')
  if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)

  let resp = await fetch(path, { ...init, headers })
  if (resp.status === 401 && refreshToken) {
    const r = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (r.ok) {
      const t = (await r.json()) as TokenResponse
      setTokens(t.access_token, t.refresh_token)
      headers.set('Authorization', `Bearer ${t.access_token}`)
      resp = await fetch(path, { ...init, headers })
    } else {
      logout()
    }
  }

  if (!resp.ok) {
    let code: string | undefined
    try { code = (await resp.json()).error } catch { /* ignore */ }
    throw new ApiError(resp.status, code)
  }

  if (resp.status === 204) return undefined as unknown as T
  const ct = resp.headers.get('Content-Type') || ''
  return ct.includes('application/json')
    ? (resp.json() as Promise<T>)
    : ((await resp.text()) as unknown as T)
}

export const api = {
  auth: {
    login: (email: string, password: string) =>
      request<TokenResponse>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      }),
    refresh: (refreshToken: string) =>
      request<TokenResponse>('/api/v1/auth/refresh', {
        method: 'POST',
        body: JSON.stringify({ refresh_token: refreshToken }),
      }),
  },

  me: {
    get: () => request<User>('/api/v1/me'),
    changePassword: (oldPassword: string, newPassword: string) =>
      request<void>('/api/v1/me/password', {
        method: 'PATCH',
        body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
      }),
  },

  stats: {
    get: () => request<Stats>('/api/v1/stats'),
  },

  peers: {
    list: () => request<Peer[]>('/api/v1/peers'),
    create: (server_id: string, name: string, public_key?: string) =>
      request<CreatePeerResponse>('/api/v1/peers', {
        method: 'POST',
        body: JSON.stringify({ server_id, name, public_key }),
      }),
    delete: (id: string) => request<void>(`/api/v1/peers/${id}`, { method: 'DELETE' }),
    config: (id: string) => request<string>(`/api/v1/peers/${id}/config`),
    toggle: (id: string, enabled: boolean) =>
      request<Peer>(`/api/v1/peers/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ enabled }),
      }),
  },

  servers: {
    list: () => request<Server[]>('/api/v1/servers'),
    create: (body: { name: string; endpoint: string }) =>
      request<CreateServerResponse>('/api/v1/servers', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    check: (id: string) => request<NodeCheckResult>(`/api/v1/servers/${id}/check`),
    delete: (id: string) => request<void>(`/api/v1/servers/${id}`, { method: 'DELETE' }),
    deploy:  (id: string) => request<void>(`/api/v1/admin/servers/${id}/deploy`, { method: 'POST' }),
    restart: (id: string) => request<void>(`/api/v1/admin/servers/${id}/restart`, { method: 'POST' }),
    rotateSecret: (id: string) =>
      request<{ server_id: string; secret: string; warning: string }>(
        `/api/v1/admin/servers/${id}/rotate-secret`, { method: 'POST' }),
    metrics: (id: string) => request<unknown>(`/api/v1/admin/servers/${id}/metrics`),
  },

  configs: {
    create: (body: CreateConfigRequest) =>
      request<Config>('/api/v1/configs', { method: 'POST', body: JSON.stringify(body) }),
    listByServer: (serverID: string) => request<Config[]>(`/api/v1/servers/${serverID}/configs`),
    activate: (id: string) => request<void>(`/api/v1/configs/${id}/activate`, { method: 'POST' }),
  },

  users: {
    list: () => request<User[]>('/api/v1/users'),
    create: (body: { email: string; password: string; role: string }) =>
      request<User>('/api/v1/users', { method: 'POST', body: JSON.stringify(body) }),
    delete: (id: string) => request<void>(`/api/v1/users/${id}`, { method: 'DELETE' }),
    setDisabled: (id: string, disabled: boolean) =>
      request<User>(`/api/v1/users/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ disabled }),
      }),
  },

  audit: {
    list: (limit = 100, before?: number) => {
      const q = new URLSearchParams({ limit: String(limit) })
      if (before) q.set('before', String(before))
      return request<AuditEntry[]>(`/api/v1/audit?${q.toString()}`)
    },
  },

  fleet: {
    redeployAll: () => request<FleetRedeployResult[]>('/api/v1/admin/servers/redeploy-all', { method: 'POST' }),
    health: () => request<FleetHealthResult[]>('/api/v1/admin/servers/health'),
    redeployServer: (id: string) => request<void>(`/api/v1/admin/servers/${id}/redeploy`, { method: 'POST' }),
  },
  system: {
    version: () => request<SystemVersionInfo>('/api/v1/admin/system/version'),
    update:  () => request<SystemUpdateResult>('/api/v1/admin/system/update', { method: 'POST' }),
  },

}
