// Derives the /floor view's worker state from data the app already has:
// the overview's agent cards, the latest session snapshot per profile (from
// useActivityFeed), and the live feed rows. No new backend endpoint.
//
// Delegation is tricky: Hermes emits subagent.start on the CHILD session
// (internal/activity/hub.go), carrying only the parent's session id, not
// its profile. So a delegation line can only be drawn once the parent
// session has also been seen — which happens when the parent's own profile
// stream has delivered at least one snapshot. Until then (or if the parent
// is outside the 10-minute activity window entirely) the child is shown
// with a "delegated" badge instead of a line. See docs/decisions/021 and
// floor.fixtures.ts for the three cases this module distinguishes; only
// same-profile delegation has been observed against a real Hermes so far.

import type { AgentCard, AgentStatus, Overview } from '@/api/types'
import type { SessionRecord } from './useActivityFeed'
import type { FeedRow } from './feed'

export type WorkerState = 'offline' | 'error' | 'delegating' | 'working' | 'idle'

export interface DelegationLink {
  parentProfile: string
  childProfile: string
  parentSessionId: string
  childSessionId: string
}

export interface Worker {
  profile: string
  status: AgentStatus | string
  state: WorkerState
  /** Name of the tool currently in flight; set only while state is 'working'. */
  toolName?: string
  /** This profile has an open child session whose parent session is unknown. */
  delegatedBadge: boolean
  /** Count of open child sessions delegated to within the same profile. */
  helperCount?: number
  openSessions: number
  toolCallsRecent: number
}

export interface FloorState {
  workers: Worker[]
  links: DelegationLink[]
}

const WORKING_WINDOW_MS = 60_000
const ERROR_WINDOW_MS = 30_000
const RECENT_TOOL_WINDOW_MS = 10 * 60_000

export function deriveFloor(
  overview: Overview | null | undefined,
  sessions: Map<string, SessionRecord>,
  rows: FeedRow[],
  now: Date = new Date(),
): FloorState {
  const agents = overview?.agents ?? []
  const nowMs = now.getTime()

  // Every known session, indexed by its own id, so a child's
  // parent_session_id can be resolved to a profile.
  const byId = new Map<string, SessionRecord>()
  for (const s of sessions.values()) byId.set(s.id, s)

  const links: DelegationLink[] = []
  const delegatingProfiles = new Set<string>()
  const helperCounts = new Map<string, number>()
  const unknownParentProfiles = new Set<string>()

  for (const s of sessions.values()) {
    if (!s.parent_session_id || !s.open) continue
    const parent = byId.get(s.parent_session_id)
    if (!parent) {
      unknownParentProfiles.add(s.profile)
      continue
    }
    if (parent.profile === s.profile) {
      helperCounts.set(s.profile, (helperCounts.get(s.profile) ?? 0) + 1)
      continue
    }
    links.push({
      parentProfile: parent.profile,
      childProfile: s.profile,
      parentSessionId: parent.id,
      childSessionId: s.id,
    })
    delegatingProfiles.add(parent.profile)
  }

  const recentErrorProfiles = new Set<string>()
  for (const r of rows) {
    if (!r.failed) continue
    const age = nowMs - Date.parse(r.at)
    if (age >= 0 && age < ERROR_WINDOW_MS) recentErrorProfiles.add(r.profile)
  }

  // First tool.started still missing its tool.completed, per profile — the
  // rows array is newest-first, so the first match is the most recent start.
  const runningTool = new Map<string, string>()
  const toolCallsRecent = new Map<string, number>()
  for (const r of rows) {
    if (r.kind !== 'tool') continue
    if (!r.done && !runningTool.has(r.profile)) runningTool.set(r.profile, r.title)
    const age = nowMs - Date.parse(r.at)
    if (age >= 0 && age < RECENT_TOOL_WINDOW_MS) {
      toolCallsRecent.set(r.profile, (toolCallsRecent.get(r.profile) ?? 0) + 1)
    }
  }

  const openSessions = new Map<string, number>()
  const recentlyActive = new Set<string>()
  for (const s of sessions.values()) {
    if (!s.open) continue
    openSessions.set(s.profile, (openSessions.get(s.profile) ?? 0) + 1)
    const age = nowMs - Date.parse(s.last_active)
    if (age >= 0 && age < WORKING_WINDOW_MS) recentlyActive.add(s.profile)
  }

  const workers: Worker[] = agents.map((a) => {
    const state = deriveState(a, {
      hasRecentError: recentErrorProfiles.has(a.profile),
      isDelegating: delegatingProfiles.has(a.profile),
      runningTool: runningTool.get(a.profile),
      isRecentlyActive: recentlyActive.has(a.profile),
    })
    return {
      profile: a.profile,
      status: a.status,
      state,
      toolName: state === 'working' ? runningTool.get(a.profile) : undefined,
      delegatedBadge: unknownParentProfiles.has(a.profile),
      helperCount: helperCounts.get(a.profile),
      openSessions: openSessions.get(a.profile) ?? 0,
      toolCallsRecent: toolCallsRecent.get(a.profile) ?? 0,
    }
  })

  return { workers, links }
}

function deriveState(
  agent: AgentCard,
  ctx: { hasRecentError: boolean; isDelegating: boolean; runningTool: string | undefined; isRecentlyActive: boolean },
): WorkerState {
  if (agent.status === 'unreachable' || agent.status === 'unauthorized') return 'offline'
  if (ctx.hasRecentError) return 'error'
  if (ctx.isDelegating) return 'delegating'
  if (ctx.runningTool || ctx.isRecentlyActive) return 'working'
  return 'idle'
}
