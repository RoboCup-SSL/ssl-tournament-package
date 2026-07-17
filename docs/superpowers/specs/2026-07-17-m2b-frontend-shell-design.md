# M2b — Frontend shell (thin vertical slice)

Status: approved (design)
Date: 2026-07-17
Follows: M2a CRUD API (docs/superpowers/specs/2026-07-14-m2a-crud-api-design.md)

## Purpose

Stand up the real M2 web frontend by proving its entire pipeline end-to-end —
Vue source → Vite build → `frontend/dist/` → `//go:embed` → Go `FileServer` →
browser → `fetch` against the JSON API — with **one** working screen. Once the
pipeline is de-risked, the six wizard steps (welcome → where/when → fields →
teams → format → run) become pure content work in follow-up specs, each adding
views without touching the toolchain.

This slice deliberately builds a shell, not wizard content.

## Background

The current UI is a single hand-written `frontend/dist/index.html` plus a
vendored `mermaid.min.js` — the "stone-age" admin serving raw CRUD and an ER
diagram over the API. It stays useful for debugging and is preserved (see
"Stone-age admin" below), not deleted.

The server (`internal/server`, Go stdlib `net/http`) registers the JSON API
under `/api/*`, `/healthz`, `/api/version`, `/api/docs`, `/api/openapi.json`,
and serves the embedded `dist/` at `/` via `frontend.Handler()` (a
`FileServer` marked `Cache-Control: no-store`). Default port 8080.

## Stack

Vue 3 (Composition API) + TypeScript + Quasar + Vite + Pinia + Vue Router.

This mirrors `ssl-game-controller`'s frontend toolchain, reusing its **exact
dependency versions** so the local npm cache resolves offline (the local Nexus
proxy is dead in this environment; CI runners use public npm and are
unaffected). We copy GC's toolchain but **not** its transport: GC streams live
state over WebSocket + protobuf; this app is request/response JSON over
`net/http`. So there is **no** `@bufbuild/protobuf`, no `@types/google-protobuf`,
no WebSocket machinery, and no `src/proto/`. Transport is a plain typed `fetch`
client.

## Scope

In scope:
- Vue/Quasar/Vite/Pinia/Router scaffold under `frontend/`.
- Typed JSON `fetch` client that surfaces the API's structured error envelope.
- A Pinia store (`tournaments`) establishing the state pattern.
- One view (`HomeView`) that reads `/api/version` and `/api/tournaments` and
  renders them, plus a link to the preserved admin.
- Build wired so `npm run build` produces `frontend/dist/`, which is committed
  and embedded unchanged by the existing `embed.go`.
- The stone-age admin preserved at `/admin/`.

Out of scope (explicit non-goals for this slice):
- Any wizard step content (welcome/where-when/fields/teams/format/run).
- OpenAPI → TypeScript type generation (types are hand-written here; codegen is
  a later concern once the surface grows).
- Vitest / component tests (the gate is type-check + build + manual verify).
- Write operations from the new UI (create/edit/delete) — reads only.
- Gitignoring `dist` + adding a CI Node step (see "dist strategy").
- History-mode routing / SPA fallback (hash routing avoids it).

## File layout

New files under `frontend/`, mirroring GC's structure:

```
frontend/
  package.json          # GC deps/versions minus protobuf+ws; scripts: dev, build, type-check, lint, preview
  vite.config.ts        # vue + quasar plugins; @ alias; proxy /api -> http://localhost:8080 (http, not ws)
  tsconfig.json  tsconfig.app.json  tsconfig.node.json   # copied from GC
  eslint.config.js      # copied from GC
  env.d.ts
  index.html            # <div id="app"> + module script; <title>SSL Tournament</title>
  .gitignore            # node_modules, logs, *.local, *.tsbuildinfo  (NOT dist — see dist strategy)
  embed.go              # UNCHANGED
  src/
    main.ts             # createApp + router + createPinia + Quasar; import quasar.css + material-icons; NO control plugin
    App.vue             # q-layout: q-header (title + version), <router-view>, link to /admin/
    router/index.ts     # createWebHashHistory; route '/' -> HomeView
    api/client.ts       # get/post/patch/del helpers; throw ApiError carrying the parsed error envelope
    api/types.ts        # hand-written Tournament, Version types
    store/tournaments.ts# Pinia store: tournaments list + fetch() action
    views/HomeView.vue  # fetch version + tournaments via store/client; render Quasar list
    assets/main.scss    # minimal global styles
  public/admin/
    index.html          # current frontend/dist/index.html moved here verbatim
    mermaid.min.js       # current frontend/dist/mermaid.min.js moved here
```

Vite copies `public/` verbatim into `dist/`, so `public/admin/index.html`
becomes `dist/admin/index.html`, served at `/admin/`. The one absolute
reference in the admin file (`/mermaid.min.js`) becomes relative
(`mermaid.min.js`) so it resolves under `/admin/`. Its `/api/...` calls stay
absolute and keep working.

## Key decisions

**Routing — hash mode.** `createWebHashHistory`, exactly like GC. URLs look
like `/#/…`. The Go `FileServer` needs no SPA fallback handler; unknown client
routes never reach the server. Simplest correct option for an embedded tool;
can switch to history mode + a fallback handler later if clean URLs are wanted.

**dist strategy — commit `dist/` (option A).** The built bundle is committed to
git, as it effectively is today. Consequences: CI stays Node-free (`go build
./...` and `go test ./...` work on a bare checkout); local `make build`/`make
run` always work; `//go:embed dist` is always satisfied. Cost: committed build
artifacts create diff noise. The GC-faithful alternative (gitignore `dist`, add
a CI Node step, `go build` requires a prior `npm run build`) is deferred to its
own small spec once the frontend stabilizes. Therefore `frontend/.gitignore`
ignores `node_modules` and friends but **not** `dist`.

**Dev proxy.** `vite.config.ts` proxies `/api` to `http://localhost:8080`
(plain http, not GC's `ws://`), so `npm run dev` hot-reloads against a locally
running `make run`.

**API client.** `api/client.ts` wraps `fetch`: sets JSON headers on write
bodies, returns `null` on 204, parses JSON otherwise, and on non-2xx throws an
`ApiError` carrying the parsed structured error envelope (the same envelope the
stone-age admin surfaces today). Reads only in this slice.

**Pinia store.** A single `tournaments` store with a list state and a `fetch()`
action, to establish the store pattern the wizard will follow. Kept minimal.

**Stone-age admin.** Preserved, not deleted — moved into `public/admin/`,
served at `/admin/`, linked from the new `App.vue` header.

**embed.go / server.** Unchanged. `frontend.Handler()` already serves `dist/`
with `no-store`; `NewMux` already routes `/api/*` before `/`.

## Verification

The slice is done when all hold:
1. `npm run build` (runs `vue-tsc` type-check then `vite build`) succeeds and
   populates `frontend/dist/`.
2. `go build ./...` and `go test ./...` stay green with the new committed
   `dist/`.
3. Manual end-to-end: start `make run`, open `/`, confirm the header shows the
   version from `/api/version` and the page lists tournaments from
   `/api/tournaments`; open `/admin/` and confirm the stone-age admin (tabs, ER
   diagram, CRUD) still works.
4. `npm run dev` proxies `/api` to the running Go server (hot-reload sanity
   check).

## npm touchpoints (run by the user)

Two, minimized to lean on the local cache:
1. `npm install` — once, after `package.json` is written (generates
   `node_modules/` and `package-lock.json`).
2. `npm run build` — produces `dist/`; re-run after frontend source edits.

`package-lock.json` is committed; `node_modules/` is gitignored.

## Follow-ups (later specs)

Reframed 2026-07-17: **proper editing UI first, wizard later.** The product is a
complete UI for editing the freeform model at any time; the guided wizard is a
later luxury layered on top (see roadmap M2). So HomeView here grows into the
instance-level tournament list, and the next specs build a tournament-scoped
workspace with per-entity sections, not wizard steps.

- **M2c** — workspace shell + Settings (where/when) section: a tournament
  workspace layout (drawer on desktop, menu on mobile) hosting per-entity
  sections, with the first section (Tournament settings: location, dates, venue
  hours, default match/gap minutes) as a real editable form. Thinnest section
  slice; proves the section pattern before fanning out. First write operations.
- **M2d** — Fields + Teams sections (the entities a match references).
- **M2e** — Schedule view (the MVP): matches on a field × time grid,
  create/move/edit freely incl. ref/assistant assignment. Replaces the Excel
  sheet — the north star per organizer feedback (see roadmap M2).
- **M2f** — ICS export from the schedule.
- **Later** — Divisions/Groups, elimination wiring, auto-advance,
  standings/bracket; catch-and-warn validation; automated ref suggestion (M3)
  and scheduling; GC result → one-click confirm (M4); guided wizard last.
- OpenAPI → TS type generation once the type surface justifies it.
- Flip dist strategy to option B (gitignore + CI Node build) when stable.
- Component tests (Vitest) once there is real interaction logic to cover.
