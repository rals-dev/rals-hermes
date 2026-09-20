// Feed state: turns the BFF's activity events into rows the UI renders.
// tool.started / tool.completed pairs are merged into one row keyed by
// call_id, so a tool shows its arguments, then its result and duration.

import type { ActivityEvent, Session } from '@/api/types'

export interface FeedRow {
  key: string
  kind: 'tool' | 'message' | 'session' | 'error'
  profile: string
  sessionId?: string
  at: string            // when it started
  title: string         // tool name, role, or session event label
  detail: string        // arguments / text preview / reason
  result?: string       // tool result preview
  done: boolean
  failed: boolean
  durationMs?: number
}

export const maxRows = 250

export function applyEvent(rows: FeedRow[], e: ActivityEvent): FeedRow[] {
  switch (e.type) {
    case 'tool.started':
      return prepend(rows, {
        key: rowKey(e, e.call_id ?? String(e.message_id)),
        kind: 'tool', profile: e.profile, sessionId: e.session_id, at: e.at,
        title: e.tool ?? 'tool', detail: e.preview ?? '', done: false, failed: false,
      })
    case 'tool.completed': {
      const idx = rows.findIndex((r) => r.kind === 'tool' && r.key === rowKey(e, e.call_id ?? ''))
      const failed = looksFailed(e.preview)
      if (idx >= 0) {
        const started = rows[idx]!
        const updated: FeedRow = {
          ...started, done: true, failed, result: e.preview ?? '',
          durationMs: Math.max(0, Date.parse(e.at) - Date.parse(started.at)),
        }
        return [...rows.slice(0, idx), updated, ...rows.slice(idx + 1)]
      }
      // Completion without a visible start (it happened before we subscribed).
      return prepend(rows, {
        key: rowKey(e, e.call_id ?? String(e.message_id)),
        kind: 'tool', profile: e.profile, sessionId: e.session_id, at: e.at,
        title: e.tool ?? 'tool', detail: '', result: e.preview ?? '', done: true, failed,
      })
    }
    case 'message':
      return prepend(rows, {
        key: rowKey(e, `m${e.message_id}`),
        kind: 'message', profile: e.profile, sessionId: e.session_id, at: e.at,
        title: e.role ?? 'message', detail: e.preview ?? '', done: true, failed: false,
      })
    case 'session.started':
    case 'session.ended':
    case 'subagent.start':
    case 'subagent.complete':
      return prepend(rows, {
        key: rowKey(e, `${e.type}:${e.session_id}`),
        kind: 'session', profile: e.profile, sessionId: e.session_id, at: e.at,
        title: sessionLabel(e), detail: sessionDetail(e), done: true, failed: false,
      })
    case 'error':
      return prepend(rows, {
        key: rowKey(e, `err:${e.at}`),
        kind: 'error', profile: e.profile, sessionId: e.session_id, at: e.at,
        title: e.error?.code ?? 'error', detail: e.error?.message ?? '', done: true, failed: true,
      })
    default:
      return rows // session.snapshot is state, handled separately
  }
}

/** Latest known session records keyed by profile/session id. */
export function applySnapshot(map: Map<string, Session & { profile: string }>, e: ActivityEvent): void {
  if (e.session) map.set(`${e.profile}/${e.session.id}`, { ...e.session, profile: e.profile })
}

function prepend(rows: FeedRow[], row: FeedRow): FeedRow[] {
  if (rows.some((r) => r.key === row.key)) return rows
  const next = [row, ...rows]
  return next.length > maxRows ? next.slice(0, maxRows) : next
}

function rowKey(e: ActivityEvent, id: string): string {
  return `${e.profile}/${e.session_id ?? ''}/${id}`
}

/** Hermes tool results are JSON with exit_code / error fields when they come from a process. */
export function looksFailed(preview: string | undefined): boolean {
  if (!preview) return false
  if (/"error":\s*(?!null)/.test(preview)) return true
  const m = /"exit_code":\s*(-?\d+)/.exec(preview)
  return m !== null && m[1] !== '0'
}

function sessionLabel(e: ActivityEvent): string {
  switch (e.type) {
    case 'session.started': return 'session started'
    case 'session.ended': return 'session ended'
    case 'subagent.start': return 'delegated'
    case 'subagent.complete': return 'delegation finished'
    default: return e.type
  }
}

function sessionDetail(e: ActivityEvent): string {
  const parts: string[] = []
  if (e.session?.source) parts.push(e.session.source)
  if (e.session?.title) parts.push(e.session.title)
  if (e.parent_session_id) parts.push(`from ${e.parent_session_id}`)
  if (e.end_reason) parts.push(e.end_reason)
  return parts.join(' · ')
}
