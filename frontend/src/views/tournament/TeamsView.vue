<script setup lang="ts">
// Teams section: list, add, edit, and delete a tournament's teams.
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useQuasar } from 'quasar'
import { useTeamsStore } from '@/store/teams'
import type { Team, TeamInput } from '@/api/types'

const route = useRoute()
const $q = useQuasar()
const store = useTeamsStore()

watch(
  () => route.params.id,
  (id) => { if (typeof id === 'string') void store.fetch(Number(id)) },
  { immediate: true },
)

const dialog = ref(false)
const editing = ref<Team | null>(null)
const form = ref<Required<Pick<TeamInput, 'name' | 'country' | 'contact' | 'notes'>>>({
  name: '',
  country: '',
  contact: '',
  notes: '',
})
const busy = ref(false)

function openAdd() {
  editing.value = null
  form.value = { name: '', country: '', contact: '', notes: '' }
  dialog.value = true
}

function openEdit(team: Team) {
  editing.value = team
  form.value = { name: team.name, country: team.country, contact: team.contact, notes: team.notes }
  dialog.value = true
}

async function submit() {
  if (!form.value.name.trim()) return
  const input: TeamInput = {
    name: form.value.name.trim(),
    country: form.value.country,
    contact: form.value.contact,
    notes: form.value.notes,
  }
  busy.value = true
  try {
    if (editing.value) await store.update(editing.value.id, input)
    else await store.create(input)
    if (store.error) {
      $q.notify({ type: 'negative', message: store.error })
    } else {
      dialog.value = false
      $q.notify({ type: 'positive', message: 'Saved' })
    }
  } finally {
    busy.value = false
  }
}

function confirmDelete(team: Team) {
  $q.dialog({ title: 'Delete team', message: `Delete "${team.name || 'this team'}"?`, cancel: true })
    .onOk(async () => {
      await store.remove(team.id)
      if (store.error) $q.notify({ type: 'negative', message: store.error })
      else $q.notify({ type: 'positive', message: 'Deleted' })
    })
}
</script>

<template>
  <q-page padding>
    <div class="row items-center q-mb-md">
      <div class="text-h6">Teams</div>
      <q-space />
      <q-btn color="primary" icon="add" label="Add team" @click="openAdd" />
    </div>

    <q-banner v-if="store.error && !store.items.length" class="bg-negative text-white q-mb-md">
      {{ store.error }}
    </q-banner>
    <q-spinner v-if="store.loading && !store.items.length" size="2em" />

    <q-list v-else-if="store.items.length" bordered separator>
      <q-item v-for="t in store.items" :key="t.id">
        <q-item-section>
          <q-item-label>{{ t.name || '(unnamed)' }}</q-item-label>
          <q-item-label v-if="t.country || t.contact" caption>
            {{ t.country }}<span v-if="t.country && t.contact"> · </span>{{ t.contact }}
          </q-item-label>
        </q-item-section>
        <q-item-section side>
          <div class="row items-center no-wrap">
            <q-btn flat round dense icon="edit" @click="openEdit(t)" />
            <q-btn flat round dense icon="delete" color="negative" @click="confirmDelete(t)" />
          </div>
        </q-item-section>
      </q-item>
    </q-list>
    <div v-else class="text-grey">No teams yet.</div>

    <q-dialog v-model="dialog">
      <q-card style="min-width: 320px; max-width: 95vw">
        <q-card-section class="text-h6">{{ editing ? 'Edit team' : 'New team' }}</q-card-section>
        <q-separator />
        <q-card-section class="q-pt-md" style="max-height: 60vh; overflow-y: auto">
          <div class="dialog-form">
            <q-input v-model="form.name" label="Name" autofocus />
            <q-input v-model="form.country" label="Country" />
            <q-input v-model="form.contact" label="Contact" />
            <q-input v-model="form.notes" label="Notes" type="textarea" autogrow />
          </div>
        </q-card-section>
        <q-separator />
        <q-card-actions align="right">
          <q-btn v-close-popup flat label="Cancel" />
          <q-btn color="primary" :label="editing ? 'Save' : 'Create'" :loading="busy" @click="submit" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>

<style scoped>
.dialog-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
