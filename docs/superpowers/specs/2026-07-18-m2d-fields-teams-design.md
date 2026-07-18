# M2d — Fields + Teams sections — design

Status: approved (design)
Date: 2026-07-18
Follows: M2c workspace + Settings (2026-07-18)

## Purpose

Add the first two **list** sections to the tournament workspace — Fields and
Teams — the entities a match references. This establishes the reusable
**list-of-child-entities CRUD** pattern (add/edit/delete rows, filtered by
tournament) that every later section (Matches, Divisions, Groups, …) follows.
M2c's Settings edited the single tournament; this edits collections.

## Scope

In scope:
- **Fields** section: list a tournament's fields; add / rename / delete.
- **Teams** section: list a tournament's teams; add / edit (name, country,
  contact, notes) / delete.
- Enable both in the workspace section nav.
- App-wide **Dialog + Notify** Quasar plugins for delete-confirm and save/error
  toasts (nicer than the current inline banners; retrofitting other views is a
  follow-up).

Out of scope (later): `division_id` on teams (needs the Divisions section),
`withdrawn_at` withdraw action, bulk import, sorting/paging, per-field inline
editing.

## Routing

Add under the existing workspace:
- `/#/t/:id/fields` → `FieldsView`
- `/#/t/:id/teams` → `TeamsView`

`App.vue`'s section list flips `fields` and `teams` to `enabled: true` (their
nav items already exist, disabled). The `:to` already targets `{ name, params: { id } }`.

## The reusable list-CRUD pattern

**Store** (one per entity, Pinia, following [[api-style]] — explicit, not generic):
```
state: { items: T[], tournamentId: number, loading: boolean, error: string }
fetch(tournamentId)   GET /api/<coll>?tournament_id=ID ; records tournamentId
create(input)         POST /api/<coll> { tournament_id: state.tournamentId, ...input } ; then fetch()
update(id, input)     PATCH /api/<coll>/{id} ; then fetch()
remove(id)            DELETE /api/<coll>/{id} ; then fetch()
```
Mutations re-`fetch()` (small lists; keeps server ordering). Errors are captured
into `error` for the view to toast. `create` injects `tournament_id` from state
(set by the preceding `fetch`).

**View** (one per entity):
- `watch(route.params.id, () => store.fetch(Number(id)), { immediate: true })`.
- A `q-list` of rows; each row has edit + delete buttons.
- An **Add** button and a single **dialog** reused for add *and* edit (a
  `editing` ref: null = add, else the row). Submit calls `create`/`update`.
- Delete via `$q.dialog({ cancel: true }).onOk(() => store.remove(id))`.
- `$q.notify` positive on success, negative on `store.error`.

## Entity shapes

- **Field**: `{ id, tournament_id, name }`. `FieldInput = { name? }`. Dialog: one
  Name input.
- **Team**: `{ id, tournament_id, division_id, name, country, contact, notes,
  withdrawn_at, created_at }`. `TeamInput = { name?, country?, contact?, notes?,
  division_id?, withdrawn_at? }` (type carries all writable fields; the M2d form
  uses name/country/contact/notes only). Row caption shows country if set.

Freeform: no client-side required-field rules beyond a non-empty name to enable
Create/Save; the server owns validation.

## Files

- `src/api/types.ts` — add `Field`, `FieldInput`, `Team`, `TeamInput`.
- `src/store/fields.ts`, `src/store/teams.ts` — **new**, the pattern above.
- `src/views/tournament/FieldsView.vue`, `TeamsView.vue` — **new**.
- `src/router/index.ts` — add the two routes.
- `src/App.vue` — enable `fields` + `teams` nav items.
- `src/main.ts` — register `Dialog` + `Notify` plugins.

## Verification

`npm run build` (vue-tsc + vite) clean; manual e2e on the hosted binary: open a
tournament → Fields: add "Field A"/"Field B", rename, delete → Teams: add a team
with country/contact, edit, delete → confirm rows persist across reload and the
delete-confirm + toasts work. dist rebuilt + committed.

## Follow-ups

M2e — the schedule view (the MVP): matches on a field × time grid, referencing
the fields and teams created here. Then division_id/withdraw, and retrofitting
Settings to the toast pattern.
