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

Design notes: system fonts only (no external requests), dark-first tokens
with a light scheme, status always spoken as a word next to a dot, motion
limited to a brief highlight on new feed rows and disabled under
`prefers-reduced-motion`.
