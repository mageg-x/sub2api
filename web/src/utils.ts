import i18n from '@/i18n'

export function formatTime(ts: number): string {
  if (!ts) return '-'
  const locale = i18n.global.locale.value
  return new Date(ts).toLocaleString(locale)
}

export function formatCurrency(value: number): string {
  const locale = i18n.global.locale.value
  return new Intl.NumberFormat(locale, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format((value || 0) / 10000)
}

export function isActiveStatus(status: string): boolean {
  return String(status || '').trim().toLowerCase() === 'active'
}

export function isPaidStatus(status: string): boolean {
  return String(status || '').trim().toLowerCase() === 'paid'
}

export function maskSecret(value: string): string {
  if (!value) return '-'
  if (value.length <= 12) return value
  return `${value.slice(0, 6)}...${value.slice(-4)}`
}

export function prettyJSON(value: unknown): string {
  return JSON.stringify(value ?? {}, null, 2)
}
