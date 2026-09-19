# ADR-006: Defer Kanban board data to v2

Date: 2026-09-19 · Status: Accepted

## Context

The Hermes Kanban board stores data in SQLite and has no REST endpoint. Reading the SQLite file requires mounting the Hermes data volume — which also holds every profile's `.env` — into the BFF. The `hermes-kanban` PyPI package is a fourth Python service on a constrained host. `docker exec … hermes kanban list --json` requires the Docker socket (root).

## Decision

Kanban is out of scope for v1. The session-based feed (ADR-004) already surfaces delegations, which is the closest signal to 'what is the agent working on'. Re-evaluate when Hermes exposes Kanban over the API server or when Kanban dispatch officially replaces `delegate_task`.

## Consequences

- Epic 3 shrinks to scheduled jobs only.
- If reconsidered, the only acceptable path is `hermes-kanban` treated as a fifth, failure-tolerant upstream.

## Addendum 2026-09-19 (after T-003)

Sessions created by Kanban dispatch carry `source: "kanban"` in
`/api/sessions`. The board itself stays out of v1, but the *work* it
dispatches is visible in the activity feed and session lists.
