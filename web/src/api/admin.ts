import { adminRequest } from './client'
import type { Account, AccountCredentials, APIKey, Announcement, Coupon, DashboardResponse, ModelPrice, OAuthStartResult, PaymentOrder, UsageLog, User } from './types'

export const adminAPI = {
  dashboard: () => adminRequest<DashboardResponse>('/api/admin/dashboard'),
  users: () => adminRequest<User[]>('/api/admin/users'),
  createUser: (payload: { email: string; name: string; password: string; balance?: number; allowed_models?: string[] }) =>
    adminRequest<User>('/api/admin/users', 'POST', payload),
  keys: () => adminRequest<APIKey[]>('/api/admin/api-keys'),
  createKey: (payload: { user_id: number; name: string; allowed_models?: string[]; expires_at_ms?: number }) =>
    adminRequest<APIKey>('/api/admin/api-keys', 'POST', payload),
  accounts: () => adminRequest<Account[]>('/api/admin/accounts'),
  createAccount: (payload: Record<string, unknown>) => adminRequest<Account>('/api/admin/accounts', 'POST', payload),
  updateAccount: (id: number, payload: Record<string, unknown>) => adminRequest(`/api/admin/accounts/${id}`, 'PATCH', payload),
  refreshAccount: (id: number) => adminRequest<Account>(`/api/admin/accounts/${id}/refresh`, 'POST'),
  oauthStart: (payload: Record<string, unknown>) => adminRequest<OAuthStartResult>('/api/admin/accounts/oauth/start', 'POST', payload),
  oauthExchange: (payload: Record<string, unknown>) => adminRequest<AccountCredentials>('/api/admin/accounts/oauth/exchange', 'POST', payload),
  oauthCreate: (payload: Record<string, unknown>) => adminRequest<Account>('/api/admin/accounts/oauth/create', 'POST', payload),
  usage: (limit = 200) => adminRequest<UsageLog[]>(`/api/admin/usage?limit=${limit}`),
  prices: () => adminRequest<ModelPrice[]>('/api/admin/model-prices'),
  createPrice: (payload: Record<string, unknown>) => adminRequest<ModelPrice>('/api/admin/model-prices', 'POST', payload),
  orders: () => adminRequest<PaymentOrder[]>('/api/admin/payment-orders'),
  refundOrder: (payload: { out_trade_no: string; amount: number }) =>
    adminRequest<{ ok: boolean }>('/api/admin/payment-orders/refund', 'POST', payload),
  announcements: () => adminRequest<Announcement[]>('/api/admin/announcements'),
  createAnnouncement: (payload: Record<string, unknown>) => adminRequest<Announcement>('/api/admin/announcements', 'POST', payload),
  coupons: () => adminRequest<Coupon[]>('/api/admin/coupons'),
  createCoupon: (payload: Record<string, unknown>) => adminRequest<Coupon>('/api/admin/coupons', 'POST', payload),
  stats: () => adminRequest<Record<string, unknown>>('/api/admin/stats'),
  errors: (limit = 100) => adminRequest<Array<Record<string, unknown>>>(`/api/admin/errors?limit=${limit}`)
}
