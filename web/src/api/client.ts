export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

import { clearAuth, saveAuth } from '@/store/session'

const API_BASE = ''
let refreshPromise: Promise<boolean> | null = null

function authHeaders(extra?: HeadersInit): HeadersInit {
  const token = localStorage.getItem('sub2api_access_token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json'
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  return { ...headers, ...(extra || {}) }
}

function canRefresh(path: string): boolean {
  return ![
    '/api/auth/login',
    '/api/auth/register',
    '/api/auth/refresh',
    '/api/auth/logout'
  ].includes(path)
}

async function refreshAccessToken(): Promise<boolean> {
  if (refreshPromise) return refreshPromise
  refreshPromise = (async () => {
    const refreshToken = localStorage.getItem('sub2api_refresh_token') || ''
    if (!refreshToken) {
      clearAuth()
      return false
    }
    const res = await fetch(API_BASE + '/api/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken })
    })
    const text = await res.text()
    const data = text ? JSON.parse(text) : null
    if (!res.ok || !data?.access_token || !data?.refresh_token || !data?.user) {
      clearAuth()
      return false
    }
    saveAuth(data.access_token, data.refresh_token, data.user)
    return true
  })().finally(() => {
    refreshPromise = null
  })
  return refreshPromise
}

export async function request<T>(path: string, method: HttpMethod = 'GET', body?: unknown, extraHeaders?: HeadersInit, retry = true): Promise<T> {
  const res = await fetch(API_BASE + path, {
    method,
    headers: authHeaders(extraHeaders),
    body: body === undefined ? undefined : JSON.stringify(body)
  })
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (res.status === 401 && retry && canRefresh(path) && localStorage.getItem('sub2api_access_token')) {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      return request<T>(path, method, body, extraHeaders, false)
    }
  }
  if (!res.ok) {
    throw new Error(data?.error || `request failed: ${res.status}`)
  }
  return data as T
}

export async function adminRequest<T>(path: string, method: HttpMethod = 'GET', body?: unknown): Promise<T> {
  const token = localStorage.getItem('sub2api_admin_token') || ''
  return request<T>(path, method, body, { 'X-Admin-Token': token })
}
