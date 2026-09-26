// Moving agents around the office (docs/decisions/023). Pure functions over
// a small Actor record so the renderer only has to call stepActor() each
// frame; routes come from BFS (path.ts) and are walked along x, then y, at
// a fixed speed, one waypoint after another.

import { TILE, seatOf, visitorSpotOf, type OfficeMap } from './map'
import { findPath, type Tile } from './path'
import type { Placement } from './placement'

export type Facing = 'down' | 'up' | 'left' | 'right'
export type Stance = 'seated' | 'standing'

export interface Pt {
  x: number
  y: number
}

/** A destination: the tile to path to, the exact pixel to stop on, and the pose to take there. */
export interface Spot {
  tile: Tile
  px: Pt
  facing: Facing
  pose: Stance
}

/**
 * One agent on the map. (x, y) is the top-left of the 16×16 box the sprite
 * stands in, in map pixels; the renderer rounds it to whole pixels.
 */
export interface Actor {
  x: number
  y: number
  waypoints: Pt[]
  facing: Facing
  pose: Stance
  /** Facing and pose to take once the last waypoint is reached. */
  rest: { facing: Facing; pose: Stance }
  visible: boolean
  /** Set when walking out of the door: the actor disappears on arrival. */
  hideOnArrival: boolean
}

/** Walking speed in map pixels per millisecond: 1/16 px/ms ≈ 3.9 tiles/s, exact in binary. */
export const WALK_SPEED = 1 / 16

export function spotFor(m: OfficeMap, p: Placement): Spot {
  const at = (t: Tile, facing: Facing, pose: Stance, dx = 0): Spot => ({
    tile: t, px: { x: t.x * TILE + dx, y: t.y * TILE }, facing, pose,
  })
  switch (p.kind) {
    case 'desk':
      // Centred across the two desk tiles.
      return at(seatOf(m.desks[p.desk]!), 'down', 'seated', TILE / 2)
    case 'visiting':
      return at(visitorSpotOf(m.desks[p.desk]!), 'left', 'standing')
    case 'lounge':
      return at(m.lounge[p.spot]!, 'down', 'standing')
    case 'absent':
      return at(m.door, 'down', 'standing')
  }
}

/** An actor already resting on a spot (used for the first render). */
export function placeAt(spot: Spot, visible: boolean): Actor {
  return {
    x: spot.px.x, y: spot.px.y, waypoints: [],
    facing: spot.facing, pose: spot.pose, rest: { facing: spot.facing, pose: spot.pose },
    visible, hideOnArrival: false,
  }
}

/**
 * Sends an actor to a spot. `instant` skips the walk (first render and
 * prefers-reduced-motion); `hideOnArrival` makes it vanish once there
 * (walking out of the door).
 */
export function moveTo(m: OfficeMap, a: Actor, spot: Spot, opts: { instant: boolean; hideOnArrival?: boolean }): Actor {
  const hide = opts.hideOnArrival ?? false
  const settled = () => placeAt(spot, !hide)
  if (opts.instant) return settled()

  const start = { x: Math.round(a.x / TILE), y: Math.round(a.y / TILE) }
  const path = findPath(m.blocked, start, spot.tile)
  if (path === null) return settled() // unreachable: better to jump than to freeze

  const waypoints: Pt[] = []
  const push = (p: Pt) => {
    const last = waypoints.at(-1) ?? { x: a.x, y: a.y }
    if (last.x !== p.x || last.y !== p.y) waypoints.push(p)
  }
  push({ x: start.x * TILE, y: start.y * TILE })
  for (const t of path) push({ x: t.x * TILE, y: t.y * TILE })
  push(spot.px)

  const moved: Actor = {
    ...a, waypoints, visible: true, hideOnArrival: hide,
    rest: { facing: spot.facing, pose: spot.pose },
  }
  return waypoints.length === 0 ? arrive(moved) : { ...moved, pose: 'standing' }
}

/** Advances an actor along its waypoints by dtMs of walking. */
export function stepActor(a: Actor, dtMs: number): Actor {
  if (a.waypoints.length === 0) return a
  let budget = dtMs * WALK_SPEED
  let { x, y, facing } = a
  const waypoints = [...a.waypoints]
  while (budget > 0 && waypoints.length > 0) {
    const t = waypoints[0]!
    if (x !== t.x) {
      const d = Math.min(budget, Math.abs(t.x - x))
      facing = t.x > x ? 'right' : 'left'
      x = d === Math.abs(t.x - x) ? t.x : x + Math.sign(t.x - x) * d
      budget -= d
    } else if (y !== t.y) {
      const d = Math.min(budget, Math.abs(t.y - y))
      facing = t.y > y ? 'down' : 'up'
      y = d === Math.abs(t.y - y) ? t.y : y + Math.sign(t.y - y) * d
      budget -= d
    }
    if (x === t.x && y === t.y) waypoints.shift()
  }
  const next: Actor = { ...a, x, y, facing, waypoints, pose: 'standing' }
  return waypoints.length === 0 ? arrive(next) : next
}

function arrive(a: Actor): Actor {
  return {
    ...a, waypoints: [], facing: a.rest.facing, pose: a.rest.pose,
    visible: a.visible && !a.hideOnArrival, hideOnArrival: false,
  }
}
