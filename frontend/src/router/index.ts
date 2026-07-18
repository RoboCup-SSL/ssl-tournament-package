// Hash-history router (no server-side SPA fallback needed). The tournament
// workspace chrome lives in App.vue; section views render here directly.
import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '@/views/HomeView.vue'
import SettingsView from '@/views/tournament/SettingsView.vue'
import FieldsView from '@/views/tournament/FieldsView.vue'
import TeamsView from '@/views/tournament/TeamsView.vue'
import ScheduleView from '@/views/tournament/ScheduleView.vue'

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/t/:id/settings', name: 'settings', component: SettingsView },
    { path: '/t/:id/fields', name: 'fields', component: FieldsView },
    { path: '/t/:id/teams', name: 'teams', component: TeamsView },
    { path: '/t/:id/matches', name: 'matches', component: ScheduleView },
    { path: '/t/:id', redirect: (to) => ({ name: 'settings', params: { id: to.params.id } }) },
  ],
})

export default router
