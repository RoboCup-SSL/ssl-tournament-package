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
