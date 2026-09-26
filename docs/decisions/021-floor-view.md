# ADR-021: Floor view — agents illustrated as workers

Date: 2026-09-20 · Status: Accepted · Art direction superseded by [ADR-024](024-agent-art-redesign.md)

## Context

The overview page answers "is anything happening?" in numbers and words.
The operator asked for something more illustrative: a page where each
profile is a worker at a station, so the state of the whole deployment
reads at a glance — including whether one profile is currently delegating
to another. This has to come entirely from data the BFF already exposes;
no new endpoint, no persisted state (consistent with ADR-006/007's "no
database" constraint).

## Decision

- New route `/floor`, in the primary nav, with a small "Floor →" link from
  Overview. Not a replacement for Overview — a separate, illustrative view.
- `deriveFloor()` (`web/src/lib/floor.ts`) turns `AgentCard` status, the
  latest per-profile session snapshot, and live feed rows into one
  `WorkerState` per profile — `offline > error > delegating > working >
  idle`, in that priority — plus a list of cross-profile delegation links.
  Thresholds: a tool failure or `error` event marks `error` for 30s: an
  open session with `last_active` under 60s old, or an in-flight tool
  call, marks `working`.
- Delegation is derived, not signalled directly: Hermes emits
  `subagent.start` on the **child** session and it only carries the
  parent's session id (`internal/activity/hub.go`), not the parent's
  profile. Three cases, distinguished because the parent may not be known
  yet:
  1. Parent session found, different profile → a delegation line.
  2. Parent session found, same profile → a "+n" helper badge, no line.
  3. Parent session not found (outside the activity window, or the
     parent's own profile hasn't reported a snapshot yet) → a "delegated"
     badge, no line.
  Only same-profile delegation has been observed against a real Hermes so
  far; cross-profile and unknown-parent are covered by synthetic fixtures
  in `floor.test.ts`, not by recorded traffic.
- Each station: a pixel-art sprite (see below), profile name, status
  word+lamp, a tool-name bubble while working, and two readouts — open
  sessions, tool calls in the last 10 minutes — both already available
  from data the page fetches for the feed.
- Delegation lines are amber, animated (marching dashes), drawn on an SVG
  layer positioned from measured DOM rects (`ResizeObserver` + resize +
  `fullscreenchange`), because the station grid wraps at arbitrary widths
  and the station count follows `upstream.profiles` in `config.yaml`
  (1–8+), not a fixed layout.
- A "Full screen" button (Fullscreen API) supports wall-display use — the
  reason a separate page was chosen over folding this into Overview.

### Sprites: full colour, deliberately off-theme

The rest of the dashboard is the "instrument panel" theme (ADR from the
2026-09-19 UI pass — warm charcoal, amber, B612). The floor view's workers
are **full-colour pixel art**, not palette-locked to the theme tokens —
an explicit, discussed departure: the theme reads correctly everywhere
else, and this page is allowed its own register because it is
illustrative rather than a readout. Each worker's shirt colour is a
deterministic hash of the profile name (`shirtColor()` in
`web/src/lib/sprites.ts`), so the same profile always wears the same
colour.

Sprites are hand-authored pixel data (`string[]` grids, one character per
pixel) rather than an asset pack — kept inside the repo, no licensing
question, editable in a PR. A shared `BASE` grid (head + torso) is merged
with a small overlay per pose carrying only what moves (arms, a phone, an
exclamation mark), so `idle`/`working`/`delegating` animate over two
frames each (`steps`-style, 5 fps) without redrawing the whole figure;
`error` and `offline` are static. `offline` has no figure at all — an
empty chair — matching "nobody's there" more literally than a slumped
character would. Animation respects `prefers-reduced-motion` (frames stop
advancing; delegation lines stop marching). `PixelSprite.vue` only depends
on `framesFor()` returning same-shaped grids, so swapping this module for
a licensed sprite-sheet later doesn't touch the renderer.

## Consequences

- No backend change at all; `/floor` is pure frontend, built from
  `/api/overview` and the existing per-profile SSE streams.
- `StatusWord.vue`'s tone mapping gained three cases (`working` → ok,
  `delegating` → warn, `offline` → bad) since it's already reused across
  several status vocabularies (session state, gateway state, stream link
  state) — extended rather than duplicated in a floor-specific component.
- The cross-profile delegation line is functionally untested against a
  real Hermes deployment; if it turns out delegated child sessions don't
  reliably arrive before (or soon after) their parent's own snapshot in
  practice, the line may show up later than the badge does, or not at
  all within the 10-minute activity window. Worth revisiting once a real
  delegation has been observed end-to-end.
- Sprite art quality is intentionally minimal for v1 (a handful of
  hand-placed pixels); `sprites.ts` is the only file that would need to
  change to upgrade it.
