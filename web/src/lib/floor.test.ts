import { describe, expect, it } from 'vitest'
import { deriveFloor } from './floor'
import type { AgentCard, Overview, Session } from '@/api/types'
import type { SessionRecord } from './useActivityFeed'
import type { FeedRow } from './feed'

const NOW = new Date('2026-09-20T10:10:00Z')

function agent(profile: string, status: AgentCard['status'] = 'healthy'): AgentCard {
  return { profile, status, latency_ms: 12 }
}

function overview(agents: AgentCard[]): Overview {
  return { generated_at: NOW.toISOString(), gateway: null, agents }
}

function session(over: Partial<Session> & { profile: string }): SessionRecord {
  return {
    id: over.id ?? `${over.profile}-s1`,
    source: 'api',
    model: 'x',
    title: null,
    preview: null,
    started_at: NOW.toISOString(),
    ended_at: null,
    end_reason: null,
    open: true,
    last_active: NOW.toISOString(),
    message_count: 1,
    tool_call_count: 0,
    api_call_count: 0,
    parent_session_id: null,
    usage: { input_tokens: 0, output_tokens: 0, cache_read_tokens: 0, cache_write_tokens: 0, reasoning_tokens: 0 },
    estimated_cost_usd: null,
    actual_cost_usd: null,
    ...over,
  }
}

function sessionMap(records: SessionRecord[]): Map<string, SessionRecord> {
  const m = new Map<string, SessionRecord>()
  for (const r of records) m.set(`${r.profile}/${r.id}`, r)
  return m
}

function row(over: Partial<FeedRow> & { profile: string; kind: FeedRow['kind'] }): FeedRow {
  return {
    key: over.key ?? `${over.profile}/k${Math.random()}`,
    at: NOW.toISOString(),
    title: 'tool',
    detail: '',
    done: true,
    failed: false,
    ...over,
  }
}

describe('deriveFloor', () => {
  it('marks unreachable/unauthorized profiles offline regardless of other signals', () => {
    const { workers } = deriveFloor(
      overview([agent('default', 'unreachable'), agent('coder-agent', 'unauthorized')]),
      sessionMap([]),
      [row({ profile: 'default', kind: 'tool', done: false })],
      NOW,
    )
    expect(workers.map((w) => w.state)).toEqual(['offline', 'offline'])
  })

  it('shows error for a tool failure within the last 30s, not older', () => {
    const fresh = row({ profile: 'default', kind: 'tool', failed: true, at: new Date(NOW.getTime() - 10_000).toISOString() })
    const stale = row({ profile: 'coder-agent', kind: 'tool', failed: true, at: new Date(NOW.getTime() - 40_000).toISOString() })
    const { workers } = deriveFloor(overview([agent('default'), agent('coder-agent')]), sessionMap([]), [fresh, stale], NOW)
    expect(workers.find((w) => w.profile === 'default')!.state).toBe('error')
    expect(workers.find((w) => w.profile === 'coder-agent')!.state).toBe('idle')
  })

  it('draws a cross-profile delegation link and marks the parent delegating', () => {
    const parent = session({ profile: 'default', id: 'p1' })
    const child = session({ profile: 'coder-agent', id: 'c1', parent_session_id: 'p1' })
    const { workers, links } = deriveFloor(
      overview([agent('default'), agent('coder-agent')]),
      sessionMap([parent, child]),
      [],
      NOW,
    )
    expect(links).toEqual([{
      parentProfile: 'default', childProfile: 'coder-agent', parentSessionId: 'p1', childSessionId: 'c1',
      childStartedAt: NOW.getTime(),
    }])
    expect(workers.find((w) => w.profile === 'default')!.state).toBe('delegating')
  })

  it('reports a same-profile delegation as a helper badge, not a line', () => {
    const parent = session({ profile: 'default', id: 'p1' })
    const child = session({ profile: 'default', id: 'c1', parent_session_id: 'p1' })
    const { workers, links } = deriveFloor(overview([agent('default')]), sessionMap([parent, child]), [], NOW)
    expect(links).toEqual([])
    expect(workers[0]!.helperCount).toBe(1)
  })

  it('badges a child whose parent session is unknown, without drawing a line', () => {
    const child = session({ profile: 'coder-agent', id: 'c1', parent_session_id: 'gone' })
    const { workers, links } = deriveFloor(overview([agent('coder-agent')]), sessionMap([child]), [], NOW)
    expect(links).toEqual([])
    expect(workers[0]!.delegatedBadge).toBe(true)
  })

  it('is working with a tool-name bubble while a tool call is in flight', () => {
    const { workers } = deriveFloor(
      overview([agent('default')]),
      sessionMap([]),
      [row({ profile: 'default', kind: 'tool', title: 'terminal', done: false })],
      NOW,
    )
    expect(workers[0]).toMatchObject({ state: 'working', toolName: 'terminal' })
  })

  it('is working from recent session activity even with no in-flight tool row', () => {
    const s = session({ profile: 'default', last_active: new Date(NOW.getTime() - 30_000).toISOString() })
    const { workers } = deriveFloor(overview([agent('default')]), sessionMap([s]), [], NOW)
    expect(workers[0]!.state).toBe('working')
  })

  it('falls back to idle once activity is older than the working window', () => {
    const s = session({ profile: 'default', last_active: new Date(NOW.getTime() - 90_000).toISOString() })
    const { workers } = deriveFloor(overview([agent('default')]), sessionMap([s]), [], NOW)
    expect(workers[0]!.state).toBe('idle')
  })

  it('orders states by priority: offline > error > delegating > working', () => {
    const parent = session({ profile: 'default', id: 'p1', last_active: NOW.toISOString() })
    const child = session({ profile: 'coder-agent', id: 'c1', parent_session_id: 'p1' })
    const failedRow = row({ profile: 'default', kind: 'tool', failed: true, at: new Date(NOW.getTime() - 5_000).toISOString() })
    const { workers } = deriveFloor(overview([agent('default')]), sessionMap([parent, child]), [failedRow], NOW)
    // default is both actively working and mid-delegation, but the recent failure wins.
    expect(workers[0]!.state).toBe('error')
  })

  it('counts open sessions and recent tool calls per profile', () => {
    const s1 = session({ profile: 'default', id: 's1', open: true })
    const s2 = session({ profile: 'default', id: 's2', open: false, ended_at: NOW.toISOString() })
    const recentTool = row({ profile: 'default', kind: 'tool', at: new Date(NOW.getTime() - 60_000).toISOString() })
    const oldTool = row({ profile: 'default', kind: 'tool', at: new Date(NOW.getTime() - 20 * 60_000).toISOString() })
    const { workers } = deriveFloor(overview([agent('default')]), sessionMap([s1, s2]), [recentTool, oldTool], NOW)
    expect(workers[0]).toMatchObject({ openSessions: 1, toolCallsRecent: 1 })
  })

  it('reports the latest activity per profile from sessions and feed rows, null when none', () => {
    const old = session({ profile: 'default', id: 's1', open: false, last_active: new Date(NOW.getTime() - 600_000).toISOString() })
    const newer = session({ profile: 'default', id: 's2', last_active: new Date(NOW.getTime() - 300_000).toISOString() })
    const newestRow = row({ profile: 'default', kind: 'message', at: new Date(NOW.getTime() - 100_000).toISOString() })
    const { workers } = deriveFloor(
      overview([agent('default'), agent('product-agent')]),
      sessionMap([old, newer]),
      [newestRow],
      NOW,
    )
    expect(workers.find((w) => w.profile === 'default')!.lastActiveAt).toBe(NOW.getTime() - 100_000)
    expect(workers.find((w) => w.profile === 'product-agent')!.lastActiveAt).toBeNull()
  })

  it('preserves the overview profile order', () => {
    const { workers } = deriveFloor(
      overview([agent('product-agent'), agent('default'), agent('tester-agent')]),
      sessionMap([]),
      [],
      NOW,
    )
    expect(workers.map((w) => w.profile)).toEqual(['product-agent', 'default', 'tester-agent'])
  })
})
