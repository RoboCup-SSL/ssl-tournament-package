// Hash-history router (no server-side SPA fallback needed). The tournament
// workspace chrome lives in App.vue; section views render here directly.
import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '@/views/HomeView.vue'
import SettingsView from '@/views/tournament/SettingsView.vue'

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/t/:id/settings', name: 'settings', component: SettingsView },
    { path: '/t/:id', redirect: (to) => ({ name: 'settings', params: { id: to.params.id } }) },
  ],
})

export default router
