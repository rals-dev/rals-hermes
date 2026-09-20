<script setup lang="ts">
import type { Gateway } from '@/api/types'
import StatusWord from './StatusWord.vue'
import { relativeTime } from '@/lib/format'

// The gateway is one process serving every profile (T-003 § 2): its numbers
// are shown once, as instrument readouts, not repeated per agent.
defineProps<{ gateway: Gateway | null; generatedAt: string }>()
</script>

<template>
  <section aria-label="Gateway" class="panel px-4 py-3">
    <template v-if="gateway">
      <dl class="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-3 lg:grid-cols-6">
        <div class="readout">
          <dt>Gateway</dt>
          <dd class="!text-[1em] pt-0.5"><StatusWord :status="gateway.state" /></dd>
        </div>
        <div class="readout">
          <dt>Readiness</dt>
          <dd class="!text-[1em] pt-0.5"><StatusWord :status="gateway.readiness" /></dd>
        </div>
        <div class="readout"><dt>Active agents</dt><dd :class="gateway.active_agents > 0 ? 'text-amber' : ''">{{ gateway.active_agents }}</dd></div>
        <div class="readout"><dt>API runs</dt><dd>{{ gateway.active_api_runs }}</dd></div>
        <div class="readout"><dt>Delegations</dt><dd>{{ gateway.active_delegations }}</dd></div>
        <div class="readout"><dt>Disk used</dt><dd>{{ gateway.disk.used_percent.toFixed(0) }}<span class="text-[0.7em] text-muted">%</span></dd></div>
      </dl>
      <p class="mt-2 flex flex-wrap gap-x-4 text-faint">
        <span>Hermes {{ gateway.version }}</span>
        <span>reported via {{ gateway.source_profile }}</span>
        <span>{{ relativeTime(generatedAt) }}</span>
      </p>
    </template>
    <template v-else>
      <p class="flex flex-wrap items-baseline gap-x-4">
        <span class="font-bold">Gateway <StatusWord status="unreachable" /></span>
        <span class="text-muted">No profile answered; the Hermes container is probably down.</span>
      </p>
    </template>
  </section>
</template>
