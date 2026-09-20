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

const navClass = (name: string) =>
  route.name === name
    ? 'text-ink border-amber'
    : 'text-muted border-transparent hover:text-ink'
</script>

<template>
  <div class="min-h-dvh flex flex-col">
    <header v-if="!isLogin" class="border-b border-line bg-panel">
      <div class="mx-auto flex max-w-7xl items-stretch gap-6 px-4">
        <RouterLink to="/" class="flex items-center gap-2 py-2.5 font-bold tracking-tight no-underline">
          <span class="lamp bg-amber" aria-hidden="true" />
          <span class="whitespace-nowrap">Hermes agents</span>
        </RouterLink>
        <nav aria-label="Primary" class="flex items-stretch gap-5">
          <RouterLink to="/" class="flex items-center border-b-2 no-underline transition-colors" :class="navClass('overview')">Overview</RouterLink>
          <RouterLink to="/jobs" class="flex items-center border-b-2 no-underline transition-colors" :class="navClass('jobs')">Scheduled jobs</RouterLink>
        </nav>
        <button type="button" class="ml-auto text-muted hover:text-ink" @click="logout">Sign out</button>
      </div>
    </header>
    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-5">
      <RouterView />
    </main>
  </div>
</template>
