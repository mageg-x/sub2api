import { request } from './client'
import type { AuthResponse, User } from './types'

export async function login(email: string, password: string): Promise<AuthResponse> {
  return request<AuthResponse>('/api/auth/login', 'POST', { email, password })
}

export async function register(name: string, email: string, password: string): Promise<AuthResponse> {
  return request<AuthResponse>('/api/auth/register', 'POST', { name, email, password })
}

export async function me(): Promise<User> {
  return request<User>('/api/auth/me')
}

export async function refresh(refreshToken: string): Promise<AuthResponse> {
  return request<AuthResponse>('/api/auth/refresh', 'POST', { refresh_token: refreshToken })
}

export async function logout(refreshToken: string): Promise<void> {
  await request('/api/auth/logout', 'POST', { refresh_token: refreshToken })
}

