<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, ApiError } from '@/api/client'

const route = useRoute()
const router = useRouter()
const key = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await api.login(key.value)
    key.value = ''
    const next = typeof route.query.next === 'string' && route.query.next.startsWith('/') ? route.query.next : '/'
    await router.replace(next)
  } catch (e) {
    if (e instanceof ApiError && e.code === 'too_many_attempts') error.value = 'Too many failed attempts. Wait a minute, then try again.'
    else if (e instanceof ApiError && e.status === 401) error.value = 'That key was not accepted.'
    else error.value = 'The dashboard could not reach its backend.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="mx-auto mt-[12vh] max-w-sm">
    <div class="panel px-5 py-5">
      <h1 class="flex items-center gap-2 text-lg font-bold tracking-tight">
        <span class="lamp bg-amber" aria-hidden="true" />
        Hermes agents
      </h1>
      <p class="mt-1 text-muted">Enter the dashboard key to open a session. It stays in a cookie on this browser, never in the page.</p>
      <form class="mt-5 space-y-3" @submit.prevent="submit">
        <label class="block">
          <span class="text-muted">Dashboard key</span>
          <input
            v-model="key"
            type="password"
            name="key"
            autocomplete="current-password"
            required
            autofocus
            class="mt-1 w-full rounded border border-line-strong bg-bg px-3 py-2 font-mono text-ink"
          />
        </label>
        <p v-if="error" role="alert" class="text-bad">{{ error }}</p>
        <button type="submit" :disabled="busy || !key" class="w-full rounded bg-amber px-3 py-2 font-bold text-amber-ink transition-opacity disabled:opacity-50">
          {{ busy ? 'Opening…' : 'Open dashboard' }}
        </button>
      </form>
    </div>
  </div>
</template>
