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
