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
  <li class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-1 py-3 sm:grid-cols-[9rem_7rem_minmax(0,1fr)_4rem]">
    <RouterLink :to="{ name: 'agent', params: { profile: agent.profile } }" class="truncate font-medium no-underline hover:underline">
      {{ agent.profile }}
    </RouterLink>
    <StatusWord :status="agent.status" class="sm:order-none" />
    <span class="col-span-2 truncate text-muted sm:col-span-1">
      <template v-if="agent.error">{{ agent.error.message }}</template>
      <template v-else>{{ platformSummary(agent) || 'no messaging platform' }}</template>
    </span>
    <span class="hidden text-right text-faint tabular sm:inline">{{ agent.latency_ms }} ms</span>
  </li>
</template>
