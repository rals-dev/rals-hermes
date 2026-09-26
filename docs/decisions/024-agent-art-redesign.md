# ADR-024: Agent art redesign — personas in a "night control room"

Date: 2026-09-27 · Status: Accepted · Supersedes in part: ADR-021 (sprite art direction), ADR-023 § 4 (art)

## Context

The floor view's agents (ADR-021, ADR-023) all share one 14×17 figure that
differs only by a hashed shirt colour. Three problems were raised in the
2026-09-27 brainstorming session:

1. **Identity** — the four agents look like twins; from across the room
   there's no telling the coder from the tester, and two profiles can hash
   to the same shirt colour.
2. **Quality** — no outline, no shading, 1 px arms, stiff proportions.
3. **Style** — the full-colour art (ADR-021's deliberate departure from the
   instrument-panel theme) now reads as disconnected from the rest of the
   dashboard.

Mockups for the style and persona questions were reviewed in a browser
companion during the session; they are not committed. The values below are
the source of truth.

## Decision

### Direction: pixel art in the instrument-panel palette ("night control room")

Pixel art stays — it's what makes the floor view an *illustration* rather
than another readout — but its palette is locked to the dashboard's own
language: a charcoal floor, black 1 px outlines, amber as the colour of
"something is on", and role colours taken from the status lamps. A schematic
/ pictogram style was rejected (it loses the character that is the point of
the view, and pictograms are hard to tell apart), and a refined full-colour
style was rejected (it doesn't fix the style problem). Warmth from the
full-colour option survives as an accent in the lounge.

The office art keeps fixed colours rather than theme tokens (as in ADR-023):
it is a dark scene in both the dark and light app themes.

### Figures: 16×24, outlined, two-tone shading

Every frame is exactly 16×24 pixels with a black (`#0f0d0c`) outline and a
highlight/shadow tone per material. The error "!" leaves the sprite and
becomes a DOM badge above the head (grid and office), so frames need no
headroom rows and the marker is crisp at every scale and readable by screen
readers.

### Personas

Built in for the four known profile names; role colour = shirt colour.

| Profile | Persona | Silhouette cues | Hair (base / highlight) | Shirt (base / shade) |
| --- | --- | --- | --- | --- |
| `default` | Orchestrator | one-sided headset with an amber mic, dark tie | `#3b3330` / `#8a8078` (greying temples) | `#e8a73a` / `#b57d22` (amber lamp) |
| `coder-agent` | Coder | two-sided headphones with blue cups, hoodie strings | `#2a2320` / `#4a3c33` | `#7fa7d9` / `#5b7fae` (auth-blue lamp) |
| `tester-agent` | Tester | goggles on the forehead, magnifying glass in hand | `#6b3f2a` / `#8c5a3e` | `#63b86b` / `#468a4d` (ok-green lamp) |
| `product-agent` | Product | ponytail, clipboard held in front | `#c9a36a` / `#e0c28e` | `#c98a6a` / `#9c6549` (terracotta) |

Silhouettes differ even without colour, which matters for colour-blind
viewers. Accessories are drawn in mid-tone greys (`#6e665d` / `#a69c90`) —
darker greys disappeared against the outline and floor in review.

**Any other profile name** gets a deterministic procedural persona from a hash
of the name: a hair style (short, messy, ponytail) × a hair colour × a light
accessory (none, glasses, hood strings) × a shirt from a muted set that
excludes the four role colours, so a fallback never impersonates a role.

| Fallback set | Values (base / shade) |
| --- | --- |
| Shirts | stone `#a69c90` / `#7d746a`, violet `#8c7fb8` / `#6a5f94`, teal `#5f9e9a` / `#467874`, olive `#a39a5b` / `#7d7542`, rose `#b87a8c` / `#8f5a6a`, slate `#7d8a99` / `#5d6875` |
| Hair | `#2a2320` / `#4a3c33`, `#4a3c33` / `#6b574a`, `#1b1917` / `#3a3532`, `#8a8078` / `#b3aaa0` |

Shared character colours: skin `#d9b99b` / `#b8957a`, eyes `#1b1917`,
trousers `#3a3532` / `#2a2624`, shoes `#151312`, paper `#ede6dc` with lines
`#8a8078`, lens `#cfe3ea`, tie `#2b2826`.

### Art model: layers composed at runtime (`web/src/lib/agents/`)

Pixel data stays hand-authored as string grids in the repository (as ADR-023
§ 4), but split into layers so the amount of art grows by addition, not
multiplication:

| File | Holds |
| --- | --- |
| `palette.ts` | shared character colours and the role colours |
| `personas.ts` | `Persona` = hair style + accessory + palette; the four built-ins; `personaFor(profile)` with the procedural fallback |
| `layers/body.ts` | body frames per pose, without hair or accessories; each frame records `headDy` |
| `layers/hair.ts` | each hair style × view (front, back, side) |
| `layers/accessories.ts` | each accessory × view; held items have an empty back view |
| `compose.ts` | `composeFrame(persona, pose, frame)` → 16×24 grid (pure); `agentFrame(...)` → cached offscreen canvas (browser only) |

Layers stack body → hair → accessory; hair and accessories shift by the body
frame's `headDy`. Only the right-facing side view is drawn; left is its
mirror. The cache key is persona × pose × frame × mirrored, built lazily and
bounded (23 body frames, 29 with mirrored side views, per persona), so drawing
an agent is one `drawImage`.

The old 14×17 `web/src/lib/sprites.ts` and its tests are removed.

### Pose inventory (body layer)

| Pose | Frames | Used for |
| --- | --- | --- |
| Seated — idle | 2 (breathing) | at the desk, idle under 2 minutes |
| Seated — typing | 2 (hands alternate) | working |
| Seated — phone | 2 | delegating with no child desk to visit |
| Seated — error | 1 (hands on head) | error |
| Standing front | 1 + blink | lounge |
| Side — talking | 2 (gesture) | visiting a child's desk |
| Walk front / back / side | 4 each (step, pass, step, pass; head up 1 px on pass) | walking |

Plus a mini helper figure (8×12) in its owner's hair and shirt colours, no
accessories. Offline stays an empty chair (furniture).

### Placement

- **Office** (16 px tiles): a standing figure's feet sit on its tile's bottom
  edge and its head rises 8 px into the row above; the existing bottom-edge
  y-sort handles the overlap. A seated figure's waist sits on the desk's top
  edge, centred across the two desk tiles; legs are not drawn. The monitor
  moves 1 px left to clear the shoulder.
- **Grid card**: `PixelSprite` stays at 4×, so the sprite becomes 64×96 px
  (was 56×68); the card grows in height only.

### Office environment

The floor plan, zones and behaviour of ADR-023 are unchanged; only palette
and shapes change.

| Element | Colours |
| --- | --- |
| Floor (tiles, seams, flecks) | `#26221f`, `#2e2a27`, `#2a2623` |
| Wall cap / face / baseboard | `#141211`, `#1f1c1a`, `#3a3532` |
| Night windows (frame, sky, city lights, stars) | `#3a3532`, `#1a2433`, `#e8a73a`, `#a69c90` |
| Desk (top, highlight, front) | `#4a3a2f`, `#5c483a`, `#33281f` |
| Monitor (bezel; screen on / error / off) | `#151312`; `#e8a73a` / `#e25b4a` / `#2a2624` |
| Chair | `#3a3532`, highlight `#4d4742` |
| Lounge rug (border, body, pattern) | `#5c2f24`, `#7a3f30`, `#a0563f` |
| Lounge lamp glow | `#f3d27a` at low opacity |
| Sofa | `#4b4f5c`, `#5c6170` |
| Plants (pot, leaves, highlight) | `#5c3a21`, `#3f6b43`, `#57865a` |
| Coffee machine (body, LED) | `#2b2826`, `#e25b4a` |
| Coffee table / meeting table (top, front) | `#4a3a2f`, `#33281f`; cups and papers `#ede6dc` |
| Keyboard | `#6e665d` |
| Bookshelf (frame, back) and books | `#33281f`, `#26201a`; `#7a3f30`, `#5b7fae`, `#b57d22`, `#468a4d`, `#6e665d` |
| Whiteboard (frame, board, lines, mark) | `#3a3532`, `#2e2a27`, `#6e665d`, `#e8a73a` |
| Meeting-room carpet (body, seam) | `#23262b`, `#2a2d33` |
| Glass partition (pane, frame) | `rgba(127, 167, 217, 0.12)`, `#4d4742` |
| Door mat | `#3a3532` |
| Outline (all furniture) | `#0f0d0c` |

All furniture gets the 1 px black outline and a 1 px highlight. Monitors are
the light sources: an amber reflection on the desk while working, red on
error, dark when off or unassigned (the reflection is the screen colour at
18 % opacity over the desk's left half). One piece of furniture is added: a
floor lamp on the free tile (29, 2) between the coffee machine and the plant;
its glow (two circles of `#f3d27a` at 10 % opacity, radii 40 and 22 px) is
painted once into the static background layer. No zone, spot or path
changes. The DOM nameplates are unchanged.

### Integration

- **Grid** — `PixelSprite.vue` takes `profile` + `pose` instead of `pose` +
  `shirtColor` and derives the persona itself; `shirtColor()` goes away. The
  `onMounted` first-paint fix stays. `WorkerStation.vue` shows the error
  badge.
- **Office** — `render.ts` draws agents via `agentFrame`, 8 px above their
  tile box, and helpers via the owner's mini figure; the state → pose mapping
  is unchanged, only its image source. `office/art.ts` holds only furniture
  and the environment palette; `drawBackground` paints the tiled floor, night
  windows and lamp glow. In `OfficeView.vue` the agent buttons become 16×24
  to match the figures, tool bubbles sit 8 px higher, and a new DOM error badge
  replaces the sprite's "!".
- **Unchanged** — reduced-motion behaviour, the adaptive loop, placement and
  motion logic, nameplates, the popover.

## Consequences

- Each agent is recognisable by silhouette, hair and role colour, and the
  floor view shares the rest of the dashboard's visual language.
- Pixel data roughly triples, but in layers: a new persona is a palette plus
  at most one hair style and one accessory (three views each), not a redraw
  of every pose.
- Persona identity is keyed to profile *names*. Renaming a profile drops it to
  the procedural fallback — still distinct, but without role meaning. Moving
  personas into `config.yaml` remains possible later without discarding this
  art.
- The art ceiling is what can be hand-authored as grids. `agentFrame` is the
  seam where licensed sprite sheets could replace composition later.
- New gestures (coffee, stretching) were considered and deferred: every new
  pose is more body frames to draw, and this redesign already needs 23.

## Testing

- `personas`: the four names map to their personas; the fallback is
  deterministic and distinct across a sample of names, and never uses a role
  colour.
- `compose`: every persona × pose × frame is exactly 16×24 with known palette
  keys; left equals mirrored right; front views show the accessory; held items
  vanish from the back view; `headDy` moves hair and accessories.
- Helper figures use their owner's hair and shirt colours.
- `office/art`: furniture shapes stay inside their footprints with known
  colours.
- Browser check against a time-varying mock BFF: both views, every state,
  all four personas plus a fallback, four-direction walks, badge, bubble and
  button positions, nameplate contrast, no console errors.
