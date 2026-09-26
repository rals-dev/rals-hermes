// Breadth-first pathfinding on the office tile grid (docs/decisions/023).
// The map is ~600 tiles, so BFS is instant and, unlike A*, needs no
// heuristic tuning. Moves are 4-directional to match the walk sprites.

export interface Tile {
  x: number
  y: number
}

// Fixed neighbour order keeps routes deterministic (same inputs, same path).
const STEPS: readonly Tile[] = [
  { x: 0, y: -1 },
  { x: 1, y: 0 },
  { x: 0, y: 1 },
  { x: -1, y: 0 },
]

/**
 * Shortest route from `from` to `to` over tiles where `blocked[y][x]` is
 * false. The result excludes `from` and ends with `to`; it is empty when
 * already there and null when `to` is blocked, off the map or unreachable.
 */
export function findPath(blocked: boolean[][], from: Tile, to: Tile): Tile[] | null {
  const rows = blocked.length
  const cols = blocked[0]?.length ?? 0
  const inside = (t: Tile) => t.x >= 0 && t.y >= 0 && t.x < cols && t.y < rows
  if (!inside(to) || blocked[to.y]![to.x]) return null
  if (from.x === to.x && from.y === to.y) return []

  const key = (t: Tile) => t.y * cols + t.x
  const cameFrom = new Map<number, number>([[key(from), -1]])
  const queue: Tile[] = [from]
  for (let head = 0; head < queue.length; head++) {
    const cur = queue[head]!
    for (const d of STEPS) {
      const next = { x: cur.x + d.x, y: cur.y + d.y }
      if (!inside(next) || blocked[next.y]![next.x] || cameFrom.has(key(next))) continue
      cameFrom.set(key(next), key(cur))
      if (next.x === to.x && next.y === to.y) return unwind(cameFrom, key(next), key(from), cols)
      queue.push(next)
    }
  }
  return null
}

function unwind(cameFrom: Map<number, number>, end: number, start: number, cols: number): Tile[] {
  const path: Tile[] = []
  for (let k = end; k !== start; k = cameFrom.get(k)!) {
    path.push({ x: k % cols, y: Math.floor(k / cols) })
  }
  return path.reverse()
}
