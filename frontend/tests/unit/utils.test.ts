import { describe, expect, it } from 'vitest'
import { apiErrorMessage, apiErrorStatus, apiFieldErrors } from '../../utils/api-error'
import { countdownParts, formatCount, formatDuration, formatRelativeTime, formatTimeUntil, isScheduleActive, slugify } from '../../utils/format'
import { safeRedirect, slugError } from '../../utils/validation'

const now = Date.parse('2026-01-01T12:00:00Z')

describe('format', () => {
  it('formats live durations', () => {
    expect(formatDuration('2026-01-01T11:59:30Z', now)).toBe('<1m')
    expect(formatDuration('2026-01-01T11:48:00Z', now)).toBe('12m')
    expect(formatDuration('2026-01-01T10:55:00Z', now)).toBe('1h 05m')
  })

  it('formats relative times', () => {
    expect(formatRelativeTime('2026-01-01T11:59:50Z', now)).toBe('just now')
    expect(formatRelativeTime('2026-01-01T11:55:00Z', now)).toBe('5m ago')
    expect(formatRelativeTime('2026-01-01T09:00:00Z', now)).toBe('3h ago')
    expect(formatRelativeTime('2025-12-30T12:00:00Z', now)).toBe('2d ago')
  })

  it('formats counts', () => {
    expect(formatCount(950)).toBe('950')
    expect(formatCount(1000)).toBe('1K')
    expect(formatCount(1250)).toBe('1.3K')
    expect(formatCount(3_400_000)).toBe('3.4M')
  })

  it('suggests channel URLs', () => {
    expect(slugify('Friday Night Music!')).toBe('friday-night-music')
    expect(slugify('  Café   del Mar ')).toBe('cafe-del-mar')
    expect(slugify('x'.repeat(50))).toHaveLength(40)
  })
})

describe('validation', () => {
  it('checks channel URLs like the API does', () => {
    expect(slugError('my-show')).toBeUndefined()
    expect(slugError('')).toBeDefined()
    expect(slugError('ab')).toBeDefined()
    expect(slugError('-show')).toBeDefined()
    expect(slugError('My-Show')).toBeDefined()
    expect(slugError('admin')).toBe('This name is reserved')
  })

  it('only allows same-site redirects', () => {
    expect(safeRedirect('/c/my-show')).toBe('/c/my-show')
    expect(safeRedirect('https://evil.example')).toBe('/')
    expect(safeRedirect('//evil.example')).toBe('/')
    expect(safeRedirect('/\\evil.example')).toBe('/')
    expect(safeRedirect(undefined)).toBe('/')
    expect(safeRedirect(['/a'])).toBe('/')
  })
})

describe('api errors', () => {
  const fetchError = { statusCode: 422, data: { success: false, message: 'Please check the highlighted fields', errors: { email: 'Enter a valid email address' } } }

  it('reads the API message and field errors', () => {
    expect(apiErrorMessage(fetchError)).toBe('Please check the highlighted fields')
    expect(apiFieldErrors(fetchError)).toEqual({ email: 'Enter a valid email address' })
    expect(apiErrorStatus(fetchError)).toBe(422)
  })

  it('falls back to safe generic messages', () => {
    expect(apiErrorMessage({ statusCode: 500 })).toBe('Something went wrong. Please try again.')
    expect(apiErrorMessage(new Error('fetch failed'))).toMatch(/reach the server/)
    expect(apiFieldErrors(null)).toEqual({})
  })
})

describe('countdown', () => {
  it('splits time into days, hours, minutes and seconds', () => {
    expect(countdownParts(((2 * 24 + 3) * 3600 + 4 * 60 + 5) * 1000)).toEqual({ days: 2, hours: 3, minutes: 4, seconds: 5 })
    expect(countdownParts(-5000)).toEqual({ days: 0, hours: 0, minutes: 0, seconds: 0 })
  })

  it('formats time until a stream', () => {
    expect(formatTimeUntil('2026-01-01T12:00:30Z', now)).toBe('<1m')
    expect(formatTimeUntil('2026-01-01T12:12:00Z', now)).toBe('12m')
    expect(formatTimeUntil('2026-01-01T15:05:00Z', now)).toBe('3h 05m')
    expect(formatTimeUntil('2026-01-03T15:00:00Z', now)).toBe('2d 3h')
  })

  it('keeps showing a schedule up to 2 hours late', () => {
    expect(isScheduleActive(null, now)).toBe(false)
    expect(isScheduleActive('2026-01-01T13:00:00Z', now)).toBe(true)
    expect(isScheduleActive('2026-01-01T11:00:00Z', now)).toBe(true)
    expect(isScheduleActive('2026-01-01T09:59:00Z', now)).toBe(false)
  })
})
