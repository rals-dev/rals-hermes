# rals-hermes — Hermes Agent Monitoring Dashboard

A read-only dashboard for a multi-profile [Hermes Agent](https://hermes-agent.nousresearch.com/)
deployment. A single Go binary acts as a backend-for-frontend: it talks to each
profile's API server with that profile's own key, aggregates health, sessions,
runs and jobs, streams live tool activity to the browser, and serves the
embedded Vue UI — without ever starting an agent run.

**Status:** M0 (prerequisites). See `docs/prd.md` § 7 for the milestone plan.

## Documents

| File | Purpose |
| --- | --- |
| [`docs/prd.md`](docs/prd.md) | Product requirements and engineering handoff (revision 2) |
| [`docs/decisions/`](docs/decisions/README.md) | Architecture decision records — these win over the PRD |
| [`docs/operator-tasks.md`](docs/operator-tasks.md) | Every step that touches production, for the operator |

## Layout

```
cmd/bff/          entry point
internal/         config, hermes client, cache, activity poller, api, observability
web/              Vue 3 frontend (embedded into the binary at build time)
deploy/           compose stack, Traefik patch, Alloy stack, Grafana dashboard
testdata/fixtures redacted real Hermes payloads captured by scripts/collect-fixtures.sh
scripts/          operator and developer scripts
```

## Development

Requirements: Go 1.25+, Node 24+ (frontend only), `jq` (fixtures only).

```
make test     # go test -race ./...
make lint     # golangci-lint run
make run      # go run ./cmd/bff
```

Tested against Hermes image digest: _to be filled in at the end of M0_.

## Security model in one paragraph

Hermes API keys are loaded from environment variables named in `config.yaml`,
held only in the BFF process, and never logged or returned. The browser
authenticates once with a separate static secret and then holds an `HttpOnly`
session cookie. The BFF is reachable only through Traefik on a Tailscale
network; `/metrics` is additionally exposed on a LAN entrypoint restricted by
IP allow-list for the Prometheus host.

## License

MIT — see `LICENSE`.
