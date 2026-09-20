<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { FeedRow } from '@/lib/feed'
import { relativeTime } from '@/lib/format'

const props = defineProps<{ row: FeedRow; showProfile: boolean; fresh: boolean; now: Date }>()

const duration = computed(() => {
  const ms = props.row.durationMs
  if (ms === undefined) return ''
  return ms < 1000 ? `${ms} ms` : `${(ms / 1000).toFixed(1)} s`
})

const state = computed(() => {
  if (props.row.kind !== 'tool') return ''
  if (!props.row.done) return 'running'
  return props.row.failed ? 'failed' : 'done'
})
</script>

<template>
  <li class="grid grid-cols-[6.5rem_minmax(0,1fr)] gap-x-3 border-b border-line py-2 last:border-b-0" :class="{ 'feed-new': fresh }">
    <div class="text-faint tabular">
      <div>{{ relativeTime(row.at, now) }}</div>
      <div v-if="showProfile" class="truncate">{{ row.profile }}</div>
    </div>
    <div class="min-w-0">
      <div class="flex flex-wrap items-baseline gap-x-2">
        <span class="font-medium" :class="{ 'text-bad': row.kind === 'error' || row.failed }">{{ row.title }}</span>
        <span v-if="state" class="text-muted">{{ state }}<template v-if="duration"> · {{ duration }}</template></span>
        <RouterLink
          v-if="row.sessionId"
          :to="{ name: 'session', params: { profile: row.profile, id: row.sessionId } }"
          class="ml-auto text-faint no-underline hover:text-ink hover:underline"
        >session</RouterLink>
      </div>
      <pre v-if="row.detail" class="mt-0.5 max-h-24 overflow-hidden whitespace-pre-wrap break-all font-mono text-[12px] text-muted">{{ row.detail }}</pre>
      <pre v-if="row.result" class="mt-1 max-h-24 overflow-hidden whitespace-pre-wrap break-all border-l-2 border-line pl-2 font-mono text-[12px] text-muted">{{ row.result }}</pre>
    </div>
  </li>
</template>
