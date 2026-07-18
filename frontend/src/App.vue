<script setup lang="ts">
// App chrome. On `/` it shows the global header; inside a tournament (/t/:id/…)
// it shows that tournament's name and a section-nav drawer (auto-collapsing to a
// menu on mobile). The tournament (re)loads whenever the route id changes.
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'
import type { Version } from '@/api/types'
import { useTournamentStore } from '@/store/tournament'

const route = useRoute()
const tournament = useTournamentStore()
const version = ref('')
const drawer = ref(false)

// workspaceId is the tournament id from the route, or null on the list page.
const workspaceId = computed(() =>
  typeof route.params.id === 'string' ? route.params.id : null)

const sections = [
  { name: 'settings', label: 'Settings', icon: 'settings', enabled: true },
  { name: 'fields', label: 'Fields', icon: 'stadium', enabled: true },
  { name: 'teams', label: 'Teams', icon: 'groups', enabled: true },
  { name: 'divisions', label: 'Divisions', icon: 'account_tree', enabled: false },
  { name: 'groups', label: 'Groups', icon: 'grid_view', enabled: false },
  { name: 'matches', label: 'Matches', icon: 'sports_soccer', enabled: false },
  { name: 'standings', label: 'Standings', icon: 'leaderboard', enabled: false },
  { name: 'placements', label: 'Placements', icon: 'emoji_events', enabled: false },
]

const headerTitle = computed(() =>
  workspaceId.value ? (tournament.current?.name ?? '…') : `SSL Tournament ${version.value}`)

watch(workspaceId, (id) => { if (id) void tournament.load(Number(id)) }, { immediate: true })

onMounted(async () => {
  try {
    version.value = (await api.get<Version>('/api/version')).version
  } catch {
    version.value = ''
  }
})
</script>

<template>
  <q-layout view="hHh LpR fFf">
    <q-header elevated>
      <q-toolbar>
        <q-btn v-if="workspaceId" flat round dense icon="menu" @click="drawer = !drawer" />
        <q-btn v-if="workspaceId" flat round dense icon="arrow_back" to="/" title="All tournaments" />
        <q-toolbar-title>
          {{ headerTitle }}
        </q-toolbar-title>
        <q-btn flat dense no-caps label="admin" type="a" href="/admin/" />
      </q-toolbar>
    </q-header>

    <q-drawer v-if="workspaceId" v-model="drawer" show-if-above bordered :width="220">
      <q-list padding>
        <q-item-label header>Sections</q-item-label>
        <q-item
          v-for="s in sections"
          :key="s.name"
          clickable
          :disable="!s.enabled"
          :active="s.enabled && route.name === s.name"
          :to="s.enabled ? { name: s.name, params: { id: workspaceId } } : undefined"
        >
          <q-item-section avatar>
            <q-icon :name="s.icon" />
          </q-item-section>
          <q-item-section>{{ s.label }}</q-item-section>
          <q-item-section v-if="!s.enabled" side>
            <q-badge color="grey-5">soon</q-badge>
          </q-item-section>
        </q-item>
      </q-list>
    </q-drawer>

    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>
