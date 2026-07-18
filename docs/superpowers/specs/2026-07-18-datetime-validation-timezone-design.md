# Date/time validation + tournament timezone — design

Status: approved (design)
Date: 2026-07-18
Milestone: M2 data-layer slice — precursor to M2c (workspace + Settings)
Follows: M2a CRUD API (docs/superpowers/specs/2026-07-14-m2a-crud-api-design.md)

## Purpose

Close a data-integrity gap: every date/time field in the model is stored as raw
`TEXT` with **no format validation**, so the API today accepts `starts_on:
"banana"`, `venue_opens: "25:99"`, or `scheduled_at: "2026-13-40"` and stores
them silently. Add server-side **format** validation for these fields, and give
the model a single, unambiguous timezone anchor per tournament.

This is a backend/data-layer slice. It unblocks M2c's Settings form (which edits
these fields) and every later schedule/ICS feature that must sort and render
times correctly.

## Background: freeform *meaning*, not freeform *shape*

The data-model design (2026-07-10) draws the line this slice enforces: the DB
guarantees *types and shape* — "it rejects a string where a number belongs;
that's not freedom, that's unreadable data" — while refusing to enforce
*tournament rules* (uniqueness, scheduling constraints, bracket completeness).

A malformed date is unreadable data, not freedom. So we validate that a value is
a **well-formed** date / time / datetime, and reject nothing about its *meaning*:
a 3 a.m. match, `venue_opens` after `venue_closes`, `ends_on` before `starts_on`,
times outside the venue window — all remain legal. Only malformed input is
rejected.

## Timezone model: single anchor, backend stays naive

Decision (organizer-facing simplicity): timezones are hard for people, so they
interact with them **once**.

- **One `time_zone` on the tournament** (IANA name, e.g. `Asia/Seoul`) is the
  single source of truth. The backend's only jobs with it are to **store it** and
  **validate it is a real IANA zone**.
- **The backend is timezone-naive.** Every scheduling instant is stored as a
  **bare local datetime with no offset** (`2026-07-15T14:00`). All matches and
  bookings are relative to each other in one implicit zone, so ordering and
  comparison need zero timezone math. The backend performs **no** timezone
  conversion.
- **The frontend owns all timezone behavior.** It loads the tournament's
  `time_zone` once (into a Pinia store) and uses it only for display/labeling
  ("times shown in Asia/Seoul") and, later, ICS export. Users never type an
  offset.
- **A mistake is a one-field fix.** Wrong zone → edit the single `time_zone`
  field; every displayed time re-interprets, no stored data migrates.

Deliberate exception: `created_at` is a **system/audit** timestamp (server-set,
UTC `Z`), not a scheduling field — it stays UTC and is not touched here. Rule of
thumb: **domain scheduling times = naive local; system audit times = UTC.**

## Scope

In scope:
- A new nullable `time_zone` column on `tournament`, validated as an IANA zone.
- Server-side format validation, applied at write time (create + update), for
  every client-writable date/time field across the model.
- Format normalization to canonical form on write.
- OpenAPI schema updated with `time_zone` and format hints/descriptions.
- `time/tzdata` embedded so IANA validation works on any host.

Out of scope (explicit non-goals):
- Any timezone *conversion* or offset math in the backend.
- Value/business rules (windows, ordering, cross-field checks) — still freeform;
  advisory "catch-and-warn" checks are a later slice.
- Frontend work (the Pinia tz store, the Settings form) — that is M2c.
- Changing `created_at` or any server-set timestamp.
- Offsets/`Z` on scheduling instants — those are **rejected** (the zone lives on
  the tournament, not the value).

## Field inventory and formats

| Field(s) | Entity | Kind | Canonical format | Go layout |
|---|---|---|---|---|
| `starts_on`, `ends_on` | Tournament | date | `2006-01-02` | `2006-01-02` |
| `venue_opens`, `venue_closes` | Tournament | wall-clock time | `15:04` | `15:04` |
| `time_zone` | Tournament | IANA zone | zone name | `time.LoadLocation` |
| `scheduled_at` | Match | naive datetime | `2006-01-02T15:04` | see below |
| `starts_at`, `ends_at` | FieldBooking | naive datetime | `2006-01-02T15:04` | see below |
| `withdrawn_at` | Team | naive datetime | `2006-01-02T15:04` | see below |

Naive-datetime parsing accepts `2006-01-02T15:04` and `2006-01-02T15:04:05`
(optional seconds), and **rejects any offset or `Z`** (an offset-bearing string
fails the naive layout, which is the desired rejection).

The `*_minutes` fields are already `integer`-typed by the API and need no change.

## Validation behavior (uniform across all fields above)

For each field, at create and update:
- **Absent** (PATCH key omitted) → leave unchanged.
- **`null`** → clear the column (all these fields are nullable).
- **Empty string `""`** → treated as `null` (clears the column). An empty value
  unambiguously means "not set"; rejecting it would be pedantic and hurts the
  raw-JSON admin UX.
- **Non-empty value** → parse with the field's layout. On failure return
  `Error{Code: INVALID_VALUE, Field: <json name>, Message: <what was expected>}`.
  On success **store the canonical re-formatted value** (parse → `Format` →
  store), so storage is normalized and sortable (e.g. `8:00`→`08:00`,
  `2026-7-5`→`2026-07-05`). For `time_zone` the validated name is stored as-is.

Values are otherwise unrestricted — no window, ordering, or cross-field checks.

## Implementation

### Store (`internal/store`)

- **Migration** `internal/store/migrations/0002_tournament_timezone.sql`:
  ```sql
  ALTER TABLE tournament ADD COLUMN time_zone TEXT;  -- IANA zone, nullable
  ```
  The runner (`migrate.go`) applies numbered files in filename order past
  `PRAGMA user_version`, so `0002_*` runs once on existing databases.
- **`Tournament` struct** (`store/tournament.go`): add `TimeZone *string
  \`json:"time_zone"\`` (nullable pointer, placed after `DefaultGapMinutes`).
- **SQL** — add `time_zone` to all four statements, in the same position:
  `CreateTournament` INSERT column list + placeholder + arg; `GetTournament`
  SELECT + `Scan`; `ListTournaments` SELECT + `Scan`; `UpdateTournament`
  `SET time_zone=?` + arg. (Column count goes 8→9 for tournament writes.)

### API (`internal/api`)

- **`TournamentPatch`** (`api/tournament.go`): add
  `TimeZone Opt[string] \`json:"time_zone"\``.
- **New file `api/datetime.go`** — pure validators returning the canonical string
  or an error, with the exact messages the apply layer wraps:
  ```go
  // Package-level validators for the model's date/time text fields. Each parses
  // its layout, rejecting malformed input, and returns the canonical form.
  func validDate(value string) (string, bool)      // "2006-01-02"
  func validClock(value string) (string, bool)     // "15:04"
  func validNaiveTime(value string) (string, bool) // "2006-01-02T15:04"[:05], no offset
  func validZone(value string) (string, bool)      // time.LoadLocation
  ```
  `validNaiveTime` tries `2006-01-02T15:04:05` then `2006-01-02T15:04`, and
  returns canonical `2006-01-02T15:04`.
- **New helper** (`api/opt.go`, beside `applyNullable`):
  ```go
  // applyNullableString applies a nullable text field, coercing "" to null and
  // running check on non-empty values; check returns the canonical form.
  func applyNullableString(field Opt[string], target **string, name string,
      check func(string) (string, bool)) *Error
  ```
  Semantics: not `Set` → return nil; `Null` or value `""` → `*target = nil`;
  else `canonical, ok := check(value)`; `!ok` →
  `&Error{Code: CodeInvalidValue, Field: name, Message: ...}`; ok →
  `*target = &canonical`.
- **Wire the validators** into each entity's `applyXPatch`, replacing the plain
  `applyNullable` call for the relevant field:
  - `applyTournamentPatch`: `starts_on`/`ends_on` → `validDate`;
    `venue_opens`/`venue_closes` → `validClock`; `time_zone` → `validZone`.
  - `applyFieldBookingPatch`: `starts_at`/`ends_at` → `validNaiveTime`.
  - `applyTeamPatch`: `withdrawn_at` → `validNaiveTime`.
  - `applyMatchPatch`: `scheduled_at` → `validNaiveTime`.
  Each returns its `*Error` up the existing `applyError` chain (create + update
  already propagate it).

### tzdata

Add a blank import `_ "time/tzdata"` in both command mains
(`cmd/ssl-tournament/main.go`, `cmd/ssl-tournament-service/main.go`) so the zone
database is embedded in the static (CGO-off) binaries — otherwise
`time.LoadLocation` depends on host tzdata and could reject valid zones on
minimal hosts.

### OpenAPI (`internal/server/openapi.yaml`)

- Add `time_zone` to the `Tournament` schema: `type: string, nullable: true`,
  `description: "IANA timezone, e.g. Asia/Seoul; the single anchor for
  interpreting this tournament's local times"`.
- Add descriptions/format hints (documentation only; validation is server-side):
  `starts_on`/`ends_on` → `format: date`; `scheduled_at`/`starts_at`/`ends_at`/
  `withdrawn_at` → `description: "naive local datetime YYYY-MM-DDTHH:MM (no
  offset; interpreted in the tournament time_zone)"`; `venue_opens`/
  `venue_closes` keep `HH:MM local`.
- The route-sync test (`internal/server`) checks routes, not schema fields, so it
  is unaffected.

## Testing

- **`api/datetime_test.go`** — table tests per validator: valid canonical
  round-trips (incl. normalization `8:00`→`08:00`, `2026-7-5`→`2026-07-05`,
  seconds dropped), and rejections (`banana`, `25:99`, `2026-13-40`,
  offset/`Z` on naive fields, `Mars/Phobos` zone).
- **`api/*_test.go`** — for each of the four apply paths: absent leaves,
  `null` clears, `""` clears, valid value normalizes-and-sets, malformed →
  `INVALID_VALUE` with the right `Field`.
- **HTTP-level** (`internal/server`) — one PATCH test per representative field
  (e.g. `PATCH /api/tournaments/{id}` with bad `starts_on` → 400 envelope with
  `field: "starts_on"`; with a good value → 200 normalized; a real `time_zone`
  → 200; a bogus zone → 400).
- **Migration** — a store test that `0002` applies and `time_zone` round-trips
  (set, read back, clear).
- `go test ./...`, `go vet ./...` green.

## What stays freeform

Everything about *values*: any date/time, any zone, out-of-window times,
open-after-close, ends-before-starts, withdrawals in the past or future. Only
malformed shape and non-IANA zones are rejected. Advisory warnings ("this match
is outside venue hours") are a later catch-and-warn slice, never hard rejections.

## Frontend note (consumed by M2c, not built here)

M2c's Settings form gains a `time_zone` picker and naive date/time inputs, and a
Pinia store holds the current tournament's zone for display. The frontend sends
naive datetimes (no offset); the backend validates and normalizes them. None of
that is in this slice.

## Follow-ups

- M2c — tournament workspace + Settings section (consumes this).
- Advisory "catch-and-warn" validation (venue-window, ordering) — later.
- ICS export (M2f) reads `time_zone` to emit correct calendar times.
