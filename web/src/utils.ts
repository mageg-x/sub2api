export function formatTime(ts: number): string {
  if (!ts) return '-'
  return new Date(ts).toLocaleString()
}

export function formatCurrency(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
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

export function parseCSV(value: string): string[] {
  return value.split(',').map((item) => item.trim()).filter(Boolean)
}

export function maskSecret(value: string): string {
  if (!value) return '-'
  if (value.length <= 12) return value
  return `${value.slice(0, 6)}...${value.slice(-4)}`
}

export function prettyJSON(value: unknown): string {
  return JSON.stringify(value ?? {}, null, 2)
}
