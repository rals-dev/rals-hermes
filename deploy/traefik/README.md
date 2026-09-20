# Traefik changes for the dashboard (ADR-008)

Two edits to the existing stack at `/srv/stacks/traefik/`. Nothing about the
Tailscale-facing `web` entrypoint changes.

## 1. `traefik.yaml` — add a LAN entrypoint for Prometheus

```yaml
entryPoints:
  web:
    address: ":80"
  metrics:
    address: ":9100"
```

## 2. `compose.yaml` — publish that entrypoint on the LAN address only

```yaml
    ports:
      - "127.0.0.1:80:80"
      - "<tailscale-ip>:80:80"
      - "127.0.0.1:8080:8080"
      - "<lan-ip>:9100:9100"      # new: Prometheus on svrdocker scrapes here
```

Then:

```
cd /srv/stacks/traefik && docker compose up -d
```

## What is routed where

| Entrypoint | Bound to | Routers |
| --- | --- | --- |
| `web` | loopback + Tailscale IP, :80 | `hermes-dashboard`: `Host(DASHBOARD_HOST) && !PathPrefix(/metrics)` |
| `metrics` | LAN IP, :9100 | `hermes-metrics`: `Path(/metrics)` behind `ipallowlist` = `METRICS_ALLOW_CIDR` |

The router labels live in `deploy/hermes-dashboard/compose.yaml`; only the
entrypoint and the port mapping belong to the Traefik stack.

## Verify

```
svrdocker$ curl -s http://<lan-ip>:9100/metrics | head -3        # 200, bff_* lines
other-lan-host$ curl -s -o /dev/null -w '%{http_code}\n' http://<lan-ip>:9100/metrics   # 403
laptop-on-tailnet$ curl -s -o /dev/null -w '%{http_code}\n' http://<dashboard-host>/metrics  # 404
```
