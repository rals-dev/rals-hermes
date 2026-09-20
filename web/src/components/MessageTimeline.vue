<script setup lang="ts">
import type { Message } from '@/api/types'
import { relativeTime } from '@/lib/format'
import Clamp from './Clamp.vue'

defineProps<{ messages: Message[] }>()

function label(m: Message): string {
  if (m.role === 'tool') return `${m.tool_name || 'tool'} result`
  if (m.role === 'assistant' && m.tool_calls?.length) return 'assistant → ' + m.tool_calls.map((c) => c.function.name).join(', ')
  return m.role
}
</script>

<template>
  <ol role="list" class="divide-y divide-line border-y border-line">
    <li v-for="m in messages" :key="m.id" class="grid grid-cols-[6.5rem_minmax(0,1fr)] gap-x-3 py-2" :class="{ 'opacity-60': m.role === 'session_meta' }">
      <div class="text-faint tabular">
        <div>{{ relativeTime(m.timestamp) }}</div>
        <div>#{{ m.id }}</div>
      </div>
      <div class="min-w-0">
        <div class="font-bold" :class="{ 'text-muted': m.role === 'user' }">{{ label(m) }}</div>
        <template v-if="m.tool_calls?.length">
          <Clamp v-for="c in m.tool_calls" :key="c.id" :text="c.function.arguments" mono />
        </template>
        <Clamp v-if="m.content" :text="m.content" :mono="m.role === 'tool'" />
      </div>
    </li>
  </ol>
</template>
