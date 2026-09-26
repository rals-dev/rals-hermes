// Who each agent looks like (docs/decisions/024). The four profiles this
// deployment runs get a designed persona — role colour, hair and an
// accessory that changes the silhouette — so they can be told apart from
// across the room, and without colour. Any other profile name gets a
// deterministic procedural persona: unique and stable, but with no role
// meaning, and never in a role colour.

import { characterPalette, fallbackHair, fallbackShirts, roleShirts, type HairTones, type ShirtTones } from './palette'

export type HairStyle = 'short' | 'messy' | 'ponytail'
export type Accessory = 'headset' | 'tie' | 'headphones' | 'hood' | 'goggles' | 'magnifier' | 'clipboard' | 'glasses'
export type Role = 'orchestrator' | 'coder' | 'tester' | 'product'

export interface Persona {
  /** Unique per profile; also the frame-cache key. */
  key: string
  role: Role | null
  hairStyle: HairStyle
  /** Stacked in this order, after the hair. */
  accessories: readonly Accessory[]
  hair: HairTones
  shirt: ShirtTones
}

const BUILT_IN: Readonly<Record<string, Omit<Persona, 'key'>>> = {
  default: {
    role: 'orchestrator', hairStyle: 'short', accessories: ['headset', 'tie'],
    hair: { color: '#3b3330', highlight: '#8a8078' }, shirt: roleShirts.amber,
  },
  'coder-agent': {
    role: 'coder', hairStyle: 'short', accessories: ['hood', 'headphones'],
    hair: { color: '#2a2320', highlight: '#4a3c33' }, shirt: roleShirts.blue,
  },
  'tester-agent': {
    role: 'tester', hairStyle: 'short', accessories: ['goggles', 'magnifier'],
    hair: { color: '#6b3f2a', highlight: '#8c5a3e' }, shirt: roleShirts.green,
  },
  'product-agent': {
    role: 'product', hairStyle: 'ponytail', accessories: ['clipboard'],
    hair: { color: '#c9a36a', highlight: '#e0c28e' }, shirt: roleShirts.terracotta,
  },
}

const FALLBACK_HAIR_STYLES: readonly HairStyle[] = ['short', 'messy', 'ponytail']
const FALLBACK_ACCESSORIES: readonly (readonly Accessory[])[] = [[], ['glasses'], ['hood']]

export function personaFor(profile: string): Persona {
  if (Object.hasOwn(BUILT_IN, profile)) return { key: profile, ...BUILT_IN[profile]! }
  // The hash is mixed so similar names don't share attributes; each
  // attribute then takes its own "digit" of the mixed value.
  let x = mix(hash(profile))
  const hairStyle = FALLBACK_HAIR_STYLES[x % 3]!
  x = Math.floor(x / 3)
  const hair = fallbackHair[x % 4]!
  x = Math.floor(x / 4)
  const accessories = FALLBACK_ACCESSORIES[x % 3]!
  x = Math.floor(x / 3)
  const shirt = fallbackShirts[x % 6]!
  return { key: profile, role: null, hairStyle, accessories, hair, shirt }
}

/** Every colour a persona's frames can use, keyed by grid character. */
export function paletteFor(p: Persona): Readonly<Record<string, string>> {
  return { ...characterPalette, h: p.hair.color, H: p.hair.highlight, c: p.shirt.color, C: p.shirt.shade }
}

function hash(s: string): number {
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0
  return h
}

function mix(x: number): number {
  x = Math.imul(x ^ (x >>> 16), 0x45d9f3b)
  x = Math.imul(x ^ (x >>> 16), 0x45d9f3b)
  return (x ^ (x >>> 16)) >>> 0
}
