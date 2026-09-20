# Operator Tasks — Actions That Touch Production

This checklist covers every step that must be performed by the operator (homelab
owner) because it touches the running Hermes stack, the Traefik stack, or hosts
that the BFF repository has no access to. Nothing in this file is executed by
the BFF itself or by CI.

Conventions:

- `server-pribadi` is the SSH alias for `srv01-rals` (user `rals`). Tailscale SSH
  may ask for a browser check on first use.
- Commands prefixed with `local$` run on your workstation; `srv01$` run on the
  server.
- Never paste key values into chat, commit messages, or shell history that is
  persisted. Use `read -rs` to load them into the environment.

---

## M0 — Before the first line of Go

### 0.1 Pin the Hermes image to the digest currently running

Why: fixtures and contract tests are only valid for one specific Hermes build.
`latest` can change under us on the next `docker compose pull`.

Digest observed on 2026-09-19 (Hermes reports version `0.21.2`):

```
nousresearch/hermes-agent@sha256:f79d70bc1d23c7553f762c4eb937ba2991e3ba07c15a5e71affa57b3d20b10a5
```

Steps:

```
srv01$ cd /srv/stacks/hermes
srv01$ cp compose.yaml compose.yaml.bak-$(date +%Y%m%d)
srv01$ sed -i 's#nousresearch/hermes-agent:latest#nousresearch/hermes-agent@sha256:f79d70bc1d23c7553f762c4eb937ba2991e3ba07c15a5e71affa57b3d20b10a5#' compose.yaml
srv01$ grep -n 'hermes-agent' compose.yaml    # verify exactly one line changed
srv01$ docker compose up -d                    # NOTE: recreates the container → Telegram is briefly down
srv01$ docker inspect --format '{{.Config.Image}}' hermes
```

Expected: the last command prints the `@sha256:` reference.

Later upgrades are deliberate: bump the digest, re-run `scripts/collect-fixtures.sh`,
inspect the fixture diff, then bump the "tested against" line in `README.md`.

- [ ] Done

### 0.2 Confirm every profile has its own API server key

Why: the BFF uses one key per profile. A profile without a key fails closed
(its `/p/<profile>/` prefix returns 401) and will show as `unauthorized` on the
dashboard until fixed. `product-agent` was not in the original PRD — its key
status is unknown.

```
srv01$ for f in /srv/data/hermes/.env /srv/data/hermes/profiles/*/.env; do
         echo "== $f"
         grep -E '^(API_SERVER_ENABLED|API_SERVER_KEY)=' "$f" | sed -E 's/(API_SERVER_KEY=).*/\1<set>/'
       done
```

Expected: four files (`default` uses `/srv/data/hermes/.env`), each with
`API_SERVER_ENABLED=true` and `API_SERVER_KEY=<set>`. If `product-agent` is
missing a key, generate one (`openssl rand -hex 32`), add it to that profile's
`.env`, and restart the gateway.

- [ ] Done

### 0.2b Bind the API server to the container network

Why: Hermes binds the API server to `127.0.0.1` by default (`API_SERVER_HOST`),
which makes it reachable only from inside the Hermes container itself — not
from the BFF container, and not through an SSH tunnel to the container IP.
Confirmed on 2026-09-19: `/health` answered on loopback inside the container
(version 0.21.2) but refused on the `proxy-net` address.

With multiplexing there is a single listener, so only the `default` profile's
`.env` needs the change. `0.0.0.0` inside the container exposes the port to
the Docker networks the container is attached to (`proxy-net`,
`hermes_default`) and nothing else; it is still not published on the host.
`API_SERVER_KEY` remains the only guard — that is the model the PRD assumes.

```
srv01$ grep -n '^API_SERVER_HOST=' /srv/data/hermes/.env || echo 'API_SERVER_HOST=0.0.0.0' >> /srv/data/hermes/.env
# if the line already exists with 127.0.0.1, edit its value instead of appending
srv01$ cd /srv/stacks/hermes && docker compose restart hermes     # Telegram is briefly down
srv01$ sleep 8; curl -s --max-time 5 http://<proxy-net-ip>:8642/health
```

Expected: `{"status": "ok", "platform": "hermes-agent", "version": "0.21.2"}`.
The gateway needs close to a minute after the restart before the listener is
up; a "connection refused" in the first 60 s is not a failure.

Hermes logs this warning after the change — read it once and take it
seriously:

> API server is network-accessible (0.0.0.0) AND the terminal backend is
> 'local' (unsandboxed). Agent work dispatched through this endpoint runs as
> the host user with full terminal/file access.

Anyone holding a profile key can run commands inside the Hermes container,
which includes reading every profile's `.env`. The port is visible only to
containers on `proxy-net` and `hermes_default` (Traefik, Obsidian, later the
BFF); it must never be published on the host, Tailscale, or the LAN.
Switching to `terminal.backend: docker` is a Hermes decision outside this
repository, but worth considering.

- [ ] Done

### 0.3 Open a port-forward to the Hermes API server

Why: port 8642 is not published on the host (only 9119 is). The tunnel must
target the container's IP on `proxy-net`.

```
srv01$ docker inspect -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}={{$v.IPAddress}} {{end}}' hermes
```

Copy the `proxy-net` IP, then from your workstation (keep this terminal open):

```
local$ ssh -N -L 8642:<proxy-net-ip>:8642 server-pribadi
```

Verify from a second terminal (no key needed for `/health`):

```
local$ curl -s http://127.0.0.1:8642/health
```

- [ ] Done

### 0.4 Load the four keys into your local shell (never into a file)

Read each key on the server (`grep '^API_SERVER_KEY=' <file>`), then in the
terminal where you will run the fixture script:

```
local$ read -rs HERMES_KEY_DEFAULT && export HERMES_KEY_DEFAULT
local$ read -rs HERMES_KEY_CODER   && export HERMES_KEY_CODER
local$ read -rs HERMES_KEY_TESTER  && export HERMES_KEY_TESTER
local$ read -rs HERMES_KEY_PRODUCT && export HERMES_KEY_PRODUCT
```

Sanity check that isolation works (second call MUST be 401):

```
local$ curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $HERMES_KEY_CODER"   http://127.0.0.1:8642/p/coder-agent/v1/capabilities
local$ curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $HERMES_KEY_DEFAULT" http://127.0.0.1:8642/p/coder-agent/v1/capabilities
```

- [ ] Done

### 0.5 Collect fixtures and answer the "where do runs come from" question

Why: the BFF's live feed design depends on whether Telegram- and
delegation-initiated work appears in `/v1/runs` at all (decision #4).

1. From Telegram, send the `default` agent a small task that will call at least
   one tool and delegate once to `coder-agent` (e.g. "list the files in the
   repo X and ask coder-agent to summarise the README").
2. While it is running, execute:

```
local$ ./scripts/collect-fixtures.sh
```

3. The script writes redacted JSON under `testdata/fixtures/<profile>/` and
   prints a short report. Paste the report (not the fixtures) into the session.

What we are looking for in that report:

- `active_runs` in `/health/detailed` for `default` while the Telegram task runs
  (0 → Telegram work is invisible to `/v1/runs`; ≥1 → it is visible).
- The shape of `/api/sessions` entries: is there a `run_id`, an `active`
  flag, `updated_at`, token usage fields?
- The shape of `/api/sessions/{id}/messages`: does it support `offset`/`after`
  for incremental reads?

- [ ] Done

### 0.6 Create the public GitHub repository

```
local$ gh repo create rals-dev/rals-hermes --public --source=. --remote=origin --push
```

GHCR packages published by GitHub Actions default to the repo's visibility,
so the server can `docker compose pull` without a token.

- [ ] Done

---

## M4 — Deployment (do not start before M3 is merged)

### 4.1 Add a LAN entrypoint to Traefik with an IP allow-list

Why: Prometheus/Alloy live on `svrdocker`, which is NOT on the tailnet. They
reach `srv01-rals` over the LAN (`eno1`, `<lan-ip>`). Traefik currently
listens only on `127.0.0.1:80` and the Tailscale IP. Decision #8.

Follow `deploy/traefik/README.md`: add entrypoint `metrics` (`:9100`) to
`traefik.yaml` and the port mapping `<lan-ip>:9100:9100` to the Traefik
compose file, then `docker compose up -d` in `/srv/stacks/traefik/`.
Verify after 4.2:

```
svrdocker$ curl -s http://<lan-ip>:9100/metrics | head -5
other-host$ curl -s -o /dev/null -w '%{http_code}\n' http://<lan-ip>:9100/metrics   # expect 403
```

- [ ] Done

### 4.2 Deploy the `hermes-dashboard` stack

See `docs/runbook.md` § 1 step 2. `.env` needs `IMAGE_OWNER`, `IMAGE_TAG`,
`DASHBOARD_HOST` (pattern `hermes.srv01-rals.<tailnet>.ts.net`, like the
existing Obsidian rule), `METRICS_ALLOW_CIDR` (`<svrdocker-lan-ip>/32`),
`BFF_API_KEY`, and the four `HERMES_KEY_*` values.

Verify: `http://<DASHBOARD_HOST>/` shows the login page; the container log
shows `configuration loaded` with four profiles.

- [ ] Done

### 4.3 Deploy the separate `alloy` stack

Why: no Alloy runs on `srv01-rals` today, so container logs never reach Loki.
Decision #9 keeps it out of the dashboard stack because it needs the Docker
socket.

```
srv01$ mkdir -p /srv/stacks/alloy && cd /srv/stacks/alloy
# copy deploy/alloy/compose.yaml and config.alloy from the repo
# set LOKI_URL=http://<svrdocker-lan-ip>:3100/loki/api/v1/push in .env
srv01$ docker compose up -d
```

Only containers labelled `logging=loki` are shipped (the dashboard sets it;
add the label to the Hermes compose service if its logs are wanted).
Verify in Grafana Explore: `{container="hermes-dashboard"}` returns JSON lines.

- [ ] Done

### 4.4 Register the scrape job in Prometheus on `svrdocker`

Add to the Prometheus config (or Alloy scrape config) on `svrdocker`:

```yaml
- job_name: hermes-bff
  scrape_interval: 15s
  static_configs:
    - targets: ['<lan-ip>:9100']
```

Reload Prometheus, then check Status → Targets shows `hermes-bff` as UP.

- [ ] Done

### 4.5 Import the Grafana dashboard

Import `deploy/grafana/hermes-bff.json` via Dashboards → New → Import, select
the existing Prometheus data source.

- [ ] Done

### 4.6 Run the end-to-end acceptance tests

Follow the numbered list in `docs/prd.md` § Definition of Done. Record the
outcome of each in `docs/acceptance-v1.md` (created during M4).

- [ ] Done
