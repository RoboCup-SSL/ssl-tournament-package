<script setup lang="ts">
// The tournament Settings section: an editable form over the Tournament fields,
// with dirty-tracked Save/Discard. Values are validated + normalized server-side
// (M2b·dt); native date/time inputs emit exactly the canonical formats.
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import { useTournamentStore } from '@/store/tournament'

const $q = useQuasar()
const router = useRouter()
const store = useTournamentStore()

// All IANA zone names from the browser (no dependency); the backend validates
// the chosen name. Guarded for older engines lacking supportedValuesOf.
const intl = Intl as { supportedValuesOf?: (key: string) => string[] }
const allZones: string[] =
  typeof intl.supportedValuesOf === 'function' ? intl.supportedValuesOf('timeZone') : []
const zoneOptions = ref<string[]>(allZones)

function filterZones(value: string, update: (fn: () => void) => void) {
  update(() => {
    const needle = value.toLowerCase()
    zoneOptions.value = needle
      ? allZones.filter((zone) => zone.toLowerCase().includes(needle))
      : allZones
  })
}

// Feedback via a Notify toast (visible at the bottom regardless of scroll) — a
// top-of-page banner is off-screen after saving from the bottom of a long form.
async function onSave() {
  await store.save()
  if (store.error) $q.notify({ type: 'negative', message: store.error })
  else $q.notify({ type: 'positive', message: 'Saved' })
}

// Delete requires typing the tournament's exact name — no accidental deletes.
const deleteDialog = ref(false)
const deleteConfirm = ref('')
const deleting = ref(false)
const canDelete = computed(() => !!store.current && deleteConfirm.value === store.current.name)

function openDelete() {
  deleteConfirm.value = ''
  deleteDialog.value = true
}
async function onDelete() {
  if (!canDelete.value) return
  deleting.value = true
  try {
    const ok = await store.remove()
    if (ok) {
      deleteDialog.value = false
      $q.notify({ type: 'positive', message: 'Tournament deleted' })
      void router.push('/')
    } else {
      $q.notify({ type: 'negative', message: store.error || 'Delete failed' })
    }
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <q-page padding>
    <q-banner v-if="store.error && !store.draft" class="bg-negative text-white q-mb-md">
      {{ store.error }}
    </q-banner>

    <div v-if="store.loading && !store.draft" class="text-grey">Loading…</div>

    <template v-else-if="store.draft">
      <div class="text-h6 q-mb-md">Settings</div>

      <div class="settings-form">
        <q-input v-model="store.draft.name" label="Name" />
        <q-input v-model="store.draft.location" label="Location" />

        <div class="field-row">
          <q-input v-model="store.draft.starts_on" label="Starts on" type="date" stack-label />
          <q-input v-model="store.draft.ends_on" label="Ends on" type="date" stack-label />
        </div>

        <div class="field-row">
          <q-input v-model="store.draft.venue_opens" label="Venue opens" type="time" stack-label />
          <q-input v-model="store.draft.venue_closes" label="Venue closes" type="time" stack-label />
        </div>

        <q-select
          v-model="store.draft.time_zone"
          :options="zoneOptions"
          label="Time zone"
          use-input
          clearable
          input-debounce="0"
          hint="One anchor for all local times in this tournament"
          @filter="filterZones"
        />

        <div class="field-row">
          <q-input v-model="store.draft.default_match_minutes" label="Default match minutes" type="number" />
          <q-input v-model="store.draft.default_gap_minutes" label="Default gap minutes" type="number" />
        </div>

        <div class="actions">
          <q-btn color="primary" label="Save" :disable="!store.dirty" :loading="store.saving" @click="onSave" />
          <q-btn flat label="Discard" :disable="!store.dirty" @click="store.discard" />
        </div>
      </div>

      <q-separator class="q-my-lg" />
      <div class="danger-zone">
        <div class="text-subtitle2 text-negative q-mb-xs">Danger zone</div>
        <div class="text-caption text-grey q-mb-sm">
          Permanently removes this tournament and everything in it — fields, teams, matches, and
          results. This cannot be undone.
        </div>
        <q-btn outline color="negative" icon="delete" label="Delete tournament" @click="openDelete" />
      </div>

      <q-dialog v-model="deleteDialog">
        <q-card style="min-width: 320px; max-width: 95vw">
          <q-card-section class="text-h6 text-negative">Delete tournament</q-card-section>
          <q-card-section class="q-pt-none">
            <p class="q-mb-sm">
              This permanently deletes <b>{{ store.current?.name }}</b> and all its fields, teams,
              matches, and results. This cannot be undone.
            </p>
            <p class="q-mb-xs text-caption">Type the tournament name to confirm:</p>
            <q-input
              v-model="deleteConfirm"
              dense
              autofocus
              :placeholder="store.current?.name"
              @keyup.enter="onDelete"
            />
          </q-card-section>
          <q-card-actions align="right">
            <q-btn v-close-popup flat label="Cancel" />
            <q-btn color="negative" label="Delete" :disable="!canDelete" :loading="deleting" @click="onDelete" />
          </q-card-actions>
        </q-card>
      </q-dialog>
    </template>
  </q-page>
</template>

<style scoped>
/* Plain flex + gap, no Quasar negative-margin gutters (which overflow the left
   edge on narrow screens). Paired fields wrap to stacked on phones. */
.settings-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 480px;
}
.field-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}
.field-row > * {
  flex: 1 1 180px;
}
.actions {
  display: flex;
  gap: 8px;
}
.danger-zone {
  max-width: 480px;
}
</style>
