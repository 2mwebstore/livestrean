// Client-side checks for instant feedback. They mirror the Go API's rules
// (backend/internal/validators); the API still validates everything.

export const SLUG_PATTERN = /^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$/
const RESERVED_SLUGS = new Set(['admin', 'api', 'dashboard', 'embed', 'live', 'login', 'logout', 'register', 'settings', 'ws'])

export function slugError(slug: string): string | undefined {
  if (!slug) return 'This field is required'
  if (!SLUG_PATTERN.test(slug)) return 'Use 3-40 lowercase letters, numbers or hyphens (not at the start or end)'
  if (RESERVED_SLUGS.has(slug)) return 'This name is reserved'
  return undefined
}

export function lengthError(value: string, { required = false, max }: { required?: boolean, max: number }): string | undefined {
  if (required && !value.trim()) return 'This field is required'
  if (value.length > max) return `Must be at most ${max} characters`
  return undefined
}

/** Only same-site paths, so ?redirect= can't send users to another site. */
export function safeRedirect(value: unknown, fallback = '/'): string {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//') || value.startsWith('/\\')) {
    return fallback
  }
  return value
}
