<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { api } from '@/api/client'
import type { JobsResponse } from '@/api/types'
import StatusWord from '@/components/StatusWord.vue'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import Skeleton from '@/components/Skeleton.vue'
import { relativeTime } from '@/lib/format'

const q = useQuery({
  queryKey: ['jobs'],
  queryFn: () => api.get<JobsResponse>('/api/jobs'),
  refetchInterval: 30_000,
})
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-lg font-semibold tracking-tight">Scheduled jobs</h1>
    <Skeleton v-if="q.isPending.value" :rows="3" label="Loading jobs" />
    <ErrorState v-else-if="q.isError.value" :error="q.error.value" :retry="() => q.refetch()" />
    <template v-else-if="q.data.value">
      <ul v-if="q.data.value.errors.length" role="list" class="space-y-1 text-warn">
        <li v-for="e in q.data.value.errors" :key="e.profile">{{ e.profile }}: {{ e.error.message }}</li>
      </ul>
      <EmptyState v-if="q.data.value.jobs.length === 0" title="No scheduled jobs" detail="Jobs created with `hermes cron` on any profile show up here." />
      <div v-else class="overflow-x-auto">
        <table class="w-full border-collapse text-left">
          <thead class="text-muted">
            <tr class="border-b border-line">
              <th scope="col" class="py-1.5 pr-3 font-normal">Job</th>
              <th scope="col" class="py-1.5 pr-3 font-normal">Profile</th>
              <th scope="col" class="py-1.5 pr-3 font-normal">Schedule</th>
              <th scope="col" class="py-1.5 pr-3 font-normal">State</th>
              <th scope="col" class="py-1.5 pr-3 font-normal">Last run</th>
              <th scope="col" class="py-1.5 font-normal">Next run</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="j in q.data.value.jobs" :key="j.profile + j.id" class="border-b border-line last:border-b-0">
              <td class="py-2 pr-3">
                <div class="font-medium">{{ j.name }}</div>
                <div v-if="j.last_error" class="text-bad">{{ j.last_error }}</div>
              </td>
              <td class="py-2 pr-3 text-muted">{{ j.profile }}</td>
              <td class="py-2 pr-3 font-mono text-[12px] text-muted">{{ j.schedule.display || j.schedule.expr }}</td>
              <td class="py-2 pr-3"><StatusWord :status="j.enabled ? j.state : 'paused'" /><span v-if="j.failure_streak" class="ml-2 text-bad">{{ j.failure_streak }} failures</span></td>
              <td class="py-2 pr-3 text-muted">{{ relativeTime(j.last_run_at) }}<span v-if="j.last_status"> · {{ j.last_status }}</span></td>
              <td class="py-2 text-muted">{{ j.next_run_at ? new Date(j.next_run_at).toLocaleString() : '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>
