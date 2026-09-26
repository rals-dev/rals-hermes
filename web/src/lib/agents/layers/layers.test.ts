import { describe, expect, it } from 'vitest'
import { BODY, SEATED_GROUND, SPRITE_H, SPRITE_W, VIEW_OF, type AgentPose } from './body'
import { HAIR } from './hair'
import { ACCESSORIES } from './accessories'
import { CHAIR, HELPER } from './extras'
import type { Layer } from './layer'
import { characterPalette } from '../palette'

const KEYS = new Set(['.', 'h', 'H', 'c', 'C', ...Object.keys(characterPalette)])
const POSES = Object.keys(BODY) as AgentPose[]
const BLANK = '.'.repeat(SPRITE_W)

function expectRows(rows: readonly string[], width: number, where: string) {
  for (const row of rows) {
    expect(row, where).toHaveLength(width)
    for (const ch of row) expect(KEYS.has(ch), `${where}: '${ch}'`).toBe(true)
  }
}

function expectLayer(layer: Layer, where: string) {
  expect(layer.top, where).toBeGreaterThanOrEqual(0)
  expect(layer.top + layer.rows.length, where).toBeLessThanOrEqual(SPRITE_H)
  expectRows(layer.rows, SPRITE_W, where)
}

describe('body frames', () => {
  it('are all 16×24 and use known keys', () => {
    for (const pose of POSES) {
      BODY[pose].forEach((f, i) => {
        expect(f.rows, `${pose}[${i}]`).toHaveLength(SPRITE_H)
        expectRows(f.rows, SPRITE_W, `${pose}[${i}]`)
      })
    }
  })

  it('have the agreed frame counts: 23 in total', () => {
    const counts = Object.fromEntries(POSES.map((p) => [p, BODY[p].length]))
    expect(counts).toEqual({
      seatedIdle: 2, seatedTyping: 2, seatedPhone: 2, seatedError: 1,
      stand: 2, talk: 2, walkFront: 4, walkBack: 4, walkSide: 4,
    })
  })

  it('draw nothing below the waist when seated', () => {
    for (const pose of POSES.filter((p) => p.startsWith('seated'))) {
      for (const f of BODY[pose]) {
        for (const row of f.rows.slice(SEATED_GROUND)) expect(row, pose).toBe(BLANK)
      }
    }
  })

  it('stand on the bottom row otherwise', () => {
    for (const pose of POSES.filter((p) => !p.startsWith('seated'))) {
      BODY[pose].forEach((f, i) => expect(f.rows[SPRITE_H - 1], `${pose}[${i}]`).toContain('o'))
    }
  })

  it('lower the head by one pixel on stride frames only', () => {
    for (const pose of POSES) {
      const expected = pose.startsWith('walk') ? [1, 0, 1, 0] : BODY[pose].map(() => 0)
      expect(BODY[pose].map((f) => f.headDy), pose).toEqual(expected)
    }
  })

  it('match headDy: a stride frame is the standing head moved down a row', () => {
    const stride = BODY.walkFront[0]!.rows
    expect(stride[0]).toBe(BLANK)
    expect(stride[1]).toBe(BODY.stand[0]!.rows[0])
  })

  it('animate each walk with two different strides and a pass', () => {
    for (const pose of ['walkFront', 'walkBack', 'walkSide'] as const) {
      const [a, pass, b] = BODY[pose].map((f) => f.rows)
      expect(a, pose).not.toEqual(b)
      expect(a, pose).not.toEqual(pass)
    }
  })

  it('show no face from behind', () => {
    for (const f of BODY.walkBack) expect(f.rows.slice(0, 11).join(''), 'walkBack').not.toContain('e')
  })

  it('map every pose to a view', () => {
    expect(VIEW_OF).toEqual({
      seatedIdle: 'front', seatedTyping: 'front', seatedPhone: 'front', seatedError: 'front',
      stand: 'front', talk: 'side', walkFront: 'front', walkBack: 'back', walkSide: 'side',
    })
  })
})

describe('hair and accessory layers', () => {
  it('fit the sprite and use known keys', () => {
    for (const [style, views] of Object.entries(HAIR)) {
      for (const [view, layer] of Object.entries(views)) expectLayer(layer, `hair ${style}/${view}`)
    }
    for (const [name, views] of Object.entries(ACCESSORIES)) {
      for (const [view, layer] of Object.entries(views)) expectLayer(layer, `${name}/${view}`)
    }
  })

  it('give every hair style something to draw from every side', () => {
    for (const [style, views] of Object.entries(HAIR)) {
      for (const [view, layer] of Object.entries(views)) expect(layer.rows.length, `${style}/${view}`).toBeGreaterThan(0)
    }
  })

  it('hide held items from behind', () => {
    expect(ACCESSORIES.magnifier.back.rows).toHaveLength(0)
    expect(ACCESSORIES.clipboard.back.rows).toHaveLength(0)
  })
})

describe('extras', () => {
  it('draw the empty chair at full size, standing on the bottom row', () => {
    expect(CHAIR).toHaveLength(SPRITE_H)
    expectRows(CHAIR, SPRITE_W, 'chair')
    expect(CHAIR[SPRITE_H - 1]).toContain('o')
  })

  it("draw the helper at 8×12 in its owner's hair and shirt colours", () => {
    expect(HELPER).toHaveLength(12)
    expectRows(HELPER, 8, 'helper')
    expect(HELPER.join('')).toContain('h')
    expect(HELPER.join('')).toContain('c')
  })
})
