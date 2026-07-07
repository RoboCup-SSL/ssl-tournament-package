# ssl-tournament-package — architecture

The **organizational layer around SSL matches** — the smallest unit useful on its own, with
**no hardware/host coupling** (no multicast/USB/GPU; pure data + HTTP). Any SSL event could
run just this. Sibling to `ssl-streaming-package` (the host-bound streaming tier, which stays
Python); see that repo's `docs/2026-postmortem.md` for the lessons that shaped this design.

## Purpose & scope

In scope:
- **Teams** + metadata (country, division, radio frequencies, team size, contact, …).
- **Tournament format** (group phase, elimination, combinations) and **brackets**.
- **Referee assignment** — which team refs which match, incl. *suggestions* as the event progresses.
- **Live run** — scores, bracket progression, standings.

Out of scope (later, as *separate* services/repos that consume this one's API): GC-operator
progress hookin, stream-health from the streaming PCs, auto "end of match" detection.

## Tech stack & distribution — **Go**

Backend and CLI are **Go**. This is a deliberate, project-specific choice (the streaming repo
stays Python; this one stands alone and shares no code).

**Why Go for this repo:**
1. **Distribution is the whole point.** The goal is "download one file on any OS and it just
   runs." Go cross-compiles all targets from a *single* CI job into **one static binary each,
   with the web frontend embedded** — no per-OS build runners, no runtime deps. (Python would
   need PyInstaller on per-OS runners for a heavier, fussier result.)
2. **Ecosystem fit.** `ssl-game-controller` and most RoboCup SSL tooling are Go — same
   protobuf conventions, same release conventions, same community patterns.
3. **Fresh, standalone repo** — no sunk Python code to preserve; integrates with everything
   else only over HTTP.

**Model to copy:** the SSL Game Controller does exactly this pattern —
**https://github.com/RoboCup-SSL/ssl-game-controller** . Worth studying in that repo:
- `frontend/embed.go` — `//go:embed dist` compiles the built web UI *into* the binary.
- `.circleci/config.yml` — one job cross-compiles `GOOS/GOARCH` for
  linux amd64/arm64/arm, darwin amd64/arm64, windows amd64/arm64, publishes them to GitHub
  Releases with `ghr`, and pushes multi-arch Docker images.
- `README.md` "Releases" — the user flow: download the `<name>_<version>_<os>-<arch>` asset,
  run it, open the browser.
- `Makefile` — `npm run build` the frontend, then `go build` (frontend gets embedded).

**Concrete stack:**
- **HTTP + routing:** Go stdlib `net/http` (+ a light router like `chi` if useful).
- **Frontend:** a component-based, mobile-first web UI (Vue or React), built to `dist/` and
  **embedded via `//go:embed`** — served by the same binary alongside the JSON API.
- **Datastore:** SQLite via a **pure-Go driver (`modernc.org/sqlite`)** — no cgo, so
  cross-compilation stays trivial. Single file = whole tournament.
- **Passwords:** `golang.org/x/crypto/bcrypt`; sessions via a signed cookie.
- **Releases:** git tag → CI cross-compiles → GitHub Releases (`ghr`) + optional multi-arch
  Docker image, mirroring ssl-game-controller.

## Principles

1. **API-first source of truth.** Everything is data behind a clean HTTP API. The web UI is
   just a client of that API. Other tools integrate via the API, never by importing code.
2. **Stable IDs everywhere** (`team_id`, `match_id`, …). This is the *only* seam external
   services need — it's what keeps the ecosystem decoupled and growable.
3. **Portable tier.** Pure software → single static binary for Linux/macOS/Windows;
   containerizes trivially. (Contrast the streaming tier, host-bound and Linux-only.)
4. **Keep it one service.** Do **not** pre-split into micro-services. The ref-suggester,
   brackets, standings are internal *packages* with clean interfaces — extractable later only
   if something genuinely needs independent scale/deploy. The real service boundary is
   *external event producers* (below), not internal logic.

## Components

- **Datastore:** SQLite (`modernc.org/sqlite`), single file → easy backup/hand-off.
- **HTTP API + web UI:** one binary. Web UI is a guided, mobile-first wizard
  (welcome → where/when → fields → teams → format → run), built to `dist/` and embedded.
- **Ref-suggestion package (internal):** constraint-based — can't ref your own match, spread
  the load, respect division/availability, avoid back-to-back. Its own package so it can get
  smarter without touching the rest.
- **Brackets/standings package (internal):** matches as edges in a bracket graph so
  "who advances" and "who's eligible to ref next" derive from the same structure.
- **Event ingest + review queue:** see below.

## Event flow (external producers → human-approved)

External programs **push** events; they never mutate the tournament directly.

1. Producer `POST /events` → e.g. `{type: "match_ended", match_id, score, source, ts, dedupe_key}`.
2. Stored as a **pending** event (status = pending). *No effect on the DB yet.*
3. Organizer (website open) sees the **pending queue** → **approve** (runs the per-type
   handler: record score, recompute standings, advance bracket) or **reject** (discard).

Rules that keep this clean:
- **Idempotency/dedupe** — producers resend; a `dedupe_key` collapses duplicates to one item.
- **Provenance** — every event carries `source` + `match_id` + `ts` so the approver knows what
  they're accepting.
- **Explicit approve-handlers** — one small, testable handler per event type; "what happens on
  accept" is never implicit.
- **The pending log is an audit trail** — who approved what, when (useful for disputes).

This human-in-the-loop step is deliberate: the 2026 event showed automated match detection is
unreliable (off-schedule matches, wrong-match guesses). A person confirms before it counts.

## Auth & permissions

Two distinct needs — keep both **simple** (no OAuth/SSO):

**Human users — session login with roles.**
- `organizer/admin`: edit everything, approve/reject events.
- `viewer` (default, incl. teams & public): **read-only** — view schedule/standings, nothing else.
- (later, optional) `scorer/referee`: submit results for their assigned matches only.
- Small `users` table, `bcrypt`-hashed passwords, signed session cookie.
- → Teams **cannot** touch the schedule because they simply have no write access.

**Machine producers — revocable API tokens.**
- A `tokens` table (name, hashed token, scope); sent as `Authorization: Bearer <token>` on
  `POST /events`. Each event attributed to its producer.

**Key security property:** a producer token *only* lets you create **pending** events —
nothing a producer sends changes the tournament until an organizer approves it. So a
leaked/fake token is at worst **queue spam, not data corruption** (bound it with dedupe +
rate-limit). Combined with organizer-only writes, that covers "teams messing with the
schedule / sending fake events."

**Mistake-proofing** (organizers are human too): confirm dialogs on irreversible actions
(finalize match, advance bracket), the approve-queue as a checkpoint, and the audit log for
"who changed this?"

## Deployment (two run models, one binary)

- **Desktop (organizers):** download the per-OS/arch build (`.exe` / macOS / `AppImage`) from
  Releases → run → it serves the embedded UI locally and opens the browser, with a tray icon
  (Open / Copy LAN URL / Quit; e.g. `fyne.io/systray`). Single-instance (second launch just
  reopens the page). No autostart. Uninstall = delete the file; data intentionally survives.
- **Headless (RPi/always-on):** the **same binary** under **systemd** (`enable --now`,
  `Restart=always`, bind `0.0.0.0`) → boots on power-up. Install/uninstall via a small script.
- **Data location:** OS-appropriate per-user data dir (via `os.UserConfigDir()` or similar),
  never next to the binary — holds `tournament.db` + an `assets/` dir (team logos, etc.). Show
  the path in the UI; offer Export/Import for backup + moving.
- Bind `0.0.0.0` so the organizer runs it on one host and refs/helpers use it from phones on
  the venue Wi-Fi (mobile-first pays off here).

## Non-goals / later

GC-operator progress, stream-health, auto match-detection are **future external producers** —
own repos/services that push events here and read via the API. They need only stable IDs +
the API + a producer token. Don't build them in; let the ecosystem grow around this core.
