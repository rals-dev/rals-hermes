<script setup lang="ts">
import { computed } from 'vue'
import type { AgentStatus } from '@/api/types'

// A lamp beside the word. The word carries the meaning; the lamp reinforces.
const props = defineProps<{ status: AgentStatus | string; large?: boolean }>()

const tone = computed(() => {
  switch (props.status) {
    case 'healthy': case 'ok': case 'connected': case 'running': case 'completed': case 'live': case 'open': case 'working': return 'bg-ok'
    case 'degraded': case 'warn': case 'waiting_for_approval': case 'stopping': case 'blocked': case 'reconnecting': case 'delegating': return 'bg-warn'
    case 'unreachable': case 'failed': case 'error': case 'interrupted': case 'offline': return 'bg-bad'
    case 'unauthorized': return 'bg-auth'
    default: return 'bg-lamp-off'
  }
})
</script>

<template>
  <span class="inline-flex items-center gap-1.5">
    <span class="lamp" :class="[tone, { 'lamp-lg': large }]" aria-hidden="true" />
    <span>{{ status }}</span>
  </span>
</template>
