<script setup lang="ts">
import { computed } from 'vue'
import type { AgentStatus } from '@/api/types'

// Status is spoken as a word beside a dot: colour reinforces, never carries.
const props = defineProps<{ status: AgentStatus | string }>()

const tone = computed(() => {
  switch (props.status) {
    case 'healthy': case 'ok': case 'connected': case 'running': case 'completed': return 'bg-ok'
    case 'degraded': case 'warn': case 'waiting_for_approval': case 'stopping': case 'blocked': return 'bg-warn'
    case 'unreachable': case 'failed': case 'error': case 'interrupted': return 'bg-bad'
    case 'unauthorized': return 'bg-auth'
    default: return 'bg-faint'
  }
})
</script>

<template>
  <span class="inline-flex items-center gap-1.5">
    <span class="inline-block size-2 rounded-full" :class="tone" aria-hidden="true" />
    <span>{{ status }}</span>
  </span>
</template>
