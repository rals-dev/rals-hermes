# T-003 Findings — Real Hermes Payloads (2026-09-19)

Source: `scripts/collect-fixtures.sh` against Hermes **0.21.2** (digest
`sha256:f79d70bc…`), four profiles, multiplexed listener. Redacted fixtures are
under `testdata/fixtures/`. Cross-profile key isolation verified: the
`default` key returns **401** on every named prefix.

## 1. There is no list-runs endpoint — confirmed

`GET /v1/runs` → **405 Method Not Allowed** on all profiles. Only `POST`
exists there. `run_id` values cannot be discovered; the SSE relay
(`/runs/{id}/stream`) is useful only when the caller already knows the id.

## 2. `/health/detailed` is gateway-global, not per-profile

The bodies returned via `/`, `/p/coder-agent/`, `/p/tester-agent/` and
`/p/product-agent/` are **byte-identical** apart from `disk.free_bytes`.
`readiness.checks.background_queues.active_api_runs`, `active_delegations`
and `active_agents` describe the whole gateway.

Consequences for the overview (T-104):

- Reachability/auth **is** per profile (a prefix with a bad key → 401), so
  `healthy | unauthorized | unreachable` per card is still meaningful.
- Gateway-wide numbers (`active_api_runs`, `active_delegations`,
  `active_agents`, `gateway_state`, disk, version) belong in a **single
  header block**, not repeated on four cards.
- `platforms` carries per-profile platform state keyed as
  `"<profile>:telegram"` (`coder-agent:telegram`, `tester-agent:telegram`);
  the `default` profile's entry is plain `telegram`. `product-agent` has no
  Telegram platform. This is the only per-profile signal in the body and
  should feed the per-profile card.
- `active_api_runs` was `0` during collection while no Telegram task was in
  flight, so the question "does Telegram work count as an API run?" is still
  open — see § 7.

## 3. `/api/sessions` — the real per-profile activity source

Shape: `{ object: "list", data: [...], has_more, limit, offset }`, ordered by
`last_active` descending. Per session (all present on every profile):

| Field | Notes |
| --- | --- |
| `id` | `YYYYMMDD_HHMMSS_<hex>` |
| `source` | Observed values: `telegram`, `cli`, **`kanban`** |
| `parent_session_id` | Delegation link (null in the sample; `include_children=true` was passed) |
| `started_at`, `ended_at`, `last_active` | Unix seconds, float |
| `end_reason` | `null` while open; `agent_close`, `cli_close` when finished |
| `message_count`, `tool_call_count`, `api_call_count` | Counters — `message_count` doubles as a cheap "did anything change" signal |
| `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_write_tokens`, `reasoning_tokens` | Usage, per session |
| `estimated_cost_usd`, `actual_cost_usd` | **Cost is already computed by Hermes** (`0.0` / `null` for the subscription-backed model in use) |
| `model`, `title`, `preview`, `user_id` | `user_id` is the Telegram numeric id — redacted in fixtures |
| `pinned`, `archived`, `hidden`, `has_system_prompt`, `has_model_config` | Flags |

Notable: the `coder-agent` and `tester-agent` sessions all have
`source: "kanban"` — Kanban-dispatched work **is visible through the sessions
API** even though the board itself is not (ADR-006 stands, with less lost).

`GET /api/sessions/{id}` returns `{ object, session }` with the same fields.

## 4. `/api/sessions/{id}/messages` — incremental reads are possible

Shape: `{ object: "list", session_id, pagination: { limit, offset, order: "oldest", returned }, data: [...] }`.
Message ids are **monotonically increasing integers** per session store.
Per message: `id`, `role` (`user` | `assistant` | `tool` | `session_meta`),
`content`, `tool_calls` (on assistant messages that call tools),
`tool_call_id` + `tool_name` (on `tool` result messages), `timestamp`,
`token_count`, `finish_reason`, `reasoning`, `reasoning_content`,
`display_kind`.

Poller strategy (T-205): keep `seen_count` per session; when the session
list shows `message_count > seen_count`, fetch
`?offset=<seen_count>&limit=<delta>` with `order=oldest`. An assistant
message with `tool_calls` ⇒ `tool.started`; a `tool` message ⇒
`tool.completed` (pair on `tool_call_id`). A new session with
`parent_session_id` set ⇒ `subagent.start`; its `end_reason` becoming
non-null ⇒ `subagent.complete`.

## 5. `/v1/skills` returns 500 on every profile

`{"error":{"type":"server_error", ...}}`. Hermes-side bug in 0.21.2.
The agent-detail endpoint must treat skills as an optional section
(`skills: null` + `warnings: ["skills_unavailable"]`), never fail the page.

## 6. Other shapes

- `/v1/capabilities.features` is a flat boolean map (`run_status`,
  `run_events_sse`, `session_resources`, `model_options`, `skills_api`, …).
  Use it to gate optional calls; note `skills_api: true` despite the 500.
- `/api/jobs` → `{ jobs: [...] }` with `schedule.expr`, `state`,
  `next_run_at`, `last_run_at`, `last_status`, `failure_streak`,
  `latest_execution{ status, … }`. Job `prompt` is free text (redacted).
- `/v1/toolsets` → `{ object: "list", data: [{ name, label, enabled, configured, tools: [...] }] }`.
- `/api/model/options` lists providers with hints such as
  `"paste FIREWORKS_API_KEY to activate"` — names only, no values.
- `product-agent` has zero sessions; every other endpoint answers normally.

## 7. Still open — one cheap check

While a Telegram task is running on `default`, poll the gateway:

```
watch -n 2 'curl -s -H "Authorization: Bearer $HERMES_KEY_DEFAULT" http://127.0.0.1:8642/health/detailed | jq -c ".readiness.checks.background_queues, .active_agents"'
```

If `active_api_runs` stays `0` while `active_agents` rises, Telegram work is
confirmed invisible to `/v1/runs` and `active_agents` becomes the
"something is running" indicator for the header. Either way the design in
ADR-004 (session poller) is required, because runs cannot be listed.
