# v1 Acceptance Run

Manual end-to-end checks from `docs/prd.md` § 10, executed against the live
deployment. Fill the result column; attach nothing that contains a key.

| Date | Hermes version | Dashboard image tag | Operator |
| --- | --- | --- | --- |
| _(pending)_ | 0.21.2 | | |

| # | Check | How | Result |
| --- | --- | --- | --- |
| 1 | Overview | Open the dashboard; four profiles show status and the gateway strip shows the right `active_agents` | |
| 2 | Degradation | `docker stop hermes`; reload; every profile `unreachable`, page intact, gateway strip explains; `docker start hermes` | |
| 3 | Partial degradation | Blank `HERMES_KEY_TESTER` in the stack `.env`, `docker compose up -d`; only tester-agent shows `unauthorized`; restore | |
| 4 | Live feed | Send a Telegram message to `default`; a `message`/`tool` row appears within 5 s with a brief highlight; no history replay | |
| 5 | Delegation | Ask the orchestrator to dispatch to `coder-agent`; `delegated` / `delegation finished` rows appear (child session with parent link) | |
| 6 | Profile isolation | `GET /api/agents/tester-agent/runs/<coder run id>` → 404 `run_not_found` | |
| 7 | Credential leakage | `docker compose logs bff \| grep -c <each key>` → 0; grep responses saved from a browser session → 0 | |
| 8 | Auth | `curl -i http://<host>/api/overview` → 401; six wrong logins → 429 | |
| 9 | Not a run trigger | Note `active_api_runs` and the session count; leave the dashboard open 30 min, closed 30 min; both unchanged; `bff_upstream_request_duration_seconds_count` stops increasing while closed | |
| 10 | Metrics and logs | Prometheus target `hermes-bff` UP; Grafana dashboard panels populated; Loki shows `{container="hermes-dashboard"}` lines | |

Observations and follow-ups:

- 
