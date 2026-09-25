<script setup lang="ts">
// Draws the cross-profile delegation lines over the floor grid, positioned
// with plain DOM measurement (FloorPage.vue) rather than a layout library.
// Same-profile delegation and unknown-parent delegation are shown as
// badges on WorkerStation instead — this only ever renders links between
// two distinct, currently-positioned stations.
import { computed } from 'vue'
import type { DelegationLink } from '@/lib/floor'

const props = defineProps<{
  links: DelegationLink[]
  positions: Record<string, { x: number; y: number }>
  width: number
  height: number
}>()

const visible = computed(() =>
  props.links
    .map((l) => ({ link: l, from: props.positions[l.parentProfile], to: props.positions[l.childProfile] }))
    .filter((v): v is { link: DelegationLink; from: { x: number; y: number }; to: { x: number; y: number } } => !!v.from && !!v.to),
)
</script>

<template>
  <svg
    v-if="width > 0 && height > 0"
    :viewBox="`0 0 ${width} ${height}`"
    class="pointer-events-none absolute inset-0"
    aria-hidden="true"
  >
    <line
      v-for="v in visible"
      :key="v.link.parentSessionId + v.link.childSessionId"
      :x1="v.from.x" :y1="v.from.y" :x2="v.to.x" :y2="v.to.y"
      class="delegation-line"
    />
  </svg>
</template>

<style scoped>
.delegation-line {
  stroke: var(--amber);
  stroke-width: 2;
  stroke-dasharray: 7 5;
  fill: none;
  animation: march 0.9s linear infinite;
}
@keyframes march {
  to { stroke-dashoffset: -24; }
}
@media (prefers-reduced-motion: reduce) {
  .delegation-line { animation: none; }
}
</style>
