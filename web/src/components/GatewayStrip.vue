<script setup lang="ts">
import type { Gateway } from '@/api/types'
import StatusWord from './StatusWord.vue'
import { relativeTime } from '@/lib/format'

// The gateway is one process serving every profile (T-003 § 2), so its
// numbers are shown once, as a single line, not repeated per agent.
defineProps<{ gateway: Gateway | null; generatedAt: string }>()
</script>

<template>
  <section aria-label="Gateway" class="flex flex-wrap items-baseline gap-x-6 gap-y-1 border-b border-line pb-3 text-muted">
    <template v-if="gateway">
      <span class="text-ink">Gateway <StatusWord :status="gateway.state" /></span>
      <span>Hermes {{ gateway.version }}</span>
      <span>readiness <span class="text-ink">{{ gateway.readiness }}</span></span>
      <span>
        <span class="text-ink tabular">{{ gateway.active_agents }}</span> active agent{{ gateway.active_agents === 1 ? '' : 's' }}
      </span>
      <span><span class="text-ink tabular">{{ gateway.active_api_runs }}</span> API runs</span>
      <span><span class="text-ink tabular">{{ gateway.active_delegations }}</span> delegations</span>
      <span>disk <span class="text-ink tabular">{{ gateway.disk.used_percent.toFixed(0) }}%</span> used</span>
      <span class="ml-auto text-faint">via {{ gateway.source_profile }} · {{ relativeTime(generatedAt) }}</span>
    </template>
    <template v-else>
      <span class="text-ink">Gateway <StatusWord status="unreachable" /></span>
      <span>No profile answered; the Hermes container is probably down.</span>
    </template>
  </section>
</template>
