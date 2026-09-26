// Composes an agent's frame from its layers (docs/decisions/024) and caches
// the painted result, so the grid and office views draw an agent with one
// drawImage. composeFrame() is pure and tested; agentFrame() and
// helperFrame() are the browser-side cache on top of it.

import type { WorkerState } from '../floor'
import { ACCESSORIES } from './layers/accessories'
import { BODY, SEATED_GROUND, SPRITE_H, SPRITE_W, VIEW_OF, type AgentPose } from './layers/body'
import { CHAIR, HELPER } from './layers/extras'
import { HAIR } from './layers/hair'
import type { Layer } from './layers/layer'
import { paletteFor, type Persona } from './personas'

export { SPRITE_H, SPRITE_W }

/** Every pose a view can ask for; 'offline' is the empty chair. */
export type Pose = AgentPose | 'offline'

export function frameCount(pose: Pose): number {
  return pose === 'offline' ? 1 : BODY[pose].length
}

/** Rows drawn from the top: seated figures stop at the waist, the rest stand on the last row. */
export function groundRow(pose: Pose): number {
  return pose.startsWith('seated') ? SEATED_GROUND : SPRITE_H
}

export function composeFrame(persona: Persona, pose: Pose, frame: number): string[] {
  if (pose === 'offline') return [...CHAIR]
  const frames = BODY[pose]
  const body = frames[frame % frames.length]!
  const view = VIEW_OF[pose]
  const out = body.rows.map((row) => [...row])
  const layers = [HAIR[persona.hairStyle][view], ...persona.accessories.map((a) => ACCESSORIES[a][view])]
  for (const layer of layers) stamp(out, layer, body.headDy)
  return out.map((row) => row.join(''))
}

function stamp(out: string[][], layer: Layer, dy: number): void {
  layer.rows.forEach((row, i) => {
    const y = layer.top + i + dy
    if (y < 0 || y >= SPRITE_H) return
    for (let x = 0; x < row.length; x++) {
      if (row[x] !== '.') out[y]![x] = row[x]!
    }
  })
}

export function mirror(grid: readonly string[]): string[] {
  return grid.map((row) => [...row].reverse().join(''))
}

// Which frame shows when: walks step at 8 fps, everything else ticks at
// 5 fps; repeats slow a cycle down (a blink every ~1.6 s, a gesture ~0.8 s).
const SEQUENCES: Readonly<Record<Pose, readonly number[]>> = {
  seatedIdle: [0, 1],
  seatedTyping: [0, 1],
  seatedPhone: [0, 1],
  seatedError: [0],
  stand: [0, 0, 0, 0, 0, 0, 0, 1],
  talk: [0, 0, 1, 1],
  walkFront: [0, 1, 2, 3],
  walkBack: [0, 1, 2, 3],
  walkSide: [0, 1, 2, 3],
  offline: [0],
}

export function frameAt(pose: Pose, nowMs: number): number {
  const seq = SEQUENCES[pose]
  const period = pose.startsWith('walk') ? 125 : 200
  return seq[Math.floor(nowMs / period) % seq.length]!
}

/** The desk pose for a worker state, shared by both views. */
export function seatedPoseFor(state: WorkerState): Pose {
  switch (state) {
    case 'working': return 'seatedTyping'
    case 'error': return 'seatedError'
    case 'delegating': return 'seatedPhone'
    case 'offline': return 'offline'
    default: return 'seatedIdle'
  }
}

// ── Browser-side cache: one small canvas per persona × pose × frame × mirror ──
const cache = new Map<string, HTMLCanvasElement>()

export function agentFrame(persona: Persona, pose: Pose, frame: number, mirrored: boolean): HTMLCanvasElement {
  const key = `${persona.key}|${pose}|${frame}|${mirrored ? 1 : 0}`
  let canvas = cache.get(key)
  if (!canvas) {
    const grid = composeFrame(persona, pose, frame)
    canvas = paint(mirrored ? mirror(grid) : grid, paletteFor(persona))
    cache.set(key, canvas)
  }
  return canvas
}

export function helperFrame(persona: Persona): HTMLCanvasElement {
  const key = `${persona.key}|helper`
  let canvas = cache.get(key)
  if (!canvas) {
    canvas = paint(HELPER, paletteFor(persona))
    cache.set(key, canvas)
  }
  return canvas
}

function paint(grid: readonly string[], palette: Readonly<Record<string, string>>): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = grid[0]!.length
  canvas.height = grid.length
  const ctx = canvas.getContext('2d')
  if (!ctx) return canvas
  grid.forEach((row, y) => {
    for (let x = 0; x < row.length; x++) {
      const ch = row[x]!
      if (ch === '.') continue
      ctx.fillStyle = palette[ch] ?? '#ff00ff' // magenta: an unmapped key is a bug, and visible
      ctx.fillRect(x, y, 1, 1)
    }
  })
  return canvas
}
