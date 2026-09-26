<script setup lang="ts">
// The shop floor: each agent illustrated as a worker, either as a grid of
// workstations (docs/decisions/021) or as a pixel-art office they walk
// around in (docs/decisions/023). Both views read the same deriveFloor()
// output (src/lib/floor.ts), built from /api/overview plus the per-profile
// activity streams — no dedicated backend endpoint.
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { api } from '@/api/client'
import type { Overview } from '@/api/types'
import { useActivityFeed } from '@/lib/useActivityFeed'
import { deriveFloor } from '@/lib/floor'
import WorkerStation from '@/components/WorkerStation.vue'
import DelegationLines from '@/components/DelegationLines.vue'
import OfficeView from '@/components/OfficeView.vue'
import ErrorState from '@/components/ErrorState.vue'
import Skeleton from '@/components/Skeleton.vue'

const overview = useQuery({
  queryKey: ['overview'],
  queryFn: () => api.get<Overview>('/api/overview'),
})

const profiles = computed(() => overview.data.value?.agents.map((a) => a.profile) ?? [])
const { rows, sessions } = useActivityFeed(profiles)

// One clock for every time-windowed state (working/error thresholds).
const now = ref(new Date())
const tick = setInterval(() => { now.value = new Date() }, 1000)
onBeforeUnmount(() => clearInterval(tick))

const floor = computed(() => deriveFloor(overview.data.value, sessions.value, rows.value, now.value))

// Grid or Office. The office is a 512 px map drawn at a whole-number scale,
// so it's only legible at 2× or more: the default only where 2× fits (page
// content is capped at max-w-7xl = 1080 px at this app's 13.5 px root font,
// which leaves ~1040 px in the tighter office panel). Whatever the viewer
// picks is remembered in this browser.
type FloorView = 'grid' | 'office'
const VIEW_KEY = 'floor.view'
function initialView(): FloorView {
  try {
    const saved = localStorage.getItem(VIEW_KEY)
    if (saved === 'grid' || saved === 'office') return saved
  } catch { /* storage unavailable: fall through to the default */ }
  return window.innerWidth >= 1100 ? 'office' : 'grid'
}
const view = ref<FloorView>(initialView())
watch(view, (v) => {
  try { localStorage.setItem(VIEW_KEY, v) } catch { /* not persisted; still switches */ }
  // Grid stations re-mount; their delegation lines need fresh positions.
  if (v === 'grid') void nextTick(recomputePositions)
})

// Delegation lines are drawn from measured DOM positions, not layout math:
// the grid wraps at arbitrary widths and station count varies with config.
const floorEl = ref<HTMLElement | null>(null)
const stationEls = new Map<string, HTMLElement>()
function setStationRef(profile: string, el: Element | null) {
  if (el) stationEls.set(profile, el as HTMLElement)
  else stationEls.delete(profile)
}
const positions = ref<Record<string, { x: number; y: number }>>({})
const size = ref({ width: 0, height: 0 })

function recomputePositions() {
  const container = floorEl.value?.getBoundingClientRect()
  if (!container) return
  size.value = { width: container.width, height: container.height }
  const next: Record<string, { x: number; y: number }> = {}
  for (const [profile, el] of stationEls) {
    const r = el.getBoundingClientRect()
    next[profile] = { x: r.left - container.left + r.width / 2, y: r.top - container.top + r.height / 2 }
  }
  positions.value = next
}

const isFullscreen = ref(false)
function onFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
  void nextTick(recomputePositions)
}
function toggleFullscreen() {
  if (document.fullscreenElement) void document.exitFullscreen()
  else void floorEl.value?.requestFullscreen()
}

let observer: ResizeObserver | undefined
onMounted(() => {
  recomputePositions()
  observer = new ResizeObserver(() => recomputePositions())
  if (floorEl.value) observer.observe(floorEl.value)
  window.addEventListener('resize', recomputePositions)
  document.addEventListener('fullscreenchange', onFullscreenChange)
})
onBeforeUnmount(() => {
  observer?.disconnect()
  window.removeEventListener('resize', recomputePositions)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
})

// Station count changes with the profile list; wait for the DOM to catch up.
watch(
  () => floor.value.workers.map((w) => w.profile).join(','),
  () => void nextTick(recomputePositions),
)
</script>

<template>
  <div class="space-y-4">
    <header class="flex flex-wrap items-baseline justify-between gap-3">
      <h1 class="text-lg font-bold tracking-tight">Floor</h1>
      <div class="flex items-baseline gap-5">
        <div role="group" aria-label="View" class="flex overflow-hidden rounded-sm border border-line">
          <button
            v-for="v in (['grid', 'office'] as const)"
            :key="v"
            type="button"
            class="px-2.5 py-0.5 capitalize transition-colors"
            :class="view === v ? 'bg-amber text-amber-ink' : 'text-muted hover:text-ink'"
            :aria-pressed="view === v"
            @click="view = v"
          >
            {{ v }}
          </button>
        </div>
        <button type="button" class="text-muted hover:text-ink" @click="toggleFullscreen">
          {{ isFullscreen ? 'Exit full screen' : 'Full screen' }}
        </button>
      </div>
    </header>
    <p v-if="view === 'grid'" class="text-faint">Each station illustrates one profile's live state; delegation lines connect a profile to the profile it delegated to.</p>
    <p v-else class="text-faint">Agents sit at their desk while working, walk over to a colleague's desk when they delegate to them, and head to the lounge after two idle minutes. Click an agent for details.</p>

    <Skeleton v-if="overview.isPending.value" :rows="2" label="Loading floor" />
    <ErrorState v-else-if="overview.isError.value" :error="overview.error.value" :retry="() => overview.refetch()" />
    <div
      v-else
      ref="floorEl"
      class="relative overflow-auto rounded-md"
      :class="[
        view === 'office' ? 'p-1.5' : 'p-6',
        isFullscreen ? ['fixed inset-0 z-50 bg-bg', view === 'office' && 'grid content-center'] : 'panel',
      ]"
    >
      <OfficeView v-if="view === 'office'" :workers="floor.workers" :links="floor.links" :now="now" @use-grid="view = 'grid'" />
      <template v-else>
        <DelegationLines :links="floor.links" :positions="positions" :width="size.width" :height="size.height" />
        <div class="relative z-10 flex flex-wrap gap-6">
          <div v-for="w in floor.workers" :key="w.profile" :ref="(el) => setStationRef(w.profile, el as Element | null)">
            <WorkerStation :worker="w" />
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
