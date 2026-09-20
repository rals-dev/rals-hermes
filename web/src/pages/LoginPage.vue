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
    <h1 class="text-xl font-semibold tracking-tight">Hermes agents</h1>
    <p class="mt-1 text-muted">Enter the dashboard key to open a session. It stays in a cookie on this browser, never in the page.</p>
    <form class="mt-6 space-y-3" @submit.prevent="submit">
      <label class="block">
        <span class="text-muted">Dashboard key</span>
        <input
          v-model="key"
          type="password"
          name="key"
          autocomplete="current-password"
          required
          autofocus
          class="mt-1 w-full rounded border border-line bg-surface px-3 py-2 text-ink"
        />
      </label>
      <p v-if="error" role="alert" class="text-bad">{{ error }}</p>
      <button type="submit" :disabled="busy || !key" class="w-full rounded bg-accent px-3 py-2 font-medium text-white disabled:opacity-60">
        {{ busy ? 'Opening…' : 'Open dashboard' }}
      </button>
    </form>
  </div>
</template>
