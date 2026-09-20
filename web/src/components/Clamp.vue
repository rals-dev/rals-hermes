<script setup lang="ts">
import { computed, ref } from 'vue'

// Long tool output is folded by default; the user opens it deliberately.
const props = defineProps<{ text: string; limit?: number; mono?: boolean }>()
const open = ref(false)
const limit = computed(() => props.limit ?? 600)
const long = computed(() => props.text.length > limit.value)
const shown = computed(() => (open.value || !long.value ? props.text : props.text.slice(0, limit.value) + '…'))
</script>

<template>
  <div>
    <pre class="mt-0.5 whitespace-pre-wrap break-words text-[13px]" :class="{ 'font-mono text-[12px] text-muted': mono }">{{ shown }}</pre>
    <button v-if="long" type="button" class="mt-1 text-accent hover:underline" :aria-expanded="open" @click="open = !open">
      {{ open ? 'Show less' : `Show all ${text.length.toLocaleString()} characters` }}
    </button>
  </div>
</template>
