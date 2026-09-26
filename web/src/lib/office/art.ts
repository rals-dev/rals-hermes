// Pixel art for the /floor Office view: the "night control room"
// environment (docs/decisions/024) — outlined furniture in the dashboard's
// palette, monitors as the light sources — and the mapping from an actor's
// state to the agent pose drawn for it. The agents themselves live in
// src/lib/agents/, shared with the grid view.

import type { WorkerState } from '../floor'
import { seatedPoseFor, type Pose } from '../agents/compose'
import type { FurnitureKind } from './map'
import type { Facing, Stance } from './motion'

export interface CharacterView {
  walking: boolean
  facing: Facing
  pose: Stance
  state: WorkerState
}

/** The agent pose (and whether to mirror it) for an actor this frame. */
export function officePose(v: CharacterView): { pose: Pose; mirrored: boolean } {
  if (v.walking) {
    switch (v.facing) {
      case 'down': return { pose: 'walkFront', mirrored: false }
      case 'up': return { pose: 'walkBack', mirrored: false }
      case 'right': return { pose: 'walkSide', mirrored: false }
      case 'left': return { pose: 'walkSide', mirrored: true }
    }
  }
  if (v.pose === 'seated') return { pose: seatedPoseFor(v.state), mirrored: false }
  switch (v.facing) {
    case 'right': return { pose: 'talk', mirrored: false }
    case 'left': return { pose: 'talk', mirrored: true }
    // No spot rests facing up today; show the front rather than walk in place.
    default: return { pose: 'stand', mirrored: false }
  }
}

// ── Office colours: the "night control room" (docs/decisions/024) ──────

export const officePalette: Record<string, string> = {
  outline: '#0f0d0c',
  floor: '#26221f', floorSeam: '#2e2a27', floorFleck: '#2a2623',
  carpet: '#23262b', carpetSeam: '#2a2d33',
  wallCap: '#141211', wallFace: '#1f1c1a', wallBase: '#3a3532', mat: '#3a3532',
  windowFrame: '#3a3532', windowSky: '#1a2433', cityLight: '#e8a73a', star: '#a69c90',
  deskTop: '#4a3a2f', deskHighlight: '#5c483a', deskFront: '#33281f',
  monitor: '#151312', screenOff: '#2a2624', screenOn: '#e8a73a', screenError: '#e25b4a',
  keyboard: '#6e665d',
  chair: '#3a3532', chairHighlight: '#4d4742',
  sofa: '#4b4f5c', sofaHighlight: '#5c6170',
  tableTop: '#4a3a2f', tableFront: '#33281f', cup: '#ede6dc',
  machine: '#2b2826', led: '#e25b4a',
  pot: '#5c3a21', leaf: '#3f6b43', leafHighlight: '#57865a',
  shelf: '#33281f', shelfBack: '#26201a',
  book1: '#7a3f30', book2: '#5b7fae', book3: '#b57d22', book4: '#468a4d', book5: '#6e665d',
  glass: 'rgba(127, 167, 217, 0.12)', glassFrame: '#4d4742',
  boardFrame: '#3a3532', board: '#2e2a27', boardLine: '#6e665d', boardMark: '#e8a73a',
  rug: '#5c2f24', rugInner: '#7a3f30', rugPattern: '#a0563f',
  lamp: '#f3d27a',
}

/** [x, y, w, h, colour] in pixels from the piece's top-left tile; y may go up to one tile above. */
export type Shape = readonly [number, number, number, number, string]

/** The monitor screen on a desk, relative to the desk — repainted live (on/off/error). */
export const MONITOR_SCREEN: Shape = [2, -7, 5, 5, 'screenOff']

/**
 * The desk chair, relative to the desk, drawn separately so it sorts
 * *behind* a seated agent while the desk itself sorts in front.
 */
export const DESK_CHAIR: readonly Shape[] = [[9, -13, 14, 12, 'outline'], [10, -12, 12, 10, 'chair'], [10, -12, 12, 2, 'chairHighlight']]

const BOOKS = ['book1', 'book2', 'book3', 'book4', 'book5']

// Every free-standing piece starts with an outline rectangle and is filled
// one pixel inside it, so pieces read as objects on the dark floor.
export function furnitureShapes(kind: FurnitureKind, w: number, h: number): Shape[] {
  const W = w * 16
  const H = h * 16
  switch (kind) {
    case 'desk':
      return [
        [0, 0, 32, 14, 'outline'], [1, 1, 30, 9, 'deskTop'], [1, 1, 30, 1, 'deskHighlight'], [1, 10, 30, 3, 'deskFront'],
        [1, 14, 3, 2, 'outline'], [28, 14, 3, 2, 'outline'],
        [3, 1, 3, 3, 'outline'], [0, -9, 9, 10, 'outline'], [1, -8, 7, 8, 'monitor'], MONITOR_SCREEN,
        [12, 3, 9, 2, 'keyboard'],
      ]
    case 'sofa':
      return [
        [0, -7, W, 21, 'outline'], [1, -6, W - 2, 8, 'sofa'], [2, 2, W - 4, 10, 'sofaHighlight'], [1, 12, W - 2, 1, 'sofa'],
        ...Array.from({ length: w - 1 }, (_, i): Shape => [(i + 1) * 16, 3, 1, 8, 'sofa']),
      ]
    case 'coffeeTable':
      return [[1, 2, 30, 12, 'outline'], [2, 3, 28, 8, 'tableTop'], [2, 11, 28, 2, 'tableFront'], [8, 5, 3, 3, 'cup'], [W - 12, 4, 6, 4, 'book2']]
    case 'coffeeMachine':
      return [[2, -7, 12, 22, 'outline'], [3, -6, 10, 20, 'machine'], [5, -3, 2, 2, 'led'], [6, 8, 4, 4, 'cup']]
    case 'lamp':
      return [[6, -9, 4, 24, 'outline'], [7, -8, 2, 22, 'machine'], [3, -13, 10, 7, 'outline'], [4, -12, 8, 5, 'lamp'], [4, 13, 8, 3, 'outline']]
    case 'plant':
      return [
        [2, -9, 12, 11, 'outline'], [3, -8, 10, 9, 'leaf'], [6, -10, 4, 3, 'leafHighlight'], [1, -4, 4, 5, 'leafHighlight'],
        [3, 2, 10, 11, 'outline'], [4, 3, 8, 9, 'pot'],
      ]
    case 'bookshelf': {
      const shapes: Shape[] = [[0, -13, W, 28, 'outline'], [1, -12, W - 2, 26, 'shelf'], [2, -10, W - 4, 7, 'shelfBack'], [2, -1, W - 4, 7, 'shelfBack']]
      for (const top of [-10, -1]) {
        for (let x = 3, i = top === -10 ? 0 : 2; x + 3 <= W - 3; x += 4, i++) {
          const short = i % 3 === 1
          shapes.push([x, top + (short ? 1 : 0), 3, short ? 6 : 7, BOOKS[i % BOOKS.length]!])
        }
      }
      return shapes
    }
    case 'meetingTable':
      return [
        [0, 3, W, H - 5, 'outline'], [1, 4, W - 2, H - 11, 'tableTop'], [1, H - 7, W - 2, 3, 'tableFront'],
        [2, H - 3, 3, 3, 'outline'], [W - 5, H - 3, 3, 3, 'outline'],
        [10, 10, 8, 6, 'cup'], [W - 24, 12, 7, 5, 'cup'], [W / 2 - 2, 14, 3, 3, 'cup'],
      ]
    case 'chair':
      return [[2, 1, 12, 13, 'outline'], [3, 2, 10, 3, 'chairHighlight'], [3, 5, 10, 7, 'chair'], [3, 14, 2, 2, 'outline'], [11, 14, 2, 2, 'outline']]
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
        [6, 4, 14, 1, 'boardLine'], [6, 7, 20, 1, 'boardLine'], [W - 20, 5, 10, 1, 'boardMark'],
        [8, 13, W - 16, 1, 'boardFrame'],
      ]
    case 'window':
      return [
        [2, 1, W - 4, 12, 'windowFrame'], [3, 2, W / 2 - 4, 10, 'windowSky'], [W / 2 + 1, 2, W / 2 - 4, 10, 'windowSky'],
        [5, 9, 1, 1, 'cityLight'], [8, 10, 1, 1, 'cityLight'], [W / 2 + 4, 8, 1, 1, 'cityLight'], [W / 2 + 9, 10, 1, 1, 'cityLight'],
        [6, 4, 1, 1, 'star'], [W / 2 + 6, 3, 1, 1, 'star'], [11, 5, 1, 1, 'star'],
      ]
    case 'rug':
      return [
        [0, 0, W, H, 'rug'], [4, 4, W - 8, H - 8, 'rugInner'],
        [8, 8, W - 16, 2, 'rugPattern'], [8, H - 10, W - 16, 2, 'rugPattern'],
        [8, 8, 2, H - 16, 'rugPattern'], [W - 10, 8, 2, H - 16, 'rugPattern'],
      ]
  }
}

