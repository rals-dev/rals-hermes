// Small, dependency-free formatting helpers. Everything the UI prints about
// time and numbers goes through here so it reads the same on every page.

import type { Usage } from '@/api/types'

function parse(iso: string | null | undefined): Date | null {
  if (!iso) return null
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? null : d
}

/** "5s ago", "13m ago", "3h ago", "3d ago"; "just now" for the future. */
export function relativeTime(iso: string | null | undefined, now: Date = new Date()): string {
  const d = parse(iso)
  if (!d) return '—'
  const s = Math.floor((now.getTime() - d.getTime()) / 1000)
  if (s < 5) return 'just now'
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

/** Elapsed time as "40s", "30m 0s", "2h 45m". */
export function durationSince(iso: string | null | undefined, now: Date = new Date()): string {
  const d = parse(iso)
  if (!d) return '—'
  const s = Math.max(0, Math.floor((now.getTime() - d.getTime()) / 1000))
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ${s % 60}s`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m`
}

/** 1552865 → "1.6M", 51431 → "51.4k", 999 → "999". */
export function compactNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`
  return String(n)
}

/** "1.6M in · 50.8k out · 17.8M cached" (cache part only when non-zero). */
export function tokensLabel(u: Usage): string {
  const parts = [`${compactNumber(u.input_tokens)} in`, `${compactNumber(u.output_tokens)} out`]
  if (u.cache_read_tokens > 0) parts.push(`${compactNumber(u.cache_read_tokens)} cached`)
  return parts.join(' · ')
}

/** "20260913_163024_5d910738" → "0913 16:30 · 5d91"; unknown shapes pass through. */
export function shortId(id: string): string {
  const m = /^\d{4}(\d{2})(\d{2})_(\d{2})(\d{2})\d{2}_([0-9a-f]{4})/.exec(id)
  if (!m) return id
  return `${m[1]}${m[2]} ${m[3]}:${m[4]} · ${m[5]}`
}

/** Cost in USD with sensible precision; null when Hermes has none. */
export function costLabel(estimated: number | null, actual: number | null): string | null {
  const v = actual ?? estimated
  if (v === null || v === undefined) return null
  if (v === 0) return '$0.00'
  return v < 0.01 ? `$${v.toFixed(4)}` : `$${v.toFixed(2)}`
}
