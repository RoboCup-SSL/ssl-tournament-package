// Hand-written API response types for the shell. OpenAPI codegen is deferred.

export interface Version {
  version: string
}

export interface Tournament {
  id: number
  name: string
  location: string
  starts_on: string | null
  ends_on: string | null
  venue_opens: string | null
  venue_closes: string | null
  default_match_minutes: number | null
  default_gap_minutes: number | null
  time_zone: string | null
  created_at: string
}

// Writable create/patch body. Omitted keys are left unchanged; null clears a
// nullable field. The server validates and normalizes date/time/zone values.
export interface TournamentInput {
  name?: string
  location?: string
  starts_on?: string | null
  ends_on?: string | null
  venue_opens?: string | null
  venue_closes?: string | null
  default_match_minutes?: number | null
  default_gap_minutes?: number | null
  time_zone?: string | null
}

export interface Field {
  id: number
  tournament_id: number
  name: string
}

export interface FieldInput {
  name?: string
}

export interface Team {
  id: number
  tournament_id: number
  division_id: number | null
  name: string
  country: string
  contact: string
  notes: string
  withdrawn_at: string | null
  created_at: string
}

// Writable team fields; the M2d form uses name/country/contact/notes.
export interface TeamInput {
  name?: string
  country?: string
  contact?: string
  notes?: string
  division_id?: number | null
  withdrawn_at?: string | null
}

export const MATCH_STATUSES = [
  'scheduled', 'playing', 'suspended', 'finished', 'forfeited', 'cancelled', 'invalidated',
] as const

export interface Match {
  id: number
  tournament_id: number
  division_id: number | null
  group_id: number | null
  label: string
  field_id: number | null
  scheduled_at: string | null
  duration_minutes: number | null
  status: string
  referee_team_id: number | null
  assistant_referee_team_id: number | null
  a_team_id: number | null
  a_score: number | null
  b_team_id: number | null
  b_score: number | null
  winner_team_id: number | null
  notes: string
  created_at: string
}

// Writable match fields the schedule edits (results/bracket-wiring excluded).
export interface MatchInput {
  label?: string
  field_id?: number | null
  scheduled_at?: string | null
  status?: string
  referee_team_id?: number | null
  assistant_referee_team_id?: number | null
  a_team_id?: number | null
  b_team_id?: number | null
}
