export interface User {
  id: number
  email: string
  name: string
  role: string
  status: string
  balance: number
  rate_percent: number
  allowed_models_json: string
  last_login_at_ms: number
}

export interface UserUpdatePayload {
  name?: string
  status?: string
  role?: string
  balance?: number
  rate_percent?: number
  allowed_models?: string[]
  password?: string
}

export interface APIKey {
  id: number
  user_id: number
  name: string
  secret: string
  status: string
  allowed_models_json: string
  expires_at_ms: number
  last_used_at_ms: number
}

export interface Account {
  id: number
  provider: string
  name: string
  auth_type: string
  status: string
  base_url: string
  model_scope_json: string
  priority: number
  concurrency_limit: number
  expires_at_ms: number
  last_refreshed_at_ms: number
  credentials?: AccountCredentialSummary
}

export interface AccountCredentialSummary {
  email?: string
  expires_at_ms?: number
  project_id?: string
  oauth_type?: string
  organization_id?: string
  account_id?: string
  plan_type?: string
  subscription_until?: string
  api_key_masked?: string
  refresh_token_masked?: string
  access_token_masked?: string
  setup_token_masked?: string
  token_url?: string
  redirect_uri?: string
}

export interface UsageLog {
  id: number
  user_id: number
  api_key_id: number
  account_id: number
  provider: string
  model: string
  endpoint: string
  input_tokens: number
  output_tokens: number
  cost: number
  created_at_ms: number
}

export interface PaymentOrder {
  id: number
  user_id: number
  provider: string
  out_trade_no: string
  provider_trade_no: string
  subject: string
  status: string
  amount: number
  credited_amount: number
  created_at_ms: number
}

export interface PaymentCreateResponse {
  order: PaymentOrder
  paying: Record<string, unknown>
}

export interface Coupon {
  id: number
  code: string
  kind: string
  amount: number
  max_uses: number
  used_count: number
  expires_at_ms: number
  status: string
}

export interface ModelPrice {
  id: number
  provider: string
  model: string
  input_price: number
  output_price: number
  cache_create_price: number
  cache_read_price: number
  currency: string
  status: string
}

export interface ModelCatalogChannel {
  key: string
  name: string
  multiplier: string
  note: string
  models: ModelPrice[]
}

export interface Announcement {
  id: number
  title: string
  content: string
  status: string
  published_at_ms: number
}

export interface DashboardResponse {
  users: User[]
  accounts: Account[]
  prices: ModelPrice[]
  orders: PaymentOrder[]
  announcements: Announcement[]
  coupons: Coupon[]
  stats: Record<string, unknown>
}

export interface AuthResponse {
  user: User
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface OAuthStartResult {
  provider: string
  session_id: string
  state: string
  auth_url: string
}

export interface AccountCredentials {
  access_token?: string
  refresh_token?: string
  client_id?: string
  client_secret?: string
  token_url?: string
  redirect_uri?: string
  expires_at_ms?: number
  project_id?: string
  oauth_type?: string
  email?: string
  organization_id?: string
  account_id?: string
  plan_type?: string
}
