import { describe, expect, it } from 'vitest'
import { framesFor, mergeGrid, shirtColor, spriteCols, spriteRows } from './sprites'

describe('shirtColor', () => {
  it('is deterministic for a given profile name', () => {
    expect(shirtColor('coder-agent')).toBe(shirtColor('coder-agent'))
  })

  it('differs for most distinct profile names (small palette, collisions allowed)', () => {
    const names = ['default', 'coder-agent', 'tester-agent', 'product-agent']
    const colors = new Set(names.map(shirtColor))
    expect(colors.size).toBeGreaterThan(1)
  })
})

describe('mergeGrid', () => {
  it('overlays non-transparent pixels onto the base, leaving the rest untouched', () => {
    const base = ['ab', 'cd']
    const overlay = ['.x', '..']
    expect(mergeGrid(base, overlay)).toEqual(['ax', 'cd'])
  })
})

describe('framesFor', () => {
  it('returns at least one frame for every worker-facing pose', () => {
    for (const pose of ['idle', 'working', 'delegating', 'error', 'offline'] as const) {
      const frames = framesFor(pose)
      expect(frames.length).toBeGreaterThanOrEqual(1)
      for (const frame of frames) {
        expect(frame).toHaveLength(spriteRows)
        for (const row of frame) expect(row).toHaveLength(spriteCols)
      }
    }
  })

  it('animates idle, working and delegating with more than one frame', () => {
    expect(framesFor('idle').length).toBeGreaterThan(1)
    expect(framesFor('working').length).toBeGreaterThan(1)
    expect(framesFor('delegating').length).toBeGreaterThan(1)
  })

  it('keeps error and offline static (single frame)', () => {
    expect(framesFor('error')).toHaveLength(1)
    expect(framesFor('offline')).toHaveLength(1)
  })

  it('working frames differ from each other (hands alternate)', () => {
    const [a, b] = framesFor('working')
    expect(a).not.toEqual(b)
  })
})
