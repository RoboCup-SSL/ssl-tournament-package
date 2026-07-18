<script setup lang="ts">
// The tournament Settings section: an editable form over the Tournament fields,
// with dirty-tracked Save/Discard. Values are validated + normalized server-side
// (M2b·dt); native date/time inputs emit exactly the canonical formats.
import { computed, ref, watch } from 'vue'
import { useTournamentStore } from '@/store/tournament'

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

const saved = ref(false)
// Show "Saved." only until the next edit; reset it when the loaded tournament
// changes (the same view instance is reused across /t/:id).
const showSaved = computed(() => saved.value && !store.dirty)
watch(() => store.current?.id, () => { saved.value = false })

async function onSave() {
  saved.value = false
  await store.save()
  if (!store.error) saved.value = true
}
</script>

<template>
  <q-page padding>
    <q-banner v-if="store.error" class="bg-negative text-white q-mb-md">
      {{ store.error }}
    </q-banner>

    <div v-if="store.loading && !store.draft" class="text-grey">Loading…</div>

    <template v-else-if="store.draft">
      <div class="text-h6 q-mb-md">Settings</div>

      <q-banner v-if="showSaved" class="bg-positive text-white q-mb-md">
        Saved.
      </q-banner>

      <div class="column q-gutter-md" style="max-width: 480px">
        <q-input v-model="store.draft.name" label="Name" />
        <q-input v-model="store.draft.location" label="Location" />

        <div class="row q-col-gutter-md">
          <q-input class="col" v-model="store.draft.starts_on" label="Starts on" type="date" stack-label />
          <q-input class="col" v-model="store.draft.ends_on" label="Ends on" type="date" stack-label />
        </div>

        <div class="row q-col-gutter-md">
          <q-input class="col" v-model="store.draft.venue_opens" label="Venue opens" type="time" stack-label />
          <q-input class="col" v-model="store.draft.venue_closes" label="Venue closes" type="time" stack-label />
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

        <div class="row q-col-gutter-md">
          <q-input class="col" v-model="store.draft.default_match_minutes" label="Default match minutes" type="number" />
          <q-input class="col" v-model="store.draft.default_gap_minutes" label="Default gap minutes" type="number" />
        </div>

        <div class="row q-gutter-sm">
          <q-btn color="primary" label="Save" :disable="!store.dirty" :loading="store.saving" @click="onSave" />
          <q-btn flat label="Discard" :disable="!store.dirty" @click="store.discard" />
        </div>
      </div>
    </template>
  </q-page>
</template>
