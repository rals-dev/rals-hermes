import { describe, expect, it } from 'vitest'
import { LOUNGE_AFTER_IDLE_MS, deriveOffice, placementKey } from './placement'
import { WORKING_WINDOW_MS, type DelegationLink, type Worker, type WorkerState } from '../floor'

const NOW = Date.parse('2026-09-26T10:00:00Z')

function w(profile: string, state: WorkerState, over: Partial<Worker> = {}): Worker {
  return {
    profile, status: state === 'offline' ? 'unreachable' : 'healthy', state,
    delegatedBadge: false, openSessions: 0, toolCallsRecent: 0, lastActiveAt: NOW, ...over,
  }
}

function link(parent: string, child: string, startedAgoMs: number): DelegationLink {
  return {
    parentProfile: parent, childProfile: child,
    parentSessionId: `${parent}-s`, childSessionId: `${child}-${startedAgoMs}`,
    childStartedAt: NOW - startedAgoMs,
  }
}

function placementOf(scene: ReturnType<typeof deriveOffice>, profile: string) {
  return scene.occupants.find((o) => o.worker.profile === profile)!.placement
}

describe('deriveOffice', () => {
  it('assigns desks in configuration order and reports profiles beyond eight as overflow', () => {
    const workers = Array.from({ length: 10 }, (_, i) => w(`p${i}`, 'working'))
    const scene = deriveOffice(workers, [], NOW)
    expect(scene.occupants.map((o) => [o.worker.profile, o.desk])).toEqual(
      Array.from({ length: 8 }, (_, i) => [`p${i}`, i]),
    )
    expect(scene.overflow).toEqual(['p8', 'p9'])
  })

  it('keeps working and erroring agents at their own desk, and offline agents out of the office', () => {
    const scene = deriveOffice([w('a', 'working'), w('b', 'error'), w('c', 'offline')], [], NOW)
    expect(placementOf(scene, 'a')).toEqual({ kind: 'desk', desk: 0 })
    expect(placementOf(scene, 'b')).toEqual({ kind: 'desk', desk: 1 })
    expect(placementOf(scene, 'c')).toEqual({ kind: 'absent' })
  })

  it('sends an idle agent to the lounge only after two minutes of idleness', () => {
    // Idle begins WORKING_WINDOW_MS after the last activity.
    const justIdle = w('a', 'idle', { lastActiveAt: NOW - WORKING_WINDOW_MS - LOUNGE_AFTER_IDLE_MS + 1_000 })
    const longIdle = w('b', 'idle', { lastActiveAt: NOW - WORKING_WINDOW_MS - LOUNGE_AFTER_IDLE_MS })
    const neverSeen = w('c', 'idle', { lastActiveAt: null })
    const scene = deriveOffice([justIdle, longIdle, neverSeen], [], NOW)
    expect(placementOf(scene, 'a')).toEqual({ kind: 'desk', desk: 0 })
    expect(placementOf(scene, 'b')).toEqual({ kind: 'lounge', spot: 1 })
    expect(placementOf(scene, 'c')).toEqual({ kind: 'lounge', spot: 2 })
  })

  it("sends a delegating agent to its most recent child's desk", () => {
    const scene = deriveOffice(
      [w('default', 'delegating'), w('coder', 'working'), w('tester', 'working')],
      [link('default', 'coder', 60_000), link('default', 'tester', 5_000)],
      NOW,
    )
    expect(placementOf(scene, 'default')).toEqual({ kind: 'visiting', desk: 2, child: 'tester' })
    expect(placementOf(scene, 'tester')).toEqual({ kind: 'desk', desk: 2 })
  })

  it('keeps a delegating agent at its own desk when the child has no desk', () => {
    const workers = [w('default', 'delegating'), ...Array.from({ length: 8 }, (_, i) => w(`p${i}`, 'working'))]
    const scene = deriveOffice(workers, [link('default', 'p7', 1_000)], NOW) // p7 is the 9th profile
    expect(placementOf(scene, 'default')).toEqual({ kind: 'desk', desk: 0 })
  })

  it('lists the cross-profile parents delegating to each child', () => {
    const scene = deriveOffice(
      [w('default', 'delegating'), w('product', 'delegating'), w('coder', 'working')],
      [link('default', 'coder', 10_000), link('product', 'coder', 20_000)],
      NOW,
    )
    expect(scene.occupants.find((o) => o.worker.profile === 'coder')!.delegatedFrom).toEqual(['default', 'product'])
  })
})

describe('placementKey', () => {
  it('distinguishes every destination', () => {
    const keys = [
      placementKey({ kind: 'absent' }),
      placementKey({ kind: 'desk', desk: 1 }),
      placementKey({ kind: 'visiting', desk: 1, child: 'x' }),
      placementKey({ kind: 'lounge', spot: 1 }),
    ]
    expect(new Set(keys).size).toBe(keys.length)
  })
})
