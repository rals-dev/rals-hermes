import { describe, expect, it } from 'vitest'
import { COLS, DESK_COUNT, ROWS, isWalkable, officeMap, seatOf, visitorSpotOf } from './map'
import { findPath } from './path'

const m = officeMap

describe('officeMap', () => {
  it('has the agreed shape: 8 desks, 8 lounge spots, one door', () => {
    expect(m.cols).toBe(COLS)
    expect(m.rows).toBe(ROWS)
    expect(m.blocked).toHaveLength(ROWS)
    for (const row of m.blocked) expect(row).toHaveLength(COLS)
    expect(m.desks).toHaveLength(DESK_COUNT)
    expect(m.desks.map((d) => d.index)).toEqual([0, 1, 2, 3, 4, 5, 6, 7])
    expect(m.lounge).toHaveLength(DESK_COUNT)
  })

  it('walls the border except where the door is', () => {
    for (let x = 0; x < COLS; x++) {
      expect(m.blocked[0]![x]).toBe(true)
      expect(m.blocked[ROWS - 1]![x]).toBe(true)
    }
    for (let y = 0; y < ROWS; y++) {
      expect(m.blocked[y]![0]).toBe(true)
      expect(m.blocked[y]![COLS - 1]).toBe(true)
    }
    expect(m.doorway.y).toBe(ROWS - 1)
    expect(isWalkable(m, m.door)).toBe(true)
  })

  it('blocks every solid furniture footprint and leaves the rest of a rug walkable', () => {
    const solid = new Set<string>()
    for (const f of m.furniture.filter((f) => f.solid)) {
      for (let y = f.y; y < f.y + f.h; y++) {
        for (let x = f.x; x < f.x + f.w; x++) {
          solid.add(`${x},${y}`)
          expect(m.blocked[y]![x], `${f.kind} at ${x},${y}`).toBe(true)
        }
      }
    }
    for (const f of m.furniture.filter((f) => !f.solid)) {
      for (let y = f.y; y < f.y + f.h; y++) {
        for (let x = f.x; x < f.x + f.w; x++) {
          if (!solid.has(`${x},${y}`)) expect(m.blocked[y]![x], `${f.kind} at ${x},${y}`).toBe(false)
        }
      }
    }
  })

  it('keeps every seat, visitor spot and lounge spot walkable and distinct', () => {
    const spots = [...m.desks.map(seatOf), ...m.desks.map(visitorSpotOf), ...m.lounge]
    for (const t of spots) expect(isWalkable(m, t), `${t.x},${t.y}`).toBe(true)
    const keys = new Set(spots.map((t) => `${t.x},${t.y}`))
    expect(keys.size).toBe(spots.length)
  })

  it('connects the door to every spot an agent can be sent to', () => {
    const spots = [...m.desks.map(seatOf), ...m.desks.map(visitorSpotOf), ...m.lounge]
    for (const t of spots) {
      expect(findPath(m.blocked, m.door, t), `door -> ${t.x},${t.y}`).not.toBeNull()
    }
  })
})
