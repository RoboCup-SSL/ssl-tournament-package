<script setup lang="ts">
// Instance landing: lists tournaments from /api/tournaments.
import { onMounted } from 'vue'
import { useTournamentsStore } from '@/store/tournaments'

const store = useTournamentsStore()

onMounted(() => store.fetch())
</script>

<template>
  <q-page padding>
    <div class="text-h5 q-mb-md">Tournaments</div>

    <q-banner v-if="store.error" class="bg-negative text-white q-mb-md">
      {{ store.error }}
    </q-banner>

    <q-spinner v-if="store.loading" size="2em" />

    <q-list v-else-if="store.tournaments.length" bordered separator>
      <q-item v-for="t in store.tournaments" :key="t.id">
        <q-item-section>
          <q-item-label>{{ t.name }}</q-item-label>
          <q-item-label caption>
            {{ t.location || 'no location' }}<span v-if="t.starts_on"> · {{ t.starts_on }}</span>
          </q-item-label>
        </q-item-section>
      </q-item>
    </q-list>

    <div v-else class="text-grey">No tournaments yet.</div>
  </q-page>
</template>
