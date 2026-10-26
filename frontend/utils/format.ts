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
    .replace(/[\u0300-\u036F]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 40)
    .replace(/-+$/g, '')
}

export interface CountdownParts {
  days: number
  hours: number
  minutes: number
  seconds: number
}

export function countdownParts(ms: number): CountdownParts {
  const total = Math.max(0, Math.floor(ms / 1000))
  return {
    days: Math.floor(total / 86_400),
    hours: Math.floor((total % 86_400) / 3600),
    minutes: Math.floor((total % 3600) / 60),
    seconds: total % 60,
  }
}

/** "2d 3h", "3h 05m", "12m", "<1m" until an ISO time. */
export function formatTimeUntil(iso: string, now = Date.now()): string {
  const ms = Date.parse(iso) - now
  if (ms < MINUTE) return '<1m'
  const { days, hours, minutes } = countdownParts(ms)
  if (days) return `${days}d ${hours}h`
  if (hours) return `${hours}h ${String(minutes).padStart(2, '0')}m`
  return `${minutes}m`
}

/** "Sat, 1 Nov, 20:00" in the viewer's time zone. */
export function formatScheduleDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}

/** How long after the scheduled time "Starting soon" keeps showing. */
const SCHEDULE_GRACE = 2 * HOUR

/** A schedule worth showing: upcoming, or less than 2 hours late. */
export function isScheduleActive(iso: string | null | undefined, now = Date.now()): iso is string {
  return !!iso && Date.parse(iso) + SCHEDULE_GRACE > now
}

/** ISO time → value for <input type="datetime-local">, in local time. */
export function toDateTimeLocal(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}
