import { describe, expect, it } from 'vitest'
import { DESK_CHAIR, HELPER, characterFrames, furnitureShapes, mirror, officePalette, spritePalette } from './art'
import { spriteCols, spriteRows } from '../sprites'
import { TILE, officeMap } from './map'

describe('character frames', () => {
  const cases = [
    { walking: true, facing: 'down' },
    { walking: true, facing: 'up' },
    { walking: true, facing: 'left' },
    { walking: true, facing: 'right' },
    { walking: false, facing: 'down', pose: 'standing' },
    { walking: false, facing: 'left', pose: 'standing' },
    { walking: false, facing: 'down', pose: 'seated', state: 'working' },
    { walking: false, facing: 'down', pose: 'seated', state: 'idle' },
    { walking: false, facing: 'down', pose: 'seated', state: 'error' },
    { walking: false, facing: 'down', pose: 'seated', state: 'delegating' },
  ] as const

  it('are all the same size as the grid-view sprites, using only known palette keys', () => {
    for (const c of cases) {
      const frames = characterFrames({ pose: 'standing', state: 'idle', ...c })
      expect(frames.length, JSON.stringify(c)).toBeGreaterThan(0)
      for (const f of frames) {
        expect(f).toHaveLength(spriteRows)
        for (const row of f) {
          expect(row).toHaveLength(spriteCols)
          for (const ch of row) expect(ch === '.' || ch === 'S' || ch in spritePalette, `${ch} in ${JSON.stringify(c)}`).toBe(true)
        }
      }
    }
  })

  it('animate walking with distinct frames in every direction', () => {
    for (const facing of ['down', 'up', 'left', 'right'] as const) {
      const [a, b] = characterFrames({ walking: true, facing, pose: 'standing', state: 'idle' })
      expect(a, facing).not.toEqual(b)
    }
  })

  it('face left as the exact mirror image of facing right', () => {
    const right = characterFrames({ walking: true, facing: 'right', pose: 'standing', state: 'idle' })
    const left = characterFrames({ walking: true, facing: 'left', pose: 'standing', state: 'idle' })
    expect(left).toEqual(right.map(mirror))
  })

  it('show the back of the head when walking away', () => {
    const [up] = characterFrames({ walking: true, facing: 'up', pose: 'standing', state: 'idle' })
    expect(up!.slice(3, 9).join('')).not.toContain('e') // head rows: no eyes from behind
  })
})

describe('HELPER', () => {
  it('is a small, rectangular sprite', () => {
    const width = HELPER[0]!.length
    expect(HELPER.length).toBeLessThan(spriteRows)
    for (const row of HELPER) expect(row).toHaveLength(width)
  })
})

describe('furnitureShapes', () => {
  it('stay within their footprint (and at most one tile above it), using known colours', () => {
    for (const f of officeMap.furniture) {
      for (const [x, y, w, h, color] of furnitureShapes(f.kind, f.w, f.h)) {
        const where = `${f.kind} ${f.w}x${f.h}: [${x},${y},${w},${h}]`
        expect(x, where).toBeGreaterThanOrEqual(0)
        expect(x + w, where).toBeLessThanOrEqual(f.w * TILE)
        expect(y, where).toBeGreaterThanOrEqual(-TILE)
        expect(y + h, where).toBeLessThanOrEqual(f.h * TILE)
        expect(color in officePalette, where).toBe(true)
      }
    }
  })

  it('keep the desk chair on the seat row, above the desk', () => {
    for (const [x, y, w, h, color] of DESK_CHAIR) {
      expect(x).toBeGreaterThanOrEqual(0)
      expect(x + w).toBeLessThanOrEqual(2 * TILE)
      expect(y).toBeGreaterThanOrEqual(-TILE)
      expect(y + h).toBeLessThanOrEqual(0)
      expect(color in officePalette).toBe(true)
    }
  })
})

describe('night control room', () => {
  it('uses the ADR-024 environment palette', () => {
    expect(officePalette).toMatchObject({
      floor: '#26221f', wallFace: '#1f1c1a', windowSky: '#1a2433', deskTop: '#4a3a2f',
      screenOn: '#e8a73a', screenError: '#e25b4a', screenOff: '#2a2624', rugInner: '#7a3f30', lamp: '#f3d27a',
    })
  })

  it('outlines every free-standing piece of furniture', () => {
    const flat = new Set(['window', 'whiteboard', 'rug', 'glass'])
    for (const f of officeMap.furniture) {
      if (flat.has(f.kind)) continue
      const colours = furnitureShapes(f.kind, f.w, f.h).map((s) => s[4])
      expect(colours, f.kind).toContain('outline')
    }
  })
})

