<script setup lang="ts">
// Renders one agent (docs/decisions/024) onto a <canvas> for the grid view,
// cycling the pose's frames. Frames come from the shared cache in
// src/lib/agents/compose.ts, so each is painted once and blitted here.
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import { SPRITE_W, agentFrame, frameAt, frameCount, groundRow, type Pose } from '@/lib/agents/compose'
import { personaFor } from '@/lib/agents/personas'

const props = withDefaults(defineProps<{ profile: string; pose: Pose; scale?: number }>(), { scale: 3 })

const canvas = useTemplateRef('canvas')
const persona = computed(() => personaFor(props.profile))
// Seated poses stop at the waist; the canvas is cropped to what's drawn.
const rows = computed(() => groundRow(props.pose))
const frame = ref(0)

let timer: ReturnType<typeof setInterval> | undefined
function reducedMotion(): boolean {
  return typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
}

function restartAnimation() {
  if (timer) clearInterval(timer)
  timer = undefined
  frame.value = 0
  if (frameCount(props.pose) <= 1 || reducedMotion()) return
  timer = setInterval(() => {
    frame.value = frameAt(props.pose, performance.now())
  }, 100)
}

function draw() {
  const el = canvas.value
  const ctx = el?.getContext('2d')
  if (!el || !ctx) return
  ctx.imageSmoothingEnabled = false
  ctx.clearRect(0, 0, el.width, el.height)
  const image = agentFrame(persona.value, props.pose, frame.value, false)
  ctx.drawImage(image, 0, 0, SPRITE_W, rows.value, 0, 0, SPRITE_W * props.scale, rows.value * props.scale)
}

watch(() => props.pose, restartAnimation, { immediate: true })
// `onMounted` guarantees the first paint: a plain immediate watcher can fire
// before the <canvas> ref is bound, and a single-frame pose has no timer to
// retry it (fixed in ead4ee4). Pose is tracked here too, so a pose change
// that lands on the same frame number still repaints.
watch([() => props.pose, frame, persona, () => props.scale], draw, { flush: 'post' })
onMounted(draw)
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <canvas
    ref="canvas"
    :width="SPRITE_W * scale"
    :height="rows * scale"
    class="[image-rendering:pixelated]"
    role="img"
    :aria-label="`${profile} ${pose}`"
  />
</template>
