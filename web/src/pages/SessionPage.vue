<script setup lang="ts">
import { computed, ref } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { RouterLink } from 'vue-router'
import { api } from '@/api/client'
import type { SessionDetail } from '@/api/types'
import MessageTimeline from '@/components/MessageTimeline.vue'
import ErrorState from '@/components/ErrorState.vue'
import Skeleton from '@/components/Skeleton.vue'
import { costLabel, relativeTime, tokensLabel } from '@/lib/format'

const props = defineProps<{ profile: string; id: string }>()
const pageSize = 50
const offset = ref(0)

const q = useQuery({
  queryKey: computed(() => ['session', props.profile, props.id, offset.value]),
  queryFn: () => api.get<SessionDetail>(`/api/agents/${encodeURIComponent(props.profile)}/sessions/${encodeURIComponent(props.id)}?limit=${pageSize}&offset=${offset.value}`),
  placeholderData: (prev) => prev,
  refetchInterval: (query) => (query.state.data?.session.open ? 5_000 : false),
})

const s = computed(() => q.data.value?.session)
const total = computed(() => s.value?.message_count ?? 0)
</script>

<template>
  <div class="space-y-5">
    <nav aria-label="Breadcrumb" class="text-muted">
      <RouterLink to="/" class="no-underline hover:text-amber hover:underline">Overview</RouterLink> /
      <RouterLink :to="{ name: 'agent', params: { profile } }" class="no-underline hover:text-amber hover:underline">{{ profile }}</RouterLink>
    </nav>

    <Skeleton v-if="q.isPending.value" :rows="4" label="Loading session" />
    <ErrorState v-else-if="q.isError.value" :error="q.error.value" :retry="() => q.refetch()" />
    <template v-else-if="q.data.value && s">
      <header>
        <h1 class="text-lg font-bold tracking-tight">{{ s.title || 'Untitled session' }}</h1>
        <p class="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-muted">
          <span class="font-mono text-[12px]">{{ s.id }}</span>
          <span>{{ s.source }}</span>
          <span>{{ s.model }}</span>
          <span v-if="s.open" class="text-ok">open · last active {{ relativeTime(s.last_active) }}</span>
          <span v-else>ended {{ relativeTime(s.ended_at) }} · {{ s.end_reason }}</span>
          <span v-if="s.parent_session_id">child of <RouterLink :to="{ name: 'session', params: { profile, id: s.parent_session_id } }" class="hover:text-amber hover:underline">{{ s.parent_session_id }}</RouterLink></span>
        </p>
        <dl class="mt-3 grid grid-cols-2 gap-x-6 gap-y-1 sm:grid-cols-4">
          <div><dt class="text-faint">Messages</dt><dd class="tabular">{{ s.message_count }}</dd></div>
          <div><dt class="text-faint">Tool calls</dt><dd class="tabular">{{ s.tool_call_count }}</dd></div>
          <div><dt class="text-faint">Tokens</dt><dd>{{ tokensLabel(s.usage) }}</dd></div>
          <div><dt class="text-faint">Cost</dt><dd class="tabular">{{ costLabel(s.estimated_cost_usd, s.actual_cost_usd) ?? 'not reported' }}</dd></div>
        </dl>
      </header>

      <section aria-label="Messages">
        <header class="flex items-baseline justify-between pb-2">
          <h2 class="font-bold">Messages</h2>
          <span class="text-faint">Previews are what Hermes stores; secrets are redacted upstream.</span>
        </header>
        <MessageTimeline :messages="q.data.value.messages.data" />
        <nav aria-label="Pagination" class="mt-3 flex items-center gap-4 text-muted">
          <button type="button" class="hover:text-ink disabled:opacity-40" :disabled="offset === 0" @click="offset = Math.max(0, offset - pageSize)">Earlier</button>
          <span class="tabular">{{ offset + 1 }}–{{ Math.min(offset + pageSize, total) }} of {{ total }}</span>
          <button type="button" class="hover:text-ink disabled:opacity-40" :disabled="offset + pageSize >= total" @click="offset += pageSize">Later</button>
          <button type="button" class="ml-auto hover:text-ink" @click="offset = Math.max(0, Math.floor((total - 1) / pageSize) * pageSize)">Jump to latest</button>
        </nav>
      </section>
    </template>
  </div>
</template>
