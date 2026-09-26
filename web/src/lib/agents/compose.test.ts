import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { agentFrame, composeFrame, frameAt, frameCount, groundRow, helperFrame, mirror, seatedPoseFor, type Pose } from './compose'
import { paletteFor, personaFor } from './personas'
import { BODY } from './layers/body'

const PERSONAS = ['default', 'coder-agent', 'tester-agent', 'product-agent', 'legacy-agent', 'writer-agent'].map(personaFor)
const POSES: Pose[] = [...(Object.keys(BODY) as Pose[]), 'offline']

describe('composeFrame', () => {
  it('composes every persona × pose × frame to 16×24 in colours the persona resolves', () => {
    for (const persona of PERSONAS) {
      const palette = paletteFor(persona)
      for (const pose of POSES) {
        for (let i = 0; i < frameCount(pose); i++) {
          const grid = composeFrame(persona, pose, i)
          const where = `${persona.key} ${pose}[${i}]`
          expect(grid, where).toHaveLength(24)
          for (const row of grid) {
            expect(row, where).toHaveLength(16)
            for (const ch of row) expect(ch === '.' || ch in palette, `${where}: '${ch}'`).toBe(true)
          }
        }
      }
    }
  })

  it('stacks hair and accessories over the body', () => {
    const coder = composeFrame(personaFor('coder-agent'), 'stand', 0)
    expect(coder[0]).toBe('....KKKKKKKK....') // headphone band over the head outline
    expect(coder[4]).toBe('ockhhsssssshhkco') // cups over both ears, hair at the temples
  })

  it('shows accessories from the front and hides held items from behind', () => {
    const tester = personaFor('tester-agent')
    expect(composeFrame(tester, 'stand', 0).join('')).toContain('g')
    expect(composeFrame(tester, 'walkBack', 1).join('')).not.toContain('g')
    const product = personaFor('product-agent')
    expect(composeFrame(product, 'stand', 0).join('')).toContain('w')
    expect(composeFrame(product, 'walkBack', 1).join('')).not.toContain('w')
  })

  it('moves hair and accessories down with the head on stride frames', () => {
    const p = personaFor('default')
    const stand = composeFrame(p, 'stand', 0)
    const stride = composeFrame(p, 'walkFront', 0)
    expect(stride[0]).toBe('.'.repeat(16))
    expect(stride.slice(1, 11)).toEqual(stand.slice(0, 10))
  })

  it('draws the same empty chair for any offline persona', () => {
    expect(composeFrame(personaFor('default'), 'offline', 0)).toEqual(composeFrame(personaFor('qa-agent'), 'offline', 0))
  })
})

describe('mirror', () => {
  it('flips each row and undoes itself', () => {
    expect(mirror(['ab.', '.cd'])).toEqual(['.ba', 'dc.'])
    const g = composeFrame(personaFor('default'), 'talk', 0)
    expect(mirror(mirror(g))).toEqual(g)
  })
})

describe('frame timing', () => {
  it('steps walks at 8 fps through all four frames', () => {
    expect([0, 125, 250, 375, 500].map((t) => frameAt('walkSide', t))).toEqual([0, 1, 2, 3, 0])
  })

  it('blinks once per 1.6 s while standing', () => {
    expect([0, 1200, 1400, 1600].map((t) => frameAt('stand', t))).toEqual([0, 0, 1, 0])
  })

  it('gestures every 0.8 s while talking', () => {
    expect([0, 200, 400, 600, 800].map((t) => frameAt('talk', t))).toEqual([0, 0, 1, 1, 0])
  })

  it('holds single-frame poses', () => {
    expect(frameAt('seatedError', 12345)).toBe(0)
    expect(frameAt('offline', 999)).toBe(0)
  })
})

describe('groundRow', () => {
  it('stops seated figures at the waist and stands everyone else on the last row', () => {
    expect(groundRow('seatedTyping')).toBe(19)
    expect(groundRow('stand')).toBe(24)
    expect(groundRow('walkBack')).toBe(24)
    expect(groundRow('offline')).toBe(24)
  })
})

describe('seatedPoseFor', () => {
  it('maps each worker state to its desk pose', () => {
    expect(seatedPoseFor('working')).toBe('seatedTyping')
    expect(seatedPoseFor('idle')).toBe('seatedIdle')
    expect(seatedPoseFor('error')).toBe('seatedError')
    expect(seatedPoseFor('delegating')).toBe('seatedPhone')
    expect(seatedPoseFor('offline')).toBe('offline')
  })
})

describe('frame cache', () => {
  // jsdom has no 2D context; the cache must still hand out stable canvases.
  beforeEach(() => {
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('returns the same canvas for the same frame and another one when mirrored', () => {
    const p = personaFor('coder-agent')
    const a = agentFrame(p, 'talk', 1, false)
    expect(agentFrame(p, 'talk', 1, false)).toBe(a)
    expect(agentFrame(p, 'talk', 1, true)).not.toBe(a)
    expect([a.width, a.height]).toEqual([16, 24])
  })

  it('caches helper figures per persona at 8×12', () => {
    const h = helperFrame(personaFor('default'))
    expect(helperFrame(personaFor('default'))).toBe(h)
    expect(helperFrame(personaFor('coder-agent'))).not.toBe(h)
    expect([h.width, h.height]).toEqual([8, 12])
  })
})
