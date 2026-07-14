# ssl-tournament-package — build roadmap

The *what/why* is in [architecture.md](architecture.md); this is the **build order**. Each
milestone leaves a runnable binary. Mirror `ssl-game-controller`
(https://github.com/RoboCup-SSL/ssl-game-controller) for the embed + release patterns.

## Milestone 0 — running skeleton
- `go mod init github.com/RoboCup-SSL/ssl-tournament-package`
- `cmd/ssl-tournament/main.go`: stdlib `net/http` server with flags (`--host`, `--port`).
- `frontend/dist/` placeholder + a `frontend/embed.go` (`//go:embed dist`) serving it at `/`
  (copy the shape of ssl-game-controller's `frontend/embed.go`).
- `/healthz` endpoint; a `/events` stub (accepts + no-ops for now).
- **Goal:** `go build` → one binary that runs and serves a page. Distribution proven before
  any real logic.

## Milestone 1 — data layer
- SQLite via `modernc.org/sqlite` (pure-Go, keeps cross-compile trivial).
- Data dir via `os.UserConfigDir()` (never next to the binary); open/create `tournament.db`.
- Schema + simple migrations. Core tables: `teams`, `tournaments`, `matches`, `assignments`,
  `events`, `tokens` (fields per architecture.md's Auth + Event-flow sections).

## Milestone 2 — core API + wizard UI
- JSON CRUD for teams / tournament / format.
- Mobile-first wizard frontend built to `dist/`, embedded.
  Flow: welcome → where/when → fields → teams → format → run.
- Tentative stack (to confirm at M2): **Vue 3 + TypeScript + Quasar + Vite**, mirroring
  `ssl-game-controller`'s frontend so its setup, dev-proxy, and component patterns copy over.
  Take GC's toolchain but *not* its transport — GC streams live state over WebSocket+protobuf;
  this app is API-first JSON over `net/http`, so use a plain fetch/JSON client.

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
