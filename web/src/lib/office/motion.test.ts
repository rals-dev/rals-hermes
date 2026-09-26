import { describe, expect, it } from 'vitest'
import { WALK_SPEED, moveTo, placeAt, spotFor, stepActor, type Actor } from './motion'
import { TILE, officeMap, seatOf } from './map'

const m = officeMap

function standingAt(x: number, y: number): Actor {
  return placeAt({ tile: { x: x / TILE, y: y / TILE }, px: { x, y }, facing: 'down', pose: 'standing' }, true)
}

describe('spotFor', () => {
  it('seats a desk occupant centred across the two desk tiles, facing the viewer', () => {
    const desk = m.desks[2]!
    const s = spotFor(m, { kind: 'desk', desk: 2 })
    expect(s.tile).toEqual(seatOf(desk))
    expect(s.px).toEqual({ x: desk.x * TILE + TILE / 2, y: (desk.y - 1) * TILE })
    expect(s).toMatchObject({ pose: 'seated', facing: 'down' })
  })

  it('stands a visitor beside the occupant, facing them', () => {
    expect(spotFor(m, { kind: 'visiting', desk: 0, child: 'x' })).toMatchObject({ pose: 'standing', facing: 'left' })
  })

  it('sends absent agents to the door', () => {
    expect(spotFor(m, { kind: 'absent' }).tile).toEqual(m.door)
  })
})

describe('stepActor', () => {
  it('walks along x before y at WALK_SPEED, facing the direction of travel', () => {
    const a: Actor = { ...standingAt(0, 0), waypoints: [{ x: 32, y: 32 }] }
    const half = stepActor(a, 16 / WALK_SPEED)
    expect(half).toMatchObject({ x: 16, y: 0, facing: 'right' })
    const turned = stepActor(half, 24 / WALK_SPEED)
    expect(turned).toMatchObject({ x: 32, y: 8, facing: 'down' })
  })

  it('crosses several waypoints in one long step and settles into the rest pose', () => {
    const a: Actor = {
      ...standingAt(0, 0),
      waypoints: [{ x: 16, y: 0 }, { x: 16, y: 16 }, { x: 24, y: 16 }],
      rest: { facing: 'down', pose: 'seated' },
    }
    const done = stepActor(a, 10_000)
    expect(done).toMatchObject({ x: 24, y: 16, waypoints: [], facing: 'down', pose: 'seated' })
  })

  it('hides an actor that arrives somewhere it was told to leave from', () => {
    const a: Actor = { ...standingAt(0, 0), waypoints: [{ x: 16, y: 0 }], hideOnArrival: true }
    expect(stepActor(a, 10_000).visible).toBe(false)
  })
})

describe('moveTo', () => {
  it('routes from a desk seat to the lounge over walkable tiles, ending exactly on the spot', () => {
    const seated = placeAt(spotFor(m, { kind: 'desk', desk: 0 }), true)
    const lounge = spotFor(m, { kind: 'lounge', spot: 5 })
    const walking = moveTo(m, seated, lounge, { instant: false })
    expect(walking.waypoints.at(-1)).toEqual(lounge.px)
    for (const p of walking.waypoints.slice(0, -1)) {
      expect(p.x % TILE === 0 && p.y % TILE === 0).toBe(true)
      expect(m.blocked[p.y / TILE]![p.x / TILE]).toBe(false)
    }
    const arrived = stepActor(walking, 60_000)
    expect(arrived).toMatchObject({ x: lounge.px.x, y: lounge.px.y, waypoints: [] })
  })

  it('jumps straight to the spot when instant (first render, reduced motion)', () => {
    const a = moveTo(m, standingAt(32, 32), spotFor(m, { kind: 'lounge', spot: 0 }), { instant: true })
    expect(a.waypoints).toEqual([])
    expect({ x: a.x, y: a.y }).toEqual(spotFor(m, { kind: 'lounge', spot: 0 }).px)
  })
})
