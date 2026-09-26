<script setup lang="ts">
import { computed } from 'vue'
import type { AgentStatus } from '@/api/types'
import { lampTone } from '@/lib/status'

// A lamp beside the word. The word carries the meaning; the lamp reinforces.
const props = defineProps<{ status: AgentStatus | string; large?: boolean }>()

const tone = computed(() => lampTone(props.status))
</script>

<template>
  <span class="inline-flex items-center gap-1.5">
    <span class="lamp" :class="[tone, { 'lamp-lg': large }]" aria-hidden="true" />
    <span>{{ status }}</span>
  </span>
</template>
