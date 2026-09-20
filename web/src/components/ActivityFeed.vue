<script setup lang="ts">
import { computed, onBeforeUnmount, ref, toRef } from 'vue'
import { useActivityFeed } from '@/lib/useActivityFeed'
import FeedRow from './FeedRow.vue'
import EmptyState from './EmptyState.vue'
import StatusWord from './StatusWord.vue'
import { RouterLink } from 'vue-router'
import { relativeTime, shortId } from '@/lib/format'

const props = defineProps<{ profiles: string[]; title?: string }>()
const { rows, sessions, status, fresh } = useActivityFeed(toRef(props, 'profiles'))

// Sessions the pollers are watching right now, most recently active first.
const watched = computed(() =>
  [...sessions.value.values()].sort((a, b) => Date.parse(b.last_active) - Date.parse(a.last_active)).slice(0, 6),
)

// One clock for every relative timestamp; ticks once a second.
const now = ref(new Date())
const tick = setInterval(() => { now.value = new Date() }, 1000)
onBeforeUnmount(() => clearInterval(tick))

const link = computed(() => {
  const s = Object.values(status.value)
  if (s.length === 0) return 'closed'
  if (s.every((v) => v === 'open')) return 'live'
  if (s.some((v) => v === 'open')) return 'partly live'
  return s.includes('reconnecting') ? 'reconnecting' : 'connecting'
})
const linkTone = computed(() => (link.value === 'live' ? 'live' : link.value === 'reconnecting' ? 'reconnecting' : 'unknown'))
</script>

<template>
  <section :aria-label="title ?? 'Activity'" class="panel flex min-h-0 flex-col px-4 py-3">
    <header class="flex items-baseline justify-between">
      <h2 class="font-bold">{{ title ?? 'Activity' }}</h2>
      <span class="text-muted"><StatusWord :status="linkTone" class="[&>span:last-child]:sr-only" /> {{ link }}</span>
    </header>
    <p class="pt-1 pb-2 text-faint">Tool calls and delegations as they happen, one poll interval behind. Previews are cut at 300 characters.</p>
    <ul v-if="watched.length" role="list" aria-label="Sessions being watched" class="mb-2 flex flex-wrap gap-x-4 gap-y-1 border-y border-line py-2 text-muted">
      <li v-for="s in watched" :key="s.profile + s.id" class="flex items-baseline gap-1.5">
        <span class="lamp !h-2 !w-2" :class="s.open ? 'bg-ok' : 'bg-lamp-off'" aria-hidden="true" />
        <RouterLink :to="{ name: 'session', params: { profile: s.profile, id: s.id } }" class="no-underline hover:text-amber">
          <span v-if="profiles.length > 1" class="text-ink">{{ s.profile }}</span>
          {{ s.title || shortId(s.id) }}
        </RouterLink>
        <span class="text-faint tabular">{{ relativeTime(s.last_active, now) }}</span>
      </li>
    </ul>
    <EmptyState v-if="rows.length === 0" title="Nothing yet" detail="Events appear here when an agent runs a tool, replies, or delegates." />
    <ul v-else role="list" aria-live="polite" aria-relevant="additions" class="-ml-1 overflow-y-auto">
      <FeedRow v-for="row in rows" :key="row.key" :row="row" :show-profile="profiles.length > 1" :fresh="fresh.has(row.key)" :now="now" />
    </ul>
  </section>
</template>
