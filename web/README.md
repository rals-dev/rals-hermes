# Web UI

Vue 3 + Vite + TypeScript, `@tanstack/vue-query` for polling, `vue-router`,
Tailwind v4. Built output is embedded into the Go binary (ADR-011).

```
npm ci            # once
npm run dev       # http://localhost:5173, proxies /api to the BFF on :8080
npm test          # vitest
npm run typecheck # vue-tsc
npm run build     # → dist/, picked up by `go build` at the repo root
```

Design notes: an instrument-panel look — warm charcoal surfaces, amber as
the only interactive colour, status always spoken as a word next to a lamp,
gateway numbers as readouts. Typeface is B612 (designed for cockpit
displays), bundled locally so the page makes no external requests. Motion
is limited to a brief highlight on new feed rows and disabled under
`prefers-reduced-motion`. A warm-grey light scheme follows the OS setting.

Development against a deployed instance: `VITE_API_TARGET=http://<dashboard-host> npm run dev`.
