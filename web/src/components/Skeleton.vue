<script setup lang="ts">
defineProps<{ rows?: number; label?: string }>()
</script>

<template>
  <div class="space-y-2" aria-busy="true" :aria-label="label ?? 'Loading'">
    <div v-for="i in rows ?? 3" :key="i" class="h-9 rounded bg-panel-2 motion-safe:animate-pulse" />
  </div>
</template>
