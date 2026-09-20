import { createRouter, createWebHistory } from 'vue-router'
import { onUnauthorized } from '@/api/client'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'overview', component: () => import('@/pages/OverviewPage.vue') },
    { path: '/floor', name: 'floor', component: () => import('@/pages/FloorPage.vue') },
    { path: '/login', name: 'login', component: () => import('@/pages/LoginPage.vue'), meta: { public: true } },
    { path: '/agents/:profile', name: 'agent', component: () => import('@/pages/AgentPage.vue'), props: true },
    { path: '/agents/:profile/sessions/:id', name: 'session', component: () => import('@/pages/SessionPage.vue'), props: true },
    { path: '/jobs', name: 'jobs', component: () => import('@/pages/JobsPage.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

// Any 401 from the API sends the user to login, remembering where they were.
onUnauthorized(() => {
  const current = router.currentRoute.value
  if (current.name !== 'login') {
    void router.replace({ name: 'login', query: { next: current.fullPath } })
  }
})
