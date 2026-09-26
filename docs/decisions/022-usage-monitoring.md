# ADR-022: Per-agent usage monitoring — on-the-fly, no persisted state

Date: 2026-09-22 · Status: Accepted

## Context

ADR-007 deferred per-agent token/cost aggregates because computing them
looked like it required either summing every session on every request or
persisting state, and "no persistent state" is a standing constraint (PRD
§ 1, Non-Goals). The operator asked
whether Hermes already does this monitoring — it doesn't, at the
aggregate level — and, given the choice between an on-the-fly aggregate
and a persisted time series, chose on-the-fly.

Two facts from `docs/t003-findings.md` § 3 shape the design:

- `GET /api/sessions` has **no date filter**, only `limit`/`offset`, and is
  ordered by `last_active` descending. There is no known retention limit
  on Hermes' side, so summing "every session a profile has ever had"
  means paginating an unbounded, ever-growing history on every cache
  miss — the exact cost ADR-007 was avoiding.
- `estimated_cost_usd`/`actual_cost_usd` are `0.0`/`null` for
  subscription-backed models. A profile with no metered usage should
  read as "no cost data", not "$0.00 from nothing" — and token counts
  stay meaningful either way.

## Decision

- New endpoint `GET /api/usage`: token and cost totals per profile over a
  **trailing 24h window**, not "everything Hermes still has". Hermes
  returns sessions newest-first, so the BFF pages through
  `GET /api/sessions` per profile and stops the moment a session's
  `last_active` falls outside the window — the common case is one
  upstream call per profile, not a full-history scan.
- A safety cap (`usageMaxPages` × `usagePageLimit` = 1,000 sessions per
  profile) backstops a pathological upstream where `has_more` never
  turns false and no session ever ages out of the window. Hitting it
  marks the profile `truncated: true` rather than either hanging or
  silently under-counting without saying so.
- Cost fields are summed only from sessions that report a non-null
  value; the total is `null` when none did, distinguishing "genuinely
  free" is not representable from "no cost data available" — the UI
  renders both cases as a dash, `$0.00` only when Hermes actually said
  so.
- Cached per profile with the existing `cache.Cache[T]` pattern (ADR
  from the original handoff), but at 60s TTL rather than the 3s
  default — this can mean several upstream calls per profile, so it's
  refreshed less eagerly than a single health probe.
- Surfaced as a compact table (`UsageStrip.vue`) on Overview, below
  `GatewayStrip`, in the same instrument-panel readout style as the rest
  of the page — not a new page. It queries independently (60s refetch)
  and degrades on its own: a slow or failing `/api/usage` never blocks
  the rest of Overview. One profile's fetch failing shows as
  "unavailable" in that row rather than dropping it or failing the
  whole response, matching `/api/jobs`'s per-profile error handling.

### Alternatives considered

- **All-time totals, unbounded**: rejected — no upper bound on Hermes'
  session history, so the per-request cost is unbounded too.
- **Persisted time series** (a small local DB recording periodic
  snapshots): would allow trends beyond what Hermes still retains and
  survive Hermes-side session pruning, but reopens the "no database"
  constraint this project has held since the original PRD. Not chosen;
  worth revisiting if the 24h window turns out to be the wrong
  granularity in practice.
- **User-selectable window** (24h/7d/30d): more flexible, but multiplies
  the cache-key space and upstream calls for a v1 with a single
  homelab operator. Deferred; 24h answers "how much did each agent do
  today", the question that prompted this feature.

## Consequences

- No new dependency, no database, no long-lived state beyond the
  existing in-memory TTL cache.
- The 24h figure is a rolling window: it silently drops older activity
  each time the window advances, by design — there is no way to see
  "yesterday's total" once that session ages out, short of the
  per-session numbers already on the Agent/Session pages.
- If `usageMaxPages` is hit in practice (a profile genuinely running
  more than ~1,000 sessions inside 24h), the reported totals
  under-count and `truncated` says so; revisit the cap if that happens.
