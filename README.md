# rals-hermes — Hermes Agent Monitoring Dashboard

A read-only dashboard for a multi-profile [Hermes Agent](https://hermes-agent.nousresearch.com/)
deployment. One Go binary is the backend-for-frontend: it talks to each
profile's API server with that profile's own key, aggregates health,
sessions, runs and scheduled jobs, streams live tool activity to the browser,
and serves the embedded Vue UI — without ever starting an agent run.

**Status:** v1 feature-complete (M0–M4); deployment and the end-to-end
acceptance run are tracked in [`docs/operator-tasks.md`](docs/operator-tasks.md).

Tested against Hermes **0.21.2**, image digest
`sha256:f79d70bc1d23c7553f762c4eb937ba2991e3ba07c15a5e71affa57b3d20b10a5`.

## What it shows

| Page | Answers |
| --- | --- |
| Overview | Is the gateway up? Which profiles are reachable and accepting their key? Is any agent working right now? A live feed of tool calls, replies and delegations across every profile. |
| Floor | Each profile illustrated as a worker at a station — idle, working, delegating, or offline — with a line drawn to whichever profile it's currently delegated to. See [ADR-021](docs/decisions/021-floor-view.md). |
| Agent | Sessions of one profile (source, open/ended, tool calls, tokens, cost), enabled toolsets, that profile's live feed. |
| Session | Metadata, usage and cost, the message timeline with tool arguments and results. |
| Scheduled jobs | Cron jobs across profiles with last/next run and failure streaks. |

Why the feed is built from sessions rather than `/v1/runs`: Hermes has no
endpoint that lists runs, and work started from Telegram or by delegation
never appears there. See [`docs/t003-findings.md`](docs/t003-findings.md).

## Architecture in one paragraph

Browser → Traefik → **BFF** (`:8080`, this repo) → Hermes API server, one
HTTP client per profile with its own bearer key (`/p/<profile>/` prefixes).
Responses are cached for 3 s and coalesced; a lazy per-profile poller tails
recent sessions only while someone is subscribed and turns new messages
into server-sent events. Keys are loaded from environment variables named
in `config.yaml`, stored as a `Secret` type that cannot be printed, and never
appear in responses or logs. The browser authenticates once with a separate
dashboard key and then holds an `HttpOnly` session cookie. Decisions and
their reasons are in [`docs/decisions/`](docs/decisions/README.md).

## Repository layout

```
cmd/bff/            entry point
internal/api        HTTP routes, auth, SSE relay and feed endpoints, embedded UI
internal/hermes     upstream client and payload types (one Client per profile)
internal/activity   session poller → activity events
internal/cache      TTL cache with singleflight
internal/config     YAML loader, Secret type, validation
internal/observ     Prometheus metrics
internal/server     http.Server lifecycle
internal/view       shared dashboard projections
web/                Vue 3 app (built output embedded via web/embed.go)
deploy/             compose stack, Traefik notes, Alloy stack, Grafana dashboard
testdata/fixtures   redacted real payloads, one directory per profile
scripts/            collect-fixtures.sh
docs/               PRD, ADRs, findings, operator checklist, runbook
```

## Running locally

Requirements: Go 1.27+, Node 24+ (frontend), `jq` (fixture script only).

```
cp config.example.yaml config.yaml          # base_url points at a tunnel to Hermes
export BFF_API_KEY=anything-for-dev
export HERMES_KEY_DEFAULT=… HERMES_KEY_CODER=… HERMES_KEY_TESTER=… HERMES_KEY_PRODUCT=…
make run                                     # CGO_ENABLED=0 go run ./cmd/bff → :8080
```

Frontend with hot reload (proxies `/api` to `:8080`):

```
cd web && npm ci && npm run dev              # http://localhost:5173
```

Single binary with the UI embedded:

```
make build                                   # builds web/dist, then bin/bff
```

Tests and checks:

```
make test        # go test (pure Go)
make test-race   # with the race detector (needs a C toolchain; CI runs it)
make lint        # golangci-lint
cd web && npm test && npm run typecheck
```

The process refuses to start if any referenced environment variable is
empty. `LOG_LEVEL` (`debug|info|warn|error`) controls verbosity; logs are JSON
on stdout, one line per request, never containing credentials.

Everything is built with `CGO_ENABLED=0`: the production image is static, and
on macOS it also sidesteps SDK/Command Line Tools mismatches in the linker.

## Configuration

`config.yaml` (see `config.example.yaml`): server address and timeouts, the
name of the env var holding the dashboard key, cache TTL, activity poller
bounds, the upstream timeout, and one entry per profile with `name`,
`base_url` and `key_env`. Adding a profile is a config change and a restart;
no code is involved.

## API

All routes under `/api` return JSON and require the session cookie except
login. `/healthz` and `/metrics` are open.

| Method | Path |
| --- | --- |
| `POST` / `DELETE` | `/api/auth/session` |
| `GET` | `/api/overview`, `/api/agents`, `/api/agents/{profile}` |
| `GET` | `/api/agents/{profile}/sessions`, `/api/agents/{profile}/sessions/{id}` |
| `GET` | `/api/agents/{profile}/runs/{run_id}`, `…/runs/{run_id}/stream` (SSE relay) |
| `GET` | `/api/agents/{profile}/activity/stream` (SSE, BFF-generated) |
| `GET` | `/api/jobs` |
| `GET` | `/healthz`, `/metrics` |

Errors always have the shape `{"code": "...", "message": "..."}`; upstream
failures are classified (`upstream_unreachable`, `upstream_unauthorized`,
`upstream_busy`, `run_not_found`, …) and never forwarded raw.

## Deployment

Images are built by GitHub Actions and published to `ghcr.io/<owner>/rals-hermes`
(`:latest`, `:<sha>`, `:<tag>`). The compose stacks (dashboard, Prometheus,
Grafana with provisioned data source and dashboard, Alloy) are under
`deploy/`; the
step-by-step procedure for the host is [`docs/runbook.md`](docs/runbook.md).

## Security model

- Hermes keys: env vars → `config.Secret`, held only in the BFF process,
  never logged or returned; one key per profile, none shared.
- The BFF calls only `GET` endpoints on Hermes; nothing it does can start,
  steer or stop a run.
- Browser auth: a single dashboard key exchanged once for a random
  `HttpOnly; SameSite=Strict` cookie; failed logins are rate-limited.
- Network: reachable only through Traefik on a Tailscale network; the BFF
  port is not published; `/metrics` is not routed by Traefik at all —
  Prometheus scrapes it over the container network.
- Container: distroless, non-root, read-only filesystem, no capabilities.

## License

MIT — see `LICENSE`.
