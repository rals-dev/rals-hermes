# ADR-023: Office view — agents walking around a 2D pixel-art office

Date: 2026-09-26 · Status: Accepted · Art (§ 4) superseded by [ADR-024](024-agent-art-redesign.md)

## Context

The floor view (ADR-021) shows each profile as a card in a grid. The
operator asked for a "virtual 2D office" instead: agents as characters in
a room, where *where they are* says what they're doing, readable at a
glance on a wall display. Everything it needs already exists —
`deriveFloor()` produces per-profile states (offline/error/delegating/
working/idle), the tool in flight and the delegation links — so this is a
new way of drawing the same data, not a new data source.

## Decision

1. **Agents walk between zones that mean something.** Not a static
   seating chart (too close to the grid) and not an explorable
   Gather-style space with an avatar for the viewer (the dashboard is
   read-only; an avatar could not *do* anything).
2. **One route, two views.** `/floor` gets a **Grid | Office** toggle.
   Office is the default only where it can be drawn at 2× or more; the
   viewer's choice is remembered per browser (`localStorage`, failing
   soft). The grid stays as the dense view and the one that works on
   narrow screens.
3. **Hand-written Canvas 2D, no engine.** The need is small — one static
   map, up to eight characters, a short path on a ~600-tile grid. Phaser
   would have added roughly seven times the app's current JS for features
   left unused. The logic (floor plan, state → destination, BFS,
   movement) is pure TypeScript under `web/src/lib/office/` with unit
   tests; only `render.ts` and `OfficeView.vue` touch a canvas.
4. **Hand-authored art, reusing the grid's sprites.** Characters share the
   grid view's head rows, palette and `shirtColor()`, so an agent looks
   the same in both views. New art: a standing body, two-frame walk cycles
   (down, up, right; left is the mirror), a blink, a talking gesture, a
   small helper figure, and furniture as flat rectangles. The repository
   is public (ADR-015), which ruled out committing commercial pixel-art
   packs whose licences forbid redistributing the raw assets.
5. **State → place** (`placement.ts`):

   | State | Where |
   | --- | --- |
   | working, error | Seated at their own desk (error shows "!" and a red monitor) |
   | delegating | Standing beside their **most recent** child's desk, facing them — the delegation's direction is visible without a line |
   | idle | At their desk at first; in the lounge once **idle for 2 minutes** |
   | offline | Out of the office: an empty desk with a nameplate |
   | same-profile sub-agents | Up to 3 small figures beside the desk, then "+n" |

   The 2-minute wait stops agents pacing between desk and lounge in the
   pauses between tool calls (idle starts 60 s after the last activity).
   It's measured from the profile's last known activity rather than a
   client-side timer, so it's stateless: a reload puts everyone where they
   would have walked to anyway.
6. **A fixed, hand-laid floor plan with eight desks** (`map.ts`, 32×18
   tiles of 16 px): desks in two rows of four, a lounge, a decorative
   meeting room behind glass, one door. Desks are assigned in
   configuration order; unassigned desks have no nameplate; profiles
   beyond eight get a note pointing to the grid. A generated layout was
   rejected — hand-placed rooms are far easier to make look right.
7. **Text and interaction in a DOM overlay.** Nameplates (lamp + name,
   "← parent", "delegated", "+n") and tool bubbles are always visible, for
   the wall display. Each agent has a transparent `<button>` that follows
   them, labelled for screen readers ("coder-agent, working, pytest");
   clicking opens a popover with that agent's `WorkerStation` card, and Esc
   or an outside click closes it and returns focus. A bare canvas would be
   mouse-only and invisible to assistive technology.
8. **An adaptive render loop.** `requestAnimationFrame` only while someone
   is walking (at 1/16 px/ms ≈ 3.9 tiles/s, exact in binary), a 5 fps
   tick for typing and blinking otherwise, and nothing while the tab is
   hidden. Moves are instant on first render, under
   `prefers-reduced-motion`, and in a hidden tab — nobody would see the
   walk, and a queue of walks shouldn't replay when the tab comes back.
9. **Scale is a whole number**, the largest that fits, so pixels stay
   square.

## Found while building it

- **2× needs ~1100 px of viewport, not 1024.** This app's 13.5 px root
  font makes `max-w-7xl` 1080 px; inside the grid's `p-6` panel the map
  fitted only at 1× on *every* screen. The office panel uses `p-1.5`
  instead (~1041 px of room): 2× in the page, 3× in full screen at
  1920 px.
- **Draw order is by bottom edge.** Furniture and agents are y-sorted each
  frame; the desk chair is a separate shape list so it sorts behind a
  seated agent while the desk (and its monitor) sorts in front of anyone
  walking behind it.
- **Frontend dependencies are not governed by ADR-014.** Earlier text
  (the floor-view PR and one commit message) cited ADR-014 as forbidding
  new npm packages; it only covers Go dependencies. No new npm package was
  added here by choice (point 3), not by rule.

## Consequences

- No backend change; the office view is two new additive fields on the
  existing derivation (`Worker.lastActiveAt`, `DelegationLink.childStartedAt`)
  plus frontend code.
- Cross-profile delegation — the "walk over to a colleague's desk" case —
  has, like the grid's delegation line, only been exercised with synthetic
  data (ADR-021), not against a real Hermes run.
- Art is deliberately simple; `art.ts` is the only file to change to
  improve it, and the renderer only depends on grids of palette keys.
- Browsers throttle `requestAnimationFrame` in unfocused or background
  panes. Each frame advances at most 100 ms of walking (so a stall never
  teleports anyone), which means walks there run slower — but still
  arrive.
