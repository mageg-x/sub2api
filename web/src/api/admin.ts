import { adminRequest } from './client'
import type { Account, AccountCredentials, Announcement, Coupon, DashboardResponse, ModelPrice, OAuthStartResult, PaymentOrder, ProviderCapability, User, UserUpdatePayload } from './types'

export const adminAPI = {
  dashboard: () => adminRequest<DashboardResponse>('/api/admin/dashboard'),
  users: () => adminRequest<User[]>('/api/admin/users'),
  updateUser: (id: number, payload: UserUpdatePayload) => adminRequest<User>(`/api/admin/users/${id}`, 'PATCH', payload),
  accounts: () => adminRequest<Account[]>('/api/admin/accounts'),
  providerCapabilities: () => adminRequest<ProviderCapability[]>('/api/admin/provider-capabilities'),
  createAccount: (payload: Record<string, unknown>) => adminRequest<Account>('/api/admin/accounts', 'POST', payload),
  deleteAccount: (id: number) => adminRequest(`/api/admin/accounts/${id}`, 'DELETE'),
  updateAccount: (id: number, payload: Record<string, unknown>) => adminRequest(`/api/admin/accounts/${id}`, 'PATCH', payload),
  refreshAccount: (id: number) => adminRequest<Account>(`/api/admin/accounts/${id}/refresh`, 'POST'),
  oauthStart: (payload: Record<string, unknown>) => adminRequest<OAuthStartResult>('/api/admin/accounts/oauth/start', 'POST', payload),
  oauthExchange: (payload: Record<string, unknown>) => adminRequest<AccountCredentials>('/api/admin/accounts/oauth/exchange', 'POST', payload),
  oauthCreate: (payload: Record<string, unknown>) => adminRequest<Account>('/api/admin/accounts/oauth/create', 'POST', payload),
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
