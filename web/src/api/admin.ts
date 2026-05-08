import { request } from './client'
import type { Account, AccountCredentials, Announcement, Coupon, DashboardResponse, ModelPrice, OAuthStartResult, PaymentOrder, ProviderCapability, User, UserUpdatePayload } from './types'

export const adminAPI = {
  dashboard: () => request<DashboardResponse>('/api/admin/dashboard'),
  users: () => request<User[]>('/api/admin/users'),
  updateUser: (id: number, payload: UserUpdatePayload) => request<User>(`/api/admin/users/${id}`, 'PATCH', payload),
  accounts: () => request<Account[]>('/api/admin/accounts'),
  providerCapabilities: () => request<ProviderCapability[]>('/api/admin/provider-capabilities'),
  createAccount: (payload: Record<string, unknown>) => request<Account>('/api/admin/accounts', 'POST', payload),
  deleteAccount: (id: number) => request(`/api/admin/accounts/${id}`, 'DELETE'),
  updateAccount: (id: number, payload: Record<string, unknown>) => request(`/api/admin/accounts/${id}`, 'PATCH', payload),
  refreshAccount: (id: number) => request<Account>(`/api/admin/accounts/${id}/refresh`, 'POST'),
  oauthStart: (payload: Record<string, unknown>) => request<OAuthStartResult>('/api/admin/accounts/oauth/start', 'POST', payload),
  oauthExchange: (payload: Record<string, unknown>) => request<AccountCredentials>('/api/admin/accounts/oauth/exchange', 'POST', payload),
  oauthCreate: (payload: Record<string, unknown>) => request<Account>('/api/admin/accounts/oauth/create', 'POST', payload),
  prices: () => request<ModelPrice[]>('/api/admin/model-prices'),
  createPrice: (payload: Record<string, unknown>) => request<ModelPrice>('/api/admin/model-prices', 'POST', payload),
  orders: () => request<PaymentOrder[]>('/api/admin/payment-orders'),
  refundOrder: (payload: { out_trade_no: string; amount: number }) =>
    request<{ ok: boolean }>('/api/admin/payment-orders/refund', 'POST', payload),
  announcements: () => request<Announcement[]>('/api/admin/announcements'),
  createAnnouncement: (payload: Record<string, unknown>) => request<Announcement>('/api/admin/announcements', 'POST', payload),
  coupons: () => request<Coupon[]>('/api/admin/coupons'),
  createCoupon: (payload: Record<string, unknown>) => request<Coupon>('/api/admin/coupons', 'POST', payload),
  stats: () => request<Record<string, unknown>>('/api/admin/stats'),
  errors: (limit = 100) => request<Array<Record<string, unknown>>>(`/api/admin/errors?limit=${limit}`)
}
