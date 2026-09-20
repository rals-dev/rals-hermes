<script setup lang="ts">
import { computed, ref } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { api } from '@/api/client'
import type { AgentDetail, SessionList } from '@/api/types'
import StatusWord from '@/components/StatusWord.vue'
import SessionTable from '@/components/SessionTable.vue'
import ActivityFeed from '@/components/ActivityFeed.vue'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import Skeleton from '@/components/Skeleton.vue'

const props = defineProps<{ profile: string }>()
const pageSize = 20
const offset = ref(0)

const detail = useQuery({
  queryKey: computed(() => ['agent', props.profile]),
  queryFn: () => api.get<AgentDetail>(`/api/agents/${encodeURIComponent(props.profile)}`),
  refetchInterval: 15_000,
})

const sessions = useQuery({
  queryKey: computed(() => ['sessions', props.profile, offset.value]),
  queryFn: () => api.get<SessionList>(`/api/agents/${encodeURIComponent(props.profile)}/sessions?limit=${pageSize}&offset=${offset.value}`),
  placeholderData: (prev) => prev,
})

const profiles = computed(() => [props.profile])
const enabledToolsets = computed(() => detail.data.value?.toolsets.filter((t) => t.enabled) ?? [])
</script>

<template>
  <div class="space-y-6">
    <header class="flex flex-wrap items-baseline gap-x-4 gap-y-1">
      <h1 class="text-lg font-semibold tracking-tight">{{ profile }}</h1>
      <template v-if="detail.data.value">
        <StatusWord :status="detail.data.value.status" />
        <span v-if="detail.data.value.model" class="text-muted">model {{ detail.data.value.model }}</span>
        <span v-if="detail.data.value.latency_ms >= 0" class="text-faint tabular">{{ detail.data.value.latency_ms }} ms</span>
        <span v-for="w in detail.data.value.warnings" :key="w" class="text-warn">{{ w.replace('_', ' ') }}</span>
      </template>
    </header>

    <ErrorState v-if="detail.isError.value" :error="detail.error.value" :retry="() => detail.refetch()" />

    <div class="grid gap-8 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)]">
      <div class="space-y-6">
        <section aria-label="Sessions">
          <header class="flex items-baseline justify-between pb-2">
            <h2 class="font-medium">Sessions</h2>
            <span class="text-faint">newest first</span>
          </header>
          <Skeleton v-if="sessions.isPending.value" :rows="5" label="Loading sessions" />
          <ErrorState v-else-if="sessions.isError.value" :error="sessions.error.value" :retry="() => sessions.refetch()" />
          <template v-else-if="sessions.data.value">
            <EmptyState v-if="sessions.data.value.data.length === 0" title="No sessions" detail="This profile has not handled any conversation or task yet." />
            <SessionTable v-else :profile="profile" :sessions="sessions.data.value.data" />
            <nav aria-label="Pagination" class="mt-3 flex items-center gap-4 text-muted">
              <button type="button" class="hover:text-ink disabled:opacity-40" :disabled="offset === 0" @click="offset = Math.max(0, offset - pageSize)">Newer</button>
              <span class="tabular">{{ offset + 1 }}–{{ offset + sessions.data.value.data.length }}</span>
              <button type="button" class="hover:text-ink disabled:opacity-40" :disabled="!sessions.data.value.has_more" @click="offset += pageSize">Older</button>
            </nav>
          </template>
        </section>

        <section v-if="detail.data.value" aria-label="Toolsets">
          <h2 class="pb-2 font-medium">Toolsets</h2>
          <ul role="list" class="grid gap-x-6 gap-y-1 sm:grid-cols-2">
            <li v-for="t in enabledToolsets" :key="t.name" class="flex justify-between gap-3 border-b border-line py-1">
              <span>{{ t.label }}</span>
              <span class="text-faint tabular">{{ t.tools.length }} tools</span>
            </li>
          </ul>
        </section>
      </div>

      <ActivityFeed :profiles="profiles" title="Live activity" class="lg:max-h-[calc(100dvh-12rem)]" />
    </div>
  </div>
</template>
