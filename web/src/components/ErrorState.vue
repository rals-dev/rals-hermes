<script setup lang="ts">
import { ApiError } from '@/api/client'

const props = defineProps<{ error: unknown; retry?: () => void }>()

function describe(e: unknown): { title: string; detail: string } {
  if (e instanceof ApiError) {
    switch (e.code) {
      case 'upstream_unreachable': return { title: 'Hermes did not answer', detail: 'The BFF is up but the agent gateway timed out or refused the connection.' }
      case 'upstream_unauthorized': return { title: 'Hermes rejected the profile key', detail: 'Check API_SERVER_KEY for this profile on the host.' }
      case 'profile_not_found': return { title: 'No such profile', detail: 'It is not in the BFF configuration.' }
      case 'run_not_found': return { title: 'No such run on this profile', detail: 'Runs belong to the profile that created them.' }
      case 'session_not_found': return { title: 'No such session', detail: 'It may have been deleted or belongs to another profile.' }
      default: return { title: e.message || e.code, detail: `${e.status} ${e.code}` }
    }
  }
  return { title: 'Something went wrong', detail: e instanceof Error ? e.message : String(e) }
}
const d = describe(props.error)
</script>

<template>
  <div role="alert" class="panel border-l-4 !border-l-bad px-4 py-3">
    <p class="font-bold">{{ d.title }}</p>
    <p class="text-muted">{{ d.detail }}</p>
    <button v-if="retry" type="button" class="mt-2 text-amber hover:text-amber hover:underline" @click="retry">Try again</button>
  </div>
</template>
