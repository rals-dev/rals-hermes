# ADR-012: Vue 3 + Vite + TypeScript for the frontend

Date: 2026-09-19 · Status: Accepted

## Context

Three pages, interval polling with caching, one `EventSource`, static build output. The portfolio focus is the Go side; the frontend should be boring and correct in whatever the owner is most comfortable with.

## Decision

Vue 3 (Composition API, `<script setup lang="ts">`), Vite, `@tanstack/vue-query` for polling and cache, `vue-router`, Tailwind. No Pinia — server state lives in vue-query, UI state in local refs.

## Consequences

- vue-query enforces the ≥ 5 s polling rule as configuration.
- Tooling is Node 24 at build time only; nothing Node runs in production.
