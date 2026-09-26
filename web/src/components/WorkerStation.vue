<script setup lang="ts">
// One workstation on the /floor grid: the agent's persona (docs/decisions/024),
// name, live status, and the two readouts decided in the floor-view grilling
// (open sessions, tool calls in the last 10 minutes). Badges cover the two
// delegation cases that don't draw a line — see src/lib/floor.ts.
import { RouterLink } from 'vue-router'
import type { Worker } from '@/lib/floor'
import { seatedPoseFor } from '@/lib/agents/compose'
import PixelSprite from './PixelSprite.vue'
import StatusWord from './StatusWord.vue'

defineProps<{ worker: Worker }>()
</script>

<template>
  <RouterLink
    :to="{ name: 'agent', params: { profile: worker.profile } }"
    class="station panel relative flex w-44 flex-col items-center gap-1 px-3 py-3 text-center no-underline transition-colors hover:bg-panel-2"
  >
    <span v-if="worker.delegatedBadge" class="absolute top-2 left-2 rounded-sm bg-panel-2 px-1.5 py-0.5 text-faint">delegated</span>
    <span v-if="worker.helperCount" class="absolute top-2 right-2 rounded-sm bg-amber px-1.5 py-0.5 font-bold text-amber-ink">+{{ worker.helperCount }}</span>

    <!-- Tall enough for a standing 16×24 figure (the offline chair) at 4×. -->
    <div class="flex h-[96px] items-end">
      <div class="relative">
        <PixelSprite :profile="worker.profile" :pose="seatedPoseFor(worker.state)" :scale="4" />
        <span
          v-if="worker.state === 'working' && worker.toolName"
          class="panel absolute -top-1 left-1/2 max-w-[9rem] -translate-x-1/2 -translate-y-full truncate px-1.5 py-0.5 text-faint"
        >
          {{ worker.toolName }}
        </span>
        <!-- The status word below already says "error"; the badge is for the eye. -->
        <span
          v-if="worker.state === 'error'"
          aria-hidden="true"
          class="absolute -top-1 left-1/2 grid h-5 w-5 -translate-x-1/2 -translate-y-full place-items-center rounded-full bg-bad font-bold text-bg"
        >!</span>
      </div>
    </div>

    <span class="truncate font-bold">{{ worker.profile }}</span>
    <StatusWord :status="worker.state" />

    <dl class="mt-1 grid w-full grid-cols-2 gap-2 border-t border-line pt-2">
      <div class="readout">
        <dt>Sessions</dt>
        <dd>{{ worker.openSessions }}</dd>
      </div>
      <div class="readout">
        <dt>Tools (10m)</dt>
        <dd>{{ worker.toolCallsRecent }}</dd>
      </div>
    </dl>
  </RouterLink>
</template>
