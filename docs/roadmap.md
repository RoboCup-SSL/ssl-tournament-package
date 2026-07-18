# ssl-tournament-package — build roadmap

The *what/why* is in [architecture.md](architecture.md); this is the **build order**. Each
milestone leaves a runnable binary. Mirror `ssl-game-controller`
(https://github.com/RoboCup-SSL/ssl-game-controller) for the embed + release patterns.

## Milestone 0 — running skeleton — **done**
- `go mod init github.com/RoboCup-SSL/ssl-tournament-package`
- `cmd/ssl-tournament/main.go`: stdlib `net/http` server with flags (`--host`, `--port`).
- `frontend/dist/` placeholder + a `frontend/embed.go` (`//go:embed dist`) serving it at `/`
  (copy the shape of ssl-game-controller's `frontend/embed.go`).
- `/healthz` endpoint; a `/events` stub (accepts + no-ops for now).
- **Goal:** `go build` → one binary that runs and serves a page. Distribution proven before
  any real logic.

## Milestone 1 — data layer — **done**
- SQLite via `modernc.org/sqlite` (pure-Go, keeps cross-compile trivial).
- Data dir via `os.UserConfigDir()` (never next to the binary); open/create `tournament.db`.
- Schema + simple migrations. Core tables: `teams`, `tournaments`, `matches`, `assignments`,
  `events`, `tokens` (fields per architecture.md's Auth + Event-flow sections).

## Milestone 2 — core API + editing UI
- **M2a (API): done** — see docs/superpowers/specs/2026-07-14-m2a-crud-api-design.md; Swagger UI at /api/docs.
- JSON CRUD for teams / tournament / format.

### North star: replace the schedule Excel sheet
The first thing worth shipping is a **schedule view** — every match on a field × time grid,
add/move/edit in a couple of clicks (not a million clicks in a Google Doc), including assigning
the referee/assistant teams. That single deliverable *is* the MVP: "move the schedule off Google
Sheets onto a homepage, with ref assignment, exportable as ICS." Everything fancier — automated
scheduling, automated ref suggestion, Game-Controller result ingestion — is added **only after**
that works and is thoroughly tested.

This scope-down came from RoboCup-SSL organizer feedback (Tobias, TIGERs): limit scope, go one
increment at a time, and treat **testing/verification as the biggest risk** — it's the first
thing to fall off the cliff when scope balloons. Their concrete asks (schedule visualization that
isn't painful, ICS export, ref-assignment help, and software that *catches mistakes*) shape the
order below.

### Running principles
- **Proper editing UI first, wizard later.** The deliverable is a complete UI that lets organizers
  create and change *anything* in the freeform model at any time (fields, teams, divisions, groups,
  matches, placements) — matching the "freeform by design" principle. The guided wizard
  (welcome → where/when → fields → teams → format → run) is a later luxury layered *on top* to
  smooth first-time setup; it does nothing the editing UI can't.
- **Freeform, never prescriptive.** The data model already supports both the freeform Excel use
  case (schedule any match, no restrictions) *and* the nice structured stuff (groups, elimination,
  auto-advancing winners/losers). We do **not** build a prescriptive "one true format" engine —
  schedules differ every time (cf. FIFA; the B-division Swiss experiment). Structure-aware help
  arrives later as **catch-and-warn** validation (flag likely mistakes, e.g. a needless
  lower-bracket match right after the upper-bracket one) — warnings, never hard constraints.
- **One increment at a time, each thoroughly tested before the next.**
- **M3 generators appear as in-UI helpers**, producing ordinary hand-editable rows — so the UI
  needs no M3 to be useful.

### Stack (confirmed at M2)
Vue 3 + TypeScript + Quasar + Vite + Pinia + Vue Router, mirroring `ssl-game-controller`'s
toolchain and versions — but *not* its transport (GC streams live state over WebSocket+protobuf;
this app is API-first JSON over `net/http`, so a plain fetch/JSON client, no protobuf/WS, no
`src/proto/`). Built to `dist/`, embedded via `//go:embed`.

### Information architecture
Instance = tournament list + "new tournament"; pick one → a tournament-scoped workspace with
per-entity sections (Overview, Settings/where-when, Fields, Teams, Divisions, Groups, Matches,
Standings/Bracket, Placements) — drawer on desktop, menu on mobile.

### Build order (value-first, toward the MVP then outward)
- **M2b** — frontend shell (thin vertical slice): toolchain + JSON client + Pinia + hash router +
  embed pipeline + one read-only HomeView (tournament list). See
  docs/superpowers/specs/2026-07-17-m2b-frontend-shell-design.md.
- **M2b·dt (data-layer)** — date/time format validation + tournament timezone: server-side
  validation for every date/time field (reject malformed shape, normalize to canonical form;
  values stay freeform), plus a single `time_zone` (IANA) anchor on the tournament. Backend stays
  timezone-naive (instants stored bare, no offset; all relative to each other); the frontend
  applies the zone for display only. Precursor to M2c. See
  docs/superpowers/specs/2026-07-18-datetime-validation-timezone-design.md.
- **M2c** — workspace shell + Settings (where/when) section: dates, venue hours, default
  match/gap minutes (the scheduling inputs) + the `time_zone` picker; stashes the zone in Pinia.
  Depends on M2b·dt. Thinnest section slice; proves the section pattern.
- **M2d** — Fields + Teams sections: the entities a match references.
- **M2e — Schedule view (the MVP):** matches on a field × time grid; create/move/edit freely,
  assign ref/assistant teams (`referee_team_id`/`assistant_referee_team_id` already in the model).
  This is the Excel replacement.
- **M2f** — ICS export from the schedule (each match → a calendar event).
- **Later (enhancements on the freeform base, only after the MVP is solid + tested):**
  - Divisions/Groups sections, elimination wiring (slot_source), auto-advance, standings/bracket views.
  - Catch-and-warn validation warnings.
  - Automated ref *suggestion* (M3 `refsuggest`) and automated scheduling.
  - Game-Controller result ingestion → one-click "yes, the score sheet agrees" confirm (ties into M4).
- **Last** — the guided wizard layer.

## Milestone 3 — domain logic (internal packages)
- `brackets`/`standings`: matches as edges in a bracket graph → "who advances" + "who's
  eligible to ref next" derive from the same structure.
- `refsuggest`: constraint-based suggestions (not your own match, spread load, division,
  no back-to-back). Clean interface so it can grow smarter in isolation.

## Milestone 4 — events + review queue
- `POST /events` (token-authed) → store as **pending** with `dedupe_key` + provenance.
- Pending-queue UI; **approve** runs a per-type handler (score → recompute standings/bracket),
  **reject** discards. Keep handlers explicit + testable. Pending log = audit trail.

## Milestone 5 — auth
- Per-tournament tokens, no user accounts (spec round 4): tournament creation returns its
  first **admin token**; organizers mint/revoke named admin tokens for each other; **producer
  tokens** authorize `POST /events`. Reads stay public. (Teams get no write access by design.)
- Recovery: CLI reset on the host now; an instance-admin reset endpoint once hosted 24/7;
  email self-service later via `tournament.contact_email`.

## Milestone 6 — release & run
- CI cross-compile (`GOOS/GOARCH`: linux amd64/arm64/arm, darwin amd64/arm64, windows
  amd64/arm64) → GitHub Releases via `ghr` (+ optional multi-arch Docker) — copy
  ssl-game-controller's `.circleci/config.yml`.
- **Desktop:** single-instance guard, tray icon (Open / Copy LAN URL / Quit), auto-open browser.
- **Headless:** `install.sh`/`uninstall.sh` for a systemd unit (`enable --now`, `Restart=always`).

---

Rough sequencing note: 0→1→2 gets a usable local tool; 3–5 make it a real tournament manager;
6 is polish/distribution and can trail behind (a `go build` binary is already shareable).
