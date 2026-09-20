import { describe, expect, it } from 'vitest'
import { compactNumber, relativeTime, shortId, tokensLabel, durationSince } from './format'

describe('relativeTime', () => {
  const now = new Date('2026-09-20T12:00:00Z')
  it('speaks in seconds, minutes, hours, days', () => {
    expect(relativeTime('2026-09-20T11:59:55Z', now)).toBe('5s ago')
    expect(relativeTime('2026-09-20T11:47:00Z', now)).toBe('13m ago')
    expect(relativeTime('2026-09-20T09:00:00Z', now)).toBe('3h ago')
    expect(relativeTime('2026-09-17T12:00:00Z', now)).toBe('3d ago')
  })
  it('handles the future and garbage without throwing', () => {
    expect(relativeTime('2026-09-20T12:00:30Z', now)).toBe('just now')
    expect(relativeTime('not a date', now)).toBe('—')
    expect(relativeTime(null, now)).toBe('—')
  })
})

describe('compactNumber', () => {
  it('abbreviates thousands and millions with one decimal', () => {
    expect(compactNumber(0)).toBe('0')
    expect(compactNumber(999)).toBe('999')
    expect(compactNumber(1552865)).toBe('1.6M')
    expect(compactNumber(17790464)).toBe('17.8M')
    expect(compactNumber(51431)).toBe('51.4k')
  })
})

describe('tokensLabel', () => {
  it('summarises usage as in/out with cache when present', () => {
    expect(tokensLabel({ input_tokens: 1552865, output_tokens: 50817, cache_read_tokens: 17790464, cache_write_tokens: 0, reasoning_tokens: 18748 }))
      .toBe('1.6M in · 50.8k out · 17.8M cached')
    expect(tokensLabel({ input_tokens: 100, output_tokens: 5, cache_read_tokens: 0, cache_write_tokens: 0, reasoning_tokens: 0 }))
      .toBe('100 in · 5 out')
  })
})

describe('shortId', () => {
  it('keeps the readable timestamp part of a Hermes session id', () => {
    expect(shortId('20260913_163024_5d910738')).toBe('0913 16:30 · 5d91')
    expect(shortId('weird')).toBe('weird')
  })
})

describe('durationSince', () => {
  it('formats elapsed time compactly', () => {
    const now = new Date('2026-09-20T12:00:00Z')
    expect(durationSince('2026-09-20T11:59:20Z', now)).toBe('40s')
    expect(durationSince('2026-09-20T11:30:00Z', now)).toBe('30m 0s')
    expect(durationSince('2026-09-20T09:15:00Z', now)).toBe('2h 45m')
  })
})
