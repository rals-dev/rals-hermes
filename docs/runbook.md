# Runbook

Operational procedures for the dashboard on the homelab host. Placeholders:
`<lan-ip>` (host address on the LAN), `<tailnet>` (Tailscale MagicDNS
suffix), `<svrdocker-lan-ip>` (Prometheus/Loki host), `<owner>` (GitHub
repository owner).

## 1. First deployment

1. Traefik: apply `deploy/traefik/README.md` (entrypoint `metrics` on
   `<lan-ip>:9100`) and `docker compose up -d` in `/srv/stacks/traefik/`.
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
5. On `svrdocker`: add the Prometheus scrape job (target `<lan-ip>:9100`)
   and import `deploy/grafana/hermes-bff.json`.

## 2. Upgrading the dashboard

```
cd /srv/stacks/hermes-dashboard
sed -i 's/^IMAGE_TAG=.*/IMAGE_TAG=<sha>/' .env     # a commit SHA from CI, not "latest"
docker compose pull && docker compose up -d
```

Rollback is the same command with the previous SHA.

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

# terminal 2 — keys stay in this shell only
hermes-keys() {
  for v in HERMES_KEY_DEFAULT HERMES_KEY_CODER HERMES_KEY_TESTER HERMES_KEY_PRODUCT; do
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
| `/metrics` 403 on svrdocker | Its IP is not in `METRICS_ALLOW_CIDR` | Traefik access log |
| Login 429 | Five wrong keys within a minute from one address | Wait 60 s |
| Start-up fails: `environment variable … is required but empty` | `.env` incomplete | Fill it; the message names the variable |

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
