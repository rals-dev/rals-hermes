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
