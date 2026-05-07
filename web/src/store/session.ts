import { reactive } from 'vue'
import type { User } from '@/api/types'

function loadStoredUser(): User | null {
  const raw = localStorage.getItem('sub2api_user')
  if (!raw) return null
  try {
    return JSON.parse(raw) as User
  } catch {
    localStorage.removeItem('sub2api_user')
    return null
  }
}

export const session = reactive<{
  user: User | null
  adminToken: string
}>({
  user: loadStoredUser(),
  adminToken: localStorage.getItem('sub2api_admin_token') || ''
})

export function saveAuth(accessToken: string, refreshToken: string, user: User) {
  localStorage.setItem('sub2api_access_token', accessToken)
  localStorage.setItem('sub2api_refresh_token', refreshToken)
  localStorage.setItem('sub2api_user', JSON.stringify(user))
  session.user = user
}

export function clearAuth() {
  localStorage.removeItem('sub2api_access_token')
  localStorage.removeItem('sub2api_refresh_token')
  localStorage.removeItem('sub2api_user')
  session.user = null
}

export function saveAdminToken(token: string) {
  localStorage.setItem('sub2api_admin_token', token)
  session.adminToken = token
}
