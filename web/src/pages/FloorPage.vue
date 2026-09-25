<script setup lang="ts">
// The shop floor: one workstation per profile, illustrating each agent as
// a worker (docs/decisions/021). Built entirely from data the app already
// fetches — /api/overview plus the per-profile activity streams — through
// deriveFloor() (src/lib/floor.ts). No new backend endpoint.
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { api } from '@/api/client'
import type { Overview } from '@/api/types'
import { useActivityFeed } from '@/lib/useActivityFeed'
import { deriveFloor } from '@/lib/floor'
import WorkerStation from '@/components/WorkerStation.vue'
import DelegationLines from '@/components/DelegationLines.vue'
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
    <header class="flex items-baseline justify-between">
      <h1 class="text-lg font-bold tracking-tight">Floor</h1>
      <button type="button" class="text-muted hover:text-ink" @click="toggleFullscreen">
        {{ isFullscreen ? 'Exit full screen' : 'Full screen' }}
      </button>
    </header>
    <p class="text-faint">Each station illustrates one profile's live state; delegation lines connect a profile to the profile it delegated to.</p>

    <Skeleton v-if="overview.isPending.value" :rows="2" label="Loading floor" />
    <ErrorState v-else-if="overview.isError.value" :error="overview.error.value" :retry="() => overview.refetch()" />
    <div
      v-else
      ref="floorEl"
      class="relative overflow-auto rounded-md p-6"
      :class="isFullscreen ? 'fixed inset-0 z-50 bg-bg' : 'panel'"
    >
      <DelegationLines :links="floor.links" :positions="positions" :width="size.width" :height="size.height" />
      <div class="relative z-10 flex flex-wrap gap-6">
        <div v-for="w in floor.workers" :key="w.profile" :ref="(el) => setStationRef(w.profile, el as Element | null)">
          <WorkerStation :worker="w" />
        </div>
      </div>
    </div>
  </div>
</template>
