# M2b Frontend Shell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the real M2 web frontend (Vue 3 + Quasar + Vite) as a thin vertical slice — one read-only screen — proving the whole Vue → dist → `//go:embed` → serve → fetch pipeline, with the stone-age admin preserved at `/admin/`.

**Architecture:** Mirror `ssl-game-controller`'s frontend toolchain and dependency versions, minus its transport layer (no protobuf, no WebSocket). A plain typed `fetch` JSON client talks to the existing `/api/*` routes. Hash routing (`createWebHashHistory`) means the Go `FileServer` needs zero changes and no SPA fallback. The built bundle is committed to `frontend/dist/` (option A) so CI stays Node-free and `//go:embed dist` is always satisfied.

**Tech Stack:** Vue 3 (Composition API) · TypeScript · Quasar · Vite · Pinia · Vue Router · Go stdlib `net/http` (unchanged).

**Spec:** docs/superpowers/specs/2026-07-17-m2b-frontend-shell-design.md

## Global Constraints

- **npm is run by the user, not the agent.** The local Nexus proxy is dead in this environment and the managed policy forbids repointing the registry. This plan has exactly **two** npm touchpoints — `npm install` (Task 1) and `npm run build` (Task 4) — each marked ⚠️ **USER RUNS THIS**. Stop and ask the user to run the command, then continue once they confirm.
- **Never** add `--registry`/`--index-url`/`-i` flags or edit `.npmrc` to point at a public registry (managed org policy).
- **Mirror GC's exact dependency versions** (below) so the user's local npm cache resolves offline.
- **No protobuf, no WebSocket, no `src/proto/`** — JSON request/response only.
- **`frontend/dist/` is committed**; `frontend/.gitignore` ignores `node_modules` (and logs/`*.local`/`*.tsbuildinfo`) but **not** `dist`.
- **Reads only** in this slice — no create/edit/delete from the new UI.
- **Comment style:** readable code; a one-line docstring per file and per function; minimal inline comments.
- **Go 1.26, CGO disabled** (unchanged; this plan does not touch Go build flags).
- **Branch:** work on `m2b-frontend-shell` (already checked out). Main is PR-only.

---

## File Structure

New/changed files, each with one responsibility:

```
frontend/
  package.json            # deps + scripts (GC versions minus protobuf/ws)
  vite.config.ts          # vue+quasar plugins, @ alias, /api dev proxy -> :8080
  tsconfig.json           # project references (copied from GC)
  tsconfig.app.json       # app TS config (copied from GC)
  tsconfig.node.json      # node/build TS config (copied from GC)
  eslint.config.js        # lint config (copied from GC)
  env.d.ts                # vite client types
  index.html              # SPA entry: <div id="app"> + module script
  .gitignore              # node_modules etc; NOT dist
  README.md               # how to dev/build the frontend
  embed.go                # UNCHANGED
  embed_test.go           # NEW: regression test — dist serves / and /admin/
  src/
    main.ts               # createApp + router + pinia + Quasar
    App.vue               # q-layout header (title+version) + <router-view> + admin link
    router/index.ts       # hash history; '/' -> HomeView
    api/client.ts         # typed fetch wrapper + ApiError
    api/types.ts          # Version, Tournament interfaces
    store/tournaments.ts  # Pinia store: tournaments list + fetch()
    views/HomeView.vue    # renders the tournament list
    assets/main.scss      # minimal Quasar color vars
  public/admin/
    index.html            # the current stone-age admin, moved here verbatim
    mermaid.min.js         # moved here; admin's ref made relative
Makefile                  # +frontend target (convenience rebuild)
```

**Exact dependency versions (GC-matched):**

```
dependencies:   @quasar/extras ^1.16.15 · pinia ^3.0.3 · quasar ^2.18.0 · vue ^3.5.16 · vue-router ^4.5.0
devDependencies: @quasar/vite-plugin ^1.8.1 · @rushstack/eslint-patch ^1.10.5 · @tsconfig/node24 ^24.0.0 ·
  @types/node ^22.15.30 · @vitejs/plugin-vue ^5.2.1 · @vue/eslint-config-typescript ^14.5.0 ·
  @vue/tsconfig ^0.9.0 · eslint ^10.0.0 · eslint-plugin-vue ^10.2.0 · npm-run-all2 ^8.0.4 ·
  sass ^1.89.1 · typescript ~5.9.0 · vite ^6.3.5 · vue-tsc ^2.2.10
```

**Note on TDD:** the spec defers Vitest (YAGNI for a shell). The automated gates here are `vue-tsc` type-check + `vite build` (Task 4), the Go `embed_test.go` regression test (Task 5), and `go test ./...` staying green — plus a manual end-to-end browser check (Task 6). This is a deliberate, spec-sanctioned deviation from unit-test-first.

---

### Task 1: Toolchain config + install

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`, `frontend/tsconfig.app.json`, `frontend/tsconfig.node.json`
- Create: `frontend/eslint.config.js`
- Create: `frontend/env.d.ts`
- Create: `frontend/index.html`
- Create: `frontend/.gitignore`

**Interfaces:**
- Produces: an installable frontend project. The `@` alias resolves to `frontend/src`. Dev server proxies `/api` → `http://localhost:8080`. Later tasks rely on the scripts `npm run build` (type-check + vite build) and `npm run dev`.

- [ ] **Step 1: Write `frontend/package.json`**

```json
{
  "name": "ssl-tournament",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "run-p type-check \"build-only {@}\" --",
    "preview": "vite preview",
    "build-only": "vite build",
    "type-check": "vue-tsc --build --force",
    "lint": "eslint . --fix"
  },
  "dependencies": {
    "@quasar/extras": "^1.16.15",
    "pinia": "^3.0.3",
    "quasar": "^2.18.0",
    "vue": "^3.5.16",
    "vue-router": "^4.5.0"
  },
  "devDependencies": {
    "@quasar/vite-plugin": "^1.8.1",
    "@rushstack/eslint-patch": "^1.10.5",
    "@tsconfig/node24": "^24.0.0",
    "@types/node": "^22.15.30",
    "@vitejs/plugin-vue": "^5.2.1",
    "@vue/eslint-config-typescript": "^14.5.0",
    "@vue/tsconfig": "^0.9.0",
    "eslint": "^10.0.0",
    "eslint-plugin-vue": "^10.2.0",
    "npm-run-all2": "^8.0.4",
    "sass": "^1.89.1",
    "typescript": "~5.9.0",
    "vite": "^6.3.5",
    "vue-tsc": "^2.2.10"
  }
}
```

- [ ] **Step 2: Write `frontend/vite.config.ts`** (GC's, with a plain-HTTP `/api` proxy instead of `ws://`)

```ts
import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { quasar, transformAssetUrls } from '@quasar/vite-plugin'

// Build config for the embedded web UI. The dev server proxies /api to the
// local Go server so `npm run dev` hot-reloads against `make run`.
export default defineConfig({
  plugins: [
    vue({ template: { transformAssetUrls } }),
    quasar(),
  ],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
  build: { chunkSizeWarningLimit: 1000 },
})
```

- [ ] **Step 3: Write the three tsconfig files** (verbatim from GC)

`frontend/tsconfig.json`:
```json
{
  "files": [],
  "references": [
    { "path": "./tsconfig.node.json" },
    { "path": "./tsconfig.app.json" }
  ]
}
```

`frontend/tsconfig.app.json`:
```json
{
  "extends": "@vue/tsconfig/tsconfig.dom.json",
  "include": ["env.d.ts", "src/**/*", "src/**/*.vue"],
  "exclude": ["src/**/__tests__/*"],
  "compilerOptions": {
    "composite": true,
    "noEmit": true,
    "baseUrl": ".",
    "paths": { "@/*": ["./src/*"] }
  }
}
```

`frontend/tsconfig.node.json`:
```json
{
  "extends": "@tsconfig/node24/tsconfig.json",
  "include": ["vite.config.*"],
  "compilerOptions": {
    "composite": true,
    "noEmit": true,
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "types": ["node"]
  }
}
```

- [ ] **Step 4: Write `frontend/eslint.config.js`** (verbatim from GC)

```js
import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import * as parserVue from 'vue-eslint-parser'
import configTypeScript from '@vue/eslint-config-typescript'

export default [
  { name: 'app/files-to-lint', files: ['**/*.{ts,mts,tsx,vue}'] },
  { name: 'app/files-to-ignore', ignores: ['**/dist/**', '**/dist-ssr/**', '**/coverage/**'] },
  js.configs.recommended,
  ...pluginVue.configs['flat/essential'],
  ...configTypeScript(),
  {
    name: 'app/vue-rules',
    files: ['**/*.vue'],
    languageOptions: { parser: parserVue, ecmaVersion: 'latest', sourceType: 'module' },
  },
]
```

- [ ] **Step 5: Write `frontend/env.d.ts`**

```ts
/// <reference types="vite/client" />
```

- [ ] **Step 6: Write `frontend/index.html`**

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SSL Tournament</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

- [ ] **Step 7: Write `frontend/.gitignore`** (note: `dist` is intentionally NOT ignored)

```gitignore
# Logs
*.log
npm-debug.log*

node_modules
.DS_Store
*.local
*.tsbuildinfo
```

- [ ] **Step 8: ⚠️ USER RUNS THIS — install dependencies**

Ask the user to run (from the repo root), e.g. via the `! ` prompt prefix:
```
cd frontend && npm install
```
Expected: `node_modules/` and `frontend/package-lock.json` are created; no registry errors. If it fails against Nexus, the user reports it (do not repoint the registry). Wait for confirmation before continuing.

- [ ] **Step 9: Commit** (config + lockfile; `node_modules` is gitignored)

```bash
git add frontend/package.json frontend/package-lock.json frontend/vite.config.ts \
  frontend/tsconfig.json frontend/tsconfig.app.json frontend/tsconfig.node.json \
  frontend/eslint.config.js frontend/env.d.ts frontend/index.html frontend/.gitignore
git commit -m "feat(frontend): scaffold Vue3+Quasar+Vite toolchain (GC-matched, no protobuf/ws)"
```

---

### Task 2: Preserve the stone-age admin at `/admin/`

Move the current hand-written UI into `public/` **before** the first vite build, because `vite build` empties `dist/` and repopulates it. Anything under `public/` is copied verbatim into `dist/`, so `public/admin/` becomes `dist/admin/`, served at `/admin/`.

**Files:**
- Move: `frontend/dist/index.html` → `frontend/public/admin/index.html`
- Move: `frontend/dist/mermaid.min.js` → `frontend/public/admin/mermaid.min.js`
- Modify: `frontend/public/admin/index.html` (one line — make the mermaid `src` relative)

**Interfaces:**
- Produces: `dist/admin/index.html` (post-build) — the preserved admin, reachable at `/admin/`.

- [ ] **Step 1: Move the two files with git**

```bash
mkdir -p frontend/public/admin
git mv frontend/dist/index.html frontend/public/admin/index.html
git mv frontend/dist/mermaid.min.js frontend/public/admin/mermaid.min.js
```

- [ ] **Step 2: Make the admin's mermaid reference relative**

In `frontend/public/admin/index.html`, change the loader line so it resolves under `/admin/`:

Find:
```js
    script.src = "/mermaid.min.js";
```
Replace with:
```js
    script.src = "mermaid.min.js";
```

- [ ] **Step 3: Verify no other absolute asset refs**

Run: `grep -n 'src=\|href=' frontend/public/admin/index.html`
Expected: the only script/asset reference is the (now relative) `mermaid.min.js`; all `/api/...` calls stay absolute (correct — they hit the server).

- [ ] **Step 4: Commit**

```bash
git add frontend/public/admin
git commit -m "feat(frontend): preserve stone-age admin at /admin/ (moved into public/)"
```

---

### Task 3: Application source

Write the shell: JSON client, types, store, router, app chrome, and the one read-only view.

**Files:**
- Create: `frontend/src/api/types.ts`
- Create: `frontend/src/api/client.ts`
- Create: `frontend/src/store/tournaments.ts`
- Create: `frontend/src/router/index.ts`
- Create: `frontend/src/assets/main.scss`
- Create: `frontend/src/App.vue`
- Create: `frontend/src/views/HomeView.vue`
- Create: `frontend/src/main.ts`

**Interfaces:**
- Produces:
  - `api/types.ts`: `interface Version { version: string }`; `interface Tournament { id: number; name: string; location: string; starts_on: string | null; ends_on: string | null; venue_opens: string | null; venue_closes: string | null; default_match_minutes: number | null; default_gap_minutes: number | null; created_at: string }`
  - `api/client.ts`: `class ApiError` and `const api` with `get<T>(path)`, `post<T>(path, body)`, `patch<T>(path, body)`, `del(path)`.
  - `store/tournaments.ts`: `useTournamentsStore()` with state `{ tournaments: Tournament[], loading: boolean, error: string }` and action `fetch()`.
  - `router/index.ts`: default export `router` (hash history, route `'/'` → HomeView).

- [ ] **Step 1: Write `frontend/src/api/types.ts`** (fields verbatim from the OpenAPI Tournament schema)

```ts
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
```

- [ ] **Step 2: Write `frontend/src/api/client.ts`**

```ts
// Minimal JSON client for the tournament API. On a non-2xx response it throws
// ApiError carrying the server's structured error envelope { code, message, field }.

export interface ApiErrorBody {
  code: string
  message: string
  field?: string
}

// ApiError surfaces the server's error envelope to callers.
export class ApiError extends Error {
  status: number
  code: string
  field?: string

  constructor(status: number, body: ApiErrorBody) {
    super(body?.message || `HTTP ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.code = body?.code ?? 'INTERNAL'
    this.field = body?.field
  }
}

// request performs one JSON call and throws ApiError on failure.
async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const options: RequestInit = { method }
  if (body !== undefined) {
    options.headers = { 'Content-Type': 'application/json' }
    options.body = JSON.stringify(body)
  }
  const response = await fetch(path, options)
  if (response.status === 204) {
    return null as T
  }
  const payload = await response.json()
  if (!response.ok) {
    throw new ApiError(response.status, (payload as { error: ApiErrorBody }).error)
  }
  return payload as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body: unknown) => request<T>('POST', path, body),
  patch: <T>(path: string, body: unknown) => request<T>('PATCH', path, body),
  del: (path: string) => request<void>('DELETE', path),
}
```

- [ ] **Step 3: Write `frontend/src/store/tournaments.ts`**

```ts
// Pinia store for the instance-level tournament list. Establishes the store
// pattern the workspace sections will follow.
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import type { Tournament } from '@/api/types'

export const useTournamentsStore = defineStore('tournaments', {
  state: () => ({
    tournaments: [] as Tournament[],
    loading: false,
    error: '',
  }),
  actions: {
    // fetch loads all tournaments, capturing any error message for display.
    async fetch() {
      this.loading = true
      this.error = ''
      try {
        this.tournaments = await api.get<Tournament[]>('/api/tournaments')
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      } finally {
        this.loading = false
      }
    },
  },
})
```

- [ ] **Step 4: Write `frontend/src/router/index.ts`**

```ts
// Hash-history router (no server-side SPA fallback needed).
import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '@/views/HomeView.vue'

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: HomeView },
  ],
})

export default router
```

- [ ] **Step 5: Write `frontend/src/assets/main.scss`**

```scss
// Quasar brand color overrides (SSL teal), light and dark.
.body--light, .body--dark {
  --q-primary: #0794B9;
  --q-negative: #C10015;
  --q-warning: #F2C037;
}
```

- [ ] **Step 6: Write `frontend/src/App.vue`** (header with title + fetched version, admin link, router outlet)

```vue
<script setup lang="ts">
// App chrome: header with the server version and a link to the preserved admin.
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Version } from '@/api/types'

const version = ref('')

onMounted(async () => {
  try {
    version.value = (await api.get<Version>('/api/version')).version
  } catch {
    version.value = ''
  }
})
</script>

<template>
  <q-layout view="hHh lpR fFf">
    <q-header elevated>
      <q-toolbar>
        <q-toolbar-title>
          SSL Tournament <span class="text-caption">{{ version }}</span>
        </q-toolbar-title>
        <q-btn flat dense no-caps label="admin" type="a" href="/admin/" />
      </q-toolbar>
    </q-header>
    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>
```

- [ ] **Step 7: Write `frontend/src/views/HomeView.vue`** (the read-only tournament list)

```vue
<script setup lang="ts">
// Instance landing: lists tournaments from /api/tournaments.
import { onMounted } from 'vue'
import { useTournamentsStore } from '@/store/tournaments'

const store = useTournamentsStore()

onMounted(() => store.fetch())
</script>

<template>
  <q-page padding>
    <div class="text-h5 q-mb-md">Tournaments</div>

    <q-banner v-if="store.error" class="bg-negative text-white q-mb-md">
      {{ store.error }}
    </q-banner>

    <q-spinner v-if="store.loading" size="2em" />

    <q-list v-else-if="store.tournaments.length" bordered separator>
      <q-item v-for="t in store.tournaments" :key="t.id">
        <q-item-section>
          <q-item-label>{{ t.name }}</q-item-label>
          <q-item-label caption>
            {{ t.location || 'no location' }}<span v-if="t.starts_on"> · {{ t.starts_on }}</span>
          </q-item-label>
        </q-item-section>
      </q-item>
    </q-list>

    <div v-else class="text-grey">No tournaments yet.</div>
  </q-page>
</template>
```

- [ ] **Step 8: Write `frontend/src/main.ts`**

```ts
// App bootstrap: router + Pinia + Quasar. No control/WS plugin (JSON only).
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { Quasar } from 'quasar'
import router from './router'
import App from './App.vue'

import '@quasar/extras/material-icons/material-icons.css'
import 'quasar/dist/quasar.css'
import '@/assets/main.scss'

createApp(App)
  .use(router)
  .use(createPinia())
  .use(Quasar, {})
  .mount('#app')
```

- [ ] **Step 9: Commit** (source only; `dist` is rebuilt in Task 4)

```bash
git add frontend/src
git commit -m "feat(frontend): app shell — JSON client, tournaments store, HomeView"
```

---

### Task 4: Build and embed

Produce the real `dist/` from the new source and commit it (option A).

**Files:**
- Generated: `frontend/dist/**` (committed)

**Interfaces:**
- Produces: `frontend/dist/index.html` (contains `id="app"`), `frontend/dist/assets/**`, and `frontend/dist/admin/index.html` (from `public/admin/`). Consumed by the unchanged `embed.go` and by Task 5's test.

- [ ] **Step 1: ⚠️ USER RUNS THIS — build the frontend**

Ask the user to run:
```
cd frontend && npm run build
```
Expected: `vue-tsc` type-check passes, then `vite build` writes `frontend/dist/` with `index.html`, an `assets/` folder, and `admin/index.html` + `admin/mermaid.min.js`. Wait for confirmation. If type-check or build fails, fix the reported source file and ask the user to re-run.

- [ ] **Step 2: Verify the build output locally (agent)**

Run: `ls frontend/dist && ls frontend/dist/admin && grep -l 'id="app"' frontend/dist/index.html`
Expected: `index.html` + `assets/` present; `admin/index.html` + `admin/mermaid.min.js` present; the grep matches `frontend/dist/index.html`.

- [ ] **Step 3: Confirm Go still builds with the new embedded dist**

Run: `go build ./...`
Expected: builds with no errors (`//go:embed dist` picks up the new bundle).

- [ ] **Step 4: Commit the built bundle**

```bash
git add -f frontend/dist
git commit -m "build(frontend): compile shell to dist/ (committed, embedded)"
```
(`-f` is belt-and-suspenders; `dist` is not gitignored, but the old committed files were removed via `git mv` in Task 2, so this records the regenerated tree.)

---

### Task 5: Embed regression test

A Go test that guards the two things this slice must keep true: the SPA is served at `/`, and the admin survives at `/admin/`.

**Files:**
- Create: `frontend/embed_test.go`

**Interfaces:**
- Consumes: `frontend.Handler()` (existing) and the committed `frontend/dist/` from Task 4.

- [ ] **Step 1: Write the failing test**

`frontend/embed_test.go`:
```go
// Guards the embedded UI: the SPA at / and the preserved admin at /admin/.
package frontend

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesSPAAndAdmin(t *testing.T) {
	handler := Handler()
	cases := []struct {
		path     string
		contains string
	}{
		{"/", `id="app"`},
		{"/admin/", "temporary admin"},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d, want 200", tc.path, recorder.Code)
		}
		body, _ := io.ReadAll(recorder.Result().Body)
		if !strings.Contains(string(body), tc.contains) {
			t.Errorf("GET %s: body missing %q", tc.path, tc.contains)
		}
	}
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./frontend/ -run TestHandlerServesSPAAndAdmin -v`
Expected: PASS (dist is built and committed from Task 4). If it fails on `/` missing `id="app"`, the build didn't run — return to Task 4. If it fails on `/admin/`, re-check Task 2.

- [ ] **Step 3: Run the whole Go suite**

Run: `go test ./...`
Expected: PASS across all packages.

- [ ] **Step 4: Commit**

```bash
git add frontend/embed_test.go
git commit -m "test(frontend): embedded UI serves SPA at / and admin at /admin/"
```

---

### Task 6: Build wiring + README + end-to-end verification

Wire a convenience rebuild target, document the workflow, and verify the running app for real.

**Files:**
- Modify: `Makefile` (add `frontend` target; add it to `.PHONY`)
- Create: `frontend/README.md`

**Interfaces:**
- Produces: `make frontend` rebuilds `frontend/dist/`. `make build`/`make run` stay Go-only (dist is committed).

- [ ] **Step 1: Add the `frontend` target to `Makefile`**

In the `.PHONY` line, add `frontend`:
```makefile
.PHONY: all build run test clean release frontend
```
Then add this target (place it after the `build` target):
```makefile
# Rebuild the embedded web UI into frontend/dist (committed). Requires Node.
# Run after changing frontend/ source, then commit the regenerated dist.
frontend:
	cd frontend && npm ci && npm run build
```

- [ ] **Step 2: Write `frontend/README.md`**

```markdown
# Frontend

Vue 3 + TypeScript + Quasar + Vite. Built to `dist/`, embedded into the Go
binary via `//go:embed` (see `embed.go`). The built `dist/` is committed so
`go build` and CI need no Node step.

## Develop

    npm install
    npm run dev        # hot-reload dev server; proxies /api to http://localhost:8080

Run the Go server separately for the API: `make run` (from the repo root).

## Build (regenerate the embedded bundle)

    npm run build      # type-check (vue-tsc) + vite build -> dist/
    # or, from the repo root:
    make frontend

Commit the regenerated `dist/` alongside your source changes.

## Notes

- Hash routing (`/#/…`) — no server-side SPA fallback needed.
- JSON over `fetch` only; no WebSocket/protobuf.
- The previous stone-age admin is preserved at `/admin/` (`public/admin/`).
```

- [ ] **Step 3: Verify the Makefile target is well-formed (dry run)**

Run: `make -n frontend`
Expected: prints `cd frontend && npm ci && npm run build` (no execution).

- [ ] **Step 4: End-to-end manual verification (agent drives, reports evidence)**

Start the server in the background and probe the real endpoints:
```bash
make run &          # serves on http://localhost:8080
sleep 2
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/            # expect 200
curl -s http://localhost:8080/ | grep -o 'id="app"'                        # expect id="app"
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/admin/       # expect 200
curl -s http://localhost:8080/api/version                                  # expect {"version":"..."}
curl -s http://localhost:8080/api/tournaments                              # expect a JSON array
kill %1
```
Expected: `/` returns 200 and contains `id="app"`; `/admin/` returns 200; `/api/version` and `/api/tournaments` return JSON. If a browser is available, additionally load `http://localhost:8080/`, confirm the header shows the version and the tournament list (or "No tournaments yet.") renders, and that the "admin" link opens the working stone-age admin. Use the `verify` skill / a browser tool if present.

- [ ] **Step 5: Commit**

```bash
git add Makefile frontend/README.md
git commit -m "build: make frontend target + frontend README"
```

---

## Self-Review

**Spec coverage** (against `2026-07-17-m2b-frontend-shell-design.md`):
- Stack + layout → Tasks 1, 3. ✓
- Typed JSON client surfacing the error envelope → Task 3 (`client.ts`, `ApiError`). ✓
- Pinia `tournaments` store → Task 3. ✓
- One read-only view reading `/api/version` + `/api/tournaments` → Task 3 (App.vue version, HomeView list). ✓
- Build to `dist/`, committed, embedded unchanged → Task 4; `embed.go` untouched. ✓
- Stone-age admin at `/admin/` → Task 2. ✓
- Hash routing, no SPA fallback → Task 3 (`router/index.ts`). ✓
- Dev proxy `/api` → `:8080` (http) → Task 1 (`vite.config.ts`). ✓
- dist strategy A (gitignore node_modules, not dist) → Task 1 (`.gitignore`); CI untouched. ✓
- Verification: build + type-check + `go test` + manual → Tasks 4, 5, 6. ✓
- Two npm touchpoints only → Task 1 Step 8, Task 4 Step 1. ✓

**Placeholder scan:** no TBD/TODO; every code step contains complete content. ✓

**Type consistency:** `Tournament`/`Version` defined in Task 3 Step 1 and consumed in Steps 3, 6, 7; `api`/`ApiError` defined in Step 2 and consumed in Steps 3, 6; `useTournamentsStore` defined in Step 3 and consumed in Step 7; `router` default export defined in Step 4 and consumed in Step 8. Names align. ✓
