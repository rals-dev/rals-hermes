<script setup lang="ts">
import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { api } from '@/api/client'
import type { Overview } from '@/api/types'
import GatewayStrip from '@/components/GatewayStrip.vue'
import AgentRow from '@/components/AgentRow.vue'
import ActivityFeed from '@/components/ActivityFeed.vue'
import ErrorState from '@/components/ErrorState.vue'
import Skeleton from '@/components/Skeleton.vue'

const overview = useQuery({
  queryKey: ['overview'],
  queryFn: () => api.get<Overview>('/api/overview'),
})

const profiles = computed(() => overview.data.value?.agents.map((a) => a.profile) ?? [])
const busy = computed(() => (overview.data.value?.gateway?.active_agents ?? 0) > 0)
</script>

<template>
  <div class="space-y-5">
    <Skeleton v-if="overview.isPending.value" :rows="2" label="Loading overview" />
    <ErrorState v-else-if="overview.isError.value" :error="overview.error.value" :retry="() => overview.refetch()" />
    <template v-else-if="overview.data.value">
      <GatewayStrip :gateway="overview.data.value.gateway" :generated-at="overview.data.value.generated_at" />

      <div class="grid gap-8 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)]">
        <section aria-label="Agents">
          <header class="flex items-baseline justify-between pb-1">
            <h1 class="text-lg font-semibold tracking-tight">
              {{ busy ? 'An agent is working' : 'All agents idle' }}
            </h1>
            <span class="text-faint">{{ overview.data.value.agents.length }} profiles</span>
          </header>
          <ul role="list" class="divide-y divide-line border-y border-line">
            <AgentRow v-for="a in overview.data.value.agents" :key="a.profile" :agent="a" />
          </ul>
          <p class="mt-2 text-faint">
            Reachability and key acceptance are per profile; the counters above are gateway-wide.
          </p>
        </section>

        <ActivityFeed :profiles="profiles" title="Live activity" class="lg:max-h-[calc(100dvh-12rem)]" />
      </div>
    </template>
  </div>
</template>
