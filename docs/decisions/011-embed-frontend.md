# ADR-011: Serve the frontend from the Go binary via embed.FS

Date: 2026-09-19 · Status: Accepted

## Context

The original PRD proposed a separate nginx container. ADR-010 requires the frontend and API to share an origin. Every additional component on `proxy-net` must be pinned, patched, and resource-limited on a constrained host.

## Decision

Build the Vue app in a Node stage of the Dockerfile and embed the output with `embed.FS`. The BFF serves `/` (SPA fallback) and `/api` from one process.

## Consequences

- One container, one Traefik router, no CORS, no nginx config.
- A UI change requires rebuilding the binary; fine for one user.
