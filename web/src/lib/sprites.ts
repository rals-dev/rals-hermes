// Hand-authored pixel-art data for the /floor page (docs/decisions/021).
// Full colour, deliberately unlike the rest of the instrument-panel theme —
// see the ADR for why. Each sprite is a grid of single-character rows; '.'
// is transparent, every other character is a palette key resolved by
// PixelSprite.vue. A worker's shirt is the one dynamic colour, picked
// deterministically from `shirtColor` so the same profile always wears the
// same colour.
//
// Frames are BASE (head + torso, per pose where the torso itself doesn't
// change) merged with a small overlay grid carrying only the moving parts
// (arms, phone, exclamation mark). This keeps each pose's art to the few
// pixels that actually move, instead of redrawing the whole figure per
// frame. Swap this module for a real sprite-sheet later without touching
// PixelSprite.vue's rendering code — see its header comment.

export const spriteCols = 14
export const spriteRows = 17

export type Pose = 'idle' | 'working' | 'delegating' | 'error' | 'offline'

/** Fixed palette; 'S' (shirt) is resolved separately per worker. */
export const palette: Record<string, string> = {
  h: '#3b2a1e', // hair
  f: '#e8b382', // skin
  e: '#241a12', // eyes / outline
  a: '#f0a838', // amber accessory (phone, pointing arm)
  r: '#e2483a', // red accessory (error)
  p: '#7d766c', // chair / desk grey
}

const SHIRT_PALETTE = ['#3f7cc9', '#4fae6b', '#c9578a', '#d98b32', '#8a63c9', '#2fb0a5']

/** Deterministic shirt colour for a profile name (stable across reloads). */
export function shirtColor(profile: string): string {
  let hash = 0
  for (let i = 0; i < profile.length; i++) hash = (hash * 31 + profile.charCodeAt(i)) >>> 0
  return SHIRT_PALETTE[hash % SHIRT_PALETTE.length]!
}

/** Overlays `over` onto `under`; '.' in `over` leaves the base pixel showing. */
export function mergeGrid(under: readonly string[], over: readonly string[]): string[] {
  return under.map((row, r) => {
    const overlay = over[r] ?? ''
    let out = ''
    for (let c = 0; c < row.length; c++) {
      const oc = overlay[c]
      out += oc && oc !== '.' ? oc : row[c]
    }
    return out
  })
}

function blank(): string[] {
  return Array.from({ length: spriteRows }, () => '.'.repeat(spriteCols))
}

function withPixels(px: Array<[row: number, col: number, ch: string]>): string[] {
  const g = blank().map((r) => r.split(''))
  for (const [r, c, ch] of px) g[r]![c] = ch
  return g.map((r) => r.join(''))
}

// Seated figure, facing forward: 3 rows of headroom for accessories, then
// hair/face/neck/torso. Arms are never in BASE — every pose supplies them.
/** Rows 0–2 of every figure: headroom for accessories ("!", a phone). */
export const HEADROOM: readonly string[] = ['.'.repeat(spriteCols), '.'.repeat(spriteCols), '.'.repeat(spriteCols)]

/** Rows 3–8: the front-facing head, shared with the office view's standing figures. */
export const HEAD: readonly string[] = [
  '....hhhhhh....', // 3  hair top
  '...hffffffh...', // 4  hair sides
  '...hfeffefh...', // 5  eyes
  '...hffffffh...', // 6  cheeks
  '....ffffff....', // 7  jaw
  '.....ffff.....', // 8  neck
]

const BASE: string[] = [
  ...HEADROOM,
  ...HEAD,
  '..SSSSSSSSSS..', // 9  shoulders
  '..SSSSSSSSSS..', // 10 torso
  '..SSSSSSSSSS..', // 11 torso
  '..SSSSSSSSSS..', // 12 torso
  '..SSSSSSSSSS..', // 13 torso
  '..SSSSSSSSSS..', // 14 torso
  '...SSSSSSSS...', // 15 waist
  '...SSSSSSSS...', // 16 waist
]

// Arms rest on the desk edge (row 15/16, columns just outside the torso).
const ARMS_IDLE_A = withPixels([
  [15, 1, 'f'],
  [15, 12, 'f'],
])
const ARMS_IDLE_B = withPixels([
  [14, 1, 'f'],
  [14, 12, 'f'],
])

// Hands alternate on the keyboard, closer to centre than idle.
const ARMS_WORKING_A = withPixels([
  [15, 4, 'f'],
  [14, 9, 'f'],
])
const ARMS_WORKING_B = withPixels([
  [14, 4, 'f'],
  [15, 9, 'f'],
])

// One arm up to a phone at the ear, the other resting; the phone bobs.
const ARMS_DELEGATING_A = withPixels([
  [15, 1, 'f'],
  [12, 12, 'f'],
  [9, 12, 'f'],
  [5, 12, 'a'],
])
const ARMS_DELEGATING_B = withPixels([
  [15, 1, 'f'],
  [11, 12, 'f'],
  [8, 12, 'f'],
  [4, 12, 'a'],
])

// Arms down; a small exclamation mark in the headroom above.
const ARMS_ERROR = withPixels([
  [15, 1, 'f'],
  [15, 12, 'f'],
  [0, 6, 'r'],
  [0, 7, 'r'],
  [2, 6, 'r'],
  [2, 7, 'r'],
])

// No BASE for offline: the desk sits empty. Grey chair silhouette only.
const CHAIR = withPixels([
  [9, 5, 'p'], [9, 6, 'p'], [9, 7, 'p'], [9, 8, 'p'],
  [10, 5, 'p'], [10, 6, 'p'], [10, 7, 'p'], [10, 8, 'p'],
  [11, 5, 'p'], [11, 6, 'p'], [11, 7, 'p'], [11, 8, 'p'],
  [12, 3, 'p'], [12, 4, 'p'], [12, 5, 'p'], [12, 6, 'p'], [12, 7, 'p'], [12, 8, 'p'], [12, 9, 'p'], [12, 10, 'p'],
  [13, 4, 'p'], [13, 9, 'p'],
  [14, 4, 'p'], [14, 9, 'p'],
])

const POSE_ARMS: Record<Exclude<Pose, 'offline'>, string[][]> = {
  idle: [ARMS_IDLE_A, ARMS_IDLE_B],
  working: [ARMS_WORKING_A, ARMS_WORKING_B],
  delegating: [ARMS_DELEGATING_A, ARMS_DELEGATING_B],
  error: [ARMS_ERROR],
}

/** Merged, ready-to-draw frames for a pose. 'offline' has no figure at all. */
export function framesFor(pose: Pose): string[][] {
  if (pose === 'offline') return [CHAIR]
  return POSE_ARMS[pose].map((arms) => mergeGrid(BASE, arms))
}
