# M2c — tournament workspace + Settings section — design

Status: approved (design)
Date: 2026-07-18
Follows: M2b frontend shell (2026-07-17), M2b·dt datetime validation + timezone (2026-07-18)

## Purpose

Turn the read-only M2b shell into a navigable, editable workspace. This is the
first slice with **write** operations and the first visible payoff of the
timezone work: create a tournament and edit its settings (including the
timezone). It proves the per-entity *section pattern* that M2d+ (Fields, Teams,
…) will replicate.

## Scope

In scope:
- **Create** a tournament from the UI (minimal: name only), landing in its Settings.
- A tournament-scoped **workspace**: a section nav (Quasar `q-drawer`,
  auto-collapsing to a menu on mobile) listing all future sections, only
  **Settings** enabled.
- A **Settings** form editing the Tournament fields, with an explicit **Save**
  (dirty-tracked `PATCH`) and **Discard**, backed by the M2b·dt validation.
- A **timezone picker** over the browser's IANA zone list.

Out of scope (later slices): Fields/Teams/etc. sections; delete/withdraw;
per-field error highlighting (show the server message); Vitest (deferred).

## Routing (hash)

- `/#/` → `HomeView` — tournament list + **New tournament**.
- `/#/t/:id/settings` → `SettingsView` (the form).
- `/#/t/:id` → redirect to `…/settings`.

The workspace chrome (section drawer + header showing the tournament name) lives
in **`App.vue`**, shown only when the route carries a `:id`. Section views render
directly into the app's `q-page-container` — no separate layout component, no
nested `q-layout`.

## Files

- `src/api/types.ts` — add `TournamentInput` (writable create/patch body).
- `src/store/tournaments.ts` — add `create(name)` (POST, refresh list, return row).
- `src/store/tournament.ts` — **new** singular store: `current`, `draft`,
  `loading`/`saving`/`error`, getter `dirty`, actions `load(id)`, `save()`,
  `discard()`.
- `src/router/index.ts` — add the settings route + `/t/:id` redirect.
- `src/App.vue` — host the workspace drawer/header; load the tournament on
  `:id` change; keep the global title + admin link on `/`.
- `src/views/HomeView.vue` — New-tournament dialog; rows navigate to Settings;
  show `time_zone` in the caption.
- `src/views/tournament/SettingsView.vue` — **new** the form.

## Store / save model

`draft` mirrors the form as strings (empty = cleared). `dirty` compares `draft`
to `toDraft(current)`. `save()` builds a `TournamentInput` via a pure
`buildInput(draft)`:
- `name`/`location` → string (may be `""`),
- date/time/zone fields → `str(v)`: empty → `null`, else the string,
- `default_match_minutes`/`default_gap_minutes` → `num(v)`: empty → `null`, else `Number(v)`.

`save()` `PATCH`es, resyncs `current`+`draft` from the response (so normalized
values like `8:00`→`08:00` reflect back), clears `dirty`. On a validation `400`
the store surfaces the server message (e.g. "invalid starts_on"); Save stays
enabled to retry. `discard()` reverts `draft` to `current`.

## Timezone picker

Options come from `Intl.supportedValuesOf('timeZone')` — the browser's full IANA
list, no dependency, no data file. A filterable `q-select` (`use-input`,
`clearable`). The backend (`validZone`) is the real guard, so any string is
safe; the picker is convenience. Empty selection → `null`.

## Form fields (Quasar inputs)

| Field | Input |
|---|---|
| name, location | `q-input` text |
| starts_on, ends_on | `q-input type="date"` (emits `YYYY-MM-DD`) |
| venue_opens, venue_closes | `q-input type="time"` (emits `HH:MM`) |
| default_match_minutes, default_gap_minutes | `q-input type="number"` |
| time_zone | filterable `q-select` (IANA) |

Native date/time inputs emit exactly the canonical formats the backend expects,
so round-trips are clean. No client-side value rules (freeform); only the
server's format validation applies.

## Verification

`npm run build` (vue-tsc type-check + vite) clean; `go test ./...` unaffected;
manual e2e on the hosted binary: create a tournament → lands in Settings → edit
fields + pick a timezone → Save persists (reload shows it) → enter a bad value
via the raw admin still rejected by the backend. dist rebuilt + committed
(option A).

## Follow-ups

M2d (Fields + Teams sections, same pattern) → M2e schedule view (MVP). Mobile
polish of the drawer, per-field error highlighting, and Vitest for
`buildInput`/`dirty` when logic grows.
