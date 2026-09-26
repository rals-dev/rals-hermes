import { describe, expect, it } from 'vitest'
import { findPath } from './path'

// '#' blocked, '.' walkable — rows top to bottom.
function grid(rows: string[]): boolean[][] {
  return rows.map((r) => [...r].map((c) => c === '#'))
}

describe('findPath', () => {
  it('returns an empty path when already at the goal', () => {
    expect(findPath(grid(['...']), { x: 1, y: 0 }, { x: 1, y: 0 })).toEqual([])
  })

  it('finds a shortest route around a wall, excluding the start and including the goal', () => {
    const g = grid([
      '.....',
      '.###.',
      '.....',
    ])
    const path = findPath(g, { x: 0, y: 1 }, { x: 4, y: 1 })!
    expect(path).not.toBeNull()
    expect(path.length).toBe(6) // 2 up/down + 4 across
    expect(path[path.length - 1]).toEqual({ x: 4, y: 1 })
    expect(path).not.toContainEqual({ x: 0, y: 1 })
  })

  it('only ever steps to 4-neighbours on walkable tiles', () => {
    const g = grid([
      '..#....',
      '..#.##.',
      '....#..',
    ])
    const from = { x: 0, y: 0 }
    const path = findPath(g, from, { x: 6, y: 0 })!
    let prev = from
    for (const t of path) {
      expect(Math.abs(t.x - prev.x) + Math.abs(t.y - prev.y)).toBe(1)
      expect(g[t.y]![t.x]).toBe(false)
      prev = t
    }
  })

  it('returns null for an unreachable or blocked goal', () => {
    const g = grid([
      '.#.',
      '.#.',
    ])
    expect(findPath(g, { x: 0, y: 0 }, { x: 2, y: 0 })).toBeNull()
    expect(findPath(g, { x: 0, y: 0 }, { x: 1, y: 0 })).toBeNull()
    expect(findPath(g, { x: 0, y: 0 }, { x: 9, y: 9 })).toBeNull()
  })
})
