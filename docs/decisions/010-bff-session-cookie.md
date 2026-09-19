# ADR-010: Authenticate the browser with a BFF-issued HttpOnly session cookie

Date: 2026-09-19 · Status: Accepted

## Context

A static bearer token bundled into a static frontend is visible to anyone who can load the page — on a tailnet that is everyone. `EventSource` cannot send an `Authorization` header, so bearer auth cannot protect the SSE endpoints at all. Traefik terminates plain HTTP (WireGuard encrypts the path), so a `Secure` cookie is impossible.

## Decision

The operator types `BFF_API_KEY` once into a login page. `POST /api/auth/session` validates it (constant-time compare, rate-limited) and sets `bff_session`: 32 random bytes, `HttpOnly`, `SameSite=Strict`, 24 h, no `Secure`. Sessions live in memory. Every other `/api` route and both SSE streams require the cookie.

## Consequences

- Frontend and API must share one origin (→ ADR-011).
- A BFF restart requires re-login; acceptable for one user.
- The login POST is the only non-GET route and never touches Hermes.
