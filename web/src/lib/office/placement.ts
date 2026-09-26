// Where each agent belongs in the office, derived from the same Worker
// states as the grid (src/lib/floor.ts) — docs/decisions/023:
//
//   working, error   → seated at their own desk
//   delegating       → standing beside their most recent child's desk
//   idle             → own desk at first; the lounge once idle for 2 min
//   offline          → not in the office at all
//
// Stateless on purpose: "idle for 2 minutes" is measured from the
// profile's last known activity, so a page reload lands everyone where
// they'd have walked to anyway instead of replaying the wait.

import { WORKING_WINDOW_MS, type DelegationLink, type Worker } from '../floor'
import { DESK_COUNT } from './map'

/** How long an agent must have been idle before it walks to the lounge. */
export const LOUNGE_AFTER_IDLE_MS = 2 * 60_000

export type Placement =
  | { kind: 'absent' }
  | { kind: 'desk'; desk: number }
  | { kind: 'visiting'; desk: number; child: string }
  | { kind: 'lounge'; spot: number }

export interface OfficeOccupant {
  worker: Worker
  /** This profile's own desk slot. */
  desk: number
  placement: Placement
  /** Cross-profile parents currently delegating to this profile, in link order. */
  delegatedFrom: string[]
}

export interface OfficeScene {
  /** One per placed profile, in configuration (= desk) order. */
  occupants: OfficeOccupant[]
  /** Profiles beyond DESK_COUNT; they only appear in the grid view. */
  overflow: string[]
}

export function deriveOffice(workers: Worker[], links: DelegationLink[], nowMs: number): OfficeScene {
  const placed = workers.slice(0, DESK_COUNT)
  const deskOf = new Map(placed.map((wk, i) => [wk.profile, i]))

  const parentsOf = new Map<string, string[]>()
  for (const l of links) {
    if (!deskOf.has(l.childProfile)) continue
    const list = parentsOf.get(l.childProfile) ?? []
    if (!list.includes(l.parentProfile)) list.push(l.parentProfile)
    parentsOf.set(l.childProfile, list)
  }

  const occupants = placed.map((worker, desk): OfficeOccupant => ({
    worker,
    desk,
    placement: place(worker, desk, links, deskOf, nowMs),
    delegatedFrom: parentsOf.get(worker.profile) ?? [],
  }))
  return { occupants, overflow: workers.slice(DESK_COUNT).map((wk) => wk.profile) }
}

function place(
  worker: Worker,
  desk: number,
  links: DelegationLink[],
  deskOf: Map<string, number>,
  nowMs: number,
): Placement {
  switch (worker.state) {
    case 'offline':
      return { kind: 'absent' }
    case 'delegating': {
      const latest = links
        .filter((l) => l.parentProfile === worker.profile && deskOf.has(l.childProfile))
        .sort((a, b) => b.childStartedAt - a.childStartedAt)[0]
      return latest
        ? { kind: 'visiting', desk: deskOf.get(latest.childProfile)!, child: latest.childProfile }
        : { kind: 'desk', desk }
    }
    case 'idle': {
      const idleFor = worker.lastActiveAt === null ? Infinity : nowMs - worker.lastActiveAt - WORKING_WINDOW_MS
      return idleFor >= LOUNGE_AFTER_IDLE_MS ? { kind: 'lounge', spot: desk } : { kind: 'desk', desk }
    }
    default:
      return { kind: 'desk', desk }
  }
}

/** A stable string per destination, so a view can tell when an agent must move. */
export function placementKey(p: Placement): string {
  switch (p.kind) {
    case 'absent': return 'absent'
    case 'desk': return `desk:${p.desk}`
    case 'visiting': return `visit:${p.desk}`
    case 'lounge': return `lounge:${p.spot}`
  }
}
