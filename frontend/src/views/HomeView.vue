<script setup lang="ts">
// Instance landing: lists tournaments, creates new ones, opens one's workspace.
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTournamentsStore } from '@/store/tournaments'

const store = useTournamentsStore()
const router = useRouter()

const dialog = ref(false)
const newName = ref('')
const creating = ref(false)

onMounted(() => store.fetch())

function openDialog() {
  store.error = ''
  newName.value = ''
  dialog.value = true
}

function open(id: number) {
  void router.push({ name: 'matches', params: { id } })
}

async function create() {
  if (!newName.value.trim()) return
  creating.value = true
  try {
    const created = await store.create(newName.value.trim())
    if (created) {
      dialog.value = false
      newName.value = ''
      void router.push({ name: 'settings', params: { id: created.id } })
    }
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <q-page padding>
    <div class="row items-center q-mb-md">
      <div class="text-h5">Tournaments</div>
      <q-space />
      <q-btn color="primary" icon="add" label="New tournament" @click="openDialog" />
    </div>

    <q-banner v-if="store.error" class="bg-negative text-white q-mb-md">
      {{ store.error }}
    </q-banner>

    <q-spinner v-if="store.loading" size="2em" />

    <q-list v-else-if="store.tournaments.length" bordered separator>
      <q-item v-for="t in store.tournaments" :key="t.id" clickable @click="open(t.id)">
        <q-item-section>
          <q-item-label>{{ t.name }}</q-item-label>
          <q-item-label caption>
            {{ t.location || 'no location' }}<span v-if="t.starts_on"> · {{ t.starts_on }}</span><span v-if="t.time_zone"> · {{ t.time_zone }}</span>
          </q-item-label>
        </q-item-section>
        <q-item-section side>
          <q-icon name="chevron_right" />
        </q-item-section>
      </q-item>
    </q-list>

    <div v-else class="text-grey">No tournaments yet.</div>

    <q-dialog v-model="dialog">
      <q-card style="min-width: 320px">
        <q-card-section class="text-h6">New tournament</q-card-section>
        <q-card-section class="q-pt-none">
          <q-input v-model="newName" label="Name" autofocus @keyup.enter="create" />
        </q-card-section>
        <q-card-section v-if="store.error" class="q-pt-none text-negative">
          {{ store.error }}
        </q-card-section>
        <q-card-actions align="right">
          <q-btn v-close-popup flat label="Cancel" />
          <q-btn color="primary" label="Create" :loading="creating" @click="create" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>
