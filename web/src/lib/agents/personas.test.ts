import { describe, expect, it } from 'vitest'
import { paletteFor, personaFor } from './personas'
import { fallbackHair, fallbackShirts, roleShirts } from './palette'

describe('personaFor', () => {
  it('gives the four known profiles their designed personas', () => {
    expect(personaFor('default')).toMatchObject({
      key: 'default', role: 'orchestrator', hairStyle: 'short', accessories: ['headset', 'tie'], shirt: roleShirts.amber,
    })
    expect(personaFor('coder-agent')).toMatchObject({ role: 'coder', accessories: ['hood', 'headphones'], shirt: roleShirts.blue })
    expect(personaFor('tester-agent')).toMatchObject({ role: 'tester', accessories: ['goggles', 'magnifier'], shirt: roleShirts.green })
    expect(personaFor('product-agent')).toMatchObject({
      role: 'product', hairStyle: 'ponytail', accessories: ['clipboard'], shirt: roleShirts.terracotta,
    })
  })

  it('is deterministic for unknown profiles', () => {
    expect(personaFor('legacy-agent')).toEqual(personaFor('legacy-agent'))
  })

  it('pins one procedural persona, so changing the hash is a deliberate decision', () => {
    expect(personaFor('legacy-agent')).toEqual({
      key: 'legacy-agent', role: null, hairStyle: 'short', accessories: [], hair: fallbackHair[0], shirt: fallbackShirts[4],
    })
  })

  it('tells a sample of unknown profiles apart', () => {
    const names = ['legacy-agent', 'ops-agent', 'qa-agent', 'research-agent', 'writer-agent', 'infra-agent', 'support-agent', 'data-agent']
    const signatures = names.map((n) => {
      const p = personaFor(n)
      return JSON.stringify([p.hairStyle, p.hair, p.accessories, p.shirt])
    })
    expect(new Set(signatures).size).toBe(names.length)
  })

  it('never dresses a procedural persona in a role colour', () => {
    const roleColours = Object.values(roleShirts).map((t) => t.color)
    for (let i = 0; i < 500; i++) {
      const p = personaFor(`profile-${i}`)
      expect(p.role).toBeNull()
      expect(roleColours).not.toContain(p.shirt.color)
      expect(fallbackShirts).toContainEqual(p.shirt)
    }
  })

  it('does not mistake Object.prototype names for built-ins', () => {
    expect(personaFor('constructor').role).toBeNull()
    expect(personaFor('toString').role).toBeNull()
  })
})

describe('paletteFor', () => {
  it('adds the persona hair and shirt to the shared colours', () => {
    expect(paletteFor(personaFor('coder-agent'))).toMatchObject({
      o: '#0f0d0c', s: '#d9b99b', h: '#2a2320', H: '#4a3c33', c: '#7fa7d9', C: '#5b7fae',
    })
  })
})
