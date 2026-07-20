# M2e — schedule grid (the MVP) — design

Status: approved (design)
Date: 2026-07-18
Follows: M2d Fields + Teams (2026-07-18)

## Purpose

The MVP: **replace the schedule Excel sheet.** A time × field grid where every
match is visible in its cell, you drag a match to move it (desktop), and tap a
match to edit it (works everywhere). Freeform — schedule any match, any time, any
field, no restrictions — exactly the Excel use case. Matches reference the Fields
and Teams from M2d.

## Scope

In scope (the *schedulable* match): two teams (direct), field, time (naive
local), referee + assistant teams, label, status. Create / edit / delete / move.

Out of scope: scores, fouls, cards, winner (results → M4); `a_source`/`b_source`
elimination bracket wiring (→ M3); division/group assignment (Groups section not
built); touch/pointer drag (HTML5 drag is desktop; mobile moves via the edit
dialog — pointer-drag is a follow-up).

## The grid

- **Columns** = the tournament's fields. **Rows** = time slots for a selected
  **day**. Plus an **Unscheduled** strip for matches missing a field or time.
- **Days**: distinct date parts of matches' `scheduled_at`, unioned with
  `starts_on..ends_on` if set; default `[starts_on || today]`. A day selector
  (`q-tabs`/`q-select`) when there's more than one.
- **Time slots** for the day: a ladder from `venue_opens`..`venue_closes`
  (defaults `08:00`..`20:00`) stepped by `default_match_minutes +
  default_gap_minutes` (default 60), **unioned** with the exact times of that
  day's matches (so nothing is ever hidden). Sorted unique `HH:MM`.
- A **cell** `(field, time)` holds matches where `field_id == field.id` and
  `scheduled_at == "<day>T<time>"` (exact string; the backend normalizes to
  minute precision). A **card** shows `A vs B`, a ref badge, and a status color.

## Interactions

- **Drag (desktop)**: card is `draggable`; cells + the Unscheduled strip are drop
  targets. Drop → `PATCH { field_id, scheduled_at }` to the target cell's field +
  `"<day>T<time>"` (or `scheduled_at: null` when dropped on Unscheduled).
- **Tap/click a card** → edit dialog (this is how you move on mobile: change the
  Field and Date/Time selects). **Add match** button → same dialog, blank.
- **Delete** from the dialog, with a `$q.dialog` confirm.
- Feedback via `$q.notify` toasts (bottom), consistent with M2c/M2d.

## Edit dialog fields

label · team A · team B · field · date + time · referee · assistant · status.
Team/field/ref are clearable `q-select`s (options from the teams/fields stores;
`emit-value map-options`, value = id, empty = null). Date+time combine to
`scheduled_at = "YYYY-MM-DDTHH:MM"`, or `null` if either is empty (→ Unscheduled).
Status is the enum (`scheduled` default).

## Files

- `src/api/types.ts` — add `Match`, `MatchInput`, `MATCH_STATUSES`.
- `src/store/matches.ts` — **new**, list-CRUD (same pattern as fields/teams).
- `src/views/tournament/ScheduleView.vue` — **new**, the grid + dialog + drag +
  day/slot logic (small inline time helpers: `parseHM`/`fmtHM`).
- `src/router/index.ts` — add `/t/:id/matches` → ScheduleView.
- `src/App.vue` — enable the `matches` nav item (label it "Schedule").

The view loads three stores on the tournament id: matches, fields, teams (fields
= columns, teams = names + dropdowns). Tournament settings (venue hours,
defaults, dates) come from the existing `tournament` store.

## Verification

`npm run build` clean; manual e2e on the hosted binary: add fields + teams, add a
match (pick teams/field/time/ref) → it lands in the right cell; drag it to
another cell → time/field update and persist; tap → edit → move via dialog;
delete. Reload persists. dist rebuilt + committed.

## Follow-ups

Pointer/touch drag for mobile; results entry (M4); elimination bracket wiring
(M3); per-cell "add match here" affordance; conflict highlighting (catch-and-warn).
