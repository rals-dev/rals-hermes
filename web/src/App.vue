<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'

const route = useRoute()
const router = useRouter()
const isLogin = computed(() => route.name === 'login')

async function logout() {
  try {
    await api.logout()
  } finally {
    void router.replace({ name: 'login' })
  }
}
</script>

<template>
  <div class="min-h-dvh flex flex-col">
    <header v-if="!isLogin" class="border-b border-line bg-surface">
      <div class="mx-auto flex max-w-7xl items-center gap-6 px-4 py-2.5">
        <RouterLink to="/" class="whitespace-nowrap font-semibold tracking-tight no-underline">Hermes agents</RouterLink>
        <nav aria-label="Primary" class="flex items-center gap-4 text-muted">
          <RouterLink to="/" class="no-underline hover:text-ink" :class="{ 'text-ink': route.name === 'overview' }">Overview</RouterLink>
          <RouterLink to="/jobs" class="no-underline hover:text-ink" :class="{ 'text-ink': route.name === 'jobs' }">Scheduled jobs</RouterLink>
        </nav>
        <button type="button" class="ml-auto text-muted hover:text-ink" @click="logout">Sign out</button>
      </div>
    </header>
    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-5">
      <RouterView />
    </main>
  </div>
</template>
