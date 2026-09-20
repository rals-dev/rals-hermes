<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { AgentCard } from '@/api/types'
import StatusWord from './StatusWord.vue'

defineProps<{ agent: AgentCard }>()

function platformSummary(a: AgentCard): string {
  if (!a.platforms) return ''
  return Object.entries(a.platforms)
    .map(([name, p]) => `${name} ${p.state}`)
    .join(', ')
}
</script>

<template>
  <li>
    <RouterLink
      :to="{ name: 'agent', params: { profile: agent.profile } }"
      class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-0.5 px-4 py-3 no-underline transition-colors hover:bg-panel-2"
    >
      <span class="truncate font-bold">{{ agent.profile }}</span>
      <StatusWord :status="agent.status" large class="justify-self-end" />
      <span class="truncate text-muted">
        <template v-if="agent.error">{{ agent.error.message }}</template>
        <template v-else>{{ platformSummary(agent) || 'no messaging platform' }}</template>
      </span>
      <span class="justify-self-end text-faint tabular">{{ agent.latency_ms }} ms</span>
    </RouterLink>
  </li>
</template>
