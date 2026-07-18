// Pinia store for a single tournament being edited (the Settings section). Holds
// the loaded row plus an editable string draft, with dirty-tracking and save.
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import type { Tournament, TournamentInput } from '@/api/types'

// Draft mirrors the form fields as strings; an empty string means "cleared".
interface Draft {
  name: string
  location: string
  starts_on: string
  ends_on: string
  venue_opens: string
  venue_closes: string
  default_match_minutes: string
  default_gap_minutes: string
  time_zone: string
}

// toDraft converts a loaded tournament into an editable string draft.
function toDraft(t: Tournament): Draft {
  return {
    name: t.name,
    location: t.location,
    starts_on: t.starts_on ?? '',
    ends_on: t.ends_on ?? '',
    venue_opens: t.venue_opens ?? '',
    venue_closes: t.venue_closes ?? '',
    default_match_minutes: t.default_match_minutes?.toString() ?? '',
    default_gap_minutes: t.default_gap_minutes?.toString() ?? '',
    time_zone: t.time_zone ?? '',
  }
}

// buildInput maps a draft to a PATCH body: empty nullable fields become null,
// minutes become numbers, name/location stay strings.
function buildInput(d: Draft): TournamentInput {
  const str = (s: string): string | null => (s.trim() === '' ? null : s)
  const num = (s: string): number | null => {
    if (s.trim() === '') return null
    const parsed = Number(s)
    return Number.isNaN(parsed) ? null : parsed
  }
  return {
    name: d.name,
    location: d.location,
    starts_on: str(d.starts_on),
    ends_on: str(d.ends_on),
    venue_opens: str(d.venue_opens),
    venue_closes: str(d.venue_closes),
    default_match_minutes: num(d.default_match_minutes),
    default_gap_minutes: num(d.default_gap_minutes),
    time_zone: str(d.time_zone),
  }
}

export const useTournamentStore = defineStore('tournament', {
  state: () => ({
    current: null as Tournament | null,
    draft: null as Draft | null,
    loading: false,
    saving: false,
    error: '',
  }),
  getters: {
    // dirty reports whether the draft differs from the loaded tournament.
    dirty(state): boolean {
      if (!state.current || !state.draft) return false
      return JSON.stringify(state.draft) !== JSON.stringify(toDraft(state.current))
    },
  },
  actions: {
    // load fetches one tournament and seeds the draft.
    async load(id: number) {
      this.loading = true
      this.error = ''
      // Clear stale data so a slow/failed load never shows the previous row.
      this.current = null
      this.draft = null
      try {
        this.current = await api.get<Tournament>(`/api/tournaments/${id}`)
        this.draft = toDraft(this.current)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      } finally {
        this.loading = false
      }
    },
    // save PATCHes the draft; on success it resyncs current + draft from the
    // response (reflecting server normalization) and clears dirty.
    async save() {
      if (!this.current || !this.draft) return
      this.saving = true
      this.error = ''
      try {
        this.current = await api.patch<Tournament>(
          `/api/tournaments/${this.current.id}`,
          buildInput(this.draft),
        )
        this.draft = toDraft(this.current)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      } finally {
        this.saving = false
      }
    },
    // discard reverts the draft to the loaded values.
    discard() {
      if (this.current) this.draft = toDraft(this.current)
    },
  },
})
