# ADR-004: Derive the live activity feed from sessions, keep the run SSE relay

Date: 2026-09-19 · Status: Accepted

## Context

The Hermes API server has no endpoint that lists runs; `/health/detailed` returns counts only; runs are scoped to 'the profile that created the run' via the API. The operator's real workload arrives via Telegram and `delegate_task`, which most likely never register in `/v1/runs`. A feed built only on `/v1/runs/{id}/events` would show almost nothing.

## Decision

1. Verify empirically in T-003 whether Telegram work appears in `/v1/runs`.
2. Build a per-profile session poller in the BFF: list recent sessions, tail their messages incrementally, and emit a BFF-generated SSE stream at `/api/agents/{profile}/activity/stream`.
3. Keep the `/v1/runs/{id}/events` relay — it is cheap and serves API-initiated runs.
4. Revise the success criterion from '< 3 s' to '≤ one poll interval (5 s)'.

## Consequences

- The BFF now holds small in-memory state (cursors per tracked session); still no database.
- Load is bounded by ADR-018 (lazy lifecycle, windows, caps).
- Delegations are visible through child sessions (`include_children`).

## Addendum 2026-09-19 (after T-003)

Confirmed by fixtures: `GET /v1/runs` is 405, `/health/detailed` is
gateway-global (identical on every prefix), and sessions expose
`message_count`, `last_active`, `parent_session_id`, `end_reason` and
integer-id messages with `offset`-based pagination — enough for an
incremental poller. See `docs/t003-findings.md` § 3–4.
