// Hand-authored pixel art for the /floor Office view (docs/decisions/023).
// Characters reuse the grid view's head, palette and per-profile shirt
// colour (src/lib/sprites.ts) so an agent looks the same in both views;
// what's new here is the standing body, walk cycles in four directions
// (left is the mirror of right), a talking gesture for visitors, and a
// small helper figure. Furniture is a list of flat rectangles per piece —
// deliberately simple "programmer art", in full colour like the sprites.

import type { WorkerState } from '../floor'
import { HEAD, HEADROOM, framesFor, palette, type Pose } from '../sprites'
import type { FurnitureKind } from './map'
import type { Facing, Stance } from './motion'

/** Sprite palette: the grid view's, plus trousers. 'S' (shirt) stays per-profile. */
export const spritePalette: Record<string, string> = { ...palette, n: '#3a4a6b' }

export function mirror(grid: readonly string[]): string[] {
  return grid.map((row) => [...row].reverse().join(''))
}

function withRows(base: readonly string[], rows: Record<number, string>): string[] {
  return base.map((row, i) => rows[i] ?? row)
}

// ── Front (walking towards the viewer) ─────────────────────────────────
const STAND_FRONT: string[] = [
  ...HEADROOM,
  ...HEAD,
  '...SSSSSSSS...', // 9  shoulders
  '..fSSSSSSSSf..', // 10 arms
  '..fSSSSSSSSf..', // 11
  '...SSSSSSSS...', // 12
  '...nnnnnnnn...', // 13 hips
  '...nnn..nnn...', // 14 legs
  '...nnn..nnn...', // 15
  '...eee..eee...', // 16 shoes
]
const BLINK_FRONT = withRows(STAND_FRONT, { 5: '...hffffffh...' })
const WALK_FRONT = [
  withRows(STAND_FRONT, { 15: '...nnn..eee...', 16: '...eee........' }),
  withRows(STAND_FRONT, { 15: '...eee..nnn...', 16: '........eee...' }),
]

// ── Back (walking away): hair where the face was ───────────────────────
const STAND_BACK = withRows(STAND_FRONT, {
  4: '...hhhhhhhh...',
  5: '...hhhhhhhh...',
  6: '...hhhhhhhh...',
  7: '....hhhhhh....',
})
const WALK_BACK = [
  withRows(STAND_BACK, { 15: '...nnn..eee...', 16: '...eee........' }),
  withRows(STAND_BACK, { 15: '...eee..nnn...', 16: '........eee...' }),
]

// ── Side, facing right (left is mirrored) ──────────────────────────────
const SIDE: string[] = [
  ...HEADROOM,
  '.....hhhhh....', // 3
  '....hhhhhff...', // 4
  '....hhhffef...', // 5  one eye
  '....hhfffff...', // 6
  '.....fffff....', // 7
  '......fff.....', // 8
  '.....SSSSS....', // 9
  '.....SSSSS....', // 10
  '.....SSSSS....', // 11
  '.....SSfSS....', // 12 hand at the side
  '.....nnnnn....', // 13
  '......nnn.....', // 14 legs together
  '......nnn.....', // 15
  '......eeee....', // 16
]
const WALK_SIDE = [
  withRows(SIDE, {
    11: '.....SSSSSf...', // arm swung forward
    12: '.....SSSSS....',
    14: '....nn..nn....',
    15: '...nn....nn...',
    16: '...ee.....ee..',
  }),
  SIDE,
]
// Talking: the front hand bobs, as if explaining something.
const TALK_SIDE = [
  withRows(SIDE, { 10: '.....SSSSSf...', 12: '.....SSSSS....' }),
  withRows(SIDE, { 11: '.....SSSSSf...', 12: '.....SSSSS....' }),
]

/** Small figure beside a desk: a same-profile sub-agent (wears the owner's shirt). */
export const HELPER: readonly string[] = [
  '.hhhh.',
  'hffffh',
  'hfefeh',
  '.ffff.',
  'SSSSSS',
  'SSSSSS',
  '.n..n.',
  '.e..e.',
]

export interface CharacterView {
  walking: boolean
  facing: Facing
  pose: Stance
  state: WorkerState
}

const SEATED_POSE: Record<WorkerState, Pose> = {
  working: 'working', idle: 'idle', error: 'error', delegating: 'delegating', offline: 'idle',
}

/**
 * Frames to cycle for an actor, all spriteCols × spriteRows. Seated agents
 * use the grid view's desk frames unchanged; standing ones get the new
 * bodies. Some frames repeat to slow an animation down at the fixed 5 fps
 * idle tick (a blink every ~1.6 s, a gesture every ~0.8 s).
 */
export function characterFrames(v: CharacterView): string[][] {
  if (v.walking) {
    switch (v.facing) {
      case 'down': return WALK_FRONT
      case 'up': return WALK_BACK
      case 'right': return WALK_SIDE
      case 'left': return WALK_SIDE.map(mirror)
    }
  }
  if (v.pose === 'seated') return framesFor(SEATED_POSE[v.state])
  switch (v.facing) {
    case 'down': return [...Array<string[]>(7).fill(STAND_FRONT), BLINK_FRONT]
    case 'up': return [STAND_BACK]
    case 'right': return [TALK_SIDE[0]!, TALK_SIDE[0]!, TALK_SIDE[1]!, TALK_SIDE[1]!]
    case 'left': return [TALK_SIDE[0]!, TALK_SIDE[0]!, TALK_SIDE[1]!, TALK_SIDE[1]!].map(mirror)
  }
}

// ── Office colours ─────────────────────────────────────────────────────
export const officePalette: Record<string, string> = {
  floor: '#caa67f', floorSeam: '#b58f69', floorLight: '#d6b48e',
  carpet: '#8ea3b7', carpetSeam: '#7f94a8',
  wallCap: '#3a3844', wallFace: '#e8ddcc', wallBase: '#8b7a66', mat: '#6b5a48',
  deskTop: '#b07a4c', deskEdge: '#8c5c36', deskFront: '#6f4629', deskLeg: '#4b2f1c',
  monitor: '#2b2d33', monitorStand: '#555a63', screenOff: '#1b252c', screenOn: '#79d4f2', screenError: '#e2483a',
  keyboard: '#d9d6cf',
  chair: '#3f4c63', chairDark: '#2c3647',
  sofa: '#5b7db0', sofaDark: '#46628f', sofaLight: '#7596c6',
  tableTop: '#9c6a42', tableEdge: '#7a5133',
  machine: '#3b3b3f', machineAccent: '#c0473a', cup: '#f3efe7',
  pot: '#b8662a', potDark: '#8f4d1c', leaf: '#4f9d4a', leafDark: '#3a7a36', leafLight: '#6fbf62',
  shelf: '#7a4e2d', shelfDark: '#5c3a21',
  book1: '#c0473a', book2: '#2f7fc1', book3: '#e0b52c', book4: '#3a9d5d', book5: '#8b54b0',
  glass: 'rgba(185, 222, 238, 0.55)', glassFrame: '#9aa7b0',
  board: '#f4f3ee', boardFrame: '#9a9a9a', marker1: '#3f78d0', marker2: '#d2584a',
  windowFrame: '#6b5a48', windowGlass: '#9fd3e6', windowShine: '#dff3f9',
  rug: '#b95a4d', rugInner: '#cf7766', rugPattern: '#e0a07f',
}

/** [x, y, w, h, colour] in pixels from the piece's top-left tile; y may go up to one tile above. */
export type Shape = readonly [number, number, number, number, string]

/** The monitor screen on a desk, relative to the desk — repainted live (on/off/error). */
export const MONITOR_SCREEN: Shape = [2, -7, 6, 6, 'screenOff']

const BOOKS = ['book1', 'book2', 'book3', 'book4', 'book5']

export function furnitureShapes(kind: FurnitureKind, w: number, h: number): Shape[] {
  const W = w * 16
  const H = h * 16
  switch (kind) {
    case 'desk':
      return [
        [10, -12, 12, 10, 'chair'], [10, -3, 12, 2, 'chairDark'], // empty chair; hidden behind a seated agent
        [0, 0, 32, 11, 'deskTop'], [0, 10, 32, 1, 'deskEdge'], [0, 11, 32, 3, 'deskFront'],
        [1, 14, 2, 2, 'deskLeg'], [29, 14, 2, 2, 'deskLeg'],
        [4, 1, 2, 3, 'monitorStand'], [1, -8, 8, 9, 'monitor'], MONITOR_SCREEN,
        [12, 3, 9, 2, 'keyboard'],
      ]
    case 'sofa':
      return [
        [0, -6, W, 8, 'sofaDark'], [2, 2, W - 4, 9, 'sofa'], [2, 11, W - 4, 2, 'sofaDark'],
        [0, 0, 4, 13, 'sofaDark'], [W - 4, 0, 4, 13, 'sofaDark'],
        ...Array.from({ length: w - 1 }, (_, i): Shape => [(i + 1) * 16 - 1, 3, 1, 7, 'sofaDark']),
        ...Array.from({ length: w }, (_, i): Shape => [i * 16 + 4, 3, 9, 2, 'sofaLight']),
      ]
    case 'coffeeTable':
      return [
        [2, 3, W - 4, 8, 'tableTop'], [2, 11, W - 4, 2, 'tableEdge'],
        [3, 13, 2, 3, 'tableEdge'], [W - 5, 13, 2, 3, 'tableEdge'],
        [8, 5, 3, 3, 'cup'], [W - 12, 4, 6, 4, 'book2'],
      ]
    case 'coffeeMachine':
      return [[3, -6, 10, 20, 'machine'], [5, -3, 6, 3, 'machineAccent'], [7, 4, 2, 2, 'monitorStand'], [6, 8, 4, 4, 'cup']]
    case 'plant':
      return [
        [3, -8, 10, 8, 'leaf'], [1, -4, 5, 5, 'leafDark'], [10, -5, 5, 6, 'leafDark'], [6, -10, 4, 4, 'leafLight'],
        [5, -2, 6, 4, 'leaf'], [4, 2, 8, 10, 'pot'], [3, 2, 10, 2, 'potDark'], [4, 10, 8, 2, 'potDark'],
      ]
    case 'bookshelf': {
      const shapes: Shape[] = [[0, -12, W, 26, 'shelf'], [2, -10, W - 4, 7, 'shelfDark'], [2, -1, W - 4, 7, 'shelfDark']]
      for (const top of [-10, -1]) {
        for (let x = 3, i = top === -10 ? 0 : 2; x + 3 <= W - 3; x += 4, i++) {
          const short = i % 3 === 1
          shapes.push([x, top + (short ? 1 : 0), 3, short ? 6 : 7, BOOKS[i % BOOKS.length]!])
        }
      }
      shapes.push([0, 14, W, 2, 'shelfDark'])
      return shapes
    }
    case 'meetingTable':
      return [
        [0, 4, W, H - 10, 'tableTop'], [0, H - 6, W, 3, 'tableEdge'],
        [2, H - 3, 3, 3, 'tableEdge'], [W - 5, H - 3, 3, 3, 'tableEdge'],
        [10, 10, 8, 6, 'board'], [W - 24, 12, 7, 5, 'board'], [W / 2 - 2, 14, 3, 3, 'cup'],
      ]
    case 'chair':
      return [[3, 2, 10, 3, 'chairDark'], [3, 5, 10, 7, 'chair'], [4, 12, 2, 2, 'chairDark'], [10, 12, 2, 2, 'chairDark']]
    case 'glass':
      if (h === 1) {
        return [
          [0, 0, W, 11, 'glass'], [0, 11, W, 2, 'glassFrame'],
          ...Array.from({ length: w - 1 }, (_, i): Shape => [(i + 1) * 16 - 1, 0, 1, 11, 'glassFrame']),
        ]
      }
      return [[5, 0, 6, H, 'glass'], [4, 0, 1, H, 'glassFrame'], [11, 0, 1, H, 'glassFrame']]
    case 'whiteboard':
      return [
        [2, 1, W - 4, 12, 'boardFrame'], [3, 2, W - 6, 10, 'board'],
        [6, 4, 14, 1, 'marker1'], [6, 7, 20, 1, 'marker1'], [W - 20, 5, 10, 1, 'marker2'], [W - 20, 8, 6, 1, 'marker2'],
        [8, 13, W - 16, 1, 'boardFrame'],
      ]
    case 'window':
      return [
        [2, 1, W - 4, 12, 'windowFrame'], [3, 2, W / 2 - 4, 10, 'windowGlass'], [W / 2 + 1, 2, W / 2 - 4, 10, 'windowGlass'],
        [4, 3, 3, 5, 'windowShine'], [W / 2 + 2, 3, 3, 5, 'windowShine'],
      ]
    case 'rug':
      return [
        [0, 0, W, H, 'rug'], [4, 4, W - 8, H - 8, 'rugInner'],
        [8, 8, W - 16, 2, 'rugPattern'], [8, H - 10, W - 16, 2, 'rugPattern'],
        [8, 8, 2, H - 16, 'rugPattern'], [W - 10, 8, 2, H - 16, 'rugPattern'],
      ]
  }
}
