<script setup lang="ts">
defineProps<{ title: string; detail?: string }>()
</script>

<template>
  <div role="status" class="rounded border border-dashed border-line-strong px-4 py-8 text-center">
    <p class="font-bold">{{ title }}</p>
    <p v-if="detail" class="mt-1 text-muted">{{ detail }}</p>
  </div>
</template>
