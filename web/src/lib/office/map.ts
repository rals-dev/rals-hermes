// The office floor plan for the /floor "Office" view (docs/decisions/023):
// a fixed, hand-laid 32×18 tile map with eight desk slots, a lounge, a
// (decorative) meeting room and one door. Fixed rather than generated from
// the profile count — hand-placed rooms are far easier to make look right,
// and eight desks cover the 1–8 profiles the floor view is designed for.
//
// Layout, in tiles: walls on rows 0–1 (cap + face, with windows and a
// whiteboard), row 17 and columns 0/31. Desks 0–3 sit on row 5 and 4–7 on
// row 11, at columns 3/7/11/15, each two tiles wide with its seat on the row
// above. The lounge (sofa, coffee machine, coffee table on a rug) fills
// columns 21–30, rows 2–8; the meeting room sits behind glass at columns
// 21–30, rows 10–16, entered at column 25. The door is at column 2 of the
// bottom wall.

import type { Tile } from './path'

export const TILE = 16
export const COLS = 32
export const ROWS = 18
export const DESK_COUNT = 8

export type FurnitureKind =
  | 'desk'
  | 'sofa'
  | 'coffeeTable'
  | 'coffeeMachine'
  | 'plant'
  | 'bookshelf'
  | 'meetingTable'
  | 'chair'
  | 'glass'
  | 'whiteboard'
  | 'window'
  | 'rug'

export interface Furniture {
  kind: FurnitureKind
  x: number
  y: number
  w: number
  h: number
  /** Solid furniture blocks walking; only rugs are not solid. */
  solid: boolean
}

/** A desk's surface covers tiles (x, y) and (x+1, y); its occupant sits on the row above. */
export interface Desk {
  index: number
  x: number
  y: number
}

export interface OfficeMap {
  cols: number
  rows: number
  /** blocked[y][x]: true for walls and solid furniture. */
  blocked: boolean[][]
  furniture: Furniture[]
  desks: Desk[]
  /** Standing spots in the lounge, one per desk slot. */
  lounge: Tile[]
  /** The walkable tile just inside the door: where absent agents enter and leave. */
  door: Tile
  /** The wall tile the door itself is drawn on. */
  doorway: Tile
}

/** Where a desk's occupant sits (the row above the desk surface). */
export function seatOf(d: Desk): Tile {
  return { x: d.x, y: d.y - 1 }
}

/** Where a visiting (delegating) agent stands: beside the occupant, one gap column over. */
export function visitorSpotOf(d: Desk): Tile {
  return { x: d.x + 2, y: d.y - 1 }
}

export function isWalkable(m: OfficeMap, t: Tile): boolean {
  return t.x >= 0 && t.y >= 0 && t.x < m.cols && t.y < m.rows && !m.blocked[t.y]![t.x]
}

const DESK_COLUMNS = [3, 7, 11, 15]
const DESK_ROWS = [5, 11]

function buildOfficeMap(): OfficeMap {
  const piece = (kind: FurnitureKind, x: number, y: number, w = 1, h = 1): Furniture => ({
    kind, x, y, w, h, solid: kind !== 'rug',
  })

  const desks: Desk[] = []
  for (const y of DESK_ROWS) {
    for (const x of DESK_COLUMNS) desks.push({ index: desks.length, x, y })
  }

  const furniture: Furniture[] = [
    // Wall decor (on already-blocked wall tiles).
    piece('window', 3, 1, 2),
    piece('whiteboard', 8, 1, 3),
    piece('window', 14, 1, 2),
    piece('window', 23, 1, 2),
    // Open-plan desk area.
    ...desks.map((d) => piece('desk', d.x, d.y, 2)),
    piece('bookshelf', 10, 2, 3),
    piece('plant', 1, 2),
    piece('plant', 19, 2),
    piece('plant', 1, 15),
    piece('plant', 19, 16),
    // Lounge.
    piece('rug', 21, 3, 9, 6),
    piece('sofa', 22, 2, 4),
    piece('coffeeMachine', 28, 2),
    piece('plant', 30, 2),
    piece('coffeeTable', 24, 5, 2),
    // Meeting room (decorative), glass partition with a doorway at x = 25.
    piece('glass', 21, 10, 4),
    piece('glass', 26, 10, 5),
    piece('glass', 21, 11, 1, 6),
    piece('meetingTable', 24, 13, 4, 2),
    ...[24, 25, 26, 27].flatMap((x) => [piece('chair', x, 12), piece('chair', x, 15)]),
  ]

  const blocked = Array.from({ length: ROWS }, (_, y) =>
    Array.from({ length: COLS }, (_, x) => y <= 1 || y === ROWS - 1 || x === 0 || x === COLS - 1),
  )
  for (const f of furniture) {
    if (!f.solid) continue
    for (let y = f.y; y < f.y + f.h; y++) {
      for (let x = f.x; x < f.x + f.w; x++) blocked[y]![x] = true
    }
  }

  const lounge: Tile[] = [4, 7].flatMap((y) => [22, 24, 26, 28].map((x) => ({ x, y })))

  return {
    cols: COLS,
    rows: ROWS,
    blocked,
    furniture,
    desks,
    lounge,
    door: { x: 2, y: ROWS - 2 },
    doorway: { x: 2, y: ROWS - 1 },
  }
}

export const officeMap: OfficeMap = buildOfficeMap()
