# Traefik and the dashboard

With Prometheus running on this host (`deploy/prometheus/`), **no change to
the Traefik stack is needed**: the dashboard registers itself through labels
on the existing `web` entrypoint, and `/metrics` is scraped over `proxy-net`
without passing through Traefik at all.

| Entrypoint | Bound to | Router |
| --- | --- | --- |
| `web` | loopback + Tailscale IP, :80 | `hermes-dashboard`: `Host(DASHBOARD_HOST) && !PathPrefix(/metrics)` |

Verify:

```
laptop-on-tailnet$ curl -s -o /dev/null -w '%{http_code}\n' http://<dashboard-host>/healthz   # 200
laptop-on-tailnet$ curl -s -o /dev/null -w '%{http_code}\n' http://<dashboard-host>/metrics   # 404 (not routed)
srv01$ docker exec prometheus wget -qO- http://hermes-dashboard:8080/metrics | head -3        # bff_* lines
```

## Alternative kept for reference: Prometheus on another LAN host

If scraping must come from `svrdocker` instead, add a LAN-only entrypoint
and an allow-listed router (this was ADR-008's original design):

`traefik.yaml`:

```yaml
entryPoints:
  web:
    address: ":80"
  metrics:
    address: ":9100"
```

Traefik `compose.yaml` ports: add `"<lan-ip>:9100:9100"`.

Dashboard labels:

```yaml
      - "traefik.http.routers.hermes-metrics.entrypoints=metrics"
      - "traefik.http.routers.hermes-metrics.rule=Path(`/metrics`)"
      - "traefik.http.routers.hermes-metrics.middlewares=hermes-metrics-allow"
      - "traefik.http.routers.hermes-metrics.service=hermes-dashboard"
      - "traefik.http.middlewares.hermes-metrics-allow.ipallowlist.sourcerange=<svrdocker-lan-ip>/32"
```
