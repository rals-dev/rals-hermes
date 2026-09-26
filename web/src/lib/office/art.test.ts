import { describe, expect, it } from 'vitest'
import { DESK_CHAIR, furnitureShapes, officePalette, officePose } from './art'
import { TILE, officeMap } from './map'

describe('officePose', () => {
  const base = { walking: false, facing: 'down', pose: 'standing', state: 'idle' } as const

  it('walks in the direction of travel, mirroring the side view for left', () => {
    expect(officePose({ ...base, walking: true, facing: 'down' })).toEqual({ pose: 'walkFront', mirrored: false })
    expect(officePose({ ...base, walking: true, facing: 'up' })).toEqual({ pose: 'walkBack', mirrored: false })
    expect(officePose({ ...base, walking: true, facing: 'right' })).toEqual({ pose: 'walkSide', mirrored: false })
    expect(officePose({ ...base, walking: true, facing: 'left' })).toEqual({ pose: 'walkSide', mirrored: true })
  })

  it('sits in the desk pose for the worker state', () => {
    expect(officePose({ ...base, pose: 'seated', state: 'working' }).pose).toBe('seatedTyping')
    expect(officePose({ ...base, pose: 'seated', state: 'error' }).pose).toBe('seatedError')
    expect(officePose({ ...base, pose: 'seated', state: 'delegating' }).pose).toBe('seatedPhone')
    expect(officePose({ ...base, pose: 'seated', state: 'idle' }).pose).toBe('seatedIdle')
  })

  it('talks while visiting (facing a colleague) and stands in the lounge', () => {
    expect(officePose({ ...base, facing: 'left' })).toEqual({ pose: 'talk', mirrored: true })
    expect(officePose({ ...base, facing: 'right' })).toEqual({ pose: 'talk', mirrored: false })
    expect(officePose({ ...base, facing: 'down' })).toEqual({ pose: 'stand', mirrored: false })
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
