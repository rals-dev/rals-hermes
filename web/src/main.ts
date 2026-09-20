import { createApp } from 'vue'
import { VueQueryPlugin, QueryClient } from '@tanstack/vue-query'
import App from './App.vue'
import { router } from './router'
import './style.css'

// Polling floor: never below 5 s (PRD § 9). The BFF caches for 3 s on top.
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 3_000,
      refetchInterval: 5_000,
      refetchOnWindowFocus: true,
      retry: 1,
    },
  },
})

createApp(App).use(router).use(VueQueryPlugin, { queryClient }).mount('#app')
