import { describe, expect, it } from 'vitest'
import { applyEvent, looksFailed, type FeedRow } from './feed'
import type { ActivityEvent } from '@/api/types'

const base = { profile: 'default', session_id: 's1' }

describe('applyEvent', () => {
  it('merges tool.started and tool.completed into one row with duration', () => {
    let rows: FeedRow[] = []
    rows = applyEvent(rows, { ...base, type: 'tool.started', at: '2026-09-20T10:00:00Z', tool: 'terminal', call_id: 'c1', preview: '{"command":"ls"}' } as ActivityEvent)
    rows = applyEvent(rows, { ...base, type: 'tool.completed', at: '2026-09-20T10:00:02.500Z', tool: 'terminal', call_id: 'c1', preview: '{"output":"a b","exit_code":0,"error":null}' } as ActivityEvent)
    expect(rows).toHaveLength(1)
    expect(rows[0]).toMatchObject({ kind: 'tool', title: 'terminal', done: true, failed: false, durationMs: 2500, detail: '{"command":"ls"}' })
  })

  it('flags a non-zero exit code or error field as a tool failure, not a run failure', () => {
    expect(looksFailed('{"output":"","exit_code":1,"error":null}')).toBe(true)
    expect(looksFailed('{"error":"denied"}')).toBe(true)
    expect(looksFailed('{"output":"ok","exit_code":0,"error":null}')).toBe(false)
    expect(looksFailed('plain text')).toBe(false)
  })

  it('newest first, deduplicated, bounded', () => {
    let rows: FeedRow[] = []
    for (let i = 0; i < 300; i++) {
      rows = applyEvent(rows, { ...base, type: 'message', at: '2026-09-20T10:00:00Z', role: 'user', message_id: i, preview: 'x' } as ActivityEvent)
    }
    rows = applyEvent(rows, { ...base, type: 'message', at: '2026-09-20T10:00:00Z', role: 'user', message_id: 299, preview: 'x' } as ActivityEvent)
    expect(rows.length).toBeLessThanOrEqual(250)
    expect(rows[0]!.key).toContain('m299')
  })

  it('ignores snapshots and labels session events', () => {
    let rows: FeedRow[] = []
    rows = applyEvent(rows, { ...base, type: 'session.snapshot', at: '2026-09-20T10:00:00Z' } as ActivityEvent)
    expect(rows).toHaveLength(0)
    rows = applyEvent(rows, { ...base, type: 'subagent.start', at: '2026-09-20T10:00:00Z', parent_session_id: 'p', session: { source: 'kanban', title: 'Fix test' } } as unknown as ActivityEvent)
    expect(rows[0]).toMatchObject({ kind: 'session', title: 'delegated', detail: 'kanban · Fix test · from p' })
  })
})
