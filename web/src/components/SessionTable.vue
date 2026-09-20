<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { Session } from '@/api/types'
import { costLabel, relativeTime, shortId, tokensLabel } from '@/lib/format'

defineProps<{ profile: string; sessions: Session[] }>()
</script>

<template>
  <div class="overflow-x-auto">
    <table class="w-full border-collapse text-left">
      <thead class="text-muted">
        <tr class="border-b border-line">
          <th scope="col" class="py-1.5 pr-3 font-normal">Session</th>
          <th scope="col" class="py-1.5 pr-3 font-normal">Source</th>
          <th scope="col" class="py-1.5 pr-3 font-normal">State</th>
          <th scope="col" class="py-1.5 pr-3 font-normal">Last active</th>
          <th scope="col" class="py-1.5 pr-3 font-normal tabular">Tools</th>
          <th scope="col" class="py-1.5 pr-3 font-normal">Tokens</th>
          <th scope="col" class="py-1.5 font-normal">Cost</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="s in sessions" :key="s.id" class="border-b border-line align-top last:border-b-0">
          <td class="py-2 pr-3">
            <RouterLink :to="{ name: 'session', params: { profile, id: s.id } }" class="no-underline hover:underline">
              <span class="font-medium">{{ s.title || shortId(s.id) }}</span>
            </RouterLink>
            <div v-if="s.title" class="text-faint tabular">{{ shortId(s.id) }}</div>
            <div v-if="s.parent_session_id" class="text-faint">child of {{ shortId(s.parent_session_id) }}</div>
          </td>
          <td class="py-2 pr-3 text-muted">{{ s.source }}</td>
          <td class="py-2 pr-3">
            <span v-if="s.open" class="text-ok">open</span>
            <span v-else class="text-muted">{{ s.end_reason ?? 'ended' }}</span>
          </td>
          <td class="py-2 pr-3 whitespace-nowrap text-muted tabular">{{ relativeTime(s.last_active) }}</td>
          <td class="py-2 pr-3 tabular">{{ s.tool_call_count }}</td>
          <td class="py-2 pr-3 whitespace-nowrap text-muted">{{ tokensLabel(s.usage) }}</td>
          <td class="py-2 text-muted tabular">{{ costLabel(s.estimated_cost_usd, s.actual_cost_usd) ?? '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
