<script setup lang="ts">
// The schedule grid (MVP): matches on a time × field grid for a selected day.
// Drag a card to move it (desktop); tap a card to edit/move it (everywhere).
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useQuasar } from 'quasar'
import { useMatchesStore } from '@/store/matches'
import { useFieldsStore } from '@/store/fields'
import { useTeamsStore } from '@/store/teams'
import { useTournamentStore } from '@/store/tournament'
import type { Match, MatchInput } from '@/api/types'
import { MATCH_STATUSES } from '@/api/types'

const route = useRoute()
const $q = useQuasar()
const matches = useMatchesStore()
const fields = useFieldsStore()
const teams = useTeamsStore()
const tournament = useTournamentStore()

watch(
  () => route.params.id,
  (id) => {
    if (typeof id !== 'string') return
    const n = Number(id)
    void matches.fetch(n)
    void fields.fetch(n)
    void teams.fetch(n)
  },
  { immediate: true },
)

// --- time helpers (naive local, minute precision) ---
function parseHM(hm: string): number {
  const [h, m] = hm.split(':').map(Number)
  return (h || 0) * 60 + (m || 0)
}
function fmtHM(mins: number): string {
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}
function todayISO(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
function nextDay(iso: string): string {
  const dt = new Date(iso + 'T00:00:00Z')
  dt.setUTCDate(dt.getUTCDate() + 1)
  return dt.toISOString().slice(0, 10)
}

// --- days (columns of the day selector) ---
const days = computed<string[]>(() => {
  const set = new Set<string>()
  for (const m of matches.items) if (m.scheduled_at) set.add(m.scheduled_at.slice(0, 10))
  const t = tournament.current
  if (t?.starts_on) {
    const end = t.ends_on || t.starts_on
    let d = t.starts_on
    let guard = 0
    while (d <= end && guard < 60) {
      set.add(d)
      d = nextDay(d)
      guard++
    }
  }
  if (set.size === 0) set.add(t?.starts_on || todayISO())
  return Array.from(set).sort()
})

const selectedDay = ref('')
watch(
  days,
  (list) => {
    if (!list.includes(selectedDay.value)) selectedDay.value = list[0] ?? ''
  },
  { immediate: true },
)

// --- time rows for the selected day ---
const slotMinutes = computed(() => {
  const t = tournament.current
  const step = (t?.default_match_minutes ?? 0) + (t?.default_gap_minutes ?? 0)
  return step > 0 ? step : 60
})

const times = computed<string[]>(() => {
  const set = new Set<string>()
  const t = tournament.current
  const open = parseHM(t?.venue_opens || '08:00')
  const close = parseHM(t?.venue_closes || '20:00')
  for (let mins = open; mins <= close && set.size < 200; mins += slotMinutes.value) set.add(fmtHM(mins))
  for (const m of matches.items) {
    if (m.scheduled_at && m.scheduled_at.slice(0, 10) === selectedDay.value) {
      set.add(m.scheduled_at.slice(11, 16))
    }
  }
  return Array.from(set).sort()
})

const fieldList = computed(() => fields.items)
const unscheduled = computed(() => matches.items.filter((m) => m.field_id == null || m.scheduled_at == null))

function cellMatches(fieldId: number, time: string): Match[] {
  const at = `${selectedDay.value}T${time}`
  return matches.items.filter((m) => m.field_id === fieldId && m.scheduled_at === at)
}

function teamName(id: number | null): string {
  if (id == null) return '—'
  return teams.items.find((t) => t.id === id)?.name || `#${id}`
}

function statusColor(status: string): string {
  if (status === 'finished') return 'positive'
  if (status === 'playing') return 'orange'
  if (status === 'cancelled' || status === 'invalidated') return 'grey'
  return 'primary'
}

// --- drag to move (desktop) ---
const dragId = ref<number | null>(null)
function onDragStart(id: number) {
  dragId.value = id
}
async function onDropCell(fieldId: number, time: string) {
  const id = dragId.value
  dragId.value = null
  if (id == null) return
  const at = `${selectedDay.value}T${time}`
  const dragged = matches.items.find((m) => m.id === id)
  if (dragged && dragged.field_id === fieldId && dragged.scheduled_at === at) return
  await matches.update(id, { field_id: fieldId, scheduled_at: at })
  notify(matches.error ? '' : 'Moved')
}
async function onDropUnscheduled() {
  const id = dragId.value
  dragId.value = null
  if (id == null) return
  await matches.update(id, { scheduled_at: null })
  notify(matches.error ? '' : 'Unscheduled')
}
function notify(ok: string) {
  if (matches.error) $q.notify({ type: 'negative', message: matches.error })
  else if (ok) $q.notify({ type: 'positive', message: ok })
}

// --- add / edit dialog ---
const teamOptions = computed(() => teams.items.map((t) => ({ label: t.name || `#${t.id}`, value: t.id })))
const fieldOptions = computed(() => fields.items.map((f) => ({ label: f.name || `#${f.id}`, value: f.id })))
const statusOptions = [...MATCH_STATUSES]

interface Form {
  label: string
  a_team_id: number | null
  b_team_id: number | null
  field_id: number | null
  date: string
  time: string
  referee_team_id: number | null
  assistant_referee_team_id: number | null
  status: string
}

function blankForm(): Form {
  return {
    label: '',
    a_team_id: null,
    b_team_id: null,
    field_id: null,
    date: selectedDay.value,
    time: '',
    referee_team_id: null,
    assistant_referee_team_id: null,
    status: 'scheduled',
  }
}

const dialog = ref(false)
const editing = ref<Match | null>(null)
const form = ref<Form>(blankForm())
const busy = ref(false)

function openAdd(prefill?: { field_id?: number; time?: string }) {
  editing.value = null
  form.value = blankForm()
  if (prefill?.field_id != null) form.value.field_id = prefill.field_id
  if (prefill?.time) form.value.time = prefill.time
  dialog.value = true
}

function openEdit(m: Match) {
  editing.value = m
  form.value = {
    label: m.label,
    a_team_id: m.a_team_id,
    b_team_id: m.b_team_id,
    field_id: m.field_id,
    date: m.scheduled_at ? m.scheduled_at.slice(0, 10) : '',
    time: m.scheduled_at ? m.scheduled_at.slice(11, 16) : '',
    referee_team_id: m.referee_team_id,
    assistant_referee_team_id: m.assistant_referee_team_id,
    status: m.status,
  }
  dialog.value = true
}

function buildInput(f: Form): MatchInput {
  return {
    label: f.label,
    a_team_id: f.a_team_id,
    b_team_id: f.b_team_id,
    field_id: f.field_id,
    scheduled_at: f.date && f.time ? `${f.date}T${f.time}` : null,
    referee_team_id: f.referee_team_id,
    assistant_referee_team_id: f.assistant_referee_team_id,
    status: f.status,
  }
}

async function submit() {
  busy.value = true
  try {
    const input = buildInput(form.value)
    if (editing.value) await matches.update(editing.value.id, input)
    else await matches.create(input)
    if (matches.error) {
      $q.notify({ type: 'negative', message: matches.error })
    } else {
      dialog.value = false
      $q.notify({ type: 'positive', message: 'Saved' })
    }
  } finally {
    busy.value = false
  }
}

function confirmDelete() {
  const m = editing.value
  if (!m) return
  $q.dialog({ title: 'Delete match', message: 'Delete this match?', cancel: true }).onOk(async () => {
    await matches.remove(m.id)
    if (matches.error) {
      $q.notify({ type: 'negative', message: matches.error })
    } else {
      dialog.value = false
      $q.notify({ type: 'positive', message: 'Deleted' })
    }
  })
}
</script>

<template>
  <q-page padding>
    <div class="row items-center q-mb-md">
      <div class="text-h6">Schedule</div>
      <q-space />
      <q-btn color="primary" icon="add" label="Add match" @click="openAdd()" />
    </div>

    <q-tabs
      v-if="days.length > 1"
      v-model="selectedDay"
      dense
      no-caps
      align="left"
      class="q-mb-sm text-primary"
    >
      <q-tab v-for="d in days" :key="d" :name="d" :label="d" />
    </q-tabs>

    <q-banner v-if="matches.error" class="bg-negative text-white q-mb-md">{{ matches.error }}</q-banner>

    <!-- Unscheduled strip -->
    <div class="unscheduled" @dragover.prevent @drop="onDropUnscheduled">
      <div class="text-caption text-grey q-mb-xs">Unscheduled (drag onto the grid, or tap to edit)</div>
      <div v-if="unscheduled.length" class="card-list">
        <div
          v-for="m in unscheduled"
          :key="m.id"
          class="match-card"
          draggable="true"
          @dragstart="onDragStart(m.id)"
          @dragend="dragId = null"
          @click="openEdit(m)"
        >
          <div class="teams">{{ teamName(m.a_team_id) }} <span class="vs">vs</span> {{ teamName(m.b_team_id) }}</div>
        </div>
      </div>
      <div v-else class="text-caption text-grey">none</div>
    </div>

    <q-banner v-if="!fieldList.length" class="bg-grey-2 q-my-md">
      Add fields in the <b>Fields</b> section to build the schedule grid.
    </q-banner>

    <!-- Grid -->
    <div v-else class="schedule-scroll">
      <table class="grid">
        <thead>
          <tr>
            <th class="time-col">Time</th>
            <th v-for="f in fieldList" :key="f.id">{{ f.name || `#${f.id}` }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="time in times" :key="time">
            <td class="time-col">{{ time }}</td>
            <td
              v-for="f in fieldList"
              :key="f.id"
              class="cell"
              @dragover.prevent
              @drop="onDropCell(f.id, time)"
              @dblclick="openAdd({ field_id: f.id, time })"
            >
              <div
                v-for="m in cellMatches(f.id, time)"
                :key="m.id"
                class="match-card"
                draggable="true"
                @dragstart="onDragStart(m.id)"
          @dragend="dragId = null"
                @click="openEdit(m)"
              >
                <div class="teams">{{ teamName(m.a_team_id) }} <span class="vs">vs</span> {{ teamName(m.b_team_id) }}</div>
                <div class="meta">
                  <span v-if="m.label">{{ m.label }}</span>
                  <span v-if="m.referee_team_id"> · ref {{ teamName(m.referee_team_id) }}</span>
                  <q-badge v-if="m.status !== 'scheduled'" :color="statusColor(m.status)" class="q-ml-xs">
                    {{ m.status }}
                  </q-badge>
                </div>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Add / edit dialog -->
    <q-dialog v-model="dialog">
      <q-card style="min-width: 340px; max-width: 95vw">
        <q-card-section class="text-h6">{{ editing ? 'Edit match' : 'New match' }}</q-card-section>
        <q-card-section class="dialog-form q-pt-none">
          <q-input v-model="form.label" label="Label (optional)" />
          <q-select v-model="form.a_team_id" :options="teamOptions" label="Team A" emit-value map-options clearable />
          <q-select v-model="form.b_team_id" :options="teamOptions" label="Team B" emit-value map-options clearable />
          <q-select v-model="form.field_id" :options="fieldOptions" label="Field" emit-value map-options clearable />
          <div class="dialog-row">
            <q-input v-model="form.date" label="Date" type="date" stack-label />
            <q-input v-model="form.time" label="Time" type="time" stack-label />
          </div>
          <q-select v-model="form.referee_team_id" :options="teamOptions" label="Referee team" emit-value map-options clearable />
          <q-select v-model="form.assistant_referee_team_id" :options="teamOptions" label="Assistant referee" emit-value map-options clearable />
          <q-select v-model="form.status" :options="statusOptions" label="Status" />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn v-if="editing" flat color="negative" label="Delete" @click="confirmDelete" />
          <q-space />
          <q-btn v-close-popup flat label="Cancel" />
          <q-btn color="primary" :label="editing ? 'Save' : 'Create'" :loading="busy" @click="submit" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>

<style scoped>
.schedule-scroll {
  overflow-x: auto;
}
.grid {
  border-collapse: collapse;
  min-width: 100%;
}
.grid th,
.grid td {
  border: 1px solid #ddd;
  vertical-align: top;
  padding: 2px;
}
.grid th {
  background: var(--q-primary);
  color: #fff;
  font-weight: 500;
  white-space: nowrap;
}
.time-col {
  position: sticky;
  left: 0;
  z-index: 1;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
td.time-col {
  background: #fafafa;
}
.cell {
  min-width: 140px;
  height: 44px;
}
.card-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.match-card {
  background: #e8f2f8;
  border-left: 3px solid var(--q-primary);
  border-radius: 3px;
  padding: 2px 5px;
  margin: 2px 0;
  cursor: pointer;
  font-size: 12px;
  line-height: 1.3;
}
.card-list .match-card {
  margin: 0;
}
.match-card .teams {
  font-weight: 500;
}
.match-card .vs {
  color: #888;
  font-weight: 400;
}
.match-card .meta {
  color: #666;
  font-size: 11px;
}
.unscheduled {
  border: 1px dashed #bbb;
  border-radius: 4px;
  padding: 8px;
  margin-bottom: 12px;
}
.dialog-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.dialog-row {
  display: flex;
  gap: 12px;
}
.dialog-row > * {
  flex: 1 1 140px;
}
</style>
