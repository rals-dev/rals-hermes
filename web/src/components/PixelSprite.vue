<script setup lang="ts">
// Renders one worker-state sprite from src/lib/sprites.ts onto a <canvas>,
// stepping through its frames at a fixed rate. Swapping sprites.ts for a
// real sprite-sheet later only needs framesFor() to keep returning
// string[][] grids of the same palette shape — nothing here would change.
import { onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { framesFor, palette, spriteCols, spriteRows, type Pose } from '@/lib/sprites'

const props = withDefaults(defineProps<{ pose: Pose; shirtColor: string; scale?: number }>(), { scale: 3 })

const canvas = useTemplateRef('canvas')
const frameIndex = ref(0)
const FPS = 5

let timer: ReturnType<typeof setInterval> | undefined
function reducedMotion(): boolean {
  return typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
}

function restartAnimation() {
  if (timer) clearInterval(timer)
  frameIndex.value = 0
  const frames = framesFor(props.pose)
  if (frames.length <= 1 || reducedMotion()) return
  timer = setInterval(() => {
    frameIndex.value = (frameIndex.value + 1) % frames.length
  }, 1000 / FPS)
}

function draw() {
  const el = canvas.value
  const ctx = el?.getContext('2d')
  if (!ctx) return
  const frames = framesFor(props.pose)
  const grid = frames[frameIndex.value % frames.length]!
  ctx.imageSmoothingEnabled = false
  ctx.clearRect(0, 0, el!.width, el!.height)
  for (let r = 0; r < grid.length; r++) {
    const row = grid[r]!
    for (let c = 0; c < row.length; c++) {
      const ch = row[c]
      if (!ch || ch === '.') continue
      ctx.fillStyle = ch === 'S' ? props.shirtColor : (palette[ch] ?? '#000')
      ctx.fillRect(c * props.scale, r * props.scale, props.scale, props.scale)
    }
  }
}

watch(() => props.pose, restartAnimation, { immediate: true })
watch([frameIndex, () => props.shirtColor, () => props.scale], draw, { immediate: true, flush: 'post' })
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>

<template>
  <canvas
    ref="canvas"
    :width="spriteCols * scale"
    :height="spriteRows * scale"
    class="[image-rendering:pixelated]"
    role="img"
    :aria-label="`worker ${pose}`"
  />
</template>
