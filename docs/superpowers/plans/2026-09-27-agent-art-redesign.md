# Agent Art Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the identical 14×17 agent sprites with four outlined 16×24 personas in the instrument-panel palette, and restyle the Office environment as a "night control room", in both the Grid and Office views of `/floor`.

**Architecture:** A new frontend module `web/src/lib/agents/` holds the art as layers — body frames per pose, hair per view, accessories per view — plus personas (built-in for four profile names, procedural otherwise). `composeFrame()` stacks the layers into a 16×24 grid (pure, unit-tested); `agentFrame()` paints and caches each composed frame on an offscreen canvas so both views draw an agent with one `drawImage`. The Grid (`PixelSprite.vue`) and the Office (`render.ts`, `OfficeView.vue`) switch to that module; the Office environment gets a new palette and outlined furniture. Placement, motion and floor-plan logic are untouched.

**Tech Stack:** Vue 3 (`<script setup lang="ts">`), TypeScript 6, Vite 8, Vitest 5 (jsdom), Canvas 2D. No new dependencies.

**Spec:** [`docs/decisions/024-agent-art-redesign.md`](../../decisions/024-agent-art-redesign.md) — read it before starting; this plan argues from it.

## Global Constraints

- Everything committed is in English — code, comments, commit messages, docs (ADR-020).
- No new npm or Go dependencies.
- Every agent frame is exactly **16×24**; the helper figure is **8×12**; the offline chair is **16×24**.
- Seated poses draw rows **0–18** only (`SEATED_GROUND = 19`); standing poses stand on row **23**.
- Colours are exactly those in ADR-024; the office art uses fixed colours, never theme tokens.
- Only the right-facing side view is authored; facing left is its mirror.
- Branch: `claude/agent-art-redesign` (already created from `main`, holds the ADR commits).
- Run frontend commands from `web/`: `npx vitest run <path>`, `npx vue-tsc -b`. Full build from the repo root: `make build` (Go is always built with `CGO_ENABLED=0` on this machine).
- End every commit message with the line: `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`

## File Structure

| File | Status | Responsibility |
| --- | --- | --- |
| `web/src/lib/agents/palette.ts` | create | Shared character colours; role shirts; fallback shirt and hair sets |
| `web/src/lib/agents/personas.ts` | create | `Persona` type, four built-ins, procedural fallback, `paletteFor()` |
| `web/src/lib/agents/personas.test.ts` | create | Persona tests |
| `web/src/lib/agents/layers/layer.ts` | create | `Layer` type (sparse rows) and `EMPTY` |
| `web/src/lib/agents/layers/body.ts` | create | Body frames per pose, `VIEW_OF`, sprite constants |
| `web/src/lib/agents/layers/hair.ts` | create | Hair styles × views |
| `web/src/lib/agents/layers/accessories.ts` | create | Accessories × views |
| `web/src/lib/agents/layers/extras.ts` | create | Offline chair, helper figure |
| `web/src/lib/agents/layers/layers.test.ts` | create | Layer data tests |
| `web/src/lib/agents/compose.ts` | create | `composeFrame`, `mirror`, frame timing, `seatedPoseFor`, frame cache |
| `web/src/lib/agents/compose.test.ts` | create | Composition and cache tests |
| `web/src/components/PixelSprite.vue` | rewrite | Grid sprite drawn from `agentFrame` |
| `web/src/components/WorkerStation.vue` | modify | Pass `profile`, error badge, taller sprite box |
| `web/src/lib/office/map.ts` | modify | Add the `lamp` furniture kind and piece |
| `web/src/lib/office/art.ts` | modify | New environment palette and furniture; later: `officePose` replaces the old character frames |
| `web/src/lib/office/art.test.ts` | modify | Environment tests; `officePose` tests |
| `web/src/lib/office/render.ts` | modify | New background; agents and helpers via the frame cache |
| `web/src/components/OfficeView.vue` | modify | Personas instead of shirt colours; overlay offsets; error badge |
| `web/src/lib/sprites.ts`, `web/src/lib/sprites.test.ts` | delete | Replaced by `lib/agents/` |
| `README.md`, `docs/prd.md` | modify | Feature rows |

---

### Task 1: Palette and personas

**Files:**
- Create: `web/src/lib/agents/palette.ts`
- Create: `web/src/lib/agents/personas.ts`
- Test: `web/src/lib/agents/personas.test.ts`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `palette.ts`: `characterPalette: Readonly<Record<string, string>>`; `interface HairTones { color: string; highlight: string }`; `interface ShirtTones { color: string; shade: string }`; `roleShirts` (`amber`, `blue`, `green`, `terracotta`: `ShirtTones`); `fallbackShirts: readonly ShirtTones[]` (6); `fallbackHair: readonly HairTones[]` (4).
  - `personas.ts`: `type HairStyle = 'short' | 'messy' | 'ponytail'`; `type Accessory = 'headset' | 'tie' | 'headphones' | 'hood' | 'goggles' | 'magnifier' | 'clipboard' | 'glasses'`; `type Role = 'orchestrator' | 'coder' | 'tester' | 'product'`; `interface Persona { key: string; role: Role | null; hairStyle: HairStyle; accessories: readonly Accessory[]; hair: HairTones; shirt: ShirtTones }`; `personaFor(profile: string): Persona`; `paletteFor(p: Persona): Readonly<Record<string, string>>` (adds keys `h`, `H`, `c`, `C`).

- [ ] **Step 1: Write the failing tests**

Create `web/src/lib/agents/personas.test.ts`:

```ts
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run (from `web/`): `npx vitest run src/lib/agents/personas.test.ts`
Expected: FAIL — `Failed to resolve import "./personas"`.

- [ ] **Step 3: Create the palette**

Create `web/src/lib/agents/palette.ts`:

```ts
// Colours of the agent figures (docs/decisions/024): the dashboard's own
// instrument-panel language — black outlines, amber for "on", and role
// colours taken from the status lamps. Keys are the single characters used
// in the pixel grids under ./layers; hair (h/H) and shirt (c/C) come from
// each persona instead (see personas.ts).

export const characterPalette: Readonly<Record<string, string>> = {
  o: '#0f0d0c', // outline
  s: '#d9b99b', // skin
  S: '#b8957a', // skin shade
  e: '#1b1917', // eyes
  p: '#3a3532', // trousers
  P: '#2a2624', // trousers shade
  b: '#151312', // shoes
  k: '#6e665d', // accessory
  K: '#a69c90', // accessory highlight
  t: '#2b2826', // tie
  w: '#ede6dc', // paper, cord
  l: '#8a8078', // paper lines
  g: '#cfe3ea', // lens
  m: '#e8a73a', // mic
  q: '#3a3532', // empty chair
  Q: '#4d4742', // empty chair highlight
}

export interface HairTones {
  color: string
  highlight: string
}

export interface ShirtTones {
  color: string
  shade: string
}

/** Shirts of the four built-in personas: the status-lamp colours. */
export const roleShirts = {
  amber: { color: '#e8a73a', shade: '#b57d22' },
  blue: { color: '#7fa7d9', shade: '#5b7fae' },
  green: { color: '#63b86b', shade: '#468a4d' },
  terracotta: { color: '#c98a6a', shade: '#9c6549' },
} as const satisfies Record<string, ShirtTones>

/** Muted shirts for procedural personas; none of them is a role colour. */
export const fallbackShirts: readonly ShirtTones[] = [
  { color: '#a69c90', shade: '#7d746a' }, // stone
  { color: '#8c7fb8', shade: '#6a5f94' }, // violet
  { color: '#5f9e9a', shade: '#467874' }, // teal
  { color: '#a39a5b', shade: '#7d7542' }, // olive
  { color: '#b87a8c', shade: '#8f5a6a' }, // rose
  { color: '#7d8a99', shade: '#5d6875' }, // slate
]

export const fallbackHair: readonly HairTones[] = [
  { color: '#2a2320', highlight: '#4a3c33' },
  { color: '#4a3c33', highlight: '#6b574a' },
  { color: '#1b1917', highlight: '#3a3532' },
  { color: '#8a8078', highlight: '#b3aaa0' },
]
```

- [ ] **Step 4: Create the personas**

Create `web/src/lib/agents/personas.ts`:

```ts
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
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `npx vitest run src/lib/agents/personas.test.ts`
Expected: PASS (7 tests).

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/agents/palette.ts web/src/lib/agents/personas.ts web/src/lib/agents/personas.test.ts
git commit -m "agents: palette and personas (ADR-024)

Four built-in personas keyed by profile name, in the status-lamp
colours, and a procedural fallback for any other name (mixed hash,
never a role colour).

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Layer data — bodies, hair, accessories, extras

**Files:**
- Create: `web/src/lib/agents/layers/layer.ts`
- Create: `web/src/lib/agents/layers/body.ts`
- Create: `web/src/lib/agents/layers/hair.ts`
- Create: `web/src/lib/agents/layers/accessories.ts`
- Create: `web/src/lib/agents/layers/extras.ts`
- Test: `web/src/lib/agents/layers/layers.test.ts`

**Interfaces:**
- Consumes: `HairStyle`, `Accessory` from `../personas` (Task 1); `characterPalette` from `../palette` (Task 1, tests only).
- Produces:
  - `layer.ts`: `interface Layer { top: number; rows: readonly string[] }`; `EMPTY: Layer`.
  - `body.ts`: `SPRITE_W = 16`, `SPRITE_H = 24`, `SEATED_GROUND = 19`; `type AgentPose = 'seatedIdle' | 'seatedTyping' | 'seatedPhone' | 'seatedError' | 'stand' | 'talk' | 'walkFront' | 'walkBack' | 'walkSide'`; `type View = 'front' | 'back' | 'side'`; `interface BodyFrame { rows: readonly string[]; headDy: 0 | 1 }`; `BODY: Readonly<Record<AgentPose, readonly BodyFrame[]>>`; `VIEW_OF: Readonly<Record<AgentPose, View>>`.
  - `hair.ts`: `HAIR: Readonly<Record<HairStyle, Readonly<Record<View, Layer>>>>`.
  - `accessories.ts`: `ACCESSORIES: Readonly<Record<Accessory, Readonly<Record<View, Layer>>>>`.
  - `extras.ts`: `CHAIR: readonly string[]` (16×24); `HELPER: readonly string[]` (8×12).

Grid characters: `.` transparent; `o` outline; `s`/`S` skin; `e` eyes; `c`/`C` shirt (persona); `h`/`H` hair (persona); `p`/`P` trousers; `b` shoes; `k`/`K` accessory; `t` tie; `w` paper; `l` lines; `g` lens; `m` mic; `q`/`Q` chair.

- [ ] **Step 1: Write the failing tests**

Create `web/src/lib/agents/layers/layers.test.ts`:

```ts
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npx vitest run src/lib/agents/layers/layers.test.ts`
Expected: FAIL — `Failed to resolve import "./body"`.

- [ ] **Step 3: Create the layer type**

Create `web/src/lib/agents/layers/layer.ts`:

```ts
/**
 * A sparse overlay for a 16×24 frame: `rows` are full-width rows starting
 * at row `top`; '.' is transparent. Hair and accessories are layers so each
 * one is drawn once per view, not once per pose.
 */
export interface Layer {
  top: number
  rows: readonly string[]
}

export const EMPTY: Layer = { top: 0, rows: [] }
```

- [ ] **Step 4: Create the body frames**

Create `web/src/lib/agents/layers/body.ts` with exactly this content:

```ts
// Body frames per pose (docs/decisions/024): the figure without hair or
// accessories, 16×24, black outline, two-tone shading. The head is bald
// here — hair and accessories are layers (hair.ts, accessories.ts) stacked
// on top by compose.ts. On stride frames the whole upper body is drawn one
// row lower (headDy = 1) so the walk bobs; layers follow via headDy.

export const SPRITE_W = 16
export const SPRITE_H = 24
/** Seated poses draw rows 0–18 only (waist on row 18); the desk hides the rest. */
export const SEATED_GROUND = 19

export type AgentPose =
  | 'seatedIdle' | 'seatedTyping' | 'seatedPhone' | 'seatedError'
  | 'stand' | 'talk' | 'walkFront' | 'walkBack' | 'walkSide'

export type View = 'front' | 'back' | 'side'

export interface BodyFrame {
  rows: readonly string[]
  /** How far the head (and every layer drawn on it) sits below its standing position. */
  headDy: 0 | 1
}

const SEATED_IDLE: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoso..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_IDLE_RAISED: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..osocccccCoso..', // 15
  '..ococcccccoco..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_TYPING_L: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..occcccccscco..', // 16
  '...oospppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_TYPING_R: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..occsccccccco..', // 16
  '...oopppppsoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_PHONE: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..ossssssssskk..', // 4
  '..ossssssssskko.', // 5
  '..ossesssseSkko.', // 6
  '..ossssssssskk..', // 7
  '...osssSSsssss..', // 8
  '....oSssssSo.co.', // 9
  '.....ooSSoo..co.', // 10
  '...ooccccccooco.', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoco..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_PHONE_UP: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..ossssssssskk..', // 3
  '..ossssssssskko.', // 4
  '..ossssssssskko.', // 5
  '..ossesssseSkk..', // 6
  '..osssssssssss..', // 7
  '...osssSSsssoco.', // 8
  '....oSssssSo.co.', // 9
  '.....ooSSoo..co.', // 10
  '...ooccccccooco.', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoco..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_ERROR: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '.osssssssssssso.', // 3
  '.ocssssssssssco.', // 4
  '.ocssssssssssco.', // 5
  '.ocssesssseSsco.', // 6
  '.ocssssssssssco.', // 7
  '.ocosssSSsssoco.', // 8
  '.oc.oSssssSo.co.', // 9
  '.oc..ooSSoo..co.', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..ococcccccoco..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const STAND: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoso..', // 16
  '...ooppppppoo...', // 17
  '....oppppppo....', // 18
  '....oppooppo....', // 19
  '....oppooppo....', // 20
  '....oPPooPPo....', // 21
  '....obboobbo....', // 22
  '....ooo..ooo....', // 23
]

const STAND_BLINK: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossSssssSSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoso..', // 16
  '...ooppppppoo...', // 17
  '....oppppppo....', // 18
  '....oppooppo....', // 19
  '....oppooppo....', // 20
  '....oPPooPPo....', // 21
  '....obboobbo....', // 22
  '....ooo..ooo....', // 23
]

const TALK_LOW: readonly string[] = [
  '.....oooooo.....', // 0
  '....osssssso....', // 1
  '...osssssssso...', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssseso...', // 5
  '...osssssssso...', // 6
  '...ossssssssso..', // 7
  '....ossssssSo...', // 8
  '.....oSsssso....', // 9
  '......oSSo......', // 10
  '....occcccoo....', // 11
  '....occccCoso...', // 12
  '....occcCCCo....', // 13
  '....occccCo.....', // 14
  '....occccCo.....', // 15
  '....occccCo.....', // 16
  '....opppppo.....', // 17
  '.....opppo......', // 18
  '.....opppo......', // 19
  '.....opppo......', // 20
  '.....oPPPo......', // 21
  '.....obbbbo.....', // 22
  '.....oooooo.....', // 23
]

const TALK_HIGH: readonly string[] = [
  '.....oooooo.....', // 0
  '....osssssso....', // 1
  '...osssssssso...', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssseso...', // 5
  '...osssssssso...', // 6
  '...ossssssssso..', // 7
  '....ossssssSo...', // 8
  '.....oSsssso....', // 9
  '......oSSo.so...', // 10
  '....occcccCo....', // 11
  '....occcCCo.....', // 12
  '....occccCo.....', // 13
  '....occccCo.....', // 14
  '....occccCo.....', // 15
  '....occccCo.....', // 16
  '....opppppo.....', // 17
  '.....opppo......', // 18
  '.....opppo......', // 19
  '.....opppo......', // 20
  '.....oPPPo......', // 21
  '.....obbbbo.....', // 22
  '.....oooooo.....', // 23
]

const WALK_FRONT_STEP_L: readonly string[] = [
  '................', // 0
  '....oooooooo....', // 1
  '...osssssssso...', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..ossesssseSso..', // 7
  '..osssssssssso..', // 8
  '...osssSSssso...', // 9
  '....oSssssSo....', // 10
  '.....ooSSoo.....', // 11
  '...ooccccccoo...', // 12
  '..occcccccccco..', // 13
  '..occcccccccCo..', // 14
  '..ococcccccoCo..', // 15
  '..ococccccCoCo..', // 16
  '..osoccccccoso..', // 17
  '...ooppppppoo...', // 18
  '....oppppppo....', // 19
  '....oppooppo....', // 20
  '....oPPoobbo....', // 21
  '....obbooooo....', // 22
  '....ooo.........', // 23
]

const WALK_FRONT_STEP_R: readonly string[] = [
  '................', // 0
  '....oooooooo....', // 1
  '...osssssssso...', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..ossesssseSso..', // 7
  '..osssssssssso..', // 8
  '...osssSSssso...', // 9
  '....oSssssSo....', // 10
  '.....ooSSoo.....', // 11
  '...ooccccccoo...', // 12
  '..occcccccccco..', // 13
  '..occcccccccCo..', // 14
  '..ococcccccoCo..', // 15
  '..ococccccCoCo..', // 16
  '..osoccccccoso..', // 17
  '...ooppppppoo...', // 18
  '....oppppppo....', // 19
  '....oppooppo....', // 20
  '....obbooPPo....', // 21
  '....ooooobbo....', // 22
  '.........ooo....', // 23
]

const WALK_BACK_STEP_L: readonly string[] = [
  '................', // 0
  '....oooooooo....', // 1
  '...osssssssso...', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..osssssssssso..', // 7
  '..osssssssssso..', // 8
  '...osssssssso...', // 9
  '....oSssssSo....', // 10
  '.....ooSSoo.....', // 11
  '...ooccccccoo...', // 12
  '..occcccccccco..', // 13
  '..occcccccccCo..', // 14
  '..ococcccccoCo..', // 15
  '..ococccccCoCo..', // 16
  '..osoccccccoso..', // 17
  '...ooppppppoo...', // 18
  '....oppppppo....', // 19
  '....oppooppo....', // 20
  '....oPPoobbo....', // 21
  '....obbooooo....', // 22
  '....ooo.........', // 23
]

const STAND_BACK: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..osssssssssso..', // 7
  '...osssssssso...', // 8
  '....osssssso....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoso..', // 16
  '...ooppppppoo...', // 17
  '....oppppppo....', // 18
  '....oppooppo....', // 19
  '....oppooppo....', // 20
  '....oPPooPPo....', // 21
  '....obboobbo....', // 22
  '....ooo..ooo....', // 23
]

const WALK_BACK_STEP_R: readonly string[] = [
  '................', // 0
  '....oooooooo....', // 1
  '...osssssssso...', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..osssssssssso..', // 7
  '..osssssssssso..', // 8
  '...osssssssso...', // 9
  '....oSssssSo....', // 10
  '.....ooSSoo.....', // 11
  '...ooccccccoo...', // 12
  '..occcccccccco..', // 13
  '..occcccccccCo..', // 14
  '..ococcccccoCo..', // 15
  '..ococccccCoCo..', // 16
  '..osoccccccoso..', // 17
  '...ooppppppoo...', // 18
  '....oppppppo....', // 19
  '....oppooppo....', // 20
  '....obbooPPo....', // 21
  '....ooooobbo....', // 22
  '.........ooo....', // 23
]

const WALK_SIDE_STRIDE_FWD: readonly string[] = [
  '................', // 0
  '.....oooooo.....', // 1
  '....osssssso....', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssssso...', // 5
  '...osssssseso...', // 6
  '...osssssssso...', // 7
  '...ossssssssso..', // 8
  '....ossssssSo...', // 9
  '.....oSsssso....', // 10
  '......oSSo......', // 11
  '....occccco.....', // 12
  '....occccCo.....', // 13
  '....occcCCoo....', // 14
  '....occccCCso...', // 15
  '....occccCoo....', // 16
  '....occccCo.....', // 17
  '....opppppo.....', // 18
  '...opp...ppo....', // 19
  '..opp.....ppo...', // 20
  '..opp.....ppo...', // 21
  '.obb......bbbo..', // 22
  '.oooo....ooooo..', // 23
]

const STAND_SIDE: readonly string[] = [
  '.....oooooo.....', // 0
  '....osssssso....', // 1
  '...osssssssso...', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssseso...', // 5
  '...osssssssso...', // 6
  '...ossssssssso..', // 7
  '....ossssssSo...', // 8
  '.....oSsssso....', // 9
  '......oSSo......', // 10
  '....occccco.....', // 11
  '....occCcCo.....', // 12
  '....occCcCo.....', // 13
  '....occCcCo.....', // 14
  '....occCcCo.....', // 15
  '....occscCo.....', // 16
  '....opppppo.....', // 17
  '.....opppo......', // 18
  '.....opppo......', // 19
  '.....opppo......', // 20
  '.....oPPPo......', // 21
  '.....obbbbo.....', // 22
  '.....oooooo.....', // 23
]

const WALK_SIDE_STRIDE_BACK: readonly string[] = [
  '................', // 0
  '.....oooooo.....', // 1
  '....osssssso....', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssssso...', // 5
  '...osssssseso...', // 6
  '...osssssssso...', // 7
  '...ossssssssso..', // 8
  '....ossssssSo...', // 9
  '.....oSsssso....', // 10
  '......oSSo......', // 11
  '....occccco.....', // 12
  '....occccCo.....', // 13
  '....ocCccCo.....', // 14
  '....oCcccCo.....', // 15
  '...osccccCo.....', // 16
  '....occccCo.....', // 17
  '....opppppo.....', // 18
  '...opp...ppo....', // 19
  '..opp.....ppo...', // 20
  '..opp.....ppo...', // 21
  '.obb......bbbo..', // 22
  '.oooo....ooooo..', // 23
]

export const BODY: Readonly<Record<AgentPose, readonly BodyFrame[]>> = {
  seatedIdle: [{ rows: SEATED_IDLE, headDy: 0 }, { rows: SEATED_IDLE_RAISED, headDy: 0 }],
  seatedTyping: [{ rows: SEATED_TYPING_L, headDy: 0 }, { rows: SEATED_TYPING_R, headDy: 0 }],
  seatedPhone: [{ rows: SEATED_PHONE, headDy: 0 }, { rows: SEATED_PHONE_UP, headDy: 0 }],
  seatedError: [{ rows: SEATED_ERROR, headDy: 0 }],
  stand: [{ rows: STAND, headDy: 0 }, { rows: STAND_BLINK, headDy: 0 }],
  talk: [{ rows: TALK_LOW, headDy: 0 }, { rows: TALK_HIGH, headDy: 0 }],
  walkFront: [{ rows: WALK_FRONT_STEP_L, headDy: 1 }, { rows: STAND, headDy: 0 }, { rows: WALK_FRONT_STEP_R, headDy: 1 }, { rows: STAND, headDy: 0 }],
  walkBack: [{ rows: WALK_BACK_STEP_L, headDy: 1 }, { rows: STAND_BACK, headDy: 0 }, { rows: WALK_BACK_STEP_R, headDy: 1 }, { rows: STAND_BACK, headDy: 0 }],
  walkSide: [{ rows: WALK_SIDE_STRIDE_FWD, headDy: 1 }, { rows: STAND_SIDE, headDy: 0 }, { rows: WALK_SIDE_STRIDE_BACK, headDy: 1 }, { rows: STAND_SIDE, headDy: 0 }],
}

export const VIEW_OF: Readonly<Record<AgentPose, View>> = {
  seatedIdle: 'front',
  seatedTyping: 'front',
  seatedPhone: 'front',
  seatedError: 'front',
  stand: 'front',
  talk: 'side',
  walkFront: 'front',
  walkBack: 'back',
  walkSide: 'side',
}
```

- [ ] **Step 5: Create the hair layers**

Create `web/src/lib/agents/layers/hair.ts` with exactly this content:

```ts
// Hair styles per view (docs/decisions/024). The side view faces right;
// compose.ts mirrors whole frames for facing left.

import type { HairStyle } from '../personas'
import type { View } from './body'
import type { Layer } from './layer'

export const HAIR: Readonly<Record<HairStyle, Readonly<Record<View, Layer>>>> = {
  short: {
    front: {
      top: 1,
      rows: [
        '....hhhhhhhh....', // 1
        '...hhHHhhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hh......hh...', // 4
        '...h........h...', // 5
        '...h........h...', // 6
        '...h........h...', // 7
      ],
    },
    back: {
      top: 1,
      rows: [
        '....hhhhhhhh....', // 1
        '...hhhHHhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hhhhhhhhhh...', // 4
        '...hhhhhhhhhh...', // 5
        '...hhhhhhhhhh...', // 6
        '...hhhhhhhhhh...', // 7
        '....hhhhhhhh....', // 8
        '.....hhhhhh.....', // 9
      ],
    },
    side: {
      top: 1,
      rows: [
        '.....hhhhhh.....', // 1
        '....hhHHhhhh....', // 2
        '....hhhhhhhh....', // 3
        '....hhhhh.......', // 4
        '....hhh.........', // 5
        '....hh..........', // 6
        '....h...........', // 7
      ],
    },
  },
  messy: {
    front: {
      top: 0,
      rows: [
        '.....h..h.h.....', // 0
        '....hhhhhhhh....', // 1
        '...hhHHhhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hh......hh...', // 4
        '...h........h...', // 5
        '...h........h...', // 6
        '...h........h...', // 7
      ],
    },
    back: {
      top: 0,
      rows: [
        '.....h..h.h.....', // 0
        '....hhhhhhhh....', // 1
        '...hhhHHhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hhhhhhhhhh...', // 4
        '...hhhhhhhhhh...', // 5
        '...hhhhhhhhhh...', // 6
        '...hhhhhhhhhh...', // 7
        '....hhhhhhhh....', // 8
        '.....hhhhhh.....', // 9
      ],
    },
    side: {
      top: 0,
      rows: [
        '......h..h......', // 0
        '.....hhhhhh.....', // 1
        '....hhHHhhhh....', // 2
        '....hhhhhhhh....', // 3
        '....hhhhh.......', // 4
        '....hhh.........', // 5
        '....hh..........', // 6
        '....h...........', // 7
      ],
    },
  },
  ponytail: {
    front: {
      top: 1,
      rows: [
        '....hhhhhhhh....', // 1
        '...hhHHhhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hh......hh...', // 4
        '...h........h...', // 5
        '...h........hho.', // 6
        '...h........hhho', // 7
        '.............oho', // 8
        '..............ho', // 9
        '..............o.', // 10
      ],
    },
    back: {
      top: 1,
      rows: [
        '....hhhhhhhh....', // 1
        '...hhhHHhhhhh...', // 2
        '...hhhhhhhhhh...', // 3
        '...hhhhhhhhhh...', // 4
        '...hhhhhhhhhh...', // 5
        '...hhhhhhhhhh...', // 6
        '...hhhhhhhhhh...', // 7
        '....hhhhhhhh....', // 8
        '.....hhhhhh.....', // 9
        '......ohho......', // 10
        '......ohho......', // 11
        '......oHho......', // 12
        '.......oo.......', // 13
      ],
    },
    side: {
      top: 1,
      rows: [
        '.....hhhhhh.....', // 1
        '....hhHHhhhh....', // 2
        '....hhhhhhhh....', // 3
        '....hhhhh.......', // 4
        '..ohhhh.........', // 5
        '.ohhhh..........', // 6
        '.oh.h...........', // 7
        '.oh.............', // 8
        '..o.............', // 9
      ],
    },
  },
}
```

- [ ] **Step 6: Create the accessory layers**

Create `web/src/lib/agents/layers/accessories.ts` with exactly this content:

```ts
// Accessories per view (docs/decisions/024), drawn in mid-tone greys so
// they read against the black outline and the charcoal floor. Held items
// (magnifier, clipboard) are hidden from behind.

import type { Accessory } from '../personas'
import type { View } from './body'
import { EMPTY, type Layer } from './layer'

export const ACCESSORIES: Readonly<Record<Accessory, Readonly<Record<View, Layer>>>> = {
  headset: {
    front: {
      top: 1,
      rows: [
        '....KKKKKKKK....', // 1
        '...K............', // 2
        '..K.............', // 3
        'okk.............', // 4
        'okk.............', // 5
        'okk.............', // 6
        'okk.............', // 7
        '...K............', // 8
        '....Kmm.........', // 9
      ],
    },
    back: {
      top: 1,
      rows: [
        '....KKKKKKKK....', // 1
        '............K...', // 2
        '.............K..', // 3
        '.............kko', // 4
        '.............kko', // 5
        '.............kko', // 6
        '.............kko', // 7
      ],
    },
    side: {
      top: 1,
      rows: [
        '.....KKKKKK.....', // 1
        '....K...........', // 2
      ],
    },
  },
  tie: {
    front: {
      top: 12,
      rows: [
        '.......tt.......', // 12
        '.......tt.......', // 13
        '.......tt.......', // 14
        '.......tt.......', // 15
      ],
    },
    back: EMPTY,
    side: {
      top: 12,
      rows: [
        '.........t......', // 12
        '.........t......', // 13
        '.........t......', // 14
      ],
    },
  },
  headphones: {
    front: {
      top: 0,
      rows: [
        '....KKKKKKKK....', // 0
        '...K........K...', // 1
        '..K..........K..', // 2
        'okk..........kko', // 3
        'ock..........kco', // 4
        'ock..........kco', // 5
        'ock..........kco', // 6
        'okk..........kko', // 7
      ],
    },
    back: {
      top: 0,
      rows: [
        '....KKKKKKKK....', // 0
        '...K........K...', // 1
        '..K..........K..', // 2
        'okk..........kko', // 3
        'ock..........kco', // 4
        'ock..........kco', // 5
        'ock..........kco', // 6
        'okk..........kko', // 7
      ],
    },
    side: {
      top: 0,
      rows: [
        '.....KKKKKK.....', // 0
        '.....K..........', // 1
        '.....K..........', // 2
        '.....K..........', // 3
        '....okkko.......', // 4
        '....okcko.......', // 5
        '....okkko.......', // 6
        '.....ooo........', // 7
      ],
    },
  },
  hood: {
    front: {
      top: 10,
      rows: [
        '....oCCCCCCo....', // 10
        '.......ww.......', // 11
      ],
    },
    back: {
      top: 10,
      rows: [
        '...oCCCCCCCCo...', // 10
        '.....CCCCCC.....', // 11
      ],
    },
    side: {
      top: 10,
      rows: [
        '...oCCC.........', // 10
        '.........w......', // 11
      ],
    },
  },
  goggles: {
    front: {
      top: 3,
      rows: [
        '...KKggKKggKK...', // 3
      ],
    },
    back: {
      top: 3,
      rows: [
        '...kkkkkkkkkk...', // 3
      ],
    },
    side: {
      top: 3,
      rows: [
        '....KKKKKggK....', // 3
      ],
    },
  },
  magnifier: {
    front: {
      top: 11,
      rows: [
        '.............kk.', // 11
        '............kggk', // 12
        '............kgwk', // 13
        '.............kk.', // 14
        '............k...', // 15
      ],
    },
    back: EMPTY,
    side: {
      top: 10,
      rows: [
        '...........kk...', // 10
        '..........kggk..', // 11
        '..........kgwk..', // 12
        '...........kk...', // 13
        '..........k.....', // 14
      ],
    },
  },
  clipboard: {
    front: {
      top: 12,
      rows: [
        '.......kk.......', // 12
        '....owwwwwwo....', // 13
        '....owllllwo....', // 14
        '....owwwwwwo....', // 15
        '....owlllwwo....', // 16
        '....owwwwwwo....', // 17
      ],
    },
    back: EMPTY,
    side: {
      top: 11,
      rows: [
        '...........k....', // 11
        '...........wo...', // 12
        '...........wo...', // 13
        '...........wo...', // 14
        '...........wo...', // 15
        '...........wo...', // 16
      ],
    },
  },
  glasses: {
    front: {
      top: 5,
      rows: [
        '....KKK..KKK....', // 5
        '....K.KKKK.K....', // 6
      ],
    },
    back: EMPTY,
    side: {
      top: 4,
      rows: [
        '.........KKK....', // 4
        '........KK.K....', // 5
      ],
    },
  },
}
```

- [ ] **Step 7: Create the extras**

Create `web/src/lib/agents/layers/extras.ts` with exactly this content:

```ts
// Figures that aren't a persona's body: the empty chair shown for an
// offline agent in the grid view, and the small helper figure drawn beside
// a desk for a same-profile sub-agent (in its owner's hair and shirt).

export const CHAIR: readonly string[] = [
  '................', // 0
  '................', // 1
  '................', // 2
  '................', // 3
  '................', // 4
  '................', // 5
  '................', // 6
  '....oooooooo....', // 7
  '....oQQQQQQo....', // 8
  '....oqqqqqqo....', // 9
  '....oqqqqqqo....', // 10
  '....oqqqqqqo....', // 11
  '....oqqqqqqo....', // 12
  '....oqqqqqqo....', // 13
  '....oqqqqqqo....', // 14
  '....oqqqqqqo....', // 15
  '....oqqqqqqo....', // 16
  '..oQQQQQQQQQQo..', // 17
  '..oqqqqqqqqqqo..', // 18
  '..oqqqqqqqqqqo..', // 19
  '..oooooooooooo..', // 20
  '......oqqo......', // 21
  '...oooooooooo...', // 22
  '...o...oo...o...', // 23
]


/** 8×12 helper figure; wears its owner's hair (h) and shirt (c/C) colours. */
export const HELPER: readonly string[] = [
  '..oooo..', // 0
  '.ohhhho.', // 1
  'ohsssSho', // 2
  'ohesseho', // 3
  '.osssso.', // 4
  '..oSSo..', // 5
  '.occcco.', // 6
  'occcccCo', // 7
  'oscccCso', // 8
  '.oppppo.', // 9
  '.opoopo.', // 10
  '.oboobo.', // 11
]
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `npx vitest run src/lib/agents/layers/layers.test.ts`
Expected: PASS (14 tests).

- [ ] **Step 9: Commit**

```bash
git add web/src/lib/agents/layers/
git commit -m "agents: body, hair and accessory layers (ADR-024)

23 body frames (seated idle/typing/phone/error, standing with blink,
talking, four-frame walks front/back/side) plus hair styles and
accessories per view, the offline chair and the helper figure. Tests
pin sizes, keys, frame counts, seated rows, stride bob and back views.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 3: Composition, timing and the frame cache

**Files:**
- Create: `web/src/lib/agents/compose.ts`
- Test: `web/src/lib/agents/compose.test.ts`

**Interfaces:**
- Consumes: `BODY`, `VIEW_OF`, `SEATED_GROUND`, `SPRITE_W`, `SPRITE_H`, `AgentPose` (Task 2); `HAIR`, `ACCESSORIES`, `CHAIR`, `HELPER`, `Layer` (Task 2); `Persona`, `personaFor`, `paletteFor` (Task 1); `WorkerState` from `web/src/lib/floor.ts` (existing).
- Produces (all from `web/src/lib/agents/compose.ts`):
  - `type Pose = AgentPose | 'offline'`
  - `SPRITE_W`, `SPRITE_H` (re-exported)
  - `frameCount(pose: Pose): number`
  - `groundRow(pose: Pose): number` — 19 for `seated*`, 24 otherwise
  - `composeFrame(persona: Persona, pose: Pose, frame: number): string[]`
  - `mirror(grid: readonly string[]): string[]`
  - `frameAt(pose: Pose, nowMs: number): number`
  - `seatedPoseFor(state: WorkerState): Pose`
  - `agentFrame(persona: Persona, pose: Pose, frame: number, mirrored: boolean): HTMLCanvasElement` (16×24, cached)
  - `helperFrame(persona: Persona): HTMLCanvasElement` (8×12, cached)

- [ ] **Step 1: Write the failing tests**

Create `web/src/lib/agents/compose.test.ts`:

```ts
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npx vitest run src/lib/agents/compose.test.ts`
Expected: FAIL — `Failed to resolve import "./compose"`.

- [ ] **Step 3: Implement the composer**

Create `web/src/lib/agents/compose.ts`:

```ts
// Composes an agent's frame from its layers (docs/decisions/024) and caches
// the painted result, so the grid and office views draw an agent with one
// drawImage. composeFrame() is pure and tested; agentFrame() and
// helperFrame() are the browser-side cache on top of it.

import type { WorkerState } from '../floor'
import { ACCESSORIES } from './layers/accessories'
import { BODY, SEATED_GROUND, SPRITE_H, SPRITE_W, VIEW_OF, type AgentPose } from './layers/body'
import { CHAIR, HELPER } from './layers/extras'
import { HAIR } from './layers/hair'
import type { Layer } from './layers/layer'
import { paletteFor, type Persona } from './personas'

export { SPRITE_H, SPRITE_W }

/** Every pose a view can ask for; 'offline' is the empty chair. */
export type Pose = AgentPose | 'offline'

export function frameCount(pose: Pose): number {
  return pose === 'offline' ? 1 : BODY[pose].length
}

/** Rows drawn from the top: seated figures stop at the waist, the rest stand on the last row. */
export function groundRow(pose: Pose): number {
  return pose.startsWith('seated') ? SEATED_GROUND : SPRITE_H
}

export function composeFrame(persona: Persona, pose: Pose, frame: number): string[] {
  if (pose === 'offline') return [...CHAIR]
  const frames = BODY[pose]
  const body = frames[frame % frames.length]!
  const view = VIEW_OF[pose]
  const out = body.rows.map((row) => [...row])
  const layers = [HAIR[persona.hairStyle][view], ...persona.accessories.map((a) => ACCESSORIES[a][view])]
  for (const layer of layers) stamp(out, layer, body.headDy)
  return out.map((row) => row.join(''))
}

function stamp(out: string[][], layer: Layer, dy: number): void {
  layer.rows.forEach((row, i) => {
    const y = layer.top + i + dy
    if (y < 0 || y >= SPRITE_H) return
    for (let x = 0; x < row.length; x++) {
      if (row[x] !== '.') out[y]![x] = row[x]!
    }
  })
}

export function mirror(grid: readonly string[]): string[] {
  return grid.map((row) => [...row].reverse().join(''))
}

// Which frame shows when: walks step at 8 fps, everything else ticks at
// 5 fps; repeats slow a cycle down (a blink every ~1.6 s, a gesture ~0.8 s).
const SEQUENCES: Readonly<Record<Pose, readonly number[]>> = {
  seatedIdle: [0, 1],
  seatedTyping: [0, 1],
  seatedPhone: [0, 1],
  seatedError: [0],
  stand: [0, 0, 0, 0, 0, 0, 0, 1],
  talk: [0, 0, 1, 1],
  walkFront: [0, 1, 2, 3],
  walkBack: [0, 1, 2, 3],
  walkSide: [0, 1, 2, 3],
  offline: [0],
}

export function frameAt(pose: Pose, nowMs: number): number {
  const seq = SEQUENCES[pose]
  const period = pose.startsWith('walk') ? 125 : 200
  return seq[Math.floor(nowMs / period) % seq.length]!
}

/** The desk pose for a worker state, shared by both views. */
export function seatedPoseFor(state: WorkerState): Pose {
  switch (state) {
    case 'working': return 'seatedTyping'
    case 'error': return 'seatedError'
    case 'delegating': return 'seatedPhone'
    case 'offline': return 'offline'
    default: return 'seatedIdle'
  }
}

// ── Browser-side cache: one small canvas per persona × pose × frame × mirror ──
const cache = new Map<string, HTMLCanvasElement>()

export function agentFrame(persona: Persona, pose: Pose, frame: number, mirrored: boolean): HTMLCanvasElement {
  const key = `${persona.key}|${pose}|${frame}|${mirrored ? 1 : 0}`
  let canvas = cache.get(key)
  if (!canvas) {
    const grid = composeFrame(persona, pose, frame)
    canvas = paint(mirrored ? mirror(grid) : grid, paletteFor(persona))
    cache.set(key, canvas)
  }
  return canvas
}

export function helperFrame(persona: Persona): HTMLCanvasElement {
  const key = `${persona.key}|helper`
  let canvas = cache.get(key)
  if (!canvas) {
    canvas = paint(HELPER, paletteFor(persona))
    cache.set(key, canvas)
  }
  return canvas
}

function paint(grid: readonly string[], palette: Readonly<Record<string, string>>): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = grid[0]!.length
  canvas.height = grid.length
  const ctx = canvas.getContext('2d')
  if (!ctx) return canvas
  grid.forEach((row, y) => {
    for (let x = 0; x < row.length; x++) {
      const ch = row[x]!
      if (ch === '.') continue
      ctx.fillStyle = palette[ch] ?? '#ff00ff' // magenta: an unmapped key is a bug, and visible
      ctx.fillRect(x, y, 1, 1)
    }
  })
  return canvas
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `npx vitest run src/lib/agents/`
Expected: PASS — personas (7), layers (14), compose (14).

- [ ] **Step 5: Typecheck**

Run: `npx vue-tsc -b`
Expected: no output (clean).

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/agents/compose.ts web/src/lib/agents/compose.test.ts
git commit -m "agents: compose layers into frames, with a canvas cache

composeFrame() stacks body, hair and accessories (shifted by headDy)
into a 16x24 grid; frameAt() holds the timing (8 fps walks, 5 fps
otherwise, blink and gesture via repeats); agentFrame()/helperFrame()
paint each frame once and cache it so views draw an agent with one
drawImage.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Grid view — `PixelSprite` and `WorkerStation`

**Files:**
- Modify (rewrite): `web/src/components/PixelSprite.vue`
- Modify (rewrite): `web/src/components/WorkerStation.vue`

**Interfaces:**
- Consumes: `SPRITE_W`, `agentFrame`, `frameAt`, `frameCount`, `groundRow`, `seatedPoseFor`, `Pose` (Task 3); `personaFor` (Task 1).
- Produces: `PixelSprite` props change to `{ profile: string; pose: Pose; scale?: number }` (was `{ pose; shirtColor; scale }`). `WorkerStation` is its only caller.

There are no component tests in this repo (jsdom has no 2D canvas); the visual check is Task 7. `web/src/lib/sprites.ts` stays until Task 6 because the office still imports it.

- [ ] **Step 1: Rewrite `PixelSprite.vue`**

Replace the whole file `web/src/components/PixelSprite.vue` with:

```vue
<script setup lang="ts">
// Renders one agent (docs/decisions/024) onto a <canvas> for the grid view,
// cycling the pose's frames. Frames come from the shared cache in
// src/lib/agents/compose.ts, so each is painted once and blitted here.
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import { SPRITE_W, agentFrame, frameAt, frameCount, groundRow, type Pose } from '@/lib/agents/compose'
import { personaFor } from '@/lib/agents/personas'

const props = withDefaults(defineProps<{ profile: string; pose: Pose; scale?: number }>(), { scale: 3 })

const canvas = useTemplateRef('canvas')
const persona = computed(() => personaFor(props.profile))
// Seated poses stop at the waist; the canvas is cropped to what's drawn.
const rows = computed(() => groundRow(props.pose))
const frame = ref(0)

let timer: ReturnType<typeof setInterval> | undefined
function reducedMotion(): boolean {
  return typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
}

function restartAnimation() {
  if (timer) clearInterval(timer)
  timer = undefined
  frame.value = 0
  if (frameCount(props.pose) <= 1 || reducedMotion()) return
  timer = setInterval(() => {
    frame.value = frameAt(props.pose, performance.now())
  }, 100)
}

function draw() {
  const el = canvas.value
  const ctx = el?.getContext('2d')
  if (!el || !ctx) return
  ctx.imageSmoothingEnabled = false
  ctx.clearRect(0, 0, el.width, el.height)
  const image = agentFrame(persona.value, props.pose, frame.value, false)
  ctx.drawImage(image, 0, 0, SPRITE_W, rows.value, 0, 0, SPRITE_W * props.scale, rows.value * props.scale)
}

watch(() => props.pose, restartAnimation, { immediate: true })
// `onMounted` guarantees the first paint: a plain immediate watcher can fire
// before the <canvas> ref is bound, and a single-frame pose has no timer to
// retry it (fixed in ead4ee4). Pose is tracked here too, so a pose change
// that lands on the same frame number still repaints.
watch([() => props.pose, frame, persona, () => props.scale], draw, { flush: 'post' })
onMounted(draw)
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <canvas
    ref="canvas"
    :width="SPRITE_W * scale"
    :height="rows * scale"
    class="[image-rendering:pixelated]"
    role="img"
    :aria-label="`${profile} ${pose}`"
  />
</template>
```

- [ ] **Step 2: Rewrite `WorkerStation.vue`**

Replace the whole file `web/src/components/WorkerStation.vue` with:

```vue
<script setup lang="ts">
// One workstation on the /floor grid: the agent's persona (docs/decisions/024),
// name, live status, and the two readouts decided in the floor-view grilling
// (open sessions, tool calls in the last 10 minutes). Badges cover the two
// delegation cases that don't draw a line — see src/lib/floor.ts.
import { RouterLink } from 'vue-router'
import type { Worker } from '@/lib/floor'
import { seatedPoseFor } from '@/lib/agents/compose'
import PixelSprite from './PixelSprite.vue'
import StatusWord from './StatusWord.vue'

defineProps<{ worker: Worker }>()
</script>

<template>
  <RouterLink
    :to="{ name: 'agent', params: { profile: worker.profile } }"
    class="station panel relative flex w-44 flex-col items-center gap-1 px-3 py-3 text-center no-underline transition-colors hover:bg-panel-2"
  >
    <span v-if="worker.delegatedBadge" class="absolute top-2 left-2 rounded-sm bg-panel-2 px-1.5 py-0.5 text-faint">delegated</span>
    <span v-if="worker.helperCount" class="absolute top-2 right-2 rounded-sm bg-amber px-1.5 py-0.5 font-bold text-amber-ink">+{{ worker.helperCount }}</span>

    <!-- Tall enough for a standing 16×24 figure (the offline chair) at 4×. -->
    <div class="flex h-[96px] items-end">
      <div class="relative">
        <PixelSprite :profile="worker.profile" :pose="seatedPoseFor(worker.state)" :scale="4" />
        <span
          v-if="worker.state === 'working' && worker.toolName"
          class="panel absolute -top-1 left-1/2 max-w-[9rem] -translate-x-1/2 -translate-y-full truncate px-1.5 py-0.5 text-faint"
        >
          {{ worker.toolName }}
        </span>
        <!-- The status word below already says "error"; the badge is for the eye. -->
        <span
          v-if="worker.state === 'error'"
          aria-hidden="true"
          class="absolute -top-1 left-1/2 grid h-5 w-5 -translate-x-1/2 -translate-y-full place-items-center rounded-full bg-bad font-bold text-bg"
        >!</span>
      </div>
    </div>

    <span class="truncate font-bold">{{ worker.profile }}</span>
    <StatusWord :status="worker.state" />

    <dl class="mt-1 grid w-full grid-cols-2 gap-2 border-t border-line pt-2">
      <div class="readout">
        <dt>Sessions</dt>
        <dd>{{ worker.openSessions }}</dd>
      </div>
      <div class="readout">
        <dt>Tools (10m)</dt>
        <dd>{{ worker.toolCallsRecent }}</dd>
      </div>
    </dl>
  </RouterLink>
</template>
```

- [ ] **Step 3: Typecheck and run all frontend tests**

Run: `npx vue-tsc -b && npx vitest run`
Expected: typecheck clean; all test files pass.

- [ ] **Step 4: Commit**

```bash
git add web/src/components/PixelSprite.vue web/src/components/WorkerStation.vue
git commit -m "floor(grid): draw agents as personas; error badge

PixelSprite takes the profile and a pose and blits cached frames from
lib/agents (seated poses cropped at the waist); WorkerStation passes
the profile, shows a DOM '!' badge on error, and grows its sprite box
to fit a 16x24 figure at 4x.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Office environment — "night control room"

**Files:**
- Modify: `web/src/lib/office/map.ts` (the `FurnitureKind` union and the lounge pieces)
- Modify: `web/src/lib/office/art.ts` (everything from the `// ── Office colours` line to the end of the file)
- Modify: `web/src/lib/office/render.ts` (`drawBackground`; the desk item in `drawScene`)
- Test: `web/src/lib/office/map.test.ts`, `web/src/lib/office/art.test.ts`

**Interfaces:**
- Consumes: existing `officeMap`, `Furniture`, `TILE` (map.ts).
- Produces: `FurnitureKind` gains `'lamp'`; `officePalette` has the ADR-024 keys (`outline`, `floor`, `floorSeam`, `floorFleck`, `carpet`, `carpetSeam`, `wallCap`, `wallFace`, `wallBase`, `mat`, `windowFrame`, `windowSky`, `cityLight`, `star`, `deskTop`, `deskHighlight`, `deskFront`, `monitor`, `screenOff`, `screenOn`, `screenError`, `keyboard`, `chair`, `chairHighlight`, `sofa`, `sofaHighlight`, `tableTop`, `tableFront`, `cup`, `machine`, `led`, `pot`, `leaf`, `leafHighlight`, `shelf`, `shelfBack`, `book1`–`book5`, `glass`, `glassFrame`, `boardFrame`, `board`, `boardLine`, `boardMark`, `rug`, `rugInner`, `rugPattern`, `lamp`); `MONITOR_SCREEN` becomes `[2, -7, 5, 5, 'screenOff']`; `DESK_CHAIR` is redrawn with an outline. Names of exports are unchanged.

The agents in the office still use the old sprites after this task; Task 6 switches them.

- [ ] **Step 1: Write the failing tests**

In `web/src/lib/office/map.test.ts`, add inside `describe('officeMap', …)`, after the last `it`:

```ts
  it('lights the lounge with a floor lamp on the free tile between the coffee machine and the plant', () => {
    expect(m.furniture).toContainEqual({ kind: 'lamp', x: 29, y: 2, w: 1, h: 1, solid: true })
  })
```

In `web/src/lib/office/art.test.ts`, add at the end of the file:

```ts
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npx vitest run src/lib/office/`
Expected: FAIL — the lamp is missing, `officePalette.floor` is `'#caa67f'`, and furniture has no `'outline'` shapes.

- [ ] **Step 3: Add the lamp to the floor plan**

In `web/src/lib/office/map.ts`:

1. In the `FurnitureKind` union, after `| 'rug'`, add `| 'lamp'`.
2. In the header comment, change `The lounge (sofa, coffee machine, coffee table on a rug)` to `The lounge (sofa, coffee machine, floor lamp, coffee table on a rug)`.
3. In `buildOfficeMap()`, in the `// Lounge.` group, directly after `piece('coffeeMachine', 28, 2),` add:

```ts
    piece('lamp', 29, 2),
```

- [ ] **Step 4: Replace the environment art**

In `web/src/lib/office/art.ts`, replace everything from the line `// ── Office colours ─────────────────────────────────────────────────────` to the end of the file with:

```ts
// ── Office colours: the "night control room" (docs/decisions/024) ──────
export const officePalette: Record<string, string> = {
  outline: '#0f0d0c',
  floor: '#26221f', floorSeam: '#2e2a27', floorFleck: '#2a2623',
  carpet: '#23262b', carpetSeam: '#2a2d33',
  wallCap: '#141211', wallFace: '#1f1c1a', wallBase: '#3a3532', mat: '#3a3532',
  windowFrame: '#3a3532', windowSky: '#1a2433', cityLight: '#e8a73a', star: '#a69c90',
  deskTop: '#4a3a2f', deskHighlight: '#5c483a', deskFront: '#33281f',
  monitor: '#151312', screenOff: '#2a2624', screenOn: '#e8a73a', screenError: '#e25b4a',
  keyboard: '#6e665d',
  chair: '#3a3532', chairHighlight: '#4d4742',
  sofa: '#4b4f5c', sofaHighlight: '#5c6170',
  tableTop: '#4a3a2f', tableFront: '#33281f', cup: '#ede6dc',
  machine: '#2b2826', led: '#e25b4a',
  pot: '#5c3a21', leaf: '#3f6b43', leafHighlight: '#57865a',
  shelf: '#33281f', shelfBack: '#26201a',
  book1: '#7a3f30', book2: '#5b7fae', book3: '#b57d22', book4: '#468a4d', book5: '#6e665d',
  glass: 'rgba(127, 167, 217, 0.12)', glassFrame: '#4d4742',
  boardFrame: '#3a3532', board: '#2e2a27', boardLine: '#6e665d', boardMark: '#e8a73a',
  rug: '#5c2f24', rugInner: '#7a3f30', rugPattern: '#a0563f',
  lamp: '#f3d27a',
}

/** [x, y, w, h, colour] in pixels from the piece's top-left tile; y may go up to one tile above. */
export type Shape = readonly [number, number, number, number, string]

/** The monitor screen on a desk, relative to the desk — repainted live (on/off/error). */
export const MONITOR_SCREEN: Shape = [2, -7, 5, 5, 'screenOff']

/**
 * The desk chair, relative to the desk, drawn separately so it sorts
 * *behind* a seated agent while the desk itself sorts in front.
 */
export const DESK_CHAIR: readonly Shape[] = [[9, -13, 14, 12, 'outline'], [10, -12, 12, 10, 'chair'], [10, -12, 12, 2, 'chairHighlight']]

const BOOKS = ['book1', 'book2', 'book3', 'book4', 'book5']

// Every free-standing piece starts with an outline rectangle and is filled
// one pixel inside it, so pieces read as objects on the dark floor.
export function furnitureShapes(kind: FurnitureKind, w: number, h: number): Shape[] {
  const W = w * 16
  const H = h * 16
  switch (kind) {
    case 'desk':
      return [
        [0, 0, 32, 14, 'outline'], [1, 1, 30, 9, 'deskTop'], [1, 1, 30, 1, 'deskHighlight'], [1, 10, 30, 3, 'deskFront'],
        [1, 14, 3, 2, 'outline'], [28, 14, 3, 2, 'outline'],
        [3, 1, 3, 3, 'outline'], [0, -9, 9, 10, 'outline'], [1, -8, 7, 8, 'monitor'], MONITOR_SCREEN,
        [12, 3, 9, 2, 'keyboard'],
      ]
    case 'sofa':
      return [
        [0, -7, W, 21, 'outline'], [1, -6, W - 2, 8, 'sofa'], [2, 2, W - 4, 10, 'sofaHighlight'], [1, 12, W - 2, 1, 'sofa'],
        ...Array.from({ length: w - 1 }, (_, i): Shape => [(i + 1) * 16, 3, 1, 8, 'sofa']),
      ]
    case 'coffeeTable':
      return [[1, 2, 30, 12, 'outline'], [2, 3, 28, 8, 'tableTop'], [2, 11, 28, 2, 'tableFront'], [8, 5, 3, 3, 'cup'], [W - 12, 4, 6, 4, 'book2']]
    case 'coffeeMachine':
      return [[2, -7, 12, 22, 'outline'], [3, -6, 10, 20, 'machine'], [5, -3, 2, 2, 'led'], [6, 8, 4, 4, 'cup']]
    case 'lamp':
      return [[6, -9, 4, 24, 'outline'], [7, -8, 2, 22, 'machine'], [3, -13, 10, 7, 'outline'], [4, -12, 8, 5, 'lamp'], [4, 13, 8, 3, 'outline']]
    case 'plant':
      return [
        [2, -9, 12, 11, 'outline'], [3, -8, 10, 9, 'leaf'], [6, -10, 4, 3, 'leafHighlight'], [1, -4, 4, 5, 'leafHighlight'],
        [3, 2, 10, 11, 'outline'], [4, 3, 8, 9, 'pot'],
      ]
    case 'bookshelf': {
      const shapes: Shape[] = [[0, -13, W, 28, 'outline'], [1, -12, W - 2, 26, 'shelf'], [2, -10, W - 4, 7, 'shelfBack'], [2, -1, W - 4, 7, 'shelfBack']]
      for (const top of [-10, -1]) {
        for (let x = 3, i = top === -10 ? 0 : 2; x + 3 <= W - 3; x += 4, i++) {
          const short = i % 3 === 1
          shapes.push([x, top + (short ? 1 : 0), 3, short ? 6 : 7, BOOKS[i % BOOKS.length]!])
        }
      }
      return shapes
    }
    case 'meetingTable':
      return [
        [0, 3, W, H - 5, 'outline'], [1, 4, W - 2, H - 11, 'tableTop'], [1, H - 7, W - 2, 3, 'tableFront'],
        [2, H - 3, 3, 3, 'outline'], [W - 5, H - 3, 3, 3, 'outline'],
        [10, 10, 8, 6, 'cup'], [W - 24, 12, 7, 5, 'cup'], [W / 2 - 2, 14, 3, 3, 'cup'],
      ]
    case 'chair':
      return [[2, 1, 12, 13, 'outline'], [3, 2, 10, 3, 'chairHighlight'], [3, 5, 10, 7, 'chair'], [3, 14, 2, 2, 'outline'], [11, 14, 2, 2, 'outline']]
    case 'glass':
      if (h === 1) {
        return [
          [0, 0, W, 11, 'glass'], [0, 11, W, 2, 'glassFrame'],
          ...Array.from({ length: w - 1 }, (_, i): Shape => [(i + 1) * 16 - 1, 0, 1, 11, 'glassFrame']),
        ]
      }
      return [[5, 0, 6, H, 'glass'], [4, 0, 1, H, 'glassFrame'], [11, 0, 1, H, 'glassFrame']]
    case 'whiteboard':
      return [
        [2, 1, W - 4, 12, 'boardFrame'], [3, 2, W - 6, 10, 'board'],
        [6, 4, 14, 1, 'boardLine'], [6, 7, 20, 1, 'boardLine'], [W - 20, 5, 10, 1, 'boardMark'],
        [8, 13, W - 16, 1, 'boardFrame'],
      ]
    case 'window':
      return [
        [2, 1, W - 4, 12, 'windowFrame'], [3, 2, W / 2 - 4, 10, 'windowSky'], [W / 2 + 1, 2, W / 2 - 4, 10, 'windowSky'],
        [5, 9, 1, 1, 'cityLight'], [8, 10, 1, 1, 'cityLight'], [W / 2 + 4, 8, 1, 1, 'cityLight'], [W / 2 + 9, 10, 1, 1, 'cityLight'],
        [6, 4, 1, 1, 'star'], [W / 2 + 6, 3, 1, 1, 'star'], [11, 5, 1, 1, 'star'],
      ]
    case 'rug':
      return [
        [0, 0, W, H, 'rug'], [4, 4, W - 8, H - 8, 'rugInner'],
        [8, 8, W - 16, 2, 'rugPattern'], [8, H - 10, W - 16, 2, 'rugPattern'],
        [8, 8, 2, H - 16, 'rugPattern'], [W - 10, 8, 2, H - 16, 'rugPattern'],
      ]
  }
}
```

- [ ] **Step 5: Repaint the background**

In `web/src/lib/office/render.ts`, replace the whole `drawBackground` function with:

```ts
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
      // Charcoal floor tiles with 1 px seams and the odd fleck.
      fill(officePalette.floor!, x, y, TILE, TILE)
      fill(officePalette.floorSeam!, x, y + TILE - 1, TILE, 1)
      fill(officePalette.floorSeam!, x + TILE - 1, y, 1, TILE)
      if ((tx * 7 + ty * 3) % 5 === 0) fill(officePalette.floorFleck!, x + 5, y + 6, 1, 1)
    }
  }

  // Walls: a dark cap all round, a dark face along the top.
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

  // The lounge lamp's glow, painted once: a warm pool on the rug.
  ctx.fillStyle = officePalette.lamp!
  ctx.globalAlpha = 0.1
  for (const f of m.furniture) {
    if (f.kind !== 'lamp') continue
    for (const r of [40, 22]) {
      ctx.beginPath()
      ctx.arc(f.x * TILE + TILE / 2, (f.y + 2) * TILE, r, 0, Math.PI * 2)
      ctx.fill()
    }
  }
  ctx.globalAlpha = 1
}
```

Then, in `drawScene`, replace the desk item's `draw` body:

```ts
      draw: () => {
        drawShapes(ctx, x, y, furnitureShapes('desk', 2, 1))
        const [sx, sy, sw, sh] = MONITOR_SCREEN
        ctx.fillStyle = officePalette[view.screen === 'on' ? 'screenOn' : view.screen === 'error' ? 'screenError' : 'screenOff']!
        ctx.fillRect(x + sx, y + sy, sw, sh)
      },
```

with:

```ts
      draw: () => {
        drawShapes(ctx, x, y, furnitureShapes('desk', 2, 1))
        const [sx, sy, sw, sh] = MONITOR_SCREEN
        ctx.fillStyle = officePalette[view.screen === 'on' ? 'screenOn' : view.screen === 'error' ? 'screenError' : 'screenOff']!
        ctx.fillRect(x + sx, y + sy, sw, sh)
        if (view.screen !== 'off') {
          // A lit screen spills onto the desk's left half (ADR-024: 18 %).
          ctx.globalAlpha = 0.18
          ctx.fillRect(x + 1, y + 1, 15, 9)
          ctx.globalAlpha = 1
        }
      },
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `npx vitest run src/lib/office/ && npx vue-tsc -b`
Expected: all office tests pass (including the existing footprint and `DESK_CHAIR` bounds tests); typecheck clean.

- [ ] **Step 7: Commit**

```bash
git add web/src/lib/office/map.ts web/src/lib/office/map.test.ts web/src/lib/office/art.ts web/src/lib/office/art.test.ts web/src/lib/office/render.ts
git commit -m "office: night control room environment (ADR-024)

Charcoal tiled floor, dark walls with night windows, outlined dark-wood
furniture, monitors as light sources (amber reflection while working,
red on error), and a warm floor lamp with a painted glow in the
lounge. Floor plan, zones and paths are unchanged; the lamp takes the
free tile (29, 2).

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Office agents — personas, overlay, cleanup

**Files:**
- Modify: `web/src/lib/office/art.ts` (the header and character section — everything above `// ── Office colours`)
- Modify: `web/src/lib/office/render.ts` (imports, `DeskView`, `ActorView`, helpers in `drawScene`, `drawActor`; delete `drawGrid`)
- Modify: `web/src/components/OfficeView.vue`
- Modify (rewrite): `web/src/lib/office/art.test.ts`
- Delete: `web/src/lib/sprites.ts`, `web/src/lib/sprites.test.ts`

**Interfaces:**
- Consumes: `agentFrame`, `frameAt`, `groundRow`, `helperFrame`, `seatedPoseFor`, `Pose` (Task 3); `Persona`, `personaFor` (Task 1); `SEATED_GROUND`, `SPRITE_H` (Task 2).
- Produces: `officePose(v: CharacterView): { pose: Pose; mirrored: boolean }` in `office/art.ts` (replaces `characterFrames`); `DeskView.owner: Persona | null` (replaces `shirt`); `ActorView.persona: Persona` (replaces `shirt`).

- [ ] **Step 1: Write the failing tests**

Replace the whole file `web/src/lib/office/art.test.ts` with:

```ts
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npx vitest run src/lib/office/art.test.ts`
Expected: FAIL — `officePose` is not exported from `./art`.

- [ ] **Step 3: Replace the character section of `office/art.ts`**

In `web/src/lib/office/art.ts`, replace everything **above** the line `// ── Office colours: the "night control room" (docs/decisions/024) ──────` with:

```ts
// Pixel art for the /floor Office view: the "night control room"
// environment (docs/decisions/024) — outlined furniture in the dashboard's
// palette, monitors as the light sources — and the mapping from an actor's
// state to the agent pose drawn for it. The agents themselves live in
// src/lib/agents/, shared with the grid view.

import type { WorkerState } from '../floor'
import { seatedPoseFor, type Pose } from '../agents/compose'
import type { FurnitureKind } from './map'
import type { Facing, Stance } from './motion'

export interface CharacterView {
  walking: boolean
  facing: Facing
  pose: Stance
  state: WorkerState
}

/** The agent pose (and whether to mirror it) for an actor this frame. */
export function officePose(v: CharacterView): { pose: Pose; mirrored: boolean } {
  if (v.walking) {
    switch (v.facing) {
      case 'down': return { pose: 'walkFront', mirrored: false }
      case 'up': return { pose: 'walkBack', mirrored: false }
      case 'right': return { pose: 'walkSide', mirrored: false }
      case 'left': return { pose: 'walkSide', mirrored: true }
    }
  }
  if (v.pose === 'seated') return { pose: seatedPoseFor(v.state), mirrored: false }
  switch (v.facing) {
    case 'right': return { pose: 'talk', mirrored: false }
    case 'left': return { pose: 'talk', mirrored: true }
    // No spot rests facing up today; show the front rather than walk in place.
    default: return { pose: 'stand', mirrored: false }
  }
}

```

- [ ] **Step 4: Draw agents and helpers from the frame cache in `render.ts`**

In `web/src/lib/office/render.ts`:

1. Replace the import line
   `import { DESK_CHAIR, HELPER, MONITOR_SCREEN, characterFrames, furnitureShapes, officePalette, spritePalette, type Shape } from './art'`
   with:

```ts
import { agentFrame, frameAt, groundRow, helperFrame } from '../agents/compose'
import type { Persona } from '../agents/personas'
import { DESK_CHAIR, MONITOR_SCREEN, furnitureShapes, officePalette, officePose, type Shape } from './art'
```

2. Replace the `DeskView` and `ActorView` interfaces with:

```ts
/** What a desk shows this frame. */
export interface DeskView {
  desk: number
  screen: 'off' | 'on' | 'error'
  /** Same-profile sub-agents to draw beside the desk (the view caps this at 3). */
  helpers: number
  /** The desk owner's persona, for the helpers; null for an unassigned desk. */
  owner: Persona | null
}

export interface ActorView {
  actor: Actor
  state: WorkerState
  persona: Persona
}
```

3. In `drawScene`, replace the helpers block

```ts
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
```

with:

```ts
    if (view.owner && view.helpers > 0) {
      const helper = helperFrame(view.owner)
      items.push({
        bottom: y, order: 1,
        draw: () => {
          for (let i = 0; i < Math.min(view.helpers, 3); i++) ctx.drawImage(helper, x - 9 * (i + 1), y - helper.height)
        },
      })
    }
```

4. Replace the whole `drawActor` function with:

```ts
function drawActor(ctx: CanvasRenderingContext2D, a: ActorView, x: number, y: number, nowMs: number, still: boolean): void {
  const walking = a.actor.waypoints.length > 0
  const { pose, mirrored } = officePose({ walking, facing: a.actor.facing, pose: a.actor.pose, state: a.state })
  const frame = still ? 0 : frameAt(pose, nowMs)
  // The 16-wide figure fills its 16 px box, its ground row on the box's bottom
  // edge: a standing head rises 8 px into the row above; a seated waist meets
  // the desk top.
  ctx.drawImage(agentFrame(a.persona, pose, frame, mirrored), x, y + TILE - groundRow(pose))
}
```

5. Delete the whole `drawGrid` function (at the end of the file).

- [ ] **Step 5: Switch `OfficeView.vue` to personas and the new figure size**

In `web/src/components/OfficeView.vue`:

1. Replace `import { shirtColor } from '@/lib/sprites'` with:

```ts
import { SEATED_GROUND, SPRITE_H } from '@/lib/agents/layers/body'
import { personaFor } from '@/lib/agents/personas'
```

2. In `deskViews()`, replace `if (!o) return { desk, screen: 'off', helpers: 0, shirt: null }` with `if (!o) return { desk, screen: 'off', helpers: 0, owner: null }`, and replace `shirt: shirtColor(o.worker.profile),` with `owner: personaFor(o.worker.profile),`.

3. In `actorViews()`, replace `views.push({ actor, state: o.worker.state, shirt: shirtColor(o.worker.profile) })` with `views.push({ actor, state: o.worker.state, persona: personaFor(o.worker.profile) })`.

4. Directly below the line `const px = (n: number) => \`${n * scale.value}px\``, add:

```ts
// Where a figure is drawn, in map pixels: seated figures stop at the waist
// (on the desk top), standing ones rise 8 px above their tile.
function figureBox(p: Placed): { top: number; height: number } {
  const rows = p.seated ? SEATED_GROUND : SPRITE_H
  return { top: p.y + TILE - rows, height: rows }
}
```

5. In the tool-bubble `<span>`, replace `top: px(placed.get(o.worker.profile)!.y + 2)` with `top: px(figureBox(placed.get(o.worker.profile)!).top - 1)`.

6. Directly after the closing `</template>` of the tool-bubble loop (the one keyed `` `tool-${o.worker.profile}` ``), add:

```vue
      <!-- Error: a "!" above the head (the sprite no longer carries one). -->
      <template v-for="o in scene.occupants" :key="`err-${o.worker.profile}`">
        <span
          v-if="o.worker.state === 'error' && placed.get(o.worker.profile)?.seated"
          aria-hidden="true"
          class="pointer-events-none absolute grid -translate-x-1/2 -translate-y-full place-items-center rounded-full bg-[#e25b4a] font-bold leading-none text-[#1b1917]"
          :style="{
            left: px(placed.get(o.worker.profile)!.x + TILE / 2),
            top: px(figureBox(placed.get(o.worker.profile)!).top - 1),
            width: px(9),
            height: px(9),
          }"
        >!</span>
      </template>
```

7. In the agent `<button>`'s `:style`, replace

```
            top: px(placed.get(o.worker.profile)!.y - 2),
```

with `top: px(figureBox(placed.get(o.worker.profile)!).top),`, and replace

```
            height: px(TILE + 2),
```

with `height: px(figureBox(placed.get(o.worker.profile)!).height),`.

- [ ] **Step 6: Delete the old sprites**

```bash
git rm web/src/lib/sprites.ts web/src/lib/sprites.test.ts
```

- [ ] **Step 7: Check nothing still references them, then run everything**

Run: `grep -rn "lib/sprites\|characterFrames\|spritePalette\|shirtColor" src; npx vue-tsc -b && npx vitest run`
Expected: `grep` prints nothing; typecheck clean; all test files pass.

- [ ] **Step 8: Commit**

```bash
git add -A web/src
git commit -m "office: agents as personas from lib/agents; remove old sprites

render.ts draws each agent with one cached frame (officePose picks the
pose and mirror from the actor) and helpers in their owner's colours;
OfficeView passes personas, sizes the agent buttons and tool bubbles to
the 16x24 figure, and shows a DOM '!' badge on error. The 14x17
sprites.ts and the office's own character frames are gone.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Verify in a browser, document, open the PR

**Files:**
- Modify: `README.md` (the `Floor` row of "What it shows")
- Modify: `docs/prd.md` (§ 11 table)
- Scratch, not committed: a mock BFF under the session scratchpad (or `/tmp`)

**Interfaces:**
- Consumes: everything above.
- Produces: a pushed branch and an open PR.

- [ ] **Step 1: Full build and tests**

Run from the repo root: `make build && CGO_ENABLED=0 go test -count=1 ./... && (cd web && npx vitest run)`
Expected: build succeeds; Go and frontend tests pass.

- [ ] **Step 2: Start a mock BFF that varies state over time**

Save this as `mock_bff.js` in the scratchpad (not in the repo) and run `node mock_bff.js &`:

```js
// Scratch mock of the BFF for checking /floor visually. Not part of the repo.
const http = require('http')
const START = Date.now()
const iso = (ms = 0) => new Date(Date.now() + ms).toISOString()
const session = (o) => ({
  id: 's', source: 'api', model: 'x', title: null, preview: null, started_at: iso(-600000), ended_at: null, end_reason: null,
  open: true, last_active: iso(0), message_count: 3, tool_call_count: 1, api_call_count: 1, parent_session_id: null,
  usage: { input_tokens: 1, output_tokens: 1, cache_read_tokens: 0, cache_write_tokens: 0, reasoning_tokens: 0 },
  estimated_cost_usd: 0, actual_cost_usd: null, ...o,
})
const overview = () => ({
  generated_at: iso(0), gateway: null,
  agents: [
    { profile: 'default', status: 'healthy', latency_ms: 12 },
    { profile: 'coder-agent', status: 'healthy', latency_ms: 20 },
    { profile: 'tester-agent', status: 'degraded', latency_ms: 340 },
    { profile: 'product-agent', status: 'healthy', latency_ms: 15 },
    { profile: 'qa-agent', status: 'healthy', latency_ms: 9 },
    Date.now() - START > 20000
      ? { profile: 'legacy-agent', status: 'healthy', latency_ms: 9 }
      : { profile: 'legacy-agent', status: 'unreachable', latency_ms: 0, error: { code: 'upstream_unreachable', message: 'refused' } },
  ],
})
function stream(profile, res) {
  res.writeHead(200, { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache', Connection: 'keep-alive' })
  const emit = (type, data) => res.write(`event: ${type}\ndata: ${JSON.stringify({ type, profile, at: iso(0), ...data })}\n\n`)
  const snap = (s) => emit('session.snapshot', { session_id: s.id, session: s })
  if (profile === 'default') {
    snap(session({ id: 's-default-1', started_at: iso(-120000) }))
    for (let i = 0; i < 4; i++) snap(session({ id: `s-default-helper-${i}`, parent_session_id: 's-default-1' }))
  } else if (profile === 'coder-agent') {
    const child = session({ id: 's-coder-child', parent_session_id: 's-default-1', started_at: iso(-30000) })
    snap(child)
    emit('tool.started', { session_id: child.id, tool: 'pytest', call_id: 'c1', preview: '{}' })
  } else if (profile === 'tester-agent') {
    snap(session({ id: 's-tester-1' }))
    emit('tool.started', { session_id: 's-tester-1', tool: 'terminal', call_id: 'c2', preview: '{}', at: iso(-2000) })
    emit('tool.completed', { session_id: 's-tester-1', tool: 'terminal', call_id: 'c2', preview: '{"exit_code":1,"error":null}' })
  } else if (profile === 'product-agent') {
    snap(session({ id: 's-product-1', last_active: iso(-600000) }))
    setTimeout(() => {
      snap(session({ id: 's-product-1', last_active: iso(0) }))
      emit('tool.started', { session_id: 's-product-1', tool: 'web_search', call_id: 'c3', preview: '{}' })
    }, 10000)
  }
  const keep = setInterval(() => res.write(': keepalive\n\n'), 15000)
  res.on('close', () => clearInterval(keep))
}
http.createServer((req, res) => {
  const url = new URL(req.url, 'http://x')
  if (url.pathname === '/api/overview') { res.writeHead(200, { 'Content-Type': 'application/json' }); return res.end(JSON.stringify(overview())) }
  if (url.pathname === '/api/usage') { res.writeHead(200, { 'Content-Type': 'application/json' }); return res.end('{"generated_at":"2026-09-27T00:00:00Z","window_hours":24,"profiles":[],"errors":[]}') }
  const m = /^\/api\/agents\/([^/]+)\/activity\/stream$/.exec(url.pathname)
  if (m) return stream(decodeURIComponent(m[1]), res)
  res.writeHead(404); res.end('{}')
}).listen(9094, () => console.log('mock on :9094'))
```

Then from `web/`: `VITE_API_TARGET=http://127.0.0.1:9094 npx vite --port 5179 --strictPort &` and open `http://localhost:5179/floor` in a browser at ≥ 1100 px wide.

- [ ] **Step 3: Check both views**

In **Grid** (toggle "grid"):
- Each of `default`, `coder-agent`, `tester-agent`, `product-agent` shows its persona (headset + tie / headphones / goggles + magnifier / ponytail + clipboard); `qa-agent` shows a procedural persona in a muted shirt.
- `tester-agent` shows the red "!" badge; `coder-agent` shows its `pytest` bubble above the head; `legacy-agent` shows the empty chair until it comes online (~20 s).
- The browser console has no errors.

In **Office** (toggle "office"):
- Dark tiled floor, night windows, outlined furniture, amber lit monitors with a reflection, a red monitor at `tester-agent`'s desk, the warm lamp glow in the lounge.
- Seated agents sit with their waist on the desk top; `default` stands beside `coder-agent`'s desk, facing left; `default`'s desk has three helper figures in amber and a "+1" nameplate tag.
- After ~10 s `product-agent` walks from the lounge to its desk (watch the four-frame walk); after ~20 s `legacy-agent` enters by the door.
- Tab through the agent buttons: each focus ring surrounds the whole 16×24 figure; the "!" badge and tool bubble sit just above the head.
- The browser console has no errors.

Fix anything that fails, rerun Step 1, and commit the fix before continuing.

- [ ] **Step 4: Stop the scratch servers**

Run: `pkill -f "vite --port 5179"; pkill -f mock_bff.js`

- [ ] **Step 5: Update README and PRD**

In `README.md`, replace the `| Floor | … |` row of "What it shows" with:

```
| Floor | Each profile illustrated as a worker — a persona per role (orchestrator, coder, tester, product) in the dashboard's instrument-panel palette ([ADR-024](docs/decisions/024-agent-art-redesign.md)) — in two views: a grid of stations with lines to whichever profile each is delegating to ([ADR-021](docs/decisions/021-floor-view.md)), or a "night control room" office where agents sit at their desk while working, walk over to a colleague's desk to delegate, and drift to the lounge when idle ([ADR-023](docs/decisions/023-office-view.md)). |
```

In `docs/prd.md`, add this row at the end of the § 11 table:

```
| Agent art redesign | Four role personas (16×24, outlined) in the instrument-panel palette, a procedural persona for any other profile, and a "night control room" office. Layered pixel art composed at runtime and cached; no new dependency, no backend change. | [ADR-024](decisions/024-agent-art-redesign.md) |
```

- [ ] **Step 6: Commit the docs**

```bash
git add README.md docs/prd.md
git commit -m "docs: README and PRD entries for the agent art redesign

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 7: Push and open the PR**

```bash
git push -u origin claude/agent-art-redesign
gh pr create --base main --head claude/agent-art-redesign \
  --title "floor: agent personas and a night-control-room office (ADR-024)" \
  --body "Implements docs/decisions/024-agent-art-redesign.md: four role personas and a procedural fallback, 16x24 outlined figures composed from layers and cached, the Grid and Office views switched to them, and the Office environment restyled in the instrument-panel palette. No backend change, no new dependency.

🤖 Generated with [Claude Code](https://claude.com/claude-code)"
```

Expected: the PR URL is printed.
