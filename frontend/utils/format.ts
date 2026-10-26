const MINUTE = 60_000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

/** "1h 05m", "12m", "<1m" since an ISO time. */
export function formatDuration(fromIso: string, now = Date.now()): string {
  const ms = Math.max(0, now - Date.parse(fromIso))
  if (ms < MINUTE) return '<1m'
  const hours = Math.floor(ms / HOUR)
  const minutes = Math.floor((ms % HOUR) / MINUTE)
  return hours ? `${hours}h ${String(minutes).padStart(2, '0')}m` : `${minutes}m`
}

/** "just now", "5m ago", "3h ago", "2d ago", or a date. */
export function formatRelativeTime(iso: string, now = Date.now()): string {
  const ms = now - Date.parse(iso)
  if (ms < MINUTE) return 'just now'
  if (ms < HOUR) return `${Math.floor(ms / MINUTE)}m ago`
  if (ms < DAY) return `${Math.floor(ms / HOUR)}h ago`
  if (ms < 7 * DAY) return `${Math.floor(ms / DAY)}d ago`
  return formatDate(iso)
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

export function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
}

/** 950, 1.2K, 3.4M */
export function formatCount(n: number): string {
  if (n < 1000) return String(n)
  if (n < 1_000_000) return `${trimZero(n / 1000)}K`
  return `${trimZero(n / 1_000_000)}M`
}

function trimZero(n: number): string {
  return n.toFixed(1).replace(/\.0$/, '')
}

/** Channel URL suggestion from a title: "My Show!" -> "my-show". */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 40)
    .replace(/-+$/g, '')
}
