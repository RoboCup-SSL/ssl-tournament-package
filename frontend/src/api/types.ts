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
  created_at: string
}
