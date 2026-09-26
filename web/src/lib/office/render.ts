// Canvas drawing for the /floor Office view (docs/decisions/023). Everything
// is drawn in map pixels (1 tile = 16 px); the component scales the context
// by a whole number so pixels stay square and sharp.
//
// Two passes: drawBackground() paints what never changes and never overlaps
// an agent (floor, walls, wall decor, rugs) once, into an offscreen canvas;
// drawScene() blits that, then paints furniture and agents sorted by their
// bottom edge, so a desk hides the legs of someone walking behind it and a
// seated agent covers their own chair.

import type { WorkerState } from '../floor'
import { DESK_CHAIR, HELPER, MONITOR_SCREEN, characterFrames, furnitureShapes, officePalette, spritePalette, type Shape } from './art'
import { COLS, ROWS, TILE, type Furniture, type OfficeMap } from './map'
import type { Actor } from './motion'

export const MAP_W = COLS * TILE
export const MAP_H = ROWS * TILE

const BACKGROUND_KINDS = new Set(['window', 'whiteboard', 'rug'])

/** What a desk shows this frame. */
export interface DeskView {
  desk: number
  screen: 'off' | 'on' | 'error'
  /** Same-profile sub-agents to draw beside the desk (the view caps this at 3). */
  helpers: number
  /** Owner's shirt colour, for the helpers; null for an unassigned desk. */
  shirt: string | null
}

export interface ActorView {
  actor: Actor
  state: WorkerState
  shirt: string
}

export function drawBackground(ctx: CanvasRenderingContext2D, m: OfficeMap): void {
  const fill = (color: string, x: number, y: number, w: number, h: number) => {
    ctx.fillStyle = color
    ctx.fillRect(x, y, w, h)
  }
  const inMeetingRoom = (x: number, y: number) => x >= 22 && x <= 30 && y >= 11 && y <= 16

  for (let ty = 2; ty < m.rows - 1; ty++) {
    for (let tx = 1; tx < m.cols - 1; tx++) {
      const x = tx * TILE
      const y = ty * TILE
      if (inMeetingRoom(tx, ty)) {
        fill(officePalette.carpet!, x, y, TILE, TILE)
        fill(officePalette.carpetSeam!, x, y + TILE - 1, TILE, 1)
        continue
      }
      // Wood planks: 4 px bands with staggered end joints.
      fill(officePalette.floor!, x, y, TILE, TILE)
      for (let band = 0; band < 4; band++) {
        fill(officePalette.floorSeam!, x, y + band * 4 + 3, TILE, 1)
        const joint = ((ty * 4 + band) * 7 + tx * 3) % TILE
        fill(officePalette.floorSeam!, x + joint, y + band * 4, 1, 3)
        fill(officePalette.floorLight!, x + ((joint + 8) % TILE), y + band * 4, 1, 1)
      }
    }
  }

  // Walls: a dark cap all round, a lit face along the top.
  fill(officePalette.wallCap!, 0, 0, MAP_W, TILE)
  fill(officePalette.wallFace!, TILE, TILE, MAP_W - 2 * TILE, TILE)
  fill(officePalette.wallBase!, TILE, 2 * TILE - 3, MAP_W - 2 * TILE, 3)
  fill(officePalette.wallCap!, 0, 0, TILE, MAP_H)
  fill(officePalette.wallCap!, MAP_W - TILE, 0, TILE, MAP_H)
  fill(officePalette.wallCap!, 0, MAP_H - TILE, MAP_W, TILE)

  // The door: a gap in the bottom wall with a mat.
  const dx = m.doorway.x * TILE
  const dy = m.doorway.y * TILE
  fill(officePalette.floor!, dx, dy, TILE, TILE)
  fill(officePalette.mat!, dx + 2, dy + 1, TILE - 4, TILE - 6)

  for (const f of m.furniture) {
    if (BACKGROUND_KINDS.has(f.kind)) drawShapes(ctx, f.x * TILE, f.y * TILE, furnitureShapes(f.kind, f.w, f.h))
  }
}

export function drawScene(
  ctx: CanvasRenderingContext2D,
  m: OfficeMap,
  background: CanvasImageSource,
  desks: DeskView[],
  actors: ActorView[],
  nowMs: number,
  still: boolean,
): void {
  ctx.imageSmoothingEnabled = false
  ctx.drawImage(background, 0, 0)

  const items: { bottom: number; order: number; draw: () => void }[] = []
  const deskAt = new Map(m.desks.map((d) => [d.index, d]))

  for (const f of m.furniture) {
    if (BACKGROUND_KINDS.has(f.kind) || f.kind === 'desk') continue
    items.push({ bottom: (f.y + f.h) * TILE, order: 0, draw: () => drawFurniture(ctx, f) })
  }
  for (const view of desks) {
    const d = deskAt.get(view.desk)!
    const x = d.x * TILE
    const y = d.y * TILE
    // The chair sorts just before anyone seated on the row above the desk.
    items.push({ bottom: y - 1, order: 0, draw: () => drawShapes(ctx, x, y, DESK_CHAIR) })
    items.push({
      bottom: y + TILE, order: 0,
      draw: () => {
        drawShapes(ctx, x, y, furnitureShapes('desk', 2, 1))
        const [sx, sy, sw, sh] = MONITOR_SCREEN
        ctx.fillStyle = officePalette[view.screen === 'on' ? 'screenOn' : view.screen === 'error' ? 'screenError' : 'screenOff']!
        ctx.fillRect(x + sx, y + sy, sw, sh)
      },
    })
    if (view.shirt && view.helpers > 0) {
      items.push({
        bottom: y, order: 1,
        draw: () => {
          for (let i = 0; i < Math.min(view.helpers, 3); i++) {
            drawGrid(ctx, HELPER, x - 7 * (i + 1), y - HELPER.length, view.shirt!)
          }
        },
      })
    }
  }
  for (const a of actors) {
    if (!a.actor.visible) continue
    const ax = Math.round(a.actor.x)
    const ay = Math.round(a.actor.y)
    items.push({ bottom: ay + TILE, order: 1, draw: () => drawActor(ctx, a, ax, ay, nowMs, still) })
  }

  items.sort((p, q) => p.bottom - q.bottom || p.order - q.order)
  for (const it of items) it.draw()
}

function drawActor(ctx: CanvasRenderingContext2D, a: ActorView, x: number, y: number, nowMs: number, still: boolean): void {
  const walking = a.actor.waypoints.length > 0
  const frames = characterFrames({ walking, facing: a.actor.facing, pose: a.actor.pose, state: a.state })
  const period = walking ? 125 : 200 // 8 fps stride, 5 fps idle animations
  const frame = still ? frames[0]! : frames[Math.floor(nowMs / period) % frames.length]!
  // The 14×17 sprite stands in the 16×16 box: centred, feet on the box's bottom edge.
  drawGrid(ctx, frame, x + 1, y + TILE - frame.length, a.shirt)
}

function drawFurniture(ctx: CanvasRenderingContext2D, f: Furniture): void {
  drawShapes(ctx, f.x * TILE, f.y * TILE, furnitureShapes(f.kind, f.w, f.h))
}

function drawShapes(ctx: CanvasRenderingContext2D, ox: number, oy: number, shapes: readonly Shape[]): void {
  for (const [x, y, w, h, color] of shapes) {
    ctx.fillStyle = officePalette[color]!
    ctx.fillRect(ox + x, oy + y, w, h)
  }
}

function drawGrid(ctx: CanvasRenderingContext2D, grid: readonly string[], ox: number, oy: number, shirt: string): void {
  for (let r = 0; r < grid.length; r++) {
    const row = grid[r]!
    for (let c = 0; c < row.length; c++) {
      const ch = row[c]!
      if (ch === '.') continue
      ctx.fillStyle = ch === 'S' ? shirt : spritePalette[ch] ?? '#000'
      ctx.fillRect(ox + c, oy + r, 1, 1)
    }
  }
}
