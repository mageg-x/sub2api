import i18n from '@/i18n'

export function formatTime(ts: number): string {
  if (!ts) return '-'
  const locale = i18n.global.locale.value
  return new Date(ts).toLocaleString(locale)
}

export function formatCurrency(value: number): string {
  const locale = i18n.global.locale.value
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: locale === 'zh-CN' ? 'CNY' : 'USD',
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

export function formatNumber(value: number): string {
  const t = i18n.global.t.bind(i18n.global)
  if (value >= 1e8) return (value / 1e8).toFixed(2) + t('common.hundredMillion')
  if (value >= 1e6) return (value / 1e6).toFixed(1) + t('common.million')
  if (value >= 1e4) return (value / 1e4).toFixed(2) + t('common.tenThousand')
  if (value >= 1e3) return (value / 1e3).toFixed(1) + t('common.thousand')
  return value.toLocaleString()
}

export async function copyToClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
  }
}
