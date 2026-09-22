<script setup lang="ts">
// Token and cost totals per profile over a trailing window, computed by the
// BFF on the fly from GET /api/sessions — no persisted state (ADR-007). See
// GET /api/usage. Degrades quietly on its own: a slow or failing profile
// (or the whole endpoint) never blocks the rest of Overview.
import { useQuery } from '@tanstack/vue-query'
import { api } from '@/api/client'
import type { UsageResponse } from '@/api/types'
import { costLabel, relativeTime, tokensLabel } from '@/lib/format'

const usage = useQuery({
  queryKey: ['usage'],
  queryFn: () => api.get<UsageResponse>('/api/usage'),
  refetchInterval: 60_000,
})
</script>

<template>
  <section v-if="usage.data.value" aria-label="Usage" class="panel px-4 py-3">
    <header class="flex items-baseline justify-between pb-2">
      <h2 class="font-bold">Usage — last {{ usage.data.value.window_hours }}h</h2>
      <span class="text-faint">{{ relativeTime(usage.data.value.generated_at) }}</span>
    </header>
    <div class="overflow-x-auto">
      <table class="w-full border-collapse text-left">
        <thead class="text-muted">
          <tr class="border-b border-line">
            <th scope="col" class="py-1.5 pr-3 font-normal">Profile</th>
            <th scope="col" class="py-1.5 pr-3 font-normal tabular">Sessions</th>
            <th scope="col" class="py-1.5 pr-3 font-normal">Tokens</th>
            <th scope="col" class="py-1.5 font-normal">Cost</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in usage.data.value.profiles" :key="p.profile" class="border-b border-line last:border-b-0">
            <td class="py-2 pr-3 font-bold">
              {{ p.profile }}
              <span v-if="p.truncated" class="text-faint" title="Hit the pagination safety cap before the window ended — this total is a lower bound.">*</span>
            </td>
            <td class="py-2 pr-3 tabular">{{ p.session_count }}</td>
            <td class="py-2 pr-3 whitespace-nowrap text-muted tabular">{{ tokensLabel(p.usage) }}</td>
            <td class="py-2 text-muted tabular">{{ costLabel(p.estimated_cost_usd, p.actual_cost_usd) ?? '—' }}</td>
          </tr>
          <tr v-for="e in usage.data.value.errors" :key="e.profile" class="border-b border-line text-faint last:border-b-0">
            <td class="py-2 pr-3 font-bold">{{ e.profile }}</td>
            <td class="py-2" colspan="3">unavailable</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
