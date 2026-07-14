# M2a — JSON CRUD API design

Status: settled 2026-07-14. Builds on the data-model spec
(2026-07-10-tournament-data-model-design.md). **No schema changes** — M2a adds no
migrations; field names below reference the existing M1 schema
(`internal/store/migrations/0001_init.sql`) as-is.

## Goal

JSON CRUD endpoints over `net/http` for the 8 planning-domain resources, layered
`server → api → store` so a future MCP server can reuse the api layer
(structure modeled on ~/projects/tdp_rust: web/mcp transports → api crate → data_access;
`internal/api` is the application's operation set, `internal/server` is one transport
for it).

Deferred: `event`, `user`, `token` endpoints (M4/M5, where their semantics live).

## Layering

Dependency direction: `server → api → store`. Nothing upward.

```
internal/
  store/                  # all SQL; one file per table; nothing above it sees SQL or driver errors
    store.go              # (exists) Open/Close/Ping + new inTx(func(*sql.Tx) error) helper
    errors.go             # ErrNotFound (moved from tournament.go); ConstraintError{Kind, Detail}
                          #   Kind ∈ {ForeignKey, Check, NotNull}; translate(err) inspects
                          #   modernc.org/sqlite extended error codes — driver knowledge stays here
    tournament.go         # (exists) gains UpdateTournament; List order changes to id ASC
    division.go, team.go, field.go, booking.go, group.go, match.go, placement.go
                          # per entity: JSON-tagged row struct + Create/Get/List/Update/Delete
                          # group.go and match.go run their multi-table writes in one tx
  api/                    # transport-free middle layer: the application's operation set;
                          # the future MCP seam. Like tdp_rust's api crate: NO central
                          # struct — free functions taking *store.Store as first argument
    errors.go             # type Error{ Code, Message, Field string }; code constants; fromStore(err)
    opt.go                # Opt[T]: the PATCH tri-state (absent / null / value)
    tournament.go … placement.go
                          # per entity: Patch struct and
                          #   ListTeams(dataStore, store.TeamFilter) / CreateTeam(dataStore, Patch) /
                          #   GetTeam(dataStore, id) / UpdateTeam(dataStore, id, Patch) /
                          #   DeleteTeam(dataStore, id)  (etc. per entity)
                          # Filter structs live in store (they parameterize its SQL);
                          # api passes them through
                          # match.go + group.go define the composite JSON views
  server/                 # HTTP transport only; handlers contain no logic
    server.go             # (exists) NewMux(st *store.Store) unchanged; Run(...)
    routes.go             # every route registration, Go 1.22 method patterns
    respond.go            # decode() (strict JSON), writeJSON(), writeError() (code → HTTP status)
    openapi.yaml          # hand-written OpenAPI 3 spec (embedded; see API docs section)
    tournament.go … placement.go   # one file per resource, 5 thin handlers each
```

No interfaces (one concrete store — a consumer can define its own interface later).
No central struct in `api` (tdp_rust convention: operations are free functions; nothing
accretes into a god object as M3–M5 add operations). No DTO package: store row structs
carry the JSON tags and are returned directly; only match and group get view structs in
`api` because their wire shape spans tables.

## API conventions

Uniform across all 8 resources:

```
GET    /api/<res>            list, ORDER BY id ASC; filters via query params
POST   /api/<res>            create → 201 + full object
GET    /api/<res>/{id}       → 200 + object, or 404
PATCH  /api/<res>/{id}       partial update → 200 + full updated object
DELETE /api/<res>/{id}       → 204, or 404 (cascades per schema ON DELETE rules)
```

Resource path names: `tournaments`, `divisions`, `teams`, `fields`, `field-bookings`,
`groups`, `matches`, `placements`.

- JSON keys are snake_case and match column names, except the composite fields below.
- `id` and `created_at` are server-set and read-only: they are absent from Patch
  structs, so sending them is an UNKNOWN_FIELD error.
- **PATCH tri-state:** key absent = leave unchanged; key `null` = set NULL (or delete
  the sub-row, for composites); key with value = set. Implemented once as
  `Opt[T] struct{ Set, Null bool; Value T }` with a custom `UnmarshalJSON`
  (`json` only calls it when the key is present; `null` sets both flags).
- **POST** decodes into the same Patch struct applied to a zero row. The `api` layer fills
  the schema's non-trivial defaults for absent fields (`field_booking.kind = "booking"`,
  `match.status = "scheduled"`); NOT NULL text columns default to `""` (Go zero value).
  INSERT/UPDATE statements are static, name every column except `id`/`created_at`
  (the DB fills `created_at`). No dynamic SQL.
- Unknown body fields are rejected (`DisallowUnknownFields`). Unknown query
  parameters are rejected the same way (UNKNOWN_FIELD) on list endpoints — typos
  must not silently return unfiltered lists. Known accepted limitation: strict
  rejection applies to top-level keys only; unknown keys nested inside composite
  sub-objects (`a_source`/`b_source`, `ranking` entries) are silently ignored,
  because Go's strict decoding does not reach inside custom unmarshalers.
- A non-integer `{id}` path value or filter value is INVALID_VALUE.

### List filters

| Resource | Query params (all optional, AND-ed) |
|---|---|
| tournaments | — |
| divisions | tournament_id |
| teams | tournament_id, division_id |
| fields | tournament_id |
| field-bookings | tournament_id, field_id, team_id |
| groups | tournament_id, division_id |
| matches | tournament_id, division_id, group_id, field_id |
| placements | tournament_id, division_id |

## Error model

One envelope everywhere; `field` is present when the error is attributable to one field:

```json
{"error": {"code": "MISSING_REFERENCE", "message": "division 3 does not exist", "field": "division_id"}}
```

| Code | HTTP | When |
|---|---|---|
| INVALID_JSON | 400 | body is not valid JSON |
| UNKNOWN_FIELD | 400 | unknown body key or query param (`field` = the key) |
| INVALID_VALUE | 400 | wrong JSON type, bad enum (CHECK), NOT NULL violation, bad id/filter syntax |
| NOT_FOUND | 404 | no row with that id |
| MISSING_REFERENCE | 409 | an FK points at a nonexistent row |
| INTERNAL | 500 | anything else |

Mapping: `store.ErrNotFound → NOT_FOUND`; `ConstraintError{ForeignKey} →
MISSING_REFERENCE`; `ConstraintError{Check|NotNull} → INVALID_VALUE` (message carries
SQLite's constraint text, e.g. `NOT NULL constraint failed: team.tournament_id`;
`field` is set when derivable from that text). Decode errors map in `respond.go`:
`json.SyntaxError → INVALID_JSON`, `json.UnmarshalTypeError → INVALID_VALUE` + field,
unknown-field error → UNKNOWN_FIELD + field.

## Freeform guarantee

The API rejects **only** what the database rejects: enum vocabulary (CHECK), NOT NULL,
and dangling FKs. No name uniqueness, no "team can't play itself", no overlap checks,
no cross-field consistency on source references (a `group_rank` source carrying a
`match_id` is legal data). Neither handlers nor `api` functions contain business-rule validation.
The advisory warnings layer is M3+ and never blocks writes.

## Resource shapes

Plain resources marshal their store row struct directly. Nullable columns are nullable
JSON fields (Go pointer types). Field lists match the schema exactly:

- **tournament**: id, name, location, starts_on?, ends_on?, venue_opens?, venue_closes?,
  default_match_minutes?, default_gap_minutes?, created_at
- **division**: id, tournament_id, name, notes
- **team**: id, tournament_id, division_id?, name, country, contact, notes,
  withdrawn_at?, created_at
- **field**: id, tournament_id, name
- **field_booking**: id, tournament_id, field_id?, team_id?, kind, label, starts_at?,
  ends_at?, notes
- **placement**: id, tournament_id, division_id?, rank, label, source_kind?,
  source_group_id?, source_rank?, source_match_id?, resolved_team_id?

(`?` = nullable. Placement's source_* columns stay flat — they are one optional
reference, not a repeating sub-structure.)

### match (composite: match + slot_source)

```json
{
  "id": 42, "tournament_id": 1, "division_id": 3, "group_id": null,
  "label": "SF1", "field_id": 2, "scheduled_at": "2026-07-18T14:00:00Z",
  "duration_minutes": null, "status": "scheduled",
  "referee_team_id": 9, "assistant_referee_team_id": null,
  "a_team_id": null, "a_score": null, "a_fouls": null, "a_yellow_cards": null, "a_red_cards": null,
  "b_team_id": 7,   "b_score": null, "b_fouls": null, "b_yellow_cards": null, "b_red_cards": null,
  "winner_team_id": null, "notes": "", "created_at": "…",
  "a_source": {"kind": "group_rank", "group_id": 3, "rank": 1},
  "b_source": null
}
```

- `a_source`/`b_source` are JSON views of the `slot_source` rows. Sub-object keys:
  `kind`, `group_id` (→ ref_group_id), `rank` (→ ref_rank), `match_id` (→ ref_match_id) —
  all nullable except `kind`.
- PATCH: `"a_source": null` deletes the row; `"a_source": {…}` replaces it wholesale
  (no partial patch inside the sub-object); absent = untouched. Same tri-state as
  scalar fields, via `Opt[SlotSourceView]`.
- Store: `CreateMatch` / `UpdateMatch` take the row plus both source values and write
  match + slot_source in one transaction (update = delete both slot rows, re-insert
  the present ones). `GetMatch` / `ListMatches` join the sources in.
- PATCHing `a_team_id` directly never touches `a_source` (resolution semantics are
  M3's; the API stores facts).

### group (composite: team_group + members + ranking)

```json
{
  "id": 3, "tournament_id": 1, "division_id": 2, "name": "Group A", "notes": "",
  "ranking_confirmed_at": null,
  "member_team_ids": [1, 4, 7],
  "ranking": [{"team_id": 4, "rank": 1}, {"team_id": 1, "rank": 2}]
}
```

- `member_team_ids` (ordered by team_id) and `ranking` (ordered by rank, then team_id)
  are JSON views of `team_group_member` and `group_ranking`.
- PATCH replaces a list wholesale (delete all, re-insert); `[]` empties it; `null` is
  equivalent to `[]` for these lists; absent = untouched. Lists are 4–6 entries —
  atomic replace, no per-row endpoints.
- Ties in `ranking` are legal (freeform); duplicate team_ids in a submitted list are a
  PK violation → INVALID_VALUE.

## API docs (Swagger)

Hand-written OpenAPI 3 spec, served by the binary:

- `internal/server/openapi.yaml` — the spec, written by hand (it lives in `server`
  because it documents the HTTP transport and is embedded from there; not to be
  confused with the `internal/api` package). The API's uniformity keeps it compact:
  shared `$ref` components for the error envelope, per-resource schemas, and the
  5-verb pattern.
- `GET /api/openapi.yaml` — serves the spec file (embedded via `go:embed`).
- `GET /api/docs` — Swagger UI, its static assets vendored into the repo
  (from the `swagger-ui-dist` npm package) and embedded in the binary.
  Self-contained; no CDN.
- **Drift guard:** a test walks every route registered in `routes.go` and asserts
  the same method+path exists in `openapi.yaml` (and vice versa). Catches the real
  failure mode — a forgotten or renamed endpoint. Deeper safeguards (schema-level
  contract checks in CI) are deferred until more people contribute.

## Request flow (reference trace)

`PATCH /api/teams/5` body `{"name":"RTT","division_id":null}`:

1. `routes.go`: `mux.HandleFunc("PATCH /api/teams/{id}", h.patchTeam)`.
2. `server/team.go`: parse id, `decode(r, &patch)` (strict), call `api.UpdateTeam(h.st, 5, patch)`.
3. `api/team.go`: `st.GetTeam(5)` (miss → NOT_FOUND); apply patch onto the row
   (`Name.Set` → overwrite; `DivisionID.Set && Null` → nil; absent → untouched);
   `st.UpdateTeam(row)`; map store errors via `fromStore`. (Store keeps M1's
   method-on-`*Store` style; only the `api` layer uses free functions.)
4. `store/team.go`: full-row `UPDATE team SET … WHERE id=?`; errors through `translate()`.
5. Handler: `writeJSON(w, 200, team)` or `writeError(w, err)`.

POST is the same with a zero base row → 201. Wiring in `cmd/ssl-tournament/main.go` is
unchanged: `store.Open → server.Run(…, st)` — handlers hold the store and pass it to
`api` functions. A future MCP server constructs the same store and calls the same
`api` functions.

## Testing

- Shared harness: `newTestServer(t)` = temp-dir store + `NewMux`, driven via
  `httptest`. Never `:memory:`.
- Per resource: full CRUD cycle; PATCH tri-state (absent kept, null cleared, value set);
  list filters filter and unknown params reject.
- Error-code coverage once per code (bad JSON, unknown field, bad enum, missing row,
  dangling FK, bad id syntax).
- Composite tests: match source set/replace/clear survives GET; group member/ranking
  replace; ranking ties accepted.
- Freeform tests at the API level: duplicate names, team vs itself, match on a blocked
  field, `group_rank` source carrying a `match_id` — all 2xx.
- `api.Opt[T]` unit tests (absent/null/value decode) — the one piece of real
  mechanism.

## Out of scope (M2a)

Auth (M5), events endpoint (M4), any resolution/standings logic and advisory warnings
(M3), pagination (lists are tournament-sized), the wizard UI (M2b), MCP server (the
`internal/api` layer is the seam; nothing is built for it now), schema-level OpenAPI drift
tooling beyond the route-sync test (revisit when contributors join).
