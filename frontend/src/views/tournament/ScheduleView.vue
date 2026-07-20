<script setup lang="ts">
// The schedule (MVP): a Google-Calendar-style day view. Columns = fields, a
// continuous vertical time axis; each match is a block positioned by its start
// and sized by its duration. Click empty space to add, click a block to edit,
// drag a block (desktop) to move it to another field/time.
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useQuasar } from 'quasar'
import { useMatchesStore } from '@/store/matches'
import { useFieldsStore } from '@/store/fields'
import { useTeamsStore } from '@/store/teams'
import { useTournamentStore } from '@/store/tournament'
import type { Match, MatchInput } from '@/api/types'
import TimeField from '@/components/TimeField.vue'
import { toast } from '@/toast'

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
function addMinutes(hm: string, mins: number): string {
  return fmtHM(Math.max(0, Math.min(parseHM(hm) + mins, 24 * 60 - 1)))
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
function defaultDuration(): number {
  return tournament.current?.default_match_minutes || 60
}

// --- days ---
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

// --- calendar geometry ---
const PX_PER_MIN = 1
// Gridlines/labels are hourly; drag + click-to-add snap to a finer resolution.
const GRID_MINUTES = 60
const SNAP_MINUTES = 15

function onDay(m: Match): boolean {
  return !!m.scheduled_at && m.scheduled_at.slice(0, 10) === selectedDay.value
}

// The vertical span shown: the venue hours, widened to include every match on
// the day (start .. start+duration), snapped out to whole hours.
const dayBounds = computed(() => {
  const t = tournament.current
  let start = parseHM(t?.venue_opens || '08:00')
  let end = parseHM(t?.venue_closes || '20:00')
  for (const m of matches.items) {
    if (!onDay(m)) continue
    const s = parseHM(m.scheduled_at!.slice(11, 16))
    const e = s + (m.duration_minutes || defaultDuration())
    if (s < start) start = s
    if (e > end) end = e
  }
  start = Math.floor(start / 60) * 60
  end = Math.ceil(end / 60) * 60
  if (end <= start) end = start + 60
  return { start, end }
})

const totalHeight = computed(() => (dayBounds.value.end - dayBounds.value.start) * PX_PER_MIN)

const gridLines = computed(() => {
  const { start, end } = dayBounds.value
  const lines: { top: number; label: string }[] = []
  for (let m = start; m <= end; m += GRID_MINUTES) {
    lines.push({ top: (m - start) * PX_PER_MIN, label: fmtHM(m) })
  }
  return lines
})

const fieldList = computed(() => fields.items)
const unscheduled = computed(() => matches.items.filter((m) => m.field_id == null || m.scheduled_at == null))

interface Block {
  m: Match
  top: number
  height: number
  lane: number
  lanes: number
}

// Positioned blocks for one field on the selected day, with side-by-side lanes
// for overlapping matches (classic calendar packing).
function fieldBlocks(fieldId: number): Block[] {
  const { start } = dayBounds.value
  const events = matches.items
    .filter((m) => m.field_id === fieldId && onDay(m))
    .map((m) => {
      const s = parseHM(m.scheduled_at!.slice(11, 16))
      const e = s + Math.max(m.duration_minutes || defaultDuration(), 15)
      return { m, s, e }
    })
    .sort((a, b) => a.s - b.s || a.e - b.e)

  const blocks: Block[] = []
  let cluster: typeof events = []
  let clusterEnd = -1
  const flush = () => {
    if (!cluster.length) return
    const laneEnds: number[] = []
    const placed = cluster.map((ev) => {
      let lane = laneEnds.findIndex((end) => end <= ev.s)
      if (lane === -1) {
        lane = laneEnds.length
        laneEnds.push(ev.e)
      } else {
        laneEnds[lane] = ev.e
      }
      return { ev, lane }
    })
    for (const { ev, lane } of placed) {
      blocks.push({
        m: ev.m,
        top: (ev.s - start) * PX_PER_MIN,
        height: (ev.e - ev.s) * PX_PER_MIN,
        lane,
        lanes: laneEnds.length,
      })
    }
    cluster = []
  }
  for (const ev of events) {
    if (cluster.length && ev.s >= clusterEnd) flush()
    clusterEnd = cluster.length === 0 ? ev.e : Math.max(clusterEnd, ev.e)
    cluster.push(ev)
  }
  flush()
  return blocks
}

function blockStyle(b: Block): Record<string, string> {
  const widthPct = 100 / b.lanes
  const style: Record<string, string> = {
    top: `${b.top}px`,
    height: `${Math.max(b.height, 18)}px`,
    left: `calc(${b.lane * widthPct}% + 1px)`,
    width: `calc(${widthPct}% - 3px)`,
  }
  const accent = statusCss(b.m.status)
  if (accent) style.borderLeftColor = accent
  return style
}

function teamName(id: number | null): string {
  if (id == null) return '—'
  return teams.items.find((t) => t.id === id)?.name || `#${id}`
}
function matchTitle(m: Match): string {
  if (m.a_team_id != null || m.b_team_id != null) {
    return `${teamName(m.a_team_id)} vs ${teamName(m.b_team_id)}`
  }
  return m.label || `Match #${m.id}`
}
function matchRange(m: Match): string {
  if (!m.scheduled_at) return ''
  const s = m.scheduled_at.slice(11, 16)
  return `${s}–${addMinutes(s, m.duration_minutes || defaultDuration())}`
}
// A CSS accent color per status (blank = scheduled, uses the default block color).
function statusCss(status: string): string {
  if (status === 'finished') return '#21BA45'
  if (status === 'playing') return '#F2A007'
  if (status === 'cancelled' || status === 'invalidated') return '#9e9e9e'
  return ''
}

// map a pointer Y within a column to a snapped time string
function timeAtY(el: HTMLElement, clientY: number): string {
  const y = clientY - el.getBoundingClientRect().top
  const raw = dayBounds.value.start + y / PX_PER_MIN
  const snapped = Math.round(raw / SNAP_MINUTES) * SNAP_MINUTES
  // A start time must stay a valid HH:MM (< 24:00); cap below midnight.
  const maxStart = Math.min(dayBounds.value.end, 24 * 60 - SNAP_MINUTES)
  return fmtHM(Math.max(dayBounds.value.start, Math.min(snapped, maxStart)))
}

function onColClick(event: MouseEvent, fieldId: number) {
  openAdd({ field_id: fieldId, time: timeAtY(event.currentTarget as HTMLElement, event.clientY) })
}

// --- drag to move ---
const dragId = ref<number | null>(null)
// Distance from the grabbed block's top to the cursor, so the drop positions
// the block's top (not the cursor) at the target time.
const dragOffsetY = ref(0)
function onDragStart(id: number, event: DragEvent) {
  dragId.value = id
  const el = event.currentTarget as HTMLElement | null
  dragOffsetY.value = el ? event.clientY - el.getBoundingClientRect().top : 0
}
async function onDropCol(event: DragEvent, fieldId: number) {
  const id = dragId.value
  dragId.value = null
  if (id == null) return
  const at = `${selectedDay.value}T${timeAtY(event.currentTarget as HTMLElement, event.clientY - dragOffsetY.value)}`
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
  if (matches.error) toast('negative', matches.error)
  else if (ok) toast('positive', ok)
}

// --- add / edit dialog ---
const teamOptions = computed(() => teams.items.map((t) => ({ label: t.name || `#${t.id}`, value: t.id })))
const fieldOptions = computed(() => fields.items.map((f) => ({ label: f.name || `#${f.id}`, value: f.id })))

interface Form {
  label: string
  a_team_id: number | null
  b_team_id: number | null
  field_id: number | null
  date: string
  time: string
  endTime: string
  referee_team_id: number | null
  assistant_referee_team_id: number | null
  a_score: string
  b_score: string
  winner_team_id: number | null
}

// Winner options are the two participating teams (freeform winners: use /admin/).
const winnerOptions = computed(() => {
  const opts: { label: string; value: number }[] = []
  if (form.value.a_team_id != null) opts.push({ label: teamName(form.value.a_team_id), value: form.value.a_team_id })
  if (form.value.b_team_id != null) opts.push({ label: teamName(form.value.b_team_id), value: form.value.b_team_id })
  return opts
})

function blankForm(): Form {
  return {
    label: '',
    a_team_id: null,
    b_team_id: null,
    field_id: null,
    date: tournament.current?.starts_on || selectedDay.value || '',
    time: '',
    endTime: '',
    referee_team_id: null,
    assistant_referee_team_id: null,
    a_score: '',
    b_score: '',
    winner_team_id: null,
  }
}

// End auto-follows the start (start + the tournament's match duration) until the
// user edits End themselves; then it's left alone. Robust to q-time's progressive
// hour-then-minute updates (recomputes on each, so 09:30 → end reflects :30).
const endTouched = ref(false)
function onStartTimeChange(value: string | number | null) {
  const start = typeof value === 'string' ? value : ''
  // Only once a full HH:MM is entered (not mid-typing) and end isn't user-set.
  if (/^\d{2}:\d{2}$/.test(start) && !endTouched.value) {
    form.value.endTime = addMinutes(start, defaultDuration())
  }
}
function onEndTimeChange() {
  endTouched.value = true
}

const dialog = ref(false)
const editing = ref<Match | null>(null)
const form = ref<Form>(blankForm())
const busy = ref(false)

function openAdd(prefill?: { field_id?: number; time?: string }) {
  editing.value = null
  form.value = blankForm()
  endTouched.value = false
  if (prefill?.field_id != null) form.value.field_id = prefill.field_id
  if (prefill?.time) {
    form.value.date = selectedDay.value || form.value.date
    form.value.time = prefill.time
    form.value.endTime = addMinutes(prefill.time, defaultDuration())
  }
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
    endTime: m.scheduled_at && m.duration_minutes ? addMinutes(m.scheduled_at.slice(11, 16), m.duration_minutes) : '',
    referee_team_id: m.referee_team_id,
    assistant_referee_team_id: m.assistant_referee_team_id,
    a_score: m.a_score?.toString() ?? '',
    b_score: m.b_score?.toString() ?? '',
    winner_team_id: m.winner_team_id,
  }
  endTouched.value = m.duration_minutes != null
  dialog.value = true
}

function buildInput(f: Form): MatchInput {
  let duration: number | null = null
  if (f.time && f.endTime) {
    const diff = parseHM(f.endTime) - parseHM(f.time)
    duration = diff > 0 ? diff : null
  }
  return {
    label: f.label,
    a_team_id: f.a_team_id,
    b_team_id: f.b_team_id,
    field_id: f.field_id,
    scheduled_at: f.date && f.time ? `${f.date}T${f.time}` : null,
    duration_minutes: duration,
    referee_team_id: f.referee_team_id,
    assistant_referee_team_id: f.assistant_referee_team_id,
    a_score: numOrNull(f.a_score),
    b_score: numOrNull(f.b_score),
    winner_team_id: f.winner_team_id,
  }
}

function numOrNull(s: string): number | null {
  const t = s.trim()
  if (t === '') return null
  const n = Number(t)
  return Number.isNaN(n) ? null : n
}

async function submit() {
  busy.value = true
  try {
    const input = buildInput(form.value)
    if (editing.value) await matches.update(editing.value.id, input)
    else await matches.create(input)
    if (matches.error) {
      toast('negative', matches.error)
    } else {
      dialog.value = false
      toast('positive', 'Saved')
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
      toast('negative', matches.error)
    } else {
      dialog.value = false
      toast('positive', 'Deleted')
    }
  })
}

// --- live updates: silently poll every 7s and on tab focus, so a second viewer
// sees another's edits within seconds. Skipped while editing/dragging so the
// user's own interaction isn't disrupted; stores only re-render on real change.
function refresh() {
  if (dialog.value || dragId.value != null) return
  const id = Number(route.params.id)
  if (!id) return
  void matches.fetch(id, true)
  void fields.fetch(id, true)
  void teams.fetch(id, true)
}
function onVisible() {
  if (document.visibilityState === 'visible') refresh()
}
let pollTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  pollTimer = setInterval(refresh, 7000)
  document.addEventListener('visibilitychange', onVisible)
  window.addEventListener('focus', onVisible)
})
onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
  document.removeEventListener('visibilitychange', onVisible)
  window.removeEventListener('focus', onVisible)
})
</script>

<template>
  <q-page padding>
    <div class="row items-center q-mb-md">
      <div class="text-h6">Schedule</div>
      <q-space />
      <q-btn color="primary" icon="add" label="Add match" @click="openAdd()" />
    </div>

    <q-tabs v-if="days.length > 1" v-model="selectedDay" dense no-caps align="left" class="q-mb-sm text-primary">
      <q-tab v-for="d in days" :key="d" :name="d" :label="d" />
    </q-tabs>

    <q-banner v-if="tournament.current && !tournament.current.starts_on" dense class="bg-amber-1 text-amber-9 q-mb-sm">
      <template #avatar><q-icon name="info" color="amber-8" /></template>
      No start date set, so the schedule defaults to today. Set it in
      <router-link :to="{ name: 'settings', params: { id: route.params.id } }" class="text-amber-9">Settings</router-link>.
    </q-banner>

    <q-banner v-if="matches.error && !matches.items.length" class="bg-negative text-white q-mb-md">
      {{ matches.error }}
    </q-banner>

    <!-- Unscheduled strip -->
    <div class="unscheduled" @dragover.prevent @drop="onDropUnscheduled">
      <div class="text-caption text-grey q-mb-xs">Unscheduled (drag onto the calendar, or tap to edit)</div>
      <div v-if="unscheduled.length" class="card-list">
        <div
          v-for="m in unscheduled"
          :key="m.id"
          class="match-card"
          draggable="true"
          @dragstart="onDragStart(m.id, $event)"
          @dragend="dragId = null"
          @click="openEdit(m)"
        >
          <div class="teams">{{ matchTitle(m) }}</div>
          <div class="meta">
            <span v-if="m.scheduled_at">{{ m.scheduled_at.slice(0, 10) }} {{ m.scheduled_at.slice(11, 16) }}</span>
            <span v-else>no time</span>
            <span v-if="!m.field_id"> · no field</span>
          </div>
        </div>
      </div>
      <div v-else class="text-caption text-grey">none</div>
    </div>

    <q-banner v-if="!fieldList.length" class="bg-grey-2 q-my-md">
      Add fields in the <b>Fields</b> section to build the calendar.
    </q-banner>

    <!-- Calendar -->
    <div v-else class="cal-scroll">
      <div class="cal">
        <div class="cal-row cal-head">
          <div class="cal-gutter-cell" />
          <div v-for="f in fieldList" :key="f.id" class="cal-col-head">{{ f.name || `#${f.id}` }}</div>
        </div>
        <div class="cal-row cal-body" :style="{ height: totalHeight + 'px' }">
          <div class="cal-gutter">
            <div v-for="l in gridLines" :key="l.top" class="cal-time" :style="{ top: l.top + 'px' }">{{ l.label }}</div>
          </div>
          <div
            v-for="f in fieldList"
            :key="f.id"
            class="cal-col"
            @click="onColClick($event, f.id)"
            @dragover.prevent
            @drop="onDropCol($event, f.id)"
          >
            <div v-for="l in gridLines" :key="l.top" class="cal-line" :style="{ top: l.top + 'px' }" />
            <div
              v-for="b in fieldBlocks(f.id)"
              :key="b.m.id"
              class="cal-block"
              draggable="true"
              :style="blockStyle(b)"
              @click.stop="openEdit(b.m)"
              @dragstart="onDragStart(b.m.id, $event)"
              @dragend="dragId = null"
            >
              <div class="cb-title">
                {{ matchTitle(b.m) }}
                <span v-if="b.m.a_score != null || b.m.b_score != null" class="cb-score">
                  {{ b.m.a_score ?? '–' }}:{{ b.m.b_score ?? '–' }}
                </span>
              </div>
              <div class="cb-time">{{ matchRange(b.m) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Add / edit dialog -->
    <q-dialog v-model="dialog">
      <q-card style="min-width: 320px; max-width: 95vw">
        <q-card-section class="text-h6">{{ editing ? 'Edit match' : 'New match' }}</q-card-section>
        <q-separator />
        <q-card-section class="q-pt-md" style="max-height: 60vh; overflow-y: auto">
          <div class="dialog-form">
            <q-input v-model="form.label" label="Label (optional)" />
            <q-select v-model="form.a_team_id" :options="teamOptions" label="Team A" emit-value map-options clearable />
            <q-select v-model="form.b_team_id" :options="teamOptions" label="Team B" emit-value map-options clearable />
            <q-select v-model="form.field_id" :options="fieldOptions" label="Field" emit-value map-options clearable />
            <q-input v-model="form.date" label="Date" type="date" stack-label />
            <div class="dialog-row">
              <TimeField v-model="form.time" label="Start time" @update:model-value="onStartTimeChange" />
              <TimeField v-model="form.endTime" label="End time" @update:model-value="onEndTimeChange" />
            </div>
            <q-select v-model="form.referee_team_id" :options="teamOptions" label="Referee team" emit-value map-options clearable />
            <q-select v-model="form.assistant_referee_team_id" :options="teamOptions" label="Assistant referee" emit-value map-options clearable />
            <q-separator class="q-my-xs" />
            <div class="text-caption text-grey">Result</div>
            <div class="dialog-row">
              <q-input v-model="form.a_score" :label="'Score ' + teamName(form.a_team_id)" type="number" />
              <q-input v-model="form.b_score" :label="'Score ' + teamName(form.b_team_id)" type="number" />
            </div>
            <q-select v-model="form.winner_team_id" :options="winnerOptions" label="Winner" emit-value map-options clearable />
          </div>
        </q-card-section>
        <q-separator />
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
.unscheduled {
  border: 1px dashed #bbb;
  border-radius: 4px;
  padding: 8px;
  margin-bottom: 12px;
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
  cursor: pointer;
  font-size: 12px;
  line-height: 1.3;
}
.match-card .teams {
  font-weight: 500;
}
.match-card .meta {
  color: #666;
  font-size: 11px;
}

.cal-scroll {
  overflow: auto;
  max-height: calc(100vh - 210px);
  border: 1px solid #e0e0e0;
  border-radius: 4px;
}
.cal {
  min-width: fit-content;
}
.cal-row {
  display: flex;
}
.cal-head {
  position: sticky;
  top: 0;
  z-index: 3;
  background: var(--q-primary);
  color: #fff;
}
.cal-gutter-cell {
  flex: 0 0 52px;
}
.cal-col-head {
  flex: 1 0 132px;
  padding: 6px 8px;
  font-weight: 500;
  white-space: nowrap;
  border-left: 1px solid rgba(255, 255, 255, 0.3);
}
.cal-gutter {
  flex: 0 0 52px;
  position: relative;
}
.cal-time {
  position: absolute;
  right: 5px;
  transform: translateY(-50%);
  font-size: 11px;
  color: #999;
  font-variant-numeric: tabular-nums;
}
.cal-col {
  flex: 1 0 132px;
  position: relative;
  border-left: 1px solid #eee;
}
.cal-line {
  position: absolute;
  left: 0;
  right: 0;
  border-top: 1px solid #f2f2f2;
  pointer-events: none;
}
.cal-block {
  position: absolute;
  background: #d7ebf5;
  border: 1px solid var(--q-primary);
  border-left: 3px solid var(--q-primary);
  border-radius: 3px;
  padding: 1px 4px;
  font-size: 11px;
  line-height: 1.25;
  overflow: hidden;
  cursor: pointer;
  box-sizing: border-box;
}
.cal-block .cb-title {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cal-block .cb-time {
  color: #557;
  font-size: 10px;
}
.cal-block .cb-score {
  font-weight: 700;
  color: #024;
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
  flex: 1 1 130px;
}
</style>
