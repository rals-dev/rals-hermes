# Runbook

Operational procedures for the dashboard on the homelab host. Placeholders:
`<lan-ip>` (host address on the LAN), `<tailnet>` (Tailscale MagicDNS
suffix), `<svrdocker-lan-ip>` (Prometheus/Loki host), `<owner>` (GitHub
repository owner).

## 1. First deployment

1. Traefik: nothing to change (see `deploy/traefik/README.md`).
2. Dashboard stack:
   ```
   mkdir -p /srv/stacks/hermes-dashboard && cd /srv/stacks/hermes-dashboard
   curl -fsSLo compose.yaml https://raw.githubusercontent.com/<owner>/rals-hermes/main/deploy/hermes-dashboard/compose.yaml
   curl -fsSLo config.yaml  https://raw.githubusercontent.com/<owner>/rals-hermes/main/deploy/hermes-dashboard/config.yaml
   curl -fsSLo .env         https://raw.githubusercontent.com/<owner>/rals-hermes/main/deploy/hermes-dashboard/.env.example
   chmod 600 .env && $EDITOR .env      # fill every value
   docker compose pull && docker compose up -d
   docker compose logs --tail=20
   ```
   Expected log lines: `configuration loaded` with four profiles, then
   `server listening`. A missing secret aborts start-up with the variable
   name in the message.
3. Open `http://<DASHBOARD_HOST>/` from a device on the tailnet, sign in with
   `BFF_API_KEY`.
4. Alloy stack (`/srv/stacks/alloy/`): copy `deploy/alloy/compose.yaml`,
   `config.alloy`, set `LOKI_URL` in `.env`, `docker compose up -d`.
5. Prometheus stack (`/srv/stacks/prometheus/`): copy
   `deploy/prometheus/compose.yaml`, `prometheus.yml`, `.env` (`PROM_BIND`
   can stay `127.0.0.1` now that Grafana is local), `docker compose up -d`.
6. Grafana stack (`/srv/stacks/grafana/`): copy `deploy/grafana/compose.yaml`,
   `provisioning/`, `hermes-bff.json`, and `.env` from `.env.example`
   (`GRAFANA_HOST`, admin password, optional `LOKI_URL`). `docker compose up -d`.
   The Prometheus data source and the "Hermes BFF" dashboard (folder
   *Hermes*) are provisioned automatically; open `http://<GRAFANA_HOST>/`.

## 2. Upgrading the dashboard

Every push to `main` that publishes an image queues the CI `deploy` job,
which waits for approval in the `production` environment. Once approved it
runs `deploy.sh <short-sha>` on the host over SSH. By hand, from the stack
directory:

```
./deploy.sh <sha>       # a 7-character commit SHA from CI, or a v* tag
./deploy.sh status      # pinned tag and live /healthz
```

`deploy.sh` pulls the tag, pins it in `.env`, recreates the container and
waits for `/healthz` to report that version; otherwise it restores the
previous tag. Rollback is `./deploy.sh <previous-sha>`; past deploys are in
`deploy-history.log`.

### Deploy job setup (once)

1. Host: copy `deploy/hermes-dashboard/deploy.sh` next to `compose.yaml`,
   generate a key pair for CI and add the public key to the stack owner's
   `~/.ssh/authorized_keys`, pinned to the script and to tailnet addresses:
   ```
   restrict,from="100.64.0.0/10,fd7a:115c:a1e0::/48",command="/srv/stacks/hermes-dashboard/deploy.sh" ssh-ed25519 AAAA... gha-deploy
   ```
   With Tailscale SSH on (`tailscale set --ssh`), tailnet connections to
   port 22 never reach sshd, so the key above would not apply. Have sshd
   also listen on 2222 (`Port 22` + `Port 2222` in
   `/etc/ssh/sshd_config.d/`, then `systemctl daemon-reload && systemctl
   restart ssh.socket`) and open it on `tailscale0` only in the firewall.
2. Tailscale: add `tag:ci` to `tagOwners`, allow `tag:ci` only to the host
   on `tcp:2222`, and create an OAuth client with the `auth_keys` write
   scope for `tag:ci`.
3. GitHub, environment `production` (required reviewer, deployment branch
   `main` only):
   - secrets `TS_OAUTH_CLIENT_ID`, `TS_OAUTH_SECRET`, `DEPLOY_SSH_KEY`
     (the private key);
   - variables `DEPLOY_HOST` (the host's tailnet IP), `DEPLOY_USER`,
     `DEPLOY_PORT` (default 2222), `DEPLOY_KNOWN_HOSTS`
     (`ssh-keyscan -p 2222 -t ed25519 <tailnet-ip>`), `DASHBOARD_URL`.

## 3. Upgrading Hermes

The BFF's payload types were captured against one Hermes build. Before
bumping the digest in `/srv/stacks/hermes/compose.yaml`:

1. Open a tunnel and export the keys (below), then run
   `./scripts/collect-fixtures.sh` against the **new** version.
2. `git diff testdata/fixtures` — look for removed fields and status changes.
3. `make test` — the contract test in `internal/hermes` decodes every
   fixture; fix types if it fails.
4. Bump the digest, restart Hermes, update the "tested against" line in
   `README.md`.

## 4. Local development against the live host

```
# terminal 1 — tunnel to the Hermes container (port 8642 is not published)
ssh -N -L 8642:$(ssh server-pribadi "docker inspect -f '{{(index .NetworkSettings.Networks \"proxy-net\").IPAddress}}' hermes"):8642 server-pribadi

# terminal 2 — keys stay in this shell only; one prompt per profile in config.yaml
hermes-keys() {
  for v in $(sed -n 's/^      key_env: //p' config.yaml); do
    printf '%s: ' "$v"; read -rs "$v"; echo; export "$v"
  done
}
hermes-keys
CGO_ENABLED=0 BFF_API_KEY=dev go run ./cmd/bff
```

Read the key values on the host with
`grep '^API_SERVER_KEY=' /srv/data/hermes/.env /srv/data/hermes/profiles/*/.env`.

## 5. Symptoms and causes

| Symptom | Likely cause | Check |
| --- | --- | --- |
| Every profile `unreachable` | Hermes container down, or `API_SERVER_HOST` back to loopback | `docker exec hermes sh -c 'grep -i :21C2 /proc/net/tcp'` → `00000000:21C2` means 0.0.0.0 |
| One profile `unauthorized` | Its `API_SERVER_KEY` changed or was blanked | Compare `.env` on the host with the stack's `.env` |
| Feed says `reconnecting` forever | Traefik buffering or the BFF restarted | `curl -N -b <cookie> http://<host>/api/agents/default/activity/stream` should print `: connected` immediately |
| Feed replays old messages | Regression of ADR-018 | `go test ./internal/activity -run Reentering` |
| Prometheus target `hermes-bff` DOWN | Dashboard container not on `proxy-net` or renamed | `docker exec prometheus wget -qO- http://hermes-dashboard:8080/healthz` |
| Login 429 | Five wrong keys within a minute from one address | Wait 60 s |
| Start-up fails: `environment variable … is required but empty` | `.env` incomplete | Fill it; the message names the variable |
| A new Hermes profile is missing from the dashboard | Profiles are read from `config.yaml` at start-up, never discovered | § 8 |
| A just-added profile stays `unauthorized` | Its key was generated but Hermes has not restarted | `cd /srv/stacks/hermes && docker compose restart hermes` |

## 6. Emergency: build and load an image without CI

```
docker build --platform linux/amd64 --build-arg VERSION=manual -t rals-hermes:manual .
docker save rals-hermes:manual | ssh server-pribadi docker load
# then set IMAGE_OWNER/IMAGE_TAG so the image reference is rals-hermes:manual,
# or retag on the host: docker tag rals-hermes:manual ghcr.io/<owner>/rals-hermes:manual
```

## 7. What the dashboard must never do

Start a run. Every upstream call is `GET`; the acceptance test "30 minutes
open, then 30 minutes closed, run count unchanged" (`docs/prd.md` § 10 item 9)
is the check to repeat after any change to `internal/hermes`.

## 8. Adding or removing a profile

`deploy/hermes-dashboard/profile.sh` (ADR-025) edits `config.yaml` and `.env`
in the stack directory, recreates the dashboard, and rolls both files back if
it does not start with the change. Run it as the user that owns
`/srv/data/hermes`, from a checkout; nothing is copied to the host:

```
ssh <host> 'cd /srv/stacks/hermes-dashboard && bash -s -- add researcher-agent' < deploy/hermes-dashboard/profile.sh
ssh <host> 'cd /srv/stacks/hermes-dashboard && bash -s -- remove researcher-agent' < deploy/hermes-dashboard/profile.sh
```

- The profile must already exist in Hermes (`/srv/data/hermes/profiles/<name>`).
- If its `.env` has an `API_SERVER_KEY`, the key is reused and Hermes needs
  nothing. If not, a key is generated and written there **after** the
  dashboard is verified; Hermes loads it on its next restart. Add
  `--restart-hermes` to `add` to restart immediately. That interrupts every
  agent's in-flight work.
- `remove` never touches Hermes' files. It refuses to remove the last
  profile.
- Backups are left as `config.yaml.bak-<timestamp>` and `.env.bak-<timestamp>`.
- The script refuses, without changing anything, when `upstream.profiles` is
  not the last block of `config.yaml`, the profile is already configured, or
  its `HERMES_KEY_*` variable is already in `.env`.
- It needs `compose.yaml` with `env_file: .env` (the repository version since
  ADR-025). With an older copy, the new variable never reaches the container,
  and the script rolls back.
