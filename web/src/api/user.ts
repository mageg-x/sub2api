import { request } from './client'
import type { APIKey, ModelCatalogChannel, ModelPrice, PaymentCreateResponse, PaymentOrder, UsageLog, User } from './types'

export const userAPI = {
  profile: () => request<User>('/api/user/profile'),
  updateProfile: (payload: { name: string }) => request<User>('/api/user/profile', 'PUT', payload),
  changePassword: (payload: { old_password: string; new_password: string }) => request<{ ok: boolean }>('/api/user/change-password', 'POST', payload),
  redeem: (payload: { code: string }) => request<Record<string, unknown>>('/api/user/redeem', 'POST', payload),
  keys: () => request<APIKey[]>('/api/keys'),
  createKey: (payload: { name: string; allowed_models?: string[]; expires_at_ms?: number }) =>
    request<APIKey>('/api/keys', 'POST', payload),
  usage: (limit = 100) => request<UsageLog[]>(`/api/usage?limit=${limit}`),
  orders: () => request<PaymentOrder[]>('/api/payment/orders/my'),
  orderByID: (id: number) => request<PaymentOrder>(`/api/payment/orders/${id}`),
  modelCatalog: () => request<ModelCatalogChannel[]>('/api/model-catalog'),
  modelPrices: () => request<ModelPrice[]>('/api/model-prices'),
  createPayment: (payload: { user_id?: number; amount: number; subject?: string; return_url?: string }) =>
    request<PaymentCreateResponse>('/api/payments/orders', 'POST', payload)
}
