import type { ApiErrorBody } from '~/types/api'

interface FetchLikeError {
  data?: Partial<ApiErrorBody>
  statusCode?: number
  response?: unknown
}

/** HTTP status of a failed API call, if the server answered. */
export function apiErrorStatus(err: unknown): number | undefined {
  return (err as FetchLikeError | null)?.statusCode
}

/** A message safe to show users: the API's own message, or a generic one. */
export function apiErrorMessage(err: unknown, fallback = 'Something went wrong. Please try again.'): string {
  const e = err as FetchLikeError | null
  if (typeof e?.data?.message === 'string' && e.data.message) return e.data.message
  if (e && e.statusCode === undefined) return 'Can\'t reach the server. Check your connection and try again.'
  return fallback
}

/** Per-field validation messages: { email: "Enter a valid email address" }. */
export function apiFieldErrors(err: unknown): Record<string, string> {
  return (err as FetchLikeError | null)?.data?.errors ?? {}
}
