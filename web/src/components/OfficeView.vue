<script setup lang="ts">
// The /floor "Office" view (docs/decisions/023): the same Worker states as
// the grid, shown as agents walking between their desk, a colleague's desk
// and the lounge on a pixel-art floor plan.
//
// Rendering is a hand-written canvas loop that only runs at 60 fps while
// someone is walking, drops to a 5 fps tick for typing/blinking, and stops
// while the tab is hidden — this is meant to sit on a wall display all day.
// Text (nameplates, tool bubbles) and everything clickable live in a DOM
// overlay above the canvas: crisp at any scale, focusable, and readable by
// screen readers, which a canvas on its own is not.
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, useTemplateRef, watch } from 'vue'
import type { DelegationLink, Worker } from '@/lib/floor'
import { DESK_COUNT, TILE, officeMap } from '@/lib/office/map'
import { deriveOffice, placementKey, type OfficeOccupant } from '@/lib/office/placement'
import { moveTo, placeAt, spotFor, stepActor, type Actor } from '@/lib/office/motion'
import { MAP_H, MAP_W, drawBackground, drawScene, type ActorView, type DeskView } from '@/lib/office/render'
import { shirtColor } from '@/lib/sprites'
import { lampTone } from '@/lib/status'
import WorkerStation from './WorkerStation.vue'

const props = defineProps<{ workers: Worker[]; links: DelegationLink[]; now: Date }>()
const emit = defineEmits<{ useGrid: [] }>()

const scene = computed(() => deriveOffice(props.workers, props.links, props.now.getTime()))
const occupantAt = computed(() => new Map(scene.value.occupants.map((o) => [o.desk, o])))

const wrap = useTemplateRef<HTMLDivElement>('wrap')
const canvas = useTemplateRef<HTMLCanvasElement>('canvas')
const scale = ref(2)

// Actors change every frame, so they live outside Vue's reactivity; the
// overlay reads a snapshot published once per drawn frame.
interface Placed { x: number; y: number; visible: boolean; walking: boolean; seated: boolean }
const actors = new Map<string, Actor>()
const keys = new Map<string, string>()
const placed = shallowRef(new Map<string, Placed>())

let background: HTMLCanvasElement | null = null
const reducedMotion = () => typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

function sync(first: boolean) {
  // Nobody sees a walk on first render, with reduced motion, or in a hidden tab.
  const instant = first || reducedMotion() || document.hidden
  const seen = new Set<string>()
  for (const o of scene.value.occupants) {
    const p = o.worker.profile
    seen.add(p)
    const key = placementKey(o.placement)
    const spot = spotFor(officeMap, o.placement)
    const leaving = o.placement.kind === 'absent'
    const current = actors.get(p)
    if (!current) {
      actors.set(p, placeAt(spot, !leaving))
      keys.set(p, key)
      continue
    }
    if (keys.get(p) === key) continue
    keys.set(p, key)
    // Arriving from outside: appear at the door, then walk in.
    const from = !current.visible && !leaving ? placeAt(spotFor(officeMap, { kind: 'absent' }), true) : current
    actors.set(p, moveTo(officeMap, from, spot, { instant, hideOnArrival: leaving }))
  }
  for (const p of [...actors.keys()]) {
    if (!seen.has(p)) {
      actors.delete(p)
      keys.delete(p)
    }
  }
  render()
  if (anyWalking()) kick()
}

function anyWalking(): boolean {
  for (const a of actors.values()) if (a.waypoints.length > 0) return true
  return false
}

function deskViews(): DeskView[] {
  return Array.from({ length: DESK_COUNT }, (_, desk): DeskView => {
    const o = occupantAt.value.get(desk)
    if (!o) return { desk, screen: 'off', helpers: 0, shirt: null }
    const s = o.worker.state
    return {
      desk,
      screen: s === 'working' || s === 'delegating' ? 'on' : s === 'error' ? 'error' : 'off',
      helpers: s === 'offline' ? 0 : o.worker.helperCount ?? 0,
      shirt: shirtColor(o.worker.profile),
    }
  })
}

function actorViews(): ActorView[] {
  const views: ActorView[] = []
  for (const o of scene.value.occupants) {
    const actor = actors.get(o.worker.profile)
    if (actor) views.push({ actor, state: o.worker.state, shirt: shirtColor(o.worker.profile) })
  }
  return views
}

function render() {
  const ctx = canvas.value?.getContext('2d')
  if (ctx && background) {
    ctx.setTransform(scale.value, 0, 0, scale.value, 0, 0)
    drawScene(ctx, officeMap, background, deskViews(), actorViews(), performance.now(), reducedMotion())
  }
  const next = new Map<string, Placed>()
  let changed = false
  for (const [p, a] of actors) {
    const v: Placed = {
      x: Math.round(a.x), y: Math.round(a.y), visible: a.visible,
      walking: a.waypoints.length > 0, seated: a.pose === 'seated' && a.waypoints.length === 0,
    }
    const prev = placed.value.get(p)
    if (!prev || prev.x !== v.x || prev.y !== v.y || prev.visible !== v.visible || prev.walking !== v.walking || prev.seated !== v.seated) changed = true
    next.set(p, v)
  }
  // The 5 fps tick repaints the canvas; the overlay only re-renders on movement.
  if (changed || next.size !== placed.value.size) placed.value = next
}

// ── Loop: rAF while walking, a 5 fps tick otherwise, nothing when hidden ──
let raf = 0
let last = 0
let idle: ReturnType<typeof setInterval> | undefined

function frame(ts: number) {
  const dt = last ? Math.min(ts - last, 100) : 16 // cap: no teleport after a stall
  last = ts
  for (const [p, a] of actors) if (a.waypoints.length > 0) actors.set(p, stepActor(a, dt))
  render()
  if (anyWalking() && !document.hidden) {
    raf = requestAnimationFrame(frame)
  } else {
    raf = 0
    last = 0
    startIdle()
  }
}
function kick() {
  stopIdle()
  if (!raf && !document.hidden) {
    last = 0
    raf = requestAnimationFrame(frame)
  }
}
function startIdle() {
  if (idle || document.hidden || reducedMotion()) return
  idle = setInterval(render, 200)
}
function stopIdle() {
  if (idle) clearInterval(idle)
  idle = undefined
}
function stopAll() {
  if (raf) cancelAnimationFrame(raf)
  raf = 0
  last = 0
  stopIdle()
}
function onVisibility() {
  if (document.hidden) return stopAll()
  render()
  if (anyWalking()) kick()
  else startIdle()
}

// ── Scale: the largest whole multiple of the map that fits ──
let observer: ResizeObserver | undefined
function measure() {
  const width = wrap.value?.clientWidth ?? MAP_W
  scale.value = Math.min(4, Math.max(1, Math.floor(width / MAP_W)))
}

onMounted(() => {
  background = document.createElement('canvas')
  background.width = MAP_W
  background.height = MAP_H
  const bg = background.getContext('2d')
  if (bg) drawBackground(bg, officeMap)
  measure()
  observer = new ResizeObserver(measure)
  if (wrap.value) observer.observe(wrap.value)
  document.addEventListener('visibilitychange', onVisibility)
  // After mount, so the <canvas> ref is bound for the very first paint.
  void nextTick(() => {
    sync(true)
    startIdle()
  })
})
onBeforeUnmount(() => {
  stopAll()
  observer?.disconnect()
  document.removeEventListener('visibilitychange', onVisibility)
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeydown)
})

watch(scene, () => sync(false))
// Resizing a canvas clears it; repaint once the new size is in the DOM.
watch(scale, () => render(), { flush: 'post' })

// ── Overlay ──
const px = (n: number) => `${n * scale.value}px`
// Nameplate text grows with the map; never below 10 px (at 1× it truncates instead).
const fontPx = computed(() => `${Math.max(10, Math.round(5 * scale.value + 1))}px`)

function deskBox(desk: number) {
  const d = officeMap.desks[desk]!
  return { left: px(d.x * TILE - TILE / 2), top: px((d.y + 1) * TILE), width: px(3 * TILE) }
}

function labelFor(o: OfficeOccupant): string {
  const parts = [o.worker.profile, o.worker.state]
  if (o.worker.toolName) parts.push(o.worker.toolName)
  if (o.placement.kind === 'visiting') parts.push(`at ${o.placement.child}'s desk`)
  if (o.placement.kind === 'lounge') parts.push('in the lounge')
  return parts.join(', ')
}

const summary = computed(() => {
  const here = scene.value.occupants.filter((o) => o.placement.kind !== 'absent').length
  return `Office floor plan: ${here} of ${scene.value.occupants.length} agents in the office`
})

// ── Popover ──
const selected = ref<string | null>(null)
const selectedOccupant = computed(() => scene.value.occupants.find((o) => o.worker.profile === selected.value) ?? null)
const popover = useTemplateRef<HTMLDivElement>('popover')
let opener: HTMLElement | null = null

function open(profile: string, ev: Event) {
  opener = ev.currentTarget as HTMLElement
  selected.value = profile
  document.addEventListener('pointerdown', onPointerDown)
  document.addEventListener('keydown', onKeydown)
}
function close() {
  selected.value = null
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeydown)
  opener?.focus()
  opener = null
}
function onPointerDown(ev: PointerEvent) {
  const t = ev.target as Node
  if (popover.value?.contains(t) || opener?.contains(t)) return
  close()
}
function onKeydown(ev: KeyboardEvent) {
  if (ev.key === 'Escape') close()
}

const popoverStyle = computed(() => {
  const at = selected.value ? placed.value.get(selected.value) : undefined
  if (!at) return {}
  const width = 190
  const mapW = MAP_W * scale.value
  const left = Math.min(Math.max((at.x + TILE) * scale.value + 6, 0), mapW - width)
  const top = Math.max((at.y - TILE) * scale.value, 0)
  return { left: `${left}px`, top: `${top}px`, width: `${width}px` }
})
</script>

<template>
  <div ref="wrap" class="w-full">
    <p v-if="scene.overflow.length" class="mb-3 text-muted">
      The office has {{ DESK_COUNT }} desks; {{ scene.overflow.join(', ') }} only
      {{ scene.overflow.length === 1 ? 'appears' : 'appear' }} in the grid.
      <button type="button" class="text-amber hover:underline" @click="emit('useGrid')">Switch to Grid</button>
    </p>

    <div class="relative mx-auto overflow-hidden rounded-sm" :style="{ width: px(MAP_W), height: px(MAP_H), fontSize: fontPx }">
      <canvas
        ref="canvas"
        :width="MAP_W * scale"
        :height="MAP_H * scale"
        class="absolute inset-0 [image-rendering:pixelated]"
        role="img"
        :aria-label="summary"
      />

      <!-- Nameplates: always visible, readable from across the room. Fixed dark
           plates rather than theme tokens: they sit on the (fixed-colour) wood
           floor, so a light-theme plate would lose contrast there. -->
      <div
        v-for="o in scene.occupants"
        :key="`plate-${o.worker.profile}`"
        class="pointer-events-none absolute flex flex-col items-center leading-tight"
        :style="deskBox(o.desk)"
      >
        <span class="flex max-w-full items-center gap-1 rounded-sm bg-[#1b1917]/80 px-1 text-[#ede6dc]">
          <span class="lamp !h-[0.6em] !w-[0.6em]" :class="lampTone(o.worker.state)" aria-hidden="true" />
          <span class="truncate font-bold">{{ o.worker.profile }}</span>
        </span>
        <span v-if="o.delegatedFrom.length" class="rounded-sm bg-[#1b1917]/70 px-1 text-[#e8a73a]">← {{ o.delegatedFrom.join(', ') }}</span>
        <span v-else-if="o.worker.delegatedBadge" class="rounded-sm bg-[#1b1917]/70 px-1 text-[#a69c90]">delegated</span>
        <span v-if="(o.worker.helperCount ?? 0) > 3 && o.worker.state !== 'offline'" class="rounded-sm bg-[#e8a73a] px-1 font-bold text-[#1b1917]">
          +{{ o.worker.helperCount! - 3 }}
        </span>
      </div>

      <!-- Tool bubbles over agents typing at their desk. -->
      <template v-for="o in scene.occupants" :key="`tool-${o.worker.profile}`">
        <span
          v-if="o.worker.state === 'working' && o.worker.toolName && placed.get(o.worker.profile)?.seated"
          class="pointer-events-none absolute -translate-x-1/2 -translate-y-full truncate rounded-sm border border-[#4d4742] bg-[#242120] px-1 text-[#ede6dc]"
          :style="{ left: px(placed.get(o.worker.profile)!.x + TILE / 2), top: px(placed.get(o.worker.profile)!.y + 2), maxWidth: px(5 * TILE) }"
        >
          {{ o.worker.toolName }}
        </span>
      </template>

      <!-- One focusable target per agent in the office, following them as they walk. -->
      <template v-for="o in scene.occupants" :key="`btn-${o.worker.profile}`">
        <button
          v-if="placed.get(o.worker.profile)?.visible"
          type="button"
          class="absolute cursor-pointer rounded-sm"
          :style="{
            left: px(placed.get(o.worker.profile)!.x),
            top: px(placed.get(o.worker.profile)!.y - 2),
            width: px(TILE),
            height: px(TILE + 2),
          }"
          :aria-label="labelFor(o)"
          :aria-expanded="selected === o.worker.profile"
          @click="open(o.worker.profile, $event)"
        />
      </template>

      <div
        v-if="selectedOccupant"
        ref="popover"
        role="dialog"
        :aria-label="`${selectedOccupant.worker.profile} details`"
        class="absolute z-20 text-[13.5px]"
        :style="popoverStyle"
      >
        <div class="relative">
          <WorkerStation :worker="selectedOccupant.worker" class="!w-full shadow-lg" />
          <button
            type="button"
            class="absolute -top-2.5 -right-2.5 z-10 grid h-6 w-6 place-items-center rounded-full border border-line bg-panel text-muted hover:text-ink"
            aria-label="Close"
            @click="close"
          >
            ×
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
